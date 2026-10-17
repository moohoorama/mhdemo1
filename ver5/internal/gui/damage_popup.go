package gui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Damage and heal numbers are brush-written (docs/ui-mockup, idea 4): the number jumps in
// large and tilted, settles and drifts up; a hit adds a slash, ink spatter and a flash, a
// critical hit a gold ring, a heal light motes rising. Sizes are game pixels at zoom 1.
const (
	numberLife = 1.5
	numberSize = 42
	hitFlash   = .17
)

type numberKind int

const (
	numberDamage numberKind = iota + 1
	numberCrit
	numberHeal
)

type spatter struct{ angle, dist, r float64 }

func (g *Game) sayNumber(u *unitVis, kind numberKind, amount int) {
	if u == nil {
		return
	}
	row := 0
	for _, p := range g.popups {
		if p.unit == u && p.t < .45 {
			row++
		}
	}
	p := &popup{unit: u, row: row, kind: kind, text: fmt.Sprint(amount)}
	if kind == numberHeal {
		p.text = fmt.Sprintf("+%d", amount)
	}
	r := rand.New(rand.NewPCG(uint64(amount), uint64(len(g.popups))))
	n := 14
	if kind == numberHeal {
		n = 10
	}
	for range n {
		p.bits = append(p.bits, spatter{r.Float64() * 2 * math.Pi, r.Float64(), r.Float64()})
	}
	g.popups = append(g.popups, p)
	if kind != numberHeal {
		g.hitFlashT, g.hitFlashCrit = hitFlash, kind == numberCrit
	}
}

func (p *popup) life() float64 {
	if p.kind != 0 {
		return numberLife
	}
	return .9
}

// drawNumber draws popup p anchored above the unit's head at screen (x, y).
func (g *Game) drawNumber(dst *ebiten.Image, p *popup, x, y float64) {
	z := g.zoom
	t := p.t
	switch p.kind {
	case numberDamage, numberCrit:
		g.drawSlash(dst, x, y+34*z, t, z)
		g.drawSpatter(dst, p, x, y+35*z, t, z)
		if p.kind == numberCrit {
			k := bezier(0, 0, .58, 1, min(1, t/.5))
			c := color.NRGBA{R: 255, G: 209, B: 102, A: uint8(255 * (1 - k))}
			r := float32(20 * z * (.2 + 2*k))
			vector.StrokeCircle(dst, float32(x*drawPx), float32((y+18*z)*drawPx), r*float32(drawPx), float32(1.8*z*drawPx), c, true)
		}
		k := bezier(0, 0, .58, 1, min(1, t/numberLife))
		at := []float64{0, .35, .8, 1}
		op := keyframe(k, at, []float64{0, 1, 1, 0})
		sc := keyframe(k, at, []float64{2.2, .9, 1, 1})
		rot := keyframe(k, at, []float64{-8, -4, -4, -4})
		dy := keyframe(k, at, []float64{0, 0, -.12, -.6})
		fill := color.NRGBA{R: 232, G: 67, B: 43, A: 255}
		img := g.numberImage(p.text, fill, color.NRGBA{R: 26, G: 7, B: 3, A: 255}, z)
		g.drawNumberImage(dst, img, x, y+(9+dy*numberSize)*z, sc, rot, op)
		if p.kind == numberCrit {
			small := g.numberImageFace("치명", g.brush(numberSize*.55*z), color.NRGBA{R: 255, G: 209, B: 102, A: 255}, color.NRGBA{R: 26, G: 7, B: 3, A: 255}, z)
			w := float64(img.Bounds().Dx()) / drawPx
			g.drawNumberImage(dst, small, x+(w/2+float64(small.Bounds().Dx())/drawPx/2)*sc*.92, y+(9+dy*numberSize+numberSize*.12)*z, sc, rot, op)
		}
	case numberHeal:
		g.drawMotes(dst, p, x, y+33*z, t, z)
		k := bezier(.2, .8, .2, 1, min(1, t/numberLife))
		at := []float64{0, .3, .8, 1}
		op := keyframe(k, at, []float64{0, 1, 1, 0})
		sc := keyframe(k, at, []float64{.7, 1.05, 1, 1})
		dy := keyframe(k, at, []float64{.3, 0, -.1, -.4})
		img := g.numberImage(p.text, color.NRGBA{R: 143, G: 227, B: 168, A: 255}, color.NRGBA{R: 11, G: 36, B: 20, A: 255}, z)
		g.drawNumberImage(dst, img, x, y+(9+dy*numberSize)*z, sc, 0, op)
	}
}

func (g *Game) numberImage(s string, fill, rim color.NRGBA, z float64) *ebiten.Image {
	return g.numberImageFace(s, g.brush(numberSize*z), fill, rim, z)
}

