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
			if len(amounts) != 2 || amounts[hero.ID] != 3 || amounts[ally.ID] != 0 {
				t.Fatalf("recovery amounts %v", amounts)
			}
		})
	}
}
