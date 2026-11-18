package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/content"
)

// areaStage puts an area on (8..10, 3..4): the troop at (9,4) and the one at (10,3) stand in it, the boss at (9,2)
// does not.
func areaStage(events ...content.BattleEvent) func(*content.Stage) {
	return func(s *content.Stage) {
		withEvents(events...)(s)
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 8, Y: 3, W: 3, H: 2}, Name: "fire"}}
	}
}

func TestAreaTargetsHitOnlyUnitsStandingInTheArea(t *testing.T) {
	t.Run("buff by side and area", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) {
			areaStage(do("buff", content.Action{Targets: []string{"enemy@area:1"}, Stat: "attack", Value: -30}))(s)
			s.Enemies[2].X, s.Enemies[2].Y = 4, 6 // out of the area
		})
		assert.Equal(t, -30, e.unit("troop#1").Status["attack"].Value, "inside")
		assert.Empty(t, e.unit("troop#2").Status, "outside")
		assert.Empty(t, e.unit("boss").Status, "outside")
		assert.Empty(t, e.unit("hero").Status, "inside no area, and not an enemy")
	})
	t.Run("damage by group and area", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) {
			areaStage(do("damage", content.Action{Targets: []string{"troops@area:1"}, Value: 50}))(s)
			s.Enemies[2].X, s.Enemies[2].Y = 4, 6
		})
		a, b := e.unit("troop#1"), e.unit("troop#2")
		assert.Less(t, a.HP, e.stats(a).MaxHP)
		assert.Equal(t, e.stats(b).MaxHP, b.HP)
	})
	t.Run("a bare area takes every unit in it", func(t *testing.T) {
		e := battleWith(t, areaStage(do("damage", content.Action{Targets: []string{"area:1"}, Value: 10})))
		for _, id := range []string{"troop#1", "troop#2"} {
			assert.Less(t, e.unit(id).HP, e.stats(e.unit(id)).MaxHP, id)
		}
		assert.Equal(t, e.stats(e.unit("boss")).MaxHP, e.unit("boss").HP)
	})
	t.Run("the units there when the event runs, not before", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) {
			areaStage(content.BattleEvent{ID: "later", If: []string{"round>=2"}, Do: []content.Action{{Kind: "buff", Targets: []string{"enemy@area:1"}, Stat: "defense", Value: -20}}})(s)
		})
		assert.Empty(t, e.unit("boss").Status)
		e.unit("boss").X, e.unit("boss").Y = 8, 3 // walks in before the event
		for e.state.Round < 2 {
			applyOK(t, e, Command{Kind: "end"})
		}
		assert.Equal(t, -20, e.unit("boss").Status["defense"].Value)
	})
}
