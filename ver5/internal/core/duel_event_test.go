package core

import (
	"srpg/internal/content"
	"testing"
)

func TestStructuredDuelEvent(t *testing.T) {
	for _, enemyAttacks := range []bool{false, true} {
		t.Run(map[bool]string{false: "ally attacks", true: "enemy attacks"}[enemyAttacks], func(t *testing.T) {
			e, hero, boss := arena(t)
			d := content.Duel{ID: "one-exchange", Ally: hero.Character, Enemy: boss.Character, Outcome: "victory", AllyResult: "stay", EnemyResult: "death", Text: "one exchange"}
			e.events = nil
			if enemyAttacks {
				e.duel(boss, hero, d)
			} else {
				e.duel(hero, boss, d)
			}
			event := e.events[0]
			if event.Kind != "duel" || event.DuelID != d.ID || event.Outcome != d.Outcome || event.Actor != hero.ID || event.Target != boss.ID {
				t.Fatalf("wrong structured event: %+v", event)
			}
			if len(e.events) < 2 || e.events[1].Kind != "death" || e.events[1].Actor != boss.ID {
				t.Fatalf("death must follow presentation: %+v", e.events)
			}
			if boss.HP != 0 || !e.char(boss.Character).Dead {
				t.Fatal("presentation changed death result")
			}
		})
	}
}
