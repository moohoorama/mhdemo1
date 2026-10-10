package sprite

import (
	"image"
	"image/color"
	"math"
)

// Groups lists the tag groups the sheet uses, in tag-table order.
func (s *Sheet) Groups() []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range s.Tags {
		if !seen[t.Group] {
			seen[t.Group] = true
			out = append(out, t.Group)
		}
	}
	return out
}

// Render draws the whole sheet: tagged pixels take their tag's colour (palette p over the defaults) and keep
// the palette pixel's alpha; untagged pixels use the palette colour.
func (s *Sheet) Render(p Palette) *image.RGBA {
	cur := make([]color.RGBA, len(s.Tags)+1)
	for i, t := range s.Tags {
		cur[i+1] = t.Color
		if cs := p[t.Group]; t.Shade <= len(cs) && cs[t.Shade-1].A != 0 {
			cur[i+1] = cs[t.Shade-1]
		}
	}
	lut := make([][256]color.RGBA, len(cur))
	for t := range lut {
		for i, c := range s.Colors {
			if t > 0 {
				c.R, c.G, c.B = cur[t].R, cur[t].G, cur[t].B
			}
			lut[t][i] = color.RGBAModel.Convert(color.NRGBA(c)).(color.RGBA)
		}
	}
	w, h, cols := s.Cell[0], s.Cell[1], s.Columns
	img := image.NewRGBA(image.Rect(0, 0, w*cols, h*len(s.Rows)))
	for c := 0; c < len(s.Rows)*cols; c++ {
		x0, y0 := c%cols*w, c/cols*h
		for y := 0; y < h; y++ {
			o := img.PixOffset(x0, y0+y)
			for x, i := 0, c*w*h+y*w; x < w; x, i = x+1, i+1 {
				v := lut[s.tag[i]][s.pix[i]]
				img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = v.R, v.G, v.B, v.A
				o += 4
			}
		}
	}
	return img
}

// Shadow constants, as in ver3/shadows.py.
const (
	grow, core, halo, blur = 1, 130, 110, 3
	shadowPad              = grow + 2*blur + 1
)

var slope = [2]float64{.6, .3}

// Shadow is a black image whose alpha is the shadow: same rows and columns as the sheet, cells of Cell with Pivot.
type Shadow struct {
	Image *image.RGBA
	Cell  [2]int
	Pivot [2]int
}

// Shadow returns the sheet's shadow, built on first use.
func (s *Sheet) Shadow() *Shadow {
	s.once.Do(s.makeShadow)
	return s.shadow
}

func (s *Sheet) makeShadow() {
	w, h := s.Cell[0], s.Cell[1]
	px, py := s.Pivot[0], s.Pivot[1]
	depth := int(math.Ceil(float64(py) * slope[1]))
	sw, sh := w+int(math.Ceil(float64(py)*slope[0]))+2*shadowPad, depth+1+2*shadowPad
	sd := &Shadow{Cell: [2]int{sw, sh}, Pivot: [2]int{px + shadowPad, shadowPad}}
	sd.Image = image.NewRGBA(image.Rect(0, 0, sw*s.Columns, sh*len(s.Rows)))
	flat := make([]uint8, sw*sh)
	for c := 0; c < len(s.Rows)*s.Columns; c++ {
		clear(flat)
		for d := 0; d <= depth; d++ {
			src := int(math.RoundToEven(float64(py) - float64(d)/slope[1]))
			if src < 0 {
				break
			}
			x, y := shadowPad+int(math.RoundToEven(float64(d)/slope[1]*slope[0])), shadowPad+d
			for i := 0; i < w && src < h; i++ {
				if s.Colors[s.pix[c*w*h+src*w+i]].A != 0 {
					flat[y*sw+x+i] = 255
				}
			}
		}
		grown := boxMax(flat, sw, sh)
		blurred := boxBlur(boxBlur(grown, sw, sh), sw, sh)
		x0, y0 := c%s.Columns*sw, c/s.Columns*sh
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				v := int(blurred[y*sw+x]) * halo / 255
				if grown[y*sw+x] != 0 {
					v = max(v, core)
				}
				sd.Image.Pix[sd.Image.PixOffset(x0+x, y0+y)+3] = uint8(v)
			}
		}
	}
	s.shadow = sd
}

func boxMax(a []uint8, w, h int) []uint8 {
	out := make([]uint8, len(a))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for dy := -grow; dy <= grow; dy++ {
				for dx := -grow; dx <= grow; dx++ {
					if xx, yy := x+dx, y+dy; xx >= 0 && yy >= 0 && xx < w && yy < h {
						out[y*w+x] = max(out[y*w+x], a[yy*w+xx])
					}
				}
			}
		}
	}
	return out
}

// boxBlur is a (2*blur+1)-wide box blur with zero outside the image.
func boxBlur(a []uint8, w, h int) []uint8 {
	tmp, out := make([]uint8, len(a)), make([]uint8, len(a))
	pass := func(dst, src []uint8, lines, n, stride, step int) {
		for l := 0; l < lines; l++ {
			for i := 0; i < n; i++ {
				sum := 0
				for j := max(0, i-blur); j <= min(n-1, i+blur); j++ {
					sum += int(src[l*stride+j*step])
				}
				dst[l*stride+i*step] = uint8((sum + blur) / (2*blur + 1))
			}
		}
	}
	pass(tmp, a, h, w, w, 1)
	pass(out, tmp, w, h, 1, w)
	return out
}
