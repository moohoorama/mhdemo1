package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// wadeDepth is how deep a unit stands in the ford, in canvas pixels above its feet: a horse's legs reach deeper.
func wadeDepth(u *unitVis) float64 {
	if u.mounted {
		return 8
	}
	return 5
}

// updateWade eases u.wade toward 1 while it stands on a ford cell and toward 0 elsewhere, so a unit sinks in as it
// walks onto the ford and comes up as it walks off.
func (f *Field) updateWade(u *unitVis) {
	target := 0.0
	if t := f.Tile(int(math.Round(u.u)), int(math.Round(u.v))); u.lift <= 0 && u.lift >= -fordDepth && (t == 's' || t == 'S') { // a battle map's ford, a scene map's
		target = 1
	}
	u.wade += math.Max(-.08, math.Min(.08, target-u.wade))
}

// drawWading draws u standing in the ford, or as deep as a flood has risen round it (u.flood): the back half of the
// ripples round its legs, the sprite cut at the water line — above it as it is, below it tinted with the water and
// half let through to the ground — then the front half of the ripples. Both halves go straight onto the canvas (no
// scratch layer per unit), so a host of hundreds wading a ford stays cheap to draw.
func (f *Field) drawWading(u *unitVis, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	d := math.Max(wadeDepth(u)*u.wade, u.flood)
	f.drawRipples(u, x, y-d, false)
	b := s.Image.Bounds()
	cut := min(max(int(math.Round(s.PivotY-d)), 0), b.Dy()) // the water line, in rows from the sprite's top
	if cut > 0 {
		top := *op
		f.blit(f.Canvas, Sprite{s.Image.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+cut)).(*ebiten.Image), s.PivotX, s.PivotY}, x, y, &top)
	}
	if cut < b.Dy() {
		under := *op
		under.ColorScale.Scale(.72, .84, .9, 1) // the water's blue-grey over what it covers
		under.ColorScale.ScaleAlpha(.55)
		f.blit(f.Canvas, Sprite{s.Image.SubImage(image.Rect(b.Min.X, b.Min.Y+cut, b.Max.X, b.Max.Y)).(*ebiten.Image), s.PivotX, s.PivotY - float64(cut)}, x, y, &under)
	}
	f.drawRipples(u, x, y-d, true)
}

// drawRipples draws the back or the front half of two rings spreading from u's legs at the water line (x, y) and
// of a steady one where the water meets them; walking spreads them faster. The rings are pictures made once per
// size (rippleRing), so hundreds of wading soldiers cost a few hundred plain draws.
func (f *Field) drawRipples(u *unitVis, x, y float64, front bool) {
	rx := 6.0
	if u.mounted {
		rx = 11
	}
	period := 1500.0
	if u.anim == "walk" {
		period = 700
	}
	px, py := f.renderPoint(x, y)
	ring := func(r float64, a float32) {
		img, cx, cy := f.rippleRing(r, front)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(px-cx), math.Round(py-cy))
		op.ColorScale.ScaleWithColor(color.NRGBA{200, 222, 230, 255})
		op.ColorScale.ScaleAlpha(a * float32(u.wade))
		f.Canvas.DrawImage(img, op)
	}
	ring(rx*.8, .45)
	phase := f.clockMS/period + (u.u*7+u.v*3)*.13
	for i := 0; i < 2; i++ {
		p := math.Mod(phase+float64(i)*.5, 1)
		ring(rx*(.8+.7*p), float32(.4*(1-p)))
	}
}

// rippleRing is the back (upper) or front (lower) half of a ring r canvas pixels across its half width and half
// that tall, stroked one render pixel wide in white at the render zoom, and where its centre lies in the picture.
// r is kept to half pixels so a few dozen pictures serve every ring.
func (f *Field) rippleRing(r float64, front bool) (*ebiten.Image, float64, float64) {
	z := math.Max(1, f.renderZoom)
	key := [3]int{int(math.Round(r * 2)), 0, int(math.Round(z * 100))}
	if front {
		key[1] = 1
	}
	r = float64(key[0]) / 2
	w, h := int(math.Ceil(2*r*z))+4, int(math.Ceil(r*.5*z))+4
	cx, cy := float64(w)/2, 2.0
	if !front {
		cy = float64(h) - 2
	}
	if img, ok := f.ripples[key]; ok {
		return img, cx, cy
	}
	if f.ripples == nil {
		f.ripples = map[[3]int]*ebiten.Image{}
	}
	img := ebiten.NewImage(w, h)
	var p vector.Path
	t0 := math.Pi // the back half lies above the centre on screen
	if front {
		t0 = 0
	}
	for i := 0; i <= 12; i++ {
		t := t0 + float64(i)/12*math.Pi
		qx, qy := float32(cx+r*z*math.Cos(t)), float32(cy+r*.5*z*math.Sin(t))
		if i == 0 {
			p.MoveTo(qx, qy)
		} else {
			p.LineTo(qx, qy)
		}
	}
	vector.StrokePath(img, &p, &vector.StrokeOptions{Width: float32(z)}, &vector.DrawPathOptions{AntiAlias: false})
	f.ripples[key] = img
	return img, cx, cy
}
