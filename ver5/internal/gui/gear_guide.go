package gui

import (
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// The gear guide leads the eye in green, step by step, to putting a piece on — 관리, the officer's row, the 장비 tab,
// the slot's 바꾸기, the piece in the picker and the button that puts it on. It has two reasons to: a piece that
// belongs to an officer (its Owner), handed over by an event of the walk, is not put on by itself; and a seal used
// off the field takes off a weapon the new class cannot carry (a sword on a 장창병), leaving the slot to fill with
// any piece that fits. It lasts while the party stays where it began (the walk's area, or the battle's sortie
// preparation) and until every slot it leads to is filled; the 길 안내 setting hides it.

// gearTarget is a slot the guide leads to: the officer's slot to fill with piece, or with any piece that fits when
// piece is "".
type gearTarget struct {
	piece, owner, slot string
}

// gearGuide is what the guide still has to lead to, and where it began (guidePlace).
type gearGuide struct {
	place   string
	targets []gearTarget
}

// guidePlace is where the guide may show: the walk's area, a battle's sortie preparation, or "" anywhere else.
func (g *Game) guidePlace(o core.Observation) string {
	switch {
	case g.S.Engine == nil:
		return ""
	case g.S.Engine.ExploreFile() != nil:
		return "walk:" + o.Spot.Area
	case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
		return "sortie:" + o.Node
	}
	return ""
}

// noteGearGrants starts (or extends) the guide for the owned pieces the walk's events handed over and the weapons a
// promotion took off.
func (g *Game) noteGearGrants(events []core.Event) {
	place := g.guidePlace(g.S.View())
	if place == "" {
		return
	}
	for _, e := range events {
		var t gearTarget
		switch eq := g.S.Data.Equipment[e.Target]; {
		case e.Kind == "grant" && eq.Owner != "":
			t = gearTarget{e.Target, eq.Owner, eq.Slot}
		case e.Kind == "unequip":
			t = gearTarget{"", e.Actor, eq.Slot}
		default:
			continue
		}
		if g.gear.place != place {
			g.gear = gearGuide{place: place}
		}
		if !slices.Contains(g.gear.targets, t) {
			g.gear.targets = append(g.gear.targets, t)
		}
	}
}

// gearStep is the slot the guide leads to now: its piece ("" for any that fits), owner and slot; ok is false when
// nothing is to be shown. The guide ends for good once the party has left its place or every slot is filled.
func (g *Game) gearStep() (piece, owner, slot string, ok bool) {
	if g.gear.place == "" || g.S.Engine == nil {
		return "", "", "", false
	}
	o := g.S.View()
	if g.guidePlace(o) != g.gear.place {
		g.gear = gearGuide{}
		return "", "", "", false
	}
	for _, t := range g.gear.targets {
		c := g.officer(o, t.owner)
		switch {
		case c == nil || c.Dead:
		case t.piece != "" && (c.Equipment[t.slot] == t.piece || o.Warehouse[t.piece] < 1):
		case t.piece == "" && (c.Equipment[t.slot] != "" || len(g.pickable(o, c, t.slot)) == 0):
		default:
			return t.piece, t.owner, t.slot, !g.S.Settings.NoGuide
		}
	}
	g.gear = gearGuide{}
	return "", "", "", false
}

// guideGreen is the guide's rim, the colours of the walk's green plates (plateMain) at rest and lit.
var guideGreen = [2]color.NRGBA{{R: 100, G: 150, B: 100, A: 255}, {R: 190, G: 236, B: 190, A: 255}}

// guideOrange is the promotion guide's rim (promote_guide.go) at rest and lit, and its glow.
var guideOrange = [3]color.NRGBA{{R: 214, G: 92, B: 20, A: 255}, {R: 255, G: 150, B: 50, A: 255}, {R: 255, G: 120, B: 30}}

// guideRim draws the guide's green rim around r, breathing between the plate's rim and its lit rim.
func (g *Game) guideRim(dst *ebiten.Image, r image.Rectangle) {
	g.breathingRim(dst, r, guideGreen[0], guideGreen[1], color.NRGBA{R: 120, G: 220, B: 120})
}

// promoteRim draws the promotion guide's orange rim around r.
func (g *Game) promoteRim(dst *ebiten.Image, r image.Rectangle) {
	g.breathingRim(dst, r, guideOrange[0], guideOrange[1], guideOrange[2])
}

func (g *Game) breathingRim(dst *ebiten.Image, r image.Rectangle, a, b, glow color.NRGBA) {
	k := .5 + .5*math.Sin(g.uiSeconds*2*math.Pi/1.4)
	mix := func(p, q uint8) uint8 { return uint8(float64(p) + (float64(q)-float64(p))*k) }
	c := color.NRGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: 255}
	r = r.Inset(-3)
	strokeRect(dst, float32(r.Min.X)+1, float32(r.Min.Y)+1, float32(r.Dx()-2), float32(r.Dy()-2), 2.5, c)
	glow.A = uint8(40 + 60*k)
	strokeRect(dst, float32(r.Min.X)-1.5, float32(r.Min.Y)-1.5, float32(r.Dx()+3), float32(r.Dy()+3), 2, glow)
}

// tabRect is the rect of tab i of a tabRow drawn from x, y.
func (g *Game) tabRect(x, y int, labels []string, i int) image.Rectangle {
	for j, l := range labels {
		w := max(72, int(g.width(l, 15))+28)
		if j == i {
			return image.Rect(x, y, x+w, y+touchH)
		}
		x += w
	}
	return image.Rectangle{}
}

// actionRect is the rect of the action labelled label in a closeRow ending at right, y with acts.
func (g *Game) actionRect(right, y int, label string, acts ...action) image.Rectangle {
	return g.rowActionRect(right-96-closeGap, y, label, acts...)
}

// rowActionRect is the rect of the action labelled label in an actionRow ending at right, y with acts.
func (g *Game) rowActionRect(right, y int, label string, acts ...action) image.Rectangle {
	x := right
	for i := len(acts) - 1; i >= 0; i-- {
		w := max(96, int(g.width(acts[i].label, 15))+36)
		if acts[i].label == label {
			return image.Rect(x-w, y, x, y+touchH)
		}
		x -= w + 8
	}
	return image.Rectangle{}
}
