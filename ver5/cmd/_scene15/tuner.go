package main

import (
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/gui"
	"srpg/internal/session"
)

// tuner runs the game with a panel on the left: the shot's values (gui.ShotTune) and the field scene's moments, all
// on keys (its clicks belong to the dialogue): [ ] pick a value, - = change it, a g s d f jump to a moment, h hides
// the panel. A frame whose key the panel took is not passed to the game (f is the game's play speed).
type tuner struct {
	g     *gui.Game
	s     *session.Session
	first string
	face  *text.GoTextFace
	hide  bool
	sel   int
	msg   string
}

type knob struct {
	name   string
	get    func() float64
	set    func(float64)
	step   float64
	lo, hi float64
	format string
}

func newTuner(g *gui.Game, s *session.Session, first string) (*tuner, error) {
	f, err := os.Open("assets/fonts/NotoSansKR.ttf")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := text.NewGoTextFaceSource(f)
	if err != nil {
		return nil, err
	}
	return &tuner{g: g, s: s, first: first, face: &text.GoTextFace{Source: src, Size: 13}}, nil
}

func knobs() []knob {
	t := &gui.ShotTune
	return []knob{
		{"c 시작 (놓은 직후)", func() float64 { return t.Chase }, func(v float64) { t.Chase = v }, .02, 0, .98, "%.2f"},
		{"c 끝 (도착 무렵)", func() float64 { return t.ChaseEnd }, func(v float64) { t.ChaseEnd = v }, .02, 0, .98, "%.2f"},
		{"도착 출렁 (낮을수록)", func() float64 { return t.Damping }, func(v float64) { t.Damping = v }, .05, .05, 1.5, "%.2f"},
		{"도착 세기", func() float64 { return t.Stiff }, func(v float64) { t.Stiff = v }, 5, 5, 300, "%.0f"},
		{"멈춤 프레임", func() float64 { return float64(t.Freeze) }, func(v float64) { t.Freeze = int(v) }, 1, 0, 60, "%.0f"},
		{"조준 시간", func() float64 { return t.Aim }, func(v float64) { t.Aim = v }, .1, .1, 5, "%.1f초"},
		{"비행 시간", func() float64 { return t.Flight }, func(v float64) { t.Flight = v }, .1, .2, 10, "%.1f초"},
		{"화살 포물선 높이", func() float64 { return t.Peak }, func(v float64) { t.Peak = v }, 10, 0, 400, "%.0f"},
		{"화극 포물선 높이", func() float64 { return t.ThrowPeak }, func(v float64) { t.ThrowPeak = v }, 10, 0, 1000, "%.0f"},
		{"화극 나는 시간", func() float64 { return t.ThrowTime }, func(v float64) { t.ThrowTime = v }, .05, .1, 3, "%.2f초"},
		{"멈춰 보기", func() float64 { return t.Hold }, func(v float64) { t.Hold = v }, .25, 0, 6, "%.2f초"},
		{"화살 위치 x", func() float64 { return t.Arrow[0] }, func(v float64) { t.Arrow[0] = v }, .05, 0, 1, "%.2f"},
		{"화살 위치 y", func() float64 { return t.Arrow[1] }, func(v float64) { t.Arrow[1] = v }, .05, 0, 1, "%.2f"},
		{"여포 위치 x", func() float64 { return t.Shooter[0] }, func(v float64) { t.Shooter[0] = v }, .05, 0, 1, "%.2f"},
		{"여포 위치 y", func() float64 { return t.Shooter[1] }, func(v float64) { t.Shooter[1] = v }, .05, 0, 1, "%.2f"},
	}
}

var jumps = []struct {
	key         ebiten.Key
	keyName     string
	label, node string
}{
	{ebiten.KeyA, "a", "대치부터", "c15-clash"},
	{ebiten.KeyG, "g", "화극 던지기 (기다리시오)", "c15-c04"},
	{ebiten.KeyS, "s", "좌하로 이동 시작", "c15-r01"},
	{ebiten.KeyD, "d", "활 달라는 줄", "c15-r19"},
	{ebiten.KeyF, "f", "활 쏘기", "c15-p11"},
}

func (t *tuner) jump(node string) {
	if err := gui.StartTrial(t.s, t.first, node); err != nil {
		t.msg = err.Error()
		return
	}
	t.g.RestartScene()
	t.msg = node + "부터"
}

func scale() float32 { return float32(ebiten.Monitor().DeviceScaleFactor()) }

func (t *tuner) Update() error {
	pressed := inpututil.IsKeyJustPressed
	ks := knobs()
	took := true // a key of the panel's own: that frame does not reach the game
	switch {
	case pressed(ebiten.KeyH):
		t.hide = !t.hide
	case pressed(ebiten.KeyBracketLeft):
		t.sel = (t.sel + len(ks) - 1) % len(ks)
	case pressed(ebiten.KeyBracketRight):
		t.sel = (t.sel + 1) % len(ks)
	case pressed(ebiten.KeyMinus), pressed(ebiten.KeyEqual):
		n := ks[t.sel]
		d := n.step
		if pressed(ebiten.KeyMinus) {
			d = -d
		}
		n.set(max(n.lo, min(n.hi, n.get()+d)))
	default:
		took = false
	}
	if took {
		return nil
	}
	for _, j := range jumps {
		if pressed(j.key) {
			t.jump(j.node)
			return nil
		}
	}
	return t.g.Update()
}

func (t *tuner) Draw(screen *ebiten.Image) {
	t.g.Draw(screen)
	if t.hide {
		return
	}
	k := scale()
	t.face.Size = 13 * float64(k)
	ks := knobs()
	row := 22 * k
	vector.FillRect(screen, 4*k, 100*k, 230*k, float32(len(ks)+len(jumps)+4)*row, color.NRGBA{12, 14, 18, 215}, false)
	label := func(s string, x, y float32, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, s, t.face, op)
	}
	y := 108 * k
	label("[ ] 고르기   - = 바꾸기", 12*k, y, color.NRGBA{170, 175, 165, 255})
	y += row
	for i, kn := range ks {
		c := color.Color(color.NRGBA{215, 215, 205, 255})
		if i == t.sel {
			vector.FillRect(screen, 8*k, y-2*k, 222*k, row-2*k, color.NRGBA{70, 90, 60, 255}, false)
			c = color.NRGBA{255, 250, 210, 255}
		}
		label(fmt.Sprintf("%s  "+kn.format, kn.name, kn.get()), 12*k, y, c)
		y += row
	}
	y += row / 2
	for _, j := range jumps {
		label(j.keyName+"   "+j.label, 12*k, y, color.NRGBA{215, 225, 240, 255})
		y += row
	}
	label("h   숨기기   "+t.msg, 12*k, y+row/4, color.NRGBA{170, 175, 165, 255})
}

func (t *tuner) Layout(w, h int) (int, int) { return t.g.Layout(w, h) }
