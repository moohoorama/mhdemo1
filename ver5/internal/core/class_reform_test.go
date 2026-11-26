package core

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"testing"
)

func TestFormationAndDeputyStatIncrease(t *testing.T) {
	e, hero, _ := arena(t)
	ally := e.unit("archer1")
	ally.X, ally.Y = hero.X-1, hero.Y
	e.data.Traits["formation_test"] = content.Trait{Effects: map[string]float64{content.FxFormationDefense: .04}}
	e.data.Traits["increase_test"] = content.Trait{Effects: map[string]float64{content.FxStatIncrease + "defense": .1}}
	base := e.stats(hero).Defense
	with(e, "hero", "formation_test")
	assert.Equal(t, int(float64(base)*1.04), e.stats(hero).Defense)
	with(e, "hero", "increase_test")
	with(e, "archer1", "increase_test")
	e.char("hero").Deputy = "archer1"
	assert.InDelta(t, .2, e.fx(hero, content.FxStatIncrease+"defense"), .00001)
}

func TestReprisalAccumulatesOnlyDamageAndExpires(t *testing.T) {
	e, hero, boss := arena(t)
	e.data.Traits["reprisal_test"] = content.Trait{Effects: map[string]float64{content.FxReprisal: .05}}
	with(e, "hero", "reprisal_test")
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	e.strike(boss, hero, e.basic(), blow{share: 0})
	assert.Zero(t, hero.Status["reprisal"].Turns)
	e.strike(boss, hero, e.basic(), blow{share: 1})
	require.Equal(t, 1, hero.Status["reprisal"].Turns)
	apply(t, e, Command{Kind: "wait", Actor: hero.ID})
	assert.Zero(t, e.unit("hero").Status["reprisal"].Turns)
}

func TestReflectedCriticalSpellLeavesMark(t *testing.T) {
	e, _, boss := arena(t)
	sage(e, 4, 6)
	e.data.Rules.SpellFloor, e.data.Rules.SpellCeil = 100, 100
	with(e, "boss", "reflect_t", "deep")
	u := e.unit("sage")
	u.HP, boss.HP = 9999, 9999
	x, y := u.X, u.Y
	r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	assert.Equal(t, byte('e'), e.Cell(x, y).State)
	found := false
	for _, ev := range r.Events {
		if ev.Kind == "counter-spell" && ev.Text == "flame" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestReprisalBoostsDirectAttackButNotCounter(t *testing.T) {
	e, hero, boss := arena(t)
	e.data.Traits["reprisal_test"] = content.Trait{Effects: map[string]float64{content.FxReprisal: .05}}
	with(e, "hero", "reprisal_test")
	hero.Status["reprisal"] = Status{Turns: 2}
	e.reprisalActor, e.reprisalBonus = hero.ID, .10
	base := e.damage(hero, boss, e.basic())
	boss.HP = 99999
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	e.data.Rules.Crit = content.Chance{}
	n, _, _ := e.strike(hero, boss, e.basic(), blow{share: 1})
	assert.Equal(t, int(base*1.1), n)
	boss.HP = 99999
	n, _, _ = e.strike(hero, boss, e.basic(), blow{share: 1, derived: true})
	assert.Equal(t, int(base), n)
}

func TestLifeStealRecoversTenPercentOfDamage(t *testing.T) {
	e, hero, _ := arena(t)
	e.data.Traits["life_test"] = content.Trait{Effects: map[string]float64{content.FxLifeSteal: .1}}
	with(e, "hero", "life_test")
	hero.HP = e.stats(hero).MaxHP - 100
	before := hero.HP
	e.leech(hero, 100)
	assert.Equal(t, before+10, hero.HP)
}
