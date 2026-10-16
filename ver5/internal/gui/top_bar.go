package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

// The battle's top bar (docs/ui-mockup, idea 3): a lacquered wood bar with gold trim and a
// cloud mark, the chapter and battle, a turn medallion that flips on a new turn, the side
// to move, the objectives on a small scroll that unrolls once per battle, and a hanging
// name tag for each officer still to act.
const (
	barH         = 42
	barShineGap  = 4.2 // seconds between gold glints
	barShine     = 1.6
	medalFlip    = .38
	scrollUnroll = .7
	tagSwing     = .8
	tagW, tagH   = 24, 74 // tag body plus string
	maxTags      = 8
)

// topBarAnim keeps what the bar animates between frames.
type topBarAnim struct {
	round     int
	flippedAt float64
	battle    int
	openedAt  float64
	done      map[string]float64 // when an officer's tag turned to "acted"
}

func (g *Game) topBar(dst *ebiten.Image, o core.Observation) {
	a := &g.barAnim
	now := g.uiSeconds
	if a.battle != o.Stage+1 {
		*a = topBarAnim{battle: o.Stage + 1, openedAt: now, round: o.Round, flippedAt: now, done: map[string]float64{}}
	}
	if a.round != o.Round {
		a.round, a.flippedAt = o.Round, now
	}
	r := image.Rect(0, 0, Width, barH)
	g.block(r)
	drawBarWood(dst, now)
	drawCloudMark(dst, 10, 9, 38)

	st := g.stageInfo(o)
	writeSpaced(dst, fmt.Sprintf("제%d장", o.Stage+1), g.serif(10, 700), 60, 2, .3, gold)
	write(dst, st.name, g.serif(17, 900), 60, 14, ink)

	// turn medallion, squashed vertically while it flips to the new number
	mx, my := float32(420), float32(barH/2)
	k := float32(bezier(.2, .8, .2, 1, phase(now-a.flippedAt, 0, medalFlip)))
	for i := float32(18); i > 0; i-- {
		t := i / 18
		c := color.NRGBA{R: uint8(29 + 45*(1-t)), G: uint8(18 + 30*(1-t)), B: uint8(12 + 20*(1-t)), A: 255}
		fillEllipse(dst, mx, my, i, i*max(.05, k), c)
	}
	strokeEllipse(dst, mx, my, 18, 18*max(.05, k), 2, gold)
	if k > .5 {
		n := fmt.Sprint(o.Round)
		nf := g.serif(16, 900)
		write(dst, n, nf, float64(mx)-measure(n, nf)/2, float64(my)-15, ink)
		if st.turnLimit > 0 {
			l := fmt.Sprintf("/ %d", st.turnLimit)
			g.label(dst, l, float64(mx)-g.width(l, 9)/2, float64(my)+3, 9, gold)
		}
	}

	// the side to move
	side, sc := g.lordName()+"군 차례", color.NRGBA{R: 159, G: 224, B: 168, A: 255}
	switch o.Turn {
	case "enemy":
		side, sc = "적군 차례", color.NRGBA{R: 232, G: 192, B: 74, A: 255}
	case "support":
		side, sc = "동맹군 차례", color.NRGBA{R: 120, G: 180, B: 240, A: 255}
	}
	glow := sc
	glow.A = 70
	fillCircle(dst, 453, float32(barH/2), 7, glow)
	fillCircle(dst, 453, float32(barH/2), 4, sc)
	write(dst, side, g.serif(14, 900), 463, 11, sc)

	// objectives on a small scroll, unrolled once at the start of the battle
	win, lose := g.objective(st)
	g.objectiveScroll(dst, 580, 7, win, lose, bezier(.3, .7, .2, 1, phase(now-a.openedAt, .3, scrollUnroll)))

	if o.Phase == "battle" {
		g.nameTags(dst, o, 1140, now)
	}
	if g.speed > 1 {
		g.label(dst, fmt.Sprintf("재생 ×%d", g.speed), 1030, 48, 13, gold)
	}
	g.btn(dst, image.Rect(1176, 6, 1272, 35), "메뉴", func() { g.openOverlay("menu") })
}

