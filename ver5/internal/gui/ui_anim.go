package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Shared animation and text helpers of the battle UI (docs/ui-mockup): easing curves,
// keyframe tracks, clipped reveals and cached text images.

func phase(t, at, dur float64) float64 { return max(0, min(1, (t-at)/dur)) }

// bezier is CSS cubic-bezier(x1, y1, x2, y2) evaluated at x.
func bezier(x1, y1, x2, y2, x float64) float64 {
	if x <= 0 || x >= 1 {
		return max(0, min(1, x))
	}
	lo, hi := 0.0, 1.0
	var t float64
	for range 30 {
		t = (lo + hi) / 2
		bx := 3*(1-t)*(1-t)*t*x1 + 3*(1-t)*t*t*x2 + t*t*t
		if bx < x {
			lo = t
		} else {
			hi = t
		}
	}
	return 3*(1-t)*(1-t)*t*y1 + 3*(1-t)*t*t*y2 + t*t*t
}

// keyframe interpolates a piecewise-linear track at progress k.
func keyframe(k float64, at []float64, vals []float64) float64 {
	for i := 1; i < len(at); i++ {
		if k <= at[i] {
			f := (k - at[i-1]) / (at[i] - at[i-1])
			return vals[i-1] + (vals[i]-vals[i-1])*f
		}
	}
	return vals[len(vals)-1]
}

// drawReveal draws the left fraction k of img at (x, y), feathering the moving edge so it
// reads like a brush tip rather than a cut.
func drawReveal(dst, img *ebiten.Image, x, y, k, alpha float64) {
	if k <= 0 || alpha <= 0 {
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	const feather = 48
	edge := int(k * float64(w+feather))
	solid := min(w, edge-feather)
	if solid > 0 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		op.ColorScale.ScaleAlpha(float32(alpha))
		drawScaled(dst, img.SubImage(image.Rect(0, 0, solid, h)).(*ebiten.Image), op)
	}
	for i := max(0, solid); i < min(w, edge); i += 4 {
		a := 1 - float64(i-solid)/feather
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x+float64(i), y)
		op.ColorScale.ScaleAlpha(float32(alpha * a))
		drawScaled(dst, img.SubImage(image.Rect(i, 0, min(w, i+4), h)).(*ebiten.Image), op)
	}
}

// textImage renders s once into a layout-sized image with a drop shadow, for clipping.
func (g *Game) textImage(s string, f *text.GoTextFace, c color.Color, shadow float64) *ebiten.Image {
	key := fmt.Sprint(s, f.Size, c, shadow)
	if im, ok := g.textCache[key]; ok {
		return im
	}
	if g.textCache == nil {
		g.textCache = map[string]*ebiten.Image{}
	}
	w := int(math.Ceil(measure(s, f))) + 8
	h := int(math.Ceil(f.Size*1.45 + shadow + 4))
	im := ebiten.NewImage(w, h)
	old := drawPx
	drawPx = 1
	write(im, s, f, 0, shadow, color.Black)
	write(im, s, f, 0, 0, c)
	drawPx = old
	g.textCache[key] = im
	return im
}

// writeSpaced draws s with extra spacing between letters, as a fraction of the face size.
func writeSpaced(dst *ebiten.Image, s string, f *text.GoTextFace, x, y, spacing float64, c color.Color) {
	for _, r := range s {
		write(dst, string(r), f, x, y, c)
		x += measure(string(r), f) + spacing*f.Size
	}
}

func spacedWidth(s string, f *text.GoTextFace, spacing float64) float64 {
	n := len([]rune(s))
	return measure(s, f) + spacing*f.Size*float64(max(0, n-1))
}

// wrapWidth breaks s into lines no wider than w in face f, at spaces where it can.
func wrapWidth(s string, tf *text.GoTextFace, w float64) string {
	out := []string{}
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Split(para, " ") {
			try := word
			if line != "" {
				try = line + " " + word
			}
			if line != "" && measure(try, tf) > w {
				out = append(out, line)
				line = word
			} else {
				line = try
			}
			for measure(line, tf) > w { // a single word wider than the line
				r := []rune(line)
				cut := len(r) - 1
				for cut > 1 && measure(string(r[:cut]), tf) > w {
					cut--
				}
				out = append(out, string(r[:cut]))
				line = string(r[cut:])
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

var courtesyName = regexp.MustCompile(`자는 ([가-힣]{1,3})[.,]`)

// courtesy is an officer's courtesy name from his biography ("자는 운장."), if written.
func (g *Game) courtesy(ref string) string {
	_, c, ok := g.character(ref)
	if !ok {
		return ""
	}
	if m := courtesyName.FindStringSubmatch(c.Biography); m != nil {
		return m[1]
	}
	return ""
}

// prefixLines keeps the first n runes of the wrapped text s, line breaks not counted.
func prefixLines(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' {
			b.WriteRune(r)
			continue
		}
		if n <= 0 {
			break
		}
		b.WriteRune(r)
		n--
	}
	return b.String()
}
