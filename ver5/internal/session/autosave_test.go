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
		s.Data.Stages[0].Enemies[i].Guard = true
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
			first, err := s.Store.Load("auto1")
			require.NoError(t, err)
			assert.Equal(t, "battle", first.Phase)
			assert.Equal(t, "ally", first.Turn)
			assert.Equal(t, 1, first.Round)
			request(t, s, Request{Op: "command", Command: core.Command{Kind: "wait", Actor: "hero"}})
			request(t, s, Request{Op: "observe"})
			assert.Len(t, s.Store.Slots(), 1)
			request(t, s, Request{Op: "save", Slot: "1"})
			for round := 2; round <= 6; round++ {
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
			for slot, round := range map[string]int{"auto1": 5, "auto2": 6, "auto3": 3, "auto4": 4} {
				state, err := s.Store.Load(slot)
				require.NoError(t, err)
				assert.Equal(t, round, state.Round)
				assert.Equal(t, "ally", state.Turn)
				for _, u := range state.Units {
					if u.ActsIn(state.Turn) {
						assert.False(t, u.Done)
						assert.False(t, u.Acted)
						assert.False(t, u.Moved)
					}
				}
			}
			manual, err := s.Store.Load("1")
			require.NoError(t, err)
			assert.Equal(t, 1, manual.Round)
			request(t, s, Request{Op: "load", Slot: "auto3", Discard: true})
			assert.Len(t, s.Store.Slots(), 5)
			request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
			_, err = s.ResolvePhase()
			require.NoError(t, err)
			latest, err := s.Store.Load("auto3")
			require.NoError(t, err)
			assert.Equal(t, 4, latest.Round)
		})
	}
}

func TestAutosaveSlotsAreReadOnlyForManualSave(t *testing.T) {
	s := autosaveBattle(t)
	before, err := s.Store.Load("auto1")
	require.NoError(t, err)
	r := s.Handle(Request{Op: "save", Slot: "auto1", Overwrite: true})
	assert.False(t, r.OK)
	assert.Equal(t, "InvalidSlot", r.Error)
	after, err := s.Store.Load("auto1")
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestAutosaveFailureKeepsAppliedCommand(t *testing.T) {
	s := autosaveBattle(t)
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
	// Block the next automatic slot; the resolved phase must still reach the new round.
	require.NoError(t, os.Mkdir(filepath.Join(s.Store.Dir, "slot-auto2.json"), 0700))
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
	s.Data.Stages[0].Party = []content.Spawn{{Character: "troop", X: 1, Y: 4, AI: true, Guard: true, Level: 1}}
	for i := range s.Data.Stages[0].Enemies {
		s.Data.Stages[0].Enemies[i].Guard = true
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
