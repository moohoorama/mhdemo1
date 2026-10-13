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
	"srpg/internal/replay"
	"srpg/internal/session"
)

func TestRenderedCombatFollowupsAndResults(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for native rendering")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage == "B03" {
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
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	before := s.Engine.Observe()
	g.ensureField(before)
	g.modals = nil
	for id, pos := range map[string][2]float64{"lubu": {24, 4}, "guanyu": {23, 4}, "zhangfei": {24, 5}} {
		u := g.vis[id]
		require.NotNil(t, u)
		u.u, u.v, u.shownHP, u.maxHP = pos[0], pos[1], 100, 100
	}
	g.camX, g.camY = g.field.Center(24, 4)
	g.zoom = 1.5
	c := core.Command{Kind: "attack", Actor: "lubu", Target: "guanyu"}
	a := replay.New(before, c.Actor)
	a.Add(c, []core.Event{
		{Kind: "damage", Actor: "lubu", Target: "guanyu", Amount: 100},
		{Kind: "retreat", Actor: "guanyu"},
		{Kind: "followup-start", Actor: "lubu", Target: "zhangfei"},
		{Kind: "damage", Actor: "lubu", Target: "zhangfei", Amount: 20},
		{Kind: "followup-end", Actor: "lubu", Target: "zhangfei"},
	})
	final := before
	final.UnitViews = append([]core.UnitView(nil), before.UnitViews...)
	for i := range final.UnitViews {
		if final.UnitViews[i].ID == "guanyu" {
			final.UnitViews[i].HP = 0
		}
		if final.UnitViews[i].ID == "zhangfei" {
			final.UnitViews[i].HP = 80
		}
	}
	g.player.Play([]*replay.Action{a}, final)
	result := before
	result.Phase, result.Result, result.UnitViews = "result", "victory", nil
	party := resultUnits(before)
	for i := 0; i < 12; i++ {
		u := party[i%len(party)]
		u.Name = fmt.Sprintf("장수 %02d", i+1)
		result.UnitViews = append(result.UnitViews, u)
	}
	support := party[0]
	support.AI, support.Name = true, "제외되어야 할 동맹군"
	result.UnitViews = append(result.UnitViews, support)
	out := "../../reports/combat-results"
	require.NoError(t, os.MkdirAll(out, 0755))
	p := &combatResultsRenderTest{g: g, result: result, out: out}
	finished := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { finished <- ebiten.RunGame(p) }
	require.NoError(t, <-finished)
	require.NoError(t, p.err)
	require.True(t, p.sawWindup)
	require.Equal(t, 2, g.resultPage)
}

type combatResultsRenderTest struct {
	g                         *Game
	result                    core.Observation
	out                       string
	ticks                     int
	results, drawn, sawWindup bool
	err                       error
}

func (p *combatResultsRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *combatResultsRenderTest) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	g := p.g
	if p.results {
		if !p.drawn {
			return nil
		}
		if g.resultPage == 2 {
			return ebiten.Termination
		}
		for _, b := range g.buttons {
			if b.label == "다음" {
				b.click()
				p.drawn = false
				return nil
			}
		}
		p.err = fmt.Errorf("result next-page button missing")
		return ebiten.Termination
	}
	g.player.Update(1.0 / 60)
	for _, u := range g.vis {
		u.update(1.0 / 60)
	}
	g.updateEffects(1.0 / 60)
	a, b := g.vis["guanyu"], g.vis["zhangfei"]
	if b.shownHP < 100 && !a.gone {
		p.err = fmt.Errorf("followup damage occurred before first retreat finished")
		return ebiten.Termination
	}
	if a.gone && b.shownHP == 100 && g.vis["lubu"].anim == "attack" {
		p.sawWindup = true
	}
	if !g.player.Busy() {
		p.results = true
		p.drawn = false
	}
	p.ticks++
	if p.ticks > 1000 {
		p.err = fmt.Errorf("combat replay never finishes")
		return ebiten.Termination
	}
	return nil
}
func (p *combatResultsRenderTest) Draw(dst *ebiten.Image) {
	if p.results {
		p.g.result(dst, p.result)
		if !p.drawn {
			p.err = saveDuelFrame(dst, filepath.Join(p.out, fmt.Sprintf("result-page-%d.png", p.g.resultPage+1)))
			p.drawn = true
		}
		return
	}
	p.g.Draw(dst)
	if p.sawWindup && !p.drawn {
		p.err = saveDuelFrame(dst, filepath.Join(p.out, "followup-after-retreat.png"))
		p.drawn = true
	}
}
