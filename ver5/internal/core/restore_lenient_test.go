package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Run with and without -tags release: a rule mismatch fails a development build and only warns a release one,
// while a broken shape fails both.
func TestARestoreFailsOrWarnsByTheBuild(t *testing.T) {
	var heard []string
	RestoreWarned = func(w []string, _ State) { heard = append(heard, w...) }
	t.Cleanup(func() { RestoreWarned = nil })

	t.Run("a status the list lacks", func(t *testing.T) {
		heard = nil
		snap := battle(t).Snapshot()
		snap.Units[0].Status["no_such"] = Status{Turns: 1}
		_, err := Restore(battle(t).Data(), snap)
		if lenientRestore {
			require.NoError(t, err)
			require.Len(t, heard, 1)
			assert.Contains(t, heard[0], `status "no_such"`)
		} else {
			assert.ErrorContains(t, err, `CorruptSave: unit `+snap.Units[0].ID+`: status "no_such"`)
			assert.Empty(t, heard)
		}
	})

	t.Run("a unit off the map", func(t *testing.T) {
		snap := battle(t).Snapshot()
		snap.Units[0].X = -1
		_, err := Restore(battle(t).Data(), snap)
		assert.ErrorContains(t, err, "off the")
	})
}
