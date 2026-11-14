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
	rect   image.Rectangle
	label  string
	click  func()
	denied bool // a greyed-out row: a click on it only sounds the refusal
}

// Screen coordinates are in one of two spaces, each uniform (the same in x and y):
//   - UI px, uiPx device pixels each: HUD, windows, dialogue, banners. The player's UI size
//     setting changes uiPx, so every piece of UI grows together.
//   - world px, worldPx device pixels each: the battlefield on screen (tiles, units,
//     effects), which the camera zoom scales on top.
//
// The helpers below draw in the space drawPx names: uiPx, except inside inWorld.
var uiPx, worldPx, drawPx = 1.0, 1.0, 1.0

// uiW×uiH is the UI canvas, in UI px; worldW×worldH is the battlefield's screen, in world px.
var uiW, uiH, worldW, worldH = Width, Height, Width, Height

// setUnits lays out a w×h device-pixel frame at base device pixels per world px, with the
// UI enlarged by ui (the UI size setting) on top.
func setUnits(w, h, base, ui float64) {
	worldPx, uiPx = base, base*ui
	drawPx = uiPx
	uiW, uiH = int(w/uiPx), int(h/uiPx)
	worldW, worldH = int(math.Ceil(w/worldPx)), int(math.Ceil(h/worldPx))
}

// fixedAt is where the fixed Width×Height canvas sits in the window, in device pixels.
var fixedAt image.Point

// A frame is a fixed-size piece of UI — a window, or a screen laid out for a w×h design —
// drawn on its own image, centred, and shrunk when the screen is too small for it. While
// one is drawn, uiPx, uiW and uiH describe the frame's design, drawOX/drawOY shift the
// helpers so the design's box lands on the image, frameAt is where the image sits on the
// screen in device pixels, frameMin the box's corner and framePx the uiPx outside, where
// clicks are tested.
var (
	drawOX, drawOY float64
	frameAt        image.Point
	frameMin       image.Point
	framePx        float64
)

// inFrame draws box, a part of a w×h design, on dst where it would fall in the design
// centred on the screen, kept on screen, and shrunk to stay margin UI px from the edges.
func (g *Game) inFrame(dst *ebiten.Image, w, h int, box image.Rectangle, margin int, draw func(dst *ebiten.Image)) {
	if g.frames == nil {
		g.frames = map[image.Point]*ebiten.Image{}
	}
	img, at := g.renderFrame(dst, w, h, box, margin, nil, draw)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	dst.DrawImage(img, op)
}

// renderFrame draws the frame inFrame describes on an image of its own — *keep, made or
// remade to size, or else one shared by size — and returns it with where it goes on dst.
func (g *Game) renderFrame(dst *ebiten.Image, w, h int, box image.Rectangle, margin int, keep **ebiten.Image, draw func(dst *ebiten.Image)) (*ebiten.Image, image.Point) {
	s := min(1, float64(uiW-2*margin)/float64(box.Dx()), float64(uiH-2*margin)/float64(box.Dy()))
	fw, fh := int(math.Ceil(float64(box.Dx())*uiPx*s)), int(math.Ceil(float64(box.Dy())*uiPx*s))
	var img *ebiten.Image
	if keep != nil {
		if *keep == nil || (*keep).Bounds().Dx() != fw || (*keep).Bounds().Dy() != fh {
			if *keep != nil {
				(*keep).Deallocate()
			}
			*keep = ebiten.NewImage(fw, fh)
		}
		img = *keep
	} else {
		key := image.Pt(fw, fh)
		if img = g.frames[key]; img == nil {
			img = ebiten.NewImage(fw, fh)
			g.frames[key] = img
		}
	}
	img.Clear()
	// where the box falls in the design centred on dst, moved only as far as it must to fit
	dw, dh := dst.Bounds().Dx(), dst.Bounds().Dy()
	at := image.Pt((dw-fw)/2, (dh-fh)/2)
	if s == 1 {
		at.X = max(0, min(dw-fw, int((float64(dw)-float64(w)*uiPx)/2+float64(box.Min.X)*uiPx)))
		at.Y = max(0, min(dh-fh, int((float64(dh)-float64(h)*uiPx)/2+float64(box.Min.Y)*uiPx)))
	}
	oldPx, oldW, oldH := uiPx, uiW, uiH
	framePx, frameAt, frameMin = uiPx, at, box.Min
	uiPx *= s
	drawPx, uiW, uiH = uiPx, w, h
	drawOX, drawOY = -float64(box.Min.X)*uiPx, -float64(box.Min.Y)*uiPx
	draw(img)
	uiPx, drawPx, uiW, uiH = oldPx, oldPx, oldW, oldH
	drawOX, drawOY = 0, 0
	framePx, frameAt, frameMin = 0, image.Point{}, image.Point{}
	return img, at
}

