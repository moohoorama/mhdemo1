package gui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/duelview"
)

func (g *Game) duelView(e core.Event) *duelview.View {
	v := &duelview.View{Bubble: g.assets.Bubble, Font: g.font, Story: g.assets.Duels.Story(e.DuelID, e.Outcome)}
	for i, id := range []string{e.Actor, e.Target} {
		u := g.unitView(g.player.final, id)
		if u == nil {
			continue
		}
		v.Fighters[i] = g.duelFighter(u.Def, u.Class, u.Color)
	}
	if a, t := g.unitView(g.player.final, e.Actor), g.unitView(g.player.final, e.Target); a != nil && t != nil {
		v.Axis = duelAxis(t.X-a.X, t.Y-a.Y)
	}
	for i, id := range g.assets.Duels[e.DuelID].Supports {
		c := g.S.Data.Characters[id]
		v.Fighters[i+2] = g.duelFighter(id, c.Class, c.Look.Faction)
		if u := g.unitView(g.player.final, id); u != nil {
			v.Fighters[i+2] = g.duelFighter(u.Def, u.Class, u.Color)
		}
	}
	if a, t := g.unitView(g.player.final, e.Actor), g.unitView(g.player.final, e.Target); a != nil && t != nil && g.field != nil {
		v.Floor, v.Staged = g.floorImage(t.X, t.Y), true
		ax, ay := g.field.Center(float64(a.X), float64(a.Y))
		tx, ty := g.field.Center(float64(t.X), float64(t.Y))
		v.Focus = [2]float64{(ax + tx) / 2, (ay + ty) / 2}
	}
	return v
}

// duelAxis names the screen direction of a map offset on the (u-v, u+v) isometric grid.
func duelAxis(dx, dy int) string {
	axis := "S"
	if dx+dy < 0 {
		axis = "N"
	}
	if dx-dy < 0 {
		return axis + "W"
	}
	return axis + "E"
}

// SkipDuel finishes only the presentation, leaving its aftermath queued.
func (p *Player) SkipDuel() {
	if p.Duel == nil {
		return
	}
	p.Duel.Finish()
	// A duel is a replay barrier: it is the sole running action.
	for _, r := range p.running {
		r.left = min(r.left, duelview.Transition.Intro())
	}
}

func (g *Game) updateDuel() {
	if keyJustPressed(ebiten.KeyEnter) || keyJustPressed(ebiten.KeySpace) || keyJustPressed(ebiten.KeyEscape) {
		g.player.SkipDuel()
	}
	if keyJustPressed(ebiten.KeyF) {
		g.speed = nextSpeed[g.speed]
	}
	if keyJustPressed(ebiten.KeyF12) {
		g.capture = true
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		x, y := cursorPos()
		for _, b := range g.buttons {
			if image.Pt(x, y).In(b.rect) {
				b.click()
				break
			}
		}
	}
}

func (g *Game) drawDuel(dst *ebiten.Image) {
	v := g.player.Duel
	pan, _, ui := v.Phase()
	if !g.duelCamHeld {
		g.duelCam, g.duelCamHeld = [2]float64{g.camX, g.camY}, true
	}
	if v.Cover() < 1 {
		g.camX = g.duelCam[0] + (v.Focus[0]-g.duelCam[0])*pan
		g.camY = g.duelCam[1] + (v.Focus[1]-g.duelCam[1])*pan
		b := dst.Bounds()
		if g.duelMap == nil || g.duelMap.Bounds() != b {
			g.duelMap, g.duelUI = ebiten.NewImage(b.Dx(), b.Dy()), ebiten.NewImage(b.Dx(), b.Dy())
		}
		g.duelMap.Fill(color.NRGBA{R: 16, G: 13, B: 12, A: 255})
		g.layer = layerMap
		g.play(g.duelMap)
		g.duelUI.Clear()
		if pan < 1 {
			g.layer = layerUI
			g.play(g.duelUI)
		}
		g.layer = layerAll
		g.buttons, g.blocks = nil, nil
		v.Compose(dst, g.duelMap, g.duelUI)
		g.drawDuelPreviewInfo(dst)
		return
	}
	v.Draw(dst)
	g.drawDuelPreviewInfo(dst)
	if ui >= 1 {
		g.btn(dst, image.Rect(1000, 805, 1254, 839), "일기토 건너뛰기", g.player.SkipDuel)
	}
}

// floorImage is the arena ground of the cell the defender stands on, at the duel's pixel
// scale: still water on a water cell, the battle map's grass pattern anywhere else. Structures
// and props are left out.
func (g *Game) floorImage(u, v int) *ebiten.Image {
	floor := ebiten.NewImage(Width, Height)
	if g.field.small(2*u, 2*v) != '~' {
		pattern := g.assets.pattern()
		w, h := float64(pattern.Bounds().Dx())*3, float64(pattern.Bounds().Dy())*3
		for y := 0.0; y < Height; y += h {
			for x := 0.0; x < Width; x += w {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(3, 3)
				op.GeoM.Translate(x, y)
				floor.DrawImage(pattern, op)
			}
		}
		return floor
	}
	s := g.assets.Tiles[g.assets.Animations["water"].Frames[0]]
	// Small tiles are half a cell apart: (8, 4) map pixels, 3x on screen.
	for i := -60; i <= 60; i++ {
		for j := -60; j <= 60; j++ {
			x, y := 640+float64(i-j)*24, 440+float64(i+j)*12
			if x < -48 || x > Width+48 || y < -24 || y > Height+24 {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(3, 3)
			op.GeoM.Translate(x-s.PivotX*3, y-s.PivotY*3-12)
			floor.DrawImage(s.Image, op)
		}
	}
	return floor
}

func (g *Game) duelFighter(id, classID, faction string) duelview.Fighter {
	c := g.S.Data.Characters[id]
	class := g.S.Data.Classes[classID]
	return duelview.Fighter{Name: c.Name, Class: class.Name, Art: g.battleClassArt(id, classID, faction), Troops: g.art(class.Sprite, faction, content.Look{}), Portrait: g.assets.Portraits[c.Look.Portrait]}
}
