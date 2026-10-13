package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
)

type battleBubble struct{ actor, name, text string }

func (g *Game) showBattleSpeech(lines []content.Speech) {
	g.player.Speech = nil
	var cx, cy float64
	count := 0
	for _, line := range lines {
		g.player.Speech = append(g.player.Speech, &battleBubble{actor: line.Actor, name: g.name(line.Actor), text: line.Text})
		if u := g.unitAt(line.Actor); u != nil {
			x, y := g.field.Center(u.u, u.v)
			cx += x
			cy += y
			count++
		}
	}
	if count > 0 {
		g.camX, g.camY = cx/float64(count), cy/float64(count)
	}
}

func (g *Game) drawBattleSpeech(dst *ebiten.Image) {
	if len(g.player.Speech) == 0 {
		return
	}
	placed := []image.Rectangle{}
	for _, bubble := range g.player.Speech {
		u := g.unitAt(bubble.actor)
		if u == nil {
			continue
		}
		x, y := g.field.Center(u.u, u.v)
		x, y = g.toScreen(x, y-u.lift-u.top)
		bw, bh := g.assets.Bubble.Size(bubble.name, bubble.text)
		w, h := int(bw), int(bh)
		px, py := max(16, min(Width-w-16, int(x)-32)), max(62, min(Height-h-190, int(y)-h-10))
		r := image.Rect(px, py, px+w, py+h)
		for _, other := range placed {
			if r.Overlaps(other) {
				py = max(62, other.Min.Y-h-12)
				r = image.Rect(px, py, px+w, py+h)
				if r.Overlaps(other) {
					py = other.Max.Y + 12
					r = image.Rect(px, py, px+w, py+h)
				}
			}
		}

		placed = append(placed, r)
		g.assets.Bubble.DrawText(dst, bubble.name, bubble.text, float64(px), float64(py), scale)
	}
}
