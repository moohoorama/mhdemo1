// Temporary check: in each explore area on the given scene maps, can one walk from the area's start to every person
// and exit? Uses the scene walk rules (scenemap.Walkable).
//
//	go run ./cmd/_walkcheck ROOT S_zhuo S_luoyang ...
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"srpg/internal/scenemap"
)

type area struct {
	Map    string
	At     *[2]int
	People map[string]struct {
		At   *[2]int
		Move *struct{ Patrol [][2]int }
	}
	Exits []struct {
		Name string
		At   [2]int
	}
}

func main() {
	root := os.Args[1]
	want := map[string]bool{}
	for _, m := range os.Args[2:] {
		want[m] = true
	}
	g := filepath.Join(root, "assets", "graphics")
	cat, err := scenemap.LoadCatalog(g)
	if err != nil {
		panic(err)
	}
	files, _ := filepath.Glob(filepath.Join(root, "assets", "scenario", "hero", "explore*.json"))
	sort.Strings(files)
	walks := map[string]*scenemap.Walk{}
	bad := 0
	for _, f := range files {
		raw, _ := os.ReadFile(f)
		var x struct{ Areas map[string]area }
		if json.Unmarshal(raw, &x) != nil {
			continue
		}
		keys := make([]string, 0, len(x.Areas))
		for k := range x.Areas {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			a := x.Areas[k]
			if !want[a.Map] || a.At == nil {
				continue
			}
			w, ok := walks[a.Map]
			if !ok {
				b, err := os.ReadFile(filepath.Join(g, "maps", a.Map+".json"))
				if err != nil {
					panic(err)
				}
				m, _, err := scenemap.Parse(b)
				if err != nil {
					panic(err)
				}
				w = scenemap.Walkable(m, cat)
				walks[a.Map] = w
			}
			seen := reach(w, *a.At)
			check := func(what string, at [2]int) {
				// a person stands on a closed tile by design (the walk opens round them in the game): reaching a
				// neighbour is enough
				for _, d := range [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if seen[[2]int{at[0] + d[0], at[1] + d[1]}] {
						return
					}
				}
				bad++
				fmt.Printf("%s %s %s %v\n", filepath.Base(f), k, what, at)
			}
			for n, p := range a.People {
				if p.At != nil {
					check("사람 "+n, *p.At)
				}
				if p.Move != nil {
					for _, t := range p.Move.Patrol {
						if !seen[t] {
							bad++
							fmt.Printf("%s %s 순찰 %s %v\n", filepath.Base(f), k, n, t)
						}
					}
				}
			}
			for _, e := range a.Exits {
				check("출구 "+e.Name, e.At)
			}
		}
	}
	fmt.Println("닿지 않는 곳", bad)
}

func reach(w *scenemap.Walk, from [2]int) map[[2]int]bool {
	seen := map[[2]int]bool{from: true}
	q := [][2]int{from}
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if !seen[n] && w.Open(n[0], n[1]) {
				seen[n] = true
				q = append(q, n)
			}
		}
	}
	return seen
}
