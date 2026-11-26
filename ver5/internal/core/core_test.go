package core

import (
	"encoding/json"
	"math"
	"slices"
	"srpg/internal/content"
	"testing"
)

func data(t *testing.T) *content.Data {
	t.Helper()
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func apply(t *testing.T, e *Engine, c Command) Result {
	t.Helper()
	r, err := e.Apply(c)
	if err != nil {
		t.Fatalf("%+v: %v", c, err)
	}
	return r
}
func battle(t *testing.T) *Engine {
	e := New(data(t), 17)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	apply(t, e, Command{Kind: "start"})
	finishDeployment(t, e)
	return e
}
func encoded(v any) string { b, _ := json.Marshal(v); return string(b) }
func assertRejected(t *testing.T, e *Engine, c Command) {
	t.Helper()
	before := encoded(e.Snapshot())
	if _, err := e.Apply(c); err == nil {
		t.Fatalf("accepted %+v", c)
	}
	if before != encoded(e.Snapshot()) {
		t.Fatal("rejected command mutated state or RNG")
	}
}

// arena sets up the hero and the boss side by side on open ground, the rest of the enemy gone.
// A successful Apply replaces the engine's state, so tests fetch units again with e.unit after each one.
// Its tests bend the rules freely (a boss of 9999 HP, an undeployed sage), so Apply skips the restore check.
func arena(t *testing.T) (*Engine, *Unit, *Unit) {
	e := battle(t)
	e.unchecked = true
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.Faction == "enemy" && id != "boss" {
			u.HP = 0
		}
	}
	a, b := e.unit("hero"), e.unit("boss")
	a.X, a.Y, b.X, b.Y = 5, 6, 6, 6
	return e, a, b
}
func with(e *Engine, id string, traits ...string) {
	c := e.char(id)
	c.Learned = append(c.Learned, traits...)
}
func TestSetupAndStats(t *testing.T) {
	e := battle(t)
	if len(e.unitIDs) != 5 {
		t.Fatalf("units %v", keys(e.unitIDs))
	}
	for _, id := range []string{"troop#1", "troop#2"} {
		c := e.char(id)
		if c == nil || !c.Temp || c.Def != "troop" {
			t.Fatalf("%s: %+v", id, c)
		}
	}
	if e.char("troop#2").Level != 5 || e.char("troop#1").Level != 3 {
		t.Fatal("level override")
	}
	if e.unit("troop#1").Color != "red" || e.unit("hero").Color != "blue" || e.unit("hero").Items["ration"] != 0 || e.unit("boss").Items["ration"] != 1 {
		t.Fatal("color or stock")
	}
	u := e.unit("hero")
	s := e.stats(u)
	if u.HP != s.MaxHP || u.MP != 0 {
		t.Fatal("initial resources")
	}
	before := s
	e.char("hero").Level = 20
	r := e.stats(u)
	if r.MaxMP != before.MaxMP || r.Attack <= before.Attack {
		t.Fatal("MP must not scale with level")
	}
}
func TestTemplateIdsSurviveCheckpointAndRestore(t *testing.T) {
	e := battle(t)
	r, err := Restore(e.data, e.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if encoded(r.Snapshot()) != encoded(e.Snapshot()) {
		t.Fatal("restore differs")
	}
	apply(t, e, Command{Kind: "wait", Actor: "hero"})
	st := e.Snapshot()
	st.Checkpoint = nil
	if _, err := Restore(e.data, st); err == nil {
		t.Fatal("battle without checkpoint accepted")
	}
}
func TestLordLossAndRoutless(t *testing.T) {
	e, a, _ := arena(t)
	a.HP = 0
	apply(t, e, Command{Kind: "wait", Actor: "archer1"})
	if e.state.Phase != "result" || e.state.Result != "defeat" {
		t.Fatalf("%s %s", e.state.Phase, e.state.Result)
	}
	if err := e.Retry(); err != nil || e.state.Phase != "scenario" {
		t.Fatal("retry", err)
	}
}
func TestCampaignFlowAndJoin(t *testing.T) {
	e, _, b := arena(t)
	b.HP = 0
	apply(t, e, Command{Kind: "wait", Actor: "hero"})
	if e.state.Result != "victory" || e.state.Inventory["tonic"] != 3 {
		t.Fatalf("%s %v", e.state.Result, e.state.Inventory)
	}
	apply(t, e, Command{Kind: "continue"})
	o := e.Observe()
	if o.Dialogue.ID != "prep2" || o.Stage != 1 || e.char("sage") == nil || e.state.Warehouse["spear"] != 1 || len(o.Deployment) != 2 {
		t.Fatalf("%s %d %v", o.Dialogue.ID, o.Stage, o.Deployment)
	}
	for _, c := range e.Snapshot().Characters {
		if c.Temp {
			t.Fatal("troop survived the battle")
		}
	}
	assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"archer1"}}) // the lord must go
	apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero", "sage"}})
	apply(t, e, Command{Kind: "start"})
	finishDeployment(t, e)
	e.unit("boss").HP = 0
	apply(t, e, Command{Kind: "wait", Actor: "hero"})
	apply(t, e, Command{Kind: "continue"})
	if e.state.Phase != "complete" {
		t.Fatal(e.state.Phase)
	}
}
func TestEquipmentRules(t *testing.T) {
	e := New(data(t), 1)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	assertRejected(t, e, Command{Kind: "equip", Actor: "hero", Item: "short_bow"}) // not in the warehouse
	e.state.Warehouse["short_bow"] = 1
	assertRejected(t, e, Command{Kind: "equip", Actor: "hero", Item: "short_bow"}) // wrong weapon type
	e.state.Warehouse["spear"] = 1
	assertRejected(t, e, Command{Kind: "equip", Actor: "hero", Item: "spear"})
	with(e, "hero", "lancer") // wield:lance lets a blade class carry a lance, but not other types
	apply(t, e, Command{Kind: "equip", Actor: "hero", Item: "spear"})
	assertRejected(t, e, Command{Kind: "equip", Actor: "hero", Item: "short_bow"})
}
func TestMovementTraitsAndRiders(t *testing.T) {
	e, a, _ := arena(t)
	base := e.movement(a)
	with(e, "hero", "dash")
	if e.movement(a) != base+1 {
		t.Fatal("dash")
	}
	with(e, "hero", "gale")
	if e.movement(a) != base+3 {
		t.Fatal("dash + gale")
	}
	arch := e.unit("archer1")
	if e.unitClass(arch).Tags != nil && e.canMelee(arch) {
		t.Fatal("archer melee")
	}
	if !e.canRange(arch) {
		t.Fatal("archer cannot shoot")
	}
	lo, hi := e.AttackRange("archer1")
	if lo != 2 || hi != 3 {
		t.Fatalf("range %d-%d", lo, hi)
	}
	e.char("hero").Class = "rider"
	e.char("hero").Equipment = map[string]string{}
	with(e, "hero", "steed")
	if lo, hi = e.AttackRange("hero"); lo != 1 || hi != 2 || e.movement(a) != 5+3 {
		t.Fatalf("steed %d-%d move %d", lo, hi, e.movement(a))
	}
}
func TestForcedCriticalsAndImmunity(t *testing.T) {
	e, a, b := arena(t)
	if e.critChance(a, b, e.basic()) >= 100 {
		t.Fatal("crit should be random by default")
	}
	with(e, "hero", "crit_t")
	if e.critChance(a, b, e.basic()) != 100 {
		t.Fatal("crit_always")
	}
	with(e, "boss", "gold")
	hp := e.unit("boss").HP
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	for _, ev := range r.Events {
		if ev.Kind == "critical" {
			t.Fatal("critical landed through crit_immune")
		}
	}
	if e.unit("boss").HP != hp {
		t.Fatal("a blocked blow dealt damage")
	}
}
func TestCritAgainstTheWounded(t *testing.T) {
	e, hero, boss := arena(t)
	with(e, "hero", "weak_crit")
	if e.critForced(hero, boss, e.basic()) {
		t.Fatal("the boss is unhurt")
	}
	boss.HP = e.stats(boss).MaxHP / 2
	if !e.critForced(hero, boss, e.basic()) || !e.critForced(hero, boss, e.data.Skills["flame"]) {
		t.Fatal("at half HP every attack on it crits")
	}
}
func TestAmbushCrit(t *testing.T) {
	e, a, b := arena(t)
	with(e, "hero", "ambush_t")
	if e.critForced(a, b, e.basic()) {
		t.Fatal("not in a forest")
	}
	a.X, a.Y = 2, 1
	if !e.critForced(a, b, e.basic()) {
		t.Fatal("forest ambush")
	}
}
func TestSecondStrikeIsRolledBeforeTheFirstLands(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "twice")
	doubles := 0
	for i := 0; i < 30; i++ {
		r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
		for _, ev := range r.Events {
			if ev.Kind == "double" {
				doubles++
			}
		}
		h, b := e.unit("hero"), e.unit("boss")
		resetActions(h)
		h.HP = e.stats(h).MaxHP
		b.HP = e.stats(b).MaxHP
		e.state.Phase, e.state.Result, e.state.Learning = "battle", "", nil
	}
	if doubles < 25 {
		t.Fatalf("double_always gave %d/30 (a strike that kills ends it)", doubles)
	}
}
func TestSplashAndPierce(t *testing.T) {
	e, a, b := arena(t)
	t1, t2 := e.unit("troop#1"), e.unit("troop#2")
	t1.HP, t2.HP = e.stats(t1).MaxHP, e.stats(t2).MaxHP
	t1.X, t1.Y = 6, 5 // above the target
	t2.X, t2.Y = 7, 5 // diagonal
	with(e, "hero", "cross_t")
	hit, share, _ := e.victims(a, b, e.basic())
	if len(hit) != 2 || share["troop#1"] != .75 || share["troop#2"] != 0 {
		t.Fatalf("cross %v %v", hit, share)
	}
	with(e, "hero", "square_t")
	hit, share, _ = e.victims(a, b, e.basic())
	if len(hit) != 3 || share["troop#2"] != .75 {
		t.Fatalf("square %v %v", hit, share)
	}
	e.char("hero").Learned = nil
	t2.X, t2.Y = 7, 6 // behind the target
	e.char("hero").Class = "rider"
	e.char("hero").Equipment = map[string]string{"weapon": "spear"}
	hit, share, _ = e.victims(a, b, e.basic())
	if len(hit) != 2 || share["troop#2"] != .75 {
		t.Fatalf("pierce %v %v", hit, share)
	}
	a.X, a.Y = 5, 5 // a diagonal blow carries on diagonally
	t1.X, t1.Y = 7, 7
	hit, share, _ = e.victims(a, b, e.basic())
	if len(hit) != 2 || share["troop#1"] != .75 || share["troop#2"] != 0 {
		t.Fatalf("diagonal pierce %v %v", hit, share)
	}
}
func TestPreviewDamageIsEachTargetsOwn(t *testing.T) {
	e, a, b := arena(t)
	t2 := e.unit("troop#2")
	t2.HP, t2.X, t2.Y = e.stats(t2).MaxHP, 7, 6 // behind the target
	e.char("hero").Class = "rider"
	e.char("hero").Equipment = map[string]string{"weapon": "spear"}
	c := Command{Kind: "attack", Actor: a.ID, Target: b.ID}
	p, err := e.Preview(c)
	if err != nil || len(p.Targets) != 2 {
		t.Fatalf("pierce %+v %v", p, err)
	}
	d := e.damage(a, b, e.basic())
	if p.Damage[b.ID] != int(math.Floor(d)) || p.CritDamage[b.ID] != int(math.Floor(d*e.data.Rules.CritMultiplier)) {
		t.Fatalf("target's own damage %v %v, raw %.1f", p.Damage, p.CritDamage, d)
	}
	if p.CritDamage[t2.ID] >= p.CritDamage[b.ID] || p.MaxDamage != p.CritDamage[b.ID]+p.CritDamage[t2.ID] {
		t.Fatalf("the one behind takes a share of the total: %v, max %d", p.CritDamage, p.MaxDamage)
	}
	with(e, "boss", "bulwark_t")
	p, _ = e.Preview(c)
	limit, crit := float64(e.stats(b).MaxHP)*.1, d*e.data.Rules.CritMultiplier
	if crit > limit && p.CritDamage[b.ID] != int(math.Floor(limit+(crit-limit)/2)) {
		t.Fatalf("bulwark: %d, raw %.1f, limit %.1f", p.CritDamage[b.ID], crit, limit)
	}
}
func answered(r Result, from, to string) bool {
	for _, ev := range r.Events {
		if (ev.Kind == "damage" || ev.Kind == "miss") && ev.Actor == from && ev.Target == to {
			return true
		}
	}
	return false
}
func TestEveryUnitCountersAndStancesCounterHarder(t *testing.T) {
	c := Command{Kind: "attack", Actor: "hero", Target: "boss"}
	e, _, _ := arena(t)
	if r := apply(t, e, c); !answered(r, "boss", "hero") {
		t.Fatal("a unit with no stance strikes back")
	}
	e2, _, b2 := arena(t)
	plain := e2.CounterDamage(c)
	b2.Status["counter"] = Status{Turns: 2}
	if stance := e2.CounterDamage(c); plain == 0 || plain >= stance {
		t.Fatalf("counter previews should grow plain %d < stance %d", plain, stance)
	}
}
func TestRetaliateAnswersFromAnyDistance(t *testing.T) {
	setup := func(t *testing.T) *Engine {
		e, _, _ := arena(t)
		e.char("hero").Class = "archer"
		e.char("hero").Equipment = map[string]string{}
		e.unit("hero").X = 3
		return e
	}
	c := Command{Kind: "attack", Actor: "hero", Target: "boss"}
	e := setup(t)
	if e.CounterDamage(c) != 0 {
		t.Fatal("the boss's sword does not reach the archer")
	}
	with(e, "boss", "counter_t")
	if e.CounterDamage(c) == 0 {
		t.Fatal("응징 strikes back beyond its reach")
	}
	r := apply(t, e, c)
	if !answered(r, "boss", "hero") {
		t.Fatal("응징 answered no shot")
	}
	e2 := setup(t)
	with(e2, "boss", "counter_t")
	sage(e2, 4, 6)
	if r := apply(t, e2, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"}); answered(r, "boss", "sage") {
		t.Fatal("응징 never answers a spell")
	}
}
func TestAClassWithoutReachNeitherStrikesNorStrikesBack(t *testing.T) {
	e, _, _ := arena(t)
	cl := e.data.Classes["guard"]
	cl.Range = 0
	e.data.Classes["guard"] = cl
	e.char("boss").Class = "guard"
	if r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"}); answered(r, "boss", "hero") {
		t.Fatal("a unit without reach struck back")
	}
	for _, c := range e.LegalActions("boss").Commands {
		if c.Kind == "attack" {
			t.Fatalf("a unit without reach may attack: %+v", c)
		}
	}
}
func TestGuardianAnswersForANeighbour(t *testing.T) {
	e, _, _ := arena(t) // hero (5,6) strikes the boss (6,6)
	t1 := e.unit("troop#1")
	t1.HP, t1.X, t1.Y = e.stats(t1).MaxHP, 5, 5 // beside the boss, and its sword reaches the hero
	with(e, "troop#1", "riposte_t")
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	if !answered(r, "troop#1", "hero") {
		t.Fatal("호위 answers a melee blow on its neighbour")
	}
	e2, _, _ := arena(t)
	t1 = e2.unit("troop#1")
	t1.HP, t1.X, t1.Y = e2.stats(t1).MaxHP, 7, 7 // beside the boss, the hero out of its reach
	with(e2, "troop#1", "riposte_t")
	if r := apply(t, e2, Command{Kind: "attack", Actor: "hero", Target: "boss"}); answered(r, "troop#1", "hero") {
		t.Fatal("호위 needs the attacker in its reach")
	}
}
func TestBulwarkHalvesTheExcess(t *testing.T) {
	e, a, b := arena(t)
	with(e, "boss", "bulwark_t")
	limit := float64(e.stats(b).MaxHP) * .1
	raw := e.damage(a, b, e.basic())
	if raw <= limit {
		t.Skip("fixture damage below the threshold")
	}
	hp := b.HP
	n, hit, _ := e.strike(a, b, e.basic(), blow{share: 1})
	if !hit {
		t.Skip("missed")
	}
	if float64(n) > limit+(raw*1.5-limit)/2+1 {
		t.Fatalf("bulwark let %d through (raw %.0f, limit %.0f, hp %d)", n, raw, limit, hp)
	}
}
func TestSpellFamiliesNeedTheirTraits(t *testing.T) {
	e := battle(t)
	if got := e.skills(e.char("hero")); len(got) != 0 {
		t.Fatalf("hero skills %v", got)
	}
	addSage(e)
	if got := e.skills(e.char("sage")); len(got) < 8 {
		t.Fatalf("sage skills %v", got)
	}
	if contains(e.skills(e.char("sage")), "inferno") {
		t.Fatal("inferno needs fire_up")
	}
	with(e, "sage", "fire_up")
	if !contains(e.skills(e.char("sage")), "inferno") {
		t.Fatal("inferno should open")
	}
}

