package gui

import (
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// busy is true while something plays that the next command must wait for, the turn
// banner included: the side to move only starts once its banner is done.
func (g *Game) busy() bool { return g.player.Busy() || g.sceneBusy() || g.turnBanner != nil }

// briefed reports whether the battle's opening replay is waiting for its first briefing to be read:
// the briefing is shown and answered then, though a replay is pending.
func (g *Game) briefed() bool {
	return len(g.modals) > 0 && g.modals[0].kind == "objective" && !g.modals[0].manual && g.player.Busy()
}

// speedRate is the replay speed as a factor: fixed in a story scene, else the player's setting.
func (g *Game) speedRate() float64 {
	if g.show != nil {
		return 1 // the flood is baked at its own pace
	}
	if g.scene != nil {
		return sceneSpeed
	}
	return float64(g.speed)
}

// advance moves the clock, the replay and every unit's animation by one frame.
func (g *Game) advance(dt float64) {
	g.uiSeconds += dt
	g.textSeconds += dt * g.speedRate()
	g.advanceTurnBanner(dt)
	if g.zoomTo != g.zoomFrom && g.zoomT < .5 {
		g.zoomT = math.Min(.5, g.zoomT+dt)
		t := 1 - math.Pow(2, -g.zoomT*10)
		if g.zoomT == .5 {
			t = 1
		}
		g.zoom = g.zoomFrom + (g.zoomTo-g.zoomFrom)*t
	}
	if g.player.Busy() {
		g.advancePan(dt * g.speedRate()) // a replay's camera keeps up with its beats
	} else {
		g.advancePan(dt)
	}
	dt *= g.speedRate()
	if g.duelPreview != nil {
		dt *= g.previewRate
		if g.previewPaused {
			dt = 0
		}
	}
	g.clockMS += dt * 1000
	if g.briefed() {
		g.player.Update(0) // the battle's opening waits until its briefing is read
	} else {
		g.player.Update(dt)
	}
	for _, u := range g.vis {
		u.update(dt)
	}
	g.followWalk(dt)
	g.updateEffects(dt)
	g.updateScene(dt)
	keptToasts := g.toasts[:0]
	for _, t := range g.toasts {
		t.t += dt / g.speedRate()
		if t.t < toastTime {
			keptToasts = append(keptToasts, t)
		}
	}
	g.toasts = keptToasts
	for _, m := range g.modals[:min(1, len(g.modals))] {
		if !g.busy() || g.briefed() {
			m.t += dt / g.speedRate()
		}
	}
	kept := g.popups[:0]
	for _, p := range g.popups {
		p.t += dt
		if p.t < p.life() {
			kept = append(kept, p)
		}
	}
	g.popups = kept
	g.bannerT = math.Max(0, g.bannerT-dt)
	g.hitFlashT = math.Max(0, g.hitFlashT-dt)
	if !g.busy() {
		o := g.S.View()
		if !g.unitsHeld && (o.Phase == "deployment" || o.Phase == "battle" || o.Phase == "result") {
			g.sync(o)
		}
		g.watchSubs(o)
	}
	if g.show != nil && g.show.Advance(dt) {
		g.show = nil
	}
	if g.prelude != nil && (g.show == nil || g.show.over) {
		g.endPrelude()
	}
}

// ensureField builds the battlefield for the observed stage and centres the camera once.
func (g *Game) ensureField(o core.Observation) {
	if o.Map == nil {
		return
	}
	if g.field != nil && g.field.Map.ID == o.Map.ID {
		return
	}
	if g.field != nil {
		g.field.dispose()
	}
	g.field = NewField(g.assets, g.mapArt(o.Map.ID))
	g.vis, g.scene = map[string]*unitVis{}, nil
	g.setZoomImmediate(g.savedZoom(false, battleZoom))
	g.pan = nil
	size := g.field.Size()
	g.camX, g.camY = float64(size.X)/2, float64(size.Y)/2
	g.clearOrder()
	g.intro = nil
	g.selected = ""
	g.resultSkip = false
	if o.Phase == "battle" && o.Round == 1 && o.Turn == "ally" {
		g.modals = append(g.modals, &modal{kind: "objective"})
	}
	if o.Phase == "deployment" && len(o.Map.Allies) > 0 {
		var x, y float64
		for _, p := range o.Map.Allies {
			cx, cy := g.field.Center(float64(p.X), float64(p.Y))
			x += cx
			y += cy
		}
		g.camX, g.camY = x/float64(len(o.Map.Allies)), y/float64(len(o.Map.Allies))
	}
	g.sync(o)
}

// sync makes the shown units match the rules state (between replays, or after one).
func (g *Game) sync(o core.Observation) {
	if o.Map == nil || g.field == nil || g.field.Map.ID != o.Map.ID {
		return
	}
	present := map[string]bool{}
	for _, u := range o.UnitViews {
		present[u.ID] = true
	}
	for id := range g.vis {
		if !present[id] {
			delete(g.vis, id)
		}
	}
	for _, u := range o.UnitViews {
		if g.ord.move != nil && u.ID == g.ord.actor {
			continue // shown at its tentative cell until the order is confirmed or cancelled
		}
		g.syncUnit(u, o)
	}
}

// zooms are the battle's canvas scales, which the wheel steps through and a pinch settles on;
// 1 shows unit pixels 1:1. A battle opens at battleZoom. A battle map smaller than the screen
// starts its steps higher, see zoomSteps.
var zooms = []float64{1, 2, 4}

const battleZoom = 2

// zoomSteps are the zooms this field steps through: on a battle map none so far out that the view
// passes the rectangle round the battle area. They stay whole numbers, so the dots stay even.
func (g *Game) zoomSteps() []float64 {
	fit := g.fitZoom()
	var steps []float64
	for _, z := range zooms {
		if z >= fit {
			steps = append(steps, z)
		}
	}
	if len(steps) == 0 {
		steps = []float64{math.Ceil(fit)}
	}
	return steps
}

// fitZoom is the least zoom at which the view stays inside the battle map's canvas, the battle area and the dark
// outside round it, or 0 where the camera is not held to the battle area (see battleBox).
func (g *Game) fitZoom() float64 {
	if _, ok := g.battleBox(); !ok {
		return 0
	}
	vp, size := g.battleViewport(), g.field.Size()
	return max(float64(vp.Dx())/float64(size.X), float64(vp.Dy())/float64(size.Y))
}

// battleBox is the rectangle round the battle area's diamond in canvas px, as {left, right} and {top,
// bottom}, on a battle map (not a story scene's or an old-format one).
func (g *Game) battleBox() ([2][2]float64, bool) {
	f := g.field
	if f == nil || !f.Map.Native || f.ShowExterior || g.scene != nil || g.exploring() {
		return [2][2]float64{}, false
	}
	c := f.mapOutline()
	return [2][2]float64{{c[3][0], c[1][0]}, {c[0][1], c[2][1]}}, true
}

// nearestZoom is the step of steps nearest z, by ratio.
func nearestZoom(steps []float64, z float64) float64 {
	best := steps[0]
	for _, s := range steps {
		if math.Abs(math.Log(z/s)) < math.Abs(math.Log(z/best)) {
			best = s
		}
	}
	return best
}

// walkView is whether the field is a walk or a story scene rather than a battle, for the zoom the player keeps.
func (g *Game) walkView() bool { return g.scene != nil || g.exploring() }

// savedZoom is the zoom the player last chose on the walk and in scenes (walk) or on a battle map, or
// fallback when none was chosen.
func (g *Game) savedZoom(walk bool, fallback float64) float64 {
	if g.S == nil {
		return fallback
	}
	z := g.S.Settings.BattleZoom
	if walk {
		z = g.S.Settings.ExploreZoom
	}
	if z <= 0 {
		return fallback
	}
	return float64(z)
}

// rememberZoom saves z, the zoom the player settled on, as the walk's or the battle's.
func (g *Game) rememberZoom(z float64) {
	if g.S == nil {
		return
	}
	n := max(1, min(4, int(math.Round(z))))
	v := g.S.Settings
	p := &v.BattleZoom
	if g.walkView() {
		p = &v.ExploreZoom
	}
	if *p == n {
		return
	}
	*p = n
	g.request(session.Request{Op: "settings", Settings: &v})
}

func (g *Game) setZoom(target float64) {
	if target == g.zoom {
		return
	}
	g.zoomFrom, g.zoomTo, g.zoomT = g.zoom, target, 0
}

func (g *Game) setZoomImmediate(target float64) {
	g.zoom, g.zoomFrom, g.zoomTo, g.zoomT = target, target, target, .5
}

func (g *Game) toCanvas(x, y float64) (float64, float64) {
	z := g.zoom
	vp := g.battleViewport()
	return (x-float64(vp.Min.X+vp.Dx()/2))/z + g.camX, (y-float64(vp.Min.Y+vp.Dy()/2))/z + g.camY
}

// toScreen maps canvas pixels to the viewport sub-image (which keeps screen coordinates).
func (g *Game) toScreen(x, y float64) (float64, float64) {
	z := g.zoom
	vp := g.battleViewport()
	return (x-g.camX)*z + float64(vp.Min.X+vp.Dx()/2), (y-g.camY)*z + float64(vp.Min.Y+vp.Dy()/2)
}

// camera pans with the arrow keys, a press on a battle map's edge arrow or a drag with either button and
// zooms with the wheel. A click without dragging selects (left) or cancels (right) instead.
// On an AI army's turn only the clicks work (cameraLocked).
func (g *Game) camera() {
	if g.field == nil || g.popupPending() || g.input {
		g.dragging = false
		return
	}
	step := 12.0 / g.zoom
	locked := g.cameraLocked()
	if locked {
		step = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.camX -= step
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.camX += step
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.camY -= step
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.camY += step
	}
	x, y := cursorPos()
	onMap := g.onMap()
	if ebiten.IsFocused() {
		g.edgeDir, g.edgeOn = g.edgeArrow(locked)
	}
	g.edgeScroll(step)
	if mouseJustPressed(ebiten.MouseButtonRight) && onMap && !g.dragging {
		g.dragging, g.dragMoved, g.dragButton, g.dragX, g.dragY = true, false, ebiten.MouseButtonRight, x, y
	}
	if !locked && (g.dragMoved || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)) {
		g.pan = nil // the player's own camera wins over a glide
	}
	if g.dragging {
		if !mousePressed(g.dragButton) {
			g.dragging = false
			if !g.dragMoved && g.dragButton == ebiten.MouseButtonRight {
				g.rightClick()
			} else if !g.dragMoved {
				g.mapRelease()
			}
		} else if !locked && (g.dragMoved || abs(x-g.dragX)+abs(y-g.dragY) > 4) {
			g.dragMoved = true
			g.camX -= float64(x-g.dragX) * uiPx / worldPx / g.zoom
			g.camY -= float64(y-g.dragY) * uiPx / worldPx / g.zoom
			g.dragX, g.dragY = x, y
		}
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 && onMap && !locked && (g.zoomTo == g.zoomFrom || g.zoomT >= .5) {
		current := g.zoom
		if g.zoomT < .5 {
			current = g.zoomTo
		}
		steps := g.zoomSteps()
		i := 0
		for j, z := range steps {
			if z <= current+.01 {
				i = j
			}
		}
		z := steps[max(0, min(len(steps)-1, i+int(math.Copysign(1, wheel))))]
		g.setZoom(z)
		g.rememberZoom(z)
	}
	g.camX, g.camY = g.clampCam(g.camX, g.camY)
}

// clampCam is (x, y) moved to where the camera's centre may go. On a battle map that is the battle area's diamond:
// the view reaches half its own size past each edge, so a cell on the edge can be brought out from under the HUD
// panels, and goes no farther, nor past the canvas. Other fields keep the centre on the canvas.
func (g *Game) clampCam(x, y float64) (float64, float64) {
	f := g.field
	if f == nil {
		return x, y
	}
	if _, ok := g.battleBox(); !ok {
		size := f.Size()
		return math.Max(0, math.Min(float64(size.X), x)), math.Max(0, math.Min(float64(size.Y), y))
	}
	u, v := g.cellAt(x, y)
	x, y = f.Center(math.Max(-.5, math.Min(float64(f.Map.W)-.5, u)), math.Max(-.5, math.Min(float64(f.Map.H)-.5, v)))
	// zoomed far out, the view also stays on the canvas
	vp, size := g.battleViewport(), f.Size()
	within := func(c, view, size float64) float64 {
		if view >= size {
			return size / 2
		}
		return math.Max(view/2, math.Min(size-view/2, c))
	}
	return within(x, float64(vp.Dx())/g.zoom, float64(size.X)), within(y, float64(vp.Dy())/g.zoom, float64(size.Y))
}

// camCanGo reports whether the camera can move a little way (dx, dy) from where it is.
func (g *Game) camCanGo(dx, dy float64) bool {
	x, y := g.clampCam(g.camX+dx, g.camY+dy)
	return math.Abs(x-g.camX)+math.Abs(y-g.camY) > .5
}

// onMap reports whether the cursor is over the battlefield and not over a HUD panel.
func (g *Game) onMap() bool {
	x, y := cursorPos()
	wx, wy := cursorWorld()
	return !g.overHUD(x, y) && image.Pt(int(wx), int(wy)).In(g.battleViewport())
}

func (g *Game) overHUD(x, y int) bool {
	for _, r := range g.blocks {
		if image.Pt(x, y).In(r) {
			return true
		}
	}
	return false
}

func cell(im *ebiten.Image, col, row, cols, rows int) *ebiten.Image {
	b := im.Bounds()
	r := image.Rect(col*b.Dx()/cols, row*b.Dy()/rows, (col+1)*b.Dx()/cols, (row+1)*b.Dy()/rows)
	return im.SubImage(r).(*ebiten.Image)
}

func blit(dst, src *ebiten.Image, x, y, w, h float64, tint color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/float64(src.Bounds().Dx()), h/float64(src.Bounds().Dy()))
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	if tint != nil {
		op.ColorScale.ScaleWithColor(tint)
	}
	drawScaled(dst, src, op)
}

