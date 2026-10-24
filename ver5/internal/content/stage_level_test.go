package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageLevelFollowsTheStoryOrder(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for i, id := range []string{"B02", "B05", "B06", "B07", "B08", "B09", "B10"} {
		assert.Equal(t, i+1, d.StageOrder(id), id)
		assert.Equal(t, 3+2*i, d.Rules.StageLevel(d.StageOrder(id)), id)
	}
	assert.Equal(t, 5, d.NextBattleOrder("c09-11"), "Jian Yong joins before Xindu")
}

func TestRankLeadKeepsItsShareOfStrength(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	t.Run("defaults", func(t *testing.T) {
		assert.Equal(t, 3, d.SpawnLevel(1, Spawn{Character: "troop_footman"}))
		assert.Equal(t, 4, d.SpawnLevel(1, Spawn{Character: "lubu"}))
	})
	t.Run("the lead grows with the level", func(t *testing.T) {
		boss := Spawn{Character: "lubu", Rank: "boss"}
		assert.Equal(t, 12, d.SpawnLevel(3, boss))
		assert.Equal(t, 15, d.SpawnLevel(4, boss))
		assert.Equal(t, 23, d.SpawnLevel(7, boss))
	})
	t.Run("shift", func(t *testing.T) {
		assert.Equal(t, 3, d.SpawnLevel(2, Spawn{Character: "troop_footman", Shift: -0.12}))
	})
	t.Run("absolute levels are refused", func(t *testing.T) {
		assert.Error(t, d.ValidateSpawn(Spawn{Character: "troop_footman", Level: 5}))
		assert.Error(t, d.ValidateSpawn(Spawn{Character: "troop_footman", Rank: "general"}))
	})
}
