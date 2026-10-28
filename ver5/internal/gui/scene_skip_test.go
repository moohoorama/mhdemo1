package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/scenes"
	"srpg/internal/session"
)

// storyAt is a game at a chapter's first node with the scene book, its scene laid out without a map.
func storyAt(t *testing.T, node string) *Game {
	t.Helper()
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	book, err := scenes.Load("../../assets/scenes")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g := &Game{S: s, scenes: book}
	r := g.send(session.Request{Op: "jump", Seed: 1, Node: node})
	require.True(t, r.OK, r.Error)
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		r := g.send(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
		require.True(t, r.OK, r.Error)
	}
	g.stage(t)
	return g
}

// stage stands in for enterScene: the node's scene, empty, as the screen would have opened it.
func (g *Game) stage(t *testing.T) {
	o := g.S.View()
	def, _ := g.scenes.For(o.Node)
	require.NotNil(t, def, o.Node)
	if g.scene == nil || g.scene.def != def {
		g.scene = &scene{def: def, actors: map[string]*unitVis{}, paths: map[string][][2]float64{}, plays: map[string]string{}, emotes: map[string]*emote{}}
	}
	g.scene.node = o.Node
	g.startLine(0)
}

func TestSkipScene(t *testing.T) {
	t.Run("stops on the first line of the next scene", func(t *testing.T) {
		g := storyAt(t, "c03-start")
		g.advanceDialogue()
		g.skipping = g.scene.def
		g.turnPage()
		o := g.S.View()
		def, at := g.scenes.For(o.Node)
		require.NotNil(t, def)
		assert.Equal(t, "c03-road", def.ID)
		assert.Equal(t, 1, at, "the scene node itself passes on its own; the first line is the next")
		assert.Nil(t, g.skipping)
	})
	t.Run("a second press right away does nothing", func(t *testing.T) {
		g := storyAt(t, "c03-start")
		g.skipping = g.scene.def
		g.turnPage()
		g.stage(t)
		require.True(t, g.sceneBusy(), "the next scene's first beats are playing")
		g.advanceDialogue()
		assert.True(t, g.sceneBusy(), "a click right after the skip is held")
		g.uiSeconds += skipGuard
		g.advanceDialogue()
		assert.False(t, g.sceneBusy(), "clicks work again once the guard is over")
	})
	t.Run("stops before the battle", func(t *testing.T) {
		g := storyAt(t, "c04-start")
		g.skipping = g.scene.def
		g.turnPage()
		g.stage(t)
		g.uiSeconds += skipGuard
		g.skipping = g.scene.def
		g.turnPage()
		o := g.S.View()
		assert.Equal(t, "battle", o.Dialogue.Kind)
		assert.Equal(t, "c04-battle", o.Node)
	})
}
