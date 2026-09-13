package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"os"
	"testing"
)

// Opt-in because RunGame owns the native desktop event loop.
// DEMO1_WINDOW_TEST=1 go test -count=1
func TestMain(m *testing.M) {
	if os.Getenv("DEMO1_WINDOW_TEST") != "1" {
		os.Exit(m.Run())
	}
	g, err := newGame("", "")
	if err != nil {
		panic(err)
	}
	probe := &animationProbe{game: g, seen: map[int]string{}, trees: map[int]string{}}
	ebiten.SetWindowSize(screenW, screenH)
	if err := ebiten.RunGame(probe); err != nil {
		panic(err)
	}
	if len(probe.seen) != 8 {
		panic(fmt.Sprintf("captured %d water frames", len(probe.seen)))
	}
	distinct := map[string]bool{}
	for _, c := range probe.seen {
		distinct[c] = true
	}
	if len(distinct) != 8 {
		panic(fmt.Sprintf("expected eight distinct water patches, got %d", len(distinct)))
	}
	treeFrames := map[string]bool{}
	for _, pixels := range probe.trees {
		treeFrames[pixels] = true
	}
	if len(treeFrames) != 4 {
		panic(fmt.Sprintf("tree animation has %d distinct frames", len(treeFrames)))
	}
	if probe.landChanged {
		panic("land changed across native frames")
	}
}

type animationProbe struct {
	*game
	seen        map[int]string
	trees       map[int]string
	land        color.Color
	landChanged bool
}

func (p *animationProbe) Update() error {
	if len(p.seen) == 8 && len(p.trees) == 4 {
		return ebiten.Termination
	}
	p.game.animate()
	return nil
}
func (p *animationProbe) Draw(dst *ebiten.Image) {
	p.game.Draw(dst)
	// Centers of known full-river and full-land cells in the demo.
	x, y := p.pos(10.5, 8.5)
	pixels := make([]byte, 0, 8*4*3)
	for dy := -2; dy < 2; dy++ {
		for dx := -4; dx < 4; dx++ {
			r, g, b, _ := dst.At(int(x)+dx, int(y)+dy).RGBA()
			pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8))
		}
	}
	p.seen[p.waterFrame] = string(pixels)
	x, y = p.pos(5.5, 12.5)
	pixels = nil
	for dy := -12; dy < 4; dy++ {
		for dx := -12; dx < 12; dx++ {
			r, g, b, _ := dst.At(int(x)+dx, int(y)+dy).RGBA()
			pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8))
		}
	}
	p.trees[p.treeTick/16] = string(pixels)
	x, y = p.pos(1.5, 1.5)
	c := dst.At(int(x), int(y))
	if p.land == nil {
		p.land = c
	} else if p.land != c {
		p.landChanged = true
	}
}
