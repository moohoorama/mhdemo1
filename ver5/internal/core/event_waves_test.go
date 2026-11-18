package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func groupCount(e *Engine, group string) int {
	n := 0
	for _, u := range e.Observe().Units {
		if u.Group == group && u.HP > 0 {
			n++
		}
	}
	return n
}

// nextRound ends the player's and the enemy's turns.
func nextRound(t *testing.T, e *Engine) {
	t.Helper()
	apply(t, e, Command{Kind: "end"})
	apply(t, e, Command{Kind: "end"})
}

func TestAnEventWithEveryComesBack(t *testing.T) {
	wave := []content.Spawn{{Character: "troop", X: 6, Y: 6, Side: "enemy", Group: "wave", Policy: "hold"}}
	e := battleWith(t, func(s *content.Stage) {
		s.Events = []content.BattleEvent{{ID: "wave", Every: 2, If: []string{"turnstart:ally", "!flag:stop"},
			Do: []content.Action{{Kind: "spawn", Spawns: wave}}}}
	})
	t.Run("it runs when its conditions hold and Every rounds have passed", func(t *testing.T) {
		assert.Equal(t, 1, groupCount(e, "wave"), "round 1")
		nextRound(t, e)
		assert.Equal(t, 1, groupCount(e, "wave"), "round 2: one round after")
		nextRound(t, e)
		assert.Equal(t, 2, groupCount(e, "wave"), "round 3")
		assert.Equal(t, []int{1, 3}, e.state.Fired["battle-S1-wave"])
	})
	t.Run("a save in the middle loads with every wave", func(t *testing.T) {
		r, err := Restore(e.Data(), e.Snapshot())
		require.NoError(t, err)
		nextRound(t, r)
		nextRound(t, r)
		assert.Equal(t, 3, groupCount(r, "wave"), "round 5")
		_, err = Restore(r.Data(), r.Snapshot())
		assert.NoError(t, err)
	})
	t.Run("it stops while its conditions fail", func(t *testing.T) {
		e.state.PendingFlags = map[string]int{"stop": 1}
		nextRound(t, e)
		nextRound(t, e)
		assert.Equal(t, 2, groupCount(e, "wave"))
	})
}

func TestABuffLaysAnAilment(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[0].Group = "front"
		s.Events = []content.BattleEvent{{ID: "panic", Do: []content.Action{
			{Kind: "buff", Targets: []string{"front"}, Stat: "confusion", Turns: 1},
			{Kind: "buff", Targets: []string{"front"}, Stat: "attack", Value: -30},
		}}}
	})
	u := e.unit("boss")
	assert.Equal(t, 1, u.Status["confusion"].Turns)
	assert.Equal(t, -30, u.Status["attack"].Value)
	assert.Equal(t, 99, u.Status["attack"].Turns, "no Turns: for the rest of the battle")
	assert.True(t, confused(u))
	_, err := Restore(e.Data(), e.Snapshot())
	assert.NoError(t, err)
}

func TestASpawnStartsWithAShareOfItsStrength(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) { s.Enemies[0].HPPct = 50 })
	u := e.unit("boss")
	assert.Equal(t, e.stats(u).MaxHP*50/100, u.HP)
	_, err := Restore(e.Data(), e.Snapshot())
	assert.NoError(t, err)
}

func TestSinceCountsRoundsFromAnEvent(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Events = []content.BattleEvent{flagEvent("ambush", "round>=2"), flagEvent("held", "since:ambush>=2")}
	})
	nextRound(t, e)
	require.True(t, fired(e, "ambush"), "round 2")
	nextRound(t, e)
	assert.False(t, fired(e, "held"), "round 3: one round since")
	nextRound(t, e)
	assert.True(t, fired(e, "held"), "round 4")
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.True(t, r.holdsIn([]string{"since:ambush>=2", "!since:ambush>=3"}, true), "kept across a save")
}