// inDesign draws box, a window on the Width×Height design, as a frame with room around it
// for a title plate and shadow.
func (g *Game) inDesign(dst *ebiten.Image, box image.Rectangle, draw func(dst *ebiten.Image)) {
	g.inFrame(dst, Width, Height, box.Inset(-24), 0, draw)
}

// unframed maps a rectangle in the frame being drawn to UI px on the screen.
func unframed(r image.Rectangle) image.Rectangle {
	if framePx == 0 {
		return r
	}
	f := func(p image.Point) image.Point {
		p = p.Sub(frameMin)
		return image.Pt(int(math.Round((float64(p.X)*uiPx+float64(frameAt.X))/framePx)), int(math.Round((float64(p.Y)*uiPx+float64(frameAt.Y))/framePx)))
	}
	return image.Rectangle{f(r.Min), f(r.Max)}
}

// addButton registers a clickable area drawn this frame.
func (g *Game) addButton(r image.Rectangle, label string, click func()) {
	if !g.buttonClip.Empty() {
		if r = r.Intersect(g.buttonClip); r.Empty() {
			return
		}
	}
	g.buttons = append(g.buttons, button{rect: unframed(r), label: label, click: click})
}

// fit picks the units for a w×h device-pixel window. A world px is as large as fits the
// Width×Height design in the window, so the design is never cut off and a window of another
// shape just shows more of the field and spreads the UI to its edges. The map editor and the
// duel gallery keep their fixed Width×Height layout, centred, at UI size 100%.
func (g *Game) fit(w, h float64) {
	base := min(w/Width, h/Height)
	fixedAt = image.Point{}
	if g.fixedLayout() {
		fixedAt = image.Pt(int((w-Width*base)/2), int((h-Height*base)/2))
		setUnits(Width*base, Height*base, base, 1)
		return
	}
	setUnits(w, h, base, g.uiScale())
}

// fixedLayout reports whether the screen is one laid out on a fixed Width×Height canvas.
func (g *Game) fixedLayout() bool { return g.editor != nil || g.gallery != nil }

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

// uiBounds is the UI canvas less margin m, below the top bar.
func uiBounds(m int) image.Rectangle { return image.Rect(m, 48, uiW-m, uiH-m) }

func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	s, ox, oy := float32(drawPx), float32(drawOX), float32(drawOY)
	vector.FillRect(dst, x*s+ox, y*s+oy, w*s, h*s, c, false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h, width float32, c color.Color) {
	s, ox, oy := float32(drawPx), float32(drawOX), float32(drawOY)
	vector.StrokeRect(dst, x*s+ox, y*s+oy, w*s, h*s, width*s, c, false)
}

func strokeLine(dst *ebiten.Image, x0, y0, x1, y1, width float32, c color.Color) {
	s, ox, oy := float32(drawPx), float32(drawOX), float32(drawOY)
	vector.StrokeLine(dst, x0*s+ox, y0*s+oy, x1*s+ox, y1*s+oy, width*s, c, true)
}

// fillCircle stamps the disc picture, so many dots cost one batched draw and no path to fill.
func fillCircle(dst *ebiten.Image, x, y, r float32, c color.Color) {
	s, ox, oy := float64(drawPx), drawOX, drawOY
	d := discImage()
	k := 2 * float64(r) * s / float64(d.Bounds().Dx())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(d.Bounds().Dx())/2, -float64(d.Bounds().Dy())/2)
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(float64(x)*s+ox, float64(y)*s+oy)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(d, op)
}

var discTex, whiteTex *ebiten.Image

