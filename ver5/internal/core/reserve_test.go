package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// reserveStage is the mini stage S1 with archer1 held in reserve until round 2, where a join event calls them to (2,6).
func reserveStage(s *content.Stage) {
	s.Max = 1
	s.Reserve = []string{"archer1"}
	s.Victory = []content.Goal{{If: []string{"round>=99"}, Text: "never"}}
	s.Events = []content.BattleEvent{{ID: "ambush", If: []string{"round>=2", "turnstart:ally"}, Do: []content.Action{
		{Kind: "join", Spawns: []content.Spawn{{Character: "archer1", X: 2, Y: 6}}}}}}
}

// reserveDeploying is the stage's deployment screen, before anyone is placed.
func reserveDeploying(t *testing.T) *Engine {
	t.Helper()
	d := data(t)
	reserveStage(&d.Stages[0])
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "S1" {
			d.Campaign.Start = n.ID
		}
	}
	e := New(d, 1)
	apply(t, e, Command{Kind: "start"})
	require.True(t, e.Deploying())
	return e
}

func TestReserveOfficersSortieWithoutACell(t *testing.T) {
	t.Run("they are in the sortie beyond Max and stand on no cell", func(t *testing.T) {
		e := reserveDeploying(t)
		assert.ElementsMatch(t, []string{"hero", "archer1"}, e.state.Deployment, "Max is 1 and both go")
		assertRejected(t, e, Command{Kind: "place", Actor: "archer1", X: 1, Y: 3})
		for _, c := range e.LegalActions("archer1").Commands {
			assert.NotEqual(t, "place", c.Kind)
		}
		plan, err := e.DeploymentPlan()
		require.NoError(t, err)
		for _, c := range plan {
			assert.NotEqual(t, "archer1", c.Actor, "the plan does not place them")
		}
	})
	t.Run("the battle starts with them off the map", func(t *testing.T) {
		e := reserveDeploying(t)
		finishDeployment(t, e)
		assert.Equal(t, "battle", e.Observe().Phase)
		assert.NotNil(t, e.unit("hero"))
		assert.Nil(t, e.unit("archer1"))
	})
	t.Run("the player cannot leave them out, and they do not take a place", func(t *testing.T) {
		e := reserveDeploying(t)
		apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero"}})
		assert.Contains(t, e.state.Deployment, "archer1")
		apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero", "archer1"}})
		assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"archer1"}})
	})
	t.Run("they keep gear changes made in the preparation", func(t *testing.T) {
		e := reserveDeploying(t)
		e.char("archer1").Level = 7
		e.char("archer1").Equipment = map[string]string{"armor": "jerkin"}
		finishDeployment(t, e)
		for e.unit("archer1") == nil && e.state.Round < 4 {
			applyOK(t, e, Command{Kind: "end"})
		}
		require.NotNil(t, e.unit("archer1"), "the join event calls them in")
		assert.Equal(t, 2, e.state.Round)
		assert.Equal(t, "ally", e.unit("archer1").Faction)
		assert.Equal(t, 7, e.char("archer1").Level)
		assert.Equal(t, map[string]string{"armor": "jerkin"}, e.char("archer1").Equipment)
		assert.False(t, e.unit("archer1").AI)
		assert.NotEmpty(t, e.LegalActions("archer1").Commands, "the player moves them")
	})
}

func TestReserveSurvivesSaveAndRetry(t *testing.T) {
	e := reserveDeploying(t)
	finishDeployment(t, e)
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Nil(t, r.unit("archer1"), "still waiting")
	assert.Contains(t, r.state.Deployment, "archer1")
	for r.state.Round < 2 {
		applyOK(t, r, Command{Kind: "end"})
	}
	require.NotNil(t, r.unit("archer1"))
	r2, err := Restore(r.Data(), r.Snapshot())
	require.NoError(t, err)
	assert.NotNil(t, r2.unit("archer1"), "on the map after the call")
	assert.True(t, r2.state.Executed["battle-S1-ambush"])
}

func TestReserveOfficerWhoIsGoneIsSkipped(t *testing.T) {
	e := reserveDeploying(t)
	e.char("archer1").Dead = true
	e.state.Deployment = []string{"hero"}
	finishDeployment(t, e)
	for e.state.Round < 2 {
		applyOK(t, e, Command{Kind: "end"})
	}
	assert.Nil(t, e.unit("archer1"), "the fallen do not come")
}
