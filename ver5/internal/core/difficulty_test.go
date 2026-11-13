package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestDifficultyLeadsEnemiesAndScalesExperience(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	dd := *d
	dd.Campaign.Start = "c10-battle"
	at := func(t *testing.T, difficulty string) *Engine {
		e, err := NewGame(&dd, 1, difficulty)
		require.NoError(t, err)
		applyOK(t, e, Command{Kind: "start"})
		return e
	}
	easy, normal, hard := at(t, "easy"), at(t, "normal"), at(t, "hard")

	t.Run("enemies lead by the difficulty's share", func(t *testing.T) {
		lv := func(e *Engine) int { return e.char("yuanshao").Level }
		assert.Less(t, lv(easy), lv(normal))
		assert.Less(t, lv(normal), lv(hard))
	})
	t.Run("an allied army and a joining officer follow the difficulty like the enemy", func(t *testing.T) {
		for _, id := range []string{"gongsunzan", "zhaoyun"} {
			assert.Less(t, easy.char(id).Level, normal.char(id).Level, id)
			assert.Less(t, normal.char(id).Level, hard.char(id).Level, id)
		}
	})
	t.Run("experience follows the difficulty", func(t *testing.T) {
		assert.Greater(t, easy.xpScale(), normal.xpScale())
		assert.Greater(t, normal.xpScale(), hard.xpScale())
	})
	t.Run("a debug difficulty plays by its own rules and keeps them through a save", func(t *testing.T) {
		flat := at(t, "flat_hard")
		assert.Equal(t, 16.0, flat.data.Rules.ExpAction)
		assert.Equal(t, 24.0, hard.data.Rules.ExpAction)
		assert.Less(t, flat.char("yuanshao").Level, hard.char("yuanshao").Level, "the flat curve's B10 is 12, the story curve's 17")
		e, err := Restore(&dd, flat.Snapshot())
		require.NoError(t, err)
		assert.Equal(t, 16.0, e.data.Rules.ExpAction)
	})
	t.Run("a save keeps its difficulty", func(t *testing.T) {
		e, err := Restore(&dd, hard.Snapshot())
		require.NoError(t, err)
		assert.Equal(t, "hard", e.state.Difficulty)
	})
	t.Run("an unknown difficulty is refused", func(t *testing.T) {
		_, err := NewGame(&dd, 1, "nightmare")
		assert.Error(t, err)
	})
}
