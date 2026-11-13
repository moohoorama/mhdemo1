package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

var heroVsBoss = content.Duel{ID: "D", Ally: "hero", Enemy: "boss", Outcome: "victory", AllyResult: "stay", EnemyResult: "retreat", Text: "d"}

func neverWon(s *content.Stage) {
	s.Victory = []content.Goal{{If: []string{"round>=99"}, Text: "never"}}
	s.Duels = []content.Duel{heroVsBoss}
}

func duelsIn(r Result) []Event {
	var out []Event
	for _, ev := range r.Events {
		if ev.Kind == "duel" {
			out = append(out, ev)
		}
	}
	return out
}

// moveAndWait moves the unit and ends its turn there, returning what the wait set off.
func moveAndWait(t *testing.T, e *Engine, id string, x, y int) Result {
	t.Helper()
	assert.Empty(t, duelsIn(applyOK(t, e, Command{Kind: "move", Actor: id, X: x, Y: y})), "a move alone is no meeting yet")
	return applyOK(t, e, Command{Kind: "wait", Actor: id})
}

func TestEndingTheTurnNextToTheDuelPartnerStartsTheDuel(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2 // the boss stands where the hero can reach
	})
	assert.Equal(t, "D", e.DuelAt("hero", 3, 2), "the cell next to the boss")
	assert.Equal(t, "D", e.DuelAt("hero", 4, 1))
	assert.Empty(t, e.DuelAt("hero", 2, 2), "two cells away")
	assert.Empty(t, e.DuelAt("archer1", 3, 2), "the archer is no one's partner")
	r := moveAndWait(t, e, "hero", 3, 2)
	duels := duelsIn(r)
	require.Len(t, duels, 1)
	assert.Equal(t, "hero", duels[0].First)
	assert.Equal(t, "hero", duels[0].Actor)
	assert.Equal(t, "boss", duels[0].Target)
	assert.Zero(t, e.unit("boss").HP, "the fixed result: the boss retreats")
	assert.True(t, e.state.Executed["duel-D"])
	assert.Empty(t, e.DuelAt("hero", 3, 2), "a duel is held once")
}

func TestADuelWaitsForItsConditions(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Duels[0].If = []string{"round>=2"}
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2
	})
	assert.Empty(t, e.DuelAt("hero", 3, 2), "the duel is not open in the first round")
	assert.Empty(t, duelsIn(moveAndWait(t, e, "hero", 3, 2)), "standing next to the boss sets nothing off yet")
	assert.False(t, e.state.Executed["duel-D"])
	e.state.Round = 2
	assert.Equal(t, "D", e.DuelAt("hero", 3, 2), "the duel opens once its condition holds")
}

func TestADuelHappensOnceAndSurvivesSaveAndRestore(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2
	})
	moveAndWait(t, e, "hero", 3, 2)
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Empty(t, duelsIn(applyOK(t, r, Command{Kind: "end"})))
}

func TestTheEnemyWalkingUpStartsTheDuelToo(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 3, 3
	})
	assert.False(t, e.state.Executed["duel-D"], "(3,3) is two cells from the hero at (1,3)")
	applyOK(t, e, Command{Kind: "end"})
	r := moveAndWait(t, e, "boss", 2, 3)
	duels := duelsIn(r)
	require.Len(t, duels, 1)
	assert.Equal(t, "boss", duels[0].First)
	assert.Equal(t, "hero", duels[0].Actor, "the event still names the ally first")
}

func TestADuelStartsWithTheBattleWhenTheyBeginNextToEachOther(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 1, 4 // beside the hero's deployment cell (1,3)
	})
	assert.True(t, e.state.Executed["duel-D"])
	assert.Zero(t, e.unit("boss").HP)
	hero := e.unit("hero")
	assert.False(t, hero.Done, "nobody moved to cause it, so nobody's action is spent")
}

func TestARangedAttackOnTheDuelPartnerIsNoDuel(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Duels[0].Ally = "archer1"
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2
	})
	for _, c := range e.LegalActions("archer1").Commands {
		if c.Kind == "attack" && c.Target == "boss" {
			r := applyOK(t, e, c)
			assert.Empty(t, duelsIn(r), "a duel needs the two to meet")
			assert.False(t, e.state.Executed["duel-D"])
			return
		}
	}
	t.Skip("the archer cannot reach the boss from here")
}

func TestASupportMeetingTheEnemyStartsTheDuel(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Duels[0].Supports = []string{"archer1", "nobody"}
		s.Enemies[0].X, s.Enemies[0].Y = 4, 3
	})
	assert.Equal(t, "D", e.DuelAt("archer1", 3, 3), "the archer joins the hero's duel")
	r := moveAndWait(t, e, "archer1", 3, 3)
	duels := duelsIn(r)
	require.Len(t, duels, 1)
	assert.Equal(t, "archer1", duels[0].First, "the staging opens with the support who met the boss")
	assert.Equal(t, "hero", duels[0].Actor, "the duel is still the hero's")
}

func TestASupportFightsTheDuelWhenItsAllyHasLeft(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Duels[0].Ally = "archer1"
		s.Duels[0].Supports = []string{"hero", "nobody"}
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2
	})
	e.unit("archer1").HP = 0 // the hero is the lord, so the archer is the one who leaves
	r := moveAndWait(t, e, "hero", 3, 2)
	duels := duelsIn(r)
	require.Len(t, duels, 1)
	assert.Equal(t, "hero", duels[0].Actor, "the support stands in for the archer")
}

func TestAnAttackAfterTheMoveIsCarriedOutBeforeTheDuel(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2
	})
	e.unit("boss").HP = 9999 // outlives one blow, so the duel still has someone to meet
	applyOK(t, e, Command{Kind: "move", Actor: "hero", X: 3, Y: 2})
	for _, c := range e.LegalActions("hero").Commands {
		if c.Kind == "attack" && c.Target == "boss" {
			r := applyOK(t, e, c)
			first, duel := -1, -1
			for i, ev := range r.Events {
				if ev.Kind == "duel" {
					duel = i
				} else if first < 0 && ev.Target == "boss" {
					first = i
				}
			}
			require.GreaterOrEqual(t, duel, 0, "the finished attack sets off the duel")
			require.GreaterOrEqual(t, first, 0)
			assert.Less(t, first, duel, "the blow lands before the duel")
			return
		}
	}
	t.Fatal("the hero has no attack on the boss from (3,2)")
}
