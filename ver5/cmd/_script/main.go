// _script serves a chapter's script (plan/script/chNN.json: lines, directions in prose, explore talk) in the browser
// for a person to edit. Edits are written to plan/script/chNN.edited.json next to the original; an agent reads the
// two and carries the changes into the story files and docs/fullstory.md.
// Run from ver5: go run ./cmd/_script -ch 10
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed index.html
var page embed.FS

var (
	portraitID = regexp.MustCompile(`^[a-z0-9_]+$`)
	shotName   = regexp.MustCompile(`^[a-z0-9_-]+\.[a-z0-9_-]+$`)
)

func main() {
	ch := flag.Int("ch", 10, "chapter")
	addr := flag.String("addr", "127.0.0.1:8795", "listen address")
	flag.Parse()

	base := filepath.Join("plan", "script", fmt.Sprintf("ch%02d.json", *ch))
	edited := filepath.Join("plan", "script", fmt.Sprintf("ch%02d.edited.json", *ch))
	if _, err := os.Stat(base); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(page))
	mux.HandleFunc("GET /api/script", func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(base)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		e, err := os.ReadFile(edited)
		if os.IsNotExist(err) {
			e = []byte("null")
		} else if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"base":%s,"edited":%s}`, b, e)
	})
	mux.HandleFunc("POST /api/save", func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		out, _ := json.MarshalIndent(v, "", " ")
		if err := os.WriteFile(edited, append(out, '\n'), 0o644); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		log.Printf("saved %s", edited)
	})
	mux.HandleFunc("GET /picture/{shot}", func(w http.ResponseWriter, r *http.Request) {
		shot := r.PathValue("shot")
		if !shotName.MatchString(shot) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, strings.TrimSuffix(base, ".json")+"."+shot+".png")
	})
	mux.HandleFunc("GET /portrait/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !portraitID.MatchString(id) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join("assets", "graphics", "portraits", id+".png"))
	})

	log.Printf("chapter %d script editor on http://%s", *ch, *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
