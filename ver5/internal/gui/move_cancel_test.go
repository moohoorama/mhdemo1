package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// moveGame is a started battle with the first ally ordered to a cell several steps away.
func moveGame(t *testing.T) (*Game, core.UnitView, core.Command) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage != "" {
			d.Campaign.Start = n.ID
			break
		}
	}
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	s.Engine, s.Screen = core.New(d, 17), "playing"
	_, err = s.Engine.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := s.Engine.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = s.Engine.Apply(c)
		require.NoError(t, err)
	}
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	t.Cleanup(g.Close)
	g.ensureField(s.Engine.Observe())
	g.modals, g.speed = nil, 1
	for range 600 {
		if !g.busy() {
			break
		}
		g.advance(1.0 / 60)
	}
	require.False(t, g.busy())
	for _, u := range s.Engine.Observe().UnitViews {
		if u.Faction != "ally" || u.AI || u.Moved || u.Done {
			continue
		}
		for _, c := range s.Engine.LegalActions(u.ID).Commands {
			if c.Kind == "move" && abs(c.X-u.X)+abs(c.Y-u.Y) >= 3 {
				return g, u, c
			}
		}
	}
	t.Fatal("no ally can walk three cells")
	return nil, core.UnitView{}, core.Command{}
}

func TestCancelledMoveStopsWalkingInPlace(t *testing.T) {
	for name, frames := range map[string]int{"mid-walk": 20, "after arriving": 600} {
		t.Run(name, func(t *testing.T) {
			g, u, move := moveGame(t)
			g.begin(u)
			g.tentative(move.X, move.Y)
			for range frames {
				g.advance(1.0 / 60)
			}
			require.Equal(t, "menu", g.ord.stage)
			if name == "mid-walk" {
				require.True(t, g.player.Busy(), "the walk is still playing")
				require.Equal(t, "walk", g.vis[u.ID].anim)
			}

			assert.True(t, g.cancel())
			for range 120 {
				g.advance(1.0 / 60)
			}

			v := g.vis[u.ID]
			assert.Equal(t, float64(u.X), v.u, "back on its own cell")
			assert.Equal(t, float64(u.Y), v.v)
			assert.Nil(t, v.walk)
			assert.NotEqual(t, "walk", v.anim, "no longer walking")
		})
	}
}
