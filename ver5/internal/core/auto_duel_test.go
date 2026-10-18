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

func TestAMoveNextToTheDuelPartnerStartsTheDuel(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2 // the boss stands where the hero can reach
	})
	assert.Equal(t, "D", e.DuelAt("hero", 3, 2), "the cell next to the boss")
	assert.Equal(t, "D", e.DuelAt("hero", 4, 1))
	assert.Empty(t, e.DuelAt("hero", 2, 2), "two cells away")
	assert.Empty(t, e.DuelAt("archer1", 3, 2), "the archer is no one's partner")
	r := applyOK(t, e, Command{Kind: "move", Actor: "hero", X: 3, Y: 2})
	duels := duelsIn(r)
	require.Len(t, duels, 1)
	assert.Equal(t, "hero", duels[0].First)
	assert.Equal(t, "hero", duels[0].Actor)
	assert.Equal(t, "boss", duels[0].Target)
	assert.Zero(t, e.unit("boss").HP, "the fixed result: the boss retreats")
	hero := e.unit("hero")
	assert.True(t, hero.Moved && hero.Acted && hero.Done, "the duel ends the mover's action")
	assert.True(t, e.state.Executed["duel-D"])
	assert.Empty(t, e.DuelAt("hero", 3, 2), "a duel is held once")
}

func TestADuelHappensOnceAndSurvivesSaveAndRestore(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		neverWon(s)
		s.Enemies[0].X, s.Enemies[0].Y = 4, 2
	})
	applyOK(t, e, Command{Kind: "move", Actor: "hero", X: 3, Y: 2})
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
	r := applyOK(t, e, Command{Kind: "move", Actor: "boss", X: 2, Y: 3})
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
