package gui

import (
	"image"
	"image/color"
	"math"
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

// Screen coordinates are in one of two spaces, each uniform (the same in x and y):
//   - UI px, uiPx device pixels each: HUD, windows, dialogue, banners. The player's UI size
//     setting changes uiPx, so every piece of UI grows together.
//   - world px, worldPx device pixels each: the battlefield on screen (tiles, units,
//     effects), which the camera zoom scales on top.
//
// The helpers below draw in the space drawPx names: uiPx, except inside inWorld.
var uiPx, worldPx, drawPx = 1.0, 1.0, 1.0

// setUnits fits the Width×Height layout to a w×h device-pixel screen.
func setUnits(w, h float64) {
	uiPx = min(w/Width, h/Height)
	worldPx, drawPx = uiPx, uiPx
}

// inWorld draws f in world px.
func inWorld(f func()) {
	drawPx = worldPx
	defer func() { drawPx = uiPx }()
	f()
}

// worldToUI maps a point on the battlefield's screen to UI px, for UI anchored to it.
func worldToUI(x, y float64) (float64, float64) {
	k := worldPx / uiPx
	return x * k, y * k
}

// cursorWorld is the cursor in world px, for picking map cells.
func cursorWorld() (float64, float64) {
	x, y := cursorPos()
	k := uiPx / worldPx
	return float64(x) * k, float64(y) * k
}

// uiZoom enlarges a block of UI about the layout point (uiOX, uiOY) on top of drawPx: the
// player's UI size setting, applied by withUI around panels anchored to a unit.
var uiZoom, uiOX, uiOY = 1.0, 0.0, 0.0

func ux(x float32) float32 { return (x-float32(uiOX))*float32(uiZoom) + float32(uiOX) }
func uy(y float32) float32 { return (y-float32(uiOY))*float32(uiZoom) + float32(uiOY) }

// toLayout maps a rectangle drawn under uiZoom to the layout rectangle it covers.
func toLayout(r image.Rectangle) image.Rectangle {
	return image.Rect(int(ux(float32(r.Min.X))), int(uy(float32(r.Min.Y))), int(math.Ceil(float64(ux(float32(r.Max.X))))), int(math.Ceil(float64(uy(float32(r.Max.Y))))))
}

// withUI draws f with the UI size setting applied about layout point (x, y).
func (g *Game) withUI(x, y float64, f func()) {
	uiZoom, uiOX, uiOY = g.uiScale(), x, y
	defer func() { uiZoom, uiOX, uiOY = 1, 0, 0 }()
	f()
}

// uiScale is the player's UI size, or the platform default when unset.
func (g *Game) uiScale() float64 {
	if p := g.S.Settings.UIScale; p > 0 {
		return float64(p) / 100
	}
	return float64(DefaultUIScale) / 100
}

// DefaultUIScale is the UI size in percent when the player has not chosen one; the
// Android build raises it for small screens.
var DefaultUIScale = 100

// uiBounds is the layout span, under uiZoom, that stays on screen with margin m.
func uiBounds(m int) image.Rectangle {
	z := uiZoom
	return image.Rect(int(uiOX+(float64(m)-uiOX)/z), int(uiOY+(float64(48)-uiOY)/z), int(uiOX+(float64(Width-m)-uiOX)/z), int(uiOY+(float64(Height-m)-uiOY)/z))
}

func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	s, z := float32(drawPx), float32(uiZoom)
	vector.FillRect(dst, ux(x)*s, uy(y)*s, w*z*s, h*z*s, c, false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h, width float32, c color.Color) {
	s, z := float32(drawPx), float32(uiZoom)
	vector.StrokeRect(dst, ux(x)*s, uy(y)*s, w*z*s, h*z*s, width*z*s, c, false)
}

func strokeLine(dst *ebiten.Image, x0, y0, x1, y1, width float32, c color.Color) {
	s := float32(drawPx)
	vector.StrokeLine(dst, ux(x0)*s, uy(y0)*s, ux(x1)*s, uy(y1)*s, width*float32(uiZoom)*s, c, true)
}

func fillCircle(dst *ebiten.Image, x, y, r float32, c color.Color) {
	s := float32(drawPx)
	vector.FillCircle(dst, ux(x)*s, uy(y)*s, r*float32(uiZoom)*s, c, true)
}

func scaled(p *vector.Path) *vector.Path {
	var q vector.Path
	o := &vector.AddPathOptions{}
	o.GeoM.Translate(-uiOX, -uiOY)
	o.GeoM.Scale(uiZoom, uiZoom)
	o.GeoM.Translate(uiOX, uiOY)
	o.GeoM.Scale(drawPx, drawPx)
	q.AddPath(p, o)
	return &q
}

func fillPath(dst *ebiten.Image, p *vector.Path, op *vector.DrawPathOptions) {
	vector.FillPath(dst, scaled(p), nil, op)
}

func strokePath(dst *ebiten.Image, p *vector.Path, so vector.StrokeOptions, op *vector.DrawPathOptions) {
	so.Width *= float32(drawPx * uiZoom)
	vector.StrokePath(dst, scaled(p), &so, op)
}

// drawScaled draws src with op's logical transform, then scales to device pixels.
func drawScaled(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) {
	op.GeoM.Translate(-uiOX, -uiOY)
	op.GeoM.Scale(uiZoom, uiZoom)
	op.GeoM.Translate(uiOX, uiOY)
	op.GeoM.Scale(drawPx, drawPx)
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

// brush is the brush-script face for turn titles, damage numbers and officer names.
func (g *Game) brush(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: g.brushFont, Size: size}
}

// serif is the bold Myeongjo face for the top bar, scrolls and name tags.
func (g *Game) serif(size float64, weight float32) *text.GoTextFace {
	f := &text.GoTextFace{Source: g.serifFont, Size: size}
	f.SetVariation(text.MustParseTag("wght"), weight)
	return f
}

// write draws s in face f at layout (x, y); op may carry extra colour or transform.
func write(dst *ebiten.Image, s string, f *text.GoTextFace, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(ux(float32(x)))*drawPx, float64(uy(float32(y)))*drawPx)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = f.Size * 1.45 * drawPx * uiZoom
	sf := *f
	sf.Size *= drawPx * uiZoom
	text.Draw(dst, s, &sf, op)
}

