package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func withEvents(events ...content.BattleEvent) func(*content.Stage) {
	return func(s *content.Stage) {
		s.Victory = []content.Goal{{If: []string{"round>=99"}, Text: "never"}}
		s.Enemies[1].Group, s.Enemies[2].Group = "troops", "troops"
		s.Events = events
	}
}

func do(kind string, a content.Action) content.BattleEvent {
	a.Kind = kind
	return content.BattleEvent{ID: kind, Do: []content.Action{a}}
}

func TestBuffIsTheSameDebuffASkillWouldGive(t *testing.T) {
	e := battleWith(t, withEvents(do("buff", content.Action{Targets: []string{"troops"}, Stat: "attack", Value: -40, Turns: 3})))
	for _, id := range []string{"troop#1", "troop#2"} {
		assert.Equal(t, -40, e.unit(id).Status["attack"].Value)
		assert.Equal(t, 3, e.unit(id).Status["attack"].Turns)
	}
	assert.Empty(t, e.unit("boss").Status, "only the named group")
	plain := battleWith(t, withEvents())
	assert.Less(t, e.stats(e.unit("troop#1")).Attack, plain.stats(plain.unit("troop#1")).Attack, "attack is cut")
}

func TestBuffForTurnsExpiresAndZeroLastsTheBattle(t *testing.T) {
	e := battleWith(t, withEvents(
		do("buff", content.Action{Targets: []string{"troop#1"}, Stat: "defense", Value: -20, Turns: 2}),
		content.BattleEvent{ID: "long", Do: []content.Action{{Kind: "buff", Targets: []string{"boss"}, Stat: "speed", Value: -2}}},
	))
	baseMove := battleWith(t, withEvents()).movement(battleWith(t, withEvents()).unit("boss"))
	assert.Equal(t, baseMove-2, e.movement(e.unit("boss")), "speed counts cells")
	for i := 0; i < 5; i++ {
		applyOK(t, e, Command{Kind: "end"})
		applyOK(t, e, Command{Kind: "end"})
	}
	assert.Empty(t, e.unit("troop#1").Status, "two turns later it is gone")
	assert.Equal(t, -2, e.unit("boss").Status["speed"].Value, "zero turns: it lasts the battle")
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, -2, r.unit("boss").Status["speed"].Value)
}

func TestAnOfficerJoinsTheBattleUnderTheirOwnControl(t *testing.T) {
	e := battleWith(t, withEvents(do("join", content.Action{Spawns: []content.Spawn{{Character: "sage", X: 2, Y: 6}}})))
	sage := e.unit("sage")
	require.NotNil(t, sage)
	assert.Equal(t, "ally", sage.Faction)
	assert.False(t, sage.AI, "the player moves them")
	assert.NotEmpty(t, e.LegalActions("sage").Commands)
	assert.False(t, e.char("sage").Temp, "they stay with the party afterwards")
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.NotNil(t, r.unit("sage"))
	// the battle won, the officer is on the roster
	snap := e.Snapshot()
	killUnits(&snap, "boss", "troop#1", "troop#2")
	snap.Executed["won-S1"] = true
	e, err = Restore(e.Data(), snap)
	require.NoError(t, err)
	assert.NotNil(t, e.char("sage"))
}

func TestJoinSkipsAnOfficerAlreadyOnTheMapOrFallen(t *testing.T) {
	e := battleWith(t, withEvents(do("join", content.Action{Spawns: []content.Spawn{{Character: "archer1", X: 2, Y: 6}}})))
	assert.Equal(t, 1, countUnits(e, "archer1"), "the archer was already on the map")
	e = battleWith(t, func(s *content.Stage) {
		withEvents(do("join", content.Action{Spawns: []content.Spawn{{Character: "sage", X: 2, Y: 6}}}))(s)
		s.Events[0].If = []string{"round>=2"}
	})
	snap := e.Snapshot()
	snap.Characters = append(snap.Characters, Character{ID: "sage", Def: "sage", Class: "mage", Level: 1, Dead: true, Equipment: map[string]string{}, Learned: []string{}})
	snap.Round = 2
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	applyOK(t, e, Command{Kind: "wait", Actor: "hero"})
	assert.Nil(t, e.unit("sage"), "the fallen do not return")
}

func countUnits(e *Engine, id string) int {
	n := 0
	for _, u := range e.Observe().UnitViews {
		if u.ID == id {
			n++
		}
	}
	return n
}

func TestMoveSendsUnitsAnywhereTogether(t *testing.T) {
	moves := []content.Move{
		{Unit: "hero", To: content.Point{X: 10, Y: 6}},  // far beyond its move range
		{Unit: "troops", To: content.Point{X: 5, Y: 6}}, // a group fills the cells around the point
		{Unit: "boss", To: content.Point{X: 10, Y: 6}},  // the cell is taken: the nearest free one
	}
	e := battleWith(t, withEvents(do("move", content.Action{Moves: moves})))
	hero := e.unit("hero")
	assert.Equal(t, [2]int{10, 6}, [2]int{hero.X, hero.Y})
	a, b := e.unit("troop#1"), e.unit("troop#2")
	assert.LessOrEqual(t, dist(a.X, a.Y, 5, 6), 1)
	assert.LessOrEqual(t, dist(b.X, b.Y, 5, 6), 1)
	assert.NotEqual(t, [2]int{a.X, a.Y}, [2]int{b.X, b.Y}, "no two on one cell")
	boss := e.unit("boss")
	assert.Equal(t, 1, dist(boss.X, boss.Y, 10, 6), "next to the hero, who already stands on the point")
}