// drawBarWood is the bar itself: lacquer, grain, gold trim, shadow and a passing glint.
func drawBarWood(dst *ebiten.Image, now float64) {
	for i := 0; i < 8; i++ {
		k := float64(i) / 7
		rect(dst, 0, float32(i*barH/8), Width, barH/8+1, color.NRGBA{R: uint8(58 - 29*k), G: uint8(36 - 18*k), B: uint8(24 - 12*k), A: 255})
	}
	for x := float32(0); x < Width; x += 7 {
		rect(dst, x, 0, 2, barH, color.NRGBA{A: 20})
	}
	rect(dst, 0, 0, Width, barH*.4, color.NRGBA{R: 255, G: 220, B: 170, A: 10})
	for i := float32(0); i < 6; i++ {
		rect(dst, 0, barH+i, Width, 1, color.NRGBA{A: uint8(90 - i*15)})
	}
	rect(dst, 0, barH-5, Width, 1, color.NRGBA{R: gold.R, G: gold.G, B: gold.B, A: 140})
	rect(dst, 0, barH-2, Width, 2, gold)
	t := math.Mod(now, barShineGap)
	if t < barShine {
		k := bezier(.42, 0, .58, 1, t/barShine)
		x := float32(-.2*Width + k*1.4*Width)
		for i := float32(0); i < 12; i++ {
			s := float32(math.Sin(float64(i/11) * math.Pi))
			rect(dst, x+i*19, 0, 19, barH-2, color.NRGBA{R: 255, G: 230, B: 180, A: uint8(34 * s)})
		}
	}
}

// drawCloudMark is the gold cloud at the left end of the bar, w wide.
func drawCloudMark(dst *ebiten.Image, x, y, w float32) {
	k := w / 64
	var p vector.Path
	p.MoveTo(6, 30)
	p.CubicTo(2, 22, 10, 14, 18, 18)
	p.CubicTo(18, 8, 32, 4, 38, 12)
	p.CubicTo(44, 6, 58, 10, 56, 20)
	p.CubicTo(62, 22, 62, 32, 54, 32)
	p.LineTo(10, 32)
	p.Close()
	p.MoveTo(24, 26)
	p.CubicTo(20, 26, 20, 20, 24, 20)
	p.CubicTo(27, 20, 27, 24, 24, 24)
	p.MoveTo(40, 24)
	p.CubicTo(44, 24, 44, 18, 40, 18)
	p.CubicTo(37, 18, 37, 22, 40, 22)
	var q vector.Path
	op := &vector.AddPathOptions{}
	op.GeoM.Scale(float64(k), float64(k))
	op.GeoM.Translate(float64(x), float64(y))
	q.AddPath(&p, op)
	do := &vector.DrawPathOptions{AntiAlias: true}
	do.ColorScale.ScaleWithColor(gold)
	strokePath(dst, &q, vector.StrokeOptions{Width: 1.7 * k, LineJoin: vector.LineJoinRound}, do)
}

func fillEllipse(dst *ebiten.Image, x, y, rx, ry float32, c color.Color) {
	var p vector.Path
	p.Arc(0, 0, 1, 0, 2*math.Pi, vector.Clockwise)
	var q vector.Path
	op := &vector.AddPathOptions{}
	op.GeoM.Scale(float64(rx), float64(ry))
	op.GeoM.Translate(float64(x), float64(y))
	q.AddPath(&p, op)
	do := &vector.DrawPathOptions{AntiAlias: true}
	do.ColorScale.ScaleWithColor(c)
	fillPath(dst, &q, do)
}

func strokeEllipse(dst *ebiten.Image, x, y, rx, ry, w float32, c color.Color) {
	var p vector.Path
	p.Arc(0, 0, 1, 0, 2*math.Pi, vector.Clockwise)
	p.Close()
	var q vector.Path
	op := &vector.AddPathOptions{}
	op.GeoM.Scale(float64(rx), float64(ry))
	op.GeoM.Translate(float64(x), float64(y))
	q.AddPath(&p, op)
	do := &vector.DrawPathOptions{AntiAlias: true}
	do.ColorScale.ScaleWithColor(c)
	strokePath(dst, &q, vector.StrokeOptions{Width: w}, do)
}

// objectiveScroll draws the win and loss conditions on parchment between two rods, the
// left fraction k of it unrolled.
func (g *Game) objectiveScroll(dst *ebiten.Image, x, y float32, win, lose string, k float64) {
	f := g.serif(12, 700)
	parts := [][2]string{{"승리 ", ""}, {win, ""}, {"  ·  패배 ", ""}, {lose, ""}}
	full := float32(20)
	for _, p := range parts {
		full += float32(measure(p[0], f))
	}
	w := full * float32(k)
	h := float32(barH - 14)
	rod := func(rx float32) {
		for i := float32(0); i < 5; i++ {
			c := color.NRGBA{R: 74, G: 44, B: 24, A: 255}
			if i == 2 {
				c = color.NRGBA{R: 168, G: 116, B: 63, A: 255}
			} else if i == 1 || i == 3 {
				c = color.NRGBA{R: 120, G: 80, B: 44, A: 255}
			}
			rect(dst, rx+i, y-3, 1, h+6, c)
		}
	}
	if w > 1 {
		for i := float32(0); i < 4; i++ {
			k := i / 3
			rect(dst, x+5, y+h*i/4, w, h/4+1, color.NRGBA{R: uint8(239 - 22*k), G: uint8(228 - 25*k), B: uint8(200 - 34*k), A: 255})
		}
		sub := dst.SubImage(image.Rect(int(float64(x+5)*scale), 0, int(float64(x+5+w)*scale), int(float64(barH)*scale))).(*ebiten.Image)
		tx := float64(x + 15)
		for i, p := range parts {
			c := color.NRGBA{R: 43, G: 29, B: 18, A: 255}
			if i%2 == 0 {
				c = color.NRGBA{R: 184, G: 50, B: 31, A: 255}
			}
			write(sub, p[0], f, tx, float64(y)+5, c)
			tx += measure(p[0], f)
		}
	}
	rod(x)
	rod(x + 5 + w)
}

