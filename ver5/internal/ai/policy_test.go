package ai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// battle starts the mini stage S1 (heroes at (1,2) and (1,3), the boss at (9,2)) after tweak,
// with a victory that cannot come by itself so the battle runs on.
func battle(t *testing.T, tweak func(*content.Stage)) *core.Engine {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	st := &d.Stages[0]
	st.Victory = []content.Goal{{If: []string{"round>=99"}, Text: "never"}}
	tweak(st)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "S1" {
			d.Campaign.Start = n.ID
		}
	}
	e := core.New(d, 7)
	for _, c := range []core.Command{{Kind: "start"}, {Kind: "place", Actor: "hero", X: 1, Y: 2}, {Kind: "place", Actor: "archer1", X: 1, Y: 3}, {Kind: "deployment_finish"}} {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	return e
}

func unit(e *core.Engine, id string) *core.UnitView {
	for _, u := range e.Observe().UnitViews {
		if u.ID == id {
			return &u
		}
	}
	return nil
}

// enemyTurn plays the enemy phase with the AI and returns to the allies' turn; ally gets nothing to do.
func enemyTurn(t *testing.T, e *core.Engine) []core.Event {
	_, err := e.Apply(core.Command{Kind: "end"})
	require.NoError(t, err)
	var events []core.Event
	for i := 0; e.Observe().Turn == "enemy" && i < 100; i++ {
		c, err := Next(e)
		require.NoError(t, err)
		r, err := e.Apply(c)
		require.NoError(t, err)
		events = append(events, r.Events...)
	}
	require.Equal(t, "ally", e.Observe().Turn)
	return events
}

func TestHoldStandsStillAndStrikesWhatComesNear(t *testing.T) {
	e := battle(t, func(s *content.Stage) { s.Enemies[0].Policy = "hold" })
	for _, c := range e.LegalActions("boss").Commands {
		assert.NotEqual(t, "move", c.Kind, "a held unit has no move")
	}
	start := unit(e, "boss")
	for i := 0; i < 3; i++ {
		enemyTurn(t, e)
	}
	assert.Equal(t, [2]int{start.X, start.Y}, [2]int{unit(e, "boss").X, unit(e, "boss").Y})
}

func TestAssaultAdvancesFromTheFirstTurnWhereTheDefaultWaits(t *testing.T) {
	idle := battle(t, func(s *content.Stage) {})
	enemyTurn(t, idle)
	first := unit(idle, "boss")
	assert.Equal(t, [2]int{9, 2}, [2]int{first.X, first.Y}, "far from the heroes the default enemy waits")

	rush := battle(t, func(s *content.Stage) { s.Enemies[0].Policy = "assault" })
	enemyTurn(t, rush)
	moved := unit(rush, "boss")
	assert.Less(t, moved.X, 9, "an assaulting unit comes at once")
}

func TestMarchReachesItsGoalWithoutFightingAFarAwayHero(t *testing.T) {
	goal := content.Point{X: 10, Y: 7}
	e := battle(t, func(s *content.Stage) { s.Enemies[0].Policy, s.Enemies[0].Goal = "march", &goal })
	for i := 0; i < 4 && (unit(e, "boss").X != goal.X || unit(e, "boss").Y != goal.Y); i++ {
		enemyTurn(t, e)
	}
	b := unit(e, "boss")
	assert.Equal(t, [2]int{goal.X, goal.Y}, [2]int{b.X, b.Y})
	assert.Positive(t, b.HP)
	enemyTurn(t, e)
	assert.Equal(t, [2]int{goal.X, goal.Y}, [2]int{unit(e, "boss").X, unit(e, "boss").Y}, "at the goal it stays")
}

func TestMarchFightsWhatItMeetsOnTheWay(t *testing.T) {
	goal := content.Point{X: 0, Y: 6}
	e := battle(t, func(s *content.Stage) { s.Enemies[0].Policy, s.Enemies[0].Goal = "march", &goal })
	hurt := false
	for i := 0; i < 6 && !hurt; i++ {
		for _, ev := range enemyTurn(t, e) {
			hurt = hurt || ev.Kind == "damage" && ev.Actor == "boss"
		}
	}
	assert.True(t, hurt, "marching past the heroes it strikes them")
}

func TestFleeRunsToItsGoalNeverFightsAndLeavesTheField(t *testing.T) {
	goal := content.Point{X: 10, Y: 7}
	e := battle(t, func(s *content.Stage) {
		s.Enemies[0].X, s.Enemies[0].Y = 2, 3 // right beside the heroes
		s.Enemies[0].Policy, s.Enemies[0].Goal = "flee", &goal
	})
	left, struck := false, false
	for i := 0; i < 6 && !left; i++ {
		for _, ev := range enemyTurn(t, e) {
			struck = struck || ev.Kind == "damage" && ev.Actor == "boss"
			left = left || ev.Kind == "retreat" && ev.Actor == "boss"
		}
	}
	assert.True(t, left, "it reached the goal and left")
	assert.False(t, struck, "a fleeing unit does not fight")
	assert.Zero(t, unit(e, "boss").HP)
}

func TestMarchingIsDeterministic(t *testing.T) {
	run := func() string {
		goal := content.Point{X: 10, Y: 7}
		e := battle(t, func(s *content.Stage) { s.Enemies[0].Policy, s.Enemies[0].Goal = "march", &goal })
		for i := 0; i < 3; i++ {
			enemyTurn(t, e)
		}
		b, _ := json.Marshal(e.Snapshot())
		return string(b)
	}
	assert.Equal(t, run(), run())
}

func TestAnAIEventChangesTheNextTurnsBehaviour(t *testing.T) {
	goal := content.Point{X: 10, Y: 7}
	e := battle(t, func(s *content.Stage) {
		s.Enemies[0].Policy = "assault"
		s.Events = []content.BattleEvent{{ID: "stand", If: []string{"round>=2"}, Do: []content.Action{{Kind: "ai", Targets: []string{"boss"}, Policy: "march", Goal: &goal}}}}
	})
	enemyTurn(t, e) // round 1 ends: the boss assaults westward
	west := unit(e, "boss")
	require.Less(t, west.X, 9)
	for i := 0; i < 4; i++ {
		enemyTurn(t, e)
	}
	b := unit(e, "boss")
	assert.Equal(t, "march", b.Policy)
	assert.Equal(t, [2]int{10, 7}, [2]int{b.X, b.Y}, "from round 2 it marches to the corner instead")
}
