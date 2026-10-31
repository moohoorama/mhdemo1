package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// partyView is what the party window shows: the officers or the belongings, and of the belongings the kind
// picked and the one looked at.
type partyView struct {
	items        bool
	filter, item string
	list         tableList
}

// belonging is a kind of thing the party has: how many are carried or stored, and who wears the rest.
type belonging struct {
	id     string
	stock  int
	wearer []string
}

func (b belonging) count() int { return b.stock + len(b.wearer) }

// belongings are the party's things, kind by kind in the order of the filters: the items carried and the
// equipment stored or worn.
func (g *Game) belongings(o core.Observation) []belonging {
	by := map[string]*belonging{}
	add := func(id string) *belonging {
		if by[id] == nil {
			by[id] = &belonging{id: id}
		}
		return by[id]
	}
	for _, store := range []map[string]int{o.Inventory, o.Warehouse} {
		for id, n := range store {
			if n > 0 {
				add(id).stock += n
			}
		}
	}
	for _, c := range g.roster(o) {
		for _, slot := range equipSlots {
			if id, ok := c.Equipment[slot]; ok {
				b := add(id)
				b.wearer = append(b.wearer, g.name(c.ID))
			}
		}
	}
	out := make([]belonging, 0, len(by))
	for _, b := range by {
		out = append(out, *b)
	}
	slices.SortFunc(out, func(a, b belonging) int {
		if d := slices.Index(belongingKinds, g.belongingKind(a.id)) - slices.Index(belongingKinds, g.belongingKind(b.id)); d != 0 {
			return d
		}
		return strings.Compare(g.itemName(a.id), g.itemName(b.id))
	})
	return out
}

// belongingKind is the filter a thing falls under: an equipment slot's name, 인수 or 소모품.
func (g *Game) belongingKind(id string) string {
	if eq, ok := g.S.Data.Equipment[id]; ok {
		return slotNames[eq.Slot]
	}
	if g.S.Data.Items[id].Effect == "promote" {
		return "인수"
	}
	return "소모품"
}

var belongingKinds = []string{"전체", "소모품", "인수", "무기", "방어구", "장신구"}

// partyWindow is the party on the walk, in two tabs: the officers, the picked one's stats and equipment beside
// them with the buttons to change the equipment and the deputy, and the belongings.
func (g *Game) partyWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.View()
	p := &g.party
	g.choiceBtn(dst, image.Rect(x+28, y+28, x+148, y+62), "장수", !p.items, func() { p.items = false })
	g.choiceBtn(dst, image.Rect(x+154, y+28, x+274, y+62), "소지품", p.items, func() { p.items = true })
	money := fmt.Sprintf("돈 %d", o.Money)
	g.label(dst, money, float64(x+w-120)-g.width(money, 16), float64(y+34), 16, ink) // left of the window's 닫기
	rect(dst, float32(x+28), float32(y+72), float32(w-56), 1, goldDim)
	if p.items {
		g.belongingsTab(dst, o, x, y+84, w, h-84)
		return
	}
	roster := g.roster(o)
	if !containsID(roster, g.selected) && len(roster) > 0 {
		g.selected = roster[0].ID
	}
	ry := y + 96
	for _, of := range roster {
		officer := of
		row := image.Rect(x+28, ry, x+440, ry+76)
		g.btn(dst, row, "", func() { g.selected = officer.ID })
		if g.selected == of.ID {
			rect(dst, float32(row.Min.X), float32(row.Min.Y), 4, float32(row.Dy()), gold)
		}
		g.portrait(dst, of.ID, of.Class, float64(row.Min.X+14), float64(ry+8), 60)
		g.label(dst, g.name(of.ID), float64(row.Min.X+88), float64(ry+8), 19, ink)
		g.label(dst, fmt.Sprintf("Lv%d · %s", of.Level, g.className(of.Class)), float64(row.Min.X+88), float64(ry+40), 14, muted)
		ry += 86
	}
	g.officerDetail(dst, o, image.Rect(x+470, y+92, x+w-28, y+h-20))
}

