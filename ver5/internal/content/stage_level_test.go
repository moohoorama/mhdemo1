package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageLevelFollowsTheStoryOrder(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	levels := []int{1, 3, 4, 6, 7, 9, 10, 12, 13, 15}
	for i, id := range []string{"B01", "B02", "B03", "B04", "B05", "B06", "B07", "B08", "B09", "B10"} {
		assert.Equal(t, i+1, d.StageOrder(id), id)
		assert.Equal(t, levels[i], d.Rules.StageLevel(d.StageOrder(id)), id)
	}
	t.Run("the flat debug curve climbs one level a battle", func(t *testing.T) {
		flat := d.ForDifficulty("flat_normal").Rules
		for i := range 10 {
			assert.Equal(t, i+1, flat.StageLevel(i+1))
		}
	})
	assert.Equal(t, 4, d.NextBattleOrder("c04-c6"), "Jian Yong joins at Pengcheng, before its battle")
}

func TestRankLeadKeepsItsShareOfStrength(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	t.Run("defaults", func(t *testing.T) {
		assert.Equal(t, 1, d.SpawnLevel(1, Spawn{Character: "troop_footman"}))
		assert.Equal(t, 2, d.SpawnLevel(1, Spawn{Character: "lubu"}))
	})
	t.Run("the lead grows with the level", func(t *testing.T) {
		boss := Spawn{Character: "lubu", Rank: "boss"}
		assert.Equal(t, 7, d.SpawnLevel(3, boss))
		assert.Equal(t, 10, d.SpawnLevel(4, boss))
		assert.Equal(t, 14, d.SpawnLevel(7, boss))
	})
	t.Run("shift", func(t *testing.T) {
		assert.Equal(t, 1, d.SpawnLevel(2, Spawn{Character: "troop_footman", Shift: -0.12}))
	})
	t.Run("absolute levels are refused", func(t *testing.T) {
		assert.Error(t, d.ValidateSpawn(Spawn{Character: "troop_footman", Level: 5}))
		assert.Error(t, d.ValidateSpawn(Spawn{Character: "troop_footman", Rank: "general"}))
	})
}
