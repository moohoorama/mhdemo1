package core

import (
	"srpg/internal/content"
	"testing"
)

func TestIncomingStrikeAwardsExperience(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		spell, block, derived, lethal bool
	}{
		{name: "physical hit"}, {name: "successful defense", block: true},
		{name: "magic hit", spell: true}, {name: "counter hit", derived: true},
		{name: "fatal hit", lethal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, ally, enemy := arena(t)
			e.data.Rules.ExpDefenseShare = .5
			e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
			e.data.Rules.SpellFloor, e.data.Rules.SpellCeil = 100, 100
			skill := e.basic()
			if tc.spell {
				skill.Kind = content.KindMagic
				skill.Coeff = 1
			}
			if tc.block {
				with(e, "boss", "crit_t")
				with(e, "hero", "gold")
			}
			if tc.lethal {
				ally.HP = 1
			}
			before := ally.HP
			e.events = nil
			n, hit := e.strike(enemy, ally, skill, blow{share: 1, derived: tc.derived})
			if tc.block {
				if hit || n != 0 || ally.HP != before {
					t.Fatal("defense did not block")
				}
			} else if !hit || n <= 0 {
				t.Fatal("strike did not land")
			}
			gained := 0
			for _, ev := range e.events {
				if ev.Kind == "experience" && ev.Actor == ally.Character {
					if ev.Amount <= 0 {
						t.Fatal("no XP")
					}
					gained++
				}
			}
			if gained != 1 {
				t.Fatalf("wanted one XP award, got %d: %+v", gained, e.events)
			}
			if tc.lethal && ally.HP != 0 {
				t.Fatal("fatal blow did not retreat")
			}
		})
	}
}

func TestEnemyReceivesNoDefenseExperience(t *testing.T) {
	e, ally, enemy := arena(t)
	e.data.Rules.ExpDefenseShare = .5
	e.events = nil
	e.strike(ally, enemy, e.basic(), blow{share: 1})
	for _, ev := range e.events {
		if ev.Kind == "experience" && ev.Actor == enemy.Character {
			t.Fatal("enemy gained XP")
		}
	}
}
