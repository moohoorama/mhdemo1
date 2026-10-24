// Temporary tool: walks Liu Bei round the scene maps of format 2 (assets/towns/tour2.json) in a window, to look at
// how they are drawn (internal/scenemap). Click the ground to walk by small tiles, a name plate, a building or a way
// out to go on to the next map; R or a right click shows the small tiles he can walk.
//
//	go run ./cmd/_town2
//	go run ./cmd/_town2 -tour assets/towns/tour2.json
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
	path := flag.String("tour", "assets/towns/tour2.json", "tour file")
	flag.Parse()
	d, err := content.Load("assets/data")
	if err != nil {
		fail(err)
	}
	dir, err := os.MkdirTemp("", "town")
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
	if err = g.OpenTour(*path); err != nil {
		fail(err)
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("형식 2 둘러보기")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
