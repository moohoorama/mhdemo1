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

func TestRetreatSweepCoefficient(t *testing.T) {
	loss := func(t *testing.T, k float64) int {
		e, a, b, c := sweepArena(t)
		e.data.Traits["sweep_test"] = content.Trait{Name: "Sweep", Effects: map[string]float64{content.FxRetreatSweep: k}}
		e.data.Rules.CritMultiplier = 1
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		return 10000 - c.HP
	}
	t.Run("half coefficient halves the follow-up", func(t *testing.T) {
		full, half := loss(t, 1), loss(t, .5)
		require.Positive(t, full)
		assert.InDelta(t, float64(full)/2, float64(half), float64(full)*.1)
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

func TestAreaSkillTargetsSplashLikeBasicAttacks(t *testing.T) {
	for _, each := range []bool{true, false} {
		t.Run(map[bool]string{true: "each", false: "at once"}[each], func(t *testing.T) {
			e, a, b := arena(t)
			t1, t2 := e.unit("troop#1"), e.unit("troop#2")
			t1.HP, t2.HP, b.HP = 10000, 10000, 10000
			t1.X, t1.Y = 5, 5 // beside the hero, struck itself
			t2.X, t2.Y = 7, 6 // out of reach, behind the boss
			with(e, "hero", "cross_t")
			s := e.basic()
			s.Shape, s.Each = content.ShapeAll, each
			hit, share, _ := e.victims(a, b, s)
			assert.Len(t, hit, 3)
			assert.Equal(t, 1.0, share[t1.ID])
			assert.Equal(t, .75, share[t2.ID])

			e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
			e.physical(a, b, s, Command{Kind: "skill", Actor: a.ID, Target: b.ID})
			assert.Less(t, t2.HP, 10000)
		})
	}
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

func TestDerivedAttacksKeepStrikeTraits(t *testing.T) {
	// the boss's cross reaches archer1 above the hero; troop#1 stands right of the boss
	setup := func(t *testing.T) (*Engine, *Unit, *Unit) {
		e, a, b := arena(t)
		w, t1 := e.unit("archer1"), e.unit("troop#1")
		w.X, w.Y, t1.X, t1.Y = 5, 5, 7, 6
		a.HP, b.HP, w.HP, t1.HP = 10000, 10000, 10000, 10000
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		return e, a, b
	}
	struck := func(e *Engine, by string) (targets []string, doubled bool) {
		for _, ev := range e.events {
			if ev.Kind == "damage" && ev.Actor == by {
				targets = append(targets, ev.Target)
			}
			doubled = doubled || ev.Kind == "double" && ev.Actor == by
		}
		return
	}
	t.Run("counter", func(t *testing.T) {
		e, a, b := setup(t)
		with(e, "boss", "cross_t", "twice")
		e.events = nil
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		got, doubled := struck(e, b.ID)
		assert.Contains(t, got, "archer1")
		assert.True(t, doubled)
	})
	t.Run("riposte", func(t *testing.T) {
		e, a, b := setup(t)
		with(e, "boss", "riposte_t", "cross_t", "twice")
		b.Status["guard"] = Status{Turns: 1}
		e.events = nil
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		got, doubled := struck(e, b.ID)
		assert.Contains(t, got, "archer1")
		assert.True(t, doubled)
	})
	t.Run("assist", func(t *testing.T) {
		e, a, b := setup(t)
		w := e.unit("archer1")
		w.X, w.Y = 6, 5
		e.char("archer1").Class = "rider"
		e.char("archer1").Equipment = map[string]string{}
		with(e, "archer1", "assist_t", "cross_t", "twice")
		e.events = nil
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
		got, doubled := struck(e, w.ID)
		assert.Contains(t, got, "troop#1")
		assert.True(t, doubled)
	})
	t.Run("no assist on a counter", func(t *testing.T) {
		e, _, b := setup(t)
		w := e.unit("archer1")
		e.char("archer1").Class = "rider"
		e.char("archer1").Equipment = map[string]string{}
		with(e, "archer1", "assist_t")
		w.X, w.Y = 6, 5
		e.state.Turn = "enemy"
		e.events = nil
		require.NoError(t, e.useSkill(b, Command{Kind: "attack", Actor: b.ID, Target: "hero"}))
		got, _ := struck(e, w.ID)
		assert.Empty(t, got)
	})
}

func TestHeavyBlowRaisesShares(t *testing.T) {
	setup := func(t *testing.T, traits ...string) (*Engine, *Unit, *Unit) {
		e, a, b := arena(t)
		w := e.unit("archer1")
		w.X, w.Y = 5, 5
		a.HP, b.HP, w.HP = 10000, 10000, 10000
		e.data.Rules.HitFloor, e.data.Rules.HitCeil, e.data.Rules.CritMultiplier = 100, 100, 1
		with(e, "boss", traits...)
		return e, a, b
	}
	dealt := func(e *Engine, by string) (out []int) {
		for _, ev := range e.events {
			if ev.Kind == "damage" && ev.Actor == by {
				out = append(out, ev.Amount)
			}
		}
		return
	}
	t.Run("second strike", func(t *testing.T) {
		for _, full := range []bool{false, true} {
			e, a, b := setup(t)
			if full {
				with(e, "boss", "full_share_t")
			}
			e.events = nil
			e.attackOn(b, a, []*Unit{a}, e.basic(), map[string]float64{a.ID: 1}, ext{}, true, false, true)
			got := dealt(e, b.ID)
			require.Len(t, got, 2)
			assert.Equal(t, full, got[0] == got[1], "full %t: %v", full, got)
		}
	})
	t.Run("counter", func(t *testing.T) {
		e, _, b := setup(t)
		assert.Equal(t, e.data.Rules.PlainCounterShare, e.counterShare(b))
		with(e, "boss", "full_share_t")
		assert.InDelta(t, e.data.Rules.PlainCounterShare+.3, e.counterShare(b), 1e-9)
	})
	t.Run("beside the target", func(t *testing.T) {
		e, a, b := setup(t, "cross_t")
		_, share, _ := e.victims(b, a, e.basic())
		before := share["archer1"]
		with(e, "boss", "full_share_t")
		_, share, _ = e.victims(b, a, e.basic())
		assert.InDelta(t, before+.3, share["archer1"], 1e-9)
	})
}

func TestRangedCrossSplash(t *testing.T) {
	setup := func(t *testing.T, mode string) []*Unit {
		e, a, b := arena(t)
		t1 := e.unit("troop#1")
		t1.X, t1.Y, t1.HP = b.X+1, b.Y, 10000
		e.data.Traits["cross_ranged_t"] = content.Trait{Name: "CrossRanged", Effects: map[string]float64{content.FxSplashCrossRanged: .5}}
		with(e, a.ID, "cross_ranged_t")
		s := e.basic()
		s.Mode = mode
		hit, _, _ := e.victims(a, b, s)
		return hit
	}
	t.Run("a ranged attack hits the cross", func(t *testing.T) {
		assert.Len(t, setup(t, content.ModeRanged), 2)
	})
	t.Run("a melee attack hits the target alone", func(t *testing.T) {
		assert.Len(t, setup(t, content.ModeMelee), 1)
	})
	t.Run("the 3x3 one too", func(t *testing.T) {
		e, a, b := arena(t)
		t1 := e.unit("troop#1")
		t1.X, t1.Y, t1.HP = b.X+1, b.Y+1, 10000
		e.data.Traits["square_ranged_t"] = content.Trait{Name: "SquareRanged", Effects: map[string]float64{content.FxSplashSquareRanged: .5}}
		with(e, a.ID, "square_ranged_t")
		s := e.basic()
		for mode, n := range map[string]int{content.ModeRanged: 2, content.ModeMelee: 1} {
			s.Mode = mode
			hit, _, _ := e.victims(a, b, s)
			assert.Len(t, hit, n, mode)
		}
	})
}

func TestCrossShot(t *testing.T) {
	data, err := content.Load("../../assets/data")
	require.NoError(t, err)

	t.Run("named officers receive the skills in their bow classes", func(t *testing.T) {
		for _, tc := range []struct {
			id, trait, class, skill string
			level                   int
		}{
			{"handang", "cross_shot_art", "archer", "cross_shot", 6},
			{"zhanghe", "cross_shot_art", "horse_archer", "cross_shot", 20},
			{"taishici", "volley", "horse_archer", "barrage", 20},
		} {
			officer := data.Characters[tc.id]
			class := data.PromotedAt(tc.id, officer.Class, tc.level)
			assert.Equal(t, tc.class, class, tc.id)
			assert.Equal(t, "bow", data.Classes[class].Weapon, tc.id)
			assert.Contains(t, data.PlannedTraits(tc.id, class, tc.level, nil), tc.trait, tc.id)
			assert.Contains(t, data.Skills[tc.skill].Requires, tc.trait, tc.id)
		}
	})

	t.Run("pays once and hits only the center and the cross arms", func(t *testing.T) {
		e, u, center := arena(t)
		e.data.Skills["cross_shot"] = data.Skills["cross_shot"]
		e.data.Traits["cross_shot_art"] = data.Traits["cross_shot_art"]
		e.char(u.Character).Class = "archer"
		e.char(u.Character).Equipment["weapon"] = "short_bow"
		with(e, u.ID, "cross_shot_art")
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		center.X, center.Y, center.HP = 7, 6, 10000
		arm, diagonal := e.unit("troop#1"), e.unit("troop#2")
		arm.X, arm.Y, arm.HP = 7, 7, 10000
		diagonal.X, diagonal.Y, diagonal.HP = 8, 7, 10000
		u.MP = 64

		s := e.data.Skills["cross_shot"]
		_, share, cells := e.victims(u, center, s)
		assert.Len(t, cells, 4)
		assert.Equal(t, map[string]float64{center.ID: 1, arm.ID: .5}, share)

		cmd := Command{Kind: "skill", Actor: u.ID, Target: center.ID, Skill: "cross_shot"}
		preview, err := e.Preview(cmd)
		require.NoError(t, err)
		assert.Equal(t, 32, preview.Cost)
		assert.ElementsMatch(t, []string{center.ID, arm.ID}, preview.Targets)
		require.NoError(t, e.useSkill(u, cmd))
		assert.Equal(t, 32, u.MP)
		assert.Less(t, center.HP, 10000)
		assert.Less(t, arm.HP, 10000)
		assert.Equal(t, 10000, diagonal.HP)
	})
}
