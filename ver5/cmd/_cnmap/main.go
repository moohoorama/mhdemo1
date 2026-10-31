// Temporary tool: walks Liu Bei over the deformed Han map (assets/worldmap/han.json, made by
// tools/worldmap/gen_hanmap.py from the RTK9 overview in koei9/) in a window. The map is drawn as one battle map with
// the mountains, hills and roads of assets/graphics/map/terrain.png (tools/worldmap/gen_terrain.py). Click the ground or
// a name plate to walk there; standing at a city shows what it is; R or a right click shows where he can walk.
//
//	go run ./cmd/_cnmap
//	go run ./cmd/_cnmap -map assets/worldmap/han.json
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/session"
)

func main() {
	path := flag.String("map", "assets/worldmap/han.json", "world map file")
	flag.Parse()
	d, err := content.Load("assets/data")
	if err != nil {
		fail(err)
	}
	dir, err := os.MkdirTemp("", "cnmap")
	if err != nil {
		fail(err)
	}
	s, err := session.New(d, dir)
	if err != nil {
		fail(err)
	}
	g, err := gui.New(s, "assets/fonts/NotoSansKR.ttf", 1, "")
	if err != nil {
		fail(err)
	}
	defer g.Close()
	if err = g.OpenWorldMap(*path); err != nil {
		fail(err)
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("중원 지도")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
