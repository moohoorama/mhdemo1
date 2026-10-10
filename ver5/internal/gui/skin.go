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
	muted    = color.NRGBA{R: 196, G: 188, B: 166, A: 255}
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

// scale maps the fixed Width×Height layout onto the window's device pixels. Everything
// drawn on the screen goes through the helpers below, which apply it.
var scale = 1.0

func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	s := float32(scale)
	vector.FillRect(dst, x*s, y*s, w*s, h*s, c, false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h, width float32, c color.Color) {
	s := float32(scale)
	vector.StrokeRect(dst, x*s, y*s, w*s, h*s, width*s, c, false)
}

func strokeLine(dst *ebiten.Image, x0, y0, x1, y1, width float32, c color.Color) {
	s := float32(scale)
	vector.StrokeLine(dst, x0*s, y0*s, x1*s, y1*s, width*s, c, true)
}

func fillCircle(dst *ebiten.Image, x, y, r float32, c color.Color) {
	s := float32(scale)
	vector.FillCircle(dst, x*s, y*s, r*s, c, true)
}

func scaled(p *vector.Path) *vector.Path {
	var q vector.Path
	o := &vector.AddPathOptions{}
	o.GeoM.Scale(scale, scale)
	q.AddPath(p, o)
	return &q
}

func fillPath(dst *ebiten.Image, p *vector.Path, op *vector.DrawPathOptions) {
	vector.FillPath(dst, scaled(p), nil, op)
}

func strokePath(dst *ebiten.Image, p *vector.Path, so vector.StrokeOptions, op *vector.DrawPathOptions) {
	so.Width *= float32(scale)
	vector.StrokePath(dst, scaled(p), &so, op)
}

// drawScaled draws src with op's logical transform, then scales to device pixels.
func drawScaled(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) {
	op.GeoM.Scale(scale, scale)
	dst.DrawImage(src, op)
}

// face is Medium for body text and Bold from heading sizes up; the bundled variable
// font defaults to Thin.
func (g *Game) face(size float64) *text.GoTextFace {
	f := &text.GoTextFace{Source: g.font, Size: size}
	weight := float32(500)
	if size >= 19 {
		weight = 700
	}
	f.SetVariation(text.MustParseTag("wght"), weight)
	return f
}

func (g *Game) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*scale, y*scale)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.45 * scale
	f := g.face(size)
	f.Size *= scale
	text.Draw(dst, s, f, op)
}

func (g *Game) width(s string, size float64) float64 {
	f := g.face(size)
	f.Size *= scale
	w, _ := text.Measure(s, f, size*1.45*scale)
	return w / scale
}

// centered draws s with its centre at x.
func (g *Game) centered(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	g.label(dst, s, x-g.width(s, size)/2, y, size, c)
}

// outlined draws white text with a dark rim, for text over the battlefield.
func (g *Game) outlined(dst *ebiten.Image, s string, x, y, size float64) {
	for _, d := range [][2]float64{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
		g.label(dst, s, x+d[0], y+d[1], size, color.NRGBA{A: 230})
	}
	g.label(dst, s, x, y, size, color.White)
}

// wrap breaks lines at n runes, at the last space before the limit when there is one.
func wrap(s string, n int) string {
	out := []string{}
	for _, line := range strings.Split(s, "\n") {
		r := []rune(line)
		for len(r) > n {
			cut := n
			for i := n; i > n/2; i-- {
				if r[i] == ' ' {
					cut = i
					break
				}
			}
			out = append(out, strings.TrimRight(string(r[:cut]), " "))
			r = []rune(strings.TrimLeft(string(r[cut:]), " "))
		}
		out = append(out, string(r))
	}
	return strings.Join(out, "\n")
}

