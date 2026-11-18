package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestPlainGearTierFollowsTheChapter(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	t.Run("stages take the tier of their chapter", func(t *testing.T) {
		want := map[string]int{"B01": 1, "B16": 1, "B17": 2, "B22": 2, "B27": 2, "B27V": 2}
		for _, s := range d.Stages {
			if tier, ok := want[s.ID]; ok {
				assert.Equal(t, tier, s.GearTier, s.ID)
				delete(want, s.ID)
			}
		}
		assert.Empty(t, want, "stages not found")
		assert.Equal(t, 3, d.Rules.GearTierOf(28))
	})
	t.Run("a piece steps along its line", func(t *testing.T) {
		assert.Equal(t, "iron_sword", d.GearAt("copper_sword", 2))
		assert.Equal(t, "steel_spear", d.GearAt("iron_spear", 3))
		assert.Equal(t, "iron_gloves", d.GearAt("cloth_gloves", 3))
		assert.Equal(t, "tiger_skin", d.GearAt("cotton_clothes", 3))
		assert.Equal(t, "copper_sword", d.GearAt("iron_sword", 1))
		assert.Equal(t, "gujeong_blade", d.GearAt("gujeong_blade", 3), "named gear stays")
		assert.Equal(t, "bamboo_fan", d.GearAt("bamboo_fan", 3))
	})
	t.Run("soldiers wear the stage's tier and an explicit list wins", func(t *testing.T) {
		e, err := NewAt(d, 1, "c17-battle", "")
		require.NoError(t, err)
		e.state.Learning = nil
		_, err = e.Apply(Command{Kind: "start"})
		require.NoError(t, err)
		require.Equal(t, 2, e.stageGearTier())
		plain := e.spawn("", "troop_footman", "enemy", 0, 0, content.Spawn{})
		assert.Equal(t, "iron_sword", e.char(plain.Character).Equipment["weapon"])
		assert.Equal(t, "bronze_armor", e.char(plain.Character).Equipment["armor"])
		archer := e.spawn("", "troop_archer", "enemy", 1, 0, content.Spawn{})
		assert.Equal(t, "long_bow", e.char(archer.Character).Equipment["weapon"])
		bandit := e.spawn("", "troop_bandit", "enemy", 2, 0, content.Spawn{})
		assert.Equal(t, "leather_clothes", e.char(bandit.Character).Equipment["armor"])
		explicit := e.spawn("", "troop_footman", "enemy", 3, 0, content.Spawn{Equipment: &[]string{"copper_sword", "leather_armor"}})
		assert.Equal(t, "copper_sword", e.char(explicit.Character).Equipment["weapon"])
		assert.Equal(t, "leather_armor", e.char(explicit.Character).Equipment["armor"])
	})
	t.Run("the first chapter's soldiers stay at the first tier", func(t *testing.T) {
		e, err := NewAt(d, 1, "c01-battle", "")
		require.NoError(t, err)
		e.state.Learning = nil
		_, err = e.Apply(Command{Kind: "start"})
		require.NoError(t, err)
		foot := e.spawn("", "troop_footman", "enemy", 0, 0, content.Spawn{})
		assert.Equal(t, "copper_sword", e.char(foot.Character).Equipment["weapon"])
	})
}
