package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestAssistFollowsAnAlliesAttack(t *testing.T) {
	e, _, _ := arena(t)
	e.unit("archer1").X, e.unit("archer1").Y = 6, 5 // adjacent to the boss, also an ally of the attacker
	e.char("archer1").Class = "rider"
	e.char("archer1").Equipment = map[string]string{}
	with(e, "archer1", "assist_t")
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	if !answered(r, "archer1", "boss") {
		t.Fatal("assist_t did not follow the attack")
	}
	e2, _, _ := arena(t)
	e2.unit("archer1").X, e2.unit("archer1").Y = 6, 5
	e2.char("archer1").Class = "rider"
	e2.char("archer1").Equipment = map[string]string{}
	with(e2, "archer1", "assist_t")
	e2.state.Turn = "enemy"
	e2.unit("boss").Faction, e2.unit("hero").Faction = "ally", "enemy"
}
func TestEightfoldHelpsAlliesAndHarmsFoes(t *testing.T) {
	e, _, _ := arena(t)
	e.data.Rules.SpellFloor, e.data.Rules.SpellCeil = 100, 100
	sage(e, 4, 6)
	with(e, "sage", "mercy")
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "eight"})
	p := e.data.Rules.BuffPercent
	b, h := e.unit("boss"), e.unit("hero")
	if b.Status["attack"].Value != -p || b.Status["defense"].Value != -p {
		t.Fatalf("boss %+v", b.Status)
	}
	if h.Status["attack"].Value != p || h.Status["defense"].Value != p {
		t.Fatalf("hero beside the boss %+v", h.Status)
	}
}
func TestLineSpellReachesAlongTheAxis(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 2, 6)
	e.unit("hero").X, e.unit("hero").Y = 1, 1
	t1 := e.unit("troop#1")
	t1.HP, t1.X, t1.Y = e.stats(t1).MaxHP, 5, 6
	line := e.data.Skills["lightning"]
	got := e.targets(e.unit("sage"), e.unit("boss"), line)
	if len(got) != 2 {
		t.Fatalf("line hit %d units", len(got))
	}
}
func TestChainReachesANeighbour(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	with(e, "sage", "chain_t")
	t1 := e.unit("troop#1")
	t1.HP, t1.X, t1.Y = e.stats(t1).MaxHP, 6, 5
	r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	if !answered(r, "sage", "troop#1") {
		t.Fatal("chain did not reach the neighbour")
	}
}
func TestSpellRangeSureAndCrit(t *testing.T) {
	e, _, b := arena(t)
	sage(e, 1, 6)
	flame := e.data.Skills["flame"]
	u := e.unit("sage")
	if e.inRange(u, b, flame) {
		t.Fatal("5 cells is out of reach 3")
	}
	with(e, "sage", "reach")
	if !e.inRange(u, b, flame) {
		t.Fatal("spell_range +3")
	}
	with(e, "sage", "firestorm", "deep")
	if e.hitChance(u, b, flame) != 100 || !e.critForced(u, b, flame) {
		t.Fatal("sure and crit for the element")
	}
	if e.fx(u, "sure:water") != 0 || e.fx(u, "crit:water") != 0 {
		t.Fatal("fire traits do not touch other elements")
	}
}
func TestCostHalf(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	u := e.unit("sage")
	flame := e.data.Skills["flame"]
	if e.cost(u, flame) != 30 {
		t.Fatal("cost")
	}
	with(e, "sage", "half")
	if e.cost(u, flame) != 15 {
		t.Fatal("cost_half")
	}
	assertRejected(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "flame"}) // friendly target
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	if !e.unit("sage").Acted || !e.unit("sage").Done {
		t.Fatal("skill did not finish the action")
	}
	assertRejected(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
}
func TestPoisonLeechAndAwe(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "poison_t", "leech_t")
	e.unit("hero").HP = 200
	e.unit("boss").HP = e.stats(e.unit("boss")).MaxHP
	for i := 0; i < 8; i++ {
		resetActions(e.unit("hero"))
		apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
		e.state.Learning = nil
		if e.unit("boss").Status["poison"].Turns > 0 {
			break
		}
	}
	if e.unit("boss").Status["poison"].Turns == 0 {
		t.Fatal("poison_on_hit never landed in 8 swings")
	}
	if e.unit("hero").HP <= 200-1 && e.unit("hero").HP == 200 {
		t.Fatal("life steal healed nothing")
	}
}
func TestStatusImmunityAndCleanse(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	with(e, "boss", "calm")
	if e.hitChance(e.unit("sage"), e.unit("boss"), e.data.Skills["weaken"]) != 0 {
		t.Fatal("status_immune")
	}
	e.unit("boss").Status["poison"] = Status{Turns: 3}
	e.state.Turn = "ally"
	e.endFaction() // enemy turn starts: cleanse
	if e.unit("boss").Status["poison"].Turns != 0 {
		t.Fatal("cleanse")
	}
}
func TestSpellCureEndsTheAllysAilments(t *testing.T) {
	for _, cure := range []bool{false, true} {
		e, h, _ := arena(t)
		sage(e, 4, 6)
		with(e, "sage", "mercy")
		if cure {
			with(e, "sage", "mercy", "needles")
		}
		h.HP = e.stats(h).MaxHP / 2
		h.Status["poison"] = Status{Turns: 3}
		apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "mend"})
		assert.Equal(t, cure, e.unit("hero").Status["poison"].Turns == 0, "spell_cure %v", cure)
	}
}
func TestRiderShootsBeyondItsSpear(t *testing.T) {
	e, _, _ := arena(t)
	e.char("archer1").Class = "rider"
	e.char("archer1").Equipment = map[string]string{}
	u, v := e.unit("archer1"), e.unit("boss")
	if e.shootClass(u) != "" {
		t.Fatal("a rider without mounted_range has a shoot look")
	}
	with(e, "archer1", "steed")
	if e.shootClass(u) != "archer" {
		t.Fatalf("shoot class %q", e.shootClass(u))
	}
	for _, c := range []struct {
		dx, dy      int
		reach, shot bool
	}{{1, 0, true, false}, {1, 1, true, true}, {2, 0, true, true}, {0, 2, true, true}, {2, 1, false, false}} {
		u.X, u.Y, v.X, v.Y = 5, 5, 5+c.dx, 5+c.dy
		if got := e.inRange(u, v, e.basic()); got != c.reach {
			t.Errorf("target at (%d,%d): reach %v, want %v", c.dx, c.dy, got, c.reach)
		}
		if got := e.rangedAttack(u, v, e.basic()); c.reach && got != c.shot {
			t.Errorf("target at (%d,%d): shot %v, want %v", c.dx, c.dy, got, c.shot)
		}
	}
}
func TestFlankStrikesBesideTheTarget(t *testing.T) {
	e, hero, boss := arena(t)
	with(e, "hero", "sweeper")
	hero.X, hero.Y, boss.X, boss.Y = 8, 8, 9, 8
	sp := content.Spawn{Character: "troop", Level: 1}
	up := e.spawn(sp.Character, sp.Character, "enemy", 9, 7, sp)
	down := e.spawn(sp.Character, sp.Character, "enemy", 9, 9, sp)
	behind := e.spawn(sp.Character, sp.Character, "enemy", 10, 8, sp)
	_, share := e.victims(hero, boss, e.basic())
	if share[boss.ID] != 1 || share[up.ID] != .5 || share[down.ID] != .5 || share[behind.ID] != 0 {
		t.Fatalf("shares %v", share)
	}
}
func TestEvadeLowersTheHitChanceOfPhysicalAttacks(t *testing.T) {
	e, hero, boss := arena(t)
	base := e.hitChance(boss, hero, e.basic())
	with(e, "hero", "evader")
	got := e.hitChance(boss, hero, e.basic())
	if base <= e.data.Rules.HitFloor+15 || got != base-15 {
		t.Fatalf("hit chance %v -> %v, want %v", base, got, base-15)
	}
}

