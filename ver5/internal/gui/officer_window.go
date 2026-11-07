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
var officerTabs = []string{"전투력", "특성", "열전", "장비"}

// partyWindow is the officers' window of the walk, filling the screen: the officers listed on the left and the
// picked one on the right in four tabs (전투력, 특성, 열전, 장비), where traits are learned and equipment changed.
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
	closeLabel := "닫기"
	if p.learning {
		closeLabel = "다 배웠음"
	}
	switch {
	case p.items:
		g.label(dst, "소지품", float64(x+16), float64(y+14), 18, gold)
		g.moneyLabel(dst, float64(x+w-16), float64(y+16), 15)
		g.belongingsTab(dst, o, x, y+56, w, h-56-64)
		g.actionRow(dst, x+w-16, y+h-60, action{"장수로", false, func() { p.items = false }}, action{closeLabel, false, g.closeOverlay})
		return
	case p.table:
		g.officerTable(dst, o, roster, image.Rect(x, y, x+w, y+h))
		g.actionRow(dst, x+w-16, y+h-60, action{"목록으로", false, func() { p.table = false }}, action{closeLabel, false, g.closeOverlay})
		return
	}
	listW := 300
	if w >= 1100 {
		listW = 340
	}
	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-16), goldDim)
	g.label(dst, "장수", float64(x+16), float64(y+12), 18, gold)
	g.label(dst, fmt.Sprintf("%d명", len(roster)), float64(x+24)+g.width("장수", 18), float64(y+17), 13, muted)
	g.mBtn(dst, image.Rect(x+listW-160, y+4, x+listW-84, y+4+touchH), "소지품", false, func() { p.items = true })
	g.mBtn(dst, image.Rect(x+listW-78, y+4, x+listW-8, y+4+touchH), "표로", false, func() { p.table = true })
	rect(dst, float32(x+8), float32(y+56), float32(listW-16), 1, goldDim)
	g.listRows(dst, &p.officers, image.Rect(x+6, y+58, x+listW-4, y+h-6), len(roster), listRowH, func(i int, rr image.Rectangle) {
		c := roster[i]
		note, noteC := fmt.Sprintf("특성치 %d", c.Points), muted
		if g.hasAffordable(c) {
			noteC = good
		}
		g.officerRow(dst, rr, c, c.ID == g.selected, note, noteC)
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
	rect(dst, float32(rx), float32(y+52), float32(x+w-16-rx), 1, goldDim)
	body := image.Rect(rx, y+60, x+w-16, y+h-64)
	acts := []action{{closeLabel, false, g.closeOverlay}}
	switch p.tab {
	case 0:
		g.officerPower(dst, o, of, body)
	case 1:
		acts = append(acts, g.officerTraits(dst, of, body)...)
	case 2:
		g.officerBio(dst, of, body)
	case 3:
		g.officerEquipment(dst, of, body)
		acts = append(acts, action{"장비 변경", true, func() { g.openOverlay("equipment") }})
	}
	g.actionRow(dst, x+w-16, y+h-60, acts...)
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

// officerPower is the 전투력 tab: the hexagon, the battle stats and the six attributes of the officer alone.
func (g *Game) officerPower(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) {
	stats, attrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return
	}
	radius := min(72.0, float64(r.Dy()-60)/2)
	g.hexChart(dst, float64(r.Min.X)+radius+64, float64(r.Min.Y)+radius+26, radius, stats, nil, g.rosterBest(o))
	x, y := float64(r.Min.X)+2*radius+150, float64(r.Min.Y+4)
	for i, s := range statRows {
		col, row := i%2, i/2
		px, py := x+float64(col*130), y+float64(row*28)
		g.label(dst, s.name, px, py, 15, muted)
		g.label(dst, fmt.Sprint(s.get(stats)), px+60, py, 15, ink)
	}
	y += 4*28 + 10
	rect(dst, float32(x), float32(y), 300, 1, goldDim)
	y += 10
	for i, a := range attrNames {
		col, row := i%3, i/3
		px, py := x+float64(col*100), y+float64(row*28)
		g.label(dst, a, px, py, 15, muted)
		g.label(dst, fmt.Sprint(attrs[i]), px+44, py, 15, ink)
	}
	progress := core.ExperienceProgress(of.XP)
	y += 2*28 + 12
	bar(dst, float32(x), float32(y+6), 180, 8, progress/100, xpC)
	g.label(dst, fmt.Sprintf("경험치 %.0f / 100", progress), x+190, y, 13, muted)
}

var attrNames = []string{"통솔", "무력", "지력", "민첩", "운", "매력"}

// officerTraits is the 특성 tab: the shelves 고유, 병종 and 장비 and the picked trait described under them. It gives
// the action that learns the picked trait when it can be learned.
func (g *Game) officerTraits(dst *ebiten.Image, of *core.Character, r image.Rectangle) []action {
	p := &g.party
	shelves := g.shelvesOf(of)
	var picked *traitEntry
	all := []traitEntry{}
	for _, s := range shelves {
		all = append(all, s.traits...)
	}
	if i := slices.IndexFunc(all, func(e traitEntry) bool { return e.id == p.trait }); i >= 0 {
		picked = &all[i]
	} else if i := slices.IndexFunc(all, func(e traitEntry) bool { return e.affordable }); i >= 0 {
		picked = &all[i]
	} else if len(all) > 0 {
		picked = &all[0]
	}
	pick := ""
	if picked != nil {
		pick = picked.id
	}
	used := g.traitShelves(dst, r, shelves, pick, func(id string) { p.trait = id })
	if picked == nil {
		return nil
	}
	note := ""
	switch {
	case picked.have:
	case !picked.learnable && picked.requires != "":
		note = "선행 특성 " + picked.requires + "을 먼저 배워야 합니다"
	case !picked.affordable:
		note = fmt.Sprintf("특성치가 %d 모자랍니다 (%d 필요, %d 있음)", picked.cost-of.Points, picked.cost, of.Points)
	case !g.canLearnNow(of.ID):
		note = "탐험이나 출전 준비 중에 배울 수 있습니다"
	default:
		note = fmt.Sprintf("특성치 %d → %d", of.Points, of.Points-picked.cost)
	}
	cardY := min(r.Min.Y+used+6, r.Max.Y-84)
	g.traitCard(dst, image.Rect(r.Min.X, cardY, r.Max.X, cardY+84), picked.id, picked.group, note)
	if picked.have || !picked.learnable {
		return nil
	}
	var learn func()
	if picked.affordable && g.canLearnNow(of.ID) {
		id, trait := of.ID, picked.id
		learn = func() { g.command(core.Command{Kind: "learn", Actor: id, Trait: trait}) }
	}
	return []action{{fmt.Sprintf("%s 배우기 · %d", g.traitName(picked.id), picked.cost), true, learn}}
}

// officerBio is the 열전 tab: the biography, the six attributes and the officer's own traits with what they do.
func (g *Game) officerBio(dst *ebiten.Image, of *core.Character, r image.Rectangle) {
	x, y := float64(r.Min.X), float64(r.Min.Y)
	bio := wrapWidth(g.S.Data.Characters[of.Def].Biography, g.face(14), float64(r.Dx()))
	g.label(dst, bio, x, y, 14, ink)
	_, h := measureLines(g, bio, 14)
	y += h + 12
	_, attrs, _ := g.S.Engine.DeputyStats(of.ID, "")
	for i, a := range attrNames {
		px := x + float64(i)*float64(r.Dx())/6
		g.label(dst, a, px, y, 15, muted)
		g.label(dst, fmt.Sprint(attrs[i]), px+44, y, 15, ink)
	}
	y += 34
	rect(dst, float32(x), float32(y), float32(r.Dx()), 1, goldDim)
	y += 8
	own := g.shelvesOf(of)[0]
	for _, e := range own.traits {
		if y+touchH > float64(r.Max.Y) {
			break
		}
		w := g.traitChip(dst, int(x), int(y), e, false, nil)
		g.label(dst, ellipsis(g, g.traitDesc(e.id), 14, r.Dx()-w-16), x+float64(w)+12, y+14, 14, muted)
		y += touchH
	}
}

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

// officerEquipment is the 장비 tab: a card for each slot with the piece, what it adds and the traits it gives;
// pressing a card opens that slot's picker.
func (g *Game) officerEquipment(dst *ebiten.Image, of *core.Character, r image.Rectangle) {
	cw := (r.Dx() - 16) / 3
	for i, slot := range equipSlots {
		c := image.Rect(r.Min.X+i*(cw+8), r.Min.Y, r.Min.X+i*(cw+8)+cw, r.Min.Y+190)
		card(dst, c)
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

// officerTable shows the officers as a table, one row each and 48 high, in the columns of the kind picked; a row
// opens that officer in the list.
func (g *Game) officerTable(dst *ebiten.Image, o core.Observation, roster []core.Character, r image.Rectangle) {
	p := &g.party
	g.label(dst, "장수 · 표", float64(r.Min.X+16), float64(r.Min.Y+12), 18, gold)
	g.tabRow(dst, r.Min.X+40+int(g.width("장수 · 표", 18)), r.Min.Y+4, officerViews, p.view, func(i int) { p.view, p.officerList.order = i, tableSort{} })
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
		cols = append(cols, column[row]{key: "class", label: "병종", text: true, get: func(r *row) (string, float64) { return g.className(r.c.Class), 0 }},
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
