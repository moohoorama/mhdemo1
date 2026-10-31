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

// The deputy window (부관 임명): the officer's stats on top, then a sortable table of the officers
// who could serve it, each with what it would add, in the roster window's three views.

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

// deputyColumns are the table's columns in the view: the name and the traits, on 기본 the class
// and level, then what the candidate would add.
func (g *Game) deputyColumns(base core.Stats, attrs [6]int) []column[deputyRow] {
	cols := []column[deputyRow]{
		{key: "name", label: "장수", w: 110, text: true, left: true, size: 1,
			get: func(r *deputyRow) (string, float64) { return r.name, 0 },
			show: func(r *deputyRow, ink color.Color, w float64) (string, color.Color) {
				if r.current {
					return r.name, gold
				}
				return r.name, ink
			}},
		{key: "traits", label: "특성", left: true, size: -2,
			get: func(r *deputyRow) (string, float64) { return "", float64(len(r.traits)) },
			show: func(r *deputyRow, ink color.Color, w float64) (string, color.Color) {
				return g.traitList(r.traits, 13, w-10), ink
			}},
	}
	if g.deputyList.kind == 0 {
		cols = append(cols,
			column[deputyRow]{key: "class", label: "병종", w: 96, text: true, get: func(r *deputyRow) (string, float64) { return g.className(r.class), 0 }},
			column[deputyRow]{key: "level", label: "레벨", w: 60, get: func(r *deputyRow) (string, float64) { return fmt.Sprint(r.level), float64(r.level) }})
	}
	for _, s := range deputyViews[g.deputyList.kind].stats {
		cols = append(cols, column[deputyRow]{key: s.key, label: s.label, w: 74,
			get: func(r *deputyRow) (string, float64) {
				d := s.get(r.stats, r.attrs) - s.get(base, attrs)
				return fmt.Sprintf("%+d", d), float64(d)
			},
			show: func(r *deputyRow, ink color.Color, w float64) (string, color.Color) {
				d := s.get(r.stats, r.attrs) - s.get(base, attrs)
				switch {
				case d == 0:
					return "—", muted
				case d > 0 && r.reason == "":
					return fmt.Sprintf("%+d", d), good
				}
				return fmt.Sprintf("%+d", d), ink
			}})
	}
	return cols
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

// deputyWindow draws the window's body inside the overlay frame at (x, y).
func (g *Game) deputyWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.View()
	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	base, attrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return
	}
	rows := g.deputyRows(o)
	pick := g.deputyPick
	at := slices.IndexFunc(rows, func(r deputyRow) bool { return r.id == pick })
	var picked *deputyRow
	if at < 0 {
		pick = of.Deputy
	} else {
		cand := rows[at]
		picked = &cand
	}
	left, right := x+32, x+w-32
	g.label(dst, "장수를 누르면 위에 그 장수를 부관으로 두었을 때의 능력치가 나옵니다. 금색 이름이 지금 부관이고, 머리글을 누르면 정렬합니다.", float64(left), float64(y+48), 14, muted)
	bw := 100
	for i, v := range deputyViews {
		idx := i
		var click func()
		if g.deputyList.kind != idx {
			click = func() { g.deputyList.kind, g.deputyList.top = idx, 0 }
		}
		bx := right - (len(deputyViews)-i)*(bw+6) + 6
		g.btn(dst, image.Rect(bx, y+74, bx+bw, y+104), v.title, click)
	}

	// the officer: its stats in the view with the picked deputy, green where the deputy raises them
	top := image.Rect(left, y+116, right, y+282)
	rect(dst, float32(top.Min.X), float32(top.Min.Y), float32(top.Dx()), float32(top.Dy()), color.NRGBA{R: 255, G: 240, B: 210, A: 12})
	g.portrait(dst, of.ID, of.Class, float64(top.Min.X+12), float64(top.Min.Y+12), 76)
	g.label(dst, g.name(of.ID), float64(top.Min.X+100), float64(top.Min.Y+14), 22, gold)
	sub := "부관 없음"
	if pick != "" {
		sub = "부관 " + g.name(pick)
		if pick != of.Deputy {
			sub += " (미리 보기)"
		}
	}
	g.label(dst, fmt.Sprintf("%s · Lv%d · %s", g.className(of.Class), of.Level, sub), float64(top.Min.X+100), float64(top.Min.Y+50), 14, ink)
	if picked != nil { // the picked candidate: why it cannot serve, or what naming it does besides
		msg, c := picked.note, gold
		if picked.reason != "" {
			msg, c = g.name(pick)+": "+picked.reason, bad
		}
		g.label(dst, msg, float64(top.Min.X+100), float64(top.Min.Y+74), 13, c)
	}
	ws, wa := base, attrs
	if pick != "" {
		ws, wa, _ = g.S.Engine.DeputyStats(of.ID, pick)
	}
	sx := top.Min.X + 360
	for _, s := range deputyViews[g.deputyList.kind].stats {
		now, bare := s.get(ws, wa), s.get(base, attrs)
		g.label(dst, s.label, float64(sx), float64(top.Min.Y+20), 13, muted)
		c := ink
		if now > bare {
			c = good
		}
		g.label(dst, fmt.Sprint(now), float64(sx), float64(top.Min.Y+44), 20, c)
		if now != bare {
			g.label(dst, fmt.Sprintf("%+d", now-bare), float64(sx), float64(top.Min.Y+72), 13, c)
		}
		sx += 92
	}
	// the traits the picked candidate brings, every one, as badges that open their description
	bx, by := float64(top.Min.X+12), float64(top.Min.Y+108)
	if picked == nil {
		g.label(dst, "장수를 누르면 그 장수가 부관으로서 주는 특성이 여기에 모두 나옵니다.", bx, by+2, 13, muted)
	} else if len(picked.traits) == 0 {
		g.label(dst, g.name(pick)+"이(가) 주는 특성은 없습니다.", bx, by+2, 13, muted)
	}
	if picked != nil {
		for i, t := range picked.traits {
			c := equipChip{"deputy", t}
			if w := traitIconSize + 20 + g.width(g.traitName(t), 14); bx+w > float64(top.Max.X-12) {
				if by += 28; by+traitIconSize > float64(top.Max.Y-4) {
					g.label(dst, fmt.Sprintf("외 %d", len(picked.traits)-i), bx, by-26, 13, muted)
					break
				}
				bx = float64(top.Min.X + 12)
			}
			bx = g.chip(dst, c, bx, by, func() { g.deputyInfo = c }) + 14
		}
	}

	// the table
	drawTable(g, dst, &g.deputyList.tableList, table[deputyRow]{
		cols: g.deputyColumns(base, attrs), rowH: 40, headH: 34, size: 15, center: true,
		rank:     func(r *deputyRow) int { return b2i(r.reason != "") },
		dim:      func(r *deputyRow) bool { return r.reason != "" },
		selected: func(r *deputyRow) bool { return r.id == pick },
		pick:     func(r *deputyRow) { g.deputyPick = r.id },
	}, rows, image.Rect(left, y+298, right, y+h-60))

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
	if appoint != nil {
		label = g.name(pick) + " 임명"
	}
	g.btn(dst, image.Rect(right-420, y+h-56, right-220, y+h-18), label, appoint)
	g.btn(dst, image.Rect(right-200, y+h-56, right, y+h-18), "부관 해제", release)
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

// updateDeputyList scrolls the deputy table by the wheel.
func (g *Game) updateDeputyList() bool {
	if g.overlay() != "deputy" {
		return false
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.deputyList.top -= int(wheel)
		return true
	}
	return false
}
