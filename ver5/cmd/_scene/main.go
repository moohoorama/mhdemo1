// _scene serves the game's scene maps in a common browser viewer.
// Run from ver5: go run ./cmd/_scene
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

type sceneInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	addr := flag.String("listen", "127.0.0.1:8765", "browser viewer address")
	flag.Parse()

	root := "assets/graphics"
	index, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		log.Fatal(err)
	}
	var catalog struct {
		Maps []string `json:"maps"`
	}
	if err := json.Unmarshal(index, &catalog); err != nil {
		log.Fatal(err)
	}
	var scenes []sceneInfo
	maps := make(map[string]json.RawMessage)
	for _, path := range catalog.Maps {
		if !strings.HasPrefix(path, "maps/S_") || !strings.HasSuffix(path, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			log.Fatal(err)
		}
		var info sceneInfo
		if err := json.Unmarshal(data, &info); err != nil {
			log.Fatal(err)
		}
		if info.ID == "" {
			log.Fatalf("%s: missing map ID", path)
		}
		scenes = append(scenes, info)
		maps[info.ID] = data
	}
	sort.Slice(scenes, func(i, j int) bool { return scenes[i].ID < scenes[j].ID })
	catalogData, err := json.Marshal(struct {
		Scenes   []sceneInfo     `json:"scenes"`
		Graphics json.RawMessage `json:"graphics"`
	}{Scenes: scenes, Graphics: index})
	if err != nil {
		log.Fatal(err)
	}

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.FileServer(http.Dir(".")))
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(catalogData)
	})
	mux.HandleFunc("/api/map/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/map/")
		data, ok := maps[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(data)
	})
	log.Printf("장면 맵 %d개: http://%s", len(scenes), *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
