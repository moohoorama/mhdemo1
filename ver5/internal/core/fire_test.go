package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// fireOn sets the given area alight at the start of the battle: a patch of burning ground ("F")
// or a wall of flames ("W"), for turns rounds (0: for good).
func fireOn(tile string, turns int, rect content.MapRect) func(*content.Stage) {
	return func(s *content.Stage) {
		s.Areas = []content.Area{{ID: 1, Rect: rect, Name: "fire"}}
		s.Events = []content.BattleEvent{{ID: "ignite", Do: []content.Action{{Kind: "terrain", Area: 1, Tile: tile, Turns: turns}}}}
	}
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

func TestFireTilesAreInTheData(t *testing.T) {
	for _, dir := range []string{"../../assets/data", "../testdata/mini"} {
		d, err := content.Load(dir)
		require.NoError(t, err)
		assert.Equal(t, -20, d.Tiles["F"].RestHP, dir)
		for family, table := range d.Terrain {
			assert.Equal(t, content.Terrain{Factor: .8, Cost: 1}, table["F"], family+" burning ground: every class at 80%")
			assert.Zero(t, table["W"].Cost, family+": a wall of flames cannot be entered")
		}
	}
}

func TestTerrainEventLightsTheAreaAndTheMapShowsIt(t *testing.T) {
	e := battleWith(t, fireOn("F", 0, content.MapRect{X: 5, Y: 6, W: 2, H: 1}))
	assert.Equal(t, "F", e.tile(5, 6))
	assert.Equal(t, "F", e.tile(6, 6))
	assert.Equal(t, ".", e.tile(7, 6))
	assert.Equal(t, 'F', rune(e.Observe().Map.Tiles[6][5]), "the observed map carries the change")
	assert.Equal(t, '.', rune(e.data.Stages[0].Tiles[6][5]), "the stage itself is untouched")
}

func TestStandingInFireHurtsAtTheStartOfYourTurnButWalkingThroughDoesNot(t *testing.T) {
	// hero (1,3) can walk across the fire at (2,3) and stand at (3,3); archer (1,2) is told to stand in it.
	e := battleWith(t, fireOn("F", 0, content.MapRect{X: 2, Y: 3, W: 1, H: 1}))
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
	e := battleWith(t, fireOn("F", 0, content.MapRect{X: 1, Y: 2, W: 1, H: 2}))
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

func TestAWallOfFlamesBlocksMovement(t *testing.T) {
	e := battleWith(t, fireOn("W", 0, content.MapRect{X: 2, Y: 2, W: 1, H: 2}))
	for _, c := range e.LegalActions("hero").Commands {
		assert.False(t, c.Kind == "move" && c.X == 2 && (c.Y == 2 || c.Y == 3), "no one steps into the flames")
	}
	assert.Zero(t, e.StepCost("hero", 2, 3))
}

func TestTemporaryFireBurnsOutAfterItsRounds(t *testing.T) {
	e := battleWith(t, fireOn("F", 2, content.MapRect{X: 5, Y: 6, W: 1, H: 1}))
	require.Equal(t, "F", e.tile(5, 6))
	round := func() {
		applyOK(t, e, Command{Kind: "end"})
		applyOK(t, e, Command{Kind: "end"})
	}
	round()
	assert.Equal(t, "F", e.tile(5, 6), "one round gone")
	round()
	assert.Equal(t, ".", e.tile(5, 6), "back to the grass")
	assert.Empty(t, e.state.TileOverrides)
}

func TestLightingTheSameCellAgainReplacesTheChange(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 5, Y: 6, W: 1, H: 1}, Name: "a"}}
		s.Events = []content.BattleEvent{
			{ID: "one", Do: []content.Action{{Kind: "terrain", Area: 1, Tile: "F"}}},
			{ID: "two", If: []string{"round>=1", "flag:never"}, Do: []content.Action{{Kind: "terrain", Area: 1, Tile: "W"}}},
		}
	})
	require.Equal(t, "F", e.tile(5, 6))
	e.state.PendingFlags = map[string]int{"never": 1}
	e.settle()
	assert.Equal(t, "W", e.tile(5, 6))
	assert.Len(t, e.state.TileOverrides, 1)
}

func TestTileChangesSurviveSaveAndAreRejectedWhenMalformed(t *testing.T) {
	e := battleWith(t, fireOn("F", 3, content.MapRect{X: 5, Y: 6, W: 2, H: 1}))
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, e.state.TileOverrides, r.state.TileOverrides)
	for name, edit := range map[string]func(*State){
		"unknown tile": func(s *State) { s.TileOverrides[0].Tile = "?" },
		"off the map":  func(s *State) { s.TileOverrides[0].X = 99 },
		"negative":     func(s *State) { s.TileOverrides[0].Turns = -1 },
		"duplicate":    func(s *State) { s.TileOverrides = append(s.TileOverrides, s.TileOverrides[0]) },
	} {
		snap := e.Snapshot()
		edit(&snap)
		_, err := Restore(e.Data(), snap)
		assert.Error(t, err, name)
	}
}

func TestRetryClearsTheFire(t *testing.T) {
	e := battleWith(t, fireOn("F", 0, content.MapRect{X: 5, Y: 6, W: 1, H: 1}))
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
	assert.Empty(t, e.state.TileOverrides)
}