// addSage puts the sage in the party, having learned the traits its definition plans.
func addSage(e *Engine) {
	e.addCharacter("sage", "sage")
	with(e, "sage", e.data.Characters["sage"].Traits...)
}
func sage(e *Engine, x, y int) {
	addSage(e)
	u := e.spawn("sage", "sage", "ally", x, y, content.Spawn{})
	u.MP = 500
}
func TestSpellEffects(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	hp := e.unit("boss").HP
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	if e.unit("boss").HP >= hp {
		t.Fatal("flame did nothing")
	}
	resetActions(e.unit("sage"))
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "weaken"})
	if v := e.unit("boss").Status["attack"].Value; v != -20 {
		t.Fatalf("debuff %d", v)
	}
	resetActions(e.unit("sage"))
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "bolster"})
	if v := e.unit("hero").Status["attack"].Value; v != 20 {
		t.Fatalf("buff %d", v)
	}
	resetActions(e.unit("sage"))
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "seal"})
	resetActions(e.unit("sage"))
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "disarm"})
	b := e.unit("boss")
	if b.Status["seal"].Turns != 2 || b.Status["disarm"].Turns != 2 {
		t.Fatalf("statuses %v", b.Status)
	}
}
func TestDisarmAndSealBlockActions(t *testing.T) {
	e, _, _ := arena(t)
	e.unit("hero").Status["disarm"] = Status{Turns: 2}
	assertRejected(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	e2, _, _ := arena(t)
	sage(e2, 4, 6)
	e2.unit("sage").Status["seal"] = Status{Turns: 2}
	assertRejected(t, e2, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
}
func TestReflectReturnsTheSpell(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	with(e, "boss", "reflect_t")
	hp := e.unit("sage").HP
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	if e.unit("sage").HP >= hp {
		t.Fatal("reflect should hit the caster")
	}
}
func TestDispelBlocksEnemySpells(t *testing.T) {
	e, _, b := arena(t)
	sage(e, 4, 6)
	b.Status = map[string]Status{}
	u := e.unit("sage")
	if e.hitChance(u, b, e.data.Skills["flame"]) == 0 {
		t.Fatal("the boss has no 간파 yet")
	}
	with(e, "boss", "see")
	e.char("sage").Level = 50
	if e.hitChance(u, b, e.data.Skills["flame"]) != 0 {
		t.Fatal("간파 stops every enemy attack spell, whatever the caster's mind")
	}
}
func TestHealingBonusItemsAndShare(t *testing.T) {
	e, _, _ := arena(t)
	e.unit("hero").HP = 10
	apply(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	if hp := e.unit("hero").HP; hp != 230 {
		t.Fatalf("ration healed to %d", hp)
	}
	e2, _, _ := arena(t)
	with(e2, "hero", "heal_t")
	e2.unit("hero").HP = 10
	apply(t, e2, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	if hp := e2.unit("hero").HP; hp != 230 {
		t.Fatalf("medicine is for heal spells, yet the ration healed to %d", hp)
	}
	e4, _, _ := arena(t)
	e4.data.Traits["power_t"] = content.Trait{Effects: map[string]float64{content.FxItemPower: .5}}
	with(e4, "hero", "power_t")
	e4.unit("hero").HP = 10
	apply(t, e4, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	if hp := e4.unit("hero").HP; hp != 340 {
		t.Fatalf("item_power 0.5 makes the ration heal 330, yet it healed to %d", hp)
	}
	e3, _, _ := arena(t)
	with(e3, "hero", "share")
	e3.unit("archer1").X, e3.unit("archer1").Y = 5, 5
	e3.unit("archer1").HP, e3.unit("hero").HP = 10, 10
	apply(t, e3, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	if e3.unit("archer1").HP != 230 {
		t.Fatal("item_share reaches an adjacent ally")
	}
}
func TestItemCompletesTheAction(t *testing.T) {
	e, _, _ := arena(t)
	apply(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "tonic"})
	assertRejected(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	assertRejected(t, e, Command{Kind: "move", Actor: "hero", X: 4, Y: 6})
}
func TestRegenAuraAndTurnBuffs(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "regen_t", "haste_t", "sight_t", "fort")
	e.unit("hero").HP = 100
	e.endFaction()
	e.endFaction()
	a := e.unit("hero")
	if a.HP <= 100 || a.Status["speed"].Value != 2 || a.Status["defense"].Value != 20 {
		t.Fatalf("hp %d %+v", a.HP, a.Status)
	}
	if e.movement(a) != e.unitClass(a).Move+2 {
		t.Fatal("haste movement")
	}
}
func TestLastStandOnce(t *testing.T) {
	e, a, _ := arena(t)
	with(e, "hero", "last")
	e.hurt(a, 1<<20, e.unit("boss"))
	if a.HP != 1 || !a.Spared {
		t.Fatal("last stand")
	}
	e.hurt(a, 1<<20, e.unit("boss"))
	if a.HP != 0 {
		t.Fatal("only once")
	}
}
func TestBloodCostPaysMissingMP(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	e.unit("sage").MP = 0
	assertRejected(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	with(e, "sage", "blood")
	hp := e.unit("sage").HP
	apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	if u := e.unit("sage"); u.HP != hp-300 || u.MP != 0 {
		t.Fatalf("blood cost: hp %d→%d mp %d", hp, u.HP, u.MP)
	}
}
func TestDeputyBlend(t *testing.T) {
	e, _, _ := arena(t)
	c := e.char("hero")
	c.Class = "guard"
	c.Deputy = "archer1"
	if attr := e.attributes(c); attr[0] != 91 {
		t.Fatalf("deputy lower: %v", attr)
	}
	d := e.data.Characters["archer1"]
	d.Stats = [6]int{100, 100, 100, 100, 100, 100}
	e.data.Characters["archer1"] = d
	if a := e.attributes(c); a[0] <= 91 || a[0] >= 100 {
		t.Fatalf("blend %v", a)
	}
	with(e, "hero", "lead") // 용인술: the deputy's attributes count 10 higher
	R, own := e.data.Rules, e.def(c).Stats
	n := (float64(own[5])*float64(R.DeputyWeight) + 110) / float64(R.DeputyDivisor)
	if a := e.attributes(c); math.Abs(a[0]-(n*110+(1-n)*float64(own[0]))) > 1e-9 {
		t.Fatalf("용인술 blend %v, n %.3f", a, n)
	}
}
func TestExperienceBarIsLevelIndependent(t *testing.T) {
	e, _, _ := arena(t)
	c := e.char("hero")
	c.XP = 3000
	level := c.Level
	e.xp("hero", XPPerLevel)
	if c.Level != level+1 || c.XP != 3000 {
		t.Fatalf("one bar of xp: level %d->%d, xp %d", level, c.Level, c.XP)
	}
}
func TestLearningWindow(t *testing.T) {
	e, _, _ := arena(t)
	e.char("hero").Points = 250
	e.xp("hero", XPPerLevel)
	if len(e.state.Learning) != 1 {
		t.Fatalf("window %v (points %d)", e.state.Learning, e.char("hero").Points)
	}
	assertRejected(t, e, Command{Kind: "wait", Actor: "hero"})
	assertRejected(t, e, Command{Kind: "learn", Actor: "hero", Trait: "crit_t"}) // costs 400, has 350
	e.char("hero").Points = 1000
	assertRejected(t, e, Command{Kind: "learn", Actor: "hero", Trait: "gale"}) // not in the pool
	assertRejected(t, e, Command{Kind: "learn", Actor: "hero", Trait: "square_t"})
	apply(t, e, Command{Kind: "learn", Actor: "hero", Trait: "crit_t"})
	c := e.char("hero")
	if !contains(e.traits(c), "crit_t") || c.Points != 600 {
		t.Fatalf("learned %v points %d", c.Learned, c.Points)
	}
	apply(t, e, Command{Kind: "learn_close", Actor: "hero"})
	if len(e.state.Learning) != 0 {
		t.Fatal("close")
	}
	apply(t, e, Command{Kind: "wait", Actor: "hero"})
}
func TestPrerequisitesGateLearning(t *testing.T) {
	e, _, _ := arena(t)
	c := e.char("hero")
	if contains(e.learnable(c), "gale") {
		t.Fatal("gale is not in any pool")
	}
	d := e.data.Characters["hero"]
	d.Learn = append(d.Learn, "square_t")
	e.data.Characters["hero"] = d
	if contains(e.learnable(c), "square_t") {
		t.Fatal("square needs cross")
	}
	c.Learned = append(c.Learned, "cross_t")
	if !contains(e.learnable(c), "square_t") {
		t.Fatal("square should open")
	}
}
func TestRejectedCommandsKeepState(t *testing.T) {
	e := battle(t)
	assertRejected(t, e, Command{Kind: "move", Actor: "boss", X: 8, Y: 2})
	assertRejected(t, e, Command{Kind: "move", Actor: "hero", X: 11, Y: 7})
	assertRejected(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	assertRejected(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "flame"})
	assertRejected(t, e, Command{Kind: "wat"})
}
func TestSaveRoundTripThroughJSON(t *testing.T) {
	e := battle(t)
	apply(t, e, Command{Kind: "move", Actor: "hero", X: 2, Y: 3})
	b, _ := json.Marshal(e.Snapshot())
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	r, err := Restore(e.data, s)
	if err != nil {
		t.Fatal(err)
	}
	if encoded(r.Snapshot()) != encoded(e.Snapshot()) {
		t.Fatal("round trip")
	}
}

func resetActions(u *Unit) {
	u.Moved, u.Acted, u.Attacked, u.Done = false, false, false, false
}

func TestDamageIsAttackMinusDefensePlusLevelTerm(t *testing.T) {
	e, a, b := arena(t)
	R := e.data.Rules
	level := float64(e.char(a.Character).Level + R.StatLevelBase)
	sa, sb := e.stats(a), e.stats(b)
	want := float64(sa.Attack)*e.terrain(a, a.X, a.Y).Factor - float64(sb.Defense)*e.terrain(b, b.X, b.Y).Factor + level*R.PhysicalLevelScale
	if got := e.damage(a, b, e.basic()); got != want {
		t.Fatalf("physical %v, want %v", got, want)
	}
	spell := content.Skill{Kind: content.KindMagic, Coeff: 1.5}
	want = (float64(sa.Mind-sb.Mind) + level*R.MagicLevelScale) * 1.5
	if got := e.damage(a, b, spell); got != want {
		t.Fatalf("spell %v, want %v", got, want)
	}
}
func TestManaFollowsCommandAndIntellect(t *testing.T) {
	e := battle(t)
	u := e.unit("hero")
	R := e.data.Rules
	a := e.attributes(e.char("hero"))
	c := e.class(e.char("hero"))
	st := e.stats(u)
	if want := int(float64(c.MPBase) + R.ManaMaxFactor*R.StatNumerator/(R.StatDenominator-a[0])); st.MaxMP != want {
		t.Fatalf("max mp %d, want %d", st.MaxMP, want)
	}
	if want := int(float64(c.RecoveryBase) + R.ManaRecoveryFactor*R.StatNumerator/(R.StatDenominator-a[2])); st.Recovery != want {
		t.Fatalf("recovery %d, want %d", st.Recovery, want)
	}
}
func TestBlendBonusRaisesTheLowerClassBonus(t *testing.T) {
	e := battle(t)
	u := e.unit("hero")
	c := e.class(e.char("hero"))
	before := e.stats(u)
	with(e, "hero", "blend_t")
	after := e.stats(e.unit("hero"))
	switch {
	case c.Bonus[0] > c.Bonus[2]:
		if after.Mind <= before.Mind || after.Attack != before.Attack {
			t.Fatalf("mind should rise alone: %+v -> %+v", before, after)
		}
	case c.Bonus[2] > c.Bonus[0]:
		if after.Attack <= before.Attack || after.Mind != before.Mind {
			t.Fatalf("attack should rise alone: %+v -> %+v", before, after)
		}
	}
}

func TestDeputyNeedsTheAppointTrait(t *testing.T) {
	e, _, _ := arena(t)
	sage(e, 4, 6)
	cmd := Command{Kind: "deputy", Actor: "hero", Target: "sage"}
	if err := e.assignDeputy(cmd); err == nil {
		t.Fatal("a deputy was named without the trait")
	}
	with(e, "hero", "deputy_t")
	if err := e.assignDeputy(cmd); err != nil {
		t.Fatal(err)
	}
	if e.char("hero").Deputy != "sage" {
		t.Fatalf("deputy %q", e.char("hero").Deputy)
	}
	e.char("hero").Class = "guard"
	if e.fxOf(e.char("hero"), content.FxDeputyAppoint) == 0 {
		t.Fatal("the trait is permanent across class changes")
	}
}

func TestSelfSkillsHealAndGuardTheCaster(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "aid_t", "brace_t")
	h := e.unit("hero")
	h.HP, h.MP = e.stats(h).MaxHP/2, 100
	before, mp := h.HP, h.MP
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "first_aid"})
	if h = e.unit("hero"); h.HP <= before || h.MP >= mp {
		t.Fatalf("first aid: hp %d→%d mp %d→%d", before, h.HP, mp, h.MP)
	}
	resetActions(h)
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "brace"})
	if s := e.unit("hero").Status["defense"]; s.Turns != 2 || s.Value != e.data.Rules.BuffPercent {
		t.Fatalf("brace status %+v", s)
	}
	assertRejected(t, e, Command{Kind: "skill", Actor: "hero", Target: "boss", Skill: "brace"})
}

func TestKnockbackPushesOrHitsHarder(t *testing.T) {
	setup := func(t *testing.T) (*Engine, *Unit, *Unit) {
		e, a, b := arena(t)
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		a.HP, b.HP = 10000, 10000
		with(e, "hero", "boost_t")
		return e, a, b
	}
	e, _, _ := setup(t)
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	if b := e.unit("boss"); b.X != 7 || b.Y != 6 {
		t.Fatalf("the boss should slide a cell back, at (%d,%d)", b.X, b.Y)
	}
	pushed := false
	for _, ev := range r.Events {
		pushed = pushed || ev.Kind == "knockback" && ev.Target == "boss" && len(ev.Path) == 2
	}
	if !pushed {
		t.Fatal("no knockback event")
	}
	e2, a2, b2 := setup(t)
	t2 := e2.unit("troop#2")
	t2.HP, t2.X, t2.Y = e2.stats(t2).MaxHP, 7, 6 // behind the boss: it cannot be pushed
	free := e2.damage(a2, b2, e2.basic())
	n, _, _ := e2.strike(a2, b2, e2.basic(), blow{share: 1})
	e3, a3, b3 := setup(t)
	e3.unit("troop#2").HP, e3.unit("troop#2").X, e3.unit("troop#2").Y = 1000, 7, 6
	dealt := e3.attackOn(a3, b3, []*Unit{b3}, e3.basic(), map[string]float64{b3.ID: 1}, false, true, false)
	if b3.X != 6 || float64(dealt[b3.ID]) < math.Floor(free*1.25)-1 || n == 0 {
		t.Fatalf("blocked: stays at %d, deals %d (free blow %.0f)", b3.X, dealt[b3.ID], free)
	}
}
func TestKnockbackOnCounter(t *testing.T) {
	e, a, b := arena(t)
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	a.HP, b.HP = 10000, 10000
	with(e, "boss", "boost_t")
	x, y := a.X, a.Y
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	pushed := false
	for _, ev := range r.Events {
		pushed = pushed || ev.Kind == "knockback" && ev.Actor == "boss" && ev.Target == "hero"
	}
	if h := e.unit("hero"); !pushed || h.X == x && h.Y == y {
		t.Fatalf("the boss's counter should push the hero: event %t, at (%d,%d)", pushed, h.X, h.Y)
	}
}
func TestSkillNeedsItsWeapon(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "thrust_t")
	assertRejected(t, e, Command{Kind: "skill", Actor: "hero", Target: "boss", Skill: "thrust_s"})
	e.char("hero").Equipment = map[string]string{"weapon": "spear"}
	e.unit("hero").MP = 50
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "boss", Skill: "thrust_s"})
}
func TestStrikesForcesTheSecondBlow(t *testing.T) {
	e, _, _ := arena(t)
	e.char("hero").Class = "archer"
	e.char("hero").Equipment = map[string]string{}
	with(e, "hero", "rapid_t")
	e.unit("hero").X, e.unit("hero").MP = 3, 50
	r := apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "boss", Skill: "rapid_s"})
	for _, ev := range r.Events {
		if ev.Kind == "double" {
			return
		}
	}
	t.Fatal("no second blow")
}
func TestMagicCoefficientAddsSpellDamage(t *testing.T) {
	e, a, b := arena(t)
	R := e.data.Rules
	skill := e.data.Skills["arrow_s"]
	plain := skill
	plain.MagicCoeff = 0
	sa, sb := e.stats(a), e.stats(b)
	level := float64(e.char(a.Character).Level + R.StatLevelBase)
	want := e.damage(a, b, plain) + (float64(sa.Mind-sb.Mind)+level*R.MagicLevelScale)*.7
	if got := e.damage(a, b, skill); math.Abs(got-want) > 1e-9 {
		t.Fatalf("damage %v, want %v", got, want)
	}
}

