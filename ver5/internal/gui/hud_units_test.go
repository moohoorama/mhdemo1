package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/core"
)

func TestUnitSortCycle(t *testing.T) {
	s := unitSort{}.cycle("hp")
	assert.Equal(t, unitSort{"hp", 1}, s)
	s = s.cycle("hp")
	assert.Equal(t, unitSort{"hp", 2}, s)
	s = s.cycle("hp")
	assert.Equal(t, unitSort{}, s)
	assert.Equal(t, unitSort{"mp", 1}, unitSort{"hp", 2}.cycle("mp"))
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
	ids := func(g *Game) []string {
		out := []string{}
		for _, u := range g.unitListRows(o) {
			out = append(out, u.ID)
		}
		return out
	}
	t.Run("filters and keeps the retired last", func(t *testing.T) {
		assert.Equal(t, []string{"a", "c", "b"}, ids(&Game{}))
		assert.Equal(t, []string{"d"}, ids(&Game{unitFilter: "enemy"}))
		assert.Equal(t, []string{"a", "c", "d", "b"}, ids(&Game{unitFilter: "all"}))
	})
	t.Run("sorts by a cell", func(t *testing.T) {
		g := &Game{unitFilter: "all", unitOrder: unitSort{"level", 2}}
		assert.Equal(t, []string{"d", "c", "a", "b"}, ids(g))
		g.unitOrder.dir = 1
		assert.Equal(t, []string{"a", "c", "d", "b"}, ids(g))
	})
	t.Run("drops a sort the view does not show", func(t *testing.T) {
		g := &Game{unitKind: 1, unitOrder: unitSort{"level", 2}}
		g.unitSortKnown()
		assert.Equal(t, unitSort{}, g.unitOrder)
		g.unitOrder = unitSort{"name", 1}
		g.unitSortKnown()
		assert.Equal(t, unitSort{"name", 1}, g.unitOrder)
	})
}
