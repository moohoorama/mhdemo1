package gui

import (
	"image"
	"image/color"
	"math"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The dialogue box is a hand scroll put together from pixel parts like the speech bubble
// (docs/ui-mockup, idea 5; parts drawn by tools/ui/scroll_parts.py): two rollers, brocade
// bands and paper. Its left roller stays beside the portrait and its width follows the
// line: it springs open to the right with a slight overshoot and resizes the same way.
const (
	scrollPx   = 4.0 // layout pixels per art pixel
	scrollRod  = 12  // art pixels
	scrollKnob = 8
	scrollBand = 10
	scrollFont = 34.0
	portraitW  = 260.0
	scrollPad  = 20 // art pixels between the paper's edge and the text
)

// scrollBox is where the dialogue goes on the UI canvas: the paper's top and height, the
// left roller's x (right of the portrait), the paper's right limit, the portrait's size and
// the text size. A low screen gets a lower scroll, a smaller portrait and smaller text.
type scrollBox struct{ top, h, left, right, portrait, font float64 }

func dialogueBox() scrollBox {
	h := max(190, min(250, math.Round(float64(uiH)*.32)))
	p := min(portraitW, h+10)
	return scrollBox{top: float64(uiH) - 35 - h, h: h, left: 36 + p + 16, right: float64(uiW) - 30, portrait: p,
		font: scrollFont * max(.8, h/250)}
}

// scrollParts are the scroll's images, one art pixel per image pixel.
type scrollParts struct {
	rodTop, rod, rodBottom, band, paper *ebiten.Image
}

func loadScrollParts(dir string) (*scrollParts, error) {
	p := &scrollParts{}
	for _, f := range []struct {
		name string
		dst  **ebiten.Image
	}{{"rod-top", &p.rodTop}, {"rod", &p.rod}, {"rod-bottom", &p.rodBottom}, {"band", &p.band}, {"paper", &p.paper}} {
		im, err := loadRGBA(filepath.Join(dir, f.name+".png"))
		if err != nil {
			return nil, err
		}
		*f.dst = ebiten.NewImageFromImage(im)
	}
	return p, nil
}

// scrollAnim springs the scroll's half width (in art pixels) to its target.
type scrollAnim struct {
	key            string
	clock          float64 // ui time of the last frame; the scroll runs on its own clock
	hw, from, to   float64
	t, dur         float64
	closing        bool
	shownAt        float64 // when the current line's speaker appeared
	speakerChanged bool
}

func easeOutBack(t float64) float64 {
	const c1 = 1.5
	return 1 + (c1+1)*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func (a *scrollAnim) retarget(to float64) {
	a.from, a.to, a.t = a.hw, to, 0
	a.dur = min(.52, math.Abs(to-a.hw)*.0022*scrollPx/2+.24)
	a.closing = to < 1
	if a.closing {
		a.dur = .3
	}
}

func (a *scrollAnim) advance(dt float64) {
	if a.t >= a.dur {
		a.hw = a.to
		return
	}
	a.t = min(a.dur, a.t+dt)
	k := a.t / a.dur
	f := easeOutBack(k)
	if a.closing {
		f = k * k * k
	}
	a.hw = max(0, a.from+(a.to-a.from)*f)
}

func (a *scrollAnim) open() bool { return a.t >= a.dur && !a.closing }

// drawScrollBody draws the open scroll with paper from cx-hw to cx+hw (layout x of the
// centre, half width in art pixels), the paper's top at y and h high.
func (g *Game) drawScrollBody(dst *ebiten.Image, cx, y, h, hw float64) {
	p := g.assets.Scroll
	if p == nil {
		return
	}
	hw = math.Round(hw)
	w := hw * 2 * scrollPx
	x0 := cx - hw*scrollPx
	tile := func(img *ebiten.Image, x, y, w, h float64) {
		iw, ih := float64(img.Bounds().Dx())*scrollPx, float64(img.Bounds().Dy())*scrollPx
		for ty := 0.0; ty < h; ty += ih {
			for tx := 0.0; tx < w; tx += iw {
				sw, sh := min(iw, w-tx)/scrollPx, min(ih, h-ty)/scrollPx
				sub := img.SubImage(image.Rect(0, 0, int(math.Ceil(sw)), int(math.Ceil(sh)))).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(scrollPx, scrollPx)
				op.GeoM.Translate(x+tx, y+ty)
				drawScaled(dst, sub, op)
			}
		}
	}
	if w > 0 {
		rect(dst, float32(x0+3*scrollPx), float32(y+5*scrollPx), float32(w), float32(h), color.NRGBA{A: 107})
		tile(p.paper, x0, y, w, h)
		band := scrollBand * scrollPx
		tile(p.band, x0, y, w, band)
		tile(p.band, x0, y+h-band, w, band)
		for i := 0; i < 6 && float64(i) < hw; i++ { // the paper curls toward each roller
			c := color.NRGBA{R: 110, G: 80, B: 35, A: uint8(255 * (.28 - float64(i)*.045))}
			rect(dst, float32(x0+float64(i)*scrollPx), float32(y+band), scrollPx, float32(h-2*band), c)
			rect(dst, float32(x0+w-float64(i+1)*scrollPx), float32(y+band), scrollPx, float32(h-2*band), c)
		}
	}
	spin := int(hw / 3)
	for side, rx := range []float64{x0 - (scrollRod+1)*scrollPx, x0 + w - 3*scrollPx} {
		knob := scrollKnob * scrollPx
		rect(dst, float32(rx+4*scrollPx), float32(y-knob+6*scrollPx), float32((scrollRod+2)*scrollPx), float32(h+2*knob), color.NRGBA{A: 115})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scrollPx, scrollPx)
		op.GeoM.Translate(rx, y-knob)
		drawScaled(dst, p.rodTop, op)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scrollPx, h)
		op.GeoM.Translate(rx, y)
		drawScaled(dst, p.rod, op)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scrollPx, scrollPx)
		op.GeoM.Translate(rx, y+h)
		drawScaled(dst, p.rodBottom, op)
		// wood grain that slides as the roller turns
		dir := 1
		if side == 1 {
			dir = -1
		}
		for gy := 3.0; gy < h/scrollPx-3; gy += 9 {
			gx := ((int(gy/9)*5+dir*spin)%scrollRod + scrollRod) % scrollRod
			rect(dst, float32(rx+float64(2+gx)*scrollPx), float32(y+gy*scrollPx), scrollPx, 4*scrollPx, color.NRGBA{R: 25, G: 12, B: 4, A: 128})
		}
	}
}

