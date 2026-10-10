package core

import "testing"

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
func TestEightfoldWidensSingleSpells(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	t1 := e.unit("troop#1")
	t1.HP, t1.X, t1.Y = e.stats(t1).MaxHP, 6, 5 // beside the boss
	flame := e.data.Skills["flame"]
	if n := len(e.targets(e.unit("sage"), e.unit("boss"), flame)); n != 1 {
		t.Fatalf("single spell hit %d", n)
	}
	e.unit("sage").Status["eightfold"] = Status{Turns: 2}
	if n := len(e.targets(e.unit("sage"), e.unit("boss"), flame)); n != 2 {
		t.Fatalf("eightfold spell hit %d", n)
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
func TestChainRepeatsOnANeighbour(t *testing.T) {
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
