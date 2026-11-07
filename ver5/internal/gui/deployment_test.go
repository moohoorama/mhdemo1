package gui

import (
	"fmt"
	"image"
	"math"
	"srpg/internal/core"
	"srpg/internal/session"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deploymentController(t *testing.T) *Game {
	t.Helper()
	g := battleController(t)
	cp := g.S.Engine.Snapshot().Checkpoint
	e, err := core.Restore(g.S.Data, *cp)
	if err != nil {
		t.Fatal(err)
	}
	g.S.Engine = e
	if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "start"}}); !r.OK {
		t.Fatal(r.Error)
	}
	return g
}
func cellOf(g *Game, id string) [2]int {
	u := g.unitView(g.S.Engine.Observe(), id)
	if u == nil {
		return [2]int{-1, -1}
	}
	return [2]int{u.X, u.Y}
}
func TestDeploymentCardsAndMap(t *testing.T) {
	g := deploymentController(t)
	require.False(t, g.battleLogVisible())
	require.Equal(t, image.Rect(0, 0, worldW, worldH), g.battleViewport(), "the map fills the screen under the panels")
	t.Run("the sortie stands on the field at once", func(t *testing.T) {
		assert.Equal(t, [2]int{1, 3}, cellOf(g, "hero"))
		assert.Equal(t, [2]int{1, 2}, cellOf(g, "archer1"))
	})
	t.Run("a map click selects and deselects", func(t *testing.T) {
		g.mapClick(1, 3)
		assert.Equal(t, "hero", g.deploymentSelected)
		assert.NotEmpty(t, g.deploymentMarks())
		g.mapClick(1, 3)
		assert.Empty(t, g.deploymentSelected)
	})
	t.Run("the selected officer moves to an empty cell", func(t *testing.T) {
		g.mapClick(1, 3)
		g.mapClick(2, 2)
		assert.Equal(t, [2]int{2, 2}, cellOf(g, "hero"))
		assert.Equal(t, "hero", g.deploymentSelected)
		assert.False(t, g.player.Busy())
	})
	t.Run("another ally's cell picks that ally", func(t *testing.T) {
		g.mapClick(1, 2)
		assert.Equal(t, [2]int{2, 2}, cellOf(g, "hero"))
		assert.Equal(t, [2]int{1, 2}, cellOf(g, "archer1"))
		assert.Equal(t, "archer1", g.deploymentSelected)
		for _, m := range g.deploymentMarks()[1:] {
			assert.NotEqual(t, [2]int{2, 2}, [2]int{m.U, m.V}, "no ally's cell is lit as a move")
		}
	})
	t.Run("the sortie is listed at the bottom right", func(t *testing.T) {
		r := g.deploymentListRect(g.S.View())
		assert.True(t, r.In(image.Rect(0, 0, uiW, uiH)), "on screen: %v", r)
		assert.Equal(t, uiH-12, r.Max.Y)
		assert.GreaterOrEqual(t, r.Dy(), 48+len(g.S.View().Deployment)*touchH)
	})
	g.finishDeployment()
	if g.deploying() || !g.battleLogVisible() || g.battleViewport() != image.Rect(0, 0, worldW, worldH) || g.ord.actor != "" {
		t.Fatal("battle HUD not restored")
	}
}

