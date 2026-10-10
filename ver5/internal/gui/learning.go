package gui

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

const learnRows = 7

// drawLearning is the learning window of the first character who has one open: the traits
// they may learn with cost and prerequisites. Affordable ones can be picked; nothing else
// accepts input until the window is closed.
func (g *Game) drawLearning(dst *ebiten.Image) {
	l := g.learn[0]
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 150})
	g.buttons, g.blocks = nil, nil
	g.buttons = append(g.buttons, button{image.Rect(0, 0, Width, Height), "", func() {}})
	r := image.Rect(Width/2-380, 80, Width/2+380, 780)
	g.titled(dst, r, "특성 학습")
	g.portrait(dst, l.ID, float64(r.Min.X+32), float64(r.Min.Y+44), 80)
	g.label(dst, l.Name, float64(r.Min.X+132), float64(r.Min.Y+46), 26, gold)
	g.label(dst, fmt.Sprintf("남은 특성치 %d", l.Points), float64(r.Min.X+132), float64(r.Min.Y+84), 17, ink)
	if len(g.learn) > 1 {
		g.label(dst, fmt.Sprintf("다음 대기 %d명", len(g.learn)-1), float64(r.Max.X-180), float64(r.Min.Y+86), 13, muted)
	}
	options := append([]string(nil), l.Options...)
	sort.SliceStable(options, func(i, j int) bool {
		a, b := g.S.Data.Traits[options[i]], g.S.Data.Traits[options[j]]
		if a.Cost != b.Cost {
			return a.Cost < b.Cost
		}
		return a.Name < b.Name
	})
	g.learnPage = max(0, min(g.learnPage, (len(options)-1)/learnRows))
	y := r.Min.Y + 136
	for _, id := range options[min(len(options), g.learnPage*learnRows):min(len(options), (g.learnPage+1)*learnRows)] {
		t := g.S.Data.Traits[id]
		row := image.Rect(r.Min.X+28, y, r.Max.X-28, y+68)
		var click func()
		c := color.Color(ink)
		if t.Cost <= l.Points {
			trait := id
			click = func() { g.command(core.Command{Kind: "learn", Actor: l.ID, Trait: trait}) }
		} else {
			c = muted
		}
		g.btn(dst, row, "", click)
		g.label(dst, t.Name, float64(row.Min.X+14), float64(row.Min.Y+6), 18, c)
		cost := fmt.Sprintf("특성치 %d", t.Cost)
		g.label(dst, cost, float64(row.Max.X-14)-g.width(cost, 15), float64(row.Min.Y+8), 15, c)
		desc := t.Desc
		if len(t.Requires) > 0 {
			req := make([]string, len(t.Requires))
			for i, q := range t.Requires {
				req[i] = g.traitName(q)
			}
			desc += "  [선행: " + strings.Join(req, " · ") + "]"
		}
		g.label(dst, wrap(desc, 62), float64(row.Min.X+14), float64(row.Min.Y+34), 13, muted)
		y += 72
	}
	if len(options) == 0 {
		g.label(dst, "지금 익힐 수 있는 특성이 없습니다.", float64(r.Min.X+40), float64(r.Min.Y+150), 17, muted)
	}
	pages := (len(options) + learnRows - 1) / learnRows
	if pages > 1 {
		g.btn(dst, image.Rect(r.Min.X+28, r.Max.Y-58, r.Min.X+148, r.Max.Y-24), "이전", func() { g.learnPage-- })
		g.btn(dst, image.Rect(r.Min.X+160, r.Max.Y-58, r.Min.X+280, r.Max.Y-24), "다음", func() { g.learnPage++ })
		g.label(dst, fmt.Sprintf("%d / %d", g.learnPage+1, pages), float64(r.Min.X+296), float64(r.Max.Y-50), 15, muted)
	}
	g.btn(dst, image.Rect(r.Max.X-188, r.Max.Y-58, r.Max.X-28, r.Max.Y-24), "닫기", func() {
		g.learnPage = 0
		g.command(core.Command{Kind: "learn_close", Actor: l.ID})
	})
}
