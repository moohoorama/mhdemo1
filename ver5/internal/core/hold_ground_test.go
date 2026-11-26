package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestHoldGroundRaisesDefenseAfterAStill(t *testing.T) {
	setup := func(t *testing.T) (*Engine, *Unit, int) {
		e, a, _ := arena(t)
		e.data.Traits["hold_t"] = content.Trait{Effects: map[string]float64{content.FxHoldGround: .5}}
		with(e, "hero", "hold_t")
		return e, a, e.stats(a).Defense
	}

	t.Run("a turn ended without walking raises the defense", func(t *testing.T) {
		e, a, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		assert.Equal(t, int(float64(raw)*1.5), e.stats(a).Defense)
	})

	t.Run("a state holding it restores", func(t *testing.T) {
		e, a, _ := setup(t)
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		require.Positive(t, a.Status[holdGround].Turns)
		_, err := Restore(e.data, e.Snapshot())
		assert.NoError(t, err)
	})

	t.Run("a turn that walked leaves it as it was", func(t *testing.T) {
		e, a, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "move", Actor: a.ID, X: 5, Y: 7}))
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		assert.Equal(t, raw, e.stats(a).Defense)
	})

	t.Run("it lasts through the other side's turn and ends when its own begins", func(t *testing.T) {
		e, a, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		e.endFaction()
		require.NotEqual(t, "ally", e.state.Turn)
		assert.Equal(t, int(float64(raw)*1.5), e.stats(a).Defense)
		for e.state.Turn != "ally" {
			e.endFaction()
		}
		assert.Zero(t, a.Status[holdGround].Turns)
		assert.Equal(t, raw, e.stats(a).Defense)
	})

	t.Run("it adds to a defense buff instead of multiplying with it", func(t *testing.T) {
		e, a, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		a.Status["defense"] = Status{Turns: 2, Value: 20}
		assert.Equal(t, int(float64(raw)*1.7), e.stats(a).Defense)
	})
}
