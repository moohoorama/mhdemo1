package gui

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
	"srpg/internal/session"
)

// stoneGame is a started battle whose first ally throws stones (the sprite of a siege class)
// at the first enemy four cells away, with a second enemy beside the landing cell.
func stoneGame(t *testing.T) (*Game, *unitVis, *unitVis, *unitVis, core.Observation) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	siege := ""
	for _, c := range d.Classes {
		if slices.Contains(c.Tags, content.TagSiege) {
			siege = c.Sprite
		}
	}
	require.NotEmpty(t, siege, "no siege class in the data")
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage != "" {
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
	t.Cleanup(g.Close)
	o := s.Engine.Observe()
	g.ensureField(o)
	g.modals, g.speed = nil, 1
	var thrower, target, beside *unitVis
	for _, u := range o.UnitViews {
		v := g.vis[u.ID]
		switch {
		case u.Faction == "ally" && thrower == nil:
			thrower = v
		case u.Faction != "ally" && target == nil:
			target = v
		case u.Faction != "ally" && beside == nil:
			beside = v
		}
	}
	require.NotNil(t, thrower)
	require.NotNil(t, target)
	require.NotNil(t, beside)
	thrower.art, thrower.siege = g.assets.Lib.Art(siege, nil), true
	open := func(u, v int) bool { return g.field.Tile(u, v) == '.' }
	cu, cv := -1, -1
	for v := 1; v < g.field.Map.H-2 && cu < 0; v++ {
		for u := 1; u < g.field.Map.W-5; u++ {
			if open(u, v) && open(u+4, v) && open(u+4, v+1) {
				cu, cv = u, v
				break
			}
		}
	}
	require.GreaterOrEqual(t, cu, 0, "no open ground for the throw")
	thrower.u, thrower.v = float64(cu), float64(cv)
	target.u, target.v = float64(cu+4), float64(cv)
	beside.u, beside.v = float64(cu+4), float64(cv+1)
	for _, u := range []*unitVis{thrower, target, beside} {
		u.lift = 0
	}
	for _, u := range g.vis { // keep the rest of the army out of the picture
		if u != thrower && u != target && u != beside {
			u.gone = true
		}
	}
	for _, u := range []*unitVis{thrower, target, beside} {
		u.shownHP, u.maxHP = 100, 100
	}
	final := o
	final.UnitViews = slices.Clone(o.UnitViews)
	for i, u := range final.UnitViews {
		if u.ID == thrower.id {
			final.UnitViews[i].X, final.UnitViews[i].Y = cu, cv
		}
	}
	return g, thrower, target, beside, final
}

func TestStoneLandsWithTheBlow(t *testing.T) {
	g, thrower, target, beside, final := stoneGame(t)
	c := core.Command{Kind: "attack", Actor: thrower.id, Target: target.id}
	a := replay.New(final, c.Actor)
	a.Add(c, []core.Event{
		{Kind: "damage", Actor: thrower.id, Target: target.id, Amount: 30},
		{Kind: "damage", Actor: thrower.id, Target: beside.id, Amount: 10}, // the cross splash
	})
	require.Equal(t, "stone", g.effectKind(thrower, c))
	g.player.Play([]*replay.Action{a}, final)
	ms := thrower.art.Animations["attack"].MS
	release := float64(ms[0]+ms[1]) / 1000
	const dt = 1.0 / 60
	var launched, flight, hit, splash, shake float64
	lobs := 0
	for clock := 0.0; clock < 5 && g.player.Busy(); clock += dt {
		g.player.Update(dt)
		g.updateEffects(dt)
		if launched == 0 && len(g.lobs) > 0 {
			launched, flight = clock, g.lobs[0].Dur
			assert.Equal(t, 100, target.shownHP, "hit before the stone left")
		}
		if hit == 0 && target.shownHP < 100 {
			hit, shake, lobs = clock, g.shakeT, len(g.lobs)
		}
		if splash == 0 && beside.shownHP < 100 {
			splash = clock
		}
	}
	require.NotZero(t, launched, "no stone thrown")
	assert.InDelta(t, release, launched, 2*dt, "the stone leaves as attack frame 2 starts")
	assert.InDelta(t, launched+flight, hit, 2*dt, "the target is hit when the stone lands")
	assert.InDelta(t, hit, splash, dt/2, "the unit beside the landing cell is hit with it")
	want := NewLob("stone", 0, 0, 0, 4*16*K, 4*8*K).Dur // four cells along u
	assert.InDelta(t, want, flight, .01)
	assert.Equal(t, 2, lobs, "the stone's dust and dust beside the landing cell")
	assert.Positive(t, shake, "a heavy blow shakes the screen")
}

