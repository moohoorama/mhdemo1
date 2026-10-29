package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func goal(text string, cond ...string) content.Goal { return content.Goal{If: cond, Text: text} }

// with sets the stage's goals and restores the engine from a snapshot edited by tweak.
func settleWith(t *testing.T, stage func(*content.Stage), tweak func(*State)) *Engine {
	e := battleWith(t, stage)
	snap := e.Snapshot()
	if tweak != nil {
		tweak(&snap)
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	applyOK(t, e, Command{Kind: "end"})
	return e
}

func killUnits(s *State, ids ...string) {
	for i := range s.Units {
		for _, id := range ids {
			if s.Units[i].ID == id {
				s.Units[i].HP = 0
			}
		}
	}
}

func TestEachKindOfVictoryGoal(t *testing.T) {
	area := content.Area{ID: 1, Rect: content.MapRect{X: 6, Y: 4, W: 1, H: 2}, Name: "stairs"}
	move := func(id string) func(*State) {
		return func(s *State) { s.Units[unitIndex(s, id)].X, s.Units[unitIndex(s, id)].Y = 6, 5 }
	}
	cases := map[string]struct {
		win   content.Goal
		tweak func(*State)
	}{
		"defeat the boss": {goal("boss", "defeated:boss"), func(s *State) { killUnits(s, "boss") }},
		"rout":            {goal("all", "alldefeated:enemy"), func(s *State) { killUnits(s, "boss", "troop#1", "troop#2") }},
		"boss below 40%":  {goal("weak", "any:[hp:boss<41, defeated:boss]"), func(s *State) { s.Units[unitIndex(s, "boss")].HP = 5 }},
		"reach with unit": {goal("reach", "at:hero:1"), move("hero")},
		"reach all": {goal("all there", "allat:ally:1"), func(s *State) {
			move("hero")(s)
			s.Units[unitIndex(s, "archer1")].X, s.Units[unitIndex(s, "archer1")].Y = 6, 4
		}},
		"survive a round": {goal("hold", "round>=2", "alive:hero"), func(s *State) { s.Round = 2 }},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := settleWith(t, func(s *content.Stage) {
				s.Areas = []content.Area{area}
				s.Victory = []content.Goal{c.win}
			}, func(s *State) {
				if c.tweak != nil {
					c.tweak(s)
				}
			})
			assert.Equal(t, "victory", e.Observe().Result, name)
		})
	}
}

func TestAllAtNeedsEveryAllyThere(t *testing.T) {
	e := settleWith(t, func(s *content.Stage) {
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 6, Y: 4, W: 1, H: 2}, Name: "stairs"}}
		s.Victory = []content.Goal{goal("all there", "allat:ally:1")}
	}, func(s *State) { s.Units[unitIndex(s, "hero")].X, s.Units[unitIndex(s, "hero")].Y = 6, 5 })
	assert.Equal(t, "battle", e.Observe().Phase, "the archer is still behind")
}

func unitIndex(s *State, id string) int {
	for i := range s.Units {
		if s.Units[i].ID == id {
			return i
		}
	}
	return -1
}

func TestDefeatGoalAndBuiltInDefeat(t *testing.T) {
	area := content.Area{ID: 1, Rect: content.MapRect{X: 0, Y: 0, W: 3, H: 6}, Name: "our camp"}
	e := settleWith(t, func(s *content.Stage) {
		s.Areas = []content.Area{area}
		s.Defeat = []content.Goal{goal("camp overrun", "at:enemy:1")}
	}, func(s *State) { s.Units[unitIndex(s, "boss")].X, s.Units[unitIndex(s, "boss")].Y = 2, 5 })
	assert.Equal(t, "defeat", e.Observe().Result)

	e = settleWith(t, func(s *content.Stage) {}, func(s *State) { killUnits(s, "hero") })
	assert.Equal(t, "defeat", e.Observe().Result, "the lord's fall always loses")
}

func TestDefeatBeatsVictoryInTheSameCommand(t *testing.T) {
	e := settleWith(t, func(s *content.Stage) {}, func(s *State) { killUnits(s, "boss", "hero") })
	assert.Equal(t, "defeat", e.Observe().Result)
	assert.NotContains(t, e.state.Executed, "won-S1")
}

