package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// The deputy window (부관 임명): the officers who could serve the selected one, each with the weight it would get
// and the attributes it would raise, and the one picked beside them with what it changes.

const deputyW, deputyH = 1040, 700

// deputyRow is one candidate: what the officer would be with it as deputy, and why it cannot be.
type deputyRow struct {
	id, name, class string
	level           int
	traits          []string
	stats           core.Stats
	attrs           [6]int
	reason          string // why it cannot be named, "" when it can
	note            string // what naming it does besides, as leaving the field
	current         bool
}

// deputyViews are the views: the stat each gain column measures, read from the row and the bare officer.
var deputyViews = []struct {
	title string
	stats []struct {
		key, label string
		get        func(s core.Stats, a [6]int) int
	}
}{
	{"기본", deputyStats([][2]string{{"hp", "병력"}, {"mp", "병법치"}})},
	{"전투력", deputyStats([][2]string{{"attack", "공격"}, {"defense", "방어"}, {"mind", "정신"}, {"agility", "순발"}, {"morale", "사기"}, {"recovery", "병법회복"}})},
	{"열전", deputyStats([][2]string{{"a0", "통솔"}, {"a1", "무력"}, {"a2", "지력"}, {"a3", "민첩"}, {"a4", "운"}, {"a5", "매력"}})},
}

func deputyStats(keys [][2]string) []struct {
	key, label string
	get        func(s core.Stats, a [6]int) int
} {
	get := map[string]func(s core.Stats, a [6]int) int{
		"hp": func(s core.Stats, a [6]int) int { return s.MaxHP }, "mp": func(s core.Stats, a [6]int) int { return s.MaxMP },
		"attack": func(s core.Stats, a [6]int) int { return s.Attack }, "defense": func(s core.Stats, a [6]int) int { return s.Defense },
		"mind": func(s core.Stats, a [6]int) int { return s.Mind }, "agility": func(s core.Stats, a [6]int) int { return s.Agility },
		"morale": func(s core.Stats, a [6]int) int { return s.Morale }, "recovery": func(s core.Stats, a [6]int) int { return s.Recovery },
	}
	for i := range 6 {
		get[fmt.Sprint("a", i)] = func(s core.Stats, a [6]int) int { return a[i] }
	}
	out := make([]struct {
		key, label string
		get        func(s core.Stats, a [6]int) int
	}, len(keys))
	for i, k := range keys {
		out[i].key, out[i].label, out[i].get = k[0], k[1], get[k[0]]
	}
	return out
}

// deputyRows are the officers who could serve the selected one, those who can first.
func (g *Game) deputyRows(o core.Observation) []deputyRow {
	actor := g.selected
	served := map[string]string{}
	for _, c := range o.Characters {
		if c.Deputy != "" {
			served[c.Deputy] = c.ID
		}
	}
	rows := []deputyRow{}
	for _, c := range g.roster(o) {
		if c.ID == actor || c.Dead {
			continue
		}
		st, at, err := g.S.Engine.DeputyStats(actor, c.ID)
		if err != nil {
			continue
		}
		r := deputyRow{id: c.ID, name: g.name(c.ID), class: c.Class, level: c.Level, traits: g.S.Engine.DeputyTraits(c.ID), stats: st, attrs: at}
		of := g.officer(o, actor)
		r.current = of != nil && of.Deputy == c.ID
		switch err := g.S.Engine.DeputyCheck(actor, c.ID); {
		case r.current:
		case err == nil:
			if g.deploying() && slices.Contains(o.Deployment, c.ID) {
				r.note = "출전 중입니다. 임명하면 전장에서 빠집니다."
			}
		case served[c.ID] != "":
			r.reason = g.name(served[c.ID]) + "의 부관이라 부관이 될 수 없습니다."
		case c.Deputy != "":
			r.reason = "부관을 거느린 장수라 부관이 될 수 없습니다."
		case err.Error() == "LordDeputy":
			r.reason = "군주라 부관이 될 수 없습니다."
		case err.Error() == "RequiredDeputy":
			r.reason = "이번 전투 필수 출전 장수라 부관이 될 수 없습니다."
		default:
			r.reason = koreanError(err.Error())
		}
		rows = append(rows, r)
	}
	return rows
}

