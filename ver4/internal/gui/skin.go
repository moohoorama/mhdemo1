package gui

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	ink      = color.NRGBA{R: 236, G: 228, B: 206, A: 255}
	muted    = color.NRGBA{R: 168, G: 160, B: 140, A: 255}
	gold     = color.NRGBA{R: 226, G: 182, B: 93, A: 255}
	goldDim  = color.NRGBA{R: 132, G: 104, B: 52, A: 255}
	lacquer  = color.NRGBA{R: 44, G: 30, B: 24, A: 240}
	lacquer2 = color.NRGBA{R: 62, G: 42, B: 30, A: 240}
	hoverC   = color.NRGBA{R: 96, G: 64, B: 38, A: 255}
	good     = color.NRGBA{R: 132, G: 214, B: 120, A: 255}
	bad      = color.NRGBA{R: 236, G: 104, B: 86, A: 255}
	hpC      = color.NRGBA{R: 104, G: 186, B: 96, A: 255}
	mpC      = color.NRGBA{R: 92, G: 156, B: 216, A: 255}
	xpC      = color.NRGBA{R: 226, G: 182, B: 93, A: 255}
	trough   = color.NRGBA{R: 18, G: 14, B: 12, A: 255}
)

type button struct {
	rect  image.Rectangle
	label string
	click func()
}

func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.FillRect(dst, x, y, w, h, c, false)
}

func (g *Game) face(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: g.font, Size: size}
}

func (g *Game) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.45
	text.Draw(dst, s, g.face(size), op)
}

func (g *Game) width(s string, size float64) float64 {
	w, _ := text.Measure(s, g.face(size), size*1.45)
	return w
}

// centered draws s with its centre at x.
func (g *Game) centered(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	g.label(dst, s, x-g.width(s, size)/2, y, size, c)
}

// outlined draws text with a dark rim, for text over the battlefield.
func (g *Game) outlined(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		g.label(dst, s, x+d[0], y+d[1], size, color.NRGBA{A: 220})
	}
	g.label(dst, s, x, y, size, c)
}

func wrap(s string, n int) string {
	out := []string{}
	for _, line := range strings.Split(s, "\n") {
		r := []rune(line)
		for len(r) > n {
			out = append(out, string(r[:n]))
			r = r[n:]
		}
		out = append(out, string(r))
	}
	return strings.Join(out, "\n")
}

// window draws the shared frame: lacquered body, gold double rim and corner studs.
func window(dst *ebiten.Image, r image.Rectangle) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	rect(dst, x+3, y+4, w, h, color.NRGBA{A: 90})
	rect(dst, x, y, w, h, lacquer)
	// a soft top-lit gradient
	for i := float32(0); i < 8; i++ {
		band := (h - 8) / 16
		rect(dst, x+4, y+4+i*band, w-8, band, color.NRGBA{R: 255, G: 210, B: 160, A: uint8(14 - i*1.7)})
	}
	vector.StrokeRect(dst, x+.5, y+.5, w-1, h-1, 1, color.NRGBA{R: 12, G: 8, B: 6, A: 255}, false)
	vector.StrokeRect(dst, x+2, y+2, w-4, h-4, 2, gold, false)
	vector.StrokeRect(dst, x+6, y+6, w-12, h-12, 1, goldDim, false)
	for _, c := range [][2]float32{{x + 2, y + 2}, {x + w - 8, y + 2}, {x + 2, y + h - 8}, {x + w - 8, y + h - 8}} {
		rect(dst, c[0], c[1], 6, 6, gold)
		rect(dst, c[0]+2, c[1]+2, 2, 2, lacquer)
	}
}

// titled draws a window with a title plate on its top edge.
func (g *Game) titled(dst *ebiten.Image, r image.Rectangle, title string) {
	window(dst, r)
	if title == "" {
		return
	}
	w := g.width(title, 17) + 36
	x := float32(r.Min.X) + 18
	rect(dst, x, float32(r.Min.Y)-12, float32(w), 26, color.NRGBA{R: 108, G: 30, B: 26, A: 255})
	vector.StrokeRect(dst, x, float32(r.Min.Y)-12, float32(w), 26, 2, gold, false)
	g.label(dst, title, float64(x)+18, float64(r.Min.Y)-11, 17, ink)
}

func hovered(r image.Rectangle) bool {
	x, y := ebiten.CursorPosition()
	return image.Pt(x, y).In(r)
}

// btn registers a clickable button; a nil click draws it disabled.
func (g *Game) btn(dst *ebiten.Image, r image.Rectangle, label string, click func()) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	body, rim, fg := lacquer2, goldDim, ink
	if click == nil {
		body, fg = color.NRGBA{R: 40, G: 34, B: 30, A: 235}, color.NRGBA{R: 110, G: 102, B: 92, A: 255}
	} else {
		g.buttons = append(g.buttons, button{r, label, click})
		if hovered(r) {
			body, rim = hoverC, gold
		}
	}
	rect(dst, x, y, w, h, body)
	vector.StrokeRect(dst, x+.5, y+.5, w-1, h-1, 1, rim, false)
	rect(dst, x+1, y+1, w-2, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 28})
	size := 15.0
	if h < 28 {
		size = 13
	}
	g.label(dst, label, float64(x)+10, float64(y)+(float64(h)-size*1.45)/2, size, fg)
}

// button keeps the older x, y, width form.
func (g *Game) button(dst *ebiten.Image, label string, x, y, w int, f func()) {
	g.btn(dst, image.Rect(x, y, x+w, y+32), label, f)
}

// block keeps map clicks from passing through a HUD panel.
func (g *Game) block(r image.Rectangle) { g.blocks = append(g.blocks, r) }

func bar(dst *ebiten.Image, x, y, w, h float32, k float64, c color.Color) {
	rect(dst, x, y, w, h, trough)
	k = max(0, min(1, k))
	rect(dst, x+1, y+1, (w-2)*float32(k), h-2, c)
}

func hpColor(k float64) color.NRGBA {
	switch {
	case k <= .25:
		return bad
	case k <= .5:
		return color.NRGBA{R: 232, G: 184, B: 64, A: 255}
	}
	return hpC
}
