package gui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// transitionFade is how long each half of a screen change takes: the old screen darkening, then the new one
// coming out of black.
const transitionFade = .35

// transition is a screen change in progress. While out is under 1 the old screen, held still, darkens and the
// game waits; then in runs from 1 to 0 over the new screen, which plays on as it comes out of black.
type transition struct {
	still   *ebiten.Image
	out, in float64
}

// screenKey names what the screen shows, so that a change between two of them fades through black: the title, a
// story on one map (a scene, the walk, an event on the walk's map), a story without a map, a battle from its
// deployment to its result. Empty while a duel or the battle node on its way to the deployment holds the screen:
// those keep the screen they found.
func (g *Game) screenKey() string {
	switch {
	case g.gallery != nil:
		return "gallery"
	case g.editor != nil:
		return "editor"
	case g.tour != nil:
		return "tour"
	case g.player.Duel != nil:
		return ""
	case g.S.Screen == "title" || g.S.Engine == nil:
		return "title"
	}
	o := g.S.View()
	switch o.Phase {
	case "scenario":
		switch o.Dialogue.Kind {
		case "battle":
			return ""
		case "explore":
			return "map:" + g.S.Engine.ExploreFile().Areas[o.Spot.Area].Map
		}
		if def, _ := g.scenes.For(o.Node); def != nil {
			return "map:" + def.Map
		}
		return "story"
	case "complete":
		return "complete"
	}
	return "battle:" + g.S.Data.Stages[o.Stage].ID
}

// watchScreen starts a transition when the screen changed since the last update. An old screen that had all but
// gone black (a scene leaving, the walk through an exit) skips straight to the new one coming out.
func (g *Game) watchScreen() {
	k := g.screenKey()
	if k == "" || k == g.shownScreen {
		return
	}
	first := g.shownScreen == ""
	g.shownScreen = k
	if first || g.Verify || g.lastFrame == nil {
		return
	}
	t := &transition{out: 0, in: 1}
	if g.darkness > .9 {
		t.out = 1
	} else {
		t.still = g.lastFrame
		g.lastFrame = nil
	}
	g.trans = t
}

// holdForTransition steps the transition by one update; true while the old screen darkens, when nothing else
// may move.
func (g *Game) holdForTransition() bool {
	t := g.trans
	if t == nil {
		return false
	}
	if t.out < 1 {
		if t.out = math.Min(1, t.out+1.0/60/transitionFade); t.out == 1 && t.still != nil {
			t.still.Deallocate()
			t.still = nil
		}
		return true
	}
	if t.in = math.Max(0, t.in-1.0/60/transitionFade); t.in == 0 {
		g.trans = nil
	}
	return false
}

// drawFrame draws the screen, or the old one held still while it darkens, with the transition's black over it,
// and keeps a copy of what it drew for the next change.
func (g *Game) drawFrame(dst *ebiten.Image) {
	t := g.trans
	if t != nil && t.still != nil {
		dst.DrawImage(t.still, nil)
		cover(dst, t.out)
		return
	}
	g.darkness = 0
	g.drawScreen(dst)
	if t != nil {
		cover(dst, t.in)
	}
	if g.screenKey() == "" {
		return // a passing screen is not the one a change leaves
	}
	b := dst.Bounds()
	if g.lastFrame == nil || g.lastFrame.Bounds() != b {
		if g.lastFrame != nil {
			g.lastFrame.Deallocate()
		}
		g.lastFrame = newLayer(b.Dx(), b.Dy())
	}
	g.lastFrame.Clear()
	g.lastFrame.DrawImage(dst, nil)
}

// cover lays black of opacity k over the whole image.
func cover(dst *ebiten.Image, k float64) {
	if k > 0 {
		b := dst.Bounds()
		vector.FillRect(dst, 0, 0, float32(b.Dx()), float32(b.Dy()), color.NRGBA{A: uint8(255 * min(1, k))}, false)
	}
}

// blackOut darkens the world by k and remembers how dark this frame went, so a transition from it knows when
// the screen is black already.
func (g *Game) blackOut(dst *ebiten.Image, k float64) {
	if k > 0 {
		g.darkness = max(g.darkness, k)
		inWorld(func() { rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{A: uint8(255 * min(1, k))}) })
	}
}