// deputyWindow is the deputy appointment, filling the screen: the candidates on the left, each with where they
// are now, their charm, the weight they would get and the attributes they would raise; the one picked on the right
// with the lead's battle stats alone, the bonus and as a unit, and the traits the deputy would bring.
func (g *Game) deputyWindow(dst *ebiten.Image, x, y, w, h int) {
	o := g.S.View()
	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	alone, aloneAttrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return
	}
	rows := g.deputyRows(o)
	slices.SortStableFunc(rows, func(a, b deputyRow) int {
		if a.current != b.current {
			return b2i(b.current) - b2i(a.current)
		}
		return b2i(a.reason != "") - b2i(b.reason != "")
	})
	pick := g.deputyPick
	if pick == "" {
		pick = of.Deputy
	}
	at := slices.IndexFunc(rows, func(r deputyRow) bool { return r.id == pick })
	if at < 0 && len(rows) > 0 {
		at, pick = 0, rows[0].id
	}
	listW := min(520, w*11/20)
	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-16), goldDim)
	g.label(dst, "부관 임명", float64(x+16), float64(y+12), 18, gold)
	sub := fmt.Sprintf("%s대 · %s 매력 %d", g.name(of.ID), g.name(of.ID), aloneAttrs[5])
	g.label(dst, sub, float64(x+28)+g.width("부관 임명", 18), float64(y+17), 13, muted)
	rect(dst, float32(x+8), float32(y+56), float32(listW-16), 1, goldDim)
	g.listRows(dst, &g.party.deputies, image.Rect(x+6, y+58, x+listW-4, y+h-6), len(rows), listRowH, func(i int, rr image.Rectangle) {
		r := rows[i]
		c := g.officer(o, r.id)
		if c == nil {
			return
		}
		rowBack(dst, rr, r.id == pick)
		ink0 := ink
		if r.reason != "" {
			ink0 = faintInk
		}
		px, py := float64(rr.Min.X+12), float64(rr.Min.Y)
		g.portrait(dst, c.ID, c.Class, px, py+8, 40)
		g.label(dst, r.name, px+50, py+7, 16, ink0)
		where := "대기"
		switch {
		case r.current:
			where = g.name(of.ID) + " 부관"
		case g.servedBy(o, r.id) != "":
			where = g.name(g.servedBy(o, r.id)) + " 부관"
		case slices.Contains(o.Deployment, r.id):
			where = "출전"
		}
		_, depAttrs, _ := g.S.Engine.DeputyStats(r.id, "")
		g.label(dst, fmt.Sprintf("%s · 매력 %d", where, depAttrs[5]), px+50, py+31, 12, muted)
		if r.reason != "" {
			g.label(dst, ellipsis(g, r.reason, 12, rr.Dx()-200), px+190, py+21, 12, faintInk)
			return
		}
		wr, _ := g.S.Engine.DeputyBlend(of.ID, r.id)
		g.label(dst, fmt.Sprintf("%.0f%%", wr*100), px+172, py+14, 18, gold)
		ups := ""
		for k, a := range attrNames {
			if d := r.attrs[k] - aloneAttrs[k]; d > 0 {
				ups += fmt.Sprintf("%s+%d ", a, d)
			}
		}
		if ups == "" {
			ups = "오르는 능력 없음"
		}
		g.label(dst, ellipsis(g, ups, 13, rr.Dx()-252), px+236, py+19, 13, good)
	}, func(i int) { g.deputyPick = rows[i].id })

	rx, right := x+listW+16, x+w-16
	var picked *deputyRow
	if at >= 0 {
		picked = &rows[at]
	}
	if picked != nil {
		c := g.officer(o, picked.id)
		if c != nil {
			g.portrait(dst, c.ID, c.Class, float64(rx), float64(y+8), 44)
		}
		g.label(dst, picked.name, float64(rx+54), float64(y+6), 18, gold)
		info := ""
		if picked.reason == "" {
			wr, _ := g.S.Engine.DeputyBlend(of.ID, picked.id)
			info = fmt.Sprintf("가중치 %.0f%%", wr*100)
		}
		if picked.note != "" {
			info += " · " + picked.note
		}
		g.label(dst, ellipsis(g, info, 12, right-rx-60), float64(rx+54), float64(y+32), 12, muted)
		ty := float64(y + 64)
		cols := []float64{float64(rx), float64(rx) + float64(right-rx)*0.42, float64(rx) + float64(right-rx)*0.68, float64(right)}
		for i, hd := range []string{"전투", "혼자", "보정", "부대"} {
			px := cols[i]
			if i > 0 {
				px -= g.width(hd, 12) + 8
			}
			g.label(dst, hd, px, ty, 12, muted)
		}
		rect(dst, float32(rx), float32(ty+20), float32(right-rx), 1, goldDim)
		ty += 26
		for _, s := range statRows {
			a, b := s.get(alone), s.get(picked.stats)
			bonus, bc := "—", faintInk
			if b != a {
				bonus, bc = fmt.Sprintf("%+d", b-a), good
			}
			for i, v := range []string{s.name, fmt.Sprint(a), bonus, fmt.Sprint(b)} {
				px, c := cols[i], color.Color(ink)
				if i > 0 {
					px -= g.width(v, 14) + 8
				}
				switch i {
				case 0:
					c = muted
				case 2:
					c = bc
				case 3:
					c = gold
				}
				g.label(dst, v, px, ty, 14, c)
			}
			ty += 24
		}
		ty += 6
		g.label(dst, "부대에 더해지는 특성", float64(rx), ty, 12, muted)
		tx := float64(rx)
		ty += 20
		if len(picked.traits) == 0 {
			g.label(dst, "없음", tx, ty, 13, faintInk)
		}
		for _, t := range picked.traits {
			tw := 26 + g.width(g.traitName(t), 13) + 12
			if tx+tw > float64(right) {
				tx, ty = float64(rx), ty+24
			}
			if ty+20 > float64(y+h-70) {
				break
			}
			g.traitIconOf(dst, t, tx, ty, 20)
			g.label(dst, g.traitName(t), tx+24, ty, 13, ink)
			tx += tw
		}
	}

	var appoint, release func()
	if picked != nil && pick != of.Deputy && picked.reason == "" {
		p := pick
		appoint = func() {
			if g.request(session.Request{Op: "command", Command: core.Command{Kind: "deputy", Actor: of.ID, Target: p}}).OK {
				g.deputyPick = ""
			}
		}
	}
	if of.Deputy != "" {
		release = func() {
			if g.request(session.Request{Op: "command", Command: core.Command{Kind: "undeputy", Actor: of.ID}}).OK {
				g.deputyPick = ""
			}
		}
	}
	label := "임명"
	switch {
	case picked != nil && picked.current:
		label = "지금 부관"
	case picked != nil:
		label = picked.name + " 임명"
	}
	g.closeRow(dst, right, y+h-60, g.closeOverlay, action{"부관 해임", false, release}, action{label, true, appoint})
}

// traitList names as many of the traits as fit in w at the size, then how many more there are.
func (g *Game) traitList(traits []string, size, w float64) string {
	if len(traits) == 0 {
		return "—"
	}
	out := g.traitName(traits[0])
	for i := 1; i < len(traits); i++ {
		next := out + ", " + g.traitName(traits[i])
		rest := ""
		if i < len(traits)-1 {
			rest = fmt.Sprintf(" 외 %d", len(traits)-i-1)
		}
		if g.width(next+rest, size) > w {
			return out + fmt.Sprintf(" 외 %d", len(traits)-i)
		}
		out = next
	}
	return out
}
