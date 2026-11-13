package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// officerTabs are the tabs of an officer's detail.
var officerTabs = []string{"열전", "특성", "장비"}

// partyWindow is the officers' window of the walk, filling the screen: the officers listed on the left and the
// picked one on the right in three tabs (열전, 특성, 장비), where traits are learned and equipment changed.
// It can show the officers as a table instead, or the belongings.
func (g *Game) partyWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.View()
	p := &g.party
	roster := g.roster(o)
	if p.learning { // a level-up's learning lists only the officers waiting to learn
		waiting := slices.DeleteFunc(slices.Clone(roster), func(c core.Character) bool {
			return !slices.ContainsFunc(g.learn, func(l core.Learner) bool { return l.ID == c.ID })
		})
		if len(waiting) > 0 {
			roster = waiting
		}
	}
	if !containsID(roster, g.selected) && len(roster) > 0 {
		g.selected = roster[0].ID
	}
	// one switch at the top left for the three views: the list beside the picked officer, the table, the belongings
	view := 0
	switch {
	case p.items:
		view = 2
	case p.table:
		view = 1
	}
	viewsEnd := g.tabRow(dst, x+8, y+4, []string{"목록", "표", "소지품"}, view, func(i int) { p.table, p.items, p.insu = i == 1, i == 2, insuState{} })
	_, gearOwner, _, guide := g.gearStep()
	guide = guide && g.overlay() == "party"
	if guide && view != 0 {
		g.guideRim(dst, g.tabRect(x+8, y+4, []string{"목록", "표", "소지품"}, 0))
	}
	switch {
	case p.items:
		rect(dst, float32(x+8), float32(y+52), float32(w-16), 1, goldDim)
		g.moneyLabel(dst, float64(x+w-16), float64(y+16), 15)
		if p.insu.item != "" {
			g.insuPicker(dst, o, x, y+64, w, h-64-64)
			g.closeRow(dst, x+w-16, y+h-60, func() { p.insu = insuState{} }, g.insuActs()...)
			return
		}
		g.belongingsTab(dst, o, x, y+64, w, h-64-64)
		g.closeRow(dst, x+w-16, y+h-60, g.closeOverlay)
		return
	case p.table:
		g.officerTable(dst, o, roster, image.Rect(x, y, x+w, y+h), viewsEnd)
		g.closeRow(dst, x+w-16, y+h-60, g.closeOverlay)
		return
	}
	listW := 300
	if w >= 1100 {
		listW = 340
	}
	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-16), goldDim)
	rect(dst, float32(x+8), float32(y+52), float32(listW-16), 1, goldDim)
	g.label(dst, fmt.Sprintf("%d명", len(roster)), float64(viewsEnd+10), float64(y+19), 13, muted)
	g.listRows(dst, &p.officers, image.Rect(x+6, y+54, x+listW-4, y+h-6), len(roster), listRowH, func(i int, rr image.Rectangle) {
		c := roster[i]
		note, noteC := fmt.Sprintf("특성치 %d", c.Points), muted
		if g.hasAffordable(c) {
			noteC = good
		}
		g.officerRow(dst, rr, c, c.ID == g.selected, note, noteC)
		if guide && c.ID == gearOwner && c.ID != g.selected {
			g.guideRim(dst, rr.Inset(4))
		}
	}, func(i int) { g.selected, p.trait = roster[i].ID, "" })

	var of *core.Character
	for i := range o.Characters {
		if o.Characters[i].ID == g.selected {
			of = &o.Characters[i]
		}
	}
	if of == nil {
		return
	}
	rx := x + listW + 16
	nx := float64(rx)
	if p.learning {
		g.tag(dst, "레벨 업", nx, float64(y+14))
		nx += g.width("레벨 업", 13) + 30
	}
	g.label(dst, g.name(of.ID), nx, float64(y+12), 18, gold)
	g.tabRow(dst, int(nx+g.width(g.name(of.ID), 18))+18, y+4, officerTabs, p.tab, func(i int) { p.tab = i })
	if guide && of.ID == gearOwner && p.tab != 2 {
		g.guideRim(dst, g.tabRect(int(nx+g.width(g.name(of.ID), 18))+18, y+4, officerTabs, 2))
	}
	rect(dst, float32(rx), float32(y+52), float32(x+w-16-rx), 1, goldDim)
	body := image.Rect(rx, y+60, x+w-16, y+h-64)
	var acts []action
	switch p.tab {
	case 0:
		g.officerBio(dst, o, of, body)
	case 1:
		acts = append(acts, g.officerTraits(dst, of, body)...)
	case 2:
		g.officerEquipment(dst, of, body)
	}
	g.closeRow(dst, x+w-16, y+h-60, g.closeOverlay, acts...)
}

// tag is a small red plate with a label, as the window titles are.
func (g *Game) tag(dst *ebiten.Image, s string, x, y float64) {
	w := g.width(s, 13) + 20
	rect(dst, float32(x), float32(y), float32(w), 24, color.NRGBA{R: 108, G: 30, B: 26, A: 255})
	strokeRect(dst, float32(x)+.5, float32(y)+.5, float32(w-1), 23, 1, gold)
	g.label(dst, s, x+10, y+3, 13, ink)
}

// hasAffordable tells whether officer c could learn a trait with the points they have.
func (g *Game) hasAffordable(c core.Character) bool {
	for _, id := range g.S.Engine.Learnable(c.ID) {
		if g.S.Data.Traits[id].Cost <= c.Points {
			return true
		}
	}
	return false
}

var attrNames = []string{"통솔", "무력", "지력", "민첩", "운", "매력"}