func TestDeploymentPreview(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*Game)
		visible bool
	}{
		{"legal empty cell", func(g *Game) {}, true},
		{"no selection", func(g *Game) { g.deploymentSelected = "" }, false},
		{"outside map or HUD", func(g *Game) { g.hover.ok = false }, false},
		{"outside deployment area", func(g *Game) { g.hover = hoverCell{0, 0, true} }, false},
		{"an ally's cell", func(g *Game) { g.hover = hoverCell{1, 2, true} }, false},
		{"overlay", func(g *Game) { g.overlays = []string{"menu"} }, false},
		{"modal", func(g *Game) { g.modals = []*modal{{kind: "objective"}} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := deploymentController(t)
			g.selectDeployment("hero", false)
			g.hover = hoverCell{2, 2, true}
			tc.change(g)
			c, ok := g.deploymentPreviewCommand()
			require.Equal(t, tc.visible, ok)
			if ok {
				assert.Equal(t, core.Command{Kind: "reposition", Actor: "hero", X: 2, Y: 2}, c)
			}
		})
	}
}
func TestDeploymentPulseAndViewport(t *testing.T) {
	for _, speed := range []int{1, 2, 4} {
		g := deploymentController(t)
		g.speed = speed
		for i, want := range []float64{.21, .30, .21, .12, .21} {
			if math.Abs(deploymentPulse(g.uiSeconds)-want) > 1e-8 {
				t.Fatalf("speed %d quarter %d", speed, i)
			}
			g.advance(.25)
		}
		g.zoom = 1.5
		g.camX = 100
		g.camY = 200
		x, y := g.toScreen(110, 215)
		cx, cy := g.toCanvas(x, y)
		if math.Abs(cx-110) > 1e-8 || math.Abs(cy-215) > 1e-8 {
			t.Fatal("viewport transform mismatch")
		}
	}
}
func TestDeploymentLoadAndPanel(t *testing.T) {
	g := deploymentController(t)
	g.mapClick(1, 3)
	g.mapClick(2, 2)
	if r := g.request(session.Request{Op: "save", Slot: "1"}); !r.OK {
		t.Fatal(r.Error)
	}
	g.selectDeployment("archer1", false)
	if r := g.request(session.Request{Op: "load", Slot: "1", Discard: true}); !r.OK {
		t.Fatal(r.Error)
	}
	if g.deploymentSelected != "" || !g.deploying() || cellOf(g, "hero") != [2]int{2, 2} {
		t.Fatal("load lost the placement")
	}
	old := cursorPos
	defer func() { cursorPos = old }()
	list := g.deploymentListRect(g.S.View())
	g.block(list)
	cursorPos = func() (int, int) { return list.Min.X + 40, list.Max.Y - 30 }
	g.hover = hoverCell{1, 3, true}
	g.deploymentSelected = "archer1"
	g.mapRelease()
	if cellOf(g, "archer1") != [2]int{1, 2} {
		t.Fatal("panel click reached map")
	}
}

func TestLongDeploymentList(t *testing.T) {
	g := deploymentController(t)
	snap := g.S.Engine.Snapshot()
	g.S.Data.Stages[snap.Stage].Max = 20
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("reserve%d", i)
		def := g.S.Data.Characters["archer1"]
		def.Lord = false
		g.S.Data.Characters[id] = def
		c := snap.Characters[0]
		c.ID, c.Def, c.Class = id, id, def.Class
		c.Equipment = map[string]string{}
		snap.Characters = append(snap.Characters, c)
		snap.Checkpoint.Characters = append(snap.Checkpoint.Characters, c)
		snap.Deployment = append(snap.Deployment, id)
	}
	e, err := core.Restore(g.S.Data, snap)
	if err != nil {
		t.Fatal(err)
	}
	g.S.Engine = e
	r := g.deploymentListRect(g.S.View())
	assert.GreaterOrEqual(t, r.Min.Y, hudTop()+64+12, "a long sortie stops under the top-right bar and scrolls")
	assert.Equal(t, uiH-12, r.Max.Y)
}

func TestStartSortie(t *testing.T) {
	g := battleController(t)
	cp := g.S.Engine.Snapshot().Checkpoint
	e, err := core.Restore(g.S.Data, *cp)
	require.NoError(t, err)
	g.S.Engine = e
	g.S.Data.Stages[cp.Stage].Max = 1
	require.Equal(t, "scenario", g.S.View().Phase)
	g.startSortie()
	require.True(t, g.deploying())
	o := g.S.View()
	assert.Equal(t, []string{"hero"}, o.Deployment, "trimmed to the stage's limit, the lord kept")
	for _, id := range o.Deployment {
		assert.NotNil(t, g.unitView(o, id), "%s stands on the field", id)
	}
	rev := o.Revision
	g.startSortie()
	assert.Equal(t, rev, g.S.View().Revision, "tried once per battle node")
}
