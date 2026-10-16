package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestTargetingCancelsOnOtherCell(t *testing.T) {
	g := battleController(t)
	u := g.unitView(g.S.Engine.Observe(), "hero")
	require.NotNil(t, u)
	g.ord = order{actor: "hero", stage: "target", pick: core.Command{Kind: "attack"}}
	g.mapClick(u.X+5, u.Y+5)
	assert.Equal(t, "menu", g.ord.stage, "a click on a cell that is not a target steps back to the menu")
	assert.Equal(t, "hero", g.ord.actor)
}
