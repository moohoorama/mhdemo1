package gui

import (
	"srpg/internal/sprite"
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
		require.NotEmpty(t, g.modals, "간옹 joins on the way: his card holds the skip")
		assert.Equal(t, gain{id: "jianyong", n: 1, officer: true}, g.modals[0].gains[0])
		g.closeModal()
		g.turnPage()
		o := g.S.View()
		assert.Equal(t, "battle", o.Dialogue.Kind)
		assert.Equal(t, "c04-battle", o.Node)
	})
}

// storyNode is a game standing on any dialogue node of the story, its scene laid out without a map.
func storyNode(t *testing.T, node string) *Game {
	t.Helper()
	g := storyAt(t, "c03-start")
	st := g.S.Engine.Snapshot()
	st.Phase, st.Node, st.Explore = "scenario", node, ""
	e, err := core.Restore(g.S.Data, st)
	require.NoError(t, err)
	g.S.Engine, g.scene = e, nil
	g.stage(t)
	return g
}

func TestSceneNoSkip(t *testing.T) {
	t.Run("a click waits for a NoSkip node's beats", func(t *testing.T) {
		g := storyNode(t, "c19-yj11")
		require.Contains(t, g.scene.def.NoSkip, "c19-yj11")
		require.True(t, g.sceneBusy())
		g.advanceDialogue()
		assert.True(t, g.sceneBusy(), "관우's charge on 문추 plays out whole")
	})
	t.Run("a click still hurries other beats", func(t *testing.T) {
		g := storyNode(t, "c19-yj05")
		require.True(t, g.sceneBusy())
		g.advanceDialogue()
		assert.False(t, g.sceneBusy())
	})
}

func TestSceneKeepInView(t *testing.T) {
	newGame := func(zoom float64) *Game {
		g := &Game{field: &Field{Map: &MapArt{}}, zoom: zoom}
		g.scene = &scene{def: &scenes.Scene{}, actors: map[string]*unitVis{"관우": {u: 6, v: 68, top: 40}}}
		return g
	}
	t.Run("an actor already on screen leaves the shot as framed", func(t *testing.T) {
		g := newGame(2)
		g.scene.focus = "관우"
		x, y := g.field.Center(6, 67)
		tx, ty := g.keepInView(x, y)
		assert.Equal(t, x, tx)
		assert.Equal(t, y, ty)
	})
	t.Run("an actor outside the still zone is brought to its centre", func(t *testing.T) {
		g := newGame(4)
		g.scene.focus = "관우"
		x, y := g.field.Center(5, 50)
		tx, ty := g.keepInView(x, y)
		ax, ay := g.field.Center(6, 68)
		assert.Equal(t, ax, tx)
		assert.InDelta(t, ay-20, ty, 1, "the middle of his body, when the scroll leaves room")
	})
	t.Run("a shot brought to the actor stays while they move inside the zone", func(t *testing.T) {
		g := newGame(4)
		g.scene.focus = "관우"
		x, y := g.field.Center(5, 50)
		held, _ := g.keepInView(x, y)
		g.scene.actors["관우"].u = 7
		tx, _ := g.keepInView(x, y)
		assert.Equal(t, held, tx, "the camera does not swing back to the authored shot")
	})
	t.Run("no focus keeps the shot", func(t *testing.T) {
		g := newGame(4)
		tx, ty := g.keepInView(10, 20)
		assert.Equal(t, 10.0, tx)
		assert.Equal(t, 20.0, ty)
	})
}

// A NoFocus node's beats leave the camera on the shot it had ("문추, 네 이놈!" in c19-yj10b does not swing to 관우
// riding in and back); other nodes follow their actors.
func TestSceneNoFocus(t *testing.T) {
	g := storyNode(t, "c19-yj10b")
	require.Contains(t, g.scene.def.NoFocus, "c19-yj10b")
	s := &scene{def: g.scene.def, node: "c19-yj10b", actors: map[string]*unitVis{"관우": {u: 6, v: 68, art: &sprite.Art{}}, "문추": {u: 6, v: 40, art: &sprite.Art{}}},
		extra: map[string]bool{}, emotes: map[string]*emote{}, looks: map[string]*look{}, plays: map[string]string{},
		paths: map[string][][2]float64{}, speed: map[string]float64{}}
	g.scene, s.focus = s, "문추"
	g.startStep(scenes.Step{Actor: "관우", Emote: "!"})
	assert.Equal(t, "문추", s.focus, "관우's beat does not draw the camera")

	s.node = "c19-yj11"
	g.startStep(scenes.Step{Actor: "관우", Emote: "!"})
	assert.Equal(t, "관우", s.focus, "an ordinary node follows its actors")
}
