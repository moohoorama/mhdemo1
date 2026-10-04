package core

import (
	"reflect"
	"srpg/internal/content"
	"testing"
)

func TestGrowthCostsOverflowAndDisplay(t *testing.T) {
	e := battle(t)
	if ExperienceRequired(1) != 100 || ExperienceRequired(2) != 104 || ExperienceRequired(50) != 540 {
		t.Fatal("cost table")
	}
	e.xp("유비", 250)
	o := e.officer("유비")
	if o.Level != 3 || o.XP != 46 || o.Points != 200 {
		t.Fatal(o)
	}
	if ExperienceProgress(o.Level, o.XP) <= 0 || ExperienceProgress(o.Level, o.XP) >= 100 {
		t.Fatal("display")
	}
	if _, err := Restore(e.data, e.Snapshot()); err != nil {
		t.Fatal(err)
	}
}
func TestActionExperienceAndNoDefenseExperience(t *testing.T) {
	e := battle(t)
	u, v := e.unit("유비"), e.unit("B01_troop_1")
	e.actionXP(u, v, 12, false, false)
	a := e.officer("유비").XP
	e.officer("유비").XP = 0
	e.actionXP(u, v, 12, true, false)
	b := e.officer("유비").XP
	if a != 14 || b != 28 {
		t.Fatal(a, b)
	}
	e.officer("유비").XP = 0
	e.actionXP(u, v, 12, false, true)
	if e.officer("유비").XP != b {
		t.Fatal("finish multiplier")
	}
	e.officer("유비").XP = 0
	e.hurt(u, 10, v)
	if e.officer("유비").XP != 0 {
		t.Fatal("defense XP")
	}
	// Tiny hits, repeated hits and restored HP never diminish the award.
	for i := 0; i < 5; i++ {
		e.hurt(v, 1, u)
		v.HP = e.stats(v).MaxHP
		e.actionXP(u, v, 12, false, false)
	}
	if e.officer("유비").XP != 5*a {
		t.Fatal("grinding denied")
	}
}
func TestPromotionDeputyAndFractionReplay(t *testing.T) {
	e := New(data(t), 1)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "결의"})
	e.officer("유비").Level = 15
	apply(t, e, Command{Kind: "start"})
	e.state.Inventory["승급:중보병"] = 1
	e.unit("유비").HP = 1
	e.unit("유비").MP = 0
	apply(t, e, Command{Kind: "item", Actor: "유비", Target: "유비", Item: "승급:중보병"})
	u := e.unit("유비")
	if u.HP != e.stats(u).MaxHP || u.MP != e.stats(u).MaxMP || !u.Acted {
		t.Fatal("promotion")
	}
	// Return to a preparation fixture without adding early campaign supplies.
	s := e.Snapshot()
	s.Phase = "scenario"
	s.Units = nil
	s.Checkpoint = nil
	e = fromState(e.data, s)
	apply(t, e, Command{Kind: "deputy", Actor: "유비", Target: "관우"})
	if len(e.officer("관우").Equipment) != 0 || contains(e.state.Deployment, "관우") {
		t.Fatal("deputy equipment/deploy")
	}
	assertRejected(t, e, Command{Kind: "deputy", Actor: "관우", Target: "장비"})
	assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"유비", "관우"}})
	for i := 0; i < 7; i++ {
		e.xp("유비", 1)
	}
	if e.officer("관우").XP != 5 || e.officer("관우").XPShareRemainder != 95 {
		t.Fatal("fraction")
	}
	r, err := Restore(e.data, e.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 13; i++ {
		e.xp("유비", 1)
		r.xp("유비", 1)
	}
	if !reflect.DeepEqual(e.Snapshot(), r.Snapshot()) || e.officer("관우").XP != 17 {
		t.Fatal("share replay")
	}
	before := e.attributes(e.officer("유비"))
	if before[1] <= float64(e.data.Officers["유비"].Stats[1]) {
		t.Fatal("deputy stats")
	}
}
func TestDuelLedgerDeathAndRetry(t *testing.T) {
	for _, result := range []string{"stay", "retreat", "death"} {
		e := battle(t)
		e.data = copyOf(e.data)
		e.data.Hash = e.state.ContentHash
		e.data.Stages[0].Duels = []content.Duel{{ID: "fixture", Ally: "관우", Enemy: "등무", Outcome: "draw", AllyResult: result, EnemyResult: "stay"}}
		u, v := e.unit("관우"), e.unit("등무")
		u.X = 4
		u.Y = 4
		v.X = 5
		v.Y = 4
		r, err := Restore(e.data, e.Snapshot())
		if err != nil {
			t.Fatal(err)
		}
		c := Command{Kind: "attack", Actor: "관우", Target: "등무"}
		if !e.DuelFor(c) {
			t.Fatal("adjacent attack does not set off the duel")
		}
		a := apply(t, e, c)
		b := apply(t, r, c)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(e.Snapshot(), r.Snapshot()) {
			t.Fatal("duel replay")
		}
		if result == "stay" && e.officer("관우").Level != 2 {
			t.Fatal("duel XP")
		}
		if result != "stay" && e.officer("관우").Level != 1 {
			t.Fatal("retreat/death XP")
		}
		if e.officer("관우").Dead != (result == "death") {
			t.Fatal("death")
		}
		if _, err := Restore(e.data, e.Snapshot()); err != nil {
			t.Fatal("after duel", err)
		}
		if e.DuelFor(c) {
			t.Fatal("duel set off twice")
		}
		e.unit("유비").HP = 0
		e.settle()
		if err := e.Retry(); err != nil {
			t.Fatal(err)
		}
		if e.officer("관우").Dead || e.state.Executed["duel-fixture"] {
			t.Fatal("retry retained death/event")
		}
	}
}