// portrait draws a framed portrait of a character (id, display name or troop id) serving as
// class (its own class when empty); someone with no portrait shows only the name plate the
// caller draws.
func (g *Game) portrait(dst *ebiten.Image, ref, class string, x, y, size float64) {
	_, c, ok := g.character(ref)
	src := g.portraitArt(c, class)
	if !ok || src == nil {
		return
	}
	blit(dst, src, x, y, size, size, nil)
	strokeRect(dst, float32(x-2), float32(y-2), float32(size+4), float32(size+4), 2, gold)
}

func (g *Game) syncUnit(u core.UnitView, o core.Observation) {
	v, ok := g.vis[u.ID]
	if !ok {
		dir := "SE"
		if u.Faction != "ally" {
			dir = "NW"
		}
		if u.Facing != "" {
			dir = u.Facing
		}
		v = &unitVis{id: u.ID, ally: u.Faction == "ally", dir: dir, anim: "idle", dying: -1}
		g.vis[u.ID] = v
	}
	if look := u.Class + ":" + u.Weapon; look != v.look {
		v.look = look
		v.art = g.battleClassArt(u.Def, u.Class, u.Weapon, u.Mount, g.unitColor(u))
		v.top = v.art.TopOffset()
	}
	if u.Facing != v.facing {
		v.facing = u.Facing
		if u.Facing != "" {
			v.dir = u.Facing
		}
	}
	v.u, v.v, v.walk = float64(u.X), float64(u.Y), nil
	v.lift = g.field.Lift(u.X, u.Y)
	v.shownHP, v.maxHP = u.HP, u.Stats.MaxHP
	v.shownMP, v.maxMP = u.MP, u.Stats.MaxMP
	v.status = u.Status
	v.inactive = o.Phase != "battle" || !u.ActsIn(o.Turn, o.Army) || u.Done
	v.greyed = u.Done && u.ActsIn(o.Turn, o.Army) && o.Phase == "battle"
	v.ranged = g.S.Data.Classes[u.Class].Attack == content.AttackRanged
	if v.shootClass != u.Shoot {
		v.shootClass, v.shoot = u.Shoot, nil
		if u.Shoot != "" {
			v.shoot = g.shootArt(u.Def, u.Shoot, u.Mount, g.unitColor(u))
		}
	}
	v.siege = slices.Contains(g.S.Data.Classes[u.Class].Tags, content.TagSiege)
	v.mounted = slices.Contains(g.S.Data.Classes[u.Class].Tags, content.TagMounted)
	v.gone = u.HP <= 0
	if !v.gone {
		v.dying, v.leave = -1, nil
		if v.anim == "idle" || v.anim == "exhausted" || v.anim == "walk" {
			v.anim = v.rest()
		}
	}
}
