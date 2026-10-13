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
	"srpg/internal/core"
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
			g.player.Update(duelview.TransitionTime)
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
	g := p.g
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(c.stage)
	g.editorTrial()
	if g.editorReturn == nil {
		return fmt.Errorf("trial failed: %s", g.editor.message)
	}
	e := g.S.Engine
	if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
		return err
	}
	plan, err := e.DeploymentPlan()
	if err != nil {
		return err
	}
	for _, cmd := range plan {
		if _, err := e.Apply(cmd); err != nil {
			return err
		}
	}
	if !g.S.Data.Characters[c.duel.Ally].Playable {
		snap := e.Snapshot()
		for i := range snap.Units {
			if snap.Units[i].Faction == "enemy" && snap.Units[i].ID != g.S.Data.Stages[c.stage].Boss {
				snap.Units[i].HP = 0
			}
		}
		e, err = core.Restore(g.S.Data, snap)
		if err != nil {
			return err
		}
		g.S.Engine = e
		if _, err = e.Apply(core.Command{Kind: "wait", Actor: g.S.Data.Campaign.Deployment[0]}); err != nil {
			return err
		}
	}
	snap := e.Snapshot()
	ai, ei := -1, -1
	for i, u := range snap.Units {
		if u.Character == c.duel.Ally {
			ai = i
		}
		if u.Character == c.duel.Enemy {
			ei = i
		}
	}
	if ai < 0 || ei < 0 {
		return fmt.Errorf("missing duel participants %s", c.duel.ID)
	}
	st := g.S.Data.Stages[c.stage]
	placed := false
	if snap.Units[ei].Guard {
		snap.Units[ai].X, snap.Units[ai].Y = snap.Units[ei].X-1, snap.Units[ei].Y
		placed = true
	}
	if snap.Units[ai].AI {
		snap.Turn = "support"
	}
	for y := 1; y < st.Height-1 && !placed; y++ {
		for x := 1; x < st.Width-2 && !placed; x++ {
			free := true
			for i, u := range snap.Units {
				if i != ai && i != ei && u.HP > 0 && u.Y == y && (u.X == x || u.X == x+1) {
					free = false
				}
			}
			if free {
				snap.Units[ai].X = x
				snap.Units[ai].Y = y
				snap.Units[ei].X = x + 1
				snap.Units[ei].Y = y
				placed = true
			}
		}
	}
	if !placed {
		return fmt.Errorf("no adjacent cells")
	}
	if c.enemyAttacks {
		snap.Turn = "enemy"
	}
	g.S.Engine, err = core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	e = g.S.Engine
	before := e.Observe()
	g.ensureField(before)
	cmd := core.Command{Kind: "attack", Actor: snap.Units[ai].ID, Target: snap.Units[ei].ID}
	if c.enemyAttacks {
		cmd.Actor, cmd.Target = cmd.Target, cmd.Actor
	}
	r, err := e.Apply(cmd)
	if err != nil {
		return err
	}
	found := false
	for _, event := range r.Events {
		if event.Kind == "duel" {
			found = event.DuelID == c.duel.ID && event.Outcome == c.duel.Outcome
		}
	}
	if !found {
		return fmt.Errorf("attack did not trigger %s", c.duel.ID)
	}
	g.replayOne(before, cmd, r.Events)
	g.player.Update(stagger)
	if g.player.Duel == nil {
		return fmt.Errorf("duel screen did not start")
	}
	if g.player.Duel.Fighters[0].Name != g.name(c.duel.Ally) {
		return fmt.Errorf("participants reversed")
	}
	return nil
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
	view.Clock = duelview.TransitionTime + view.Time/duelview.PlaybackSpeed
	g.Draw(dst)
	if p.beat == 0 && p.sample == 0 {
		length := view.Length()
		for i, clock := range []float64{.2, .4, .55, length - .55, length - .3} {
			view.Clock = clock
			g.Draw(dst)
			p.save(dst, fmt.Sprintf("%s-%d-transition-%d.png", p.cases[p.index].duel.ID, p.index, i))
		}
		view.Clock = duelview.TransitionTime
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
