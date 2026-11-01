package core

import (
	"srpg/internal/content"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	zero := 0
	e.data.Terrain[e.data.Classes[e.char("hero").Class].Terrain].Ground["i"] = content.TerrainRule{Cost: &zero}
	setCell(e, 2, 2, "i--")
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
	if commands := e.LegalActions("hero").Commands; len(commands) < 2 || commands[0].Kind != "unplace" || commands[1].Kind != "reposition" {
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

func TestForcedSortie(t *testing.T) {
	d := data(t)
	for i := range d.Stages {
		d.Stages[i].Forced = true
	}
	e := New(d, 17)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	r := apply(t, e, Command{Kind: "start"})
	finished := false
	for _, ev := range r.Events {
		finished = finished || ev.Kind == "deployment_finish"
	}
	if e.Deploying() || e.state.Phase != "battle" || !finished {
		t.Fatal("a forced sortie goes straight to battle")
	}
	for _, id := range e.state.Deployment {
		if u := e.unit(id); u == nil || !containsPoint(e.data.Stages[e.state.Stage].Allies, content.Point{X: u.X, Y: u.Y}) {
			t.Fatalf("%s is not on an ally cell", id)
		}
	}
	if _, err := Restore(d, e.Snapshot()); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentRedeploy(t *testing.T) {
	setup := func(t *testing.T) *Engine {
		e := deployment(t)
		apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero", "archer1"}})
		for _, id := range []string{"hero", "archer1"} {
			if e.unit(id) == nil {
				t.Fatalf("%s did not stand on the field", id)
			}
		}
		return e
	}
	t.Run("leaving the sortie leaves the field", func(t *testing.T) {
		e := setup(t)
		hero := copyOf(*e.unit("hero"))
		apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero"}})
		assert.Nil(t, e.unit("archer1"))
		assert.Equal(t, hero.X, e.unit("hero").X)
		assert.Equal(t, hero.Y, e.unit("hero").Y)
		assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"archer1"}})
		assertRejected(t, e, Command{Kind: "start"})
	})
	t.Run("new gear stands again in place", func(t *testing.T) {
		e := setup(t)
		before := copyOf(*e.unit("hero"))
		apply(t, e, Command{Kind: "unequip", Actor: "hero", Item: "jerkin"})
		u := e.unit("hero")
		assert.Equal(t, before.X, u.X)
		assert.Equal(t, before.Y, u.Y)
		assert.Equal(t, e.stats(u).MaxHP, u.HP)
		assert.Equal(t, 1, e.state.Warehouse["jerkin"])
	})
	t.Run("reposition moves and trades cells", func(t *testing.T) {
		e := setup(t)
		hero, archer := copyOf(*e.unit("hero")), copyOf(*e.unit("archer1"))
		apply(t, e, Command{Kind: "reposition", Actor: "hero", X: archer.X, Y: archer.Y})
		assert.Equal(t, [2]int{archer.X, archer.Y}, [2]int{e.unit("hero").X, e.unit("hero").Y})
		assert.Equal(t, [2]int{hero.X, hero.Y}, [2]int{e.unit("archer1").X, e.unit("archer1").Y})
		assertRejected(t, e, Command{Kind: "reposition", Actor: "hero", X: archer.X, Y: archer.Y})
		assertRejected(t, e, Command{Kind: "reposition", Actor: "hero", X: 3, Y: 0})
		_, err := Restore(e.data, e.Snapshot())
		require.NoError(t, err)
	})
	t.Run("a deputy leaves the field and stands behind its officer", func(t *testing.T) {
		e := setup(t)
		with(e, "hero", "deputy_t")
		hero := copyOf(*e.unit("hero"))
		before := e.stats(e.unit("hero"))
		apply(t, e, Command{Kind: "deputy", Actor: "hero", Target: "archer1"})
		assert.Nil(t, e.unit("archer1"))
		assert.Equal(t, []string{"hero"}, e.state.Deployment)
		u := e.unit("hero")
		assert.Equal(t, [2]int{hero.X, hero.Y}, [2]int{u.X, u.Y})
		assert.Equal(t, "archer1", e.char("hero").Deputy)
		assert.Equal(t, e.stats(u).MaxHP, u.HP)
		assertRejected(t, e, Command{Kind: "deploy", Deployment: []string{"hero", "archer1"}})
		apply(t, e, Command{Kind: "undeputy", Actor: "hero"})
		assert.Equal(t, before, e.stats(e.unit("hero")))
		assert.Empty(t, e.char("hero").Deputy)
		assert.Nil(t, e.unit("archer1"), "a released deputy waits until sent out")
		apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero", "archer1"}})
		assert.NotNil(t, e.unit("archer1"))
		_, err := Restore(e.data, e.Snapshot())
		require.NoError(t, err)
	})
	t.Run("the lord and the required officers cannot serve as deputies", func(t *testing.T) {
		e := setup(t)
		with(e, "hero", "deputy_t")
		with(e, "archer1", "deputy_t")
		e.data.Stages[e.state.Stage].Required = []string{"archer1"}
		defer func() { e.data.Stages[e.state.Stage].Required = nil }()
		assert.EqualError(t, e.DeputyCheck("archer1", "hero"), "LordDeputy")
		assert.EqualError(t, e.DeputyCheck("hero", "archer1"), "RequiredDeputy")
		assertRejected(t, e, Command{Kind: "deputy", Actor: "hero", Target: "archer1"})
		before, _, err := e.DeputyStats("hero", "")
		require.NoError(t, err)
		_, _, err = e.DeputyStats("hero", "archer1")
		require.NoError(t, err)
		after, _, _ := e.DeputyStats("hero", "")
		assert.Equal(t, before, after, "a preview leaves the officer as it was")
		assert.Empty(t, e.char("hero").Deputy)
	})
	t.Run("the battle's sortie is kept for a retry", func(t *testing.T) {
		e := setup(t)
		apply(t, e, Command{Kind: "unequip", Actor: "hero", Item: "jerkin"})
		apply(t, e, Command{Kind: "deploy", Deployment: []string{"hero"}})
		apply(t, e, Command{Kind: "deployment_finish"})
		cp := e.state.Checkpoint
		require.NotNil(t, cp)
		assert.Equal(t, []string{"hero"}, cp.Deployment)
		assert.Equal(t, 1, cp.Warehouse["jerkin"])
		for _, o := range cp.Characters {
			if o.ID == "hero" {
				assert.Empty(t, o.Equipment["armor"])
			}
		}
		_, err := Restore(e.data, e.Snapshot())
		require.NoError(t, err)
	})
}

