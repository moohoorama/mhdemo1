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
		left := map[string]bool{}
		for _, n := range d.Nodes {
			if n.ID == last.Node {
				break
			}
			if n.Kind == "leave" {
				left[n.Character] = true
			}
		}
		for _, n := range d.Nodes {
			if n.Kind == "join" && d.ChapterIndex(n.ID) < d.ChapterIndex(last.Node) && !left[n.Character] {
				assert.NotNil(t, e.char(n.Character), n.Character)
			}
		}
		for id := range left {
			assert.Nil(t, e.char(id), id+" has left")
		}
	})

	t.Run("officers wear the pieces they own", func(t *testing.T) {
		e, err := NewAt(d, 1, ways[0].Node, "")
		require.NoError(t, err)
		_, more := e.OwnerPiece()
		assert.False(t, more, "nothing is left for an owner to put on")
		worn := 0
		for id, eq := range d.Equipment {
			if o := e.char(eq.Owner); o != nil && o.Equipment[eq.Slot] == id {
				worn++
			}
		}
		assert.Positive(t, worn)
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
