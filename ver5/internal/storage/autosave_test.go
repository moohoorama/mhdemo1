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
	for revision := uint64(1); revision <= 6; revision++ {
		// A new store instance must continue the disk-backed rotation.
		store = Store{Dir: dir}
		require.NoError(t, store.Autosave(core.State{Revision: revision}))
	}
	for slot, revision := range map[string]uint64{"auto1": 5, "auto2": 6, "auto3": 3, "auto4": 4, "1": 999} {
		state, err := store.Load(slot)
		require.NoError(t, err)
		assert.Equal(t, revision, state.Revision)
	}
	assert.Len(t, store.Slots(), 5)
	// Reading an older save does not change the next slot.
	_, err := store.Load("auto3")
	require.NoError(t, err)
	require.NoError(t, store.Autosave(core.State{Revision: 7}))
	state, err := store.Load("auto3")
	require.NoError(t, err)
	assert.Equal(t, uint64(7), state.Revision)
}

func TestAutosaveFailurePreservesRotation(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	require.NoError(t, store.Autosave(core.State{Revision: 1}))
	blocked := filepath.Join(store.Dir, "slot-auto2.json")
	require.NoError(t, os.Mkdir(blocked, 0700))
	require.Error(t, store.Autosave(core.State{Revision: 2}))
	first, err := store.Load("auto1")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), first.Revision)
	require.NoError(t, os.Remove(blocked))
	require.NoError(t, store.Autosave(core.State{Revision: 2}))
	second, err := store.Load("auto2")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), second.Revision)
	assert.False(t, store.Exists("auto3"))
}
