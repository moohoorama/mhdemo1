package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// battleWith starts the mini stage S1 after tweaking it: heroes at (1,2),(1,3); boss (9,2);
// troops #1 (9,4) and #2 (10,3).
func battleWith(t *testing.T, tweak func(*content.Stage)) *Engine {
	return battleOn(t, tweak, false)
}

// bentBattleWith is battleWith for a tweak the rules do not allow (a unit set on water), so Apply skips the
// restore check.
func bentBattleWith(t *testing.T, tweak func(*content.Stage)) *Engine {
	return battleOn(t, tweak, true)
}

func battleOn(t *testing.T, tweak func(*content.Stage), bent bool) *Engine {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	tweak(&d.Stages[0])
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "S1" {
			d.Campaign.Start = n.ID
		}
	}
	e := New(d, 1)
	e.unchecked = bent
	_, err = e.Apply(Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	require.Equal(t, "battle", e.Observe().Phase)
	return e
}

func flagEvent(id string, cond ...string) content.BattleEvent {
	return content.BattleEvent{ID: id, If: cond, Do: []content.Action{{Kind: "flag", Flags: map[string]int{id: 1}}}}
}

func fired(e *Engine, id string) bool { return e.state.Executed["battle-S1-"+id] }

func applyOK(t *testing.T, e *Engine, c Command) Result {
	r, err := e.Apply(c)
	require.NoError(t, err, "%+v", c)
	return r
}

func TestTurnstartWaitsForTheUnitToSurviveAPlayerTurn(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[1].Group = "g"
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 9, Y: 5, W: 1, H: 1}, Name: "gate"}}
		s.Events = []content.BattleEvent{flagEvent("burn", "turnstart:enemy", "at:g:1")}
	})
	apply(t, e, Command{Kind: "end"}) // enemy turn starts, the troop is not in the area yet
	assert.False(t, fired(e, "burn"))
	apply(t, e, Command{Kind: "move", Actor: "troop#1", X: 9, Y: 5}) // it arrives during its own turn
	assert.False(t, fired(e, "burn"), "arriving is not surviving")
	apply(t, e, Command{Kind: "end"}) // player turn
	assert.False(t, fired(e, "burn"))
	apply(t, e, Command{Kind: "end"}) // the enemy turn starts with the troop still there
	assert.True(t, fired(e, "burn"))
	assert.Equal(t, 1, e.flag("burn"))
	assert.Zero(t, e.state.Flags["burn"], "a battle's flags stay pending until it is won")
}

func TestAreaEventWaitsForTheMoverToFinishItsAction(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 2, Y: 2, W: 1, H: 1}, Name: "ford"}}
		s.Events = []content.BattleEvent{flagEvent("pincer", "at:hero:1")}
	})
	apply(t, e, Command{Kind: "move", Actor: "hero", X: 2, Y: 2})
	assert.False(t, fired(e, "pincer"), "a move still being ordered sets nothing off")
	apply(t, e, Command{Kind: "wait", Actor: "hero"})
	assert.True(t, fired(e, "pincer"))
}

func TestTurnstartEventDoesNotFireWhenTheUnitIsGone(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[1].Group = "g"
		s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 9, Y: 4, W: 1, H: 1}, Name: "gate"}}
		s.Events = []content.BattleEvent{flagEvent("burn", "turnstart:enemy", "at:g:1")}
	})
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].Group == "g" {
			snap.Units[i].HP = 0
		}
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	apply(t, e, Command{Kind: "end"})
	assert.False(t, fired(e, "burn"))
}

