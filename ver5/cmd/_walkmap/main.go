// Temporary tool: prints which small tiles of a scene map of format 2 can be walked (scenemap.Walkable), with the
// anchors of its standing objects, to place people and exits on the explore files.
//
//	go run ./cmd/_walkmap S_peach_grove2
//	go run ./cmd/_walkmap -fmt S_militia_camp   rewrite the map file laid out as the map files are, and bake its
//	                                             diorama board edge (tools/scene_maps/diorama.py)
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"srpg/internal/scenemap"
)

func main() {
	cat, err := scenemap.LoadCatalog("assets/graphics")
	if err != nil {
		fail(err)
	}
	ids, format := os.Args[1:], false
	if len(ids) > 0 && ids[0] == "-fmt" {
		ids, format = ids[1:], true
	}
	for _, id := range ids {
		path := "assets/graphics/maps/" + id + ".json"
		data, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		m, _, err := scenemap.Parse(data)
		if err != nil {
			fail(err)
		}
		if format {
			out, err := m.Encode()
			if err == nil {
				err = os.WriteFile(path, out, 0o644)
			}
			if err == nil {
				bake := exec.Command("python3", "tools/scene_maps/diorama.py", id)
				bake.Stdout, bake.Stderr = os.Stdout, os.Stderr
				err = bake.Run()
			}
			if err != nil {
				fail(err)
			}
			continue
		}
		w := scenemap.Walkable(m, cat)
		marks := map[[2]int]byte{}
		var names []string
		for i, ob := range m.Objects {
			if s, ok := cat.Sprites[ob.Name]; ok && !s.Ground {
				c := byte('A' + i%26)
				marks[[2]int{ob.U, ob.V}] = c
				names = append(names, fmt.Sprintf("%c=%s(%d,%d)", c, ob.Name, ob.U, ob.V))
			}
		}
		fmt.Printf("%s: origin %v, %dx%d small tiles; . open, # shut, letters objects\n", id, w.Origin, w.W, w.H)
		fmt.Print("      ")
		for x := 0; x < w.W; x++ {
			fmt.Print((w.Origin[0] + x + 100) % 10)
		}
		fmt.Println()
		for y := 0; y < w.H; y++ {
			v := w.Origin[1] + y
			fmt.Printf("%5d ", v)
			for x := 0; x < w.W; x++ {
				u := w.Origin[0] + x
				switch c, ok := marks[[2]int{u, v}]; {
				case ok:
					fmt.Printf("%c", c)
				case w.Open(u, v):
					fmt.Print(".")
				default:
					fmt.Print("#")
				}
			}
			fmt.Println()
		}
		sort.Strings(names)
		fmt.Println(names)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
