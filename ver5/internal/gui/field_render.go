package gui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Native fields retain one ground cache but render moving layers at viewport resolution.
func (f *Field) prepareRender(z, x, y float64, w, h int) {
	if !f.Map.Native {
		return
	}
	f.renderZoom, f.renderX, f.renderY = z, x, y
	if f.Canvas.Bounds().Dx() != w || f.Canvas.Bounds().Dy() != h {
		f.Canvas.Deallocate()
		f.shadows.Deallocate()
		f.Canvas = newLayer(w, h)
		f.shadows = newLayer(w, h)
		for i := range f.levels {
			f.levels[i].shadows.Deallocate()
			f.levels[i].shadows = newLayer(w, h)
		}
	}
}
func (f *Field) renderOp(dst *ebiten.Image, op *ebiten.DrawImageOptions) {
	if f.renderZoom > 0 && (dst == f.Canvas || dst == f.shadows || dst == f.afloat || dst == f.wading || f.levelShadows(dst)) {
		op.GeoM.Translate(-f.renderX, -f.renderY)
		op.GeoM.Scale(f.renderZoom, f.renderZoom)
	}
}
func (f *Field) renderPoint(x, y float64) (float64, float64) {
	if f.renderZoom > 0 {
		return (x - f.renderX) * f.renderZoom, (y - f.renderY) * f.renderZoom
	}
	return x, y
}
func (f *Field) renderBackground(dst *ebiten.Image, blend ebiten.Blend) {
	p := f.A.pattern()
	z := f.renderZoom
	w, h := 64*z, 32*z
	ox, oy := -f.renderX*z, -f.renderY*z
	for y := math.Mod(oy, h) - h; y < float64(dst.Bounds().Dy()); y += h {
		for x := math.Mod(ox, w) - w; x < float64(dst.Bounds().Dx()); x += w {
			op := &ebiten.DrawImageOptions{Blend: blend}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(x, y)
			if !f.ShowExterior {
				op.ColorScale.Scale(.45, .45, .45, 1)
			}
			dst.DrawImage(p, op)
		}
	}
}
