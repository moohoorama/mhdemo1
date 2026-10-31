package gui

import (
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
		sortRows(rows, tb.cols, l.order, tb.first)
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