// discImage is a white disc with a one-texel soft rim, the picture fillCircle stamps.
func discImage() *ebiten.Image {
	if discTex == nil {
		const n = 16
		img := image.NewNRGBA(image.Rect(0, 0, n, n))
		for y := range n {
			for x := range n {
				a := max(0, min(1, n/2-math.Hypot(float64(x)+.5-n/2, float64(y)+.5-n/2)))
				img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(math.Round(255 * a))})
			}
		}
		discTex = ebiten.NewImageFromImage(img)
	}
	return discTex
}

// whitePixel is one white pixel with white around it, a source for flat-coloured triangles: sample it at (1.5, 1.5).
func whitePixel() *ebiten.Image {
	if whiteTex == nil {
		im := ebiten.NewImage(3, 3)
		im.Fill(color.White)
		whiteTex = im.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	}
	return whiteTex
}

func scaled(p *vector.Path) *vector.Path {
	var q vector.Path
	o := &vector.AddPathOptions{}
	o.GeoM.Scale(drawPx, drawPx)
	o.GeoM.Translate(drawOX, drawOY)
	q.AddPath(p, o)
	return &q
}

func fillPath(dst *ebiten.Image, p *vector.Path, op *vector.DrawPathOptions) {
	vector.FillPath(dst, scaled(p), nil, op)
}

func strokePath(dst *ebiten.Image, p *vector.Path, so vector.StrokeOptions, op *vector.DrawPathOptions) {
	so.Width *= float32(drawPx)
	vector.StrokePath(dst, scaled(p), &so, op)
}

// drawScaled draws src with op's logical transform, then scales to device pixels.
func drawScaled(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) {
	op.GeoM.Scale(drawPx, drawPx)
	op.GeoM.Translate(drawOX, drawOY)
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
	op.GeoM.Translate(x*drawPx+drawOX, y*drawPx+drawOY)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = f.Size * 1.45 * drawPx
	sf := *f
	sf.Size *= drawPx
	text.Draw(dst, s, &sf, op)
}

// measure is the layout width of s in face f.
func measure(s string, f *text.GoTextFace) float64 {
	w, _ := text.Measure(s, f, f.Size*1.45)
	return w
}

func (g *Game) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*drawPx+drawOX, y*drawPx+drawOY)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.45 * drawPx
	f := g.face(size)
	f.Size *= drawPx
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

// window draws the shared frame: the lacquered body laid with the dialogue box's ruyi clouds at their full
// contrast, a top-lit wash, under the UI sheet's gold double rim and corner studs sliced nine ways.
func (g *Game) window(dst *ebiten.Image, r image.Rectangle) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	rect(dst, x+3, y+4, w, h, color.NRGBA{A: 90})
	rect(dst, x, y, w, h, lacquer)
	g.ruyiPanel(dst, float64(x+4), float64(y+4), float64(w-8), float64(h-8), lacquer, 1)
	op := &ebiten.DrawImageOptions{} // the wash: eight bands, each 1/16 of the body high
	op.GeoM.Scale(float64(w-8), float64(h-8)/16)
	op.GeoM.Translate(float64(x+4), float64(y+4))
	drawScaled(dst, g.assets.UI["window_glow"].Frames[0], op)
	nineSlice(dst, g.assets.UI["window_rim"].Frames[0], 8, r)
}

// ruyiPanel lays the ruyi clouds over x, y, w, h so that they average to c, their light and dark spread by k
// (1 keeps the dialogue box's contrast): c laid at 1-k and the clouds tinted c/mean over it at k.
func (g *Game) ruyiPanel(dst *ebiten.Image, x, y, w, h float64, c color.NRGBA, k float32) {
	if k < 1 {
		rect(dst, float32(x), float32(y), float32(w), float32(h), c)
	}
	m := float32(max(.01, g.assets.RuyiMean)) * 255
	tint := [3]float32{float32(c.R) / m, float32(c.G) / m, float32(c.B) / m}
	g.drawPattern(dst, g.assets.Ruyi, x, y, w, h, 0, 0, 1, tint, k)
}

