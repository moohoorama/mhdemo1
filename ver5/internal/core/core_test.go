package core

import (
	"encoding/json"
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
func arena(t *testing.T) (*Engine, *Unit, *Unit) {
	e := battle(t)
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
	if o.Dialogue.ID != "prep2" || o.Stage != 1 || e.char("sage") == nil || e.state.Warehouse["spear"] != 1 || len(o.Deployment) != 3 {
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
	with(e, "archer1", "far_t")
	if lo, _ = e.AttackRange("archer1"); lo != 1 {
		t.Fatal("no_min_range")
	}
	e.char("hero").Class = "rider"
	e.char("hero").Equipment = map[string]string{}
	with(e, "hero", "steed")
	if lo, hi = e.AttackRange("hero"); lo != 1 || hi != 2 || e.movement(a) != 5+1+3 {
		t.Fatalf("steed %d-%d move %d", lo, hi, e.movement(a))
	}
}
func TestForcedCriticalsAndImmunity(t *testing.T) {
	e, a, b := arena(t)
	if e.critChance(a, b) >= 100 {
		t.Fatal("crit should be random by default")
	}
	with(e, "hero", "crit_t")
	if e.critChance(a, b) != 100 {
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
func TestCritAgainstWeakerMight(t *testing.T) {
	e, hero, boss := arena(t) // might: hero 76, boss 80
	with(e, "hero", "weak_crit")
	if e.critForced(hero, boss, e.basic()) {
		t.Fatal("the hero is weaker than the boss")
	}
	with(e, "boss", "weak_crit")
	if !e.critForced(boss, hero, e.basic()) {
		t.Fatal("the boss out-mights the hero")
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
		e.state.Learning = nil
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
	hit, share := e.victims(a, b, e.basic())
	if len(hit) != 2 || share["troop#1"] != .75 || share["troop#2"] != 0 {
		t.Fatalf("cross %v %v", hit, share)
	}
	with(e, "hero", "square_t")
	hit, share = e.victims(a, b, e.basic())
	if len(hit) != 3 || share["troop#2"] != .75 {
		t.Fatalf("square %v %v", hit, share)
	}
	e.char("hero").Learned = nil
	t2.X, t2.Y = 7, 6 // behind the target
	e.char("hero").Class = "rider"
	e.char("hero").Equipment = map[string]string{"weapon": "spear"}
	hit, share = e.victims(a, b, e.basic())
	if len(hit) != 2 || share["troop#2"] != .75 {
		t.Fatalf("pierce %v %v", hit, share)
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
func TestCounterStanceAndForcedCounter(t *testing.T) {
	e, _, _ := arena(t)
	if r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"}); answered(r, "boss", "hero") {
		t.Fatal("no stance, no counter")
	}
	e2, _, b2 := arena(t)
	b2.Status["counter"] = Status{Turns: 2}
	if e2.CounterDamage(Command{Kind: "attack", Actor: "hero", Target: "boss"}) == 0 {
		t.Fatal("counter damage preview")
	}
	if r := apply(t, e2, Command{Kind: "attack", Actor: "hero", Target: "boss"}); !answered(r, "boss", "hero") {
		t.Fatal("stance counter")
	}
	// 투신-like: the boss out-mights the hero, so a hero attack is answered with no stance at all
	e3, _, _ := arena(t)
	with(e3, "boss", "counter_t")
	if r := apply(t, e3, Command{Kind: "attack", Actor: "hero", Target: "boss"}); !answered(r, "boss", "hero") {
		t.Fatal("forced counter against a weaker attacker")
	}
}
func TestRiposteAnswersEveryMiss(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "boss", "riposte_t", "shield")
	e.char("hero").Class = "archer"
	e.char("hero").Equipment = map[string]string{}
	e.unit("hero").X = 3
	for i := 0; i < 4; i++ {
		resetActions(e.unit("hero"))
		r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
		if !answered(r, "boss", "hero") {
			t.Fatal("ranged_immune makes every shot miss and the riposte answers")
		}
		e.unit("boss").HP = e.stats(e.unit("boss")).MaxHP
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
	n, hit := e.strike(a, b, e.basic(), blow{share: 1})
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
	e.addCharacter("sage", "sage")
	if got := e.skills(e.char("sage")); len(got) < 8 {
		t.Fatalf("sage skills %v", got)
	}
	if contains(e.skills(e.char("sage")), "inferno") {
		t.Fatal("inferno needs fire_up")
	}
	e.char("sage").Learned = []string{"fire_up"}
	if !contains(e.skills(e.char("sage")), "inferno") {
		t.Fatal("inferno should open")
	}
}
func sage(e *Engine, x, y int) {
	e.addCharacter("sage", "sage")
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
func TestDispelBlocksLowerMind(t *testing.T) {
	e, _, b := arena(t)
	sage(e, 4, 6)
	with(e, "boss", "see")
	b.Status = map[string]Status{}
	u := e.unit("sage")
	if e.stats(b).Mind > e.stats(u).Mind {
		t.Skip("fixture: the boss out-minds the sage")
	}
	if e.hitChance(u, b, e.data.Skills["flame"]) == 0 {
		t.Fatal("dispel only stops lower-mind casters")
	}
	e.char("sage").Level = 1
	b.HP = e.stats(b).MaxHP
	d := e.data.Characters["boss"]
	d.Stats[2] = 100
	e.data.Characters["boss"] = d
	if e.stats(b).Mind > e.stats(u).Mind && e.hitChance(u, b, e.data.Skills["flame"]) != 0 {
		t.Fatal("dispel")
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
	if hp := e2.unit("hero").HP; hp != 450 {
		t.Fatalf("ration with heal_bonus healed to %d", hp)
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
func TestPromotionItem(t *testing.T) {
	e, _, _ := arena(t)
	assertRejected(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "promo_guard"}) // level 1
	e.char("hero").Level = 15
	apply(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "promo_guard"})
	if e.char("hero").Class != "guard" || e.state.Inventory["promo_guard"] != 0 {
		t.Fatal("promotion")
	}
	if u := e.unit("hero"); !u.Done || !u.Moved || !u.Acted {
		t.Fatal("promotion did not complete the turn")
	}
}
func TestRegenAuraAndTurnBuffs(t *testing.T) {
	e, _, _ := arena(t)
	with(e, "hero", "regen_t", "haste_t", "sight_t", "fort")
	e.unit("hero").HP = 100
	e.endFaction()
	e.endFaction()
	a := e.unit("hero")
	if a.HP <= 100 || a.Status["speed"].Value != 2 || a.Status["range"].Value != 1 {
		t.Fatalf("hp %d %+v", a.HP, a.Status)
	}
	a.X, a.Y = 8, 1 // the keep
	e.state.Turn = "enemy"
	e.endFaction()
	if a.Status["defense"].Value != 20 {
		t.Fatalf("fortify on the keep: %+v", a.Status)
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
	with(e, "hero", "lead")
	if a := e.attributes(c); int(a[0]) != 110 {
		t.Fatalf("deputy_rate %v", a)
	}
}
func TestLearningWindow(t *testing.T) {
	e, _, _ := arena(t)
	e.char("hero").Points = 250
	e.xp("hero", ExperienceRequired(e.data, e.char("hero").Level))
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
