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
		f.Canvas = ebiten.NewImage(w, h)
		f.shadows = ebiten.NewImage(w, h)
	}
}
func (f *Field) renderOp(dst *ebiten.Image, op *ebiten.DrawImageOptions) {
	if f.renderZoom > 0 && (dst == f.Canvas || dst == f.shadows) {
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
func (f *Field) renderBackground(dst *ebiten.Image) {
	p := f.A.pattern()
	z := f.renderZoom
	w, h := 64*z, 32*z
	ox, oy := -f.renderX*z, -f.renderY*z
	for y := math.Mod(oy, h) - h; y < float64(dst.Bounds().Dy()); y += h {
		for x := math.Mod(ox, w) - w; x < float64(dst.Bounds().Dx()); x += w {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(x, y)
			if !f.ShowExterior {
				op.ColorScale.Scale(.45, .45, .45, 1)
			}
			dst.DrawImage(p, op)
		}
	}
}