func TestConditionsAgainstTheUnitsOnTheMap(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[0].Group = "lords"
		s.Enemies[1].Group = "troops"
		s.Enemies[2].Group = "troops"
		s.Areas = []content.Area{
			{ID: 1, Rect: content.MapRect{X: 8, Y: 1, W: 4, H: 4}, Name: "camp"},
			{ID: 2, Rect: content.MapRect{X: 0, Y: 0, W: 2, H: 5}, Name: "ours"},
		}
	})
	check := func(want bool, cond string) {
		c, err := content.ParseCond(cond)
		require.NoError(t, err, cond)
		assert.Equal(t, want, e.holds(c, true) != c.Not, cond)
	}
	check(true, "alive:boss")
	check(true, "alive:troops")
	check(false, "defeated:troops")
	check(false, "alldefeated:troops")
	check(true, "round>=1")
	check(false, "round>=2")
	check(true, "turnstart:ally")
	check(false, "turnstart:enemy")
	check(true, "at:boss:1")
	check(true, "allat:troops:1")
	check(false, "at:boss:2")
	check(true, "at:hero:2")
	check(false, "adjacent:hero:boss")
	check(false, "hp:boss<100")
	check(true, "any:[round>=9, at:boss:1]")
	check(false, "any:[round>=9, at:boss:2]")
	check(true, "!any:[round>=9, at:boss:2]")
	check(false, "duel:nothing-yet")

	snap := e.Snapshot()
	for i := range snap.Units {
		switch {
		case snap.Units[i].Group == "troops" && snap.Units[i].ID == "troop#1":
			snap.Units[i].HP = 0
		case snap.Units[i].ID == "boss":
			snap.Units[i].X, snap.Units[i].Y, snap.Units[i].HP = 1, 1, snap.Units[i].HP/2
		}
	}
	snap.Executed["duel-x"] = true
	snap.PendingFlags = map[string]int{"seen": 2}
	var err error
	e, err = Restore(e.Data(), snap)
	require.NoError(t, err)
	check(true, "defeated:troops")
	check(false, "alldefeated:troops")
	check(true, "allat:troops:1") // the dead one does not count, the other still stands in the camp
	check(true, "adjacent:archer1:boss")
	check(false, "adjacent:hero:boss")
	check(true, "at:boss:2")
	check(true, "hp:boss<51")
	check(false, "hp:boss<49")
	check(true, "flag:seen>=2")
	check(false, "flag:seen>=3")
}

func TestRepeatingEventFiresOnEachRiseOfItsCondition(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[0].Group = "boss"
		s.Events = []content.BattleEvent{{ID: "wince", Repeat: true, If: []string{"hp:boss<50"},
			Do: []content.Action{{Kind: "say", Lines: []content.Speech{{Actor: "boss", Text: "으윽"}}}}}}
	})
	speeches := func(r Result) int {
		n := 0
		for _, ev := range r.Events {
			if ev.Kind == "speech" {
				n++
			}
		}
		return n
	}
	setBossHP := func(pct int) {
		snap := e.Snapshot()
		for i := range snap.Units {
			if snap.Units[i].ID == "boss" {
				snap.Units[i].HP = e.stats(e.unit("boss")).MaxHP * pct / 100
			}
		}
		var err error
		e, err = Restore(e.Data(), snap)
		require.NoError(t, err)
	}
	wait := func() Result { return apply(t, e, Command{Kind: "wait", Actor: "hero"}) }
	assert.Equal(t, 0, speeches(wait()))
	setBossHP(30)
	assert.Equal(t, 1, speeches(apply(t, e, Command{Kind: "wait", Actor: "archer1"})))
	setBossHP(30)
	assert.Equal(t, 0, speeches(apply(t, e, Command{Kind: "end"})), "still true: no second line")
	setBossHP(90)
	apply(t, e, Command{Kind: "end"})
	setBossHP(10)
	assert.Equal(t, 1, speeches(apply(t, e, Command{Kind: "wait", Actor: "hero"})), "it rose again")
}

func TestOnceEventsStayDoneAcrossSaveAndRestore(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) { s.Events = []content.BattleEvent{flagEvent("opening")} })
	assert.True(t, fired(e, "opening"), "no conditions: fires with the first settle of the battle")
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	res := apply(t, r, Command{Kind: "wait", Actor: "hero"})
	for _, ev := range res.Events {
		assert.NotEqual(t, "speech", ev.Kind)
	}
	assert.True(t, fired(r, "opening"))
}

func TestEventChainsStopAfterEightPasses(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		// Written backwards, each pass can run only the one event whose predecessor ran last pass.
		for i := 10; i >= 1; i-- {
			if i == 1 {
				s.Events = append(s.Events, flagEvent("e1"))
			} else {
				s.Events = append(s.Events, flagEvent(fmt.Sprintf("e%d", i), fmt.Sprintf("flag:e%d", i-1)))
			}
		}
	})
	count := func() int {
		n := 0
		for i := 1; i <= 10; i++ {
			if fired(e, fmt.Sprintf("e%d", i)) {
				n++
			}
		}
		return n
	}
	assert.Equal(t, eventPasses, count())
	apply(t, e, Command{Kind: "wait", Actor: "hero"})
	assert.Equal(t, 10, count(), "the rest follow with the next command")
}

