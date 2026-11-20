package gui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
)

// The dialogue box uses the ruyi pattern in a black frame, tinted with the speaker's faction colour. Its
// left edge stays beside the portrait and its width follows the line: it springs open to the
// right with a slight overshoot and resizes the same way, a black bar marking the edge that
// unrolls.
const (
	boxFont     = 26.0
	portraitW   = 200.0
	boxPad      = 52  // between the box's edge and the text
	dragonScale = 2.0 // layout pixels per pixel of the dragon tile
)

// dragonTeal is the colour scale that gives the grey pattern its painting's own teal.
var dragonTeal = [3]float32{.31, .40, .40}

// dialogueLayout is where the dialogue goes on the UI canvas: the box's top and height, its
// left edge (right of the portrait), its right limit, the portrait's size and the text size.
// A low screen gets a lower box, a smaller portrait and smaller text.
type dialogueLayout struct{ top, h, left, right, portrait, font float64 }

func dialogueBox() dialogueLayout {
	return dialogueBoxFor(uiW, uiH)
}

func dialogueBoxFor(width, height int) dialogueLayout {
	h := max(150, min(190, math.Round(float64(height)*.24)))
	p := min(portraitW, h+10)
	return dialogueLayout{top: float64(height) - 30 - h, h: h, left: 36 + p + 16, right: float64(width) - 30, portrait: p,
		font: boxFont * max(.85, h/190)}
}

func (g *Game) dialogueFit(s string, width, height int) (dialogueLayout, string, bool) {
	b := dialogueBoxFor(width, height)
	f := g.serif(b.font, 700)
	maxText := b.right - b.left - 2*boxPad
	if maxText <= 0 {
		return b, "", false
	}
	wrapped := wrapWidth(s, f, maxText)
	lines := strings.Count(wrapped, "\n") + 1
	need := math.Max(b.h, 34+float64(lines)*b.font*1.45+18)
	if need > float64(height)*.8 {
		return b, wrapped, false
	}
	b.h, b.top = need, float64(height)-30-need
	return b, wrapped, true
}

// dialogueAnim springs the box's width to its target and fades its tint to the speaker's.
type dialogueAnim struct {
	key            string
	clock          float64 // ui time of the last frame; the box runs on its own clock
	w, from, to    float64
	t, dur         float64
	tint, tintFrom [3]float32
	tintTo         [3]float32
	shownAt        float64 // when the current line's speaker appeared
	speakerChanged bool
}