// with picks the 와/과 particle for a Korean name.
func with(name string) string {
	r := []rune(name)
	if len(r) > 0 && r[len(r)-1] >= 0xAC00 && r[len(r)-1] <= 0xD7A3 && (r[len(r)-1]-0xAC00)%28 != 0 {
		return name + "과"
	}
	return name + "와"
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
	strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, color.NRGBA{R: 12, G: 8, B: 6, A: 255})
	strokeRect(dst, x+2, y+2, w-4, h-4, 2, gold)
	strokeRect(dst, x+6, y+6, w-12, h-12, 1, goldDim)
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
	strokeRect(dst, x, float32(r.Min.Y)-12, float32(w), 26, 2, gold)
	g.label(dst, title, float64(x)+18, float64(r.Min.Y)-11, 17, ink)
}

// cursorPos is the mouse position in layout coordinates; screenshot drivers replace it.
var cursorPos = func() (int, int) {
	x, y := ebiten.CursorPosition()
	return int(float64(x) / scale), int(float64(y) / scale)
}

func hovered(r image.Rectangle) bool {
	x, y := cursorPos()
	return image.Pt(x, y).In(r)
}

// btn registers a clickable button; a nil click draws it disabled.
func (g *Game) btn(dst *ebiten.Image, r image.Rectangle, label string, click func()) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	body, rim, fg := lacquer2, goldDim, ink
	if click == nil {
		body, fg = color.NRGBA{R: 40, G: 34, B: 30, A: 235}, color.NRGBA{R: 138, G: 130, B: 118, A: 255}
	} else {
		g.buttons = append(g.buttons, button{r, label, click})
		if hovered(r) {
			body, rim = hoverC, gold
		}
	}
	rect(dst, x, y, w, h, body)
	strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, rim)
	rect(dst, x+1, y+1, w-2, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 28})
	size := 15.0
	if h < 28 {
		size = 13
	}
	g.label(dst, label, float64(x)+10, float64(y)+(float64(h)-size*1.45)/2, size, fg)
}

// arrow is a scroll button: raised with a white mark while it can scroll, flat and dark
// (and not clickable) when it cannot.
func (g *Game) arrow(dst *ebiten.Image, r image.Rectangle, up bool, click func()) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	fg := color.Color(color.White)
	if click == nil {
		rect(dst, x, y, w, h, color.NRGBA{R: 30, G: 24, B: 20, A: 235})
		strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, color.NRGBA{R: 58, G: 48, B: 40, A: 255})
		fg = color.NRGBA{R: 74, G: 66, B: 58, A: 255}
	} else {
		g.buttons = append(g.buttons, button{r, "", click})
		body := lacquer2
		if hovered(r) {
			body = hoverC
		}
		rect(dst, x, y, w, h, body)
		rect(dst, x, y, w, 2, color.NRGBA{R: 255, G: 244, B: 220, A: 150})
		rect(dst, x, y, 2, h, color.NRGBA{R: 255, G: 244, B: 220, A: 150})
		rect(dst, x, y+h-2, w, 2, color.NRGBA{A: 200})
		rect(dst, x+w-2, y, 2, h, color.NRGBA{A: 200})
	}
	var p vector.Path
	cx, cy := x+w/2, y+h/2
	if up {
		p.MoveTo(cx-7, cy+4)
		p.LineTo(cx+7, cy+4)
		p.LineTo(cx, cy-4)
	} else {
		p.MoveTo(cx-7, cy-4)
		p.LineTo(cx+7, cy-4)
		p.LineTo(cx, cy+4)
	}
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(fg)
	fillPath(dst, &p, op)
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

// tooltip draws wrapped help text in a small window at (x, y), kept on screen.
func (g *Game) tooltip(dst *ebiten.Image, s string, x, y int) {
	s = wrap(s, 20)
	lines := strings.Count(s, "\n") + 1
	w, h := 300, 20+lines*20
	if x+w > Width-8 {
		x = max(8, x-w-330)
	}
	y = max(48, min(Height-h-8, y))
	r := image.Rect(x, y, x+w, y+h)
	window(dst, r)
	g.label(dst, s, float64(x+14), float64(y+9), 13, ink)
}
