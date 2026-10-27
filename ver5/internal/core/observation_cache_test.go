package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservationCache(t *testing.T) {
	t.Run("builds once while the state stands", func(t *testing.T) {
		e := battle(t)
		var c ObservationCache
		first := c.Of(e)
		again := c.Of(e)
		require.NotEmpty(t, first.UnitViews)
		assert.Same(t, &first.UnitViews[0], &again.UnitViews[0])
	})
	t.Run("rebuilds after a command", func(t *testing.T) {
		e := battle(t)
		var c ObservationCache
		before := c.Of(e)
		_, err := e.Apply(Command{Kind: "wait", Actor: "hero"})
		require.NoError(t, err)
		after := c.Of(e)
		assert.Equal(t, before.Revision+1, after.Revision)
		assert.Equal(t, e.Observe(), after)
	})
	t.Run("rebuilds after a setting", func(t *testing.T) {
		e := battle(t)
		var c ObservationCache
		c.Of(e)
		e.SetDamagePct(50)
		assert.Equal(t, 50, c.Of(e).DamagePct)
		e.SetXPPct(70)
		assert.Equal(t, 70, c.Of(e).XPPct)
		e.SetCounterAll(true)
		assert.True(t, c.Of(e).CounterAll)
	})
	t.Run("rebuilds for another engine", func(t *testing.T) {
		e := battle(t)
		var c ObservationCache
		c.Of(e)
		other, err := Restore(e.Data(), e.Snapshot())
		require.NoError(t, err)
		other.SetDamagePct(40)
		assert.Equal(t, 40, c.Of(other).DamagePct)
	})
}

// An observation shares the engine's state instead of copying it, so commands and settings after it
// must leave it as it was.
func TestObservationStaysAfterChanges(t *testing.T) {
	e, _, _ := arena(t)
	o := e.Observe()
	want := encoded(o)
	apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	e.SetDamagePct(50)
	e.SetCounterAll(true)
	assert.NotEqual(t, want, encoded(e.Observe()))
	assert.Equal(t, want, encoded(o))
}
