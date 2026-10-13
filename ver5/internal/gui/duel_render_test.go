package gui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/duelview"
	"srpg/internal/session"
)

func TestRenderedCampaignDuels(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for the native renderer")
	}
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(d, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	out := "../../reports/duel-integration"
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &duelRenderTest{g: g, out: out, index: -1}
	for stage, st := range d.Stages {
		for _, duel := range st.Duels {
			p.cases = append(p.cases, duelRenderCase{stage: stage, duel: duel})
		}
	}
	// Enemy-initiated presentation must keep the ally on the left.
	p.cases = append(p.cases, duelRenderCase{stage: 0, duel: d.Stages[0].Duels[0], enemyAttacks: true})
	result := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
}

type duelRenderCase struct {
	stage        int
	duel         content.Duel
	enemyAttacks bool
}
type duelRenderTest struct {
	g           *Game
	out         string
	cases       []duelRenderCase
	index, beat int
	sample      int
	drawn       bool
	err         error
}

func (p *duelRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *duelRenderTest) Update() error {
	if p.index >= 0 && !p.drawn {
		return nil
	}
	p.drawn = false
	if p.index >= 0 {
		p.sample++
		if p.sample < 3 {
			return nil
		}
		p.sample = 0
		p.beat++
		if p.beat < len(p.g.player.Duel.Story.Beats) {
			return nil
		}
		g := p.g
		before := g.S.Engine.Snapshot()
		if p.index%2 == 0 {
			g.player.SkipDuel()
			g.player.Update(duelview.Transition.Intro())
		} else {
			g.player.Update(g.player.Duel.Length())
		}
		if g.player.Duel != nil {
			return p.fail(fmt.Errorf("duel did not return to map"))
		}
		for i := 0; i < 100 && g.player.Busy(); i++ {
			g.player.Update(.5)
		}
		if g.player.Busy() {
			return p.fail(fmt.Errorf("aftermath never completed"))
		}
		if before.Revision != g.S.Engine.Snapshot().Revision || before.RNG != g.S.Engine.Snapshot().RNG {
			return p.fail(fmt.Errorf("presentation mutated battle state"))
		}
		c := p.cases[p.index]
		if c.duel.EnemyResult == "death" || c.duel.EnemyResult == "retreat" {
			if v := g.vis[c.duel.Enemy]; v == nil || !v.gone || v.shownHP != 0 {
				return p.fail(fmt.Errorf("aftermath did not remove enemy"))
			}
		}
		g.endEditorTrial()
	}
	p.index++
	p.beat = 0
	if p.index >= len(p.cases) {
		return ebiten.Termination
	}
	if err := p.start(p.cases[p.index]); err != nil {
		return p.fail(err)
	}
	return nil
}
func (p *duelRenderTest) fail(err error) error { p.err = err; return ebiten.Termination }
func (p *duelRenderTest) start(c duelRenderCase) error {
	return p.g.startDuelScene(c.stage, c.duel, c.enemyAttacks)
}
func (p *duelRenderTest) Draw(dst *ebiten.Image) {
	if p.drawn || p.index < 0 || p.index >= len(p.cases) {
		return
	}
	g := p.g
	view := g.player.Duel
	duration := view.Story.Beats[p.beat].Duration
	local := []float64{min(.2, duration/4), min(.57, duration/2), duration - .001}[p.sample]
	view.Time = view.Story.Start(p.beat) + local
	view.Clock = duelview.Transition.Intro() + view.Time/duelview.PlaybackSpeed
	g.Draw(dst)
	if p.beat == 0 && p.sample == 0 {
		length := view.Length()
		tr := duelview.Transition
		for i, clock := range []float64{tr.PanTime / 2, tr.PanTime + tr.ZoomTime/3, tr.PanTime + tr.ZoomTime*2/3, tr.PanTime + tr.ZoomTime + tr.UITime/2, length - tr.PanTime - tr.ZoomTime/2} {
			view.Clock = clock
			g.Draw(dst)
			p.save(dst, fmt.Sprintf("%s-%d-transition-%d.png", p.cases[p.index].duel.ID, p.index, i))
		}
		// Retina windows render at device resolution; the transition must scale with it.
		scale = 2
		large := ebiten.NewImage(Width*2, Height*2)
		view.Clock = duelview.Transition.PanTime + duelview.Transition.ZoomTime/2
		g.Draw(large)
		p.save(large, fmt.Sprintf("%s-%d-transition-2x.png", p.cases[p.index].duel.ID, p.index))
		large.Deallocate()
		scale = 1
		view.Clock = duelview.Transition.Intro()
		g.Draw(dst)
	}
	// Verify shared renderer at twice the logical resolution too.
	if p.beat == 0 && p.sample == 0 {
		large := ebiten.NewImage(Width*2, Height*2)
		view.Draw(large)
		pixels := make([]byte, Width*Height*16)
		large.ReadPixels(pixels)
		if pixels[len(pixels)-1] == 0 {
			p.err = fmt.Errorf("duel does not fill resized screen")
		}
		large.Deallocate()
	}
	p.save(dst, fmt.Sprintf("%s-%d-%02d-%d.png", p.cases[p.index].duel.ID, p.index, p.beat, p.sample))
	p.drawn = true
}

func (p *duelRenderTest) save(dst *ebiten.Image, name string) {
	img := image.NewRGBA(dst.Bounds())
	dst.ReadPixels(img.Pix)
	f, err := os.Create(filepath.Join(p.out, name))
	if err == nil {
		err = png.Encode(f, img)
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		p.err = err
	}
}
