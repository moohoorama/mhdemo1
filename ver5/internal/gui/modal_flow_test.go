package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestEndPromptWaitsForLevelUpAndLearning(t *testing.T) {
	g := battleController(t)
	state := g.S.Engine.Snapshot()
	for i := range state.Units {
		if state.Units[i].Faction == "ally" {
			state.Units[i].Done, state.Units[i].Acted, state.Units[i].Moved = true, true, true
		}
	}
	for i := range state.Characters {
		if state.Characters[i].ID == "hero" {
			state.Characters[i].Points = 100
		}
	}
	state.Learning = []string{"hero"}
	var err error
	g.S.Engine, err = core.Restore(g.S.Data, state)
	require.NoError(t, err)
	require.NotEmpty(t, g.S.Engine.Observe().Learners)
	g.modals = []*modal{{kind: "levelup"}, {kind: "levelup"}}
	for i := 0; i < 2; i++ {
		g.offerEnd(g.S.Engine.Observe())
		assert.Empty(t, g.overlays)
		assert.Empty(t, g.endAsked)
		assert.False(t, g.openOverlay("menu"))
		g.closeModal()
	}
	// Check authoritative engine learning even before the cached UI list refreshes.
	g.offerEnd(g.S.Engine.Observe())
	assert.Empty(t, g.overlays)
	assert.Empty(t, g.endAsked)
	assert.False(t, g.askWith("blocked", "yes", "no", nil))
	before := g.S.Engine.Observe()
	g.endPhase()
	assert.Equal(t, before.Revision, g.S.Engine.Observe().Revision)
	_, err = g.S.Engine.Apply(core.Command{Kind: "learn_close", Actor: "hero"})
	require.NoError(t, err)
	g.offerEnd(g.S.Engine.Observe())
	require.Equal(t, "confirm", g.overlay())
	require.NotEmpty(t, g.endAsked)
	require.NotNil(t, g.answer)
	g.closeOverlay()
	g.answer()
	assert.Equal(t, "enemy", g.S.Engine.Observe().Turn)
}

func TestEndPromptDeclineIsOncePerTurn(t *testing.T) {
	g := battleController(t)
	o := g.S.Engine.Observe()
	for i := range o.UnitViews {
		if o.UnitViews[i].Faction == "ally" {
			o.UnitViews[i].Done = true
		}
	}
	g.offerEnd(o)
	require.Equal(t, "confirm", g.overlay())
	g.closeOverlay()
	g.offerEnd(o)
	assert.Empty(t, g.overlays)
}

func TestOverlayClosesBackToTheOneBeneath(t *testing.T) {
	g := battleController(t)
	require.True(t, g.openOverlay("party"))
	require.True(t, g.openOverlay("equipment"))
	assert.Equal(t, "equipment", g.overlay())

	t.Run("close returns to the window beneath", func(t *testing.T) {
		g.closeOverlay()
		assert.Equal(t, "party", g.overlay())
	})
	t.Run("escape and right click close one at a time", func(t *testing.T) {
		require.True(t, g.openOverlay("deputy"))
		g.rightClick()
		assert.Equal(t, "party", g.overlay())
		g.rightClick()
		assert.Empty(t, g.overlays)
	})
	t.Run("a confirmation answered returns to its window", func(t *testing.T) {
		require.True(t, g.openOverlay("menu"))
		require.True(t, g.askWith("?", "예", "아니오", func() {}))
		g.closeOverlay()
		assert.Equal(t, "menu", g.overlay())
		g.closeOverlays()
		assert.Empty(t, g.overlays)
	})
}
