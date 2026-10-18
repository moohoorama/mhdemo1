package gui

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

// The unit list (docs_old/plans/PLAN_hud-panel.md): one line per unit. The HUD panel's tab keeps
// to name, troops, tactics points and order; the roster window (기타 › 병력일람) fills the screen
// with four views in larger text. A header cell sorts by it: none, descending, ascending.
const (
	unitToolsH   = 38
	unitPadBelow = 10
)

// unitList is the state of one list: its filter, view, first row shown and sort.
type unitList struct {
	filter    string // ally, enemy or all ("" is ally)
	kind, top int
	order     unitSort
}

// unitTable is how a list is laid out: its columns, their widths and the text size. pick runs
// on a click on a living unit's row.
type unitTable struct {
	cells       []unitCell
	nameW       int
	widths      []int
	rowH, headH int
	size        float64
	center      bool
	pick        func(id string)
}

func (t unitTable) width() int {
	w := t.nameW
	for _, cw := range t.widths {
		w += cw
	}
	return w
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
	return u.Faction == "ally" && !u.AI && u.HP > 0 && !u.Done && u.ActsIn(o.Turn)
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

// unitSort orders the list by a cell key: none, descending or ascending.
type unitSort struct {
	key string
	dir int // 0 none, 1 ascending, 2 descending
}

// cycle is a click on a header: a new key starts descending, then ascending, then none.
func (s unitSort) cycle(key string) unitSort {
	switch {
	case s.key != key:
		return unitSort{key, 2}
	case s.dir == 2:
		return unitSort{key, 1}
	}
	return unitSort{}
}

// unitListRows are the units the list shows: the filter's, sorted by a cell, the retired last.
func (g *Game) unitListRows(o core.Observation, l *unitList, cells []unitCell) []core.UnitView {
	rows := []core.UnitView{}
	for _, u := range o.UnitViews {
		if l.filter == "all" || u.Faction == l.filter || l.filter == "" && u.Faction == "ally" {
			rows = append(rows, u)
		}
	}
	sortBy := func(a, b *core.UnitView) bool { return false }
	if s := l.order; s.dir != 0 {
		if s.key == "name" {
			sortBy = func(a, b *core.UnitView) bool { return a.Name < b.Name }
		}
		for _, c := range cells {
			if c.key != s.key {
				continue
			}
			c := c
			sortBy = func(a, b *core.UnitView) bool {
				at, an := c.get(g, o, a)
				bt, bn := c.get(g, o, b)
				if c.text {
					return at < bt
				}
				return an < bn
			}
		}
		if s.dir == 2 {
			asc := sortBy
			sortBy = func(a, b *core.UnitView) bool { return asc(b, a) }
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := &rows[i], &rows[j]
		if (a.HP > 0) != (b.HP > 0) {
			return a.HP > 0
		}
		return sortBy(a, b)
	})
	return rows
}

// unitSortKnown drops a sort whose cell the list does not show.
func (l *unitList) unitSortKnown(cells []unitCell) {
	if l.order.key == "name" {
		return
	}
	for _, c := range cells {
		if c.key == l.order.key {
			return
		}
	}
	l.order = unitSort{}
}

// unitListHeight is the tab's content height: the tools, the header and the rows that fit.
func (g *Game) unitListHeight(o core.Observation, room int) int {
	t := unitPanelTable
	return min(room, unitToolsH+t.headH+max(3, len(g.unitListRows(o, &g.unitsList, t.cells)))*t.rowH+unitPadBelow)
}

// updateUnitList scrolls the list by the wheel while the pointer is over it (anywhere, for the
// roster window).
func (g *Game) updateUnitList() bool {
	if g.S.Engine == nil || len(g.modals) > 0 || g.dragging {
		return false
	}
	l := &g.unitsList
	switch {
	case g.overlay == "roster":
		l = &g.rosterList
	case g.hudShown == "units" && g.overlay == "":
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

// unitTable draws the header and as many rows as fit in r, with a scrollbar when more exist.
func (g *Game) unitTable(dst *ebiten.Image, o core.Observation, l *unitList, t unitTable, r image.Rectangle) {
	l.unitSortKnown(t.cells)
	x, hy := r.Min.X, r.Min.Y
	g.unitHeader(dst, l, "name", "장수", image.Rect(x, hy, x+t.nameW, hy+t.headH), t.size-1, t.center)
	cx := x + t.nameW
	for i, c := range t.cells {
		g.unitHeader(dst, l, c.key, c.label, image.Rect(cx, hy, cx+t.widths[i], hy+t.headH), t.size-1, t.center)
		cx += t.widths[i]
	}
	rect(dst, float32(x), float32(hy+t.headH-1), float32(t.width()), 1, goldDim)

	rows := g.unitListRows(o, l, t.cells)
	fit := max(1, (r.Dy()-t.headH-unitPadBelow)/t.rowH)
	l.top = max(0, min(l.top, len(rows)-fit))
	ry := hy + t.headH
	for i := l.top; i < len(rows) && i < l.top+fit; i++ {
		g.unitRow(dst, o, l, t, &rows[i], x, ry+(i-l.top)*t.rowH)
	}
	if len(rows) == 0 {
		g.label(dst, "해당하는 장수가 없습니다.", float64(x+4), float64(ry+8), t.size, muted)
	}
	if len(rows) > fit {
		track := image.Rect(r.Max.X-6, ry, r.Max.X, ry+fit*t.rowH)
		rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
		th := max(18, track.Dy()*fit/len(rows))
		ty := track.Min.Y + (track.Dy()-th)*l.top/(len(rows)-fit)
		rect(dst, float32(track.Min.X), float32(ty), float32(track.Dx()), float32(th), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
	}
}

// unitHeader draws a header cell with the sort mark and registers its click.
func (g *Game) unitHeader(dst *ebiten.Image, l *unitList, key, label string, r image.Rectangle, size float64, center bool) {
	g.addButton(r, label, func() { l.order = l.order.cycle(key) })
	c := muted
	sorted := l.order.key == key && l.order.dir != 0
	if sorted {
		c = gold
		rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()-2), float32(r.Dy()), color.NRGBA{R: 255, G: 210, B: 120, A: 18})
	} else if hovered(r) {
		c = ink
	}
	ty := float64(r.Min.Y) + (float64(r.Dy())-size*1.25)/2
	lx := float64(r.Min.X + 4)
	if center {
		lx = float64(r.Min.X) + (float64(r.Dx())-g.width(label, size))/2
	}
	g.label(dst, label, lx, ty, size, c)
	if sorted {
		g.sortMark(dst, float32(lx+g.width(label, size)+9), float32(ty+size*.6), l.order.dir == 1, gold)
	}
}

// sortMark is a small triangle centred on (cx, cy): its tip up for ascending, down for
// descending.
func (g *Game) sortMark(dst *ebiten.Image, cx, cy float32, up bool, c color.Color) {
	var p vector.Path
	if up {
		p.MoveTo(cx-4, cy+3)
		p.LineTo(cx+4, cy+3)
		p.LineTo(cx, cy-3)
	} else {
		p.MoveTo(cx-4, cy-3)
		p.LineTo(cx+4, cy-3)
		p.LineTo(cx, cy+3)
	}
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	fillPath(dst, &p, op)
}

// boldFace is the sans face at the weight used for the values a list is sorted by.
func (g *Game) boldFace(size float64) *text.GoTextFace {
	f := &text.GoTextFace{Source: g.font, Size: size}
	f.SetVariation(text.MustParseTag("wght"), 800)
	return f
}

// unitRow draws one unit's row and registers its click.
func (g *Game) unitRow(dst *ebiten.Image, o core.Observation, l *unitList, t unitTable, u *core.UnitView, x, y int) {
	r := image.Rect(x, y, x+t.width(), y+t.rowH)
	var click func()
	if u.HP > 0 {
		id := u.ID
		click = func() { t.pick(id) }
	}
	g.addButton(r, u.Name, click)
	dim := u.HP <= 0 || u.Faction == "ally" && !u.AI && !unitOrderable(o, u)
	ink2 := ink
	if dim {
		ink2 = color.NRGBA{R: 150, G: 142, B: 128, A: 255}
	}
	switch {
	case g.selected == u.ID:
		strokeRect(dst, float32(r.Min.X), float32(r.Min.Y+1), float32(r.Dx()-2), float32(r.Dy()-2), 1.5, gold)
	case hovered(r) && click != nil:
		rect(dst, float32(r.Min.X), float32(r.Min.Y+1), float32(r.Dx()-2), float32(r.Dy()-2), color.NRGBA{R: 255, G: 255, B: 255, A: 16})
	}
	rect(dst, float32(r.Min.X), float32(r.Max.Y-1), float32(r.Dx()-2), 1, color.NRGBA{R: 255, G: 255, B: 255, A: 14})

	ty := float64(y) + (float64(t.rowH)-t.size*1.25)/2
	nc := teamColor(u)
	if dim {
		nc.A = 150
	}
	name := g.clip(u.Name, t.size+1, float64(t.nameW-16))
	g.label(dst, name, g.cellX(t, x, t.nameW, name, t.size+1), ty-1, t.size+1, nc)
	if unitOrderable(o, u) && !t.center {
		fillCircle(dst, float32(x+t.nameW-8), float32(y+t.rowH/2), 3, good)
	}
	cx := x + t.nameW
	for i, c := range t.cells {
		w := t.widths[i]
		text, _ := c.get(g, o, u)
		text = g.clip(text, t.size, float64(w-6))
		tx := g.cellX(t, cx, w, text, t.size)
		switch {
		case c.key == "order" && text == "O":
			g.label(dst, text, tx, ty, t.size, good)
		case l.order.key == c.key && l.order.dir != 0:
			write(dst, text, g.boldFace(t.size), tx, ty, ink2)
		default:
			g.label(dst, text, tx, ty, t.size, ink2)
		}
		if c.key == "hp" {
			k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
			bar(dst, float32(cx+4), float32(y+t.rowH-3), float32(w-12), 2, k, hpColor(k))
		}
		cx += w
	}
}

// cellX is where the text starts in a cell of width w at x: centred or left.
func (g *Game) cellX(t unitTable, x, w int, s string, size float64) float64 {
	if t.center {
		return float64(x) + (float64(w)-g.width(s, size))/2
	}
	return float64(x + 4)
}

// clip shortens s with an ellipsis to the width w at the size.
func (g *Game) clip(s string, size, w float64) string {
	if g.width(s, size) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && g.width(string(r)+"…", size) > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}
