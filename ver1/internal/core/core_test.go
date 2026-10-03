package core

import (
	"encoding/json"
	"reflect"
	"srpg/internal/content"
	"testing"
)

func data(t *testing.T) *content.Data {
	t.Helper()
	d, err := content.Load("../../assets/content/campaign.json")
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
	apply(t, e, Command{Kind: "choose", Option: "결의"})
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
func TestFormulaAndResources(t *testing.T) {
	e := battle(t)
	u := e.unit("유비")
	s := e.stats(u)
	if s.Attack != 178 || s.Defense != 201 || s.Mind != 141 || s.MaxHP != 948 || s.MaxMP != 65 || s.Recovery != 8 {
		t.Fatalf("stats: %+v", s)
	}
	if u.HP != s.MaxHP || u.MP != 32 {
		t.Fatal("initial resources")
	}
	e.officer("유비").Level = 20
	r := e.stats(u)
	if r.MaxMP != s.MaxMP || r.Recovery != s.Recovery || r.Attack <= s.Attack {
		t.Fatal("MP must not scale with level")
	}
}
func TestAtomicRejectionAndCopies(t *testing.T) {
	e := battle(t)
	for _, c := range []Command{{Kind: "move", Actor: "유비", X: -1}, {Kind: "attack", Actor: "유비", Target: "장각"}, {Kind: "attack", Actor: "장각", Target: "유비"}, {Kind: "equip", Actor: "유비", Item: "item_024"}, {Kind: "skill", Actor: "유비", Skill: "초열", Target: "장각"}, {Kind: "learn", Actor: "유비", Trait: "속공"}, {Kind: "item", Actor: "유비", Item: "missing", Target: "유비"}, {Kind: "unknown"}} {
		assertRejected(t, e, c)
	}
	o := e.Observe()
	o.Inventory["소군량"] = 0
	o.UnitViews[0].Status["poison"] = Status{Turns: 3, Value: 0}
	o.Officers[0].Equipment["weapon"] = "bad"
	if !reflect.DeepEqual(e.Observe().Inventory, e.Snapshot().Inventory) || e.Snapshot().Inventory["소군량"] == 0 {
		t.Fatal("mutable observation")
	}
	snap := e.Snapshot()
	snap.Checkpoint.Inventory["소군량"] = 0
	if e.Snapshot().Checkpoint.Inventory["소군량"] == 0 {
		t.Fatal("mutable checkpoint")
	}
}
func TestActionBudgetsAndPermissions(t *testing.T) {
	e := battle(t)
	p := e.moves(e.unit("유비"))[0]
	apply(t, e, Command{Kind: "move", Actor: "유비", X: p.X, Y: p.Y})
	assertRejected(t, e, Command{Kind: "move", Actor: "유비", X: 2, Y: 5})
	apply(t, e, Command{Kind: "skill", Actor: "유비", Skill: "반격"})
	assertRejected(t, e, Command{Kind: "item", Actor: "유비", Item: "소군량", Target: "유비"})
	e = New(data(t), 1)
	assertRejected(t, e, Command{Kind: "item", Actor: "유비", Item: "소군량", Target: "유비"})
	assertRejected(t, e, Command{Kind: "learn", Actor: "유비", Trait: "속공"})
}
func TestLearningAndFreeActions(t *testing.T) {
	e := battle(t)
	u := e.unit("유비")
	o := e.officer("유비")
	o.Points = 400
	apply(t, e, Command{Kind: "learn", Actor: "유비", Trait: "속공"})
	if o == e.officer("유비") {
		t.Fatal("transaction retained component pointer")
	}
	o = e.officer("유비")
	if o.Points != 300 || !e.has(e.unit("유비"), "속공") {
		t.Fatal("learning")
	}
	assertRejected(t, e, Command{Kind: "learn", Actor: "유비", Trait: "속공"})
	apply(t, e, Command{Kind: "skill", Actor: "유비", Skill: "신속"})
	assertRejected(t, e, Command{Kind: "learn", Actor: "유비", Trait: "반격술"})
	u = e.unit("유비")
	if !u.Acted || u.Status["speed"].Turns != 1 {
		t.Fatal("skill action")
	}
	o = e.officer("유비")
	o.Learned = append(o.Learned, "국사무쌍", "부호")
	u.Acted = false
	apply(t, e, Command{Kind: "skill", Actor: "유비", Skill: "신속"})
	apply(t, e, Command{Kind: "skill", Actor: "유비", Skill: "신속"})
	u = e.unit("유비")
	if u.Acted || u.Status["speed"].Turns != 1 {
		t.Fatal("buff stacked or spent")
	}
	n := e.state.Inventory["소군량"]
	apply(t, e, Command{Kind: "item", Actor: "유비", Item: "소군량", Target: "유비"})
	if e.unit("유비").Acted || e.state.Inventory["소군량"] != n-1 {
		t.Fatal("free item generated stock")
	}
	apply(t, e, Command{Kind: "end"})
	apply(t, e, Command{Kind: "end"})
	if e.unit("유비").Status["speed"].Turns != 0 {
		t.Fatal("turn buff did not expire")
	}
}
func TestTraitsDedupAndNoBasicCounter(t *testing.T) {
	e := battle(t)
	u := e.unit("장비")
	e.officer("장비").Equipment["aux"] = "item_024"
	count := 0
	for _, v := range e.traits(u, false) {
		if v == "위압" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("duplicate trait")
	}
	target := e.unit("등무")
	target.X = 3
	target.Y = 7
	hp := u.HP
	apply(t, e, Command{Kind: "attack", Actor: "장비", Target: "등무"})
	if e.unit("장비").HP != hp {
		t.Fatal("default counter")
	}
	assertRejected(t, e, Command{Kind: "attack", Actor: "장비", Target: "등무"})
}
func TestCounterDamageAndExpiration(t *testing.T) {
	e := battle(t)
	e.unit("등무").X = 3
	e.unit("등무").Y = 5
	apply(t, e, Command{Kind: "skill", Actor: "유비", Skill: "반격"})
	apply(t, e, Command{Kind: "end"})
	hp := e.unit("등무").HP
	r := apply(t, e, Command{Kind: "attack", Actor: "등무", Target: "유비"})
	damage := 0
	for _, ev := range r.Events {
		if ev.Kind == "damage" && ev.Actor == "유비" {
			damage += ev.Amount
		}
	}
	if e.unit("등무").HP != hp-damage {
		t.Fatal("counter damage")
	}
	apply(t, e, Command{Kind: "end"})
	if e.unit("유비").Status["counter"].Turns != 0 {
		t.Fatal("counter persists after next own start")
	}
}
func TestTerrainZOCAndCharge(t *testing.T) {
	e := battle(t)
	u := e.unit("관우")
	if e.terrain(u, 0, 0).Factor != 1.1 {
		t.Fatal("cavalry grass")
	}
	u.X = 4
	u.Y = 4
	e.data.Stages[0].Tiles[3] = "~~~~~~~~~~~~~~~~~~"
	e.data.Stages[0].Tiles[5] = "~~~~~.~~~~~~~~~~~~"
	e.unit("등무").X = 5
	e.unit("등무").Y = 5
	if containsPoint(e.moves(u), content.Point{X: 7, Y: 4}) {
		t.Fatal("walked through ZOC")
	}
	u.Status["charge"] = Status{Turns: 1, Value: 0}
	if !containsPoint(e.moves(u), content.Point{X: 7, Y: 4}) {
		t.Fatal("charge does not bypass ZOC")
	}
	e.data.Stages[0].Tiles[4] = "................~."
	if containsPoint(e.moves(u), content.Point{X: 16, Y: 4}) {
		t.Fatal("water traversable")
	}
}
func TestReplayAtScenarioMoveEnemyAndResult(t *testing.T) {
	e := New(data(t), 5)
	for _, c := range []Command{{Kind: "next"}, {Kind: "choose", Option: "결의"}, {Kind: "start"}, {Kind: "move", Actor: "유비", X: 3, Y: 5}, {Kind: "end"}, {Kind: "wait", Actor: "등무"}} {
		restored, err := Restore(e.data, e.Snapshot())
		if err != nil {
			t.Fatal(err)
		}
		a := apply(t, e, c)
		b := apply(t, restored, c)
		if !reflect.DeepEqual(a, b) || encoded(e.Snapshot()) != encoded(restored.Snapshot()) {
			t.Fatal("replay mismatch")
		}
	}
}
func TestRewardsRetryAndCompatibility(t *testing.T) {
	e := battle(t)
	cp := e.Snapshot().Checkpoint
	e.unit("유비").HP = 0
	e.settle()
	if err := e.Retry(); err != nil {
		t.Fatal(err)
	}
	s := e.Snapshot()
	s.Revision = cp.Revision
	if encoded(s) != encoded(cp) {
		t.Fatal("retry not exact checkpoint")
	}
	apply(t, e, Command{Kind: "start"})
	e.unit("장각").HP = 0
	apply(t, e, Command{Kind: "wait", Actor: "유비"})
	xp := e.officer("유비").Level
	restored, err := Restore(e.data, e.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	apply(t, restored, Command{Kind: "continue"})
	if restored.officer("간옹") == nil || restored.officer("유비").Level != xp {
		t.Fatal("join or reward duplicated")
	}
	assertRejected(t, restored, Command{Kind: "continue"})
	s = restored.Snapshot()
	s.Core = "future"
	if _, err := Restore(e.data, s); err == nil {
		t.Fatal("version accepted")
	}
	s = restored.Snapshot()
	s.Officers[0].Level = -1
	if _, err := Restore(e.data, s); err == nil {
		t.Fatal("corrupt officer accepted")
	}
}

func TestPoisonKillCreditAndCalm(t *testing.T) {
	e := battle(t)
	e.unit("B01_troop_1").HP = 1
	e.unit("B01_troop_1").Status["poison"] = Status{Turns: 3, Source: "유비"}
	apply(t, e, Command{Kind: "end"})
	if e.officer("유비").XP != 56 || e.unit("B01_troop_1").HP != 0 {
		t.Fatal("poison kill has no credit")
	}
	u := e.unit("유비")
	u.Status["poison"] = Status{Turns: 3, Source: "장각"}
	u.Status["attack"] = Status{Turns: 3, Value: 30}
	e.officer("유비").Learned = append(e.officer("유비").Learned, "침착")
	hp := u.HP
	apply(t, e, Command{Kind: "end"})
	u = e.unit("유비")
	if u.Status["poison"].Turns != 0 || u.Status["attack"].Turns != 3 || u.HP != hp {
		t.Fatal("calm clears positive buffs or poison dealt damage")
	}
}
