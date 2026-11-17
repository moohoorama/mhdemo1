package autoplay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestFrontierProgressObservesRealEnemyDamage(t *testing.T) {
	d, e := battleModelFixture(t)
	before := e.Snapshot()
	initial := frontierProgress(e)
	assert.Equal(t, before, e.Snapshot())
	damaged := e.Snapshot()
	found := false
	for i := range damaged.Units {
		if damaged.Units[i].Faction == "enemy" {
			damaged.Units[i].HP /= 2
			found = true
		}
	}
	require.True(t, found)
	other, err := core.Restore(d, damaged)
	require.NoError(t, err)
	progress := frontierProgress(other)
	assert.Greater(t, progress, initial)
	assert.GreaterOrEqual(t, progress, 0.0)
	assert.LessOrEqual(t, progress, 1.0)
	assert.Less(t, .25*progress, 1.0, "losing progress must remain below one real stage win")
	assert.Equal(t, before, e.Snapshot())
}
