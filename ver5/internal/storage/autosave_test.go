package storage

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"srpg/internal/core"
	"testing"
)

func TestAutosaveRotationAndManualIsolation(t *testing.T) {
	dir := t.TempDir()
	store := Store{Dir: dir}
	manual := core.State{Revision: 999}
	require.NoError(t, store.Save("1", manual, false))
	require.NoError(t, store.save("auto1", core.State{Phase: "scenario", Revision: 998}, false, 21))
	for revision := uint64(1); revision <= 12; revision++ {
		// A new store instance must continue the disk-backed rotation.
		store = Store{Dir: dir}
		require.NoError(t, store.Autosave(core.State{Phase: "battle", Revision: revision}))
		if revision <= 2 {
			require.NoError(t, store.Autosave(core.State{Phase: "scenario", Revision: revision + 20}))
		}
	}
	for slot, revision := range map[string]uint64{
		"auto-battle1": 11, "auto-battle2": 12, "auto-battle3": 3, "auto-battle10": 10,
		"auto-scenario1": 21, "auto-scenario2": 22, "auto1": 998, "1": 999,
	} {
		state, err := store.Load(slot)
		require.NoError(t, err)
		assert.Equal(t, revision, state.Revision)
	}
	assert.Len(t, store.Slots(), 14)
	// Reading an older save does not change the next slot.
	_, err := store.Load("auto-battle3")
	require.NoError(t, err)
	require.NoError(t, store.Autosave(core.State{Phase: "battle", Revision: 13}))
	state, err := store.Load("auto-battle3")
	require.NoError(t, err)
	assert.Equal(t, uint64(13), state.Revision)
}

func TestAutosaveFailurePreservesRotation(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	require.NoError(t, store.Autosave(core.State{Phase: "battle", Revision: 1}))
	blocked := filepath.Join(store.Dir, "slot-auto-battle2.json")
	require.NoError(t, os.Mkdir(blocked, 0700))
	require.Error(t, store.Autosave(core.State{Phase: "battle", Revision: 2}))
	first, err := store.Load("auto-battle1")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), first.Revision)
	require.NoError(t, os.Remove(blocked))
	require.NoError(t, store.Autosave(core.State{Phase: "battle", Revision: 2}))
	second, err := store.Load("auto-battle2")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), second.Revision)
	assert.False(t, store.Exists("auto-battle3"))
}