func TestSpellWardedByTargetsIntelligence(t *testing.T) {
	e, a, b := arena(t)
	fire := content.Skill{Kind: content.KindMagic, Coeff: 1, Element: "fire"}
	e.data.Rules.MagicWardSpan = 0
	plain := e.damage(a, b, fire)
	wits := e.attributes(e.char(b.Character))[2]
	e.data.Rules.MagicWardFrom, e.data.Rules.MagicWardSpan = wits-30, 150
	if got := e.damage(a, b, fire); math.Abs(got-plain*0.8) > 1e-9 {
		t.Fatalf("30 points of 지력 above the floor take a fifth off: %v, want %v", got, plain*0.8)
	}
	e.data.Rules.MagicWardFrom = wits + 1
	if got := e.damage(a, b, fire); math.Abs(got-plain) > 1e-9 {
		t.Fatalf("지력 under the floor takes nothing off: %v, want %v", got, plain)
	}
	strike := content.Skill{Kind: content.KindPhysical, Coeff: 1}
	e.data.Rules.MagicWardFrom = wits - 30
	ward := e.damage(a, b, strike)
	e.data.Rules.MagicWardSpan = 0
	if got := e.damage(a, b, strike); got != ward {
		t.Fatalf("physical blows are not warded: %v, want %v", ward, got)
	}
}

