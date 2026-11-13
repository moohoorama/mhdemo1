package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/core"
)

// promoteC marks the class of an officer who has reached its promotion level.
var promoteC = color.NRGBA{R: 255, G: 146, B: 48, A: 255}

// canPromote reports whether a party officer of def in class at level may promote now: the level reaches its tier's
// promotion level and the class leads somewhere the officer has a look for.
func (g *Game) canPromote(def, class string, level int) bool {
	d := g.S.Data
	cl, ok := d.Classes[class]
	if !ok || !d.Characters[def].Playable {
		return false
	}
	pl := d.Rules.PromoteLevels
	return cl.Tier < len(pl) && level >= pl[cl.Tier] && len(d.Promotes(def, class)) > 0
}

// classRank draws an officer's class (in orange when it may promote) followed by rest in c, and returns the width.
func (g *Game) classRank(dst *ebiten.Image, def, class string, level int, rest string, x, y, size float64, c color.Color) float64 {
	name, nc := g.className(class), c
	if g.canPromote(def, class, level) {
		nc = promoteC
	}
	g.label(dst, name, x, y, size, nc)
	w := g.width(name, size)
	if rest != "" {
		g.label(dst, rest, x+w, y, size, c)
		w += g.width(rest, size)
	}
	return w
}

// gearIcon is the icon of the first piece (by id) of the slot and type, the look of what the slot is missing.
func (g *Game) gearIcon(slot, typ string) string {
	for _, id := range sorted(g.S.Data.Equipment) {
		if eq := g.S.Data.Equipment[id]; eq.Slot == slot && eq.Type == typ {
			return eq.Icon
		}
	}
	return ""
}

// missingGear draws, from x, a small crossed-out icon of the class's weapon and armor for each of the two slots the
// officer leaves empty, and returns the width it took.
func (g *Game) missingGear(dst *ebiten.Image, class string, equipment map[string]string, x, y float64) float64 {
	const k, side = .75, 24.0
	cl := g.S.Data.Classes[class]
	w := 0.0
	for _, s := range []struct{ slot, typ string }{{"weapon", cl.Weapon}, {"armor", cl.Armor}} {
		if equipment[s.slot] != "" {
			continue
		}
		ix := x + w
		if icon := g.gearIcon(s.slot, s.typ); icon != "" {
			g.drawUI(dst, "item_"+icon, 0, ix, y, k, white)
		}
		cx, cy := float32(ix+side-6), float32(y+side-6) // a red X on a dark disc at the icon's lower right
		fillCircle(dst, cx, cy, 8, color.NRGBA{R: 30, G: 20, B: 16, A: 235})
		strokeLine(dst, cx-4.5, cy-4.5, cx+4.5, cy+4.5, 3, bad)
		strokeLine(dst, cx+4.5, cy-4.5, cx-4.5, cy+4.5, 3, bad)
		w += side + 3
	}
	return w
}

// deputyTag draws, from x on a 13-size text line at y, the deputy's small face and "부관 이름", cut to end by right.
func (g *Game) deputyTag(dst *ebiten.Image, o core.Observation, id string, x, y, right float64) {
	const face = 20.0
	if d := g.officer(o, id); d != nil {
		g.portrait(dst, d.ID, d.Class, x, y-1, face)
	}
	g.label(dst, "부관", x+face+5, y+1, 12, muted)
	nx := x + face + 9 + g.width("부관", 12)
	g.label(dst, ellipsis(g, g.name(id), 13, int(right-nx)), nx, y, 13, ink)
}