func TestSpawnEventWaitsWhenNoRoomForTheWholeWave(t *testing.T) {
	wave := []content.Spawn{}
	for i := 0; i < 100; i++ {
		wave = append(wave, content.Spawn{Character: "troop", X: 5, Y: 5, Side: "enemy"})
	}
	e := battleWith(t, func(s *content.Stage) {
		s.Events = []content.BattleEvent{{ID: "flood", Do: []content.Action{{Kind: "spawn", Spawns: wave}}}}
	})
	assert.False(t, fired(e, "flood"))
	assert.Len(t, e.Observe().Units, 3+2+0, "nobody came: the heroes and the stage's own enemies")
}

func TestSpawnAndRemoveByGroup(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[1].Group = "guards"
		s.Enemies[2].Group = "guards"
		s.Events = []content.BattleEvent{
			{ID: "ambush", If: []string{"round>=1"}, Do: []content.Action{{Kind: "spawn", Spawns: []content.Spawn{
				{Character: "troop", X: 6, Y: 6, Side: "enemy", Group: "ambushers"}, {Character: "troop", X: 6, Y: 7, Side: "enemy", Group: "ambushers"}}}}},
			{ID: "flee", If: []string{"alldefeated:ambushers"}, Do: []content.Action{{Kind: "say", Lines: []content.Speech{{Actor: "boss", Text: "!"}}}}},
			{ID: "retreat", If: []string{"round>=1"}, Do: []content.Action{{Kind: "remove", Targets: []string{"guards"}}}},
		}
	})
	groups := map[string]int{}
	for _, u := range e.Observe().Units {
		if u.HP > 0 {
			groups[u.Group]++
		}
	}
	assert.Equal(t, 2, groups["ambushers"])
	assert.Zero(t, groups["guards"], "removed from the map")
	assert.False(t, fired(e, "flee"))
}

func TestAnEventChangesAUnitsPolicyAndTheSaveStillLoads(t *testing.T) {
	goal := content.Point{X: 10, Y: 7}
	e := battleWith(t, func(s *content.Stage) {
		s.Events = []content.BattleEvent{{ID: "turn", Do: []content.Action{{Kind: "ai", Targets: []string{"boss"}, Policy: "march", Goal: &goal}}}}
	})
	u := e.unit("boss")
	require.Equal(t, "march", u.Policy)
	require.Equal(t, &goal, u.Goal)
	assert.NotSame(t, &goal, u.Goal, "the unit has its own copy")
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, "march", r.unit("boss").Policy)

	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == "boss" {
			snap.Units[i].Goal = nil
		}
	}
	_, err = Restore(e.Data(), snap)
	assert.Error(t, err, "a marching unit needs its goal")
}

func TestAFleeingUnitLeavesWhenItStandsOnItsGoal(t *testing.T) {
	goal := content.Point{X: 10, Y: 3}
	e := battleWith(t, func(s *content.Stage) {
		s.Enemies[2].Policy, s.Enemies[2].Goal = "flee", &goal // troop #2 starts at (10,3): already there
	})
	assert.Zero(t, e.unit("troop#2").HP)
	assert.Positive(t, e.unit("troop#1").HP)
}

func TestSurgeStepNamesTheUnitThatLeadsTheArmy(t *testing.T) {
	steps := [][]content.Step{{{Kind: "surge", Text: "flood", Unit: "boss"}, {Kind: "say", Unit: "boss", Text: "건너라!", Seconds: .6}}}
	e := battleWith(t, withEvents())
	e.state.Executed = map[string]bool{}
	e.data.Stages[0].Events = []content.BattleEvent{{ID: "flood", Do: []content.Action{{Kind: "cutscene", Steps: steps}}}}
	e.events = nil
	e.settle()
	var surge []CutStep
	for _, ev := range e.events {
		for _, group := range ev.Cutscene {
			for _, st := range group {
				if st.Kind == "surge" {
					surge = append(surge, st)
				}
			}
		}
	}
	require.Len(t, surge, 1)
	assert.Equal(t, "flood", surge[0].Text)
	assert.Equal(t, e.unit("boss").ID, surge[0].Unit, "the show needs to know who walks at the head of the army")
}

