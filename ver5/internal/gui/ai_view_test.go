package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestNearestZoom(t *testing.T) {
	for _, tc := range []struct{ in, want float64 }{
		{.7, 1}, {1.3, 1}, {1.5, 2}, {2.7, 2}, {3, 4}, {6, 4},
	} {
		assert.Equal(t, tc.want, nearestZoom(tc.in), "zoom %v", tc.in)
	}
}

// TestAITurnReplay plays B02's first allied and enemy turns: the camera is locked while they
// play, the log grows with the replay instead of ahead of it, and every turn plays to the end.
func TestAITurnReplay(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	require.NoError(t, (&Demo{g: g}).battle("B02"))
	assert.Equal(t, float64(battleZoom), g.zoom)
	assert.False(t, g.cameraLocked())
	g.command(core.Command{Kind: "end"})
	for _, setting := range []string{"together", "one"} {
		g.S.Settings.Replay = setting
		for f := 0; f < 20000 && g.S.View().Phase == "battle"; f++ {
			if !g.busy() {
				if g.S.View().Turn == "ally" {
					break
				}
				g.resolvePhase()
				require.True(t, g.player.ai)
				assert.Empty(t, g.logPending, "nothing is logged before it plays")
			}
			if g.player.Busy() {
				assert.True(t, g.cameraLocked())
			}
			g.advance(1.0 / 60)
			g.publishBattleLog()
		}
		require.Equal(t, "ally", g.S.View().Turn, setting)
		assert.NotEmpty(t, g.history)
		g.command(core.Command{Kind: "end"})
	}
}
