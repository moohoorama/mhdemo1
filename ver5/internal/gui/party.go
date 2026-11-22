package gui

import (
	"fmt"
	"image"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
)

// partyView is what the party window shows: the officers or the belongings, and of the belongings the kind
// picked and the one looked at.
type partyView struct {
	items            bool
	table            bool // the officers as a table rather than the list beside the one picked
	tab, view        int  // the picked officer's tab (officerTabs) and the table's kind (officerViews)
	learning         bool // opened by a level-up for the officers still learning; closing it ends their learning
	filter, item     string
	trait            string
	tip              string // the equipment trait whose description shows on 장비, as "slot|trait"
	sortieFades      map[[2]string]sortieTransition
	sortieBrightness map[string]sortieTransition
	sortieView       int       // table or editing view in the sortie
	sortieTab        int       // the sortie's tab of the picked officer (sortieTabs)
	depLead          string    // the lead the 부관 tab last showed, so another lead starts on their own deputy
	depAll           bool      // the traits the candidate picked brings are all shown, past the lines they get
	shelves          tableList // the 특성 tab's shelves scrolled, in steps of shelfStep
	list             tableList
	officers         tableList
	officerList      tableList
	deputies         tableList
	placing          tableList // the deployment's list of the sortie
	insu             insuState // the seal being used from the belongings
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

// belongingKind is the filter a thing falls under: an equipment slot's name or 소모품, the seals among them.
func (g *Game) belongingKind(id string) string {
	if eq, ok := g.S.Data.Equipment[id]; ok {
		return slotNames[eq.Slot]
	}
	return "소모품"
}

var belongingKinds = []string{"전체", "무기", "방어구", "장신구", "소모품"}

// belongingsTab lists the party's things of the kind picked in a table on the left, the one looked at described on
// the right, and gives that one's id ("" when there is nothing).
func (g *Game) belongingsTab(dst *ebiten.Image, o core.Observation, x, y, w, h int) string {
	p := &g.party
	if p.filter == "" {
		p.filter = belongingKinds[0]
	}
	g.tabRow(dst, x+16, y, belongingKinds, slices.Index(belongingKinds, p.filter), func(i int) { p.filter, p.item, p.list.top = belongingKinds[i], "", 0 })
	seal, guide := g.promoteGuide()
	if consumables := slices.Index(belongingKinds, "소모품"); guide && p.filter != belongingKinds[consumables] {
		g.promoteRim(dst, g.tabRect(x+16, y, belongingKinds, consumables))
	}
	var shown []belonging
	for _, b := range g.belongings(o) {
		if p.filter == belongingKinds[0] || g.belongingKind(b.id) == p.filter {
			shown = append(shown, b)
		}
	}
	if len(shown) == 0 {
		g.label(dst, "가진 물건이 없습니다", float64(x+20), float64(y+64), 15, muted)
		return ""
	}
	cur := shown[0]
	rows := make([]goodsRow, len(shown))
	for i, b := range shown {
		if b.id == p.item {
			cur = b
		}
		rows[i] = g.goodsRow(b.id)
		rows[i].count = b.count()
		rows[i].ok = g.insuUsable(o, b.id) // only what can be used here is lit
		switch {
		case len(b.wearer) == 1:
			rows[i].wear = b.wearer[0]
		case len(b.wearer) > 1:
			rows[i].wear = fmt.Sprintf("%d명", len(b.wearer))
		case g.S.Data.Equipment[b.id].Slot != "":
			rows[i].wear = "창고"
		}
	}
	detailW := min(320, w/3)
	tableW := w - 32 - detailW - 16
	drawTable(g, dst, &p.list, table[goodsRow]{
		cols: g.goodsColumns(goodsNumber("count", "수량", func(r *goodsRow) int { return r.count }),
			column[goodsRow]{key: "wear", label: "장착", w: 150, text: true, get: func(r *goodsRow) (string, float64) { return r.wear, 0 }}),
		rowH: touchH, headH: 32, size: 15, center: true,
		dim:      func(r *goodsRow) bool { return !r.ok },
		selected: func(r *goodsRow) bool { return r.id == cur.id },
		pick:     func(r *goodsRow) { p.item = r.id },
		guide:    func(r *goodsRow) bool { return guide && r.id == seal && cur.id != seal },
	}, rows, image.Rect(x+16, y+56, x+16+tableW, y+h))
	g.belongingDetail(dst, cur, image.Rect(x+16+tableW+16, y, x+w-16, y+h))
	return cur.id
}

// belongingDetail describes a thing: its picture, name and kind, what it does, how many there are and who wears
// them; equipment gets the button to the equipment window of its first wearer or of the officer picked.
func (g *Game) belongingDetail(dst *ebiten.Image, b belonging, r image.Rectangle) {
	g.card(dst, r)
	x, y := float64(r.Min.X+14), float64(r.Min.Y+14)
	g.itemIcon(dst, g.itemIconOf(b.id), x+3, y+3, 1.5, false)
	g.label(dst, g.itemName(b.id), x+64, y, 18, gold)
	kind := g.belongingKind(b.id)
	if g.S.Data.Items[b.id].Effect == content.ItemPromote {
		kind += " · 인수"
	}
	g.label(dst, kind, x+64, y+28, 12, muted)
	desc := g.S.Data.Items[b.id].Desc
	eq, isEq := g.S.Data.Equipment[b.id]
	if isEq {
		desc = eq.Desc
	}
	y += 62
	desc = wrapWidth(desc, g.face(14), float64(r.Dx()-28))
	g.label(dst, desc, x, y, 14, ink)
	_, dh := measureLines(g, desc, 14)
	y += dh + 8
	worn, where := "—", "전투 · 도구 명령"
	if g.S.Data.Items[b.id].Effect == content.ItemPromote {
		where = "탐험 · 관리 · 소지품, 출전 준비 · 소지품"
	}
	if isEq {
		worn, where = "없음", "장비"
		if len(b.wearer) > 0 {
			worn = strings.Join(b.wearer, " · ")
		}
		if b.stock > 0 {
			worn += fmt.Sprintf(" (창고 %d)", b.stock)
		}
	}
	rows := [][2]string{{"가진 수", fmt.Sprintf("×%d", b.count())}, {"장착", worn}, {"쓰는 곳", where}}
	if isEq {
		rows = append([][2]string{{"보정", bonusText(eq.Bonus, nil)}}, rows...)
	}
	for _, row := range rows {
		g.label(dst, row[0], x, y, 13, muted)
		g.label(dst, ellipsis(g, row[1], 13, r.Dx()-100), x+70, y, 13, ink)
		y += 22
	}
	if !isEq {
		return
	}
	g.mBtn(dst, image.Rect(r.Min.X+14, r.Max.Y-touchH-12, r.Max.X-14, r.Max.Y-12), "장비 바꾸기", false, func() {
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