func easeOutBack(t float64) float64 {
	const c1 = 1.5
	return 1 + (c1+1)*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func (a *dialogueAnim) retarget(to float64, tint [3]float32) {
	a.from, a.to, a.t = a.w, to, 0
	a.dur = min(.52, math.Abs(to-a.w)*.00055+.24)
	a.tintFrom, a.tintTo = a.tint, tint
	if a.w < 1 {
		a.tintFrom = tint
	}
}

func (a *dialogueAnim) advance(dt float64) {
	if a.t >= a.dur {
		a.w, a.tint = a.to, a.tintTo
		return
	}
	a.t = min(a.dur, a.t+dt)
	k := a.t / a.dur
	a.w = max(0, a.from+(a.to-a.from)*easeOutBack(k))
	for i := range a.tint {
		a.tint[i] = a.tintFrom[i] + (a.tintTo[i]-a.tintFrom[i])*float32(k)
	}
}

func (a *dialogueAnim) open() bool { return a.t >= a.dur }

// factionTint is the colour scale for a speaker's faction: its middle shade, muted and brought
// to the teal's brightness so the text reads the same on every faction. Narration and speakers
// without a faction get the teal.
func (g *Game) factionTint(speakers []string) [3]float32 {
	if len(speakers) == 0 {
		return dragonTeal
	}
	_, c, _ := g.character(speakers[0])
	f, ok := g.S.Data.Factions[c.Look.Faction]
	if !ok || len(f.Colors) == 0 {
		return dragonTeal
	}
	cols := colorsOf(f.Colors)
	if len(cols) == 0 {
		return dragonTeal
	}
	m := cols[len(cols)/2]
	luma := func(r, g, b float32) float32 { return .299*r + .587*g + .114*b }
	want := luma(dragonTeal[0], dragonTeal[1], dragonTeal[2])
	rgb := [3]float32{float32(m.R) / 255, float32(m.G) / 255, float32(m.B) / 255}
	have := max(luma(rgb[0], rgb[1], rgb[2]), .01)
	var t [3]float32
	for i, v := range rgb {
		t[i] = want + (v*want/have-want)*.6
	}
	return t
}

// patternScale is how many UI px one texel of the ruyi and shou patterns covers: they were painted finer than the
// screen needs and are kept at three quarters of that size.
const patternScale = 1254.0 / 940

// drawPattern fills the layout rectangle with a repeated tile in colour scale tint, the
// pattern shifted by (ox, oy). One quad repeats the tile, so linear filtering wraps across its
// seams instead of fading to the transparent outside.
func (g *Game) drawPattern(dst *ebiten.Image, tile *ebiten.Image, x, y, w, h, ox, oy, scale float64, tint [3]float32, alpha float32) {
	if tile == nil || w <= 0 || h <= 0 {
		return
	}
	var vs [4]ebiten.Vertex
	for i := range vs {
		lx, ly := x+w*float64(i%2), y+h*float64(i/2)
		vs[i] = ebiten.Vertex{
			DstX: float32(lx*drawPx + drawOX), DstY: float32(ly*drawPx + drawOY),
			SrcX: float32((lx + ox) / scale), SrcY: float32((ly + oy) / scale),
			ColorR: tint[0], ColorG: tint[1], ColorB: tint[2], ColorA: alpha,
		}
	}
	op := &ebiten.DrawTrianglesOptions{Address: ebiten.AddressRepeat, Filter: ebiten.FilterLinear}
	dst.DrawTriangles(vs[:], []uint16{0, 1, 2, 1, 3, 2}, tile, op)
}

func (g *Game) drawDragon(dst *ebiten.Image, x, y, w, h, ox, oy float64, tint [3]float32) {
	g.drawPattern(dst, g.assets.Dragon, x, y, w, h, ox, oy, dragonScale, tint, 1)
}

// drawBoxBody draws the open box from x, w wide, its top at y and h high.
func (g *Game) drawBoxBody(dst *ebiten.Image, x, y, w, h float64, tint [3]float32) {
	if w < 1 {
		return
	}
	x0, y0, fw, fh := float32(x), float32(y), float32(w), float32(h)
	rect(dst, x0+9, y0+15, fw, fh, color.NRGBA{A: 107})
	g.drawPattern(dst, g.assets.Ruyi, x, y, w, h, 0, 0, patternScale, tint, 1)
	if w > 12 {
		strokeRect(dst, x0+5.5, y0+5.5, fw-11, fh-11, 1, color.NRGBA{R: 170, G: 195, B: 185, A: 71})
	}
	strokeRect(dst, x0+1.5, y0+1.5, fw-3, fh-3, 3, color.Black)
	// the edge that unrolls
	rect(dst, x0+fw+1, y0, 10, fh+12, color.NRGBA{A: 102})
	rect(dst, x0+fw-3, y0-6, 10, fh+12, color.Black)
	rect(dst, x0+fw, y0-2, 2, fh+4, color.NRGBA{R: 170, G: 195, B: 185, A: 89})
}

// storyPortrait is the character's own illustration. The story shows no face for troops or for
// officers who have none, rather than their class's stand-in.
func (g *Game) storyPortrait(c content.Character) *ebiten.Image {
	if c.Template || c.Look.NoPortrait {
		return nil
	}
	return g.assets.portrait(g.laterLook(c.Look).Portrait)
}

// drawDialogue draws one line of dialogue in the box with the speaker's portrait and name
// plate; key identifies the line so the box springs to its width when it changes.
func (g *Game) drawDialogue(dst *ebiten.Image, l line, text string, done bool, key string) {
	a := &g.dialogueAnim
	a.advance(g.uiSeconds - a.clock)
	a.clock = g.uiSeconds
	b, wrapped, _ := g.dialogueFit(l.text, uiW, uiH)
	f := g.serif(b.font, 700)
	if a.key != key {
		widest := 0.0
		for _, s := range strings.Split(wrapped, "\n") {
			widest = max(widest, measure(s, f))
		}
		target := max(120, min(math.Ceil(widest+2*boxPad), b.right-b.left))
		a.speakerChanged = a.key == "" || strings.Join(l.speakers, "·") != strings.SplitN(a.key, "|", 2)[0]
		a.key = key
		a.retarget(target, g.factionTint(l.speakers))
		if a.speakerChanged {
			a.shownAt = g.uiSeconds
		}
	}
	w := math.Round(a.w)
	g.drawBoxBody(dst, b.left, b.top, w, b.h, a.tint)

	// the speaker's portrait fades and rises in beside the box
	in := bezier(.2, .8, .2, 1, phase(g.uiSeconds-a.shownAt, 0, .3))
	if len(l.speakers) > 0 {
		size := b.portrait
		px, py := 36.0, b.top+b.h-size+8*(1-in)
		src := g.assets.portrait(l.portrait)
		if _, c, ok := g.character(l.speakers[0]); ok && l.portrait == "" {
			src = g.storyPortrait(c)
		}
		if src != nil {
			rect(dst, float32(px-3), float32(py-3), float32(size+6), float32(size+6), color.NRGBA{A: uint8(255 * in)})
			blit(dst, src, px, py, size, size, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * in)})
		}
	}
	if !a.open() && a.speakerChanged {
		return
	}
	// text, clipped to the box while it still moves
	clip := image.Rect(int(b.left*drawPx+drawOX), int(b.top*drawPx+drawOY), int((b.left+w)*drawPx+drawOX), int((b.top+b.h)*drawPx+drawOY))
	sub := dst.SubImage(clip).(*ebiten.Image)
	shown := wrapped
	if !done {
		shown = prefixLines(wrapped, len([]rune(text)))
	}
	tx, ty := b.left+boxPad, b.top+34
	write(sub, shown, f, tx, ty+2, color.NRGBA{A: 178})
	write(sub, shown, f, tx, ty, color.NRGBA{R: 236, G: 230, B: 214, A: 255})

	if len(l.speakers) > 0 {
		pk := bezier(.2, .8, .2, 1, phase(g.uiSeconds-a.shownAt-a.dur, 0, .24))
		name := strings.Join(l.speakers, "·")
		nf := g.serif(17, 900)
		sub := ""
		if len(l.speakers) == 1 {
			if z := g.courtesy(l.speakers[0]); z != "" {
				sub = "자 " + z
			}
		}
		sf := g.serif(12, 700)
		pw := measure(name, nf) + 26
		if sub != "" {
			pw += measure(sub, sf) + 7
		}
		x, y := b.left+8, b.top-38+6*(1-pk)
		al := uint8(255 * pk)
		rect(dst, float32(x+2), float32(y+3), float32(pw), 28, color.NRGBA{A: uint8(float64(al) * .5)})
		for i := float32(0); i < 4; i++ {
			k := float64(i) / 3
			rect(dst, float32(x), float32(y)+i*7, float32(pw), 7.5, color.NRGBA{R: uint8(163 - 41*k), G: uint8(58 - 22*k), B: uint8(36 - 16*k), A: al})
		}
		strokeRect(dst, float32(x), float32(y), float32(pw), 28, 2, color.NRGBA{A: al})
		write(dst, name, nf, x+12, y+2, color.NRGBA{R: 251, G: 238, B: 224, A: al})
		if sub != "" {
			write(dst, sub, sf, x+12+measure(name, nf)+7, y+7, color.NRGBA{R: 251, G: 238, B: 224, A: uint8(float64(al) * .8)})
		}
	}
	if done && a.open() {
		bob := 3 * math.Sin(g.uiSeconds*7)
		g.label(dst, "▼", b.left+w-boxPad+8, b.top+b.h-36+bob, 13, color.NRGBA{R: 224, G: 90, B: 62, A: 255})
	}
}

