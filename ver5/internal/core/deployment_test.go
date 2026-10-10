package core

import (
	"srpg/internal/content"
	"testing"
)

func deployment(t *testing.T) *Engine {
	t.Helper()
	e := New(data(t), 17)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	apply(t, e, Command{Kind: "start"})
	return e
}
func finishDeployment(t *testing.T, e *Engine) {
	t.Helper()
	commands, err := e.DeploymentPlan()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range commands {
		apply(t, e, c)
	}
}
func TestDeploymentLifecycle(t *testing.T) {
	e := deployment(t)
	if !e.Deploying() || e.Observe().Map == nil || e.unit("hero") != nil {
		t.Fatal("must begin unplaced")
	}
	for _, c := range []Command{{Kind: "deployment_finish"}, {Kind: "move", Actor: "hero", X: 1, Y: 2}, {Kind: "end"}, {Kind: "attack", Actor: "boss", Target: "hero"}, {Kind: "unplace", Actor: "boss"}} {
		assertRejected(t, e, c)
	}
	apply(t, e, Command{Kind: "place", Actor: "hero", X: 1, Y: 2})
	before := encoded(e.char("hero"))
	u := copyOf(*e.unit("hero"))
	assertRejected(t, e, Command{Kind: "place", Actor: "archer1", X: 1, Y: 2})
	assertRejected(t, e, Command{Kind: "place", Actor: "hero", X: 2, Y: 2})
	apply(t, e, Command{Kind: "unplace", Actor: "hero"})
	if e.unit("hero") != nil || before != encoded(e.char("hero")) {
		t.Fatal("unplacement removed character")
	}
	apply(t, e, Command{Kind: "place", Actor: "hero", X: 2, Y: 2})
	if e.unit("hero").HP != u.HP || e.unit("hero").MP != u.MP {
		t.Fatal("resources changed")
	}
	assertRejected(t, e, Command{Kind: "deployment_finish"})
	finishDeployment(t, e)
	if e.Deploying() || e.state.Phase != "battle" {
		t.Fatal("battle not started")
	}
	assertRejected(t, e, Command{Kind: "unplace", Actor: "hero"})
}
func TestDeploymentRestore(t *testing.T) {
	e := deployment(t)
	for _, placed := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "partial"}[placed], func(t *testing.T) {
			if placed {
				apply(t, e, Command{Kind: "place", Actor: "hero", X: 1, Y: 2})
			}
			r, err := Restore(e.data, e.Snapshot())
			if err != nil {
				t.Fatal(err)
			}
			if encoded(r.Snapshot()) != encoded(e.Snapshot()) {
				t.Fatal("restore changed placement")
			}
			s := e.Snapshot()
			s.Round = 2
			if _, err := Restore(e.data, s); err == nil {
				t.Fatal("invalid deployment turn accepted")
			}
		})
	}
}
func TestDeploymentFixedPartyAndTerrain(t *testing.T) {
	d := data(t)
	hp, mp := 10, 3
	d.Stages[0].Party = []content.Spawn{{Character: "troop", X: 2, Y: 2, Class: "guard", Level: 1, HP: &hp, MP: &mp, Facing: "NE"}}
	e := New(d, 1)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	apply(t, e, Command{Kind: "start"})
	assertRejected(t, e, Command{Kind: "unplace", Actor: "troop#1"})
	assertRejected(t, e, Command{Kind: "place", Actor: "hero", X: 2, Y: 2})
	assertRejected(t, e, Command{Kind: "place", Actor: "hero", X: 4, Y: 3})
	finishDeployment(t, e)
	if u := e.unit("troop#1"); u.HP != 10 || u.MP != 3 || u.Facing != "NE" {
		t.Fatal("fixed ally overrides changed")
	}
	if _, err := Restore(d, e.Snapshot()); err != nil {
		t.Fatal(err)
	}
}
func TestDeploymentMatchingAvoidsGreedyTrap(t *testing.T) {
	e := deployment(t)
	// The archer can enter this terrain, while the hero cannot.
	e.data.Terrain[e.data.Classes[e.char("hero").Class].Terrain]["i"] = content.Terrain{Cost: 0}
	e.data.Stages[0].Tiles[2] = "..i........."
	e.data.Stages[0].Allies = []content.Point{{X: 1, Y: 2}, {X: 2, Y: 2}}
	finishDeployment(t, e)
	if e.unit("hero").X != 1 || e.unit("archer1").X != 2 {
		t.Fatal("failed constrained matching")
	}
}

func TestDeploymentLegalActionsAndFailure(t *testing.T) {
	e := deployment(t)
	if len(e.LegalActions("").Commands) != 0 || len(e.LegalActions("boss").Commands) != 0 {
		t.Fatal("offered illegal placement")
	}
	commands := e.LegalActions("hero").Commands
	if len(commands) == 0 || commands[0].Kind != "place" {
		t.Fatal("no empty cells")
	}
	apply(t, e, commands[0])
	if commands := e.LegalActions("hero").Commands; len(commands) != 1 || commands[0].Kind != "unplace" {
		t.Fatal("placed officer options")
	}
	e.data.Stages[0].Allies = []content.Point{{X: e.unit("hero").X, Y: e.unit("hero").Y}}
	before := encoded(e.Snapshot())
	if commands, err := e.DeploymentPlan(); err == nil || commands != nil {
		t.Fatal("impossible allocation accepted")
	}
	if before != encoded(e.Snapshot()) {
		t.Fatal("planner mutated state")
	}
}
func TestDeploymentRejectsCorruptPlacement(t *testing.T) {
	e := deployment(t)
	apply(t, e, Command{Kind: "place", Actor: "hero", X: 1, Y: 2})
	for _, change := range []struct {
		name   string
		mutate func(*State)
	}{
		{"outside deployment", func(s *State) {
			for i := range s.Units {
				if s.Units[i].ID == "hero" {
					s.Units[i].X = 3
				}
			}
		}},
		{"resource reset exploit", func(s *State) {
			for i := range s.Units {
				if s.Units[i].ID == "hero" {
					s.Units[i].HP--
				}
			}
		}},
		{"moved enemy", func(s *State) {
			for i := range s.Units {
				if s.Units[i].ID == "boss" {
					s.Units[i].X--
				}
			}
		}},
		{"enemy removed", func(s *State) { s.Units = s.Units[1:] }},
		{"battle with unplaced officer", func(s *State) { s.Phase = "battle" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			s := e.Snapshot()
			change.mutate(&s)
			if _, err := Restore(e.data, s); err == nil {
				t.Fatal("corrupt state accepted")
			}
		})
	}
}
