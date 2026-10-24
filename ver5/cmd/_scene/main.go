// _scene serves the game's scene maps in a common browser viewer.
// Run from ver5: go run ./cmd/_scene
//
// A map of the scene map format 2 is drawn from the plan the game draws from (internal/scenemap); an old map by the
// old viewer code (static/legacy.js). For a format 2 map the viewer also draws the same map as it was at the -base
// revision, the old way, and compares the two picture by picture.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"srpg/internal/scenemap"
)

//go:embed static/*
var staticFiles embed.FS

const root = "assets/graphics"

type sceneInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Format int    `json:"format"`
}

// mapFiles lists the scene maps of index.json by id. Files are read on every request, so an edited map shows on
// reload.
func mapFiles() (map[string]string, json.RawMessage, error) {
	index, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		return nil, nil, err
	}
	var catalog struct {
		Maps []string `json:"maps"`
	}
	if err := json.Unmarshal(index, &catalog); err != nil {
		return nil, nil, err
	}
	files := map[string]string{}
	for _, path := range catalog.Maps {
		if strings.HasPrefix(path, "maps/S_") && strings.HasSuffix(path, ".json") {
			files[strings.TrimSuffix(strings.TrimPrefix(path, "maps/"), ".json")] = filepath.Join(root, path)
		}
	}
	return files, index, nil
}

func main() {
	addr := flag.String("listen", "127.0.0.1:8765", "browser viewer address")
	base := flag.String("base", "scene-render-base", "git revision whose old maps a format 2 map is compared with")
	flag.Parse()

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	fail := func(w http.ResponseWriter, err error, code int) {
		http.Error(w, err.Error(), code)
	}
	writeJSON := func(w http.ResponseWriter, data []byte) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(data)
	}
	readMap := func(w http.ResponseWriter, r *http.Request, prefix string) ([]byte, string, bool) {
		files, _, err := mapFiles()
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return nil, "", false
		}
		id := strings.TrimPrefix(r.URL.Path, prefix)
		path, ok := files[id]
		if !ok {
			http.NotFound(w, r)
			return nil, "", false
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return nil, "", false
		}
		return data, id, true
	}
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.FileServer(http.Dir(".")))
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		files, index, err := mapFiles()
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		var scenes []sceneInfo
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				fail(w, err, http.StatusInternalServerError)
				return
			}
			var info sceneInfo
			if err := json.Unmarshal(data, &info); err != nil {
				fail(w, err, http.StatusInternalServerError)
				return
			}
			scenes = append(scenes, info)
		}
		sort.Slice(scenes, func(i, j int) bool { return scenes[i].ID < scenes[j].ID })
		rules, err := os.ReadFile(filepath.Join(root, "scene.json"))
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		scenery, err := os.ReadFile(filepath.Join(root, "map", "scenery.json"))
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(struct {
			Scenes   []sceneInfo     `json:"scenes"`
			Graphics json.RawMessage `json:"graphics"`
			Rules    json.RawMessage `json:"rules"`
			Scenery  json.RawMessage `json:"scenery"`
		}{scenes, index, rules, scenery})
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, data)
	})
	mux.HandleFunc("/api/map/", func(w http.ResponseWriter, r *http.Request) {
		if data, _, ok := readMap(w, r, "/api/map/"); ok {
			writeJSON(w, data)
		}
	})
	// the plan of a format 2 map: ?turn=0..3&decor=1
	mux.HandleFunc("/api/plan/", func(w http.ResponseWriter, r *http.Request) {
		data, _, ok := readMap(w, r, "/api/plan/")
		if !ok {
			return
		}
		m, v2, err := scenemap.Parse(data)
		if err == nil && !v2 {
			err = os.ErrInvalid
		}
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		cat, err := scenemap.LoadCatalog(root)
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		turn, _ := strconv.Atoi(r.URL.Query().Get("turn"))
		plan, err := scenemap.Build(m, cat, scenemap.Options{Turn: turn, Decor: r.URL.Query().Get("decor") == "1"})
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		out, err := json.Marshal(plan)
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, out)
	})
	// the map as it was at the -base revision
	mux.HandleFunc("/api/base/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/base/")
		if strings.ContainsAny(id, "/.:") {
			http.NotFound(w, r)
			return
		}
		data, err := exec.Command("git", "show", *base+":./"+root+"/maps/"+id+".json").Output()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, data)
	})
	log.Printf("장면 맵: http://%s (비교 기준 %s)", *addr, *base)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