// measure is the layout width of s in face f.
func measure(s string, f *text.GoTextFace) float64 {
	w, _ := text.Measure(s, f, f.Size*1.45)
	return w
}

func (g *Game) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(ux(float32(x)))*drawPx, float64(uy(float32(y)))*drawPx)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.45 * drawPx * uiZoom
	f := g.face(size)
	f.Size *= drawPx * uiZoom
	text.Draw(dst, s, f, op)
}

func (g *Game) width(s string, size float64) float64 {
	f := g.face(size)
	f.Size *= drawPx
	w, _ := text.Measure(s, f, size*1.45*drawPx)
	return w / drawPx
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

// cursorPos is the mouse position in UI px; screenshot drivers replace it.
var cursorPos = func() (int, int) {
	x, y := ebiten.CursorPosition()
	if pointer.used {
		x, y = pointer.x, pointer.y
	}
	return int(float64(x) / uiPx), int(float64(y) / uiPx)
}

func hovered(r image.Rectangle) bool {
	x, y := cursorPos()
	return image.Pt(x, y).In(toLayout(r))
}

// btn registers a clickable button; a nil click draws it disabled.
func (g *Game) btn(dst *ebiten.Image, r image.Rectangle, label string, click func()) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	body, rim, fg := lacquer2, goldDim, ink
	if click == nil {
		body, fg = color.NRGBA{R: 40, G: 34, B: 30, A: 235}, color.NRGBA{R: 138, G: 130, B: 118, A: 255}
	} else {
		g.buttons = append(g.buttons, button{toLayout(r), label, click})
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
		g.buttons = append(g.buttons, button{toLayout(r), "", click})
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
func (g *Game) block(r image.Rectangle) { g.blocks = append(g.blocks, toLayout(r)) }

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
	b := uiBounds(8)
	if x+w > b.Max.X {
		x = max(b.Min.X, x-w-330)
	}
	y = max(b.Min.Y, min(b.Max.Y-h, y))
	r := image.Rect(x, y, x+w, y+h)
	window(dst, r)
	g.label(dst, s, float64(x+14), float64(y+9), 13, ink)
}
