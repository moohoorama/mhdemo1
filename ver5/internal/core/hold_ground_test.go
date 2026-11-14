package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestHoldGroundHalvesBlowsAfterAStill(t *testing.T) {
	setup := func(t *testing.T) (*Engine, *Unit, *Unit, float64) {
		e, a, b := arena(t)
		e.data.Traits["hold_t"] = content.Trait{Effects: map[string]float64{content.FxHoldGround: .5}}
		with(e, "hero", "hold_t")
		return e, a, b, e.damage(b, a, e.basic())
	}

	t.Run("a turn ended without walking halves the blows on it", func(t *testing.T) {
		e, a, b, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		assert.InDelta(t, raw/2, e.damage(b, a, e.basic()), 1e-6)
	})

	t.Run("a turn that walked leaves them whole", func(t *testing.T) {
		e, a, b, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "move", Actor: a.ID, X: 5, Y: 7}))
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		assert.InDelta(t, raw, e.damage(b, a, e.basic()), 1e-6)
	})

	t.Run("it lasts through the other side's turn and ends when its own begins", func(t *testing.T) {
		e, a, b, raw := setup(t)
		require.NoError(t, e.execute(Command{Kind: "wait", Actor: a.ID}))
		e.endFaction()
		require.NotEqual(t, "ally", e.state.Turn)
		assert.InDelta(t, raw/2, e.damage(b, a, e.basic()), 1e-6)
		for e.state.Turn != "ally" {
			e.endFaction()
		}
		assert.Zero(t, a.Status[holdGround].Turns)
	})
}
