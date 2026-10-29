package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// TestPlayerFollowsAChangedGoal: at the Bian River the player's side first fights Xu Rong's rear guard,
// and once Lu Bu comes and the goal turns to the east end, Liu Bei gets there.
func TestPlayerFollowsAChangedGoal(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B07" {
			d.Campaign.Start = n.ID
		}
	}
	for id, c := range d.Characters {
		if c.Playable {
			c.Level = d.Rules.StageLevel(7)
			d.Characters[id] = c
		}
	}
	e := core.New(d, 1)
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	struck := false
	for i := 0; i < 5000 && e.Observe().Phase != "result"; i++ {
		c, err := Next(e)
		require.NoError(t, err)
		struck = struck || c.Kind == "attack" && e.Observe().Round < 3
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	assert.True(t, struck, "the rear guard is fought first")
	assert.Equal(t, "victory", e.Observe().Result)
	assert.LessOrEqual(t, e.Observe().Round, 8)
}
