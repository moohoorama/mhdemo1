package gui

import (
	"image"
	"image/color"

	"srpg/internal/sprite"
)

// ropeDots is the rope round a bound prisoner (a scene cast's Bound): two wraps over the arms and the waist, each
// a twisted strand over its shade, its ends curling round behind the body a dot higher. One pattern fits every
// non-combat sheet, whose bodies share one frame: the top-left dot sits ropeAt from the pivot.
var ropeDots = []string{
	"KRK..........KRK",
	"KdRrRrRrRrRrRrdK",
	".KddddddddddddK.",
	"................",
	"KRK..........KRK",
	"KdRrRrRrRrRrRrdK",
	".KddddddddddddK.",
}

var (
	ropeAt     = image.Point{-8, -15}
	ropeColors = map[byte]color.RGBA{'K': {40, 24, 12, 255}, 'R': {214, 178, 112, 255}, 'r': {170, 128, 66, 255}, 'd': {112, 76, 36, 255}}
	// ropeArms are the rows, from the pivot, where a hand swinging out past the rope is cut off: arms bound to the body.
	ropeArms = [2]int{-17, -8}
)

// boundArt is art with the rope drawn into every frame.
func boundArt(art *sprite.Art) *sprite.Art { return art.Variant("bound", bindRope) }

// bindRope draws the rope into every cell of a rendered sheet and cuts off whatever sticks out sideways past it.
func bindRope(img *image.RGBA, cell, pivot [2]int) {
	w := len(ropeDots[0])
	for y0 := img.Rect.Min.Y; y0 < img.Rect.Max.Y; y0 += cell[1] {
		for x0 := img.Rect.Min.X; x0 < img.Rect.Max.X; x0 += cell[0] {
			px, py := x0+pivot[0], y0+pivot[1]
			for y := py + ropeArms[0]; y <= py+ropeArms[1]; y++ {
				for x := x0; x < x0+cell[0]; x++ {
					if x < px+ropeAt.X || x >= px+ropeAt.X+w {
						img.SetRGBA(x, y, color.RGBA{})
					}
				}
			}
			for dy, row := range ropeDots {
				for dx := 0; dx < w; dx++ {
					if c, ok := ropeColors[row[dx]]; ok {
						img.SetRGBA(px+ropeAt.X+dx, py+ropeAt.Y+dy, c)
					}
				}
			}
		}
	}
}