// drawDialogue draws one line of dialogue in the scroll with the speaker's portrait and
// name plate; key identifies the line so the scroll springs to its width when it changes.
func (g *Game) drawDialogue(dst *ebiten.Image, l line, text string, done bool, key string) {
	a := &g.scrollAnim
	a.advance(g.uiSeconds - a.clock)
	a.clock = g.uiSeconds
	b := dialogueBox()
	f := g.serif(b.font, 700)
	area := b.right - b.left - 2*(scrollRod+scrollPad)*scrollPx
	wrapped := wrapWidth(l.text, f, area)
	if a.key != key {
		lines := strings.Split(wrapped, "\n")
		widest := 0.0
		for _, s := range lines {
			widest = max(widest, measure(s, f))
		}
		target := math.Ceil((widest/scrollPx + 2*scrollPad) / 2)
		target = max(60, min(target, (b.right-b.left)/scrollPx/2-scrollRod-1))
		a.speakerChanged = a.key == "" || strings.Join(l.speakers, "·") != strings.SplitN(a.key, "|", 2)[0]
		a.key = key
		a.retarget(target)
		if a.speakerChanged {
			a.shownAt = g.uiSeconds
		}
	}
	// whole art pixels, so the left roller stays put while the scroll springs
	hwArt := math.Round(a.hw)
	cx := b.left + (scrollRod+1)*scrollPx + hwArt*scrollPx
	g.drawScrollBody(dst, cx, b.top, b.h, hwArt)

	// the speaker's portrait fades and rises in beside the scroll
	in := bezier(.2, .8, .2, 1, phase(g.uiSeconds-a.shownAt, 0, .3))
	if len(l.speakers) > 0 {
		size := b.portrait
		px, py := 36.0, b.top+b.h-size+8*(1-in)
		if _, c, ok := g.character(l.speakers[0]); ok {
			if src := g.portraitArt(c, ""); src != nil {
				rect(dst, float32(px-5), float32(py-5), float32(size+10), float32(size+10), color.NRGBA{R: 26, G: 18, B: 12, A: uint8(255 * in)})
				blit(dst, src, px, py, size, size, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * in)})
				strokeRect(dst, float32(px-2), float32(py-2), float32(size+4), float32(size+4), 2, color.NRGBA{R: goldDim.R, G: goldDim.G, B: goldDim.B, A: uint8(255 * in)})
			}
		}
	}
	if !a.open() && a.speakerChanged {
		return
	}
	// text, clipped to the paper while the scroll still moves
	hw := hwArt * scrollPx
	clip := image.Rect(int((cx-hw)*drawPx), int(b.top*drawPx), int((cx+hw)*drawPx), int((b.top+b.h)*drawPx))
	sub := dst.SubImage(clip).(*ebiten.Image)
	shown := wrapped
	if !done {
		shown = prefixLines(wrapped, len([]rune(text)))
	}
	tx := cx - hw + scrollPad*scrollPx
	write(sub, shown, f, tx, b.top+scrollBand*scrollPx+12, color.NRGBA{R: 29, G: 20, B: 12, A: 255})

	if len(l.speakers) > 0 {
		pk := bezier(.2, .8, .2, 1, phase(g.uiSeconds-a.shownAt-a.dur, 0, .24))
		name := strings.Join(l.speakers, "·")
		nf := g.serif(20, 900)
		sub := ""
		if len(l.speakers) == 1 {
			if z := g.courtesy(l.speakers[0]); z != "" {
				sub = "자 " + z
			}
		}
		sf := g.serif(14, 700)
		w := measure(name, nf) + 30
		if sub != "" {
			w += measure(sub, sf) + 8
		}
		x, y := cx-hw+8, b.top-scrollKnob*scrollPx-10+6*(1-pk)
		al := uint8(255 * pk)
		rect(dst, float32(x+2), float32(y+3), float32(w), 34, color.NRGBA{R: 59, G: 14, B: 7, A: al})
		for i := float32(0); i < 4; i++ {
			k := float64(i) / 3
			rect(dst, float32(x), float32(y)+i*8.5, float32(w), 9, color.NRGBA{R: uint8(163 - 41*k), G: uint8(58 - 22*k), B: uint8(36 - 16*k), A: al})
		}
		strokeRect(dst, float32(x)+.5, float32(y)+.5, float32(w)-1, 33, 1, color.NRGBA{R: 227, G: 167, B: 127, A: al})
		write(dst, name, nf, x+14, y+2, color.NRGBA{R: 251, G: 238, B: 224, A: al})
		if sub != "" {
			write(dst, sub, sf, x+14+measure(name, nf)+8, y+8, color.NRGBA{R: 251, G: 238, B: 224, A: uint8(float64(al) * .8)})
		}
	}
	if done && a.open() {
		bob := 3 * math.Sin(g.uiSeconds*7)
		g.label(dst, "▼", cx+hw-scrollPad*scrollPx, b.top+b.h-scrollBand*scrollPx-30+bob, 16, color.NRGBA{R: 184, G: 50, B: 31, A: 255})
	}
}
