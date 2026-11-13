package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestNewAt(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	ways := d.Waypoints()
	require.NotEmpty(t, ways)

	t.Run("every waypoint starts where it names", func(t *testing.T) {
		for _, w := range ways {
			e, err := NewAt(d, 1, w.Node, "")
			require.NoError(t, err, w.Node)
			o := e.Observe()
			level := d.Rules.StageLevel(d.NextBattleOrder(w.Node))
			for _, c := range o.Characters {
				assert.GreaterOrEqual(t, c.Level, level, "%s %s", w.Node, c.ID)
			}
			for id := range d.Items {
				assert.GreaterOrEqual(t, o.Inventory[id], waypointStock, "%s %s", w.Node, id)
			}
			if w.Stage != "" {
				assert.Equal(t, "scenario", o.Phase, w.Node)
				assert.Equal(t, w.Stage, d.Stages[o.Stage].ID, w.Node)
			}
			_, err = Restore(d, e.Snapshot())
			assert.NoError(t, err, w.Node)
		}
	})

	t.Run("the story before counts as played", func(t *testing.T) {
		last := ways[len(ways)-1]
		e, err := NewAt(d, 1, last.Node, "")
		require.NoError(t, err)
		s := e.Snapshot()
		for _, w := range ways {
			if w.Stage != "" && w.Node != last.Node {
				assert.True(t, s.Executed["won-"+w.Stage], w.Stage)
			}
		}
		// in story order, the last join or leave of each officer before the waypoint says whether they are with the party
		with := map[string]bool{}
		for _, n := range d.Nodes {
			if n.ID == last.Node {
				break
			}
			switch {
			case n.Kind == "leave":
				with[n.Character] = false
			case n.Kind == "join":
				with[n.Character] = true
			}
		}
		for id, in := range with {
			if in {
				assert.NotNil(t, e.char(id), id)
			} else {
				assert.Nil(t, e.char(id), id+" has left")
			}
		}
	})

	t.Run("officers wear the pieces the story has handed them", func(t *testing.T) {
		at := func(chapter int) *Engine {
			for _, w := range ways {
				if w.Chapter == chapter && w.Stage == "" {
					e, err := NewAt(d, 1, w.Node, "")
					require.NoError(t, err)
					return e
				}
			}
			t.Fatalf("no waypoint opens chapter %d", chapter)
			return nil
		}
		e := at(3)
		assert.Equal(t, "fire_manual", e.char("liubei").Equipment["accessory"])
		assert.Equal(t, "qinglong_blade", e.char("guanyu").Equipment["weapon"])

		e = at(1)
		assert.NotEqual(t, "fire_manual", e.char("liubei").Equipment["accessory"], "화계서 comes after the second battle")
		assert.Positive(t, e.state.Warehouse["fire_manual"], "the pieces to come wait in the warehouse")
	})

	t.Run("officers are promoted as far as the level allows", func(t *testing.T) {
		last := ways[len(ways)-1]
		e, err := NewAt(d, 1, last.Node, "")
		require.NoError(t, err)
		promoted := 0
		for _, id := range keys(e.charIDs) {
			o := e.char(id)
			assert.Equal(t, e.promotedAt(o.Def, o.Class, o.Level), o.Class, id)
			if d.Classes[o.Class].Tier > 0 {
				promoted++
			}
		}
		assert.Positive(t, promoted)
	})

	t.Run("the difficulty is kept", func(t *testing.T) {
		require.NotEmpty(t, d.Rules.Difficulties)
		id := d.Rules.Difficulties[len(d.Rules.Difficulties)-1].ID
		e, err := NewAt(d, 1, ways[0].Node, id)
		require.NoError(t, err)
		assert.Equal(t, id, e.Snapshot().Difficulty)
		_, err = NewAt(d, 1, ways[0].Node, "nope")
		assert.EqualError(t, err, "UnknownDifficulty")
	})

	t.Run("an unknown node is refused", func(t *testing.T) {
		_, err := NewAt(d, 1, "nowhere", "")
		assert.EqualError(t, err, "UnknownWaypoint")
	})
}