// nameTags hangs a wooden tag with each allied officer's name written top to bottom,
// ending at right: light while the officer may act, dark once done (swinging as it
// turns), ringed in gold while selected.
func (g *Game) nameTags(dst *ebiten.Image, o core.Observation, right float32, now float64) {
	ids := []string{}
	for _, u := range o.UnitViews {
		if u.Faction == "ally" && !u.AI && u.HP > 0 {
			ids = append(ids, u.ID)
		}
	}
	extra := 0
	if len(ids) > maxTags {
		left := []string{}
		for _, id := range ids {
			if u := g.unitView(o, id); u != nil && !u.Done {
				left = append(left, id)
			}
		}
		extra = len(ids) - min(len(left), maxTags)
		ids = left[:min(len(left), maxTags)]
	}
	x := right - float32(len(ids))*(tagW+6)
	if extra > 0 {
		g.label(dst, fmt.Sprintf("+%d", extra), float64(x)-24, 12, 13, gold)
	}
	for _, id := range ids {
		u := g.unitView(o, id)
		at, seen := g.barAnim.done[id]
		if u.Done && !seen {
			g.barAnim.done[id], at = now, now
		} else if !u.Done && seen {
			delete(g.barAnim.done, id)
		}
		swing := 0.0
		if u.Done {
			k := phase(now-at, 0, tagSwing)
			swing = keyframe(bezier(0, 0, .58, 1, k), []float64{0, .25, .5, .75, 1}, []float64{0, 9, -5, 2, 0})
		}
		img := g.tagImage(g.name(id), u.Done, g.selected == id || g.ord.actor == id)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(img.Bounds().Dx())/2, 0)
		op.GeoM.Rotate(swing * math.Pi / 180)
		op.GeoM.Translate(float64(x+tagW/2)*scale, 0)
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(img, op)
		g.block(image.Rect(int(x), barH, int(x+tagW), tagH))
		x += tagW + 6
	}
}

// tagImage renders one name tag (string, body, vertical name) at device resolution.
func (g *Game) tagImage(name string, done, lit bool) *ebiten.Image {
	key := fmt.Sprint("tag", name, done, lit, scale)
	if im, ok := g.textCache[key]; ok {
		return im
	}
	if g.textCache == nil {
		g.textCache = map[string]*ebiten.Image{}
	}
	pad := 4
	im := ebiten.NewImage(int(float64(tagW+pad*2)*scale), int(float64(tagH+pad)*scale))
	old := scale
	defer func() { scale = old }()
	s := float32(old)
	draw := func(x, y, w, h float32, c color.Color) {
		vector.FillRect(im, (x+float32(pad))*s, y*s, w*s, h*s, c, false)
	}
	draw(tagW/2, 0, 1, 12, color.NRGBA{R: 199, G: 154, B: 85, A: 255})
	body := color.NRGBA{R: 220, G: 184, B: 120, A: 255}
	edge := color.NRGBA{R: 242, G: 215, B: 158, A: 255}
	side := color.NRGBA{R: 167, G: 122, B: 63, A: 255}
	fg := color.NRGBA{R: 42, G: 26, B: 12, A: 255}
	if done {
		body, edge, side, fg = color.NRGBA{R: 75, G: 58, B: 42, A: 255}, color.NRGBA{R: 90, G: 72, B: 54, A: 255}, color.NRGBA{R: 51, G: 39, B: 28, A: 255}, color.NRGBA{R: 122, G: 106, B: 85, A: 255}
	}
	if lit {
		draw(-2, 10, tagW+4, tagH-10, gold)
	}
	draw(0, 12, tagW, tagH-12, edge)
	draw(1, 13, tagW-2, tagH-14, body)
	draw(1, 13, 4, tagH-14, side)
	draw(tagW-5, 13, 4, tagH-14, side)
	draw(2, tagH-2, tagW-4, 2, color.NRGBA{A: 90})
	scale = old
	f := g.serif(13, 900)
	runes := []rune(name)
	lh := 17.0
	y := 12 + (float64(tagH-12)-lh*float64(len(runes)))/2
	for _, r := range runes {
		ch := string(r)
		write(im, ch, f, float64(pad)+float64(tagW)/2-measure(ch, f)/2, y-1, fg)
		y += lh
	}
	g.textCache[key] = im
	return im
}
