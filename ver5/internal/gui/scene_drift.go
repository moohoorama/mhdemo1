//go:build ignore

package gui

import (
	"math"

	"srpg/internal/scenes"
)

// sceneDrift is an actor of a Drift step being carried over water: not a walk (it crosses cells the map's rules close
// and bobs on the swell), so the scene's walk checks leave it alone.
type sceneDrift struct {
	v          *unitVis
	fu, fv     float64
	tu, tv     float64
	t, dur     float64
	base       float64 // the lift it started with
	sink       bool    // sinks from sight at the end
	wadeAtLand bool
}

const (
	driftCellsPerSecond = 1.6
	driftBob            = 1.6 // canvas px the swell lifts and drops it
	driftSinkShare      = .3  // the last share of the drift in which a sinking actor fades
)

// startDrift sets v carried to the step's [u, v]; with Exit it sinks from sight there, otherwise it stands up on
// whatever cell it reaches (out of the water when that is the bank).
func (g *Game) startDrift(v *unitVis, st scenes.Step) {
	s := g.scene
	if s.drifts == nil {
		s.drifts = map[string]*sceneDrift{}
	}
	s.paths[v.id] = nil
	v.walk, v.leave = nil, nil
	d := math.Hypot(st.Drift[0]-v.u, st.Drift[1]-v.v)
	s.drifts[v.id] = &sceneDrift{v: v, fu: v.u, fv: v.v, tu: st.Drift[0], tv: st.Drift[1], dur: math.Max(.3, d/driftCellsPerSecond),
		base: v.lift, sink: st.Exit}
	if anim := motion(v.art, "exhausted"); anim != "" {
		v.play(anim)
	}
}

func (g *Game) updateDrifts(dt float64) {
	for key, d := range g.scene.drifts {
		d.t += dt
		g.placeDrift(d)
		if d.t >= d.dur {
			g.landDrift(d)
			delete(g.scene.drifts, key)
		}
	}
}

// placeDrift puts d's actor where it is at d.t: slid along the line, rising and falling on the swell.
func (g *Game) placeDrift(d *sceneDrift) {
	k := math.Min(1, d.t/d.dur)
	d.v.u, d.v.v = d.fu+(d.tu-d.fu)*k, d.fv+(d.tv-d.fv)*k
	d.v.lift = d.base + driftBob*math.Sin(d.t*5+d.fu+d.fv)
	if d.sink {
		d.v.arriving = leaveTime * math.Max(0, (k-(1-driftSinkShare))/driftSinkShare)
	}
}

// landDrift ends a drift: a sinking actor is gone, any other stands on its cell and rests.
func (g *Game) landDrift(d *sceneDrift) {
	v := d.v
	v.u, v.v = d.tu, d.tv
	v.lift = g.field.Lift(int(math.Round(v.u)), int(math.Round(v.v)))
	if d.sink {
		v.gone, v.arriving = true, 0
		return
	}
	v.wadeAlways = false
	v.play(v.rest())
}

// settleDrifts finishes every drift at once (a click skipping the beats).
func (g *Game) settleDrifts() {
	for key, d := range g.scene.drifts {
		g.landDrift(d)
		delete(g.scene.drifts, key)
	}
}