// belongingsTab lists the party's things of the kind picked in a table on the left, the one looked at described on
// the right.
func (g *Game) belongingsTab(dst *ebiten.Image, o core.Observation, x, y, w, h int) {
	p := &g.party
	if p.filter == "" {
		p.filter = belongingKinds[0]
	}
	for i, k := range belongingKinds {
		kind := k
		g.choiceBtn(dst, image.Rect(x+28+i*102, y, x+28+i*102+96, y+30), kind, p.filter == kind, func() { p.filter, p.item, p.list.top = kind, "", 0 })
	}
	var shown []belonging
	for _, b := range g.belongings(o) {
		if p.filter == belongingKinds[0] || g.belongingKind(b.id) == p.filter {
			shown = append(shown, b)
		}
	}
	if len(shown) == 0 {
		g.label(dst, "가진 물건이 없습니다", float64(x+32), float64(y+52), 15, muted)
		return
	}
	cur := shown[0]
	rows := make([]goodsRow, len(shown))
	for i, b := range shown {
		if b.id == p.item {
			cur = b
		}
		rows[i] = g.goodsRow(b.id)
		rows[i].count = b.count()
		switch {
		case len(b.wearer) == 1:
			rows[i].wear = b.wearer[0]
		case len(b.wearer) > 1:
			rows[i].wear = fmt.Sprintf("%d명", len(b.wearer))
		case g.S.Data.Equipment[b.id].Slot != "":
			rows[i].wear = "창고"
		}
	}
	const tableW = 648
	drawTable(g, dst, &p.list, table[goodsRow]{
		cols: g.goodsColumns(goodsNumber("count", "수량", func(r *goodsRow) int { return r.count }),
			column[goodsRow]{key: "wear", label: "장착", w: 150, text: true, get: func(r *goodsRow) (string, float64) { return r.wear, 0 }}),
		rowH: 36, headH: 32, size: 15, center: true,
		selected: func(r *goodsRow) bool { return r.id == cur.id },
		pick:     func(r *goodsRow) { p.item = r.id },
	}, rows, image.Rect(x+28, y+44, x+28+tableW, y+h-20))
	g.belongingDetail(dst, cur, image.Rect(x+28+tableW+16, y, x+w-28, y+h-20))
}

// belongingDetail describes a thing: its picture, kind and use, how many there are and who wears them; equipment
// gets the button to the equipment window of its first wearer or of the officer picked.
func (g *Game) belongingDetail(dst *ebiten.Image, b belonging, r image.Rectangle) {
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.NRGBA{A: 56})
	strokeRect(dst, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, goldDim)
	x, y := float64(r.Min.X+20), float64(r.Min.Y+20)
	g.itemIcon(dst, g.itemIconOf(b.id), x+3, y+3, 2, false)
	g.label(dst, g.itemName(b.id), x+90, y+8, 22, gold)
	g.label(dst, g.belongingKind(b.id), x+90, y+42, 14, muted)
	desc := g.S.Data.Items[b.id].Desc
	eq, isEq := g.S.Data.Equipment[b.id]
	if isEq {
		desc = eq.Desc
	}
	g.label(dst, wrap(desc, 20), x, y+96, 15, ink)
	worn, where := "—", "전투 · 도구 명령"
	if isEq {
		worn, where = "없음", "장비"
		if len(b.wearer) > 0 {
			worn = strings.Join(b.wearer, " · ")
		}
		if b.stock > 0 {
			worn += fmt.Sprintf(" (창고 %d)", b.stock)
		}
	}
	for i, row := range [][2]string{{"가진 수", fmt.Sprintf("×%d", b.count())}, {"장착", worn}, {"쓰는 곳", where}} {
		ry := float64(r.Max.Y) - 160 + float64(i*28)
		g.label(dst, row[0], x, ry, 14, muted)
		g.label(dst, row[1], x+80, ry, 14, ink)
	}
	if !isEq {
		return
	}
	g.centeredBtn(dst, image.Rect(r.Min.X+20, r.Max.Y-58, r.Max.X-20, r.Max.Y-20), "장비 바꾸기", func() {
		for _, c := range g.roster(g.S.View()) {
			if slices.Contains(b.wearer, g.name(c.ID)) {
				g.selected = c.ID
				break
			}
		}
		g.party.items = false
		g.openOverlay("equipment")
	})
}
