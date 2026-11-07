package gui

import (
	"fmt"
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// levelUpBox is the level-up window on the Width×Height design.
var levelUpBox = image.Rect(Width/2-320, 250, Width/2+320, 600)

// levelUpChoice tells whether the level-up of m leaves the officer something to learn now: the window then asks
// 나중에 or 특성 배우기 instead of closing on any click.
func (g *Game) levelUpChoice(m *modal) bool {
	return slices.ContainsFunc(g.S.View().Learners, func(l core.Learner) bool { return l.ID == m.level.id })
}

// levelUpWindow is a level-up: who, the levels and the points gained, the stats with what they grew by, and the
// traits the officer may learn now as the same chips as the trait shelves. With points to spend it ends in
// 특성 배우기, which opens the officer's trait tab, beside 닫기, which leaves the points for the walk or the next
// preparation.
func (g *Game) levelUpWindow(dst *ebiten.Image, m *modal) {
	l := m.level
	r := levelUpBox
	g.window(dst, r)
	x, y := float64(r.Min.X+20), float64(r.Min.Y+18)
	g.portrait(dst, l.id, "", x, y, 64)
	g.label(dst, g.name(l.id), x+80, y-2, 20, gold)
	lv := fmt.Sprintf("Lv %d → %d", l.from, l.to)
	g.label(dst, lv, x+90+g.width(g.name(l.id), 20), y+3, 15, ink)
	of := g.officer(g.S.View(), l.id)
	gain := 100 * (l.to - l.from)
	if of != nil {
		g.label(dst, fmt.Sprintf("특성치 %d → %d", of.Points-gain, of.Points), x+80, y+28, 15, good)
	}
	k := min(1, m.t/.8) * l.progress / 100
	bar(dst, float32(x+80), float32(y+56), 200, 8, k, xpC)
	g.label(dst, fmt.Sprintf("경험치 %.0f / 100", l.progress), x+290, y+50, 12, muted)
	y += 84
	colW := float64(r.Dx()-40) / 3
	for i, s := range statRows {
		a, b := s.get(l.before), s.get(l.after)
		px, py := x+float64(i%3)*colW, y+float64(i/3)*26
		g.label(dst, s.name, px, py, 14, muted)
		g.label(dst, fmt.Sprint(b), px+58, py, 14, ink)
		if b != a {
			g.label(dst, fmt.Sprintf("+%d", b-a), px+112, py, 14, good)
		}
	}
	y += 3*26 + 8
	rect(dst, float32(x), float32(y), float32(r.Dx()-40), 1, goldDim)
	y += 8
	if of == nil {
		return
	}
	open := []traitEntry{}
	for _, sh := range g.shelvesOf(of)[:2] {
		for _, e := range sh.traits {
			if !e.have {
				open = append(open, e)
			}
		}
	}
	if len(open) == 0 {
		g.label(dst, "더 배울 특성이 없습니다.", x, y+6, 13, muted)
	} else {
		g.label(dst, "지금 배울 수 있는 특성", x, y, 13, muted)
		cx, cy := int(x), int(y)+16
		for _, e := range open {
			if cx+g.traitChipW(e) > r.Max.X-20 {
				if cy+2*touchH > r.Max.Y-touchH-16 {
					break
				}
				cx, cy = int(x), cy+touchH
			}
			cx += g.traitChip(dst, cx, cy, e, false, nil) + 6
		}
	}
	if !g.levelUpChoice(m) {
		g.closeRow(dst, r.Max.X-16, r.Max.Y-touchH-12, g.closeModal)
		return
	}
	id := l.id
	later := func() { // closing puts the learning off to the walk or the next preparation
		g.closeModal()
		g.command(core.Command{Kind: "learn_close", Actor: id})
	}
	g.closeRow(dst, r.Max.X-16, r.Max.Y-touchH-12, later, action{"특성 배우기 ›", true, g.closeModal})
}
