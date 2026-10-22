package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func sweepArena(t *testing.T) (*Engine, *Unit, *Unit, *Unit) {
	t.Helper()
	e, a, b := arena(t)
	e.data.Stages[e.state.Stage].Duels = nil
	e.data.Traits["sweep_test"] = content.Trait{Name: "Sweep", Effects: map[string]float64{content.FxRetreatSweep: 1}}
	with(e, a.ID, "sweep_test", "twice")
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	c := e.unit("troop#1")
	c.X, c.Y, c.HP = 5, 5, 10000
	b.HP = 1
	e.events = nil
	return e, a, b, c
}

func TestRetreatSweepUsesNormalExchange(t *testing.T) {
	t.Run("retreat then double attack and counter", func(t *testing.T) {
		e, a, b, c := sweepArena(t)
		c.Status["counter"] = Status{Turns: 1}
		with(e, c.ID, "twice")
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		retreat, start := -1, -1
		attacks, counters := 0, 0
		for i, ev := range e.events {
			if ev.Kind == "retreat" && ev.Actor == b.ID {
				retreat = i
			}
			if ev.Kind == "followup-start" && ev.Target == c.ID {
				start = i
			}
			if ev.Kind == "damage" && ev.Actor == a.ID && ev.Target == c.ID {
				attacks++
			}
			if ev.Kind == "damage" && ev.Actor == c.ID && ev.Target == a.ID {
				counters++
			}
		}
		require.GreaterOrEqual(t, retreat, 0)
		assert.Greater(t, start, retreat)
		assert.Equal(t, 2, attacks)
		assert.Equal(t, 2, counters)
	})
	t.Run("a retreating sweep target sweeps on", func(t *testing.T) {
		e, a, b, c := sweepArena(t)
		c.HP = 1
		near := e.unit("troop#2")
		near.X, near.Y, near.HP = 4, 6, 10000
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		assert.Zero(t, c.HP)
		assert.Less(t, near.HP, 10000)
		count := 0
		for _, ev := range e.events {
			if ev.Kind == "retreat-sweep" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})
	t.Run("one foe per retreat, the weakest", func(t *testing.T) {
		e, a, b, c := sweepArena(t)
		near := e.unit("troop#2")
		near.X, near.Y, near.HP = 4, 6, 9000
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		assert.Less(t, near.HP, 9000)
		assert.Equal(t, 10000, c.HP)
	})
	t.Run("counter can trigger sweep", func(t *testing.T) {
		e, a, b, c := sweepArena(t)
		a.HP, b.HP = 10000, 1
		a.Status["counter"] = Status{Turns: 1}
		require.NoError(t, e.useSkill(b, Command{Kind: "attack", Actor: b.ID, Target: a.ID}))
		assert.Zero(t, b.HP)
		assert.Less(t, c.HP, 10000)
	})
	t.Run("stop after attacker retreats", func(t *testing.T) {
		e, a, b, c := sweepArena(t)
		a.HP = 1
		c.Status["counter"] = Status{Turns: 1}
		far := e.unit("troop#2")
		far.X, far.Y, far.HP = 4, 6, 10000
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		assert.Zero(t, a.HP)
		assert.Equal(t, 10000, far.HP)
	})
}

func TestSplashSharesOneDoubleRoll(t *testing.T) {
	setup := func(t *testing.T, traits ...string) (*Engine, *Unit, []string) {
		e, a, b := arena(t)
		t1, t2 := e.unit("troop#1"), e.unit("troop#2")
		t1.HP, t2.HP, b.HP = 10000, 10000, 10000
		t1.X, t1.Y = 6, 5
		t2.X, t2.Y = 6, 7
		with(e, "hero", traits...)
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		e.events = nil
		return e, a, []string{b.ID, t1.ID, t2.ID}
	}
	t.Run("every victim takes the first swing, then every victim the second", func(t *testing.T) {
		e, a, all := setup(t, "cross_t", "twice")
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: "boss"}))
		var first, second []string
		doubled := false
		for _, ev := range e.events {
			if ev.Kind == "double" && ev.Actor == a.ID {
				doubled = true
			}
			if ev.Kind == "damage" && ev.Actor == a.ID {
				if doubled {
					second = append(second, ev.Target)
				} else {
					first = append(first, ev.Target)
				}
			}
		}
		assert.ElementsMatch(t, all, first)
		assert.ElementsMatch(t, all, second)
	})
	t.Run("no victim strikes twice when the aimed one does not", func(t *testing.T) {
		e, a, _ := setup(t, "cross_t")
		e.data.Rules.Double = content.Chance{}
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: "boss"}))
		for _, ev := range e.events {
			assert.False(t, ev.Kind == "double" && ev.Actor == a.ID)
		}
	})
}

func TestEachSkillAttacksTargetsOneByOne(t *testing.T) {
	e, a, b := arena(t)
	t1 := e.unit("troop#1")
	t1.HP, b.HP, a.HP = 10000, 10000, 10000
	t1.X, t1.Y = 5, 5
	b.Status["counter"] = Status{Turns: 1}
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	s := e.basic()
	s.Shape, s.Each = content.ShapeAll, true
	e.events = nil
	e.physical(a, b, s, Command{Kind: "skill", Actor: a.ID, Target: b.ID})
	var order []string
	for _, ev := range e.events {
		switch {
		case ev.Kind == "damage" && ev.Actor == a.ID:
			order = append(order, "hit "+ev.Target)
		case ev.Kind == "damage" && ev.Actor == b.ID:
			order = append(order, "counter")
		case ev.Kind == "next-strike":
			order = append(order, "next "+ev.Target)
		}
	}
	order = slices.Compact(order)
	assert.Equal(t, []string{"hit boss", "counter", "next troop#1", "hit troop#1"}, order)
}

func TestTurnLimitBoundary(t *testing.T) {
	for _, tc := range []struct {
		name         string
		limit, round int
		turn         string
		defeat       bool
	}{
		{"last ally turn", 3, 3, "ally", false},
		{"last enemy turn", 3, 3, "enemy", false},
		{"next round", 3, 4, "ally", true},
		{"unlimited", 0, 100, "ally", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := battle(t)
			e.data.Stages[e.state.Stage].TurnLimit = tc.limit
			e.state.Round, e.state.Turn = tc.round, tc.turn
			e.settle()
			assert.Equal(t, tc.defeat, e.state.Result == "defeat")
		})
	}
	t.Run("enemy phase ends at deadline", func(t *testing.T) {
		e := battle(t)
		e.data.Stages[e.state.Stage].TurnLimit = 3
		e.state.Round, e.state.Turn = 3, "enemy"
		r := apply(t, e, Command{Kind: "end"})
		assert.Equal(t, "defeat", e.state.Result)
		assert.Contains(t, r.Events, Event{Kind: "turn-limit", Amount: 3})
	})
	t.Run("victory on final turn", func(t *testing.T) {
		e, _, b := arena(t)
		e.data.Stages[e.state.Stage].TurnLimit = 3
		e.state.Round, b.HP = 3, 0
		e.settle()
		assert.Equal(t, "victory", e.state.Result)
	})
}
