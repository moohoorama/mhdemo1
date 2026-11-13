package gui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// The unit list (docs_old/plans/PLAN_hud-panel.md): one line per unit. The HUD panel's tab keeps
// to name, troops, tactics points and order; the roster window (기타 › 병력일람) fills the screen
// with four views in larger text. A header cell sorts by it: none, descending, ascending.
const unitToolsH = 38

// unitList is the state of one list: its filter, view, first row shown and sort.
type unitList struct {
	filter string // ally, enemy or all ("" is ally)
	kind   int
	tableList
}

// unitTable is how a list is laid out: its columns, their widths (none: the cells share what the
// name leaves) and the text size. pick runs on a click on a living unit's row.
type unitTable struct {
	cells       []unitCell
	nameW       int
	widths      []int
	rowH, headH int
	size        float64
	center      bool
	pick        func(id string)
}

// unitCell is one column of the grid: how a unit fills it, and whether it sorts as text.
type unitCell struct {
	key, label string
	get        func(g *Game, o core.Observation, u *core.UnitView) (string, float64)
	text       bool
}

func statCell(key, label string, get func(core.Stats) int) unitCell {
	return unitCell{key: key, label: label, get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		v := get(u.Stats)
		return fmt.Sprint(v), float64(v)
	}}
}

func attributeCell(i int, label string) unitCell {
	return unitCell{key: "attr" + label, label: label, get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		return fmt.Sprint(u.Attributes[i]), float64(u.Attributes[i])
	}}
}

func equipmentCell(slot, label string) unitCell {
	return unitCell{key: slot, label: label, text: true, get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		for _, c := range o.Characters {
			if c.ID == u.Character {
				if eq, ok := g.S.Data.Equipment[c.Equipment[slot]]; ok {
					return eq.Name, 0
				}
			}
		}
		return "-", 0
	}}
}

func countCell(key, label string, get func(*core.UnitView) int) unitCell {
	return unitCell{key: key, label: label, get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		v := get(u)
		return fmt.Sprint(v), float64(v)
	}}
}

// unitOrderable reports whether the player may still give the unit an order this turn.
func unitOrderable(o core.Observation, u *core.UnitView) bool {
	return u.Faction == "ally" && !u.AI && u.HP > 0 && !u.Done && u.ActsIn(o.Turn, o.Army)
}

var (
	classCell = unitCell{key: "class", label: "병종", text: true, get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		return g.className(u.Class), 0
	}}
	hpCell = unitCell{key: "hp", label: "병력", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		return fmt.Sprintf("%d/%d", u.HP, u.Stats.MaxHP), float64(u.HP)
	}}
	mpCell = unitCell{key: "mp", label: "병법치", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		return fmt.Sprintf("%d/%d", u.MP, u.Stats.MaxMP), float64(u.MP)
	}}
	orderCell = unitCell{key: "order", label: "명령", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
		switch {
		case unitOrderable(o, u):
			return "O", 2
		case u.Faction == "ally" && !u.AI:
			return "X", 1
		}
		return "-", 0
	}}
)

// unitPanelTable is the HUD tab's list: name, troops, tactics points and order.
var unitPanelTable = unitTable{
	cells:  []unitCell{hpCell, mpCell, orderCell},
	nameW:  92,
	widths: []int{86, 68, 44},
	rowH:   26, headH: 26, size: 13,
}

// unitViewKinds are the roster window's views, each with the cells after the name.
var unitViewKinds = []struct {
	title string
	cells []unitCell
}{
	{"기본", []unitCell{classCell, countCell("level", "레벨", func(u *core.UnitView) int { return u.Level }), hpCell, mpCell, orderCell}},
	{"전투력", []unitCell{
		statCell("attack", "공격", func(s core.Stats) int { return s.Attack }),
		statCell("defense", "방어", func(s core.Stats) int { return s.Defense }),
		statCell("mind", "정신", func(s core.Stats) int { return s.Mind }),
		statCell("agility", "순발", func(s core.Stats) int { return s.Agility }),
		statCell("morale", "사기", func(s core.Stats) int { return s.Morale }),
		statCell("recovery", "병법회복", func(s core.Stats) int { return s.Recovery }),
	}},
	{"열전", []unitCell{
		attributeCell(0, "통솔"), attributeCell(1, "무력"), attributeCell(2, "지력"),
		attributeCell(3, "민첩"), attributeCell(4, "운"), attributeCell(5, "매력"),
	}},
	{"장비", []unitCell{
		equipmentCell("weapon", "무기"), equipmentCell("armor", "방어구"), equipmentCell("accessory", "장신구"),
		countCell("traits", "특성", func(u *core.UnitView) int { return len(u.Traits) }),
		countCell("skills", "기술", func(u *core.UnitView) int { return len(u.Skills) }),
		countCell("points", "특성치", func(u *core.UnitView) int { return u.Points }),
	}},
}

var unitFilters = []struct{ id, title string }{{"ally", "아군"}, {"enemy", "적군"}, {"all", "전체"}}

// unitListRows are the units the list shows: the filter's, in the order the observation has them.
func unitListRows(o core.Observation, l *unitList) []core.UnitView {
	rows := []core.UnitView{}
	for _, u := range o.UnitViews {
		if l.filter == "all" || u.Faction == l.filter || l.filter == "" && u.Faction == "ally" {
			rows = append(rows, u)
		}
	}
	return rows
}

