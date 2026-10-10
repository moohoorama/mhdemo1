package core

import (
	"testing"

	"srpg/internal/content"
)

func TestActionsCompleteTheUnitsTurn(t *testing.T) {
	for _, kind := range []string{"attack", "skill", "item", "wait"} {
		for _, moved := range []bool{false, true} {
			name := kind + "/in_place"
			if moved {
				name = kind + "/after_move"
			}
			t.Run(name, func(t *testing.T) {
				e, hero, _ := arena(t)
				if moved {
					apply(t, e, Command{Kind: "move", Actor: hero.ID, X: 6, Y: 5})
				}
				c := Command{Kind: kind, Actor: hero.ID, Target: "boss"}
				if kind == "skill" {
					with(e, hero.ID, "stance_art")
					e.unit(hero.ID).MP = e.stats(e.unit(hero.ID)).MaxMP
					c.Skill, c.Target = "stance", hero.ID
				} else if kind == "item" {
					c.Item, c.Target = "tonic", hero.ID
				}
				apply(t, e, c)
				u := e.unit(hero.ID)
				if !u.Moved || !u.Acted || !u.Done {
					t.Fatalf("unfinished unit: %+v", u)
				}
				if actions := e.LegalActions(u.ID).Commands; len(actions) != 0 {
					t.Fatalf("finished unit has legal actions: %+v", actions)
				}
				for _, next := range []Command{
					{Kind: "move", Actor: u.ID, X: 4, Y: 6},
					{Kind: "attack", Actor: u.ID, Target: "boss"},
					{Kind: "skill", Actor: u.ID, Skill: "stance", Target: u.ID},
					{Kind: "item", Actor: u.ID, Item: "ration", Target: u.ID},
				} {
					assertRejected(t, e, next)
				}
				apply(t, e, Command{Kind: "end"})
				apply(t, e, Command{Kind: "end"})
				u = e.unit(hero.ID)
				if u.Done || u.Acted || u.Moved {
					t.Fatalf("unit did not regain its turn: %+v", u)
				}
				if len(e.LegalActions(u.ID).Commands) == 0 {
					t.Fatal("new turn has no legal actions")
				}
			})
		}
	}
}

func TestFailedActionKeepsMovementAndActionAvailable(t *testing.T) {
	e, hero, _ := arena(t)
	assertRejected(t, e, Command{Kind: "attack", Actor: hero.ID, Target: hero.ID})
	u := e.unit(hero.ID)
	if u.Done || u.Moved || u.Acted {
		t.Fatal("failed action spent the turn")
	}
	apply(t, e, Command{Kind: "move", Actor: hero.ID, X: 6, Y: 5})
	apply(t, e, Command{Kind: "attack", Actor: hero.ID, Target: "boss"})
}

func TestDuelCompletesTheAttackersTurn(t *testing.T) {
	e, hero, boss := arena(t)
	e.data.Stages[e.state.Stage].Duels = []content.Duel{{ID: "test", Ally: hero.ID, Enemy: boss.ID, Outcome: "draw"}}
	r := apply(t, e, Command{Kind: "attack", Actor: hero.ID, Target: boss.ID})
	duel := false
	for _, event := range r.Events {
		duel = duel || event.Kind == "duel"
	}
	u := e.unit(hero.ID)
	if !duel || !u.Done || !u.Moved || !u.Acted || !u.Attacked {
		t.Fatalf("duel did not finish the attack: %+v", u)
	}
}