// nineSlice stretches UI picture im (one art pixel per UI pixel) over r: its corner UI px
// corners keep their size while the edges and the middle stretch, nearest sampled like the
// solid rectangles of the rest of the UI.
func nineSlice(dst, im *ebiten.Image, corner int, r image.Rectangle) {
	b, c := im.Bounds(), corner
	src := [4][2]int{{b.Min.X, b.Min.Y}, {b.Min.X + c, b.Min.Y + c}, {b.Max.X - c, b.Max.Y - c}, {b.Max.X, b.Max.Y}}
	to := [4][2]int{{r.Min.X, r.Min.Y}, {r.Min.X + corner, r.Min.Y + corner}, {r.Max.X - corner, r.Max.Y - corner}, {r.Max.X, r.Max.Y}}
	for i := range 3 {
		for j := range 3 {
			part := im.SubImage(image.Rect(src[i][0], src[j][1], src[i+1][0], src[j+1][1])).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(to[i+1][0]-to[i][0])/float64(src[i+1][0]-src[i][0]), float64(to[j+1][1]-to[j][1])/float64(src[j+1][1]-src[j][1]))
			op.GeoM.Translate(float64(to[i][0]), float64(to[j][1]))
			drawScaled(dst, part, op)
		}
	}
}

// titled draws a window with a title plate on its top edge.
func (g *Game) titled(dst *ebiten.Image, r image.Rectangle, title string) {
	g.window(dst, r)
	g.titlePlate(dst, r, title)
}

// titlePlate draws the title plate on r's top edge, over whatever the window's inside holds.
func (g *Game) titlePlate(dst *ebiten.Image, r image.Rectangle, title string) {
	if title == "" {
		return
	}
	w := g.width(title, 17) + 36
	x := float32(r.Min.X) + 18
	rect(dst, x, float32(r.Min.Y)-12, float32(w), 26, color.NRGBA{R: 108, G: 30, B: 26, A: 255})
	strokeRect(dst, x, float32(r.Min.Y)-12, float32(w), 26, 2, gold)
	g.label(dst, title, float64(x)+18, float64(r.Min.Y)-11, 17, ink)
}

func (g *Game) titledPattern(dst *ebiten.Image, r image.Rectangle, title string) {
	g.window(dst, r)
	g.drawPattern(dst, g.assets.Shou, float64(r.Min.X+9), float64(r.Min.Y+9), float64(r.Dx()-18), float64(r.Dy()-18), 0, 0, 1, [3]float32{.52, .35, .25}, .62)
	g.titlePlate(dst, r, title)
}

// cursorPos is the mouse position in UI px; screenshot drivers replace it.
var cursorPos = func() (int, int) {
	x, y := ebiten.CursorPosition()
	if pointer.used {
		x, y = pointer.x, pointer.y
	}
	return int(float64(x-fixedAt.X-frameAt.X)/uiPx) + frameMin.X, int(float64(y-fixedAt.Y-frameAt.Y)/uiPx) + frameMin.Y
}

func hovered(r image.Rectangle) bool {
	x, y := cursorPos()
	return image.Pt(x, y).In(r)
}

// btn registers a clickable button; a nil click draws it disabled.
func (g *Game) btn(dst *ebiten.Image, r image.Rectangle, label string, click func()) {
	fg, size := g.btnFace(dst, r, label, click)
	g.label(dst, label, float64(r.Min.X)+10, float64(r.Min.Y)+(float64(r.Dy())-size*1.45)/2, size, fg)
}

// centeredBtn is btn with its label centred, as the corner panels' buttons are.
func (g *Game) centeredBtn(dst *ebiten.Image, r image.Rectangle, label string, click func()) {
	fg, size := g.btnFace(dst, r, label, click)
	g.label(dst, label, float64(r.Min.X)+(float64(r.Dx())-g.width(label, size))/2, float64(r.Min.Y)+(float64(r.Dy())-size*1.45)/2, size, fg)
}

// btnFace draws a raised button, registers it unless click is nil, and gives its label's colour and size.
func (g *Game) btnFace(dst *ebiten.Image, r image.Rectangle, label string, click func()) (color.NRGBA, float64) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	body, rim, fg := lacquer2, goldDim, ink
	if click == nil {
		body, fg = color.NRGBA{R: 40, G: 34, B: 30, A: 235}, color.NRGBA{R: 138, G: 130, B: 118, A: 255}
	} else {
		g.addButton(r, label, click)
		if hovered(r) {
			body, rim = hoverC, gold
		}
	}
	rect(dst, x, y, w, h, body)
	strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, rim)
	rect(dst, x+1, y+1, w-2, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 28})
	if h < 28 {
		return fg, 13
	}
	return fg, 15
}

