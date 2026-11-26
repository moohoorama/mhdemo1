package session

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func autosaveBattle(t *testing.T) *Session {
	t.Helper()
	s := setup(t)
	for i := range s.Data.Stages[0].Enemies {
		s.Data.Stages[0].Enemies[i].Policy = "hold"
	}
	request(t, s, Request{Op: "new", Seed: 3})
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}, {Kind: "place", Actor: "hero", X: 1, Y: 2}, {Kind: "place", Actor: "archer1", X: 1, Y: 3}} {
		request(t, s, Request{Op: "command", Command: c})
	}
	require.Empty(t, s.Store.Slots())
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "deployment_finish"}})
	return s
}

func TestAutosaveAtAlliedRoundStarts(t *testing.T) {
	for _, mode := range []string{"resolve", "advance", "auto", "ai"} {
		t.Run(mode, func(t *testing.T) {
			s := autosaveBattle(t)
			first, err := s.Store.Load("auto-battle1")
			require.NoError(t, err)
			assert.Equal(t, "battle", first.Phase)
			assert.Equal(t, "ally", first.Turn)
			assert.Equal(t, 1, first.Round)
			request(t, s, Request{Op: "command", Command: core.Command{Kind: "wait", Actor: "hero"}})
			request(t, s, Request{Op: "observe"})
			assert.Len(t, s.Store.Slots(), 1)
			request(t, s, Request{Op: "save", Slot: "1"})
			for round := 2; round <= 12; round++ {
				request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
				require.Equal(t, "enemy", s.Engine.Observe().Turn)
				switch mode {
				case "resolve":
					_, err = s.ResolvePhase()
					require.NoError(t, err)
				case "advance":
					_, err = s.AdvanceEnemies()
					require.NoError(t, err)
				case "auto":
					request(t, s, Request{Op: "auto"})
				case "ai":
					for i := 0; s.Engine.Observe().Turn == "enemy" && i < 50; i++ {
						request(t, s, Request{Op: "ai"})
					}
				}
				require.Equal(t, "ally", s.Engine.Observe().Turn)
				assert.Equal(t, round, s.Engine.Observe().Round)
			}
			for slot, round := range map[string]int{"auto-battle1": 11, "auto-battle2": 12, "auto-battle3": 3, "auto-battle10": 10} {
				state, err := s.Store.Load(slot)
				require.NoError(t, err)
				assert.Equal(t, round, state.Round)
				assert.Equal(t, "ally", state.Turn)
				for _, u := range state.Units {
					if u.ActsIn(state.Turn, state.Army) {
						assert.False(t, u.Done)
						assert.False(t, u.Acted)
						assert.False(t, u.Moved)
					}
				}
			}
			manual, err := s.Store.Load("1")
			require.NoError(t, err)
			assert.Equal(t, 1, manual.Round)
			request(t, s, Request{Op: "load", Slot: "auto-battle3", Discard: true})
			assert.Len(t, s.Store.Slots(), 11)
			request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
			_, err = s.ResolvePhase()
			require.NoError(t, err)
			latest, err := s.Store.Load("auto-battle3")
			require.NoError(t, err)
			assert.Equal(t, 4, latest.Round)
		})
	}
}

func TestAutosaveSlotsAreReadOnlyForManualSave(t *testing.T) {
	s := autosaveBattle(t)
	before, err := s.Store.Load("auto-battle1")
	require.NoError(t, err)
	for _, slot := range []string{"auto1", "auto-scenario1", "auto-battle1"} {
		r := s.Handle(Request{Op: "save", Slot: slot, Overwrite: true})
		assert.False(t, r.OK)
		assert.Equal(t, "InvalidSlot", r.Error)
	}
	after, err := s.Store.Load("auto-battle1")
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestAutosaveFailureKeepsAppliedCommand(t *testing.T) {
	s := autosaveBattle(t)
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
	// Block the next automatic slot; the resolved phase must still reach the new round.
	require.NoError(t, os.Mkdir(filepath.Join(s.Store.Dir, "slot-auto-battle2.json"), 0700))
	actions, err := s.ResolvePhase()
	require.NoError(t, err)
	assert.Equal(t, 2, s.Engine.Observe().Round)
	failed := false
	for _, a := range actions {
		for _, step := range a.Steps {
			for _, event := range step.Events {
				failed = failed || event.Kind == "autosave-failed"
			}
		}
	}
	assert.True(t, failed)
}

func TestSupportPhaseDoesNotRotateAutosave(t *testing.T) {
	s := setup(t)
	s.Data.Stages[0].Party = []content.Spawn{{Character: "troop", X: 1, Y: 4, AI: true, Policy: "hold", Level: 1}}
	for i := range s.Data.Stages[0].Enemies {
		s.Data.Stages[0].Enemies[i].Policy = "hold"
	}
	request(t, s, Request{Op: "new", Seed: 3})
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}, {Kind: "place", Actor: "hero", X: 1, Y: 2}, {Kind: "place", Actor: "archer1", X: 1, Y: 3}, {Kind: "deployment_finish"}, {Kind: "end"}} {
		request(t, s, Request{Op: "command", Command: c})
	}
	require.Equal(t, "support", s.Engine.Observe().Turn)
	require.Len(t, s.Store.Slots(), 1)
	_, err := s.ResolvePhase()
	require.NoError(t, err)
	require.Equal(t, "enemy", s.Engine.Observe().Turn)
	assert.Len(t, s.Store.Slots(), 1)
	_, err = s.ResolvePhase()
	require.NoError(t, err)
	assert.Equal(t, "ally", s.Engine.Observe().Turn)
	assert.Len(t, s.Store.Slots(), 2)
}

