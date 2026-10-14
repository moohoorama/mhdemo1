package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
)

// viewport is the screen area of the battlefield: the whole window, HUD drawn over it.
var viewport = image.Rect(0, 0, Width, Height)

func (g *Game) busy() bool { return g.player.Busy() || g.sceneBusy() }

// advance moves the clock, the replay and every unit's animation by one frame.
func (g *Game) advance(dt float64) {
	g.uiSeconds += dt
	if g.zoomTo != g.zoomFrom && g.zoomT < .5 {
		g.zoomT = math.Min(.5, g.zoomT+dt)
		t := 1 - math.Pow(2, -g.zoomT*10)
		if g.zoomT == .5 {
			t = 1
		}
		g.zoom = g.zoomFrom + (g.zoomTo-g.zoomFrom)*t
	}
	dt *= float64(g.speed)
	if g.duelPreview != nil {
		dt *= g.previewRate
		if g.previewPaused {
			dt = 0
		}
	}
	g.clockMS += dt * 1000
	g.player.Update(dt)
	for _, u := range g.vis {
		u.update(dt)
	}
	g.updateEffects(dt)
	g.updateScene(dt)
	keptToasts := g.toasts[:0]
	for _, t := range g.toasts {
		t.t += dt / float64(g.speed)
		if t.t < toastTime {
			keptToasts = append(keptToasts, t)
		}
	}
	g.toasts = keptToasts
	for _, m := range g.modals[:min(1, len(g.modals))] {
		if !g.busy() {
			m.t += dt / float64(g.speed)
		}
	}
	kept := g.popups[:0]
	for _, p := range g.popups {
		p.t += dt
		if p.t < .9 {
			kept = append(kept, p)
		}
	}
	g.popups = kept
	g.bannerT = math.Max(0, g.bannerT-dt)
	if !g.busy() && g.S.Engine != nil {
		o := g.S.Engine.Observe()
		if o.Phase == "deployment" || o.Phase == "battle" || o.Phase == "result" {
			g.sync(o)
		}
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
	g.field = NewField(g.assets, g.assets.Maps[o.Map.ID])
	g.vis, g.scene, g.zoom = map[string]*unitVis{}, nil, 1
	g.zoomFrom, g.zoomTo, g.zoomT = 1, 1, 0
	size := g.field.Size()
	g.camX, g.camY = float64(size.X)/2, float64(size.Y)/2
	g.clearOrder()
	g.selected = ""
	g.battleXP, g.startLevel = map[string]int{}, map[string]int{}
	g.resultPage = 0
	for _, c := range o.Characters {
		g.startLevel[c.ID] = c.Level
	}
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

// zooms are the canvas scales the wheel steps through; 1 shows unit pixels 1:1.
var zooms = []float64{.5, 1, 2, 4}

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

// camera pans with the arrow keys, the window's edges or a drag with either button and
// zooms with the wheel. A click without dragging selects (left) or cancels (right) instead.
func (g *Game) camera() {
	if g.field == nil || g.popupPending() || g.input {
		g.dragging = false
		return
	}
	vp := g.battleViewport()
	step := 12.0 / g.zoom
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
	// the window's edge scrolls too (not while a screenshot run drives the game)
	const edge = 8
	if !pointer.used && len(g.Shots) == 0 && !g.dragging && ebiten.IsFocused() && image.Pt(x, y).In(vp) && !g.overHUD(x, y) {
		switch {
		case x < edge:
			g.camX -= step
		case x >= Width-edge:
			g.camX += step
		}
		switch {
		case y < edge:
			g.camY -= step
		case y >= vp.Max.Y-edge:
			g.camY += step
		}
	}
	if mouseJustPressed(ebiten.MouseButtonRight) && image.Pt(x, y).In(vp) && !g.overHUD(x, y) && !g.dragging {
		g.dragging, g.dragMoved, g.dragButton, g.dragX, g.dragY = true, false, ebiten.MouseButtonRight, x, y
	}
	if g.dragging {
		if !mousePressed(g.dragButton) {
			g.dragging = false
			if !g.dragMoved && g.dragButton == ebiten.MouseButtonRight {
				g.rightClick()
			} else if !g.dragMoved {
				g.mapRelease()
			}
		} else if g.dragMoved || abs(x-g.dragX)+abs(y-g.dragY) > 4 {
			g.dragMoved = true
			g.camX -= float64(x-g.dragX) / g.zoom
			g.camY -= float64(y-g.dragY) / g.zoom
			g.dragX, g.dragY = x, y
		}
	}
	if _, wy := ebiten.Wheel(); wy != 0 && image.Pt(x, y).In(vp) && !g.overHUD(x, y) && (g.zoomTo == g.zoomFrom || g.zoomT >= .5) {
		current := g.zoom
		if g.zoomT < .5 {
			current = g.zoomTo
		}
		i := 0
		for j, z := range zooms {
			if z <= current {
				i = j
			}
		}
		g.setZoom(zooms[max(0, min(len(zooms)-1, i+int(math.Copysign(1, wy))))])
	}
	size := g.field.Size()
	g.camX = math.Max(0, math.Min(float64(size.X), g.camX))
	g.camY = math.Max(0, math.Min(float64(size.Y), g.camY))
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

// portrait draws a framed portrait of a character (id, display name or troop id); someone
// without one shows only the name plate the caller draws.
func (g *Game) portrait(dst *ebiten.Image, ref string, x, y, size float64) bool {
	_, c, ok := g.character(ref)
	src := g.assets.Portraits[c.Look.Portrait]
	if !ok || src == nil {
		return false
	}
	blit(dst, src, x, y, size, size, nil)
	strokeRect(dst, float32(x-2), float32(y-2), float32(size+4), float32(size+4), 2, gold)
	return true
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
		art := g.battleClassArt(u.Def, u.Class, u.Color)
		v = &unitVis{id: u.ID, ally: u.Faction == "ally", art: art, dir: dir, anim: "idle", dying: -1, top: art.TopOffset()}
		g.vis[u.ID] = v
	}
	v.u, v.v, v.walk = float64(u.X), float64(u.Y), nil
	v.lift = g.field.Lift(u.X, u.Y)
	v.shownHP, v.maxHP = u.HP, u.Stats.MaxHP
	v.shownMP, v.maxMP = u.MP, u.Stats.MaxMP
	v.status = u.Status
	v.inactive = o.Phase != "battle" || !u.ActsIn(o.Turn) || u.Done
	v.greyed = u.Done && u.ActsIn(o.Turn) && o.Phase == "battle"
	v.ranged = g.S.Data.Classes[u.Class].Attack == content.AttackRanged
	v.gone = u.HP <= 0
	if !v.gone {
		v.dying = -1
		if v.anim == "idle" || v.anim == "exhausted" {
			v.anim = v.rest()
		}
	}
}
