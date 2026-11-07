package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// briefH is the 장수 tab's height: the selected unit in short.
const briefH = 290

// hudBrief is the selected unit in short for the HUD's 장수 tab: who it is with the deputy's face, its troops and
// strategy points, the hexagon against the battle's best (the lead alone dashed in gold) with its traits beside
// it, and 자세히 보기, which opens the unit's detail to look at.
func (g *Game) hudBrief(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	u := g.detailUnit(o)
	if u == nil {
		return
	}
	if g.ord.sandbox != nil {
		o = g.view()
	}
	x, y := float64(r.Min.X+12), float64(r.Min.Y+8)
	g.portrait(dst, u.ID, u.Class, x, y, 40)
	g.label(dst, u.Name, x+50, y-1, 16, teamColor(u))
	g.label(dst, fmt.Sprintf("%s · Lv%d", g.className(u.Class), u.Level), x+50, y+22, 12, muted)
	of := g.officer(o, u.Character)
	if of != nil && of.Deputy != "" {
		dx := float64(r.Max.X-14) - g.width(g.name(of.Deputy), 12) - 34
		if d := g.officer(o, of.Deputy); d != nil {
			g.portrait(dst, d.ID, d.Class, dx, y+4, 28)
		}
		g.label(dst, "부관", dx+34, y+2, 12, muted)
		g.label(dst, g.name(of.Deputy), dx+34, y+18, 12, ink)
	}
	y += 50
	for _, b := range []struct {
		name   string
		v, max int
		c      color.Color
	}{{"병력", u.HP, u.Stats.MaxHP, hpColor(float64(u.HP) / float64(max(1, u.Stats.MaxHP)))}, {"병법치", u.MP, u.Stats.MaxMP, mpC}} {
		g.label(dst, b.name, x, y, 12, muted)
		bar(dst, float32(x+48), float32(y+5), float32(r.Dx()-170), 8, float64(b.v)/float64(max(1, b.max)), b.c)
		s := fmt.Sprintf("%d/%d", b.v, b.max)
		g.label(dst, s, float64(r.Max.X-12)-g.width(s, 12), y, 12, ink)
		y += 20
	}
	y += 4
	var best [6]int
	for _, v := range o.UnitViews {
		for i, a := range hexAxes {
			best[i] = max(best[i], a.get(v.Stats))
		}
	}
	room := float64(r.Max.Y) - y - touchH - 12
	radius := max(20, min(38, (room-30)/2))
	var alone *core.Stats
	if of != nil && of.Deputy != "" {
		if s, _, err := g.S.Engine.DeputyStats(of.ID, ""); err == nil {
			alone = &s
		}
	}
	g.hexChart(dst, x+radius+52, y+radius+16, radius, u.Stats, alone, best)
	tx, ty := x+2*radius+120, y+2
	traits := slices.Clone(u.Traits)
	g.label(dst, fmt.Sprintf("특성 %d", len(traits)), tx, ty, 12, muted)
	ty += 18
	for _, t := range traits {
		if ty+18 > y+room {
			break
		}
		g.traitIconOf(dst, t, tx, ty, 18)
		g.label(dst, ellipsis(g, g.traitName(t), 12, int(float64(r.Max.X)-tx-30)), tx+22, ty, 12, ink)
		ty += 20
	}
	var more func()
	if of != nil && g.S.Data.Characters[of.Def].Playable {
		id := of.ID
		more = func() {
			g.selected = id
			g.openUnit(true)
		}
	}
	g.mBtn(dst, image.Rect(r.Min.X+10, r.Max.Y-touchH-8, r.Max.X-10, r.Max.Y-8), "자세히 보기 ›", false, more)
}
