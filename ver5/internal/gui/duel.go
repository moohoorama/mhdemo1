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
		v.Floor = g.floorTiles(t.X, t.Y)
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
	pan, _ := v.Phase()
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
	g.btn(dst, image.Rect(1000, 805, 1254, 839), "일기토 건너뛰기", g.player.SkipDuel)
}

// floorTiles are the ground layers of a map cell without structures or props: water
// shows its first frame, and cells with no tile layers (native maps) use the ground pattern.
func (g *Game) floorTiles(u, v int) []duelview.Tile {
	x, y := (u-v)*16, (u+v+1)*8
	layers := []int{23}
	for _, t := range g.field.Map.Ground {
		if t.X == x && t.Y == y {
			layers = t.Layers
			break
		}
	}
	var tiles []duelview.Tile
	for _, id := range layers {
		if id == 0 {
			id = g.assets.Animations["water"].Frames[0]
		}
		s := g.assets.Tiles[id]
		tiles = append(tiles, duelview.Tile{Image: s.Image, PivotX: s.PivotX, PivotY: s.PivotY})
	}
	return tiles
}

func (g *Game) duelFighter(id, classID, faction string) duelview.Fighter {
	c := g.S.Data.Characters[id]
	class := g.S.Data.Classes[classID]
	return duelview.Fighter{Name: c.Name, Class: class.Name, Art: g.battleClassArt(id, classID, faction), Troops: g.art(class.Sprite, faction, content.Look{}), Portrait: g.assets.Portraits[c.Look.Portrait]}
}
