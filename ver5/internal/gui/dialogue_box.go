package gui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
)

// The dialogue box is the dragon pattern (assets/graphics/ui/dragon.png, made by
// tools/ui/dragon_tile.py) in a black frame, tinted with the speaker's faction colour. Its
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
	h := max(150, min(190, math.Round(float64(uiH)*.24)))
	p := min(portraitW, h+10)
	return dialogueLayout{top: float64(uiH) - 30 - h, h: h, left: 36 + p + 16, right: float64(uiW) - 30, portrait: p,
		font: boxFont * max(.85, h/190)}
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

// drawDragon fills the layout rectangle with the dragon pattern in colour scale tint, the
// pattern shifted by (ox, oy). One quad repeats the tile, so linear filtering wraps across its
// seams instead of fading to the transparent outside.
func (g *Game) drawDragon(dst *ebiten.Image, x, y, w, h, ox, oy float64, tint [3]float32) {
	tile := g.assets.Dragon
	if tile == nil || w <= 0 || h <= 0 {
		return
	}
	var vs [4]ebiten.Vertex
	for i := range vs {
		lx, ly := x+w*float64(i%2), y+h*float64(i/2)
		vs[i] = ebiten.Vertex{
			DstX: float32(lx*drawPx + drawOX), DstY: float32(ly*drawPx + drawOY),
			SrcX: float32((lx + ox) / dragonScale), SrcY: float32((ly + oy) / dragonScale),
			ColorR: tint[0], ColorG: tint[1], ColorB: tint[2], ColorA: 1,
		}
	}
	op := &ebiten.DrawTrianglesOptions{Address: ebiten.AddressRepeat, Filter: ebiten.FilterLinear}
	dst.DrawTriangles(vs[:], []uint16{0, 1, 2, 1, 3, 2}, tile, op)
}

// drawBoxBody draws the open box from x, w wide, its top at y and h high.
func (g *Game) drawBoxBody(dst *ebiten.Image, x, y, w, h float64, tint [3]float32) {
	if w < 1 {
		return
	}
	x0, y0, fw, fh := float32(x), float32(y), float32(w), float32(h)
	rect(dst, x0+9, y0+15, fw, fh, color.NRGBA{A: 107})
	g.drawDragon(dst, x, y, w, h, 0, 0, [3]float32{tint[0] * .75, tint[1] * .75, tint[2] * .75})
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
	if c.Template {
		return nil
	}
	return g.assets.Portraits[c.Look.Portrait]
}

// drawDialogue draws one line of dialogue in the box with the speaker's portrait and name
// plate; key identifies the line so the box springs to its width when it changes.
func (g *Game) drawDialogue(dst *ebiten.Image, l line, text string, done bool, key string) {
	a := &g.dialogueAnim
	a.advance(g.uiSeconds - a.clock)
	a.clock = g.uiSeconds
	b := dialogueBox()
	f := g.serif(b.font, 700)
	wrapped := wrapWidth(l.text, f, b.right-b.left-2*boxPad)
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
		src := g.assets.Portraits[l.portrait]
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
