package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// learnWindow is the learning window's state: whose it is, its table and the trait picked.
type learnWindow struct {
	of   string
	list tableList
	pick string
}

// learnRow is one trait the character may learn.
type learnRow struct {
	id, name, kind, requires string
	cost                     int
	ok                       bool // affordable now
}

// drawLearning is the learning window of the first character who has one open: the traits
// they may learn with cost and prerequisites. A row picks a trait and shows it in full below;
// 확인 learns it. Nothing else accepts input until the window is closed.
func (g *Game) drawLearning(dst *ebiten.Image) {
	l := g.learn[0]
	if g.learnWin.of != l.ID {
		g.learnWin = learnWindow{of: l.ID}
	}
	g.buttons, g.blocks = nil, nil
	g.addButton(image.Rect(0, 0, uiW, uiH), "", func() {})
	g.popup(dst, "learn:"+l.ID, 150, Width, Height, image.Rect(Width/2-380, 80, Width/2+380, 780).Inset(-24), func(dst *ebiten.Image) { g.learningWindow(dst, l) })
}

// learnRows are the traits l may learn, cheapest first, then by name.
func (g *Game) learnRows(l core.Learner) []learnRow {
	rows := make([]learnRow, 0, len(l.Options))
	for _, id := range l.Options {
		t := g.S.Data.Traits[id]
		req := make([]string, len(t.Requires))
		for i, q := range t.Requires {
			req[i] = g.traitName(q)
		}
		rows = append(rows, learnRow{id: id, name: t.Name, kind: g.traitKindName(id), requires: strings.Join(req, " · "), cost: t.Cost, ok: t.Cost <= l.Points})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].cost != rows[j].cost {
			return rows[i].cost < rows[j].cost
		}
		return rows[i].name < rows[j].name
	})
	return rows
}

// learnColumns are the learning table's columns: the trait with its icon, its kind, cost and
// prerequisites.
func (g *Game) learnColumns() []column[learnRow] {
	return []column[learnRow]{
		{key: "name", label: "특성", w: 230, text: true, left: true, pad: 30, size: 1,
			get: func(r *learnRow) (string, float64) { return r.name, 0 },
			after: func(dst *ebiten.Image, r *learnRow, cell image.Rectangle) {
				g.traitIconOf(dst, r.id, float64(cell.Min.X+6), float64(cell.Min.Y+cell.Dy()/2-11), 22)
			}},
		{key: "kind", label: "종류", w: 130, text: true, get: func(r *learnRow) (string, float64) { return r.kind, 0 }},
		{key: "cost", label: "특성치", w: 90, get: func(r *learnRow) (string, float64) { return fmt.Sprint(r.cost), float64(r.cost) }},
		{key: "requires", label: "선행", text: true, left: true,
			get: func(r *learnRow) (string, float64) {
				if r.requires == "" {
					return "—", 0
				}
				return r.requires, 0
			}},
	}
}

// learningWindow draws the learning window, laid out on the Width×Height design.
func (g *Game) learningWindow(dst *ebiten.Image, l core.Learner) {
	w := &g.learnWin
	r := image.Rect(uiW/2-380, 80, uiW/2+380, 780)
	g.titled(dst, r, "특성 학습")
	g.portrait(dst, l.ID, "", float64(r.Min.X+32), float64(r.Min.Y+44), 80)
	g.label(dst, l.Name, float64(r.Min.X+132), float64(r.Min.Y+46), 26, gold)
	g.label(dst, fmt.Sprintf("남은 특성치 %d", l.Points), float64(r.Min.X+132), float64(r.Min.Y+84), 17, ink)
	if len(g.learn) > 1 {
		g.label(dst, fmt.Sprintf("다음 대기 %d명", len(g.learn)-1), float64(r.Max.X-180), float64(r.Min.Y+86), 13, muted)
	}

	rows := g.learnRows(l)
	at := slices.IndexFunc(rows, func(row learnRow) bool { return row.id == w.pick })
	var picked *learnRow
	if at >= 0 {
		row := rows[at]
		picked = &row
	} else {
		w.pick = ""
	}
	drawTable(g, dst, &w.list, table[learnRow]{
		cols: g.learnColumns(), rowH: 36, headH: 32, size: 15, center: true,
		empty:    "지금 익힐 수 있는 특성이 없습니다.",
		rank:     func(row *learnRow) int { return b2i(!row.ok) },
		dim:      func(row *learnRow) bool { return !row.ok },
		selected: func(row *learnRow) bool { return row.id == w.pick },
		pick:     func(row *learnRow) { w.pick = row.id },
	}, rows, image.Rect(r.Min.X+28, r.Min.Y+136, r.Max.X-28, r.Max.Y-226))

	// the trait picked, in full
	box := image.Rect(r.Min.X+28, r.Max.Y-218, r.Max.X-28, r.Max.Y-72)
	rect(dst, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), color.NRGBA{R: 255, G: 240, B: 210, A: 12})
	var learn func()
	if picked == nil {
		g.label(dst, "특성을 누르면 설명이 여기에 나옵니다.", float64(box.Min.X+16), float64(box.Min.Y+16), 14, muted)
	} else {
		x, y := float64(box.Min.X+16), float64(box.Min.Y+14)
		g.traitIconOf(dst, picked.id, x, y, 32)
		g.label(dst, picked.name, x+44, y+2, 20, gold)
		g.label(dst, picked.kind, x+52+g.width(picked.name, 20), y+8, 13, muted)
		left := l.Points - picked.cost
		msg, c := fmt.Sprintf("배우면 남는 특성치 %d → %d", l.Points, left), color.Color(ink)
		if !picked.ok {
			msg, c = fmt.Sprintf("특성치가 %d 모자랍니다", -left), bad
		}
		g.label(dst, msg, float64(box.Max.X-16)-g.width(msg, 14), y+6, 14, c)
		desc := g.S.Data.Traits[picked.id].Desc
		if picked.requires != "" {
			desc += "\n선행: " + picked.requires
		}
		g.label(dst, wrapWidth(desc, g.face(15), float64(box.Dx()-32)), x, y+46, 15, ink)
		if picked.ok {
			id := picked.id
			learn = func() {
				g.command(core.Command{Kind: "learn", Actor: l.ID, Trait: id})
				w.pick = ""
			}
		}
	}

	g.btn(dst, image.Rect(r.Max.X-368, r.Max.Y-58, r.Max.X-208, r.Max.Y-24), "확인", learn)
	g.btn(dst, image.Rect(r.Max.X-188, r.Max.Y-58, r.Max.X-28, r.Max.Y-24), "닫기", func() {
		g.learnWin = learnWindow{}
		g.command(core.Command{Kind: "learn_close", Actor: l.ID})
	})
}

// updateLearnList scrolls the learning table by the wheel.
func (g *Game) updateLearnList() bool {
	if len(g.learn) == 0 {
		return false
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.learnWin.list.top -= int(wheel)
		return true
	}
	return false
}
