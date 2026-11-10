package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// TestReplayCamera checks the replay camera on B02: the ally turn's banner glides to the lord when
// the camera is off it, a cutscene step off screen glides there first, and a walker leaving the
// middle of the screen is followed.
func TestReplayCamera(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	require.NoError(t, (&Demo{g: g}).battle("B02"))
	g.setZoomImmediate(4)
	var lord, far core.UnitView
	for _, u := range g.S.View().UnitViews {
		if u.Faction == "ally" && g.S.Data.Characters[u.Def].Lord {
			lord = u
		}
	}
	require.NotEmpty(t, lord.ID)
	farD := -1
	for _, u := range g.S.View().UnitViews {
		if dist := abs(u.X-lord.X) + abs(u.Y-lord.Y); u.Faction == "enemy" && dist > farD {
			far, farD = u, dist
		}
	}
	require.NotEmpty(t, far.ID)
	g.camX, g.camY = g.field.Center(float64(far.X), float64(far.Y))
	lordCell := []content.Point{{X: lord.X, Y: lord.Y}}
	require.False(t, g.onScreen(lordCell))

	t.Run("ally turn glides to the lord", func(t *testing.T) {
		g.showTurn(2, "ally", "")
		require.NotNil(t, g.pan)
		for g.pan != nil {
			g.advancePan(1.0 / 60)
		}
		assert.True(t, g.onScreen(lordCell))
		g.turnBanner = nil
	})

	t.Run("cutscene glides to an off-screen face", func(t *testing.T) {
		entries, total := g.cutsceneTimeline([][]core.CutStep{{{Kind: "face", Unit: far.ID, Facing: "NE"}}})
		assert.Len(t, entries, 2, "a glide, then the turn")
		assert.InDelta(t, panTime, total, 1e-9)
		entries, total = g.cutsceneTimeline([][]core.CutStep{{{Kind: "face", Unit: lord.ID, Facing: "NE"}}})
		assert.Len(t, entries, 1, "no glide when on screen")
		assert.Zero(t, total)
	})

	t.Run("a walker leaving the middle is followed", func(t *testing.T) {
		v := g.vis[far.ID]
		require.NotNil(t, v)
		require.False(t, g.nearMiddle(v))
		g.player.waiting = []int{0} // a replay is playing
		defer func() { g.player.waiting = nil }()
		g.walkTo(v, v.u+1, v.v)
		x0, y0 := g.camX, g.camY
		g.followWalk(1.0 / 60)
		assert.Equal(t, far.ID, g.follow)
		assert.NotEqual(t, [2]float64{x0, y0}, [2]float64{g.camX, g.camY})
	})
}
