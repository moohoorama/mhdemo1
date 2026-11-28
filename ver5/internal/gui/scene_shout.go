package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// A shout is words cried out over an actor's head: big bold letters stamped on, shrinking a little as they rise and
// fade (Step.Shout) — a cry, not a speech balloon. The words are the step's own text, drawn in the game's sans face,
// so a translation changes only the text.

const (
	shoutLife = 1.6  // seconds a shout stays up
	shoutSize = 30.0 // UI px of the letters when the step gives none
)

var (
	shoutFill = color.NRGBA{R: 243, G: 210, B: 58, A: 255}
	shoutRim  = color.NRGBA{R: 58, G: 36, B: 8, A: 255}
)

type shout struct {
	actor, text string
	size, t     float64
}

func (g *Game) startShout(actor, text string, size float64) {
	if size <= 0 {
		size = shoutSize
	}
	g.scene.shouts = append(g.scene.shouts, &shout{actor: actor, text: text, size: size})
}

func (g *Game) updateShouts(dt float64) {
	s := g.scene
	kept := s.shouts[:0]
	for _, sh := range s.shouts {
		if sh.t += dt; sh.t < shoutLife {
			kept = append(kept, sh)
		}
	}
	s.shouts = kept
}

// drawShouts stamps the shouts over their actors' heads, in UI px: popping on large, settling, then rising away.
func (g *Game) drawShouts(dst *ebiten.Image) {
	s := g.scene
	for _, sh := range s.shouts {
		v := s.actors[sh.actor]
		if v == nil || v.gone {
			continue
		}
		x, y := g.field.Center(v.u, v.v)
		sx, sy := worldToUI(g.toScreen(x, y-v.lift-g.field.groundLift(v.u, v.v)-v.top))
		k := sh.t / shoutLife
		at := []float64{0, .12, .2, .75, 1}
		scale := keyframe(k, at, []float64{1.7, .96, 1, .98, .95})
		alpha := keyframe(k, at, []float64{0, 1, 1, 1, 0})
		rise := keyframe(k, at, []float64{0, 0, 0, .3, .6}) * sh.size
		img := g.numberImageFace(sh.text, g.boldFace(sh.size), shoutFill, shoutRim, sh.size/shoutSize)
		h := float64(img.Bounds().Dy()) / drawPx
		g.drawNumberImage(dst, img, sx, sy-h/2-4-rise, scale, 0, alpha)
	}
}