// TestRenderedStoneThrow plays the stone throw in the real battle view and saves every third
// frame (and a contact sheet) to SRPG_STONE_OUT, default reports/stone-throw.
func TestRenderedStoneThrow(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for native rendering")
	}
	g, thrower, target, beside, final := stoneGame(t)
	out := os.Getenv("SRPG_STONE_OUT")
	if out == "" {
		out = "../../reports/stone-throw"
	}
	require.NoError(t, os.MkdirAll(out, 0755))
	c := core.Command{Kind: "attack", Actor: thrower.id, Target: target.id}
	a := replay.New(final, c.Actor)
	a.Add(c, []core.Event{
		{Kind: "damage", Actor: thrower.id, Target: target.id, Amount: 30},
		{Kind: "damage", Actor: thrower.id, Target: beside.id, Amount: 10},
	})
	ax, ay := g.field.Center(thrower.u, thrower.v)
	bx, by := g.field.Center(target.u, target.v)
	g.camX, g.camY, g.zoom = (ax+bx)/2, (ay+by)/2-40, 2
	g.setZoomImmediate(2)
	g.player.Play([]*replay.Action{a}, final)
	p := &stoneRenderTest{g: g, out: out}
	finished := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { finished <- ebiten.RunGame(p) }
	require.NoError(t, <-finished)
	require.NoError(t, p.err)
	require.NotEmpty(t, p.frames)
	b := p.frames[0].Bounds()
	crop := image.Rect(b.Dx()/2-320, b.Dy()/2-200, b.Dx()/2+320, b.Dy()/2+260).Intersect(b)
	cols := 5
	sheet := image.NewRGBA(image.Rect(0, 0, cols*crop.Dx(), (len(p.frames)+cols-1)/cols*crop.Dy()))
	for i, im := range p.frames {
		at := image.Pt(i%cols*crop.Dx(), i/cols*crop.Dy())
		draw.Draw(sheet, image.Rectangle{at, at.Add(crop.Size())}, im, crop.Min, draw.Src)
	}
	f, err := os.Create(filepath.Join(out, "contact.png"))
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, sheet))
	require.NoError(t, f.Close())
}

type stoneRenderTest struct {
	g      *Game
	out    string
	ticks  int
	frames []*image.RGBA
	err    error
}

func (p *stoneRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *stoneRenderTest) Update() error {
	if p.err != nil || !p.g.player.Busy() && len(p.g.lobs) == 0 {
		return ebiten.Termination
	}
	p.g.player.Update(1.0 / 60)
	for _, u := range p.g.vis {
		u.update(1.0 / 60)
	}
	p.g.updateEffects(1.0 / 60)
	p.ticks++
	if p.ticks > 600 {
		p.err = fmt.Errorf("stone throw never finishes")
		return ebiten.Termination
	}
	return nil
}
func (p *stoneRenderTest) Draw(dst *ebiten.Image) {
	p.g.Draw(dst)
	if p.ticks%3 != 0 {
		return
	}
	im := image.NewRGBA(dst.Bounds())
	dst.ReadPixels(im.Pix)
	p.frames = append(p.frames, im)
	p.err = saveDuelFrame(dst, filepath.Join(p.out, fmt.Sprintf("f%03d.png", p.ticks/3)))
}
