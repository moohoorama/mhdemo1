package gui

import (
	"image"
	"math"

	"srpg/internal/content"
	"srpg/internal/replay"
)

// An AI army's turn (enemy or allied) plays where the player can see it: an action starts only
// when every cell it touches is on screen, and when nothing is playing the camera glides to the
// waiting action nearest it. The player's camera is locked meanwhile (cameraLocked).

// actionCells are the cells an action plays out on: the actor's cell before it, where it moves
// and whom it strikes or helps.
func actionCells(a *replay.Action) []content.Point {
	out := []content.Point{}
	if p, ok := a.Before[a.Actor]; ok {
		out = append(out, p)
	}
	for _, s := range a.Steps {
		c := s.Command
		if c.Kind == "move" {
			out = append(out, content.Point{X: c.X, Y: c.Y})
		}
		if p, ok := a.Before[c.Target]; ok && c.Target != a.Actor {
			out = append(out, p)
		}
	}
	return out
}

// actionCenter is the canvas point the camera glides to for an action: the middle of its cells.
func (g *Game) actionCenter(cells []content.Point) (float64, float64) {
	var x0, y0, x1, y1 float64
	for i, c := range cells {
		x, y := g.field.Center(float64(c.X), float64(c.Y))
		if i == 0 {
			x0, y0, x1, y1 = x, y, x, y
		}
		x0, y0, x1, y1 = math.Min(x0, x), math.Min(y0, y), math.Max(x1, x), math.Max(y1, y)
	}
	return (x0 + x1) / 2, (y0 + y1) / 2
}

// onScreen reports whether every cell, with a unit standing on it, is inside the battle view and
// clear of the HUD.
func (g *Game) onScreen(cells []content.Point) bool {
	vp := g.battleViewport()
	m := 24 * K * g.zoom
	view := image.Rect(vp.Min.X+int(m), vp.Min.Y+int(2*m), vp.Max.X-int(m), vp.Max.Y-int(m))
	for _, c := range cells {
		x, y := g.toScreen(g.field.Center(float64(c.X), float64(c.Y)))
		if !image.Pt(int(x), int(y)).In(view) || g.overHUD(int(x*worldPx/uiPx), int((y-m)*worldPx/uiPx)) {
			return false
		}
	}
	return true
}

// nextShown picks the waiting action to start (its index in p.waiting), or -1: an AI replay
// starts a ready action that is on screen (only when nothing plays, for one at a time), and with
// nothing playing glides to the ready one nearest the camera and starts it once there.
func (p *Player) nextShown(ready []int) int {
	g := p.g
	if !p.ai {
		if len(ready) == 0 {
			return -1
		}
		return ready[0]
	}
	if g.pan != nil || g.field == nil || g.S.Settings.Replay == "one" && len(p.running) > 0 {
		return -1
	}
	for _, k := range ready {
		if cells := actionCells(p.actions[p.waiting[k]]); len(cells) == 0 || g.onScreen(cells) || p.waiting[k] == p.aimed {
			return k
		}
	}
	if len(p.running) > 0 || len(ready) == 0 {
		return -1
	}
	best, bestD := ready[0], math.Inf(1)
	for _, k := range ready {
		x, y := g.actionCenter(actionCells(p.actions[p.waiting[k]]))
		if d := math.Hypot(x-g.camX, y-g.camY); d < bestD {
			best, bestD = k, d
		}
	}
	p.aimed = p.waiting[best]
	g.panTo(g.actionCenter(actionCells(p.actions[p.aimed])))
	return -1
}

// cameraLocked reports whether the player's camera (scroll, zoom, minimap) waits: on an AI
// army's turn and while one plays.
func (g *Game) cameraLocked() bool {
	if g.player.ai && g.player.Busy() {
		return true
	}
	if g.S == nil || g.S.Engine == nil {
		return false
	}
	o := g.S.View()
	return o.Phase == "battle" && o.Turn != "ally"
}