func TestPouchEatsAWholeBeanFromTheStock(t *testing.T) {
	setup := func(t *testing.T) (*Engine, *Unit, int) {
		e, _, _ := arena(t)
		with(e, "boss", "pouch_t")
		boss := e.unit("boss")
		return e, boss, int(e.fx(boss, content.FxPouchHeal))
	}
	t.Run("a loss smaller than a bean leaves the pouch shut", func(t *testing.T) {
		e, boss, k := setup(t)
		boss.HP = e.stats(boss).MaxHP - k + 1
		e.pouchHeal(boss)
		assert.Equal(t, e.stats(boss).MaxHP-k+1, boss.HP)
		assert.Equal(t, 1, boss.Items[e.data.Rules.PouchItem])
	})
	t.Run("a loss of a whole bean eats one", func(t *testing.T) {
		e, boss, k := setup(t)
		boss.HP = e.stats(boss).MaxHP - k - 7
		e.pouchHeal(boss)
		assert.Equal(t, e.stats(boss).MaxHP-7, boss.HP)
		assert.Zero(t, boss.Items[e.data.Rules.PouchItem])
	})
	t.Run("an empty pouch heals nothing", func(t *testing.T) {
		e, boss, k := setup(t)
		boss.Items[e.data.Rules.PouchItem] = 0
		boss.HP = e.stats(boss).MaxHP - 2*k
		e.pouchHeal(boss)
		assert.Equal(t, e.stats(boss).MaxHP-2*k, boss.HP)
	})
	t.Run("a blow that leaves it a bean short makes it eat once", func(t *testing.T) {
		e, _, k := setup(t)
		for i := 0; i < 8; i++ {
			boss := e.unit("boss")
			boss.HP = e.stats(boss).MaxHP - k
			boss.Items[e.data.Rules.PouchItem] = 1
			resetActions(e.unit("hero"))
			apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
			e.state.Learning = nil
			dealt, healed := 0, 0
			for _, ev := range e.events {
				if ev.Kind == "damage" && ev.Target == "boss" {
					dealt += ev.Amount
				}
				if ev.Kind == "pouch-heal" && ev.Actor == "boss" {
					healed += ev.Amount
				}
			}
			boss = e.unit("boss")
			if dealt == 0 || boss.HP <= 0 {
				continue
			}
			assert.Equal(t, k, healed)
			assert.Zero(t, boss.Items[e.data.Rules.PouchItem])
			return
		}
		t.Fatal("no blow landed in 8 swings")
	})
	t.Run("물심양면 makes the bean heal more", func(t *testing.T) {
		e, boss, k := setup(t)
		with(e, "boss", "bounty_t")
		boss.HP = e.stats(boss).MaxHP - 2*k
		e.pouchHeal(boss)
		assert.Equal(t, e.stats(boss).MaxHP-2*k+k*3/2, boss.HP)
	})
	t.Run("보급 shares the bean with the weakest ally around", func(t *testing.T) {
		e, boss, k := setup(t)
		with(e, "boss", "share")
		mate := e.unit("troop#1")
		mate.X, mate.Y, mate.HP = boss.X, boss.Y+1, 1
		boss.HP = e.stats(boss).MaxHP - k
		e.pouchHeal(boss)
		assert.Equal(t, e.stats(boss).MaxHP, boss.HP)
		assert.Equal(t, min(e.stats(mate).MaxHP, 1+k), mate.HP)
		assert.Zero(t, boss.Items[e.data.Rules.PouchItem], "one bean feeds both")
	})
}