func TestSpellGainsOnItsElementsTerrain(t *testing.T) {
	e, a, b := arena(t)
	fire := content.Skill{Kind: content.KindMagic, Coeff: 1, Element: "fire"}
	land := func(s content.Skill, crit bool) float64 { return e.damage(a, b, s) * e.boost(a, b, s, crit) }
	plain := land(fire, false)
	ground := string(e.Cell(b.X, b.Y).Ground)
	e.data.Rules.ElementTerrain = map[string]map[string]int{"fire": {ground: 30}, "water": {ground: 15}}
	if got := land(fire, false); math.Abs(got-plain*1.3) > 1e-9 {
		t.Fatalf("fire on its terrain: %v, want %v", got, plain*1.3)
	}
	if got, want := land(fire, true), plain*(1.3+e.data.Rules.CritMultiplier-1); math.Abs(got-want) > 1e-9 {
		t.Fatalf("a critical on its terrain adds the two bonuses: %v, want %v", got, want)
	}
	e.data.Rules.ElementTerrain = map[string]map[string]int{"water": {ground: 15}}
	if got := land(fire, false); got != plain {
		t.Fatalf("another element's terrain: %v, want %v", got, plain)
	}
	e.data.Rules.ElementTerrain = map[string]map[string]int{"fire": {ground: 30}}
	e.data.Rules.CritMarks, e.data.Rules.CritMarkPower = map[string]string{"fire": "e", "water": "w"}, 45
	e.state.TileStates = append(e.state.TileStates, TileState{X: b.X, Y: b.Y, State: "w"})
	water := content.Skill{Kind: content.KindMagic, Coeff: 1, Element: "water"}
	if got := land(water, false); math.Abs(got-plain*1.45) > 1e-9 {
		t.Fatalf("water on its own mark: %v, want %v", got, plain*1.45)
	}
	if got := land(fire, false); math.Abs(got-plain*1.3) > 1e-9 {
		t.Fatalf("fire on water's mark keeps its ground's bonus: %v, want %v", got, plain*1.3)
	}
	e.state.TileStates = nil
	strike := content.Skill{Kind: content.KindPhysical, Coeff: 1, Element: "fire"}
	before := land(strike, false)
	e.data.Rules.ElementTerrain = map[string]map[string]int{"fire": {ground: 30}}
	if got := land(strike, false); got != before {
		t.Fatalf("a strike with an element: %v, want %v", got, before)
	}
}

