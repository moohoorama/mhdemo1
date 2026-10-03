// Command exportmap writes an editor map as draw data for ver3, reusing the editor's
// terrain rules (autotile masks, layered ground subtiles, decoration placement).
//
//	go run ./ver3/tools/exportmap ver3/asset/maps/demo-map.json > map.json
package main

import (
	"demo1/internal/terrain"
	"encoding/json"
	"fmt"
	"os"
)

type ground struct {
	X, Y   int
	Layers []int // tileset sprite ids; 0 is the animated water placeholder
}

type decoration struct {
	X, Y      int
	Animation string
	Phase     int
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: exportmap MAP.json")
		os.Exit(2)
	}
	w, err := terrain.Load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := terrain.BuildMapScene(w)
	out := struct {
		Width, Height int
		Bounds        [4]int
		Cells         [][2]int // ground kind, decoration; row-major
		Ground        []ground
		Decorations   []decoration
	}{Width: w.Width, Height: w.Height,
		Bounds: [4]int{s.Bounds.Min.X, s.Bounds.Min.Y, s.Bounds.Max.X, s.Bounds.Max.Y}}
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			c := w.Cell(x, y)
			out.Cells = append(out.Cells, [2]int{int(c.Ground), int(c.Decoration)})
		}
	}
	for _, t := range s.Ground {
		out.Ground = append(out.Ground, ground{t.Position.X, t.Position.Y, t.Layers})
	}
	for _, d := range s.Decorations {
		out.Decorations = append(out.Decorations, decoration{d.Position.X, d.Position.Y, terrain.ObjectAnimation(d.Object), d.Phase})
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