func TestAutosaveBeforeSortie(t *testing.T) {
	s := setup(t)
	flow := []content.Node{
		{ID: "walk", Kind: "explore", Explore: "explore01"},
		{ID: "pre", Kind: "narrate", Text: "off we go"},
		{ID: "fight", Kind: "battle", Stage: "S1"},
	}
	s.Data.Nodes = flow
	s.Data.Chapters = []content.Chapter{{ID: "1", File: "x.json", Nodes: []string{"walk", "pre", "fight"}}}
	s.Data.Campaign.Start = "walk"
	s.Data.Explores = map[string]*content.Explore{"explore01": {
		Hero:  "hero",
		Start: content.Spot{Area: "street", At: [2]int{5, 5}, Face: "SE"},
		Areas: map[string]*content.ExploreArea{"street": {Map: "S_a", Name: "거리", At: [2]int{1, 1},
			People: map[string]*content.ExplorePerson{"관우": {Face: "SW", Sortie: &content.ExploreSortie{Ask: "출진?"}}}}},
	}}
	require.NoError(t, s.Data.Validate())
	request(t, s, Request{Op: "new", Seed: 1})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "walk", Option: "street", X: 2, Y: 3, Target: "NW"}})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "sortie", Target: "관우"}})
	assert.Equal(t, "pre", s.Engine.Observe().Node)

	request(t, s, Request{Op: "load", Slot: "auto-scenario1", Discard: true})
	o := s.Engine.Observe()
	assert.Equal(t, "walk", o.Node, "the save walks again before the confidant is answered")
	assert.Equal(t, content.Spot{Area: "street", At: [2]int{2, 3}, Face: "NW"}, o.Spot)
	assert.Equal(t, []string{"auto-scenario1"}, s.Store.Slots())
}

func TestAutosaveOnMarkedExploreEntry(t *testing.T) {
	s := setup(t)
	s.Data.Nodes = []content.Node{
		{ID: "intro", Kind: "narrate", Text: "farewell"},
		{ID: "walk", Kind: "explore", Explore: "explore01", Autosave: true},
		{ID: "after", Kind: "narrate", Text: "after"},
	}
	s.Data.Chapters = []content.Chapter{{ID: "1", File: "x.json", Nodes: []string{"intro", "walk", "after"}}}
	s.Data.Campaign.Start = "intro"
	s.Data.Explores = map[string]*content.Explore{"explore01": {
		Hero: "hero", Start: content.Spot{Area: "street", At: [2]int{5, 5}, Face: "SE"},
		Areas: map[string]*content.ExploreArea{"street": {Map: "S_a", Name: "거리", At: [2]int{5, 5}}},
	}}
	require.NoError(t, s.Data.Validate())
	request(t, s, Request{Op: "new", Seed: 1})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "next"}})
	saved, err := s.Store.Load("auto-scenario1")
	require.NoError(t, err)
	assert.Equal(t, "walk", saved.Node)
	assert.Equal(t, "walk", saved.Explore)
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "walk", Option: "street", X: 2, Y: 3, Target: "NW"}})
	assert.Equal(t, []string{"auto-scenario1"}, s.Store.Slots())
	request(t, s, Request{Op: "load", Slot: "auto-scenario1", Discard: true})
	assert.Equal(t, "walk", s.Engine.Observe().Node)
	assert.Equal(t, content.Spot{Area: "street", At: [2]int{5, 5}, Face: "SE"}, s.Engine.Observe().Spot)
	assert.Equal(t, []string{"auto-scenario1"}, s.Store.Slots())
}
