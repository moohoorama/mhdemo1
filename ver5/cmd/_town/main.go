// Temporary tool: walks Liu Bei round a town (assets/towns/<id>.json) in a window. Click the ground to walk, a
// person to talk, a building or a way out to go through its door; R or a right click shows the tiles he can walk.
//
//	go run ./cmd/_town
//	go run ./cmd/_town -town assets/towns/zhuo.json
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
	path := flag.String("town", "assets/towns/zhuo.json", "town file")
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
	if err = g.OpenTown(*path); err != nil {
		fail(err)
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("탁군")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
