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
	where           string  // 대기, 출전 or whose deputy it is now
	charm           int     // the candidate's own charm
	weight          float64 // the share of the deputy in the unit, 0 when it cannot be named
	gain            string  // the attribute it raises most, as "지력 +13", and how many more it raises
	gainV           int
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

// deputyRows are the officers who could serve the selected one, each with where it is, its charm, the weight
// it would get and the attributes it would raise.
func (g *Game) deputyRows(o core.Observation) []deputyRow {
	actor := g.selected
	_, alone, err := g.S.Engine.DeputyStats(actor, "")
	if err != nil {
		return nil
	}
	of := g.officer(o, actor)
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
		r.current = of != nil && of.Deputy == c.ID
		lead := g.servedBy(o, c.ID)
		switch err := g.S.Engine.DeputyCheck(actor, c.ID); {
		case r.current:
		case err == nil:
			if slices.Contains(o.Deployment, c.ID) {
				r.note = "출전 중입니다. 임명하면 전장에서 빠집니다."
			}
		case lead != "":
			r.reason = g.name(lead) + "의 부관이라 부관이 될 수 없습니다."
		case c.Deputy != "":
			r.reason = "부관을 거느린 장수라 부관이 될 수 없습니다."
		case err.Error() == "LordDeputy":
			r.reason = "군주라 부관이 될 수 없습니다."
		case err.Error() == "RequiredDeputy":
			r.reason = "이번 전투 필수 출전·복병 장수라 부관이 될 수 없습니다."
		default:
			r.reason = koreanError(err.Error())
		}
		switch {
		case lead != "":
			r.where = g.name(lead) + " 부관"
		case slices.Contains(o.Deployment, c.ID):
			r.where = "출전"
		default:
			r.where = "대기"
		}
		_, own, _ := g.S.Engine.DeputyStats(c.ID, "")
		r.charm = own[5]
		if r.reason == "" {
			r.weight, _, _ = g.S.Engine.DeputyBlend(actor, c.ID)
		}
		ups := 0
		for k, a := range attrNames {
			if d := at[k] - alone[k]; d > 0 {
				ups++
				if d > r.gainV {
					r.gain, r.gainV = fmt.Sprintf("%s +%d", a, d), d
				}
			}
		}
		if ups > 1 {
			r.gain += fmt.Sprintf(" 외 %d", ups-1)
		}
		rows = append(rows, r)
	}
	return rows
}

// deputyRank keeps the deputy now on top and those who cannot serve at the bottom, whatever the sort.
func deputyRank(r *deputyRow) int {
	switch {
	case r.current:
		return 0
	case r.reason != "":
		return 2
	}
	return 1
}

// deputyColumns are the candidates' table: the face and name, where they are now (the deputy now on a blue
// plate), their charm, the weight with its bar, the attribute raised most and how many traits they bring.
func (g *Game) deputyColumns() []column[deputyRow] {
	return []column[deputyRow]{
		{key: "name", label: "후보", w: 120, text: true, pad: 36,
			get: func(r *deputyRow) (string, float64) { return r.name, 0 },
			after: func(dst *ebiten.Image, r *deputyRow, cell image.Rectangle) {
				g.portrait(dst, r.id, r.class, float64(cell.Min.X+4), float64(cell.Min.Y+(cell.Dy()-30)/2), 30)
			}},
		{key: "where", label: "지금", w: 88, text: true,
			get: func(r *deputyRow) (string, float64) { return r.where, 0 },
			show: func(r *deputyRow, ink color.Color, w float64) (string, color.Color) {
				if r.current {
					return "", ink
				}
				return r.where, muted
			},
			after: func(dst *ebiten.Image, r *deputyRow, cell image.Rectangle) {
				if r.current {
					g.depPlate(dst, "현재 부관", float64(cell.Min.X+4), float64(cell.Min.Y+(cell.Dy()-20)/2))
				}
			}},
		{key: "charm", label: "매력", w: 52,
			get: func(r *deputyRow) (string, float64) { return fmt.Sprint(r.charm), float64(r.charm) }},
		{key: "weight", label: "가중치", w: 100, pad: 44,
			get: func(r *deputyRow) (string, float64) {
				if r.reason != "" {
					return "—", 0
				}
				return fmt.Sprintf("%.0f%%", r.weight*100), r.weight
			},
			show: func(r *deputyRow, ink color.Color, w float64) (string, color.Color) {
				if r.reason != "" {
					return "—", ink
				}
				return fmt.Sprintf("%.0f%%", r.weight*100), gold
			},
			after: func(dst *ebiten.Image, r *deputyRow, cell image.Rectangle) {
				if r.reason == "" {
					bar(dst, float32(cell.Min.X+4), float32(cell.Min.Y+cell.Dy()/2-3), 38, 6, r.weight, gold)
				}
			}},
		{key: "gain", label: "오르는 능력", text: false,
			get: func(r *deputyRow) (string, float64) { return r.gain, float64(r.gainV) },
			show: func(r *deputyRow, ink color.Color, w float64) (string, color.Color) {
				switch {
				case r.gain == "":
					return "—", faintInk
				case r.reason != "":
					return r.gain, ink
				}
				return r.gain, good
			}},
		{key: "traits", label: "특성", w: 48,
			get: func(r *deputyRow) (string, float64) { return fmt.Sprint(len(r.traits)), float64(len(r.traits)) }},
	}
}

