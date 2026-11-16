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

// An 인수 is used outside a battle from the belongings (관리 · 소지품 · 소모품, also opened from the sortie
// preparation): a seal someone can take now, picked, lights 승급하기 left of 닫기, which opens the picker, a table of the officers (이름 · 병종 · 레벨) where only
// those it can promote are lit; 미리 보기 shows the promotion window of the one picked, a tab for each class when the
// seal leads to more than one, with the traits the new class teaches, and its 승급하기 promotes him (core
// promoteAway). In battle the seal is refused.

// insuWhere tells where a seal is used: off the field, where the promoted officer can change weapons.
const insuWhere = "인수는 탐험과 출전 준비의 소지품에서 쓸 수 있습니다"

// insuInBattle is the note on a seal in the battle's item list, where it is greyed out.
const insuInBattle = "전투 중에는 쓸 수 없습니다. 탐험이나 출전 준비에서 소지품을 열어 쓰십시오."

// insuState is the seal picker of the party window: the seal, the officer picked, the
// promotions it would make looked at before one is made, and the one shown.
type insuState struct {
	item, pick string
	list       tableList
	preview    []promotionUp
	at         int
}

// insuClasses are the classes the seal would make the officer now, or nothing.
func (g *Game) insuClasses(item, id string) []string {
	classes, err := g.S.Engine.PromotionsAway(item, id)
	if err != nil {
		return nil
	}
	return classes
}

// insuUsable reports whether the seal is one some officer of the party can take now.
func (g *Game) insuUsable(o core.Observation, item string) bool {
	if g.S.Data.Items[item].Effect != content.ItemPromote || o.Inventory[item] < 1 {
		return false
	}
	for _, c := range g.roster(o) {
		if len(g.insuClasses(item, c.ID)) > 0 {
			return true
		}
	}
	return false
}

// openInsu opens the picker of the seal.
func (g *Game) openInsu(item string) {
	g.party.insu = insuState{item: item}
}

// insuPicker is the party window's 소지품 while a seal is being used: the officers in a table on the left, the seal and
// the officer picked with the classes ahead on the right, and 미리 보기 left of 닫기, which goes back to the belongings.
func (g *Game) insuPicker(dst *ebiten.Image, o core.Observation, x, y, w, h int) {
	s := &g.party.insu
	it := g.S.Data.Items[s.item]
	type row struct {
		c       core.Character
		classes []string
	}
	var rows []row
	for _, c := range g.roster(o) {
		rows = append(rows, row{c, g.insuClasses(s.item, c.ID)})
	}
	open := func(r *row) bool { return len(r.classes) > 0 }
	var picked *row // a copy: the table sorts rows in place
	for i := range rows {
		if rows[i].c.ID == s.pick && open(&rows[i]) {
			r := rows[i]
			picked = &r
		}
	}
	g.itemIcon(dst, g.itemIconOf(s.item), float64(x+19), float64(y+3), .75, false)
	g.label(dst, it.Name+" — 누구에게 쓸까요?", float64(x+56), float64(y), 16, gold)
	g.label(dst, fmt.Sprintf("남은 수 ×%d · 승급 레벨 %d 이상", o.Inventory[s.item], g.S.Data.Rules.PromoteLevels[it.Tier-1]), float64(x+56), float64(y+24), 12, muted)
	detailW := min(320, w/3)
	tableW := w - 32 - detailW - 16
	num := func(key, label string, f func(*row) int) column[row] {
		return column[row]{key: key, label: label, w: 90, get: func(r *row) (string, float64) { v := f(r); return fmt.Sprint(v), float64(v) }}
	}
	drawTable(g, dst, &s.list, table[row]{
		cols: []column[row]{
			{key: "name", label: "이름", text: true, left: true, pad: 12, get: func(r *row) (string, float64) { return g.name(r.c.ID), 0 }},
			{key: "class", label: "병종", w: 150, text: true, get: func(r *row) (string, float64) { return g.className(r.c.Class), 0 }},
			num("level", "레벨", func(r *row) int { return r.c.Level }),
		},
		rowH: touchH, headH: 32, size: 15, center: true,
		rank:     func(r *row) int { return map[bool]int{true: 0, false: 1}[open(r)] },
		open:     open,
		dim:      func(r *row) bool { return !open(r) },
		selected: func(r *row) bool { return picked != nil && r.c.ID == picked.c.ID },
		pick:     func(r *row) { s.pick = r.c.ID },
		guide: func(r *row) bool {
			seal, ok := g.promoteGuide()
			return ok && seal == s.item && picked == nil && open(r)
		},
	}, rows, image.Rect(x+16, y+56, x+16+tableW, y+h))

	card := image.Rect(x+16+tableW+16, y, x+w-16, y+h)
	g.card(dst, card)
	cx, cy := float64(card.Min.X+14), float64(card.Min.Y+14)
	g.itemIcon(dst, g.itemIconOf(s.item), cx+3, cy+3, 1.5, false)
	g.label(dst, it.Name, cx+64, cy, 18, gold)
	g.label(dst, "소모품 · 인수", cx+64, cy+28, 12, muted)
	cy += 62
	desc := wrapWidth(it.Desc, g.face(14), float64(card.Dx()-28))
	g.label(dst, desc, cx, cy, 14, ink)
	_, dh := measureLines(g, desc, 14)
	cy += dh + 14
	if picked == nil {
		g.label(dst, wrapWidth("밝게 보이는 장수를 고르세요.", g.face(14), float64(card.Dx()-28)), cx, cy, 14, muted)
		return
	}
	names := make([]string, len(picked.classes))
	for i, c := range picked.classes {
		names[i] = g.className(c)
	}
	for _, line := range [][2]string{{"장수", fmt.Sprintf("%s · Lv%d", g.name(picked.c.ID), picked.c.Level)}, {"병종", g.className(picked.c.Class)}, {"승급", strings.Join(names, " · ")}} {
		g.label(dst, line[0], cx, cy, 13, muted)
		g.label(dst, ellipsis(g, line[1], 13, card.Dx()-100), cx+70, cy, 13, ink)
		cy += 22
	}
}

