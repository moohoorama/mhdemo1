package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestMeleeReachHasAShapeOfItsOwn(t *testing.T) {
	shape := func(e *Engine) []content.Point {
		offsets := e.offsets(e.unit("hero"), e.basic())
		slices.SortFunc(offsets, func(a, b content.Point) int { return a.Y*10 + a.X - b.Y*10 - b.X })
		return offsets
	}
	cross := []content.Point{{Y: -1}, {X: -1}, {X: 1}, {Y: 1}}
	square := []content.Point{{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1}, {X: -1}, {X: 1}, {X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1}}
	bigCross := []content.Point{{Y: -2}, {Y: -1}, {X: -2}, {X: -1}, {X: 1}, {X: 2}, {Y: 1}, {Y: 2}}
	diamond := []content.Point{{Y: -2}, {X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1}, {X: -2}, {X: -1}, {X: 1}, {X: 2},
		{X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1}, {Y: 2}}

	e, _, _ := arena(t)
	assert.Equal(t, cross, shape(e), "the four straight neighbours")
	with(e, "hero", "diag_t")
	assert.Equal(t, square, shape(e), "대각공격: the eight around")

	e2, _, _ := arena(t)
	with(e2, "hero", "long_t")
	assert.Equal(t, bigCross, shape(e2), "장창: two along the straight lines")
	with(e2, "hero", "diag_t")
	assert.Equal(t, diamond, shape(e2), "both: the diamond of two")
}

func TestALongSpearBlowStaysMelee(t *testing.T) {
	e, a, b := arena(t)
	with(e, "hero", "long_t")
	a.X = b.X - 2
	require.True(t, e.inRange(a, b, e.basic()))
	assert.False(t, e.rangedAttack(a, b, e.basic()), "two cells away, still the spear")
	b.Status["guard"] = Status{Turns: 2}
	assert.Zero(t, e.hitChance(a, b, e.basic()), "경계 dodges the melee blow")
	with(e, "boss", "shield")
	delete(b.Status, "guard")
	assert.NotZero(t, e.hitChance(a, b, e.basic()), "간접공격면역 does not stop a spear")
}

func TestNPCsHoldTheirPlansFromALevel(t *testing.T) {
	e, _, _ := arena(t)
	class := e.char("troop#2").Class // a level 5 troop
	cl := e.data.Classes[class]
	cl.Plan = []content.LevelTrait{{Level: 5, Trait: "diag_t"}, {Level: 6, Trait: "long_t"}}
	e.data.Classes[class] = cl
	sp := content.Spawn{Character: "troop", Level: 5}
	u := e.spawn("troop", "troop", "enemy", 9, 9, sp)
	c := e.char(u.Character)
	assert.True(t, e.hasTrait(c, "diag_t"))
	assert.False(t, e.hasTrait(c, "long_t"), "not yet at level 5")
	ally := e.spawn("troop", "troop", "ally", 9, 8, content.Spawn{Character: "troop", Level: 5, AI: true})
	assert.True(t, e.hasTrait(e.char(ally.Character), "diag_t"), "allies follow the plan too")

	e.data.Rules.TraitPlanStep = 15
	def := e.data.Characters["troop"]
	def.Traits = []string{"diag_t", "long_t", "shield"}
	e.data.Characters["troop"] = def
	cl.Plan = nil
	e.data.Classes[class] = cl
	for level, want := range map[int][]string{1: {"diag_t"}, 15: {"diag_t"}, 16: {"diag_t", "long_t"}, 31: {"diag_t", "long_t", "shield"}, 60: {"diag_t", "long_t", "shield"}} {
		assert.Equal(t, want, e.data.PlannedTraits("troop", class, level, nil), "level %d", level)
	}
}

func TestChainAndSupplyReachTheWeakestAround(t *testing.T) {
	e, _, _ := arena(t)
	e.data.Rules.SpellFloor, e.data.Rules.SpellCeil = 100, 100
	sage(e, 4, 6)
	with(e, "sage", "chain_t")
	t1, t2 := e.unit("troop#1"), e.unit("troop#2")
	t1.HP, t1.X, t1.Y = e.stats(t1).MaxHP, 7, 6
	t2.HP, t2.X, t2.Y = e.stats(t2).MaxHP/3, 7, 7 // diagonal to the boss, and weaker
	r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	assert.True(t, answered(r, "sage", "troop#2"), "연환 reaches the weakest of the eight around")
	assert.False(t, answered(r, "sage", "troop#1"), "and only that one")

	e2, _, _ := arena(t)
	with(e2, "hero", "share")
	a1 := e2.unit("archer1")
	a1.X, a1.Y, a1.HP = 4, 5, 10 // diagonal to the hero
	e2.unit("hero").HP = 10
	apply(t, e2, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	assert.Greater(t, e2.unit("archer1").HP, 10, "보급 reaches a diagonal ally too")
}
