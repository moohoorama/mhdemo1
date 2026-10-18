package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// fireOn sets the given area alight at the start of the battle: embers ("e") or flames ("F"), for
// turns rounds (0: for good).
func fireOn(state string, turns int, rect content.MapRect) func(*content.Stage) {
	return func(s *content.Stage) {
		s.Areas = []content.Area{{ID: 1, Rect: rect, Name: "fire"}}
		s.Events = []content.BattleEvent{{ID: "ignite", Do: []content.Action{{Kind: "terrain", Area: 1, State: state, Turns: turns}}}}
	}
}

// setCell puts a cell of three layers into the first stage.
func setCell(e *Engine, x, y int, cell string) {
	row := e.data.Stages[0].Tiles[y]
	e.data.Stages[0].Tiles[y] = row[:x*content.CellWidth] + cell + row[(x+1)*content.CellWidth:]
}

func damageTo(r Result, id string) int {
	n := 0
	for _, ev := range r.Events {
		if ev.Kind == "damage" && ev.Target == id {
			n += ev.Amount
		}
	}
	return n
}

func TestFireStatesAreInTheData(t *testing.T) {
	for _, dir := range []string{"../../assets/data", "../testdata/mini"} {
		d, err := content.Load(dir)
		require.NoError(t, err)
		require.NotNil(t, d.States["e"].RestHP, dir)
		assert.Equal(t, -20, *d.States["e"].RestHP, dir)
		for family := range d.Terrain {
			embers := d.Move(family, content.Cell{Ground: 'g', Structure: content.NoLayer, State: 'e'})
			assert.Equal(t, content.Terrain{Factor: .8, Cost: 1}, embers, family+" embers on grass: every class at 80%")
			assert.Zero(t, d.Move(family, content.Cell{Ground: 'g', Structure: content.NoLayer, State: 'F'}).Cost, family+": flames cannot be entered")
		}
	}
}

func TestTerrainEventLightsTheAreaAndTheMapShowsIt(t *testing.T) {
	e := battleWith(t, fireOn("e", 0, content.MapRect{X: 5, Y: 6, W: 2, H: 1}))
	assert.Equal(t, byte('e'), e.Cell(5, 6).State)
	assert.Equal(t, byte('e'), e.Cell(6, 6).State)
	assert.Equal(t, byte(content.NoLayer), e.Cell(7, 6).State)
	assert.Equal(t, byte('g'), e.Cell(5, 6).Ground, "the ground under the fire is still there")
	assert.Equal(t, byte('e'), e.Observe().Map.Cell(5, 6).State, "the observed map carries the state")
	assert.Equal(t, byte(content.NoLayer), e.data.Stages[0].Cell(5, 6).State, "the stage itself is untouched")
}

func TestStandingInFireHurtsAtTheStartOfYourTurnButWalkingThroughDoesNot(t *testing.T) {
	// hero (1,3) can walk across the fire at (2,3) and stand at (3,3); archer (1,2) is told to stand in it.
	e := battleWith(t, fireOn("e", 0, content.MapRect{X: 2, Y: 3, W: 1, H: 1}))
	maxHP := e.stats(e.unit("hero")).MaxHP
	applyOK(t, e, Command{Kind: "move", Actor: "hero", X: 3, Y: 3}) // across the fire, ends beside it
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == "archer1" {
			snap.Units[i].X, snap.Units[i].Y = 2, 3
		}
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	applyOK(t, e, Command{Kind: "end"})
	r := applyOK(t, e, Command{Kind: "end"}) // the allies' turn starts
	assert.Zero(t, damageTo(r, "hero"), "passing through costs nothing")
	assert.Equal(t, maxHP, e.unit("hero").HP)
	assert.Equal(t, e.stats(e.unit("archer1")).MaxHP*20/100, damageTo(r, "archer1"), "standing in it costs a fifth")
}

func TestFireCanDriveAUnitOffTheField(t *testing.T) {
	e := battleWith(t, fireOn("e", 0, content.MapRect{X: 1, Y: 2, W: 1, H: 2}))
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == "archer1" {
			snap.Units[i].HP = 3
		}
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	applyOK(t, e, Command{Kind: "end"})
	r := applyOK(t, e, Command{Kind: "end"})
	assert.Zero(t, e.unit("archer1").HP)
	retreated := false
	for _, ev := range r.Events {
		retreated = retreated || ev.Kind == "retreat" && ev.Actor == "archer1"
	}
	assert.True(t, retreated)
}