// deputyTable sorts the candidates by l's sort (the weight, highest first, when the list opens) and draws them in
// r: the deputy now on top on blue with a blue band, the one picked rimmed in gold, those who cannot serve dimmed
// at the bottom. Pressing a row picks it. It gives the id picked: g.deputyPick, else the deputy now, else the top.
func (g *Game) deputyTable(dst *ebiten.Image, l *tableList, of *core.Character, rows []deputyRow, r image.Rectangle) string {
	cols := g.deputyColumns()
	keepSort(l, cols)
	sortRows(rows, cols, l.order, deputyRank)
	pick := g.deputyPick
	if pick == "" {
		pick = of.Deputy
	}
	if !slices.ContainsFunc(rows, func(r deputyRow) bool { return r.id == pick }) {
		pick = ""
		if len(rows) > 0 {
			pick = rows[0].id
		}
	}
	drawTable(g, dst, l, table[deputyRow]{
		cols: cols, rowH: touchH, headH: 32, size: 14,
		empty:    "부관이 될 장수가 없습니다.",
		rank:     deputyRank,
		dim:      func(r *deputyRow) bool { return r.reason != "" },
		selected: func(r *deputyRow) bool { return r.id == pick },
		pick:     func(r *deputyRow) { g.deputyPick, g.deputyInfo, g.party.depAll = r.id, equipChip{}, false },
		back: func(dst *ebiten.Image, r *deputyRow, rr image.Rectangle) {
			if r.current {
				depBand(dst, rr, false)
			}
		},
	}, rows, r)
	return pick
}

// deputyFresh starts the candidates' table of a lead: at the top, sorted by the weight, nothing previewed.
func (g *Game) deputyFresh() {
	g.deputyPick, g.deputyInfo, g.party.depAll = "", equipChip{}, false
	g.party.deputies = tableList{order: tableSort{"weight", 2}}
}

// deputyWindow is the deputy appointment, filling the screen: the candidates' table on the left (deputyTable); the
// one picked on the right with the lead's battle stats alone, the bonus and as a unit, and the traits the deputy
// would bring as small chips.
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
	listW := min(560, w*3/5)
	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-76), goldDim)
	g.label(dst, "부관 임명", float64(x+16), float64(y+12), 18, gold)
	sub := fmt.Sprintf("%s대 · %s 매력 %d", g.name(of.ID), g.name(of.ID), aloneAttrs[5])
	g.label(dst, sub, float64(x+28)+g.width("부관 임명", 18), float64(y+17), 13, muted)
	rect(dst, float32(x+8), float32(y+56), float32(listW-16), 1, goldDim)
	pick := g.deputyTable(dst, &g.party.deputies, of, rows, image.Rect(x+10, y+62, x+listW-8, y+h-66))
	at := slices.IndexFunc(rows, func(r deputyRow) bool { return r.id == pick })

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
			info = fmt.Sprintf("가중치 %.0f%%", picked.weight*100)
		}
		if picked.note != "" {
			info += " · " + picked.note
		}
		if picked.reason != "" {
			info = picked.reason
		}
		g.label(dst, ellipsis(g, info, 12, right-rx-60), float64(rx+54), float64(y+32), 12, muted)
		ty := float64(y + 64)
		if !g.party.depAll {
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
		}
		g.label(dst, fmt.Sprintf("부대에 더해지는 특성 %d", len(picked.traits)), float64(rx), ty, 12, muted)
		ty += 22
		if len(picked.traits) == 0 {
			g.label(dst, "없음", float64(rx), ty, 13, faintInk)
		}
		chips := image.Rect(rx, int(ty), right, y+h-68)
		folded := y + 64 + 26 + len(statRows)*24 + 6 + 22 // where the chips start under the stats
		lines := max(1, (y+h-68-folded+smallChipPitch-smallChipH)/smallChipPitch)
		g.smallChips(dst, picked.traits, chips, lines, g.party.depAll,
			func() { g.party.depAll = !g.party.depAll },
			func(id string) { g.deputyInfo = equipChip{"deputy", id} })
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