func TestGuardSkillSetsItsStatus(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "guard_t")
	e.unit("hero").MP = 100
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "guard_s"})
	if g := e.unit("hero").Status["guard"]; g.Turns != 2 {
		t.Fatalf("guard status %+v", g)
	}
}
func TestGuardBlocksMeleeWhollyAndIsSpent(t *testing.T) {
	e, a, b := arena(t)
	b.Status["guard"] = Status{Turns: 2}
	if e.hitChance(a, b, e.basic()) != 0 {
		t.Fatal("a guarded unit must not be hit in melee")
	}
	with(e, "hero", "twice")
	hp := b.HP
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	for _, ev := range r.Events {
		if ev.Kind == "damage" && ev.Target == "boss" {
			t.Fatal("a guarded unit took damage")
		}
	}
	if b = e.unit("boss"); b.HP != hp || b.Status["guard"].Turns != 0 {
		t.Fatalf("hp %d→%d guard %+v", hp, b.HP, b.Status["guard"])
	}
	if e.hitChance(e.unit("hero"), b, e.basic()) == 0 {
		t.Fatal("the guard is spent")
	}
}
func TestGuardDoesNotStopRangedAttacks(t *testing.T) {
	e, a, b := arena(t)
	e.char("hero").Class = "archer"
	e.char("hero").Equipment = map[string]string{}
	a.X = 3
	b.Status["guard"] = Status{Turns: 2}
	if e.hitChance(a, b, e.basic()) == 0 {
		t.Fatal("guard stops melee only")
	}
}
func TestAimBuffsItselfAndSnipeShoots(t *testing.T) {
	archer := func(t *testing.T) (*Engine, *Unit, *Unit) {
		e, a, b := arena(t)
		e.char("hero").Class = "archer"
		e.char("hero").Equipment = map[string]string{}
		with(e, "hero", "aim_t", "snipe_t")
		a.X, a.MP = 3, 100
		return e, a, b
	}
	e, a, b := archer(t)
	R := e.data.Rules
	agility, morale := e.stats(a).Agility, e.stats(a).Morale
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "aim_s"})
	h := e.unit("hero")
	if st := e.stats(h); st.Agility != percent(agility, R.BuffPercent) || st.Morale != percent(morale, R.BuffPercent) || h.Status["agility"].Turns != 3 {
		t.Fatalf("aim must raise its own agility and morale for 3 turns: %+v %+v", st, h.Status)
	}

	e, a, b = archer(t)
	_, hi := e.rangeFor(a, e.basic())
	if _, now := e.rangeFor(a, e.data.Skills["snipe_s"]); now != hi+2 {
		t.Fatalf("snipe range %d -> %d", hi, now)
	}
	a.X = b.X - (hi + 2)
	if e.inRange(a, b, e.basic()) || !e.inRange(a, b, e.data.Skills["snipe_s"]) {
		t.Fatal("snipe reaches two cells beyond the basic attack")
	}
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "boss", Skill: "snipe_s"})
}
func TestProvokeLowersMoraleAroundTheCaster(t *testing.T) {
	e, _, _ := arena(t)
	e.data.Rules.SpellFloor, e.data.Rules.SpellCeil = 100, 100
	with(e, "hero", "provoke_t")
	e.unit("hero").MP = 100
	t1 := e.unit("troop#1")
	t1.HP, t1.X, t1.Y = e.stats(t1).MaxHP, 4, 5 // diagonal to the hero: outside the cross
	apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "provoke_s"})
	if s := e.unit("boss").Status["morale"]; s.Value != -e.data.Rules.BuffPercent {
		t.Fatalf("boss morale %+v", s)
	}
	if s := e.unit("troop#1").Status["morale"]; s.Turns != 0 {
		t.Fatalf("a diagonal foe is outside the cross: %+v", s)
	}
}
func TestDisruptCutsAgilityAndMorale(t *testing.T) {
	e, _, b := arena(t)
	R := e.data.Rules
	before := e.stats(b)
	b.Status["disrupt"] = Status{Turns: 2, Value: -R.BuffPercent}
	after := e.stats(b)
	if after.Agility != percent(before.Agility, -R.BuffPercent) || after.Morale != percent(before.Morale, -R.BuffPercent) || after.Attack != before.Attack {
		t.Fatalf("disrupt stats %+v -> %+v", before, after)
	}
	delete(b.Status, "disrupt")
	with(e, "hero", "disrupt_t")
	e.unit("hero").MP = 50
	for i := 0; i < 30 && e.unit("boss").Status["disrupt"].Turns == 0; i++ {
		resetActions(e.unit("hero"))
		apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "boss", Skill: "disrupt_s"})
	}
	if s := e.unit("boss").Status["disrupt"]; s.Turns != 2 || s.Value != -R.BuffPercent {
		t.Fatalf("disrupt status %+v", s)
	}
}
func TestAccuracyAddsToThePhysicalHitChance(t *testing.T) {
	e, a, b := arena(t)
	e.data.Traits["sharp"] = content.Trait{Name: "sharp", Kind: "talent", Tone: "attack", Effects: map[string]float64{content.FxAccuracy: 10}}
	before := e.hitChance(a, b, e.basic())
	with(e, "hero", "sharp")
	after := e.hitChance(e.unit("hero"), b, e.basic())
	if before < e.data.Rules.HitCeil-10 && after != before+10 {
		t.Fatalf("hit chance %.1f → %.1f", before, after)
	}
}
func TestSideHitsNeverPassTheRulesShare(t *testing.T) {
	e, a, b := arena(t)
	e.data.Rules.SideShareMax = 0.5
	e.data.Traits["deep"] = content.Trait{Name: "deep", Kind: "talent", Tone: "attack", Effects: map[string]float64{content.FxPierce: 0.9}}
	with(e, "hero", "deep")
	behind := *b
	behind.ID, behind.X, behind.Y = "behind", b.X+(b.X-a.X), b.Y+(b.Y-a.Y)
	e.unitIDs[behind.ID] = e.units.NewEntity(&behind)
	_, share, _ := e.victims(e.unit("hero"), b, e.basic())
	if share["behind"] != 0.5 || share[b.ID] != 1 {
		t.Fatalf("shares %v", share)
	}
}

