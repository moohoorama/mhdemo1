package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/core"
)

func TestUnitSortCycle(t *testing.T) {
	s := unitSort{}.cycle("hp")
	assert.Equal(t, unitSort{"hp", 2}, s)
	s = s.cycle("hp")
	assert.Equal(t, unitSort{"hp", 1}, s)
	s = s.cycle("hp")
	assert.Equal(t, unitSort{}, s)
	assert.Equal(t, unitSort{"mp", 2}, unitSort{"hp", 1}.cycle("mp"))
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
	cells := unitViewKinds[0].cells
	ids := func(l *unitList) []string {
		out := []string{}
		for _, u := range (&Game{}).unitListRows(o, l, cells) {
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
		l := &unitList{filter: "all", order: unitSort{"level", 2}}
		assert.Equal(t, []string{"d", "c", "a", "b"}, ids(l))
		l.order.dir = 1
		assert.Equal(t, []string{"a", "c", "d", "b"}, ids(l))
	})
	t.Run("drops a sort the view does not show", func(t *testing.T) {
		l := &unitList{order: unitSort{"level", 2}}
		l.unitSortKnown(unitPanelTable.cells)
		assert.Equal(t, unitSort{}, l.order)
		l.order = unitSort{"name", 1}
		l.unitSortKnown(unitPanelTable.cells)
		assert.Equal(t, unitSort{"name", 1}, l.order)
	})
}
