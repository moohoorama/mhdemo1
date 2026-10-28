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
			e, err := NewAt(d, 1, w.Node)
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
		e, err := NewAt(d, 1, last.Node)
		require.NoError(t, err)
		s := e.Snapshot()
		for _, w := range ways {
			if w.Stage != "" && w.Node != last.Node {
				assert.True(t, s.Executed["won-"+w.Stage], w.Stage)
			}
		}
		for _, n := range d.Nodes {
			if n.Kind == "join" && d.ChapterIndex(n.ID) < d.ChapterIndex(last.Node) {
				assert.NotNil(t, e.char(n.Character), n.Character)
			}
		}
	})

	t.Run("officers wear the loadout", func(t *testing.T) {
		e, err := NewAt(d, 1, ways[0].Node)
		require.NoError(t, err)
		for id, eqs := range d.Campaign.Loadout {
			if o := e.char(id); o != nil {
				for _, eq := range eqs {
					assert.Equal(t, eq, o.Equipment[d.Equipment[eq].Slot], id)
				}
			}
		}
	})

	t.Run("an unknown node is refused", func(t *testing.T) {
		_, err := NewAt(d, 1, "nowhere")
		assert.EqualError(t, err, "UnknownWaypoint")
	})
}