// toggleBtn is a switch: pressed in and gold while on, a raised button with a muted label while off.
func (g *Game) toggleBtn(dst *ebiten.Image, r image.Rectangle, label string, on bool, click func()) {
	if !on {
		_, size := g.btnFace(dst, r, label, click)
		g.label(dst, label, float64(r.Min.X)+(float64(r.Dx())-g.width(label, size))/2, float64(r.Min.Y)+(float64(r.Dy())-size*1.45)/2, size, muted)
		return
	}
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	g.addButton(r, label, click)
	rect(dst, x, y, w, h, color.NRGBA{R: 132, G: 98, B: 34, A: 255})
	rect(dst, x+1, y+1, w-2, 3, color.NRGBA{A: 110})
	strokeRect(dst, x+.5, y+.5, w-1, h-1, 1.5, gold)
	g.label(dst, label, float64(x)+(float64(w)-g.width(label, 15))/2, float64(y)+(float64(h)-15*1.45)/2+1, 15, color.NRGBA{R: 255, G: 238, B: 170, A: 255})
}

// choiceBtn is a button of a group of which one is chosen: the chosen one is pressed in, gold
// and does nothing, the others are raised buttons. Labels are centred.
func (g *Game) choiceBtn(dst *ebiten.Image, r image.Rectangle, label string, on bool, click func()) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	body, rim, fg := lacquer2, goldDim, ink
	switch {
	case on:
		body, rim, fg = color.NRGBA{R: 26, G: 19, B: 15, A: 255}, gold, gold
	case click != nil:
		g.addButton(r, label, click)
		if hovered(r) {
			body, rim = hoverC, gold
		}
	}
	rect(dst, x, y, w, h, body)
	strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, rim)
	if on {
		rect(dst, x+1, y+1, w-2, 3, color.NRGBA{A: 150})
	} else {
		rect(dst, x+1, y+1, w-2, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 28})
	}
	size := 15.0
	ty := float64(y) + (float64(h)-size*1.45)/2
	if on {
		ty++
	}
	g.label(dst, label, float64(x)+(float64(w)-g.width(label, size))/2, ty, size, fg)
}

// arrow is a scroll button: raised with a white mark while it can scroll, flat and dark
// (and not clickable) when it cannot.
func (g *Game) arrow(dst *ebiten.Image, r image.Rectangle, up bool, click func()) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	fg := white
	if click == nil {
		rect(dst, x, y, w, h, color.NRGBA{R: 30, G: 24, B: 20, A: 235})
		strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, color.NRGBA{R: 58, G: 48, B: 40, A: 255})
		fg = color.NRGBA{R: 74, G: 66, B: 58, A: 255}
	} else {
		g.addButton(r, "", click)
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
	frame := 1
	if up {
		frame = 0
	}
	g.drawUI(dst, "scroll_mark", frame, float64(x+w/2), float64(y+h/2), 1, fg)
}

// button keeps the older x, y, width form.
func (g *Game) button(dst *ebiten.Image, label string, x, y, w int, f func()) {
	g.btn(dst, image.Rect(x, y, x+w, y+32), label, f)
}

// block keeps map clicks from passing through a HUD panel.
func (g *Game) block(r image.Rectangle) { g.blocks = append(g.blocks, unframed(r)) }

func bar(dst *ebiten.Image, x, y, w, h float32, k float64, c color.Color) {
	rect(dst, x, y, w, h, trough)
	k = max(0, min(1, k))
	rect(dst, x+1, y+1, (w-2)*float32(k), h-2, c)
}

// mpColor is the spell point colour: mpC up to the max, and past it (a drink poured over the brim)
// a blue that brightens and dims smoothly once every two seconds.
func mpColor(mp, maxMP int, seconds float64) color.NRGBA {
	if mp <= maxMP {
		return mpC
	}
	return lighten(mpC, -.35+.95*overMPPulse(seconds))
}

// overMPPulse runs 0 → 1 → 0 once every two seconds: sin turning 360° in 2 s.
func overMPPulse(seconds float64) float64 { return (1 + math.Sin(seconds*math.Pi)) / 2 }

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
	g.window(dst, r)
	g.label(dst, s, float64(x+14), float64(y+9), 13, ink)
}
