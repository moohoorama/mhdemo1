// Command _reelgrid prints, as JSON, which game cells of scene maps a person can walk (scenemap.Walkable), for
// tools/title_reel/build.py to lay out and check the title reel: "." open ground, ":" open raised ground (a rampart,
// a tower top: scenemap.Footing), "#" shut, "~" water, " " off the map.
//
//	go run ./cmd/_reelgrid S_luoyang S_long_field
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"srpg/internal/scenemap"
)

type grid struct {
	U0, V0 int
	Rows   []string
}

func main() {
	cat, err := scenemap.LoadCatalog("assets/graphics")
	if err != nil {
		fail(err)
	}
	out := map[string]grid{}
	for _, id := range os.Args[1:] {
		data, err := os.ReadFile("assets/graphics/maps/" + id + ".json")
		if err != nil {
			fail(err)
		}
		m, _, err := scenemap.Parse(data)
		if err != nil {
			fail(err)
		}
		w, f := scenemap.Walkable(m, cat), scenemap.Footing(m, cat)
		// a person on cell (u, v) stands over the small tiles 2u..2u+1, 2v..2v+1
		u0, v0 := (w.Origin[0]+1)/2, (w.Origin[1]+1)/2
		u1, v1 := (w.Origin[0]+w.W)/2-1, (w.Origin[1]+w.H)/2-1
		g := grid{U0: u0, V0: v0}
		for v := v0; v <= v1; v++ {
			row := make([]byte, 0, u1-u0+1)
			for u := u0; u <= u1; u++ {
				c := byte('.')
				for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
					su, sv := 2*u+d[0], 2*v+d[1]
					switch {
					case !m.Inside(su, sv):
						c = ' '
					case m.Glyph(su, sv) == '~' && c != ' ':
						c = '~'
					case !w.Open(su, sv) && f.Open(su, sv) && m.Step(su, sv) > 0 && (c == '.' || c == ':'):
						c = ':'
					case !w.Open(su, sv) && (c == '.' || c == ':'):
						c = '#'
					}
				}
				row = append(row, c)
			}
			g.Rows = append(g.Rows, string(row))
		}
		out[id] = g
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
