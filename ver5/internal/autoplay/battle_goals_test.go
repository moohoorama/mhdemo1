package autoplay

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestGoalInputsDescribeObjectiveWithoutChangingInitialPolicy(t *testing.T) {
	d, e := battleModelFixture(t)
	d.Stages[0].Victory = []content.Goal{{If: []string{"at:hero:1"}}}
	d.Stages[0].Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 1, Y: 0, W: 1, H: 1}}}
	m := NewBattleModel(8, 1)
	m.ContentHash = d.Hash
	before := e.Snapshot()
	old := m.policy(m.inputs(d, e).Features)
	m.EnableGoalInputs()
	m.EnableGoalInputs()
	require.NoError(t, m.Validate())
	in := m.inputs(d, e)
	next := m.policy(in.Features)
	require.Len(t, old, len(next))
	checked := false
	for i, c := range in.Commands {
		assert.InDelta(t, old[i], next[i], 1e-12)
		require.Len(t, in.Features[i], 128+BattleGoalLen)
		g := in.Features[i][128:]
		assert.Equal(t, 1.0, g[0])
		if c.Actor == "hero" {
			assert.Equal(t, 1.0, g[1])
			if c.Kind == "move" && c.Y < 2 && c.X == 1 {
				assert.Positive(t, g[3])
				checked = true
			}
		} else {
			assert.Zero(t, g[1])
		}
	}
	assert.True(t, checked)
	assert.Equal(t, before, e.Snapshot())
	path := filepath.Join(t.TempDir(), "goal.json")
	require.NoError(t, m.Save(path))
	loaded, err := LoadBattleModel(path)
	require.NoError(t, err)
	assert.True(t, loaded.GoalInputs)
	assert.Equal(t, next, loaded.policy(loaded.inputs(d, e).Features))
}
