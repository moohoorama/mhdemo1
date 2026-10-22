package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// A trait's icon is its kind's picture (책략 a scroll, 전법 crossed weapons, 특기 a medallion),
// drawn in grey and tinted by its tone: what the trait does.

const traitIconSize = 22

var kindNames = map[string]string{"strategy": "책략", "tactic": "전법", "talent": "특기"}

var toneNames = map[string]string{"attack": "공격", "heal": "회복", "buff": "강화", "debuff": "약화",
	"guard": "방어", "move": "이동", "craft": "책략술"}

var toneColors = map[string]color.NRGBA{
	"attack": {R: 224, G: 80, B: 64, A: 255},
	"heal":   {R: 255, G: 255, B: 255, A: 255},
	"buff":   {R: 108, G: 200, B: 96, A: 255},
	"debuff": {R: 74, G: 58, B: 90, A: 255},
	"guard":  {R: 90, G: 156, B: 232, A: 255},
	"move":   {R: 240, G: 224, B: 96, A: 255},
	"craft":  {R: 192, G: 112, B: 224, A: 255},
}

// debuffHalo rims the near-black debuff icon so it reads on the dark windows.
var debuffHalo = color.NRGBA{R: 200, G: 184, B: 216, A: 255}

// traitLook is a trait's kind and tone.
func (g *Game) traitLook(id string) (kind, tone string) {
	t := g.S.Data.Traits[id]
	return t.Kind, t.Tone
}

// skillLook is the kind and tone of a skill by what it does: a physical attack is a 전법, the
// rest are 책략.
func (g *Game) skillLook(id string) (kind, tone string) {
	switch g.S.Data.Skills[id].Kind {
	case "physical":
		return "tactic", "attack"
	case "magic":
		return "strategy", "attack"
	case "status":
		return "strategy", "debuff"
	case "heal":
		return "strategy", "heal"
	}
	return "strategy", "buff"
}

// traitIcon draws a trait icon s UI px square from its top-left corner.
func (g *Game) traitIcon(dst *ebiten.Image, kind, tone string, x, y, s float64) {
	name := "trait_" + kind
	if _, ok := g.assets.UI[name]; !ok {
		return
	}
	k := s / traitIconSize
	if tone == "debuff" {
		for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			g.drawUI(dst, name, 0, x+d[0], y+d[1], k, debuffHalo)
		}
	}
	g.drawUI(dst, name, 0, x, y, k, toneColors[tone])
}

// traitIconOf draws trait id's icon.
func (g *Game) traitIconOf(dst *ebiten.Image, id string, x, y, s float64) {
	kind, tone := g.traitLook(id)
	g.traitIcon(dst, kind, tone, x, y, s)
}

// traitKindName is "특기 · 방어" and the like.
func (g *Game) traitKindName(id string) string {
	kind, tone := g.traitLook(id)
	return kindNames[kind] + " · " + toneNames[tone]
}