func TestChargeAccuracyGrowsWithTheCellsWalked(t *testing.T) {
	e, a, b := arena(t)
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 0, 100
	e.data.Traits["ambush_t"] = content.Trait{Name: "기습", Kind: "talent", Tone: "attack", Effects: map[string]float64{content.FxChargeAccuracy: 3}}
	still := e.hitChance(a, b, e.basic())
	with(e, "hero", "ambush_t")
	hero := e.unit("hero")
	assert.Equal(t, still, e.hitChance(hero, b, e.basic()), "no walk, no bonus")
	hero.Steps = 4
	assert.InDelta(t, still+12, e.hitChance(hero, b, e.basic()), 1e-9, "four cells walked: +12%p")
}

func TestOpeningDealsMoreOnAFreshTarget(t *testing.T) {
	e, a, b := arena(t)
	full := e.stats(b).MaxHP
	b.HP = full
	plain := e.damage(a, b, e.basic())
	e.data.Traits["open_t"] = content.Trait{Name: "개전", Kind: "talent", Tone: "attack", Effects: map[string]float64{content.FxOpening: .5}}
	with(e, "hero", "open_t")
	hero := e.unit("hero")
	assert.InDelta(t, plain*1.25, e.damage(hero, b, e.basic()), 1e-6, "full HP: +25%")
	b.HP = full / 2
	half := e.damage(hero, b, e.basic())
	b.HP = full / 4
	assert.InDelta(t, half, e.damage(hero, b, e.basic()), 1e-6, "half HP or less: no bonus")
}

func TestCrushTakesTheDefeatedFoesSpellPoints(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	with(e, "sage", "drain")
	u, b := e.unit("sage"), e.unit("boss")
	flame := e.data.Skills["flame"]
	u.MP, b.HP, b.MP = e.cost(u, flame), 1, 30
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	require.Equal(t, 0, e.unit("boss").HP, "the flame defeats the boss")
	assert.Equal(t, min(e.stats(u).MaxMP, 30), e.unit("sage").MP)
	assert.Equal(t, 30-e.unit("sage").MP, e.unit("boss").MP)
}

