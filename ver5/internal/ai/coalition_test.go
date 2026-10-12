package ai

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func TestCoalitionAIReachesGuardAndTriggersDuels(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage == "B03" {
			d.Campaign.Start = n.ID
			break
		}
	}
	e := core.New(d, 17)
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].Faction == "enemy" && snap.Units[i].ID != "lubu" {
			snap.Units[i].HP = 0
		}
	}
	e, err = core.Restore(d, snap)
	require.NoError(t, err)
	_, err = e.Apply(core.Command{Kind: "wait", Actor: "liubei"})
	require.NoError(t, err)
	boss := e.Observe().UnitViews[0]
	for _, u := range e.Observe().UnitViews {
		if u.ID == "lubu" {
			boss = u
		}
	}
	for _, u := range e.Observe().UnitViews {
		if u.AI {
			assert.Greater(t, u.Y, boss.Y)
			assert.LessOrEqual(t, u.X, boss.X)
			assert.Equal(t, "NE", u.Facing)
			assert.Positive(t, d.Terrain[d.Classes[u.Class].Terrain][string(e.Observe().Map.Tiles[u.Y][u.X])].Cost)
		}
	}
	duels := map[string]bool{}
	firstDuelRound := -1
	moves := 0
	for round := 0; round < 15 && len(duels) < 3; round++ {
		_, err = e.Apply(core.Command{Kind: "end"})
		require.NoError(t, err)
		for step := 0; e.Observe().Turn != "ally" && step < 200; step++ {
			c, err := Next(e)
			require.NoError(t, err)
			if c.Kind == "move" {
				for _, u := range e.Observe().UnitViews {
					if u.ID == c.Actor {
						require.True(t, u.AI)
						moves++
					}
				}
			}
			r, err := e.Apply(c)
			require.NoError(t, err)
			for _, v := range r.Events {
				if v.Kind == "duel" {
					duels[v.DuelID] = true
					if firstDuelRound < 0 {
						firstDuelRound = round
					}
				}
			}
			require.Equal(t, "battle", e.Observe().Phase)
		}
		require.Equal(t, "ally", e.Observe().Turn)
	}
	assert.Equal(t, 0, firstDuelRound, "coalition must reach the guard before the player resumes advancing")
	assert.Positive(t, moves)
	assert.Len(t, duels, 3)
	for _, u := range e.Observe().UnitViews {
		if u.ID == "lubu" {
			assert.Equal(t, 25, u.X)
			assert.Equal(t, 4, u.Y)
		}
	}
}
