package ai

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func TestCoalitionAIReachesGuardAndTriggersDuels(t *testing.T) { coalitionDuels(t, nil) }

// A boss who can shoot from range (the white horse) still waits for his duel partners: his duel-first setting
// keeps him from picking them off at a distance before they reach him.
func TestDuelFirstBossDoesNotShootHisPartners(t *testing.T) {
	coalitionDuels(t, func(d *content.Data) {
		lubu := d.Characters["lubu"]
		lubu.Traits = append(lubu.Traits, "white_horse")
		d.Characters["lubu"] = lubu
	})
}

func coalitionDuels(t *testing.T, tweak func(*content.Data)) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	if tweak != nil {
		tweak(d)
	}
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B03" {
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
			assert.Greater(t, u.X, boss.X, "the coalition comes in from the east")
			assert.Equal(t, "NW", u.Facing)
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
						require.True(t, u.AI || u.ID == "lubu", "only the coalition and the boss move")
						if u.AI {
							moves++
						}
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
			if e.Observe().Phase != "battle" {
				break // the duels decided the battle
			}
		}
		if e.Observe().Phase != "battle" {
			break
		}
		require.Equal(t, "ally", e.Observe().Turn)
	}
	assert.GreaterOrEqual(t, firstDuelRound, 0, "the coalition and the boss meet within a few rounds")
	assert.Positive(t, moves)
	assert.NotEmpty(t, duels)
}
