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

// Shadow constants, as in tools/spritebuild/shadows.py.
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
	sw, sh := shadowCell(w, s.Pivot[1])
	sd := &Shadow{Cell: [2]int{sw, sh}, Pivot: [2]int{s.Pivot[0] + shadowPad, shadowPad}}
	sd.Image = image.NewRGBA(image.Rect(0, 0, sw*s.Columns, sh*len(s.Rows)))
	for c := 0; c < len(s.Rows)*s.Columns; c++ {
		alpha := castShadow(w, h, s.Pivot[1], func(x, y int) bool { return s.Colors[s.pix[c*w*h+y*w+x]].A != 0 })
		x0, y0 := c%s.Columns*sw, c/s.Columns*sh
		for y := 0; y < sh; y++ {
			for x := 0; x < sw; x++ {
				sd.Image.Pix[sd.Image.PixOffset(x0+x, y0+y)+3] = alpha[y*sw+x]
			}
		}
	}
	s.shadow = sd
}

// ImageShadow is the shadow of one picture standing on pivot, as a sheet's shadow is of each of its cells.
func ImageShadow(img *image.RGBA, pivot [2]int) *Shadow {
	b := img.Bounds()
	sw, sh := shadowCell(b.Dx(), pivot[1])
	sd := &Shadow{Image: image.NewRGBA(image.Rect(0, 0, sw, sh)), Cell: [2]int{sw, sh}, Pivot: [2]int{pivot[0] + shadowPad, shadowPad}}
	alpha := castShadow(b.Dx(), b.Dy(), pivot[1], func(x, y int) bool { return img.Pix[img.PixOffset(b.Min.X+x, b.Min.Y+y)+3] != 0 })
	for i, v := range alpha {
		sd.Image.Pix[4*i+3] = v
	}
	return sd
}

func shadowCell(w, py int) (int, int) {
	return w + int(math.Ceil(float64(py)*slope[0])) + 2*shadowPad, int(math.Ceil(float64(py)*slope[1])) + 1 + 2*shadowPad
}

// castShadow lays the opaque pixels of a w×h picture above row py flat on the ground and returns the shadow's
// alpha in a shadowCell.
func castShadow(w, h, py int, opaque func(x, y int) bool) []uint8 {
	sw, sh := shadowCell(w, py)
	depth := int(math.Ceil(float64(py) * slope[1]))
	flat := make([]uint8, sw*sh)
	for d := 0; d <= depth; d++ {
		src := int(math.RoundToEven(float64(py) - float64(d)/slope[1]))
		if src < 0 {
			break
		}
		x, y := shadowPad+int(math.RoundToEven(float64(d)/slope[1]*slope[0])), shadowPad+d
		for i := 0; i < w && src < h; i++ {
			if opaque(i, src) {
				flat[y*sw+x+i] = 255
			}
		}
	}
	return soften(flat, sw, sh)
}

// MaskShadow is the shadow of a prop that knows where its shadow falls (a 3D prop's, laid flat from its model): the
// opaque pixels of mask, softened as a cast silhouette is. pivot is the prop's footprint centre in mask.
func MaskShadow(mask *image.RGBA, pivot [2]int) *Shadow {
	b := mask.Bounds()
	sw, sh := b.Dx()+2*shadowPad, b.Dy()+2*shadowPad
	flat := make([]uint8, sw*sh)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if mask.Pix[mask.PixOffset(b.Min.X+x, b.Min.Y+y)+3] != 0 {
				flat[(y+shadowPad)*sw+x+shadowPad] = 255
			}
		}
	}
	sd := &Shadow{Image: image.NewRGBA(image.Rect(0, 0, sw, sh)), Cell: [2]int{sw, sh}, Pivot: [2]int{pivot[0] + shadowPad, pivot[1] + shadowPad}}
	for i, v := range soften(flat, sw, sh) {
		sd.Image.Pix[4*i+3] = v
	}
	return sd
}

// soften grows the shadow's shape and fills it with core alpha inside a blurred halo, in place.
func soften(flat []uint8, sw, sh int) []uint8 {
	grown := boxMax(flat, sw, sh)
	blurred := boxBlur(boxBlur(grown, sw, sh), sw, sh)
	for i := range blurred {
		v := int(blurred[i]) * halo / 255
		if grown[i] != 0 {
			v = max(v, core)
		}
		flat[i] = uint8(v)
	}
	return flat
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
