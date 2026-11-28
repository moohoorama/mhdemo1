package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// floodTile is the flood water over one small tile (scenemap.DrawFlood), 32 x 16: the rules' muddy colour over its
// diamond, thinning toward the sides that meet dry ground and rimmed with foam there, with a few lighter ripples
// that differ by variant so the sheet does not repeat tile by tile.
func (a *Assets) floodTile(tile int) *ebiten.Image {
	if img, ok := a.floods[tile]; ok {
		return img
	}
	if a.floods == nil {
		a.floods = map[int]*ebiten.Image{}
	}
	fl := a.SceneRules.Rules.Flood
	water, foam := hexColor(fl.Color), hexColor(fl.Foam)
	sides, variant := tile&15, tile>>4
	pix := make([]byte, 32*16*4)
	for y := range 16 {
		for x := range 32 {
			cx, cy := float64(x)+.5-16, float64(y)+.5-8
			du, dv := cx/32+cy/16, cy/16-cx/32 // toward the +u and +v edges, ±.5 on them
			if math.Abs(du) > .5 || math.Abs(dv) > .5 {
				continue
			}
			shore := 1.0 // how far, in small tiles, to the nearest side on dry ground
			for bit, d := range []float64{dv + .5, .5 - du, .5 - dv, du + .5} {
				if sides&(1<<bit) != 0 {
					shore = math.Min(shore, d)
				}
			}
			c, alpha := water, fl.Alpha
			switch {
			case shore < .09:
				c, alpha = mixRGBA(water, foam, .6), .7
			case shore < .4:
				alpha *= .55 + .45*shore/.4 // shallow by the bank: the ground shows more
			}
			if shore >= .09 {
				switch rippled(x, y, variant) {
				case 1:
					c = mixRGBA(c, foam, .45)
				case 2:
					c = mixRGBA(c, color.RGBA{A: 255}, .18)
				}
			}
			i := 4 * (y*32 + x)
			pix[i], pix[i+1], pix[i+2], pix[i+3] = premul(c.R, alpha), premul(c.G, alpha), premul(c.B, alpha), uint8(alpha*255+.5)
		}
	}
	img := ebiten.NewImage(32, 16)
	img.WritePixels(pix)
	a.floods[tile] = img
	return img
}

// rippled marks the short streaks on the flood, four to a tile laid by variant: 1 for a streak's light top, 2 for
// its dark underside, 0 elsewhere.
func rippled(x, y, variant int) int {
	for k := range 4 {
		h := hashInt(variant*7+k, 31)
		rx, ry, n := 4+int(h%22), 3+int(h/22%10), 2+int(h/220%3)
		switch {
		case y == ry && x >= rx && x < rx+n:
			return 1
		case y == ry+1 && x > rx && x <= rx+n:
			return 2
		}
	}
	return 0
}

func hashInt(a, b int) uint32 {
	h := uint32(a)*374761393 + uint32(b)*668265263
	h = (h ^ h>>13) * 1274126177
	return h ^ h>>16
}

func premul(v uint8, alpha float64) uint8 { return uint8(float64(v)*alpha + .5) }

func mixRGBA(a, b color.RGBA, k float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-k) + float64(y)*k + .5) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// waterline is the foam line along where the flood meets something standing in it (scenemap.DrawWaterline: its
// left, bottom and right footprint corners, canvas points), and where its top left sits.
func (a *Assets) waterline(pts [][2]int) (*ebiten.Image, image.Point) {
	if len(pts) != 3 {
		return nil, image.Point{}
	}
	w, s, e := pts[0], pts[1], pts[2]
	x0, y0 := w[0], min(w[1], e[1])-1
	key := [4]int{s[0] - w[0], s[1] - w[1], e[0] - w[0], e[1] - w[1]}
	if img, ok := a.waterlines[key]; ok {
		return img, image.Pt(x0, y0)
	}
	if a.waterlines == nil {
		a.waterlines = map[[4]int]*ebiten.Image{}
	}
	foam := hexColor(a.SceneRules.Rules.Flood.Foam)
	width, height := e[0]-x0+1, s[1]-y0+2
	pix := make([]byte, width*height*4)
	set := func(x, y int, alpha float64) {
		if x < 0 || y < 0 || x >= width || y >= height {
			return
		}
		i := 4 * (y*width + x)
		pix[i], pix[i+1], pix[i+2], pix[i+3] = premul(foam.R, alpha), premul(foam.G, alpha), premul(foam.B, alpha), uint8(alpha*255+.5)
	}
	for x := w[0]; x <= e[0]; x++ {
		var y float64
		if x <= s[0] {
			y = float64(w[1]) + float64(s[1]-w[1])*float64(x-w[0])/math.Max(1, float64(s[0]-w[0]))
		} else {
			y = float64(s[1]) + float64(e[1]-s[1])*float64(x-s[0])/math.Max(1, float64(e[0]-s[0]))
		}
		set(x-x0, int(math.Round(y))-y0, .85)
		if (x/2)%2 == 0 {
			set(x-x0, int(math.Round(y))-y0+1, .4)
		}
	}
	img := ebiten.NewImage(width, height)
	img.WritePixels(pix)
	a.waterlines[key] = img
	return img, image.Pt(x0, y0)
}