// narrationIn is how long a narration takes to rise in; clicks wait for it so a double click
// cannot skip the line.
const narrationIn = .52

// drawNarration shows the whole line at once; the box and its text fade in and out together.
// cardIn is how long a closing card takes to come out of the black: a held breath, then the words.
const cardIn = 2.2

// riseTime is how long a narration node takes to come in before a click turns it.
func riseTime(n content.Node) float64 {
	if n.Card {
		return cardIn
	}
	return narrationIn
}

// drawCard is a part's closing card: the screen stays black (the scene before has darkened into it) and the words
// come up in the middle, the first line large ("1장 난세도래"), the rest below it ("끝").
func (g *Game) drawCard(dst *ebiten.Image, l line, key string) {
	a := &g.dialogueAnim
	if a.key != key {
		a.key, a.shownAt = key, g.uiSeconds
	}
	g.blackOut(dst, 1)
	alpha := phase(g.uiSeconds-a.shownAt, .7, cardIn)
	if g.narrationLeaving {
		alpha = 1 - phase(g.uiSeconds-g.narrationExitAt, 0, .43)
	}
	lines := strings.Split(l.text, "\n")
	title, rest := g.serif(46, 700), g.serif(34, 700)
	h := title.Size*1.6 + float64(len(lines)-1)*rest.Size*1.6
	y := (float64(uiH) - h) / 2
	for i, s := range lines {
		f := title
		if i > 0 {
			f, y = rest, y+title.Size*.3
		}
		write(dst, s, f, (float64(uiW)-measure(s, f))/2, y, color.NRGBA{R: 238, G: 226, B: 196, A: uint8(255 * alpha)})
		y += f.Size * 1.6
	}
}

