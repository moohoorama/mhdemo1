package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/session"
)

// TestZoomKept: the zoom the player settles on is saved apart for battles and for the walk, and a battle map or a
// walk area opened afterwards opens at it instead of its own.
func TestZoomKept(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	demo := &Demo{g: g}

	t.Run("battle", func(t *testing.T) {
		require.NoError(t, demo.battle("B02"))
		assert.Equal(t, float64(battleZoom), g.zoom)
		g.rememberZoom(4)
		assert.Equal(t, 4, g.S.Settings.BattleZoom)
		assert.Zero(t, g.S.Settings.ExploreZoom)
		g.field = nil
		g.ensureField(g.S.View())
		assert.Equal(t, 4.0, g.zoom)
	})

	t.Run("walk", func(t *testing.T) {
		node := firstWalk(d, "")
		require.NotEmpty(t, node)
		require.NoError(t, demo.demoJump(node))
		g.held = nil
		g.layOutExplore(g.S.View())
		require.True(t, g.walkView())
		g.rememberZoom(1)
		assert.Equal(t, 1, g.S.Settings.ExploreZoom)
		assert.Equal(t, 4, g.S.Settings.BattleZoom)
		g.layOutExplore(g.S.View())
		assert.Equal(t, 1.0, g.zoom)
	})

	t.Run("saved", func(t *testing.T) {
		s2, err := session.New(d, g.S.Store.Dir) // the battle demo moved the game to a trial folder
		require.NoError(t, err)
		assert.Equal(t, 1, s2.Settings.ExploreZoom)
		assert.Equal(t, 4, s2.Settings.BattleZoom)
	})
}
