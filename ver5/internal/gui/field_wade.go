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
	if u.lift == 0 && f.Tile(int(math.Round(u.u)), int(math.Round(u.v))) == 's' {
		target = 1
	}
	u.wade += math.Max(-.08, math.Min(.08, target-u.wade))
}

// drawWading draws u standing in the ford: the back half of the ripples round its legs, the sprite put together on a
// scratch layer with the part under the water line tinted with the water and half let through to the ground, then the
// front half of the ripples.
func (f *Field) drawWading(u *unitVis, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	cb := f.Canvas.Bounds()
	if f.wading == nil || f.wading.Bounds() != cb {
		if f.wading != nil {
			f.wading.Deallocate()
		}
		f.wading = ebiten.NewImage(cb.Dx(), cb.Dy())
	}
	b := s.Image.Bounds()
	x0, y0 := f.renderPoint(x-s.PivotX, y-s.PivotY)
	x1, y1 := f.renderPoint(x-s.PivotX+float64(b.Dx()), y-s.PivotY+float64(b.Dy()))
	area := image.Rect(int(x0)-2, int(y0)-2, int(x1)+2, int(y1)+2).Intersect(cb)
	if area.Empty() {
		return
	}
	d := wadeDepth(u) * u.wade
	f.drawRipples(u, x, y-d, false)
	layer := f.wading.SubImage(area).(*ebiten.Image)
	layer.Clear()
	f.blit(f.wading, s, x, y, op)
	_, wy := f.renderPoint(x, y-d)
	band := func(top float64, c color.NRGBA, blend ebiten.Blend) {
		var p vector.Path
		p.MoveTo(float32(area.Min.X), float32(top))
		p.LineTo(float32(area.Max.X), float32(top))
		p.LineTo(float32(area.Max.X), float32(area.Max.Y))
		p.LineTo(float32(area.Min.X), float32(area.Max.Y))
		p.Close()
		var cs ebiten.ColorScale
		cs.ScaleWithColor(c)
		vector.FillPath(layer, &p, nil, &vector.DrawPathOptions{ColorScale: cs, Blend: blend})
	}
	wy = math.Round(wy)
	band(wy, color.NRGBA{108, 138, 150, 110}, ebiten.BlendSourceAtop)
	band(wy+1, color.NRGBA{96, 128, 140, 90}, ebiten.BlendSourceAtop)
	band(wy+1, color.NRGBA{255, 255, 255, 120}, ebiten.BlendDestinationOut)
	cop := &ebiten.DrawImageOptions{}
	cop.GeoM.Translate(float64(area.Min.X), float64(area.Min.Y))
	f.Canvas.DrawImage(layer, cop)
	f.drawRipples(u, x, y-d, true)
}

// drawRipples strokes the back or the front half of two rings spreading from u's legs at the water line (x, y) and
// of a steady one where the water meets them; walking spreads them faster.
func (f *Field) drawRipples(u *unitVis, x, y float64, front bool) {
	rx := 6.0
	if u.mounted {
		rx = 11
	}
	period := 1500.0
	if u.anim == "walk" {
		period = 700
	}
	ring := func(r float64, a float32) {
		var p vector.Path
		t0 := math.Pi // the back half lies above the centre on screen
		if front {
			t0 = 0
		}
		for i := 0; i <= 12; i++ {
			t := t0 + float64(i)/12*math.Pi
			px, py := f.renderPoint(x+r*math.Cos(t), y+r*.5*math.Sin(t))
			if i == 0 {
				p.MoveTo(float32(px), float32(py))
			} else {
				p.LineTo(float32(px), float32(py))
			}
		}
		var cs ebiten.ColorScale
		cs.ScaleWithColor(color.NRGBA{200, 222, 230, 255})
		cs.ScaleAlpha(a * float32(u.wade))
		z := float32(math.Max(1, f.renderZoom))
		vector.StrokePath(f.Canvas, &p, &vector.StrokeOptions{Width: z}, &vector.DrawPathOptions{ColorScale: cs, AntiAlias: false})
	}
	ring(rx*.8, .45)
	phase := f.clockMS/period + (u.u*7+u.v*3)*.13
	for i := 0; i < 2; i++ {
		p := math.Mod(phase+float64(i)*.5, 1)
		ring(rx*(.8+.7*p), float32(.4*(1-p)))
	}
}
