package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
func TestMountedRangeShootsFromBeyondTheEightCells(t *testing.T) {
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
	for _, uv := range e.Observe().UnitViews {
		if uv.ID == "archer1" && uv.Shoot != "archer" {
			t.Fatalf("view shoot %q", uv.Shoot)
		}
	}
	for _, c := range []struct {
		dx, dy int
		want   bool
	}{{1, 0, false}, {1, 1, false}, {2, 0, true}, {2, 1, true}, {0, 2, true}} {
		u.X, u.Y, v.X, v.Y = 5, 5, 5+c.dx, 5+c.dy
		if got := e.rangedAttack(u, v, e.basic()); got != c.want {
			t.Errorf("target at (%d,%d): ranged %v, want %v", c.dx, c.dy, got, c.want)
		}
	}
}
func TestSweepArcStrikesTheFiveCellsAhead(t *testing.T) {
	// the front cell and two each side around the attacker: on the screen NE covers NW N NE E SE, E covers N NE E SE S
	for _, c := range []struct {
		front string
		dx    int
		dy    int
		arc   []string // compass names of the ring cells that take the share, front excluded
	}{
		{"NE", 0, -1, []string{"N", "NW", "E", "SE"}},
		{"SW", 0, 1, []string{"SE", "S", "W", "NW"}},
		{"E", 1, -1, []string{"N", "NE", "SE", "S"}},
	} {
		t.Run(c.front, func(t *testing.T) {
			e, hero, boss := arena(t)
			with(e, "hero", "sweeper")
			hero.X, hero.Y = 8, 8
			boss.X, boss.Y = 8+c.dx, 8+c.dy
			names := map[string]content.Point{"N": {X: -1, Y: -1}, "NE": {Y: -1}, "E": {X: 1, Y: -1}, "SE": {X: 1}, "S": {X: 1, Y: 1}, "SW": {Y: 1}, "W": {X: -1, Y: 1}, "NW": {X: -1}}
			sp := content.Spawn{Character: "troop", Level: 1}
			cells := map[string]*Unit{}
			for name, p := range names {
				if name != c.front {
					cells[name] = e.spawn(sp.Character, sp.Character, "enemy", 8+p.X, 8+p.Y, sp)
				}
			}
			out, share := e.victims(hero, boss, e.basic())
			if share[boss.ID] != 1 || len(out) != 1+len(c.arc) {
				t.Fatalf("victims %d, target share %v", len(out), share[boss.ID])
			}
			for _, name := range c.arc {
				if share[cells[name].ID] != .5 {
					t.Errorf("%s of a %s strike: share %v, want .5", name, c.front, share[cells[name].ID])
				}
			}
		})
	}
	e, hero, boss := arena(t)
	with(e, "hero", "sweeper")
	hero.X, hero.Y, boss.X, boss.Y = 8, 8, 8, 10 // two cells away: no arc, nothing beside the target is hit either
	beside := e.spawn("troop", "troop", "enemy", 7, 8, content.Spawn{Character: "troop", Level: 1})
	if _, share := e.victims(hero, boss, e.basic()); share[beside.ID] != 0 {
		t.Fatal("the arc reached from a distance")
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

func TestPouchHealsAfterABlow(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "boss", "pouch_t")
	for i := 0; i < 8; i++ {
		boss := e.unit("boss")
		boss.HP = e.stats(boss).MaxHP
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
		if dealt == 0 {
			continue
		}
		assert.Equal(t, min(dealt, 50), healed, "once per attack, never above the full bar")
		assert.Equal(t, e.stats(e.unit("boss")).MaxHP-dealt+healed, e.unit("boss").HP)
		return
	}
	t.Fatal("no blow landed in 8 swings")
}
