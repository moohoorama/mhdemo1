package core

import (
	"slices"
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
	_, share, _ := e.victims(hero, boss, e.basic())
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

func TestWeakPointCritsAStatWeakenedFoe(t *testing.T) {
	e, hero, boss := arena(t)
	flame := e.data.Skills["flame"]
	assert.False(t, e.critForced(hero, boss, e.basic()))
	with(e, "hero", "exploit")
	assert.False(t, e.critForced(hero, boss, e.basic()), "a healthy foe")
	boss.Status["confusion"] = Status{Turns: 2}
	assert.False(t, e.critForced(hero, boss, e.basic()), "confusion alone does not lower a stat")
	boss.Status = map[string]Status{"attack": {Turns: 2, Value: -20}}
	assert.True(t, e.critForced(hero, boss, e.basic()), "an attack")
	assert.True(t, e.critForced(hero, boss, flame), "a spell")
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

func TestPlunderPaysPerLandedBlow(t *testing.T) {
	e, _, _ := arena(t)
	e.data.Traits["plunder_t"] = content.Trait{Name: "탈취", Kind: "talent", Effects: map[string]float64{content.FxPlunder: 10}}
	with(e, "hero", "plunder_t")
	with(e, "boss", "plunder_t")
	e.unit("boss").HP = e.stats(e.unit("boss")).MaxHP
	for i := 0; i < 8 && e.state.Money == 0; i++ {
		resetActions(e.unit("hero"))
		apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
		e.state.Learning = nil
	}
	if e.state.Money == 0 || e.state.Money%10 != 0 {
		t.Fatalf("money after the hero's landed blows: %d, want a positive multiple of 10", e.state.Money)
	}
	before := e.state.Money
	e.unit("hero").HP = e.stats(e.unit("hero")).MaxHP
	e.strike(e.unit("boss"), e.unit("hero"), e.basic(), blow{share: 1})
	if e.state.Money != before {
		t.Fatalf("an enemy's blow paid the party: %d, want %d", e.state.Money, before)
	}
}

func TestOfficerSpellSpecialties(t *testing.T) {
	data, err := content.Load("../../assets/data")
	require.NoError(t, err)

	t.Run("debate and feint have separate accuracy scopes", func(t *testing.T) {
		e, u, v := arena(t)
		e.data.Traits["debate_test"] = data.Traits["confusion_debate"]
		e.data.Traits["feint_test"] = data.Traits["feint"]
		e.data.Traits["immunity_test"] = content.Trait{Effects: map[string]float64{content.FxStatusImmune: 1}}
		single := data.Skills["confuse"]
		area := data.Skills["great_confusion"]
		attack := data.Skills["scorch"]
		attack.Hit = 40

		with(e, u.ID, "debate_test")
		assert.Equal(t, 100.0, e.hitChance(u, v, single))
		assert.Less(t, e.hitChance(u, v, area), 100.0)
		assert.Less(t, e.hitChance(u, v, attack), 100.0)
		with(e, v.ID, "immunity_test")
		assert.Zero(t, e.hitChance(u, v, single))

		e.char(u.Character).Learned = []string{"feint_test"}
		e.char(v.Character).Learned = nil
		assert.Equal(t, 100.0, e.hitChance(u, v, attack))
		assert.Less(t, e.hitChance(u, v, single), 100.0)
		assert.Less(t, e.hitChance(u, v, area), 100.0)
	})

	t.Run("fire critical and movement penalty reach the extra enemy through chain", func(t *testing.T) {
		e, u, v := arena(t)
		side := e.unit("troop#1")
		side.X, side.Y, side.HP = v.X, v.Y+1, 10000
		v.HP = 10000
		for _, id := range []string{"chain", "firecraft", "spell_slowness"} {
			e.data.Traits[id] = data.Traits[id]
			with(e, u.ID, id)
		}
		fire, water := data.Skills["scorch"], data.Skills["torrent"]
		fire.Sure = true
		assert.True(t, e.critForced(u, v, fire))
		assert.False(t, e.critForced(u, v, water))
		e.spell(u, v, fire, Command{Kind: "skill"})
		for _, target := range []*Unit{v, side} {
			assert.Less(t, target.HP, 10000)
			assert.Equal(t, Status{Value: -2, Turns: 1}, target.Status["speed"])
			assert.Equal(t, e.unitClass(target).Move-2, e.movement(target))
		}
	})

	t.Run("data assigns personal traits without class overlap", func(t *testing.T) {
		for _, tc := range []struct {
			id, class string
			traits    []string
		}{
			{"guojia", "diplomat", []string{"confusion_debate"}},
			{"pangtong", "strategist", []string{"chain", "firecraft", "spell_slowness"}},
		} {
			officer := data.Characters[tc.id]
			assert.Equal(t, tc.class, officer.Class)
			assert.Equal(t, tc.traits, officer.Traits)
			for _, id := range tc.traits {
				assert.NotContains(t, data.Classes[tc.class].AllPool(), id)
			}
		}
	})
}

func TestOfficerReformBatch(t *testing.T) {
	contentData, err := content.Load("../../assets/data")
	require.NoError(t, err)

	t.Run("personal traits do not repeat inherited class learning", func(t *testing.T) {
		for id, officer := range contentData.Characters {
			pool := contentData.Classes[officer.Class].AllPool()
			for _, trait := range append(slices.Clone(officer.Traits), officer.Learn...) {
				assert.NotContains(t, pool, trait, id)
			}
			for trait := range officer.TraitAt {
				assert.NotContains(t, pool, trait, id)
			}
		}
		assert.Equal(t, .5, contentData.Traits["drain"].Effects[content.FxLifeSteal])
		assert.Contains(t, contentData.Characters["lumeng"].TraitAt, "drain")
		assert.Contains(t, contentData.Characters["luxun"].TraitAt, "noble_house")
		assert.Equal(t, "pangtong", contentData.Node("c27v-join-pangtong").Character)
		assert.Equal(t, []string{"meditate_art"}, contentData.Equipment["sanlue"].Traits)
	})

	t.Run("pa jin removes boosts before multiple magical debuffs", func(t *testing.T) {
		e, u, v := arena(t)
		for _, id := range []string{"break_formation", "spell_lower_agility", "spell_lower_defense", "lasting"} {
			e.data.Traits[id] = contentData.Traits[id]
			with(e, u.ID, id)
		}
		attack := e.data.Skills["flame"]
		attack.Sure = true
		v.HP = 10000
		e.applyStatus(v, "defense", 3, 20)
		e.spell(u, v, attack, Command{Kind: "skill"})
		for _, stat := range []string{"agility", "defense"} {
			assert.Equal(t, Status{Value: -20, Turns: e.data.Rules.BuffTurns + 2}, v.Status[stat])
		}
	})

	t.Run("terrain spells select empty cells and leave two-round marks", func(t *testing.T) {
		e, u, _ := arena(t)
		s := contentData.Skills["terrain_fire"]
		e.data.Skills["terrain_test"] = s
		e.data.Traits["terrain_craft"] = contentData.Traits["terrain_craft"]
		with(e, u.ID, "terrain_craft", "fire")
		u.MP = 100
		var choice Command
		for _, c := range e.LegalActions(u.ID).Commands {
			if c.Skill == "terrain_test" && e.occupied(c.X, c.Y) == nil &&
				e.data.ValidCell(withState(e.Cell(c.X, c.Y), e.data.Rules.CritMarks[s.Element])) == "" {
				choice = c
				break
			}
		}
		require.NotEmpty(t, choice.Skill)
		assert.Empty(t, choice.Target)
		_, err := e.Preview(choice)
		require.NoError(t, err)
		require.NoError(t, e.useSkill(u, choice))
		assert.Equal(t, 100-s.Cost, u.MP)
		assert.Equal(t, e.data.Rules.CritMarks["fire"], string(e.Cell(choice.X, choice.Y).State))
		for _, tile := range e.state.TileStates {
			if tile.X == choice.X && tile.Y == choice.Y {
				assert.Equal(t, 2, tile.Turns)
			}
		}
	})

	t.Run("confusion duration and fan buff use distinct effects", func(t *testing.T) {
		e, u, v := arena(t)
		e.data.Traits["lasting"] = contentData.Traits["lasting"]
		with(e, u.ID, "lasting")
		confuse := contentData.Skills["confuse"]
		confuse.Sure = true
		e.spell(u, v, confuse, Command{Kind: "skill"})
		assert.Equal(t, 3, v.Status["confusion"].Turns)
		assert.Equal(t, 100.0, e.hitChance(u, v, e.basic()))
		assert.Equal(t, 100.0, e.hitChance(u, v, contentData.Skills["scorch"]))
		assert.Less(t, e.hitChance(u, v, contentData.Skills["great_confusion"]), 100.0)
		assert.Zero(t, e.hitChance(v, u, e.basic()))

		class := e.class(e.char(u.Character))
		class.Weapon = "fan"
		e.data.Classes[e.char(u.Character).Class] = class
		buff := contentData.Skills["rouse"]
		buff.Hit = 100
		e.spell(u, u, buff, Command{Kind: "skill"})
		assert.Positive(t, u.Status["mind"].Value)
		assert.Zero(t, u.Status["attack"].Value)
	})
}

func TestOfficerReformTraitsInBattle(t *testing.T) {
	data, err := content.Load("../../assets/data")
	require.NoError(t, err)

	t.Run("physical critical confuses and the first strike drains spell points", func(t *testing.T) {
		e, u, v := arena(t)
		for _, id := range []string{"liaolai", "deadly", "brocade_raider"} {
			e.data.Traits[id] = data.Traits[id]
			with(e, u.ID, id)
		}
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		u.MP = 0
		v.HP, v.MP = 10000, 15
		require.GreaterOrEqual(t, e.stats(v).MaxMP/2, 15)
		s := e.basic()
		s.Strikes = 2
		e.physical(u, v, s, Command{Kind: "skill"})
		assert.Equal(t, 1, v.Status["confusion"].Turns)
		assert.Zero(t, v.MP)
		assert.Equal(t, 7, u.MP)
	})

	t.Run("named guard reduces only its damage kind", func(t *testing.T) {
		e, u, v := arena(t)
		physical, magic := e.basic(), e.data.Skills["flame"]
		blow, spell := e.damage(u, v, physical), e.damage(u, v, magic)
		v.Status["physical_guard"] = Status{Value: 1, Turns: 3}
		assert.InDelta(t, blow*.8, e.damage(u, v, physical), 0.01)
		assert.InDelta(t, spell, e.damage(u, v, magic), 0.01)
		delete(v.Status, "physical_guard")
		v.Status["magic_guard"] = Status{Value: 1, Turns: 3}
		assert.InDelta(t, blow, e.damage(u, v, physical), 0.01)
		assert.InDelta(t, spell*.8, e.damage(u, v, magic), 0.01)
	})

	t.Run("spread is learned in Jingzhou and Pang Tong can join", func(t *testing.T) {
		e := New(data, 1)
		yiji := e.addCharacter("yiji", "yiji")
		assert.NotContains(t, e.learnable(yiji), "spread")
		e.state.Node = "c22-join-yiji"
		assert.Contains(t, e.learnable(yiji), "spread")
		e.once(data.Node("c27v-join-pangtong"))
		require.NotNil(t, e.char("pangtong"))
		assert.Contains(t, e.def(e.char("pangtong")).Learn, "chain")
	})
}
