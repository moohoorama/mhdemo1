package core

import (
	"strings"
	"testing"
)

func TestNobodyLeavesTheMap(t *testing.T) {
	e, hero, _ := arena(t)
	e.data.Stages[e.state.Stage].Boats = true // the edge cell is water: boats could sail onto it
	hero.X, hero.Y = 0, 0
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {e.data.Stages[e.state.Stage].Width, 0}} {
		if c := e.StepCost(hero.ID, p[0], p[1]); c != 0 {
			t.Fatalf("stepping to (%d,%d) costs %d", p[0], p[1], c)
		}
	}
	for _, p := range e.moves(hero) {
		if p.X < 0 || p.Y < 0 {
			t.Fatalf("can move to %+v", p)
		}
	}
	hero.X = -1
	if _, err := e.Apply(Command{Kind: "end"}); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("a command left a unit off the map: %v", err)
	}
}