func TestStatusDoubleLengthensTheCastersAilments(t *testing.T) {
	for _, linger := range []bool{false, true} {
		e, _, _ := arena(t)
		sage(e, 4, 6)
		if linger {
			with(e, "sage", "linger")
		}
		apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "seal"})
		want := map[bool]int{false: 2, true: 4}[linger]
		assert.Equal(t, want, e.unit("boss").Status["seal"].Turns, "status_double %v", linger)
	}
}

func TestWeakPointCritsAnAilingFoe(t *testing.T) {
	e, hero, boss := arena(t)
	flame := e.data.Skills["flame"]
	assert.False(t, e.critForced(hero, boss, e.basic()))
	with(e, "hero", "exploit")
	assert.False(t, e.critForced(hero, boss, e.basic()), "a healthy foe")
	boss.Status["confusion"] = Status{Turns: 2}
	assert.True(t, e.critForced(hero, boss, e.basic()), "an attack")
	assert.True(t, e.critForced(hero, boss, flame), "a spell")
	boss.Status = map[string]Status{"attack": {Turns: 2, Value: -20}}
	assert.False(t, e.critForced(hero, boss, e.basic()), "a debuff is no ailment")
}

func TestDeputySharesExperience(t *testing.T) {
	t.Run("alone the officer keeps all of it", func(t *testing.T) {
		e, _, _ := arena(t)
		e.xp("hero", 1000)
		assert.Equal(t, 1000, e.char("hero").XP)
	})
	t.Run("with a deputy the officer keeps 80 and the deputy gets 60", func(t *testing.T) {
		e, _, _ := arena(t)
		e.char("hero").Deputy = "archer1"
		before := e.char("archer1").XP
		e.xp("hero", 1000)
		assert.Equal(t, 800, e.char("hero").XP)
		assert.Equal(t, before+600, e.char("archer1").XP)
	})
	t.Run("fractions carry over", func(t *testing.T) {
		e, _, _ := arena(t)
		e.char("hero").Deputy = "archer1"
		before := e.char("archer1").XP
		for i := 0; i < 5; i++ {
			e.xp("hero", 1) // 0.8 and 0.6 each time
		}
		assert.Equal(t, 4, e.char("hero").XP)
		assert.Equal(t, before+3, e.char("archer1").XP)
	})
}

func TestDeputyFightsItsDuel(t *testing.T) {
	e, hero, boss := arena(t)
	e.unit("archer1").HP = 0 // off the field: it rides with the hero now
	e.char("hero").Deputy = "archer1"
	stage := &e.data.Stages[e.state.Stage]
	stage.Duels = append(stage.Duels, content.Duel{ID: "deputy-duel", Ally: "archer1", Enemy: boss.Character, Outcome: "draw", AllyResult: "stay", EnemyResult: "stay", Text: "t"})
	assert.Contains(t, e.DuelMarks(), DuelMark{Unit: hero.ID, Group: len(stage.Duels) - 1}, "the deputy's mark stands on his officer")
	assert.True(t, e.DuelPending(hero.ID, boss.ID))
	heroLevel, deputyLevel := e.char("hero").Level, e.char("archer1").Level
	e.events = nil
	e.autoDuels()
	var duel Event
	for _, ev := range e.events {
		if ev.Kind == "duel" {
			duel = ev
		}
	}
	require.Equal(t, "deputy-duel", duel.DuelID)
	assert.Equal(t, hero.ID, duel.Actor)
	assert.Equal(t, "archer1", duel.Fighter)
	assert.Equal(t, deputyLevel+1, e.char("archer1").Level, "the deputy who fought earns the duel's level")
	assert.Equal(t, heroLevel, e.char("hero").Level, "his officer earns nothing from it")
}

func TestChargeEndsWithTheWalkersTurn(t *testing.T) {
	e, _, _ := arena(t)
	e.data.Traits["rush_t"] = content.Trait{Name: "돌진", Kind: "talent", Tone: "attack", Effects: map[string]float64{content.FxChargePower: .05}}
	with(e, "hero", "rush_t")
	hero := e.unit("hero")
	still := e.stats(hero).Attack
	hero.Steps = 6
	require.Greater(t, e.stats(hero).Attack, still, "six cells walked charge the attack")
	applyOK(t, e, Command{Kind: "end"})
	hero = e.unit("hero")
	assert.Zero(t, hero.Steps)
	assert.Equal(t, still, e.stats(hero).Attack, "the enemy's turn meets the walker uncharged")
}
