package graphics

import (
	"image"
	"image/draw"
	"sort"
)

// Position is the projected ground anchor, independent of the output camera.
type DrawItem struct {
	Sprite   int
	Position image.Point
	Layer    int
	Depth    int
}

// Stable ordering preserves ground overlay order and makes equal-depth ties deterministic.
func Sort(items []DrawItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Layer != items[j].Layer {
			return items[i].Layer < items[j].Layer
		}
		return items[i].Depth < items[j].Depth
	})
}

// RenderPNG uses the same sprite rectangles, pivots and ordering as the screen renderer.
// bounds is in projected world pixels, including the desired export margin.
func RenderPNG(c *Catalog, items []DrawItem, bounds image.Rectangle, scale int) *image.NRGBA {
	raw := image.NewNRGBA(image.Rectangle{Max: bounds.Size()})
	for _, item := range items {
		s := c.Sprites[item.Sprite]
		src := c.Rect(item.Sprite)
		p := item.Position.Sub(s.Pivot).Sub(bounds.Min)
		draw.Draw(raw, image.Rectangle{Min: p, Max: p.Add(src.Size())}, c.Images[s.Sheet], src.Min, draw.Over)
	}
	return Enlarge(raw, scale)
}

func Enlarge(src image.Image, scale int) *image.NRGBA {
	if scale < 1 {
		panic("image scale must be positive")
	}
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, src.At(b.Min.X+x/scale, b.Min.Y+y/scale))
		}
	}
	return out
}