// measureLines is the width and height of text s at size.
func measureLines(g *Game, s string, size float64) (float64, float64) {
	n := 1
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return g.width(s, size), float64(n) * size * 1.45
}

// unitEquipment is the unit detail's 장비 tab: a card for each slot with the piece, what it adds and the traits it
// gives; pressing a card opens that slot's picker.
func (g *Game) unitEquipment(dst *ebiten.Image, of *core.Character, r image.Rectangle) {
	cw := (r.Dx() - 16) / 3
	for i, slot := range equipSlots {
		c := image.Rect(r.Min.X+i*(cw+8), r.Min.Y, r.Min.X+i*(cw+8)+cw, r.Min.Y+190)
		g.card(dst, c)
		slot := slot
		g.addButton(c, slotNames[slot], func() {
			g.openOverlay("equipment")
			g.openPicker(slot)
		})
		if hovered(c) {
			strokeRect(dst, float32(c.Min.X)+.5, float32(c.Min.Y)+.5, float32(c.Dx()-1), float32(c.Dy()-1), 1, gold)
		}
		x, y := float64(c.Min.X+14), float64(c.Min.Y+14)
		id, worn := of.Equipment[slot]
		if !worn {
			g.itemIcon(dst, slotIcons[slot], x+3, y+3, 1.25, true)
			g.label(dst, "비어 있음", x, y+56, 16, muted)
			g.label(dst, slotNames[slot], x, y+80, 12, muted)
			continue
		}
		eq := g.S.Data.Equipment[id]
		g.itemIcon(dst, eq.Icon, x+3, y+3, 1.25, false)
		g.label(dst, eq.Name, x, y+56, 16, ink)
		g.label(dst, ellipsis(g, slotNames[slot]+" · "+bonusText(eq.Bonus, nil), 12, cw-28), x, y+80, 12, muted)
		cx := int(x)
		for _, t := range eq.Traits {
			e := traitEntry{id: t, have: true}
			cx += g.traitChip(dst, cx, int(y+104), e, false, nil) + 6
		}
	}
	g.label(dst, "칸을 누르면 그 슬롯의 장비를 고릅니다.", float64(r.Min.X), float64(r.Min.Y+202), 13, muted)
}

// officerView is a kind of the officers' table: the columns after the name.
var officerViews = []string{"기본", "전투력", "열전", "장비"}

// officerTable shows the officers as a table, one row each and 48 high, in the columns of the kind picked (its tabs
// right of the view switch, after); a row opens that officer in the list.
func (g *Game) officerTable(dst *ebiten.Image, o core.Observation, roster []core.Character, r image.Rectangle, after int) {
	p := &g.party
	rect(dst, float32(after+12), float32(r.Min.Y+14), 1, 28, goldDim)
	g.tabRow(dst, after+24, r.Min.Y+4, officerViews, p.view, func(i int) { p.view, p.officerList.order = i, tableSort{} })
	rect(dst, float32(r.Min.X+8), float32(r.Min.Y+52), float32(r.Dx()-16), 1, goldDim)
	type row struct {
		c     core.Character
		s     core.Stats
		attrs [6]int
	}
	rows := make([]row, 0, len(roster))
	for _, c := range roster {
		s, a, _ := g.S.Engine.DeputyStats(c.ID, "")
		rows = append(rows, row{c, s, a})
	}
	num := func(key, label string, f func(*row) int) column[row] {
		return column[row]{key: key, label: label, get: func(r *row) (string, float64) { v := f(r); return fmt.Sprint(v), float64(v) }}
	}
	cols := []column[row]{{key: "name", label: "장수", w: 180, text: true, left: true, pad: 44,
		get: func(r *row) (string, float64) { return g.name(r.c.ID), 0 },
		after: func(dst *ebiten.Image, r *row, cell image.Rectangle) {
			g.portrait(dst, r.c.ID, r.c.Class, float64(cell.Min.X+4), float64(cell.Min.Y+(cell.Dy()-32)/2), 32)
		}}}
	switch p.view {
	case 0:
		cols = append(cols, column[row]{key: "class", label: "병종", text: true, get: func(r *row) (string, float64) { return g.className(r.c.Class), 0 },
			show: func(r *row, ink color.Color, w float64) (string, color.Color) {
				if g.canPromote(r.c.Def, r.c.Class, r.c.Level) {
					return g.className(r.c.Class), promoteC
				}
				return g.className(r.c.Class), ink
			}},
			num("level", "레벨", func(r *row) int { return r.c.Level }), num("hp", "병력", func(r *row) int { return r.s.MaxHP }),
			num("mp", "병법치", func(r *row) int { return r.s.MaxMP }), num("points", "특성치", func(r *row) int { return r.c.Points }))
	case 1:
		for _, s := range statRows[2:] {
			s := s
			cols = append(cols, num(s.name, s.name, func(r *row) int { return s.get(r.s) }))
		}
	case 2:
		for i, a := range attrNames {
			i := i
			cols = append(cols, num(a, a, func(r *row) int { return r.attrs[i] }))
		}
	case 3:
		for _, slot := range equipSlots {
			slot := slot
			cols = append(cols, column[row]{key: slot, label: slotNames[slot], text: true, get: func(r *row) (string, float64) {
				if id, ok := r.c.Equipment[slot]; ok {
					return g.itemName(id), 0
				}
				return "—", 0
			}})
		}
	}
	drawTable(g, dst, &p.officerList, table[row]{
		cols: cols, rowH: touchH, headH: 32, size: 15, center: true,
		selected: func(r *row) bool { return r.c.ID == g.selected },
		pick:     func(r *row) { g.selected, p.table = r.c.ID, false },
	}, rows, image.Rect(r.Min.X+12, r.Min.Y+60, r.Max.X-12, r.Max.Y-72))
}
