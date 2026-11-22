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
	story := core.New(d, 17).Snapshot()
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
	require.NoError(t, s.Store.Autosave(story))
	require.NoError(t, s.Store.Save("auto1", story, false))
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
		counts := map[storage.SlotGroup]int{}
		for _, r := range g.saves.rows {
			if r.ok {
				counts[r.group]++
			}
		}
		if counts[storage.AutoBattle] != 4 || counts[storage.AutoScenario] != 2 || counts[storage.Manual] != 1 ||
			g.saves.group != storage.Manual || g.saves.pick != "1" {
			p.err = fmt.Errorf("slots: %v, tab %q, pick %s", counts, g.saves.group, g.saves.pick)
			return ebiten.Termination
		}
		g.openOverlay("save")
	case 1:
		if g.saves.group != storage.Manual {
			p.err = fmt.Errorf("automatic slots writable in manual menu")
			return ebiten.Termination
		}
		g.openOverlay("load")
	case 2:
		for _, b := range g.buttons {
			if b.label == "자동 시나리오" && b.click != nil {
				b.click()
				break
			}
		}
		if g.saves.group != storage.AutoScenario {
			p.err = fmt.Errorf("scenario autosave tab cannot be selected")
			return ebiten.Termination
		}
	case 3:
		for _, b := range g.buttons {
			if b.label == "자동 전투" && b.click != nil {
				b.click()
				break
			}
		}
		if g.saves.group != storage.AutoBattle {
			p.err = fmt.Errorf("battle autosave tab cannot be selected")
			return ebiten.Termination
		}
		g.saves.pick = "auto-battle4"
	case 4:
		for _, b := range g.buttons {
			if b.label == "불러오기" && b.click != nil {
				b.click()
				p.step++
				return nil
			}
		}
		p.err = fmt.Errorf("automatic slot 4 cannot be loaded")
		return ebiten.Termination
	case 5:
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
	if name, ok := map[int]string{0: "load", 1: "save", 3: "load-auto-scenario", 4: "load-auto-battle"}[p.step]; ok {
		p.err = saveDuelFrame(dst, filepath.Join(p.out, name+".png"))
	}
	p.drawn = true
}
