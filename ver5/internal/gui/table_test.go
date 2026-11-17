package gui

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/core"
)

func TestTableSortCycle(t *testing.T) {
	s := tableSort{}.cycle("hp")
	assert.Equal(t, tableSort{"hp", 2}, s)
	s = s.cycle("hp")
	assert.Equal(t, tableSort{"hp", 1}, s)
	s = s.cycle("hp")
	assert.Equal(t, tableSort{}, s)
	assert.Equal(t, tableSort{"mp", 2}, tableSort{"hp", 1}.cycle("mp"))
}

func TestTableWidths(t *testing.T) {
	tb := table[int]{cols: []column[int]{{w: 100}, {}, {w: 50}, {}}}
	assert.Equal(t, []int{100, 75, 50, 75}, tb.widths(300))
}

func TestUnitListRows(t *testing.T) {
	unit := func(id, faction string, hp, level int) core.UnitView {
		u := core.UnitView{Name: id, Level: level}
		u.ID, u.Faction, u.HP = id, faction, hp
		return u
	}
	o := core.Observation{UnitViews: []core.UnitView{
		unit("a", "ally", 10, 3), unit("b", "ally", 0, 9), unit("c", "ally", 10, 5), unit("d", "enemy", 10, 7),
	}}
	tb := unitTable{cells: unitViewKinds[0].cells}.table(&Game{}, o)
	ids := func(l *unitList) []string {
		rows := unitListRows(o, l)
		sortRows(rows, tb.cols, l.order, tb.rank)
		out := []string{}
		for _, u := range rows {
			out = append(out, u.ID)
		}
		return out
	}
	t.Run("filters and keeps the retired last", func(t *testing.T) {
		assert.Equal(t, []string{"a", "c", "b"}, ids(&unitList{}))
		assert.Equal(t, []string{"d"}, ids(&unitList{filter: "enemy"}))
		assert.Equal(t, []string{"a", "c", "d", "b"}, ids(&unitList{filter: "all"}))
	})
	t.Run("sorts by a cell", func(t *testing.T) {
		l := &unitList{filter: "all", tableList: tableList{order: tableSort{"level", 2}}}
		assert.Equal(t, []string{"d", "c", "a", "b"}, ids(l))
		l.order.dir = 1
		assert.Equal(t, []string{"a", "c", "d", "b"}, ids(l))
	})
	t.Run("drops a sort the view does not show", func(t *testing.T) {
		cols := unitPanelTable.table(&Game{}, o).cols
		l := &tableList{order: tableSort{"level", 2}}
		keepSort(l, cols)
		assert.Equal(t, tableSort{}, l.order)
		l.order = tableSort{"name", 1}
		keepSort(l, cols)
		assert.Equal(t, tableSort{"name", 1}, l.order)
	})
}

func TestTableDrag(t *testing.T) {
	oldCursor, oldPointer := cursorPos, pointer
	defer func() { cursorPos, pointer = oldCursor, oldPointer }()
	x, y := 100, 300
	cursorPos = func() (int, int) { return x, y }
	g := &Game{}
	picked := 0
	pick := func() { picked++ }

	t.Run("a press let go where it was picks the row", func(t *testing.T) {
		l := &tableList{top: 5}
		g.pressTable(l, 36, pick)
		pointer.held, y = true, 304
		assert.True(t, g.updateTableDrag())
		pointer.held = false
		assert.True(t, g.updateTableDrag())
		assert.Equal(t, 1, picked)
		assert.Equal(t, 5, l.top)
		assert.Nil(t, g.tableDrag)
	})
	t.Run("a drag scrolls the rows and picks nothing", func(t *testing.T) {
		y = 300
		l := &tableList{top: 5}
		g.pressTable(l, 36, pick)
		pointer.held, y = true, 300-80
		g.updateTableDrag()
		assert.Equal(t, 7, l.top, "up two rows' height shows two rows further down")
		y = 300 + 40
		g.updateTableDrag()
		assert.Equal(t, 4, l.top)
		pointer.held = false
		g.updateTableDrag()
		assert.Equal(t, 1, picked)
	})
	t.Run("the thumb follows the pointer at once", func(t *testing.T) {
		y = 300
		l := &tableList{top: 2}
		g.tableDrag = &tableDrag{l: l, press: image.Pt(x, y), top: 2, perPx: .25}
		pointer.held, y = true, 304
		g.updateTableDrag()
		assert.Equal(t, 3, l.top)
		pointer.held = false
		g.updateTableDrag()
		assert.Equal(t, 1, picked)
	})
}

func TestTableFractionalWheel(t *testing.T) {
	l := &tableList{top: 5}
	for range 3 {
		l.scroll(-.25)
	}
	assert.Equal(t, 5, l.top)
	l.scroll(-.25)
	assert.Equal(t, 6, l.top, "small trackpad deltas accumulate to a row")
	for range 4 {
		l.scroll(.25)
	}
	assert.Equal(t, 5, l.top)
	assert.Zero(t, l.wheel)
}