func (g *Game) drawNarration(dst *ebiten.Image, l line, key string, below bool) {
	a := &g.dialogueAnim
	if a.key != key {
		a.key, a.shownAt = key, g.uiSeconds
	}
	f := g.serif(28, 700)
	w := min(760, float64(uiW)*.8)
	wrapped := wrapWidth(l.text, f, w-84)
	lines := strings.Count(wrapped, "\n") + 1
	h := max(170, 58+float64(lines)*f.Size*1.45)
	x, y := (float64(uiW)-w)/2, (float64(uiH)-h)/2
	if below {
		y = float64(uiH) - h - 34
	}
	alpha, dy := 1.0, 0.0
	if g.narrationLeaving {
		p := phase(g.uiSeconds-g.narrationExitAt, 0, .43)
		alpha, dy = 1-p, float64(uiH)*.05*p
	} else {
		p := bezier(.18, .77, .18, 1, phase(g.uiSeconds-a.shownAt, 0, narrationIn))
		alpha, dy = p, float64(uiH)*.05*(1-p)
	}
	y += dy
	shadow := uint8(150 * alpha)
	rect(dst, float32(x+12), float32(y+16), float32(w), float32(h), color.NRGBA{A: shadow})
	rect(dst, float32(x), float32(y), float32(w), float32(h), color.NRGBA{R: 20, G: 43, B: 45, A: uint8(204 * alpha)})
	g.drawPattern(dst, g.assets.Dragon, x, y, w, h, 0, 0, dragonScale, dragonTeal, float32(.22*alpha))
	strokeRect(dst, float32(x+1), float32(y+1), float32(w-2), float32(h-2), 3, color.NRGBA{A: uint8(255 * alpha)})
	strokeRect(dst, float32(x+7), float32(y+7), float32(w-14), float32(h-14), 1, color.NRGBA{R: 180, G: 195, B: 178, A: uint8(90 * alpha)})
	for i, line := range strings.Split(wrapped, "\n") {
		write(dst, line, f, x+(w-measure(line, f))/2, y+(h-float64(lines)*f.Size*1.45)/2+float64(i)*f.Size*1.45,
			color.NRGBA{R: 252, G: 247, B: 231, A: uint8(255 * alpha)})
	}
}