// table is the list as a table: the name, then the cells, the retired last and dim.
func (t unitTable) table(g *Game, o core.Observation) table[core.UnitView] {
	name := column[core.UnitView]{key: "name", label: "장수", w: t.nameW, text: true, size: 1,
		get: func(u *core.UnitView) (string, float64) { return u.Name, 0 },
		show: func(u *core.UnitView, ink color.Color, w float64) (string, color.Color) {
			nc := teamColor(u)
			if ink == color.Color(dimInk) {
				nc.A = 150
			}
			return g.clip(u.Name, t.size+1, w-10), nc
		},
	}
	if !t.center {
		name.after = func(dst *ebiten.Image, u *core.UnitView, cell image.Rectangle) {
			if unitOrderable(o, u) {
				fillCircle(dst, float32(cell.Max.X-8), float32(cell.Min.Y+cell.Dy()/2), 3, good)
			}
		}
	}
	cols := []column[core.UnitView]{name}
	for i, c := range t.cells {
		col := column[core.UnitView]{key: c.key, label: c.label, text: c.text,
			get: func(u *core.UnitView) (string, float64) { return c.get(g, o, u) }}
		if i < len(t.widths) {
			col.w = t.widths[i]
		}
		switch c.key {
		case "class":
			col.show = func(u *core.UnitView, ink color.Color, w float64) (string, color.Color) {
				s, _ := c.get(g, o, u)
				if u.Faction == "ally" && !u.AI && g.canPromote(u.Def, u.Class, u.Level) {
					return s, promoteC
				}
				return s, ink
			}
		case "order":
			col.show = func(u *core.UnitView, ink color.Color, w float64) (string, color.Color) {
				if s, _ := c.get(g, o, u); s == "O" {
					return s, good
				} else {
					return s, ink
				}
			}
		case "hp":
			col.after = func(dst *ebiten.Image, u *core.UnitView, cell image.Rectangle) {
				k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
				bar(dst, float32(cell.Min.X+4), float32(cell.Max.Y-3), float32(cell.Dx()-12), 2, k, hpColor(k))
			}
		}
		cols = append(cols, col)
	}
	return table[core.UnitView]{
		cols: cols, rowH: t.rowH, headH: t.headH, size: t.size, center: t.center,
		empty:    "해당하는 장수가 없습니다.",
		rank:     func(u *core.UnitView) int { return b2i(u.HP <= 0) },
		open:     func(u *core.UnitView) bool { return u.HP > 0 },
		dim:      func(u *core.UnitView) bool { return u.HP <= 0 || u.Faction == "ally" && !u.AI && !unitOrderable(o, u) },
		selected: func(u *core.UnitView) bool { return g.selected == u.ID },
		pick:     func(u *core.UnitView) { t.pick(u.ID) },
	}
}

// unitListHeight is the tab's content height: the tools, the header and the rows that fit.
func (g *Game) unitListHeight(o core.Observation, room int) int {
	t := unitPanelTable
	return min(room, unitToolsH+t.headH+max(3, len(unitListRows(o, &g.unitsList)))*t.rowH+tablePadBelow)
}

// updateUnitList scrolls the list by the wheel while the pointer is over it (anywhere, for the
// roster window).
func (g *Game) updateUnitList() bool {
	if g.S.Engine == nil || len(g.modals) > 0 || g.dragging {
		return false
	}
	l := &g.unitsList
	switch {
	case g.overlay() == "roster":
		l = &g.rosterList
	case g.hudShown == "units" && g.overlay() == "":
		if x, y := cursorPos(); !image.Pt(x, y).In(g.hudBody) {
			return false
		}
	default:
		return false
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		l.top -= int(wheel)
		return true
	}
	return false
}

// hudUnits draws the tab.
func (g *Game) hudUnits(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	t := unitPanelTable
	t.pick = g.focusOfficer
	x := r.Min.X + 10
	g.unitFilterRow(dst, &g.unitsList, image.Rect(x, r.Min.Y+6, r.Max.X-10, r.Min.Y+32))
	g.unitTable(dst, o, &g.unitsList, t, image.Rect(x, r.Min.Y+unitToolsH, r.Max.X-10, r.Max.Y))
}

// unitFilterRow draws the three filter buttons across r.
func (g *Game) unitFilterRow(dst *ebiten.Image, l *unitList, r image.Rectangle) {
	w := (r.Dx() - 12) / 3
	for i, f := range unitFilters {
		id := f.id
		var click func()
		if l.filter != id && !(l.filter == "" && id == "ally") {
			click = func() { l.filter, l.top = id, 0 }
		}
		bx := r.Min.X + i*(w+6)
		g.btn(dst, image.Rect(bx, r.Min.Y, bx+w, r.Max.Y), f.title, click)
	}
}

// unitTable draws the list's table in r.
func (g *Game) unitTable(dst *ebiten.Image, o core.Observation, l *unitList, t unitTable, r image.Rectangle) {
	drawTable(g, dst, &l.tableList, t.table(g, o), unitListRows(o, l), r)
}
