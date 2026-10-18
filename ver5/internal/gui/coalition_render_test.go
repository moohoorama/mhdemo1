package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestRenderedHulaoCoalition(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for native rendering")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(d.StageIndex("B03"))
	g.editorTrial()
	require.NotNil(t, g.editorReturn)
	e := g.S.Engine
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	g.ensureField(e.Observe())
	g.zoom = 1.5
	x, y := g.field.Center(24, 4)
	g.camX, g.camY = x, y
	g.modals = nil
	out := "../../reports/hulao-coalition"
	require.NoError(t, os.MkdirAll(out, 0755))
	p := &coalitionRenderTest{g: g, out: out}
	result := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
	require.True(t, p.simultaneous, "both brothers must have visible bubbles at once")
}

type coalitionRenderTest struct {
	g                   *Game
	out                 string
	stage, ticks        int
	drawn, simultaneous bool
	seen                map[string]bool
	err                 error
}

func (p *coalitionRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *coalitionRenderTest) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	g := p.g
	if p.stage == 0 {
		if !p.drawn {
			return nil
		}
		snap := g.S.Engine.Snapshot()
		for i := range snap.Units {
			if snap.Units[i].Faction == "enemy" && snap.Units[i].ID != "lubu" {
				snap.Units[i].HP = 0
			}
		}
		var err error
		g.S.Engine, err = core.Restore(g.S.Data, snap)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		before := g.S.Engine.Observe()
		g.sync(before)
		c := core.Command{Kind: "wait", Actor: "liubei"}
		r, err := g.S.Engine.Apply(c)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		g.replayOne(before, c, r.Events)
		p.stage = 1
		p.seen = map[string]bool{}
	}
	if p.stage == 1 && !g.player.Busy() {
		snap := g.S.Engine.Snapshot()
		snap.Turn = "support"
		for i := range snap.Units {
			if snap.Units[i].ID == "mushun" {
				snap.Units[i].X, snap.Units[i].Y = 24, 4
			}
		}
		var err error
		g.S.Engine, err = core.Restore(g.S.Data, snap)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		before := g.S.Engine.Observe()
		g.sync(before)
		c := core.Command{Kind: "attack", Actor: "mushun", Target: "lubu"}
		r, err := g.S.Engine.Apply(c)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		g.replayOne(before, c, r.Events)
		p.stage = 2
	}
	if p.stage == 2 && g.player.Duel != nil {
		g.player.SkipDuel()
	}
	g.advance(1.0 / 60)
	p.ticks++
	if p.stage == 2 && !g.player.Busy() {
		return ebiten.Termination
	}
	if p.ticks > 3000 {
		p.err = fmt.Errorf("dialogue replay never finishes")
		return ebiten.Termination
	}
	return nil
}
func (p *coalitionRenderTest) Draw(dst *ebiten.Image) {
	g := p.g
	g.Draw(dst)
	if p.stage == 0 && !p.drawn {
		p.err = saveDuelFrame(dst, filepath.Join(p.out, "guard-camp.png"))
		p.drawn = true
	}
	if len(g.player.Speech) > 0 {
		key := g.player.Speech[0].actor
		if len(g.player.Speech) == 2 {
			key = "brothers"
			p.simultaneous = true
		}
		if !p.seen[key] {
			p.seen[key] = true
			p.err = saveDuelFrame(dst, filepath.Join(p.out, key+".png"))
		}
	}
}
