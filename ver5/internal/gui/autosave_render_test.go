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
	"srpg/internal/storage"
)

func TestRenderedAutosaveSlots(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for native rendering")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == d.Stages[0].ID {
			d.Campaign.Start = n.ID
			break
		}
	}
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	s.Engine = core.New(d, 17)
	s.Screen = "playing"
	_, err = s.Engine.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := s.Engine.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = s.Engine.Apply(c)
		require.NoError(t, err)
	}
	for round := 1; round <= 4; round++ {
		state := s.Engine.Snapshot()
		state.Round = round
		require.NoError(t, s.Store.Autosave(state))
	}
	require.NoError(t, s.Store.Save("1", s.Engine.Snapshot(), false))
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	g.openOverlay("load")
	out := "../../reports/autosave"
	require.NoError(t, os.MkdirAll(out, 0755))
	p := &autosaveRenderTest{g: g, out: out}
	result := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
	assertRound := g.S.Engine.Observe().Round
	require.Equal(t, 4, assertRound)
}

type autosaveRenderTest struct {
	g     *Game
	out   string
	step  int
	drawn bool
	err   error
}

func (p *autosaveRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *autosaveRenderTest) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	if !p.drawn {
		return nil
	}
	p.drawn = false
	g := p.g
	switch p.step {
	case 0:
		auto, manual := 0, 0
		for _, r := range g.saves.rows {
			if r.ok && storage.IsAutoSlot(r.slot) {
				auto++
			}
			if r.ok && storage.IsManualSlot(r.slot) {
				manual++
			}
		}
		if auto != 4 || manual != 1 || g.saves.auto || g.saves.pick != "1" {
			p.err = fmt.Errorf("slots: %d automatic, %d manual, auto tab %t, pick %s", auto, manual, g.saves.auto, g.saves.pick)
			return ebiten.Termination
		}
		g.openOverlay("save")
	case 1:
		if g.saves.auto {
			p.err = fmt.Errorf("automatic slots writable in manual menu")
			return ebiten.Termination
		}
		g.openOverlay("load")
	case 2:
		g.showSaveTab(true)
		g.saves.pick = "auto4"
		p.step++
		return nil
	case 3:
		for _, b := range g.buttons {
			if b.label == "확인" && b.click != nil {
				b.click()
				p.step++
				return nil
			}
		}
		p.err = fmt.Errorf("automatic slot 4 cannot be loaded")
		return ebiten.Termination
	case 4:
		return ebiten.Termination
	}
	p.step++
	return nil
}
func (p *autosaveRenderTest) Draw(dst *ebiten.Image) {
	if p.drawn {
		return
	}
	p.g.Draw(dst)
	if name, ok := map[int]string{0: "load", 1: "save", 3: "load-auto"}[p.step]; ok {
		p.err = saveDuelFrame(dst, filepath.Join(p.out, name+".png"))
	}
	p.drawn = true
}
