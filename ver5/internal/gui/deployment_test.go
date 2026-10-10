package gui

import (
	"fmt"
	"math"
	"srpg/internal/core"
	"srpg/internal/session"
	"testing"
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
func TestDeploymentCardsAndMap(t *testing.T) {
	g := deploymentController(t)
	if g.battleLogVisible() || g.battleViewport().Max.Y != deploymentTop {
		t.Fatal("battle HUD remained")
	}
	g.selectDeployment("hero")
	if g.deploymentSelected != "hero" || len(g.deploymentMarks()) == 0 {
		t.Fatal("card not selected")
	}
	g.selectDeployment("hero")
	if g.deploymentSelected != "" {
		t.Fatal("same card did not cancel")
	}
	g.selectDeployment("hero")
	g.mapClick(1, 2)
	if g.unitView(g.S.Engine.Observe(), "hero") == nil || g.deploymentSelected != "" || g.player.Busy() {
		t.Fatal("placement was not immediate")
	}
	g.mapClick(1, 2)
	if g.unitView(g.S.Engine.Observe(), "hero") != nil || g.deploymentSelected != "" {
		t.Fatal("map unplacement failed")
	}
	g.selectDeployment("hero")
	g.mapClick(1, 2)
	g.selectDeployment("archer1")
	g.mapClick(1, 2)
	if g.unitView(g.S.Engine.Observe(), "hero") != nil {
		t.Fatal("placed ally click should remove, never swap")
	}
	g.selectDeployment("hero")
	g.mapClick(1, 2)
	g.selectDeployment("archer1")
	g.mapClick(1, 3)
	g.selectDeployment("archer1")
	if g.unitView(g.S.Engine.Observe(), "archer1") != nil {
		t.Fatal("placed card did not remove")
	}
	g.selectDeployment("archer1")
	g.mapClick(1, 3)
	g.finishDeployment()
	if g.deploying() || !g.battleLogVisible() || g.battleViewport() != viewport || g.ord.actor != "" {
		t.Fatal("battle HUD not restored")
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
	g.selectDeployment("hero")
	g.mapClick(1, 2)
	if r := g.request(session.Request{Op: "save", Slot: "1"}); !r.OK {
		t.Fatal(r.Error)
	}
	g.selectDeployment("archer1")
	if r := g.request(session.Request{Op: "load", Slot: "1", Discard: true}); !r.OK {
		t.Fatal(r.Error)
	}
	if g.deploymentSelected != "" || !g.deploying() || g.unitView(g.S.Engine.Observe(), "hero") == nil {
		t.Fatal("partial load lost state")
	}
	old := cursorPos
	defer func() { cursorPos = old }()
	cursorPos = func() (int, int) { return 500, Height - 30 }
	g.hover = hoverCell{2, 2, true}
	g.deploymentSelected = "archer1"
	g.mapRelease()
	if g.unitView(g.S.Engine.Observe(), "archer1") != nil {
		t.Fatal("panel click reached map")
	}
	g.scrollDeployment(10000)
	if g.deploymentScroll != 0 {
		t.Fatal("short list scrolled outside")
	}
}

func TestLongDeploymentList(t *testing.T) {
	g := deploymentController(t)
	snap := g.S.Engine.Snapshot()
	g.S.Data.Stages[snap.Stage].Limit = 20
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
	g.scrollDeployment(100000)
	want := len(snap.Deployment)*deploymentCardWidth - g.deploymentCardsRect().Dx()
	if g.deploymentScroll != want {
		t.Fatalf("scroll %d want %d", g.deploymentScroll, want)
	}
	g.scrollDeployment(-100000)
	if g.deploymentScroll != 0 {
		t.Fatal("negative scroll")
	}
}