// insuActs is the picker's 미리 보기: the promotion window, with a tab for each class when the seal leads to several.
func (g *Game) insuActs() []action {
	s := &g.party.insu
	classes := g.insuClasses(s.item, s.pick)
	if len(classes) == 0 {
		return []action{{insuPreviewLabel, true, nil}}
	}
	cmds := make([]core.Command, len(classes))
	for i, c := range classes {
		cmds[i] = core.Command{Kind: "item", Actor: s.pick, Target: s.pick, Item: s.item, Option: c}
	}
	return []action{{insuPreviewLabel, true, func() { s.preview, s.at = g.branches(cmds), 0 }}}
}

const insuPreviewLabel = "미리 보기"

// useInsu sends the seal's command and goes back to the belongings; the promotion was seen in its preview, so
// its window does not open again.
func (g *Game) useInsu(c core.Command) {
	g.party.insu = insuState{}
	g.party.item = c.Item
	g.sfx("ok")
	g.command(c)
	g.modals = slices.DeleteFunc(g.modals, func(m *modal) bool { return m.kind == "promotion" })
}

// drawInsuPreview lays the promotion window of the seal over the party window, before anything is spent: the
// classes it leads to as tabs over the one shown, 승급하기 making that one, 취소 or outside stepping back to the table.
func (g *Game) drawInsuPreview(dst *ebiten.Image) {
	s := &g.party.insu
	back := func() { s.preview = nil }
	g.addButton(image.Rect(0, 0, uiW, uiH), "", back)
	p := s.preview[s.at]
	head := 0
	if len(s.preview) > 1 {
		head = promotionTabsH
	}
	r := g.promotionBox(p, uiW, head)
	g.popup(dst, "overlay:insu-preview", 120, uiW, uiH, r.Inset(-24), func(dst *ebiten.Image) {
		g.titled(dst, r, "승급")
		if head > 0 {
			labels := make([]string, len(s.preview))
			for i, b := range s.preview {
				labels[i] = g.className(b.to)
			}
			g.tabRow(dst, r.Min.X+40, r.Min.Y+28, labels, s.at, func(i int) { s.at = i; g.sfx("ok") })
			rect(dst, float32(r.Min.X+40), float32(r.Min.Y+28+touchH), float32(r.Dx()-80), 1, goldDim)
		}
		g.promotionBody(dst, p, 1, r, float64(r.Min.Y+head))
		act := action{"승급하기", true, func() { g.useInsu(p.cmd) }}
		g.endRow(dst, r.Max.X-16, r.Max.Y-touchH-12, "취소", back, act)
		if seal, ok := g.promoteGuide(); ok && seal == p.cmd.Item {
			g.promoteRim(dst, g.actionRect(r.Max.X-16, r.Max.Y-touchH-12, act.label, act))
		}
	})
}

// traitOverWindow reports whether a trait opened from the seal's preview is showing over the party window, which
// stays open beneath it.
func (g *Game) traitOverWindow() bool {
	return g.overlay() == "party" && len(g.modals) > 0 && g.modals[0].kind == "trait"
}
