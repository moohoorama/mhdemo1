package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"srpg/internal/core"
)

// viewport is the screen area of the battlefield: the whole window, HUD drawn over it.
var viewport = image.Rect(0, 0, Width, Height)

// stageFaction colours each battle's enemies (unit team keys swapped to the faction ramp).
var stageFaction = map[string]string{"B01": "turban", "B02": "dong", "B03": "dong"}

// heroArt are officers with their own sprite; everyone else uses their class's.
var heroArt = map[string]string{"유비": "liubei", "관우": "guanyu", "장비": "zhangfei", "간옹": "jianyong",
	"장각": "zhangjiao", "화웅": "huaxiong", "여포": "lubu"}

func classArt(class string) string {
	switch class {
	case "경기병", "중기병", "친위대":
		return "cavalry"
	case "궁병", "노병", "연노병":
		return "archer"
	case "사마", "참모", "군사":
		return "strategist"
	case "적병", "흉적", "의적":
		return "bandit"
	}
	return "infantry"
}

func (g *Game) busy() bool { return g.player.Busy() || g.sceneBusy() }

// advance moves the clock, the replay and every unit's animation by one frame.
func (g *Game) advance(dt float64) {
	dt *= float64(g.speed)
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
		if o.Phase == "battle" || o.Phase == "result" {
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
	g.field = NewField(g.assets, g.assets.Maps[o.Map.ID])
	g.vis, g.scene, g.zoom = map[string]*unitVis{}, nil, 1
	size := g.field.Size()
	g.camX, g.camY = float64(size.X)/2, float64(size.Y)/2
	g.clearOrder()
	g.selected = ""
	g.battleXP, g.startLevel = map[string]int{}, map[string]int{}
	for _, of := range o.Officers {
		g.startLevel[of.ID] = of.Level
	}
	if o.Phase == "battle" && o.Round == 1 && o.Turn == "ally" {
		g.modals = append(g.modals, &modal{kind: "objective"})
	}
	g.sync(o)
}

// sync makes the shown units match the rules state (between replays, or after one).
func (g *Game) sync(o core.Observation) {
	if o.Map == nil || g.field == nil || g.field.Map.ID != o.Map.ID {
		return
	}
	for _, u := range o.UnitViews {
		if g.ord.move != nil && u.ID == g.ord.actor {
			continue // shown at its tentative cell until the order is confirmed or cancelled
		}
		v, ok := g.vis[u.ID]
		if !ok {
			key := heroArt[u.Officer]
			if key == "" {
				key = classArt(u.Class)
			}
			faction := "shu"
			dir := "SE"
			if u.Faction != "ally" {
				faction, dir = stageFaction[o.Map.ID], "NW"
			}
			art := g.assets.Units[key]
			v = &unitVis{id: u.ID, faction: faction, ally: u.Faction == "ally", art: art, dir: dir, anim: "idle",
				dying: -1, top: art.TopOffset()}
			g.vis[u.ID] = v
		}
		v.u, v.v, v.walk = float64(u.X), float64(u.Y), nil
		v.lift = g.field.Lift(u.X, u.Y)
		v.shownHP, v.maxHP = u.HP, u.Stats.MaxHP
		v.greyed = u.Done && u.Faction == o.Turn && o.Phase == "battle"
		v.ranged = g.S.Data.Classes[u.Class].Weapon == "활"
		v.gone = u.HP <= 0
		if !v.gone {
			v.dying = -1
			if v.anim == "idle" || v.anim == "exhausted" {
				v.anim = v.rest()
			}
		}
	}
}

// zooms are the canvas scales the wheel steps through; 1 shows unit pixels 1:1.
var zooms = []float64{.5, 1, 1.5, 2}

func (g *Game) toCanvas(x, y float64) (float64, float64) {
	z := g.zoom
	return (x-float64(viewport.Min.X+viewport.Dx()/2))/z + g.camX, (y-float64(viewport.Min.Y+viewport.Dy()/2))/z + g.camY
}

// toScreen maps canvas pixels to the viewport sub-image (which keeps screen coordinates).
func (g *Game) toScreen(x, y float64) (float64, float64) {
	z := g.zoom
	return (x-g.camX)*z + float64(viewport.Min.X+viewport.Dx()/2), (y-g.camY)*z + float64(viewport.Min.Y+viewport.Dy()/2)
}

// camera pans with the arrow keys, the window's edges or a drag with either button and
// zooms with the wheel. A click without dragging selects (left) or cancels (right) instead.
func (g *Game) camera() {
	if g.field == nil || g.overlay != "" || g.input || len(g.modals) > 0 {
		g.dragging = false
		return
	}
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
	if len(g.Shots) == 0 && !g.dragging && ebiten.IsFocused() && image.Pt(x, y).In(viewport) {
		switch {
		case x < edge:
			g.camX -= step
		case x >= Width-edge:
			g.camX += step
		}
		switch {
		case y < edge:
			g.camY -= step
		case y >= Height-edge:
			g.camY += step
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && image.Pt(x, y).In(viewport) && !g.dragging {
		g.dragging, g.dragMoved, g.dragButton, g.dragX, g.dragY = true, false, ebiten.MouseButtonRight, x, y
	}
	if g.dragging {
		if !ebiten.IsMouseButtonPressed(g.dragButton) {
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
	if _, wy := ebiten.Wheel(); wy != 0 && image.Pt(x, y).In(viewport) && !g.overHUD(x, y) {
		i := 0
		for j, z := range zooms {
			if z <= g.zoom {
				i = j
			}
		}
		g.zoom = zooms[max(0, min(len(zooms)-1, i+int(math.Copysign(1, wy))))]
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

func (g *Game) portrait(dst *ebiten.Image, id string, x, y, size float64) {
	col, known := map[string]int{"유비": 0, "관우": 1, "장비": 2, "간옹": 3}[id]
	sheet := g.assets.Portraits
	if !known {
		sheet = g.assets.Enemies
		col = 3
		if c, ok := map[string]int{"여포": 0, "화웅": 1, "장각": 2}[id]; ok {
			col = c
		}
	}
	src := cell(sheet, col, 0, 4, 1)
	b := src.Bounds()
	// Square viewport crops the lower chest rather than distorting the face.
	src = src.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+b.Dx())).(*ebiten.Image)
	blit(dst, src, x, y, size, size, nil)
	strokeRect(dst, float32(x-2), float32(y-2), float32(size+4), float32(size+4), 2, gold)
}
