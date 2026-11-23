package gui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// scrollBarStyle picks how the common scrollbar (tableBar) is drawn: an index into scrollBarStyles. 0 is the
// bar the game has always had; the others are the design options compared in the scrollbar demo.
var scrollBarStyle = 0

// scrollBarStyles draw a track and its thumb; c is the thumb's ink, which lights up when hovered or dragged.
var scrollBarStyles = [...]func(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA){
	barFlat,
	barSlip,
	barGroove,
	barGrip,
	barRuler,
	barInk,
}

// scrollBarNames are the labels of the styles in the scrollbar demo, in the order of scrollBarStyles.
var scrollBarNames = [...]string{"현행", "A 죽간", "B 금테 홈", "C 그립", "D 눈금", "E 먹"}

// lit is the thumb's hovered or dragged ink (table.go's tableBar).
func lit(c color.NRGBA) bool { return c.R >= 225 }

// barFlat is the bar as it has been: a translucent track and a plain thumb.
func barFlat(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y), float32(thumb.Dx()), float32(thumb.Dy()), c)
}

// barSlip (A): a bamboo slip, the track a dark rounded groove and the thumb a beige pill with two grip lines.
func barSlip(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	capsule(dst, track, color.NRGBA{R: 40, G: 32, B: 24, A: 200})
	capsule(dst, thumb.Inset(1), color.NRGBA{R: uint8(min(255, int(c.R)+20)), G: uint8(min(255, int(c.G)+20)), B: uint8(min(255, int(c.B)+10)), A: 255})
	mid := float32(thumb.Min.Y+thumb.Dy()/2) - 2
	rect(dst, float32(thumb.Min.X+2), mid, float32(thumb.Dx()-4), 1, color.NRGBA{R: 110, G: 92, B: 60, A: 255})
	rect(dst, float32(thumb.Min.X+2), mid+3, float32(thumb.Dx()-4), 1, color.NRGBA{R: 110, G: 92, B: 60, A: 255})
}

// barGroove (B): a gold outline track with ticks at both ends and a solid gold thumb, the ends read as the arrows.
func barGroove(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	strokeRect(dst, float32(track.Min.X)+.5, float32(track.Min.Y)+.5, float32(track.Dx()-1), float32(track.Dy()-1), 2, goldDim)
	gold := color.NRGBA{R: 226, G: 182, B: 93, A: 255}
	if lit(c) {
		gold = color.NRGBA{R: 255, G: 226, B: 150, A: 255}
	}
	rect(dst, float32(thumb.Min.X+2), float32(thumb.Min.Y+2), float32(thumb.Dx()-4), float32(thumb.Dy()-4), gold)
	rect(dst, float32(track.Min.X+2), float32(track.Min.Y+2), float32(track.Dx()-4), 3, goldDim)
	rect(dst, float32(track.Min.X+2), float32(track.Max.Y-5), float32(track.Dx()-4), 3, goldDim)
}

// barGrip (C): a wide thumb with three dark grip ridges, on a track drawn as a thin line.
func barGrip(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	rect(dst, float32(track.Min.X+4), float32(track.Min.Y), 2, float32(track.Dy()), color.NRGBA{R: 255, G: 255, B: 255, A: 40})
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y), float32(thumb.Dx()), float32(thumb.Dy()), c)
	rect(dst, float32(thumb.Min.X), float32(thumb.Max.Y-2), float32(thumb.Dx()), 2, color.NRGBA{R: 40, G: 30, B: 20, A: 255})
	cy := float32(thumb.Min.Y+thumb.Dy()/2) - 3
	for k := range 3 {
		rect(dst, float32(thumb.Min.X+2), cy+float32(k)*3, float32(thumb.Dx()-4), 1, color.NRGBA{R: 60, G: 46, B: 28, A: 255})
	}
}

// barRuler (D): a ruler of ticks, one every 5 px down the track, and a gold block over the ticks the view covers.
func barRuler(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	for y := track.Min.Y; y < track.Max.Y; y += 5 {
		rect(dst, float32(track.Min.X), float32(y), float32(track.Dx()), 1, color.NRGBA{R: 225, G: 205, B: 150, A: 200})
	}
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y), float32(thumb.Dx()), float32(thumb.Dy()), color.NRGBA{R: 226, G: 182, B: 93, A: 255})
	strokeRect(dst, float32(thumb.Min.X)+.5, float32(thumb.Min.Y)+.5, float32(thumb.Dx()-1), float32(thumb.Dy()-1), 1, color.NRGBA{R: 40, G: 28, B: 14, A: 255})
}

// barInk (E): an ink stroke, the track a faint wash and the thumb near-black with a gold rim on its left edge.
func barInk(dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	for k := range 4 {
		rect(dst, float32(track.Min.X+k*2), float32(track.Min.Y), 2, float32(track.Dy()), color.NRGBA{R: 30, G: 24, B: 20, A: uint8(20 + k*8)})
	}
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y+2), float32(thumb.Dx()), float32(thumb.Dy()-4), color.NRGBA{R: 26, G: 20, B: 16, A: 245})
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y+2), 2, float32(thumb.Dy()-4), color.NRGBA{R: 226, G: 182, B: 93, A: 255})
	if lit(c) {
		strokeRect(dst, float32(thumb.Min.X), float32(thumb.Min.Y), float32(thumb.Dx()), float32(thumb.Dy()), 1, color.NRGBA{R: 255, G: 226, B: 150, A: 255})
	}
}

// capsule fills r with its short sides rounded.
func capsule(dst *ebiten.Image, r image.Rectangle, c color.NRGBA) {
	rad := float32(r.Dx()) / 2
	rect(dst, float32(r.Min.X), float32(r.Min.Y)+rad, float32(r.Dx()), float32(r.Dy())-2*rad, c)
	vectorCircle(dst, float32(r.Min.X)+rad, float32(r.Min.Y)+rad, rad, c)
	vectorCircle(dst, float32(r.Min.X)+rad, float32(r.Max.Y)-rad, rad, c)
}

// vectorCircle fills the circle at (x, y) with radius r in the units of the frame being drawn, as rect does.
func vectorCircle(dst *ebiten.Image, x, y, r float32, c color.Color) {
	s, ox, oy := float32(drawPx), float32(drawOX), float32(drawOY)
	vector.FillCircle(dst, x*s+ox, y*s+oy, r*s, c, true)
}