func TestRequiredDeputyGoesToBattle(t *testing.T) {
	e := New(data(t), 17)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	with(e, "hero", "deputy_t")
	apply(t, e, Command{Kind: "deputy", Actor: "hero", Target: "archer1"})
	require.Equal(t, "archer1", e.char("hero").Deputy)
	require.NotContains(t, e.state.Deployment, "archer1")
	e.data.Stages[e.state.Stage].Required = []string{"archer1"}
	e.enter() // the battle node again, now requiring the deputy
	assert.Empty(t, e.char("hero").Deputy)
	assert.Contains(t, e.state.Deployment, "archer1")
	assert.Equal(t, "undeputy", e.events[len(e.events)-1].Kind)
	apply(t, e, Command{Kind: "start"})
	_, err := Restore(e.data, e.Snapshot())
	require.NoError(t, err)
}

func TestDeploymentStandsRecommendedOfficersOnTheirCells(t *testing.T) {
	e := deployment(t)
	e.data.Stages[0].Allies = []content.Point{{X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}}
	e.data.Stages[0].Recommend = map[string]content.Point{"archer1": {X: 1, Y: 2}}
	finishDeployment(t, e)
	assert.Equal(t, content.Point{X: 1, Y: 2}, content.Point{X: e.unit("archer1").X, Y: e.unit("archer1").Y}, "the recommended cell")
	assert.Equal(t, content.Point{X: 2, Y: 2}, content.Point{X: e.unit("hero").X, Y: e.unit("hero").Y}, "the others keep the cells' order past it")

	e = deployment(t)
	e.data.Stages[0].Allies = []content.Point{{X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}}
	e.data.Stages[0].Recommend = map[string]content.Point{"archer1": {X: 5, Y: 5}}
	finishDeployment(t, e)
	assert.NotNil(t, e.unit("archer1"), "a recommendation off the ally cells is passed over")
}
