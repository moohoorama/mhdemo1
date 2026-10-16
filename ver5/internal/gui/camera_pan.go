package gui

import "math"

// camPan glides the camera to a point (docs/ui-mockup, idea 3): eased in and out over
// about half a second, dropped as soon as the player moves the camera. The name tags and
// the enemy phase camera use it.
type camPan struct {
	fx, fy, tx, ty, t float64
}

const panTime = .56

// panTo starts gliding the camera to world point (x, y).
func (g *Game) panTo(x, y float64) {
	if math.Hypot(x-g.camX, y-g.camY) < 1 {
		return
	}
	g.pan = &camPan{fx: g.camX, fy: g.camY, tx: x, ty: y}
}

// panToCell glides the camera to a cell's centre.
func (g *Game) panToCell(u, v int) {
	if g.field != nil {
		g.panTo(g.field.Center(float64(u), float64(v)))
	}
}

func (g *Game) advancePan(dt float64) {
	p := g.pan
	if p == nil {
		return
	}
	p.t += dt
	k := bezier(.45, 0, .15, 1, min(1, p.t/panTime))
	g.camX, g.camY = p.fx+(p.tx-p.fx)*k, p.fy+(p.ty-p.fy)*k
	if p.t >= panTime {
		g.pan = nil
	}
}
