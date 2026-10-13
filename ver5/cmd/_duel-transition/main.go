// Temporary tool: loops the battle-to-duel transition of one stage duel.
//
//	go run ./cmd/_duel-transition [-duel B01-duel-1]
//
// Tune live with Q/W A/S Z/X T/G E/D (see the on-screen line) or with the flags.
// Enter/Space skips the duel; F12 saves a screenshot to reports/gui.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/duelview"
	"srpg/internal/gui"
	"srpg/internal/session"
)

func main() {
	duelID := flag.String("duel", "B01-duel-1", "stage duel ID")
	path := flag.String("content", "assets/data", "content directory")
	font := flag.String("font", "assets/fonts/NotoSansKR.ttf", "Korean font")
	flag.Float64Var(&duelview.Transition.ZoomFactor, "zoom", duelview.Transition.ZoomFactor, "battle zoom factor")
	flag.Float64Var(&duelview.Transition.PanTime, "pan", duelview.Transition.PanTime, "scroll and UI fade seconds")
	flag.Float64Var(&duelview.Transition.ZoomTime, "zoomtime", duelview.Transition.ZoomTime, "zoom seconds")
	flag.Float64Var(&duelview.Transition.UITime, "ui", duelview.Transition.UITime, "duel UI fade-in seconds")
	flag.Float64Var(&duelview.Transition.StartScale, "start", duelview.Transition.StartScale, "duel start scale")
	flag.Parse()
	d, err := content.Load(*path)
	if err != nil {
		fail(err)
	}
	s, err := session.New(d, os.TempDir())
	if err != nil {
		fail(err)
	}
	g, err := gui.New(s, *font, 1, "")
	if err != nil {
		fail(err)
	}
	defer g.Close()
	if err := g.PreviewDuelTransition(*duelID); err != nil {
		fail(err)
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("duel transition")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
