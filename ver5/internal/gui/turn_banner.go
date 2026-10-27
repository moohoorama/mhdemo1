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
	army    string // the AI army whose turn it is (a faction color); empty for the player's
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

func (g *Game) showTurn(round int, faction, army string) {
	g.turnBanner = &turnBanner{round: round, faction: faction, army: army}
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
func (g *Game) turnSide(faction, army string) string {
	return g.armyName(faction, army) + "의 차례"
}

// armyName is what an army is called: the player's after its lord, an AI army by the stage's Armies
// (유비군, 조조군, 황건군), else by its faction color (위군), else by its side (동맹군, 적군).
func (g *Game) armyName(faction, army string) string {
	if faction == "ally" {
		return g.lordName() + "군"
	}
	if g.S != nil && g.S.Engine != nil {
		if st := g.S.View().Map; st != nil {
			for _, a := range st.Armies {
				if a.Faction == army {
					return a.Name
				}
			}
		}
		if f, ok := g.S.Data.Factions[army]; ok && f.Name != "" {
			return f.Name + "군"
		}
	}
	return factionName(faction)
}

// armyColor is the color the turn is written in: the AI army's own faction color when it has one.
func (g *Game) armyColor(faction, army string) color.NRGBA {
	if g.S != nil {
		if f, ok := g.S.Data.Factions[army]; ok && len(f.Colors) > 0 && faction != "ally" {
			if c := colorsOf(f.Colors[:1]); len(c) == 1 {
				return color.NRGBA{R: c[0].R, G: c[0].G, B: c[0].B, A: 255}
			}
		}
	}
	return turnColor(faction)
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
	// laid out about the screen's middle, smaller on a low screen
	z := min(1, float64(uiH)/Height)
	at := func(y float64) float64 { return float64(uiH)/2 + (y-Height/2)*z }
	// two ink bands sweep in from the left: black, then rust laid over its lower edge
	sweep := func(at float64, y, h float64, c color.NRGBA, alpha float64) {
		k := bezier(.6, 0, .2, 1, phase(b.t, at, turnStroke))
		c.A = uint8(float64(c.A) * alpha * fade)
		sweepBand(dst, float32(y), float32(h), k, c)
	}
	sweep(0, at(410), 24*z, color.NRGBA{R: 13, G: 10, B: 8, A: 255}, 1)
	sweep(turnStroke2At, at(422), 16*z, color.NRGBA{R: 58, G: 18, B: 10, A: 255}, .75)

	title := fmt.Sprintf("제%d턴", b.round)
	face := g.brush(204 * z)
	k := bezier(.4, .1, .3, 1, phase(b.t, turnWriteAt, turnWrite))
	img := g.textImage(title, face, color.NRGBA{R: 246, G: 238, B: 219, A: 255}, 6)
	drawReveal(dst, img, float64(uiW)/2-measure(title, face)/2, at(282), k, fade)

	side := g.turnSide(b.faction, b.army)
	c := g.armyColor(b.faction, b.army)
	s := phase(b.t, turnSubAt, turnSub)
	sf := g.serif(46*z, 900)
	sw := spacedWidth(side, sf, .35)
	c.A = uint8(255 * s * fade)
	writeSpaced(dst, side, sf, float64(uiW)/2-sw/2, at(492+15*(1-s)), .35, c)
	lw := 415 * z * bezier(.2, .8, .2, 1, phase(b.t, turnLineAt, turnLine))
	rect(dst, float32(float64(uiW)/2-lw/2), float32(at(582)), float32(lw), float32(6*z), c)
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
