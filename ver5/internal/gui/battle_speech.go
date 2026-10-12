package gui

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"srpg/internal/content"
)

type battleBubble struct {
	actor string
	image *ebiten.Image
}

func (g *Game) showBattleSpeech(lines []content.Speech) {
	g.player.Speech = nil
	var cx, cy float64
	count := 0
	for _, line := range lines {
		body := wrap(line.Text, 21)
		name := g.name(line.Actor)
		face := &text.GoTextFace{Source: g.font, Size: 16}
		face.SetVariation(text.MustParseTag("wght"), 400)
		w, _ := text.Measure(body, face, 23)
		h := 53 + 23*(1+strings.Count(body, "\n"))
		im := ebiten.NewImage(int(max(160, w+32)), h)
		g.assets.Bubble.Draw(im, 0, 0, float64(im.Bounds().Dx()), float64(h-20))
		op := &text.DrawOptions{}
		op.GeoM.Translate(16, 10)
		op.ColorScale.ScaleWithColor(color.NRGBA{R: 35, G: 25, B: 20, A: 255})
		face.SetVariation(text.MustParseTag("wght"), 800)
		text.Draw(im, name, face, op)
		face.SetVariation(text.MustParseTag("wght"), 400)
		op.GeoM.Reset()
		op.GeoM.Translate(16, 34)
		op.LineSpacing = 23
		text.Draw(im, body, face, op)
		g.player.Speech = append(g.player.Speech, &battleBubble{actor: line.Actor, image: im})
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
		w, h := bubble.image.Bounds().Dx(), bubble.image.Bounds().Dy()
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
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(px), float64(py))
		drawScaled(dst, bubble.image, op)
	}
}
