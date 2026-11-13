package core

import (
	"fmt"
	"testing"
)

func TestRefillEventReportsActualRecovery(t *testing.T) {
	for _, missing := range []int{0, 3, 50} {
		t.Run(fmt.Sprintf("missing_%d", missing), func(t *testing.T) {
			e, hero, _ := arena(t)
			sage(e, 4, 6)
			s := e.data.Skills["bolster"]
			s.Effect, s.Coeff = "refill", 24
			e.data.Skills["bolster"] = s
			hero.MP = e.stats(hero).MaxMP - missing
			before := hero.MP
			r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "bolster"})
			hero = e.unit("hero")
			for _, event := range r.Events {
				if event.Kind == "effect" && event.Target == "hero" {
					if event.Amount != hero.MP-before || event.Amount != min(24, missing) {
						t.Fatalf("event %+v, MP change %d", event, hero.MP-before)
					}
					return
				}
			}
			t.Fatal("no recovery effect event")
		})
	}
}

func TestItemEventsReportEachTargetsActualRecovery(t *testing.T) {
	for _, item := range []string{"ration", "tonic"} {
		t.Run(item, func(t *testing.T) {
			e, hero, _ := arena(t)
			with(e, "hero", "share")
			ally := e.unit("archer1")
			ally.X, ally.Y = 5, 5
			hero.HP, hero.MP = e.stats(hero).MaxHP-3, e.stats(hero).MaxMP-3
			ally.HP, ally.MP = e.stats(ally).MaxHP, e.stats(ally).MaxMP
			r := apply(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: item})
			amounts := map[string]int{}
			for _, event := range r.Events {
				if event.Kind == "item" {
					amounts[event.Target] = event.Amount
				}
			}
			want := map[string]int{hero.ID: 3, ally.ID: 0}
			if item == "tonic" { // a drink pours past the brim
				want = map[string]int{hero.ID: 15, ally.ID: 15}
			}
			if len(amounts) != 2 || amounts[hero.ID] != want[hero.ID] || amounts[ally.ID] != want[ally.ID] {
				t.Fatalf("recovery amounts %v, want %v", amounts, want)
			}
		})
	}
}

func TestDrinkOverfillsMPAndTheTurnKeepsIt(t *testing.T) {
	e, hero, _ := arena(t)
	max := e.stats(hero).MaxMP
	hero.MP = max - 5
	apply(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "tonic"})
	hero = e.unit("hero")
	if hero.MP != max+10 {
		t.Fatalf("MP %d after the tonic, want %d", hero.MP, max+10)
	}
	e.endFaction()
	for e.state.Turn != "ally" {
		e.endFaction()
	}
	if hero = e.unit("hero"); hero.MP != max+10 {
		t.Fatalf("MP %d after the turn's recovery, want %d kept", hero.MP, max+10)
	}
	if _, err := Restore(e.data, e.Snapshot()); err != nil {
		t.Fatalf("a save with MP over the max is refused: %v", err)
	}
}