func TestRestoreChangedFitsThePartyToTheData(t *testing.T) {
	e := battle(t)
	s := e.Snapshot()
	h := &s.Characters[0]
	for i := range s.Characters {
		if s.Characters[i].ID == "hero" {
			h = &s.Characters[i]
		}
	}
	h.Learned = append(h.Learned, "dropped_trait")
	h.Equipment["weapon"] = "dropped_blade"
	r, err := RestoreChanged(e.data, s)
	if err != nil {
		t.Fatal(err)
	}
	c := r.char("hero")
	if slices.Contains(c.Learned, "dropped_trait") || c.Equipment["weapon"] != "" {
		t.Fatalf("hero keeps what the data dropped: %v %v", c.Learned, c.Equipment)
	}
}

func TestRestoreChangedFindsTheBattleByNodeAfterStageOrderChanges(t *testing.T) {
	e := battle(t)
	s := e.Snapshot()
	original := s.Stage
	s.Stage++
	if s.Checkpoint == nil {
		t.Fatal("battle has no preparation checkpoint")
	}
	s.Checkpoint.Stage++
	loaded, err := RestoreChanged(e.data, s)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Snapshot()
	if got.Stage != original || got.Checkpoint.Stage != original {
		t.Fatalf("restored stage %d and checkpoint %d, want %d", got.Stage, got.Checkpoint.Stage, original)
	}
}