func TestEnemyHealingAndSpellAwards(t *testing.T) {
	e := battle(t)
	u := e.unit("유비")
	v := e.unit("B01_troop_1")
	u.X, u.Y = 4, 5
	v.X, v.Y = 5, 5
	v.HP = 1
	before := e.officer("유비").XP
	apply(t, e, Command{Kind: "item", Actor: "유비", Target: v.ID, Item: "소군량"})
	if e.unit(v.ID).HP <= 1 || e.officer("유비").XP != before {
		t.Fatal("enemy healing item")
	}
	e.unit("유비").Acted = false
	e.officer("유비").Learned = append(e.officer("유비").Learned, "계략")
	// Teach a healing spell through a diagnostic class instead of supplying it
	// to classes that do not possess that capability.
	e.officer("유비").Class = "사마"
	e.unit("유비").MP = 60
	e.unit(v.ID).HP = 1
	apply(t, e, Command{Kind: "skill", Actor: "유비", Target: v.ID, Skill: "소회복"})
	if e.unit(v.ID).HP <= 1 || e.officer("유비").XP <= before {
		t.Fatal("enemy healing spell XP")
	}
}
func TestBuffWeakOrderAndPromotionLevels(t *testing.T) {
	e := battle(t)
	u := e.unit("유비")
	u.Status["attack"] = Status{Turns: 3, Value: 70}
	u.Status["weak"] = Status{Turns: 3, Value: 1000}
	for i := 0; i < 100; i++ {
		if e.stats(u).Attack != 1 {
			t.Fatal("unordered status effects")
		}
	}
	e.officer("유비").Level = 14
	e.state.Inventory["승급:중보병"] = 1
	assertRejected(t, e, Command{Kind: "item", Actor: "유비", Target: "유비", Item: "승급:중보병"})
	e.officer("유비").Level = 15
	apply(t, e, Command{Kind: "item", Actor: "유비", Target: "유비", Item: "승급:중보병"})
	e.unit("유비").Acted = false
	e.state.Inventory["승급:근위대"] = 1
	assertRejected(t, e, Command{Kind: "item", Actor: "유비", Target: "유비", Item: "승급:근위대"})
	e.officer("유비").Level = 30
	apply(t, e, Command{Kind: "item", Actor: "유비", Target: "유비", Item: "승급:근위대"})
	if e.officer("유비").Class != "근위대" {
		t.Fatal("advanced")
	}
}