// numberImageFace renders s with a dark rim and a drop shadow at device resolution.
func (g *Game) numberImageFace(s string, f *text.GoTextFace, fill, rim color.NRGBA, z float64) *ebiten.Image {
	key := fmt.Sprint("num", s, f.Size, fill, drawPx)
	if im, ok := g.textCache[key]; ok {
		return im
	}
	if g.textCache == nil {
		g.textCache = map[string]*ebiten.Image{}
	}
	pad := 4 * z
	w := int(math.Ceil((measure(s, f) + pad*2) * drawPx))
	h := int(math.Ceil((f.Size*1.2 + pad*2) * drawPx))
	im := ebiten.NewImage(w, h)
	draw := func(dx, dy float64, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate((pad+dx)*drawPx, (pad-f.Size*.12+dy)*drawPx)
		op.ColorScale.ScaleWithColor(c)
		sf := *f
		sf.Size *= drawPx
		text.Draw(im, s, &sf, op)
	}
	r := 1.2 * z
	draw(0, 1.8*z, rim)
	for i := range 12 {
		a := float64(i) * math.Pi / 6
		draw(math.Cos(a)*r, math.Sin(a)*r, rim)
	}
	draw(0, 0, fill)
	g.textCache[key] = im
	return im
}

// drawNumberImage draws img centred on layout (x, y), scaled, rotated by deg and faded.
func (g *Game) drawNumberImage(dst, img *ebiten.Image, x, y, sc, deg, alpha float64) {
	if alpha <= 0 {
		return
	}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Rotate(deg * math.Pi / 180)
	op.GeoM.Translate(x*drawPx, y*drawPx)
	op.ColorScale.ScaleAlpha(float32(alpha))
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, op)
}

// drawSlash is the white and gold cut across the target in the first quarter second.
func (g *Game) drawSlash(dst *ebiten.Image, x, y, t, z float64) {
	for _, s := range []struct {
		dx, w float64
		c     color.NRGBA
		delay float64
	}{{0, 3, color.NRGBA{R: 255, G: 251, B: 233, A: 255}, 0}, {-3, 1.25, color.NRGBA{R: 255, G: 207, B: 122, A: 255}, .03}} {
		k := (t - s.delay) / .24
		if k <= 0 || k >= 1 {
			continue
		}
		head := min(1, k/.6)
		tail := max(0, (k-.6)/.4)
		seg := func(u float64) (float64, float64) { // quadratic from upper left to lower right
			ax, ay, cx, cy, bx, by := -15.0, -17.0, 0.0, -5.0, 16.0, 14.0
			px := (1-u)*(1-u)*ax + 2*(1-u)*u*cx + u*u*bx
			py := (1-u)*(1-u)*ay + 2*(1-u)*u*cy + u*u*by
			return x + (px+s.dx)*z, y + py*z
		}
		var path vector.Path
		x0, y0 := seg(tail)
		path.MoveTo(float32(x0*drawPx), float32(y0*drawPx))
		for u := tail; u <= head; u += .05 {
			px, py := seg(u)
			path.LineTo(float32(px*drawPx), float32(py*drawPx))
		}
		so := &vector.StrokeOptions{Width: float32(s.w * z * drawPx), LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(s.c)
		vector.StrokePath(dst, &path, so, op)
	}
}

// drawSpatter flings ink drops out from the hit point and lets them fall and fade.
func (g *Game) drawSpatter(dst *ebiten.Image, p *popup, x, y, t, z float64) {
	k := bezier(.2, .8, .3, 1, min(1, t/.85))
	if k >= 1 {
		return
	}
	for _, b := range p.bits {
		d := 10 + b.dist*25
		cx, cy := math.Cos(b.angle)*d, math.Sin(b.angle)*d*.6
		px := keyframe(k, []float64{0, .5, 1}, []float64{0, cx, cx * 1.1})
		py := keyframe(k, []float64{0, .5, 1}, []float64{0, cy + 5, cy + 13})
		sc := keyframe(k, []float64{0, .5, 1}, []float64{.3, 1, 1})
		a := keyframe(k, []float64{0, .5, 1}, []float64{1, .9, 0})
		r := (.75 + b.r*2.25) * sc * z
		vector.FillCircle(dst, float32((x+px*z)*drawPx), float32((y+py*z)*drawPx), float32(r*drawPx), color.NRGBA{R: 18, G: 8, B: 4, A: uint8(255 * a)}, true)
	}
}

// drawMotes are the pale green streaks rising around a healed unit.
func (g *Game) drawMotes(dst *ebiten.Image, p *popup, x, y, t, z float64) {
	for i, b := range p.bits {
		k := (t - float64(i)*.05) / 1
		if k <= 0 || k >= 1 {
			continue
		}
		e := bezier(0, 0, .58, 1, k)
		a := keyframe(e, []float64{0, .3, 1}, []float64{0, 1, 0})
		mx := x + (b.angle/(2*math.Pi)*15-7.5)*z
		my := y + (b.dist*12-35*e)*z
		h := (4 + b.r*6) * z
		rect(dst, float32(mx), float32(my), float32(1.25*z), float32(h), color.NRGBA{R: 185, G: 245, B: 201, A: uint8(255 * a)})
	}
}

// drawHitFlash is the brief white wash of a landed blow.
func (g *Game) drawHitFlash(dst *ebiten.Image) {
	if g.hitFlashT <= 0 {
		return
	}
	a := .22
	if g.hitFlashCrit {
		a = .32
	}
	rect(dst, 0, 0, float32(uiW), float32(uiH), color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * a * g.hitFlashT / hitFlash)})
}
