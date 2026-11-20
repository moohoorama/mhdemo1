package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestQuickAttack(t *testing.T) {
	g, actor, move := moveGame(t)
	state := g.S.Engine.Snapshot()
	var enemy string
	for i := range state.Units {
		u := &state.Units[i]
		if u.Faction != actor.Faction && u.HP > 0 {
			enemy = u.ID
			u.X, u.Y = move.X, move.Y
			break
		}
	}
	require.NotEmpty(t, enemy)
	var err error
	g.S.Engine, err = core.Restore(g.S.Data, state)
	require.NoError(t, err)
	g.begin(actor)
	target := g.unitView(g.S.View(), enemy)
	_, ok := g.ord.attackFrom[enemy]
	require.True(t, ok, "enemy is reachable with a basic attack")
	g.mapClick(target.X, target.Y)
	require.Equal(t, "target", g.ord.stage)
	assert.Equal(t, "attack", g.ord.pick.Kind)
	assert.NotEmpty(t, g.options(g.ord.pick))
	assert.True(t, g.cancel())
	assert.Equal(t, "menu", g.ord.stage)
	assert.True(t, g.cancel())
	assert.Equal(t, "move", g.ord.stage)
	assert.Equal(t, actor.X, g.unitView(g.view(), actor.ID).X)
	assert.Equal(t, actor.Y, g.unitView(g.view(), actor.ID).Y)
}
