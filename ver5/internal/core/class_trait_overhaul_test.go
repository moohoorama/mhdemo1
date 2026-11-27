package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// The 2026-11-26 class trait overhaul: shares a blow gains or loses are added into one coefficient, defense
// raises share one defense coefficient, and walking effects last through the other side's turn.
func TestClassTraitOverhaul(t *testing.T) {
	trait := func(e *Engine, id string, fx map[string]float64) {
		e.data.Traits[id] = content.Trait{Name: id, Kind: "talent", Effects: fx}
	}
	shooter := func(e *Engine, x int) *Unit {
		u := e.unit("archer1")
		u.X, u.Y = x, 6
		return u
	}

	t.Run("방패술 raises defense only against shots and adds to other defense raises", func(t *testing.T) {
		e, hero, boss := arena(t)
		trait(e, "shield_t", map[string]float64{content.FxRangedDefense: .3})
		arch := shooter(e, 4)
		shot, melee := e.damage(arch, boss, e.basic()), e.damage(hero, boss, e.basic())
		with(e, "boss", "shield_t")
		assert.Less(t, e.damage(arch, boss, e.basic()), shot)
		assert.InDelta(t, melee, e.damage(hero, boss, e.basic()), 1e-9, "melee blows meet the plain defense")

		boss.Status = map[string]Status{"defense": {Turns: 2, Value: 20}}
		st := e.stats(boss)
		_, kd := e.coefficients(boss, st.MaxHP)
		require.InDelta(t, 1.2, kd, 1e-9)
		want := math.Floor(float64(e.baseStats(boss)[1]) * 1.5) // 1 + 20% + 30%, added, not multiplied
		tf := e.terrain(boss, boss.X, boss.Y).Factor
		without := e.damage(arch, boss, e.basic())
		boss.Status = map[string]Status{"defense": {Turns: 2, Value: 20}}
		e.data.Traits["shield_t"] = content.Trait{}
		plain := e.damage(arch, boss, e.basic())
		assert.InDelta(t, (want-float64(st.Defense))*tf, plain-without, 1e-6)
	})

	t.Run("physical, ranged and melee wards are added and applied once", func(t *testing.T) {
		e, hero, boss := arena(t)
		trait(e, "wards_t", map[string]float64{content.FxPhysicalWard: .2, content.FxRangedWard: .5, content.FxMeleeWard: .15})
		arch := shooter(e, 4)
		shot, melee := e.damage(arch, boss, e.basic()), e.damage(hero, boss, e.basic())
		with(e, "boss", "wards_t")
		assert.InDelta(t, shot*(1-.7), e.damage(arch, boss, e.basic()), 1e-6)
		assert.InDelta(t, melee*(1-.35), e.damage(hero, boss, e.basic()), 1e-6)
	})

	t.Run("a critical, a counter, a second strike and 요극 add into the same coefficient", func(t *testing.T) {
		e, hero, boss := arena(t)
		trait(e, "ward_t", map[string]float64{content.FxPhysicalWard: .2})
		with(e, "boss", "ward_t")
		R := e.data.Rules
		full := e.damage(hero, boss, e.basic()) / .8
		assert.InDelta(t, full*(1+R.CritMultiplier-1-.2), e.damageAt(hero, boss, e.basic(), ext{crit: true}), 1e-6, "critical adds its 0.5")
		assert.InDelta(t, full*(R.PlainCounterShare-.2), e.damageAt(hero, boss, e.basic(), ext{rule: R.PlainCounterShare}), 1e-6, "a counter starts from 0.5")
		assert.InDelta(t, full*(R.PlainCounterShare+R.CritMultiplier-1-.2), e.damageAt(hero, boss, e.basic(), ext{rule: R.PlainCounterShare, crit: true}), 1e-6)
		sweep := e.basic()
		sweep.Rule = .5
		assert.InDelta(t, full*(.5-.2), e.damage(hero, boss, sweep), 1e-6, "요극 starts from 0.5")
	})

	t.Run("원거리 위력 grows by 5% for each cell beyond two up to four, 밀집사격 by 4% per adjacent ally", func(t *testing.T) {
		e, _, boss := arena(t)
		trait(e, "range_t", map[string]float64{content.FxRangePower: .05})
		trait(e, "dense_t", map[string]float64{content.FxDenseFire: .04})
		for x, want := range map[int]float64{4: 1, 3: 1.05, 2: 1.10, 1: 1.15, 0: 1.20} {
			arch := shooter(e, x)
			e.char("archer1").Learned = nil
			base := e.damage(arch, boss, e.basic())
			with(e, "archer1", "range_t")
			assert.InDelta(t, base*want, e.damage(arch, boss, e.basic()), 1e-6, "from x=%d", x)
		}
		arch := shooter(e, 4) // the hero stands next to it
		e.char("archer1").Learned = nil
		base := e.damage(arch, boss, e.basic())
		with(e, "archer1", "dense_t")
		assert.InDelta(t, base*1.04, e.damage(arch, boss, e.basic()), 1e-6)
	})

	t.Run("a walk lasts through the other side's turn for 돌진, 기동공격 and 기동방어", func(t *testing.T) {
		e, hero, boss := arena(t)
		trait(e, "charge_t", map[string]float64{content.FxChargePower: .05, content.FxMobileAttack: .1})
		trait(e, "guard_t", map[string]float64{content.FxMobileGuard: .1})
		with(e, "hero", "charge_t")
		with(e, "boss", "guard_t")
		plain, atk := e.damage(hero, boss, e.basic()), e.stats(hero).Attack

		require.NoError(t, e.execute(Command{Kind: "move", Actor: hero.ID, X: 5, Y: 7}))
		require.Equal(t, 1, hero.Steps)
		walkedBlow := e.damage(hero, boss, e.basic())
		assert.Greater(t, walkedBlow, plain)
		assert.Equal(t, int(float64(atk)*1.05), e.stats(hero).Attack, "5% for each cell walked")

		e.endFaction()
		require.NotEqual(t, "ally", e.state.Turn)
		assert.Zero(t, hero.Steps)
		assert.Equal(t, 1, hero.LastSteps, "kept for the counters of the other side's turn")
		assert.Equal(t, int(float64(atk)*1.05), e.stats(hero).Attack)

		for e.state.Turn != "ally" {
			e.endFaction()
		}
		assert.Zero(t, hero.LastSteps)
		assert.Equal(t, atk, e.stats(hero).Attack)

		boss.LastSteps = 1
		assert.Less(t, e.damage(hero, boss, e.basic()), plain, "a foe that walked takes 10% less")
	})

	t.Run("저지 slows after a landed melee blow too", func(t *testing.T) {
		e, hero, boss := arena(t)
		trait(e, "slow_t", map[string]float64{content.FxRangedSlow: 2, content.FxAccuracy: 100})
		with(e, "hero", "slow_t")
		boss.Status = map[string]Status{}
		require.NoError(t, e.execute(Command{Kind: "attack", Actor: hero.ID, Target: boss.ID}))
		assert.Equal(t, -2, boss.Status["speed"].Value)
	})
}