func TestObjectiveRewardsFlagsAndRetry(t *testing.T) {
	sub := content.Objective{ID: "spared", Text: "spare the troops", If: []string{"alive:troops"}, Flag: "flag-spared", Reward: map[string]int{"tonic": 2}}
	setup := func(s *content.Stage) {
		s.Enemies[1].Group, s.Enemies[2].Group = "troops", "troops"
		s.Sub = []content.Objective{sub}
		s.Events = []content.BattleEvent{flagEvent("noise")}
	}
	tonic := battleWith(t, func(*content.Stage) {}).state.Inventory["tonic"]
	e := settleWith(t, setup, func(s *State) { killUnits(s, "boss") })
	require.Equal(t, "victory", e.Observe().Result)
	assert.Equal(t, 1, e.state.Flags["flag-spared"])
	assert.Equal(t, 1, e.state.Flags["noise"], "the battle's pending flags join the story's at the victory")
	assert.Empty(t, e.state.PendingFlags)
	assert.Equal(t, tonic+1+2, e.state.Inventory["tonic"], "stage reward and the objective's")
	assert.Equal(t, []bool{true}, e.SubDone())

	e = settleWith(t, setup, func(s *State) { killUnits(s, "boss", "troop#1", "troop#2") })
	require.Equal(t, "victory", e.Observe().Result)
	assert.Zero(t, e.state.Flags["flag-spared"])
	assert.Equal(t, tonic+1, e.state.Inventory["tonic"])
	assert.Equal(t, []bool{false}, e.SubDone())

	e = settleWith(t, setup, func(s *State) { killUnits(s, "hero") })
	require.Equal(t, "defeat", e.Observe().Result)
	require.NoError(t, e.Retry())
	assert.Empty(t, e.state.Flags, "a lost battle sets no story flags")
	assert.Empty(t, e.state.PendingFlags)
}

func TestSubDoneShowsTheCurrentStateDuringTheBattle(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[0].Group = "lords"
		s.Sub = []content.Objective{{ID: "a", Text: "a", If: []string{"defeated:lords"}}}
	})
	assert.Equal(t, []bool{false}, e.SubDone())
	snap := e.Snapshot()
	killUnits(&snap, "boss")
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	assert.Equal(t, []bool{true}, e.SubDone())
}

func TestRequiredOfficersAndTheSortieLimit(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Stages[0].Required = []string{"archer1"}
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "S1" {
			d.Campaign.Start = n.ID
		}
	}
	d.Campaign.Deployment = []string{"hero"}
	e := New(d, 1)
	assert.Equal(t, []string{"archer1", "hero"}, e.Observe().Deployment, "the required officer joins the sortie by itself")
	assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"hero"}})
	_, err = e.Apply(Command{Kind: "deploy", Deployment: []string{"hero", "archer1"}})
	assert.NoError(t, err)
	assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"hero", "archer1", "sage"}}) // Max is 2

	d.Stages[0].Required = []string{"archer1"}
	d.Campaign.Deployment = []string{"hero", "archer1"}
	e = New(d, 1)
	snap := e.Snapshot()
	for i := range snap.Characters {
		if snap.Characters[i].ID == "archer1" {
			snap.Characters[i].Dead = true
		}
	}
	snap.Deployment = []string{"hero"}
	r, err := Restore(d, snap)
	require.NoError(t, err)
	_, err = r.Apply(Command{Kind: "deploy", Deployment: []string{"hero"}})
	assert.NoError(t, err, "a dead required officer is not demanded")
}

func TestAnEventPutsABonusObjectiveInPlace(t *testing.T) {
	sub := content.Objective{ID: "spared", Text: "two troops stand", If: []string{"alive:troops>=2"}, Flag: "flag-spared"}
	setup := func(s *content.Stage) {
		s.Enemies[1].Group, s.Enemies[2].Group = "troops", "troops"
		s.Events = []content.BattleEvent{{ID: "late", If: []string{"defeated:boss"}, Do: []content.Action{{Kind: "goals", Sub: []content.Objective{sub}}}}}
	}
	assert.Empty(t, battleWith(t, setup).SubDone(), "no bonus objective before the event")

	e := settleWith(t, setup, func(s *State) { killUnits(s, "boss") })
	require.Equal(t, "victory", e.Observe().Result)
	assert.Equal(t, []content.Objective{sub}, e.Observe().Map.Sub, "the briefing shows the event's objective")
	assert.Equal(t, 1, e.state.Flags["flag-spared"])

	e = settleWith(t, setup, func(s *State) { killUnits(s, "boss", "troop#1") })
	require.Equal(t, "victory", e.Observe().Result)
	assert.Zero(t, e.state.Flags["flag-spared"], "one troop is fewer than two")
}
