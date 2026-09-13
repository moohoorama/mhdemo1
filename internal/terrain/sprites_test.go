package terrain

import (
	"image"
	"image/color"
	"testing"
)

func TestBlackOutlineKeepsArtworkAndAddsOnlyOnePixel(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	ink := color.NRGBA{120, 160, 80, 255}
	source.SetNRGBA(0, 0, ink)
	source.SetNRGBA(1, 0, ink)
	source.SetNRGBA(1, 1, ink)
	got := blackOutline(source)
	if got.Bounds() != image.Rect(0, 0, 5, 5) {
		t.Fatal("outline padding")
	}
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			c := got.NRGBAAt(x, y)
			if source.NRGBAAt(x-1, y-1).A != 0 {
				if c != ink {
					t.Fatal("artwork changed")
				}
				continue
			}
			border := false
			for _, d := range []image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				if source.NRGBAAt(x-1+d.X, y-1+d.Y).A != 0 {
					border = true
				}
			}
			if border {
				if c != (color.NRGBA{0, 0, 0, 255}) {
					t.Fatal("outline not opaque black")
				}
			} else if c.A != 0 {
				t.Fatal("outline thicker than one pixel")
			}
		}
	}
}
