package gui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// sceneArrow is an arrow of a Play step with an Arrow: it leaves the archer's chest as the bow is let go (the end of
// the attack motion's drawing frames) and comes down on a cell, flying on a shallow arc.
type sceneArrow struct {
	fu, fv, fh float64 // where it leaves: the archer's cell and the height of its chest above the ground
	tu, tv     float64
	delay, t   float64
	dur        float64
	loosed     bool
}

const (
	arrowPeak       = 44.0 // canvas px the arc climbs above a straight line
	arrowMinFlight  = .55
	arrowFlightCell = .07 // seconds per cell, the battle map's pace
)

// looseArrow queues the arrow of an archer playing anim: it leaves after the motion's first two frames, as in battle.
func (g *Game) looseArrow(v *unitVis, anim string, tu, tv float64) {
	ms := v.art.Animations[anim].MS
	release := 0.0
	for i := 0; i < len(ms) && i < 2; i++ {
		release += float64(ms[i]) / 1000
	}
	d := math.Hypot(tu-v.u, tv-v.v)
	s := g.scene
	s.arrows = append(s.arrows, &sceneArrow{fu: v.u, fv: v.v, fh: g.field.Lift(int(math.Round(v.u)), int(math.Round(v.v))) + v.top*.55, tu: tu, tv: tv,
		delay: release, dur: math.Max(arrowMinFlight, d*arrowFlightCell)})
}

func (g *Game) updateArrows(dt float64) {
	s := g.scene
	kept := s.arrows[:0]
	for _, a := range s.arrows {
		a.t += dt
		if !a.loosed && a.t >= a.delay {
			a.loosed = true
			g.sfxAtCell("shoot", a.fu, a.fv)
		}
		if a.t < a.delay+a.dur {
			kept = append(kept, a)
		}
	}
	s.arrows = kept
}

// drawArrows draws the arrows in flight with a thin shadow on the ground beneath.
func (g *Game) drawArrows(dst *ebiten.Image) {
	fx := g.assets.effect("arrow")
	if len(fx.Frames) == 0 {
		return
	}
	n := float64(len(fx.Frames))
	for _, a := range g.scene.arrows {
		if !a.loosed {
			continue
		}
		k := math.Min(1, (a.t-a.delay)/a.dur)
		at := func(k float64) (x, y float64) {
			fx0, fy0 := g.field.Center(a.fu, a.fv)
			tx, ty := g.field.Center(a.tu, a.tv)
			fy0 -= a.fh
			ty -= g.field.Lift(int(math.Round(a.tu)), int(math.Round(a.tv))) + 4
			return fx0 + (tx-fx0)*k, fy0 + (ty-fy0)*k - 4*arrowPeak*k*(1-k)
		}
		x, y := at(k)
		x2, y2 := at(math.Min(1, k+.02))
		i := int(math.Round(math.Atan2(y2-y, x2-x)/(2*math.Pi)*n+n)) % len(fx.Frames)
		fx.draw(dst, i, x, y, g.toScreen, g.zoom)
	}
}
