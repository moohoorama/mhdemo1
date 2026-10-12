package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestRenderedAutosaveSlots(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for native rendering")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage == d.Stages[0].ID {
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
		for _, b := range g.buttons {
			if strings.HasPrefix(b.label, "자동 저장 ") {
				auto++
			}
			if strings.HasPrefix(b.label, "슬롯 ") {
				manual++
			}
			if b.rect.Min.Y < 0 || b.rect.Max.Y > Height {
				p.err = fmt.Errorf("save button outside screen")
				return ebiten.Termination
			}
		}
		if auto != 4 || manual != 1 {
			p.err = fmt.Errorf("slots: %d automatic, %d manual", auto, manual)
			return ebiten.Termination
		}
		g.openOverlay("save")
	case 1:
		manual := 0
		for _, b := range g.buttons {
			if strings.HasPrefix(b.label, "슬롯 ") {
				manual++
			}
			if strings.HasPrefix(b.label, "자동 저장") {
				p.err = fmt.Errorf("automatic slots writable in manual menu")
				return ebiten.Termination
			}
		}
		if manual != 10 {
			p.err = fmt.Errorf("manual save slots: %d", manual)
			return ebiten.Termination
		}
		g.openOverlay("load")
	case 2:
		for _, b := range g.buttons {
			if strings.HasPrefix(b.label, "자동 저장 4 ") {
				if b.click == nil {
					p.err = fmt.Errorf("autosave cannot be loaded")
					return ebiten.Termination
				}
				b.click()
				p.step++
				return nil
			}
		}
		p.err = fmt.Errorf("missing automatic slot 4")
		return ebiten.Termination
	case 3:
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
	if p.step < 2 {
		name := []string{"load", "save"}[p.step]
		p.err = saveDuelFrame(dst, filepath.Join(p.out, name+".png"))
	}
	p.drawn = true
}
