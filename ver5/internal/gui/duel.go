package gui

import (
	"image"

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
	p.Duel.Time = p.Duel.Story.Duration()
	// A duel is a replay barrier: it is the sole running action.
	for _, r := range p.running {
		r.left = 0
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
	g.player.Duel.Draw(dst)
	g.btn(dst, image.Rect(1000, 805, 1254, 839), "일기토 건너뛰기", g.player.SkipDuel)
}

func (g *Game) duelFighter(id, classID, faction string) duelview.Fighter {
	c := g.S.Data.Characters[id]
	class := g.S.Data.Classes[classID]
	return duelview.Fighter{Name: c.Name, Class: class.Name, Art: g.battleClassArt(id, classID, faction), Troops: g.art(class.Sprite, faction, content.Look{}), Portrait: g.assets.Portraits[c.Look.Portrait]}
}
