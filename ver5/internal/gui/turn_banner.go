package gui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// turnBanner is the phase change: ink bands sweep across the screen, the turn is written
// above them left to right, then the side to move appears underlined in its colour.
type turnBanner struct {
	round   int
	faction string
	t       float64
}

// Timeline of the banner in seconds (docs/ui-mockup, idea 2).
const (
	turnDim       = .30
	turnStroke    = .42
	turnStroke2At = .08
	turnWriteAt   = .38
	turnWrite     = .52
	turnSubAt     = .90
	turnSub       = .30
	turnLineAt    = 1.02
	turnLine      = .42
	turnHoldUntil = 2.20
	turnFade      = .40
	turnLength    = turnHoldUntil + turnFade
)

func (g *Game) showTurn(round int, faction string) {
	g.turnBanner = &turnBanner{round: round, faction: faction}
}

// advanceTurnBanner runs on real time; fast replay plays it twice as fast.
func (g *Game) advanceTurnBanner(dt float64) {
	b := g.turnBanner
	if b == nil {
		return
	}
	if g.speed > 1 || g.autoplay {
		dt *= 2
	}
	if b.t += dt; b.t >= turnLength {
		g.turnBanner = nil
	}
}

// turnSide is the side to move as the banner names it.
func (g *Game) turnSide(faction string) string {
	if faction == "ally" {
		return g.lordName() + "군의 차례"
	}
	return factionName(faction) + "의 차례"
}

func turnColor(faction string) color.NRGBA {
	switch faction {
	case "enemy":
		return color.NRGBA{R: 232, G: 192, B: 74, A: 255}
	case "support":
		return color.NRGBA{R: 120, G: 180, B: 240, A: 255}
	}
	return color.NRGBA{R: 143, G: 217, B: 156, A: 255}
}

func (g *Game) drawTurnBanner(dst *ebiten.Image) {
	b := g.turnBanner
	if b == nil {
		return
	}
	fade := 1 - phase(b.t, turnHoldUntil, turnFade)
	rect(dst, 0, 0, float32(uiW), float32(uiH), color.NRGBA{A: uint8(255 * .45 * phase(b.t, 0, turnDim) * fade)})
	// two ink bands sweep in from the left: black, then rust laid over its lower edge
	sweep := func(at float64, y, h float32, c color.NRGBA, alpha float64) {
		k := bezier(.6, 0, .2, 1, phase(b.t, at, turnStroke))
		c.A = uint8(float64(c.A) * alpha * fade)
		sweepBand(dst, y, h, k, c)
	}
	sweep(0, 410, 24, color.NRGBA{R: 13, G: 10, B: 8, A: 255}, 1)
	sweep(turnStroke2At, 422, 16, color.NRGBA{R: 58, G: 18, B: 10, A: 255}, .75)

	title := fmt.Sprintf("제%d턴", b.round)
	face := g.brush(204)
	k := bezier(.4, .1, .3, 1, phase(b.t, turnWriteAt, turnWrite))
	img := g.textImage(title, face, color.NRGBA{R: 246, G: 238, B: 219, A: 255}, 6)
	drawReveal(dst, img, float64(uiW)/2-measure(title, face)/2, 282, k, fade)

	side := g.turnSide(b.faction)
	c := turnColor(b.faction)
	s := phase(b.t, turnSubAt, turnSub)
	sf := g.serif(46, 900)
	sw := spacedWidth(side, sf, .35)
	c.A = uint8(255 * s * fade)
	writeSpaced(dst, side, sf, float64(uiW)/2-sw/2, 492+15*(1-s), .35, c)
	lw := 415 * bezier(.2, .8, .2, 1, phase(b.t, turnLineAt, turnLine))
	rect(dst, float32(float64(uiW)/2-lw/2), 582, float32(lw), 6, c)
}

// sweepBand draws the left fraction k of a full-width band, its head feathered.
func sweepBand(dst *ebiten.Image, y, h float32, k float64, c color.NRGBA) {
	const feather = 40
	edge := float32(k) * (float32(uiW) + feather)
	rect(dst, 0, y, max(0, edge-feather), h, c)
	for x := max(0, edge-feather); x < min(float32(uiW), edge); x += 4 {
		a := c
		a.A = uint8(float32(c.A) * (1 - (x-(edge-feather))/feather))
		rect(dst, x, y, 4, h, a)
	}
}
