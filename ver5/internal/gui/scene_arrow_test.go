package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSceneArrowLifecycle(t *testing.T) {
	g := &Game{}
	g.scene = &scene{}
	a := &sceneArrow{fu: 3, fv: 0, tu: 5, tv: -5, delay: .2, dur: .5}
	g.scene.arrows = []*sceneArrow{a}
	t.Run("waits for the bow to be let go", func(t *testing.T) {
		g.updateArrows(.1)
		require.Len(t, g.scene.arrows, 1)
		assert.False(t, a.loosed)
	})
	t.Run("flies after the release", func(t *testing.T) {
		g.updateArrows(.2)
		require.Len(t, g.scene.arrows, 1)
		assert.True(t, a.loosed)
	})
	t.Run("lands and goes", func(t *testing.T) {
		g.updateArrows(.6)
		assert.Empty(t, g.scene.arrows)
	})
}

func TestSceneWithoutArrowsHasNone(t *testing.T) {
	g := &Game{scene: &scene{}}
	g.updateArrows(1)
	assert.Empty(t, g.scene.arrows)
}
