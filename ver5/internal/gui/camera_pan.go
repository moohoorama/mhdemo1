package gui

import (
	"math"

	"srpg/internal/content"
)

// camPan glides the camera to a point (docs/ui-mockup, idea 3): eased in and out over
// about half a second, dropped as soon as the player moves the camera. The name tags and
// the enemy phase camera use it.
type camPan struct {
	fx, fy, tx, ty, t float64
}

const panTime = .56

// panTo starts gliding the camera to world point (x, y).
func (g *Game) panTo(x, y float64) {
	x, y = g.clampCam(x, y)
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

// onScreenAt reports whether every cell would be on screen (onScreen) with the camera at (x, y).
func (g *Game) onScreenAt(cells []content.Point, x, y float64) bool {
	cx, cy := g.camX, g.camY
	g.camX, g.camY = x, y
	in := g.onScreen(cells)
	g.camX, g.camY = cx, cy
	return in
}

// glideFirst lays out, for a stretch of beats starting at moment at, a glide to cells when they would
// be off screen with the camera at cam, where the stretch's camera is when it is built; it returns
// the beat and how long the stretch waits for the glide, and moves cam to where the glide ends. The
// beat glides only if the cells are still off screen when it comes (a walk followed may have got there).
func (g *Game) glideFirst(cam *[2]float64, cells []content.Point, at float64) ([]timedFn, float64) {
	if g.field == nil || len(cells) == 0 || g.onScreenAt(cells, cam[0], cam[1]) {
		return nil, 0
	}
	x, y := g.actionCenter(cells)
	*cam = [2]float64{x, y}
	return []timedFn{{at, func() {
		if !g.onScreen(cells) {
			g.panTo(x, y)
		}
	}}}, panTime
}

// followWalk keeps a unit walking in a replay (an AI army's move, a cutscene's or an event's walk) in
// view: once it leaves the middle of the screen the camera eases after it until its walk ends. A glide,
// a duel or the player's own drag wins over it.
func (g *Game) followWalk(dt float64) {
	if g.field == nil || g.scene != nil || !g.player.Busy() || g.pan != nil || g.player.Duel != nil || g.dragging && g.dragMoved {
		g.follow = ""
		return
	}
	walking := func(v *unitVis) bool { return v != nil && v.walk != nil && v.anim == "walk" && v.leave == nil }
	if !walking(g.vis[g.follow]) {
		g.follow = ""
		for _, id := range sorted(g.vis) {
			if v := g.vis[id]; walking(v) && !g.nearMiddle(v) {
				g.follow = id
				break
			}
		}
	}
	v := g.vis[g.follow]
	if v == nil {
		return
	}
	x, y := g.field.Center(v.u, v.v)
	ease := math.Min(1, dt*camEase)
	g.camX, g.camY = g.clampCam(g.camX+(x-g.camX)*ease, g.camY+(y-v.lift-g.camY)*ease)
}

// nearMiddle reports whether a unit stands inside the middle of the battle view (a fifth of it in
// from every side) and clear of the HUD.
func (g *Game) nearMiddle(v *unitVis) bool {
	vp := g.battleViewport()
	x, y := g.field.Center(v.u, v.v)
	sx, sy := g.toScreen(x, y-v.lift)
	mx, my := float64(vp.Dx())/5, float64(vp.Dy())/5
	return sx > float64(vp.Min.X)+mx && sx < float64(vp.Max.X)-mx && sy > float64(vp.Min.Y)+my && sy < float64(vp.Max.Y)-my &&
		!g.overHUD(int(sx*worldPx/uiPx), int(sy*worldPx/uiPx))
}

// glideToLead glides the camera, as the player's turn begins, to the lord, or to the first of the
// player's units still to act, when it is off screen: the camera is left where the other side's last
// action played.
func (g *Game) glideToLead() {
	if g.field == nil || g.S == nil {
		return
	}
	o := g.S.View()
	lead := ""
	for _, u := range o.UnitViews {
		if u.Faction != "ally" || u.AI || u.HP <= 0 || g.vis[u.ID] == nil {
			continue
		}
		if g.S.Data.Characters[u.Def].Lord {
			lead = u.ID
			break
		}
		if lead == "" && !u.Done {
			lead = u.ID
		}
	}
	v := g.vis[lead]
	if v == nil {
		return
	}
	cell := []content.Point{{X: int(math.Round(v.u)), Y: int(math.Round(v.v))}}
	if !g.onScreen(cell) {
		g.panTo(g.field.Center(v.u, v.v))
	}
}