func TestMoveEventCarriesThePathWalked(t *testing.T) {
	e := battleWith(t, func(*content.Stage) {})
	d := e.Data()
	d.Stages[0].Events = []content.BattleEvent{{ID: "go", If: []string{"round>=1"}, Do: []content.Action{{Kind: "move", Moves: []content.Move{{Unit: "boss", To: content.Point{X: 6, Y: 2}}}}}}}
	e.events = nil
	e.settle()
	var path []content.Point
	for _, ev := range e.events {
		if ev.Kind == "move" {
			path = ev.Path
		}
	}
	require.NotEmpty(t, path)
	assert.Equal(t, content.Point{X: 9, Y: 2}, path[0])
	assert.Equal(t, content.Point{X: 6, Y: 2}, path[len(path)-1])
	assert.Len(t, path, 4)
}

func TestCutsceneMovesFacesAndSpeaksWithoutTouchingHealth(t *testing.T) {
	steps := [][]content.Step{
		{{Kind: "move", Unit: "troops", To: &content.Point{X: 5, Y: 6}}, {Kind: "play", Unit: "boss", Motion: "attack"}},
		{{Kind: "say", Unit: "boss", Text: "돌격!"}, {Kind: "face", Unit: "boss", Facing: "SW"}},
		{{Kind: "wait", Seconds: 1}},
		{{Kind: "play", Unit: "troop#1", Motion: "hit"}, {Kind: "play", Unit: "ghost-not-here", Motion: "hit"}},
	}
	e := battleWith(t, withEvents())
	before := map[string][2]int{}
	for _, u := range e.Observe().UnitViews {
		before[u.ID] = [2]int{u.HP, u.MP}
	}
	e.state.Executed = map[string]bool{}
	e.data.Stages[0].Events = []content.BattleEvent{{ID: "scene", Do: []content.Action{{Kind: "cutscene", Steps: steps}}}}
	e.events = nil
	e.settle()
	var cut [][]CutStep
	for _, ev := range e.events {
		if ev.Kind == "cutscene" {
			cut = ev.Cutscene
		}
	}
	require.Len(t, cut, 4)
	assert.Len(t, cut[0], 3, "two troops move, the boss plays: a group's steps are told apart per unit")
	assert.Equal(t, "SW", e.unit("boss").Facing)
	assert.LessOrEqual(t, dist(e.unit("troop#1").X, e.unit("troop#1").Y, 5, 6), 1)
	assert.Equal(t, CutStep{Kind: "wait", Seconds: 1}, cut[2][0])
	assert.Len(t, cut[3], 1, "the unit that is not on the field is left out")
	for _, u := range e.Observe().UnitViews {
		assert.Equal(t, before[u.ID], [2]int{u.HP, u.MP}, u.ID)
	}
	snap := e.Snapshot()
	_, err := Restore(e.Data(), snap)
	require.NoError(t, err)
}

func TestForcedDuelHoldsTheDuelWhereverTheyStand(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		withEvents(do("duel", content.Action{Duel: "D"}))(s)
		s.Duels = []content.Duel{heroVsBoss}
	})
	assert.True(t, e.state.Executed["duel-D"], "far apart, yet the event had them meet")
	assert.Zero(t, e.unit("boss").HP, "the fixed result: the boss retreats")
}

func TestACutsceneEndsInTheSameStateEveryTime(t *testing.T) {
	steps := [][]content.Step{{{Kind: "move", Unit: "troops", To: &content.Point{X: 5, Y: 6}}, {Kind: "face", Unit: "boss", Facing: "SW"}}}
	run := func() string {
		e := battleWith(t, withEvents(do("cutscene", content.Action{Steps: steps})))
		b, err := json.Marshal(e.Snapshot())
		require.NoError(t, err)
		return string(b)
	}
	assert.Equal(t, run(), run())
}

func TestCutsceneWalksUpToAUnitWhereverItStands(t *testing.T) {
	steps := [][]content.Step{
		{{Kind: "move", Unit: "hero", By: "boss"}},
		{{Kind: "face", Unit: "boss", By: "hero"}},
	}
	e := battleWith(t, withEvents(do("cutscene", content.Action{Steps: steps})))
	hero, boss := e.unit("hero"), e.unit("boss")
	assert.Equal(t, 1, dist(hero.X, hero.Y, boss.X, boss.Y), "beside the boss, who has not moved")
	assert.Equal(t, content.Toward(content.Point{X: hero.X, Y: hero.Y}, content.Point{X: boss.X, Y: boss.Y}), hero.Facing)
	assert.Equal(t, content.Toward(content.Point{X: boss.X, Y: boss.Y}, content.Point{X: hero.X, Y: hero.Y}), boss.Facing)
}
