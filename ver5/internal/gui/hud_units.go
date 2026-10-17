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

// The unit list tab (docs/plans/PLAN_hud-panel.md): one row per unit, two lines tall. The
// left cell is the unit's name and class; the right cells are a 3×2 grid whose values depend
// on the view (basics, unit stats, biography, equipment). A cell's header sorts by it.
const (
	unitRowH     = 40
	unitCellH    = unitRowH / 2
	unitNameW    = 80
	unitCellW    = 68
	unitToolsH   = 66
	unitHeadH    = unitRowH
	unitPadBelow = 10
)

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

var orderCell = unitCell{key: "order", label: "명령", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
	switch {
	case unitOrderable(o, u):
		return "O", 2
	case u.Faction == "ally" && !u.AI:
		return "X", 1
	}
	return "-", 0
}}

// unitViewKinds are the list's views, each with the six cells of its grid (row by row).
var unitViewKinds = []struct {
	title string
	cells []unitCell
}{
	{"기본", []unitCell{
		{key: "hp", label: "병력", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
			return fmt.Sprintf("%d/%d", u.HP, u.Stats.MaxHP), float64(u.HP)
		}},
		{key: "mp", label: "병법치", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
			return fmt.Sprintf("%d/%d", u.MP, u.Stats.MaxMP), float64(u.MP)
		}},
		orderCell,
		countCell("level", "레벨", func(u *core.UnitView) int { return u.Level }),
		{key: "xp", label: "경험치", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
			if u.Faction != "ally" {
				return "-", -1
			}
			return fmt.Sprintf("%d%%", int(u.XPProgress)), u.XPProgress
		}},
		{key: "pos", label: "위치", get: func(g *Game, o core.Observation, u *core.UnitView) (string, float64) {
			return fmt.Sprintf("%d,%d", u.X, u.Y), float64(u.Y*1000 + u.X)
		}},
	}},
	{"부대", []unitCell{
		statCell("attack", "공격", func(s core.Stats) int { return s.Attack }),
		statCell("defense", "방어", func(s core.Stats) int { return s.Defense }),
		statCell("mind", "정신", func(s core.Stats) int { return s.Mind }),
		statCell("agility", "순발", func(s core.Stats) int { return s.Agility }),
		statCell("morale", "사기", func(s core.Stats) int { return s.Morale }),
		statCell("recovery", "회복", func(s core.Stats) int { return s.Recovery }),
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

// unitSort orders the list by a cell key: none, ascending or descending.
type unitSort struct {
	key string
	dir int // 0 none, 1 ascending, 2 descending
}

// cycle is a click on a header: a new key starts ascending, then descending, then none.
func (s unitSort) cycle(key string) unitSort {
	if s.key != key {
		return unitSort{key, 1}
	}
	if s.dir >= 2 {
		return unitSort{}
	}
	return unitSort{key, s.dir + 1}
}

// unitListRows are the units the list shows: the filter's, sorted, the retired last.
func (g *Game) unitListRows(o core.Observation) []core.UnitView {
	rows := []core.UnitView{}
	for _, u := range o.UnitViews {
		if g.unitFilter == "all" || u.Faction == g.unitFilter || g.unitFilter == "" && u.Faction == "ally" {
			rows = append(rows, u)
		}
	}
	cells := unitViewKinds[g.unitKind].cells
	sortBy := func(a, b *core.UnitView) bool { return false }
	if s := g.unitOrder; s.dir != 0 {
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

// unitSortKnown drops a sort whose cell the current view does not show.
func (g *Game) unitSortKnown() {
	if g.unitOrder.key == "name" {
		return
	}
	for _, c := range unitViewKinds[g.unitKind].cells {
		if c.key == g.unitOrder.key {
			return
		}
	}
	g.unitOrder = unitSort{}
}

// unitListHeight is the tab's content height: the tools, the header and the rows that fit.
func (g *Game) unitListHeight(o core.Observation, room int) int {
	return min(room, unitToolsH+unitHeadH+max(3, len(g.unitListRows(o)))*unitRowH+unitPadBelow)
}

// unitListRowsFit is how many rows the content rectangle holds.
func unitListRowsFit(r image.Rectangle) int {
	return max(1, (r.Dy()-unitToolsH-unitHeadH-unitPadBelow)/unitRowH)
}

// updateUnitList scrolls the list by the wheel while the pointer is over it.
func (g *Game) updateUnitList() bool {
	if g.hudShown != "units" || g.S.Engine == nil || g.overlay != "" || len(g.modals) > 0 || g.dragging {
		return false
	}
	x, y := cursorPos()
	if !image.Pt(x, y).In(g.hudBody) {
		return false
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.unitTop -= int(wheel)
		return true
	}
	return false
}

// hudUnits draws the tab.
func (g *Game) hudUnits(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	g.unitSortKnown()
	kind := unitViewKinds[g.unitKind]
	x := r.Min.X + 10

	half := (infoW - 20 - 12) / 3
	for i, f := range unitFilters {
		id := f.id
		var click func()
		if g.unitFilter != id && !(g.unitFilter == "" && id == "ally") {
			click = func() { g.unitFilter, g.unitTop = id, 0 }
		}
		bx := x + i*(half+6)
		g.btn(dst, image.Rect(bx, r.Min.Y+6, bx+half, r.Min.Y+32), f.title, click)
	}
	quarter := (infoW - 20 - 18) / 4
	for i, k := range unitViewKinds {
		idx := i
		var click func()
		if g.unitKind != idx {
			click = func() { g.unitKind = idx }
		}
		bx := x + i*(quarter+6)
		g.btn(dst, image.Rect(bx, r.Min.Y+36, bx+quarter, r.Min.Y+62), k.title, click)
	}

	// header: the same grid as the rows, every cell a sort button
	hy := r.Min.Y + unitToolsH
	g.unitHeader(dst, "name", "부대", image.Rect(x, hy, x+unitNameW, hy+unitHeadH))
	for i, c := range kind.cells {
		g.unitHeader(dst, c.key, c.label, g.unitCellRect(x, hy, i))
	}
	rect(dst, float32(x), float32(hy+unitHeadH-1), float32(unitNameW+3*unitCellW), 1, goldDim)

	rows := g.unitListRows(o)
	fit := unitListRowsFit(r)
	g.unitTop = max(0, min(g.unitTop, len(rows)-fit))
	ry := hy + unitHeadH
	for i := g.unitTop; i < len(rows) && i < g.unitTop+fit; i++ {
		g.unitRow(dst, o, &rows[i], kind.cells, x, ry+(i-g.unitTop)*unitRowH)
	}
	if len(rows) == 0 {
		g.label(dst, "해당하는 부대가 없습니다.", float64(x+4), float64(ry+12), 13, muted)
	}
	if len(rows) > fit {
		track := image.Rect(r.Max.X-16, ry, r.Max.X-10, ry+fit*unitRowH)
		rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
		th := max(18, track.Dy()*fit/len(rows))
		ty := track.Min.Y + (track.Dy()-th)*g.unitTop/(len(rows)-fit)
		rect(dst, float32(track.Min.X), float32(ty), float32(track.Dx()), float32(th), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
	}
}

// unitCellRect is the i'th grid cell (row by row, three to a row) of a row at y.
func (g *Game) unitCellRect(x, y, i int) image.Rectangle {
	cx := x + unitNameW + i%3*unitCellW
	cy := y + i/3*unitCellH
	return image.Rect(cx, cy, cx+unitCellW, cy+unitCellH)
}

// unitHeader draws a header cell with the sort mark and registers its click.
func (g *Game) unitHeader(dst *ebiten.Image, key, label string, r image.Rectangle) {
	g.addButton(r, label, func() { g.unitOrder = g.unitOrder.cycle(key) })
	c := muted
	sorted := g.unitOrder.key == key && g.unitOrder.dir != 0
	if sorted {
		c = gold
		rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()-2), float32(r.Dy()), color.NRGBA{R: 255, G: 210, B: 120, A: 18})
	} else if hovered(r) {
		c = ink
	}
	ty := float64(r.Min.Y + 3)
	if key == "name" {
		ty = float64(r.Min.Y + (r.Dy()-16)/2)
	}
	g.label(dst, label, float64(r.Min.X+4), ty, 12, c)
	if sorted {
		g.sortMark(dst, float32(r.Min.X+4)+float32(g.width(label, 12))+9, float32(ty)+8, g.unitOrder.dir == 1, gold)
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
func (g *Game) unitRow(dst *ebiten.Image, o core.Observation, u *core.UnitView, cells []unitCell, x, y int) {
	r := image.Rect(x, y, x+unitNameW+3*unitCellW, y+unitRowH)
	var click func()
	if u.HP > 0 {
		id := u.ID
		click = func() { g.focusOfficer(id) }
	}
	g.addButton(r, u.Name, click)
	dim := u.HP <= 0 || u.Faction == "ally" && !u.AI && !unitOrderable(o, u)
	ink2, soft := ink, muted
	if dim {
		ink2 = color.NRGBA{R: 150, G: 142, B: 128, A: 255}
		soft = color.NRGBA{R: 112, G: 104, B: 94, A: 255}
	}
	switch {
	case g.selected == u.ID:
		strokeRect(dst, float32(r.Min.X), float32(r.Min.Y+1), float32(r.Dx()-2), float32(r.Dy()-2), 1.5, gold)
	case hovered(r) && click != nil:
		rect(dst, float32(r.Min.X), float32(r.Min.Y+1), float32(r.Dx()-2), float32(r.Dy()-2), color.NRGBA{R: 255, G: 255, B: 255, A: 16})
	}
	rect(dst, float32(r.Min.X), float32(r.Max.Y-1), float32(r.Dx()-2), 1, color.NRGBA{R: 255, G: 255, B: 255, A: 14})

	nc := teamColor(u)
	if dim {
		nc.A = 150
	}
	g.label(dst, g.clip(u.Name, 14, unitNameW-16), float64(x+4), float64(y+3), 14, nc)
	g.label(dst, g.clip(g.className(u.Class), 11, unitNameW-8), float64(x+4), float64(y+22), 11, soft)
	if unitOrderable(o, u) {
		fillCircle(dst, float32(x+unitNameW-8), float32(y+10), 3, good)
	}
	for i, c := range cells {
		cr := g.unitCellRect(x, y, i)
		text, _ := c.get(g, o, u)
		if c.key == "order" && text == "O" {
			g.label(dst, text, float64(cr.Min.X+4), float64(cr.Min.Y+3), 13, good)
			continue
		}
		text = g.clip(text, 13, unitCellW-6)
		if g.unitOrder.key == c.key && g.unitOrder.dir != 0 {
			write(dst, text, g.boldFace(13), float64(cr.Min.X+4), float64(cr.Min.Y+3), ink2)
		} else {
			g.label(dst, text, float64(cr.Min.X+4), float64(cr.Min.Y+3), 13, ink2)
		}
		if c.key == "hp" {
			k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
			bar(dst, float32(cr.Min.X+4), float32(cr.Max.Y-2), float32(unitCellW-12), 2, k, hpColor(k))
		}
	}
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
