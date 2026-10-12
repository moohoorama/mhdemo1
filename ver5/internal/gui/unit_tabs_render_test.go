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

func TestRenderedUnitTabs(t *testing.T) {
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
	s.Engine, s.Screen = core.New(d, 17), "playing"
	_, err = s.Engine.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := s.Engine.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = s.Engine.Apply(c)
		require.NoError(t, err)
	}
	state := s.Engine.Snapshot()
	selected := ""
	for _, u := range state.Units {
		if u.Faction == "ally" {
			selected = u.ID
			break
		}
	}
	require.NotEmpty(t, selected)
	traits := sorted(d.Traits)[:12]
	for i := range state.Characters {
		if state.Characters[i].ID == selected {
			state.Characters[i].TraitsOverride = &traits
		}
	}
	s.Engine, err = core.Restore(d, state)
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	g.ensureField(s.Engine.Observe())
	g.modals = nil
	g.selected = selected
	require.NotEmpty(t, d.Characters[g.unitView(s.Engine.Observe(), selected).Def].Biography)
	out := "../../reports/unit-tabs"
	require.NoError(t, os.MkdirAll(out, 0755))
	p := &unitTabsRenderTest{g: g, out: out}
	result := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
}

type unitTabsRenderTest struct {
	g     *Game
	out   string
	step  int
	drawn bool
	err   error
}

func (p *unitTabsRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *unitTabsRenderTest) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	if !p.drawn {
		return nil
	}
	p.drawn = false
	g := p.g
	for _, b := range g.buttons {
		if b.rect.Min.X >= Width-320 && b.rect.Min.Y < Height-240 && b.rect.Max.Y > Height-240 {
			p.err = fmt.Errorf("detail button outside panel: %s", b.label)
			return ebiten.Termination
		}
	}
	if p.step == 1 {
		count := 0
		for _, b := range g.buttons {
			for _, id := range g.unitView(g.S.Engine.Observe(), g.selected).Traits {
				if b.label == g.traitName(id) {
					count++
				}
			}
		}
		if count < 5 {
			p.err = fmt.Errorf("too few visible traits: %d", count)
			return ebiten.Termination
		}
	}
	if p.step < 3 {
		target := []string{"능력", "특성", "장비", "열전"}[p.step+1]
		found := false
		for _, b := range g.buttons {
			if b.label == target {
				b.click()
				found = true
				break
			}
		}
		if !found {
			p.err = fmt.Errorf("missing tab %s", target)
			return ebiten.Termination
		}
	} else if p.step == 3 {
		g.detailTab, g.traitTop = 1, 100
	} else if p.step == 4 {
		state := g.S.Engine.Snapshot()
		for i := range state.Units {
			if state.Units[i].Faction == "ally" {
				state.Units[i].Done, state.Units[i].Moved, state.Units[i].Acted = true, true, true
			}
		}
		for i := range state.Characters {
			if state.Characters[i].ID == g.selected {
				state.Characters[i].Points = 1000
			}
		}
		state.Learning = []string{g.selected}
		g.S.Engine, p.err = core.Restore(g.S.Data, state)
		if p.err != nil {
			return ebiten.Termination
		}
		o := g.S.Engine.Observe()
		g.learn = o.Learners
		u := g.unitView(o, g.selected)
		g.modals = []*modal{{kind: "levelup", level: levelUp{id: u.ID, from: 1, to: 2, before: u.Stats, after: u.Stats, progress: 20}}}
		g.offerEnd(o)
		if g.overlay != "" {
			p.err = fmt.Errorf("confirmation overlaps level-up")
			return ebiten.Termination
		}
	} else if p.step == 5 {
		if len(g.buttons) != 1 {
			p.err = fmt.Errorf("background buttons active in level-up")
			return ebiten.Termination
		}
		g.buttons[0].click()
		g.offerEnd(g.S.Engine.Observe())
		if g.overlay != "" {
			p.err = fmt.Errorf("confirmation overlaps learning")
			return ebiten.Termination
		}
	} else if p.step == 6 {
		found := false
		for _, b := range g.buttons {
			if b.label == "닫기" {
				b.click()
				found = true
				break
			}
		}
		if !found {
			p.err = fmt.Errorf("missing learning close button")
			return ebiten.Termination
		}
		g.player.Skip()
		g.learn = g.S.Engine.Observe().Learners
		g.offerEnd(g.S.Engine.Observe())
		if g.overlay != "confirm" {
			p.err = fmt.Errorf("end confirmation missing after learning")
			return ebiten.Termination
		}
	} else {
		for _, b := range g.buttons {
			if b.label == "예" {
				b.click()
				if g.S.Engine.Observe().Turn == "ally" {
					p.err = fmt.Errorf("confirmation did not end turn")
				}
				return ebiten.Termination
			}
		}
		p.err = fmt.Errorf("missing confirmation button")
		return ebiten.Termination
	}
	p.step++
	return nil
}
func (p *unitTabsRenderTest) Draw(dst *ebiten.Image) {
	if p.err != nil || p.drawn {
		return
	}
	p.g.Draw(dst)
	p.err = saveDuelFrame(dst, filepath.Join(p.out, fmt.Sprintf("tab-%d.png", p.step)))
	p.drawn = true
}
