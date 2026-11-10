package gui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/duelview"
)

// battleBubble is a line a unit says on the battlefield, coming out letter by letter.
type battleBubble struct {
	actor, name, text string
	typing            typing
	fading            float64 // when it began to fade out (textSeconds); 0 while it is shown
}

// speechFade is how long a battlefield line takes to fade out once it is done with.
const speechFade = 1.0

// dropSpeech lets the shown lines fade out instead of vanishing.
func (g *Game) dropSpeech() {
	if g == nil {
		return
	}
	for _, b := range g.player.Speech {
		b.fading = g.textSeconds
		g.player.Fading = append(g.player.Fading, b)
	}
	g.player.Speech = nil
}

// rushSpeech answers a click on the battlefield lines: lines still coming out come out at once;
// when they are all out, the click moves on to the next line.
func (g *Game) rushSpeech() {
	typing := false
	for _, b := range g.player.Speech {
		if n := len([]rune(b.text)); g.letters(b.typing, n, g.textSeconds) < n {
			typing = true
		}
	}
	if !typing && len(g.player.Speech) > 0 {
		g.player.nextLine = true
	}
	for _, b := range g.player.Speech {
		if n := len([]rune(b.text)); g.letters(b.typing, n, g.textSeconds) < n && math.IsInf(b.typing.rush, 1) {
			b.typing.rush = g.textSeconds
		}
	}
}

func (g *Game) showBattleSpeech(lines []content.Speech) {
	g.dropSpeech()
	var cells []content.Point
	for _, line := range lines {
		g.player.Speech = append(g.player.Speech, &battleBubble{actor: line.Actor, name: g.name(line.Actor), text: line.Text, typing: startTyping(g.textSeconds)})
		if u := g.unitAt(line.Actor); u != nil {
			cells = append(cells, content.Point{X: int(math.Round(u.u)), Y: int(math.Round(u.v))})
		}
	}
	if len(cells) > 0 && g.field != nil && !g.onScreen(cells) {
		g.panTo(g.actionCenter(cells)) // the speakers glide into view
	}
}

func (g *Game) drawBattleSpeech(dst *ebiten.Image) {
	kept := g.player.Fading[:0]
	for _, b := range g.player.Fading {
		if g.textSeconds-b.fading < speechFade {
			kept = append(kept, b)
		}
	}
	g.player.Fading = kept
	var placed []image.Rectangle
	for _, bubble := range g.player.Fading {
		g.drawBubble(dst, bubble, &placed, 1-(g.textSeconds-bubble.fading)/speechFade)
	}
	placed = nil
	for _, bubble := range g.player.Speech {
		g.drawBubble(dst, bubble, &placed, 1)
	}
}

// drawBubble draws one battlefield line over its speaker, clear of the bubbles already placed.
func (g *Game) drawBubble(dst *ebiten.Image, bubble *battleBubble, placed *[]image.Rectangle, alpha float64) {
	u := g.unitAt(bubble.actor)
	if u == nil {
		return
	}
	x, y := g.field.Center(u.u, u.v)
	x, y = worldToUI(g.toScreen(x, y-u.lift-u.top))
	bw, bh := g.assets.Bubble.Size(bubble.name, bubble.text)
	k := 2.0 / 3 / smallScreen()
	w, h := int(bw*k), int(bh*k)
	px, py := max(16, min(uiW-w-16, int(x)-32)), max(62, min(uiH-h-190, int(y)-h-10))
	r := image.Rect(px, py, px+w, py+h)
	for _, other := range *placed {
		if r.Overlaps(other) {
			py = max(62, other.Min.Y-h-12)
			r = image.Rect(px, py, px+w, py+h)
			if r.Overlaps(other) {
				py = other.Max.Y + 12
				r = image.Rect(px, py, px+w, py+h)
			}
		}
	}
	*placed = append(*placed, r)
	shown := -1 // the whole line, once it is out
	if n, out := len([]rune(bubble.text)), g.letters(bubble.typing, len([]rune(bubble.text)), g.textSeconds); out < n && bubble.fading == 0 {
		shown = out
	}
	g.assets.Bubble.DrawTyped(dst, bubble.name, bubble.text, shown, duelview.Tail{}, float64(px)*drawPx, float64(py)*drawPx, drawPx*k, alpha)
}
