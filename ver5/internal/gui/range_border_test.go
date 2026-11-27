package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestRangeBorderDuringOrder(t *testing.T) {
	g := battleController(t)
	g.threats = map[string]core.Ranges{}
	hero := g.unitView(g.S.View(), "hero")
	require.NotNil(t, hero)
	want := g.S.Engine.Ranges("hero").Strike
	require.NotEmpty(t, want)

	t.Run("none without a selection", func(t *testing.T) {
		assert.Empty(t, g.rangeBorder(g.S.View()))
	})
	g.begin(*hero)
	t.Run("the unit under orders rings what it strikes from its own cell", func(t *testing.T) {
		assert.Equal(t, want, g.rangeBorder(g.S.View()))
	})
	t.Run("after a tentative move it rings what it strikes from the new cell", func(t *testing.T) {
		require.NotEmpty(t, g.ord.moves)
		to := g.ord.moves[len(g.ord.moves)-1]
		g.tentative(to.X, to.Y)
		require.Equal(t, "menu", g.ord.stage)
		g.player.waiting, g.player.running = nil, nil
		assert.Equal(t, g.ord.sandbox.Ranges("hero").Strike, g.rangeBorder(g.S.View()))
		assert.NotEqual(t, want, g.rangeBorder(g.S.View()))
	})
	t.Run("choosing a target takes the ring away", func(t *testing.T) {
		g.ord.stage = "target"
		assert.Empty(t, g.rangeBorder(g.S.View()))
	})
}