func TestFlamesBlockMovement(t *testing.T) {
	e := battleWith(t, fireOn("F", 0, content.MapRect{X: 2, Y: 2, W: 1, H: 2}))
	for _, c := range e.LegalActions("hero").Commands {
		assert.False(t, c.Kind == "move" && c.X == 2 && (c.Y == 2 || c.Y == 3), "no one steps into the flames")
	}
	assert.Zero(t, e.StepCost("hero", 2, 3))
}

func TestFlamesBurnOutIntoSootAfterTheirRounds(t *testing.T) {
	e := battleWith(t, fireOn("F", 2, content.MapRect{X: 5, Y: 6, W: 1, H: 1}))
	require.Equal(t, byte('F'), e.Cell(5, 6).State)
	round := func() {
		applyOK(t, e, Command{Kind: "end"})
		applyOK(t, e, Command{Kind: "end"})
	}
	round()
	assert.Equal(t, byte('F'), e.Cell(5, 6).State, "one round gone")
	round()
	assert.Equal(t, byte('s'), e.Cell(5, 6).State, "soot is left")
	assert.NotZero(t, e.StepCost("hero", 5, 6), "soot costs a step like the ground under it")
	round()
	assert.Equal(t, byte('s'), e.Cell(5, 6).State, "soot stays for good")
}

func TestEmbersStayForGoodAndLeaveGroundAndStructureAlone(t *testing.T) {
	e := battle(t)
	setCell(e, 4, 2, "gv-")
	e.setState(e.data.Stages[0], 4, 2, "e", 0)
	for i := 0; i < 3; i++ {
		applyOK(t, e, Command{Kind: "end"})
		applyOK(t, e, Command{Kind: "end"})
	}
	assert.Equal(t, content.Cell{Ground: 'g', Structure: 'v', State: 'e'}, e.Cell(4, 2))
	assert.Equal(t, -20, e.CellProps(4, 2).RestHP, "a burning village no longer heals")
}

func TestLightingTheSameCellAgainReplacesTheState(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 5, Y: 6, W: 1, H: 1}, Name: "a"}}
		s.Events = []content.BattleEvent{
			{ID: "one", Do: []content.Action{{Kind: "terrain", Area: 1, State: "e"}}},
			{ID: "two", If: []string{"round>=1", "flag:never"}, Do: []content.Action{{Kind: "terrain", Area: 1, State: "F"}}},
		}
	})
	require.Equal(t, byte('e'), e.Cell(5, 6).State)
	e.state.PendingFlags = map[string]int{"never": 1}
	e.settle()
	assert.Equal(t, byte('F'), e.Cell(5, 6).State)
	assert.Len(t, e.state.TileStates, 1)
}

func TestClearingAStateRemovesIt(t *testing.T) {
	e := battleWith(t, fireOn("e", 0, content.MapRect{X: 5, Y: 6, W: 1, H: 1}))
	require.Equal(t, byte('e'), e.Cell(5, 6).State)
	e.changeTerrain(e.data.Stages[0], content.Action{Kind: "terrain", Area: 1, State: string(content.NoLayer)})
	assert.Equal(t, byte(content.NoLayer), e.Cell(5, 6).State)
	assert.Empty(t, e.state.TileStates)
}

func TestStatesSurviveSaveAndAreRejectedWhenMalformed(t *testing.T) {
	e := battleWith(t, fireOn("F", 3, content.MapRect{X: 5, Y: 6, W: 2, H: 1}))
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, e.state.TileStates, r.state.TileStates)
	for name, edit := range map[string]func(*State){
		"unknown state": func(s *State) { s.TileStates[0].State = "?" },
		"off the map":   func(s *State) { s.TileStates[0].X = 99 },
		"negative":      func(s *State) { s.TileStates[0].Turns = -1 },
		"duplicate":     func(s *State) { s.TileStates = append(s.TileStates, s.TileStates[0]) },
	} {
		snap := e.Snapshot()
		edit(&snap)
		_, err := Restore(e.Data(), snap)
		assert.Error(t, err, name)
	}
}

func TestRetryClearsTheFire(t *testing.T) {
	e := battleWith(t, fireOn("e", 0, content.MapRect{X: 5, Y: 6, W: 1, H: 1}))
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].Faction == "ally" {
			snap.Units[i].HP = 0
		}
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	applyOK(t, e, Command{Kind: "end"})
	require.NoError(t, e.Retry())
	assert.Empty(t, e.state.TileStates)
}
