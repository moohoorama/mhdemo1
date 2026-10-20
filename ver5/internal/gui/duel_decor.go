package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/duelview"
)

// duelDecor gathers the duel arena's scenery from the game's own image files: the tileset's
// rocks and grass, the props' banners, the projectile sheet's dust and the fx sheet's clash
// and ground flecks.
func (g *Game) duelDecor() *duelview.Decor {
	a := g.assets
	tile := func(id int) duelview.Sprite {
		s := a.Tiles[id]
		return duelview.Sprite{Image: s.Image, PivotX: s.PivotX, PivotY: s.PivotY}
	}
	d := &duelview.Decor{}
	for _, n := range []string{"rock_small", "rock_medium", "rock_medium2", "rock_large"} {
		if t, ok := a.Animations[n]; ok {
			d.Rocks = append(d.Rocks, tile(t.Frames[0]))
		}
	}
	for _, n := range []string{"grass_0", "grass_1", "grass_2", "grass_3"} {
		t, ok := a.Animations[n]
		if !ok {
			continue
		}
		var frames []duelview.Sprite
		for _, id := range t.Frames {
			frames = append(frames, tile(id))
		}
		d.Grass = append(d.Grass, frames)
	}
	for name, dst := range map[string]*duelview.Sprite{"banner": &d.Banner, "command_flag": &d.Flag} {
		p := a.Props[name]
		*dst = duelview.Sprite{Image: p.Image, PivotX: p.PivotX, PivotY: p.PivotY}
	}
	shot := func(name string) []duelview.Sprite {
		var out []duelview.Sprite
		s := a.FX[name]
		for _, f := range s.Frames {
			out = append(out, duelview.Sprite{Image: f, PivotX: s.Anchor[0], PivotY: s.Anchor[1]})
		}
		return out
	}
	d.Clash, d.ClashCrit = shot("clash"), shot("clash_crit")
	if s := shot("speck"); len(s) > 0 {
		d.Speck = s[0]
	}
	dust := a.Projectiles["dust"]
	for i, f := range dust.Frames {
		d.Dust = append(d.Dust, duelview.Sprite{Image: f, PivotX: dust.Anchor[0], PivotY: dust.Anchor[1]})
		d.DustMS = append(d.DustMS, float64(dust.MS[i]))
	}
	return d
}

// A card popup under the duel names the two officers: ally on the left, enemy on the right.
// It sits below the line where the speech bubbles end, so it never covers them.
const (
	cardW, cardH = 300, 112
	cardGap      = 12
	cardPad      = 20
)

// smallScreen is the share of the design height a short (phone) screen has: the cards
// shrink by it and the speech bubbles grow by its inverse.
func smallScreen() float64 { return min(1, float64(uiH)/Height) }

func (g *Game) drawDuelCards(dst *ebiten.Image, v *duelview.View) {
	s := smallScreen()
	cw, ch, gap, pad := int(cardW*s), int(cardH*s), int(cardGap*s), int(cardPad*s)
	w, h := 2*cw+gap+2*pad, ch+2*pad
	x, y := uiW/2-w/2, uiH-h-int(48*s)
	box := image.Rect(x-24, y-24, x+w+24, y+h+24)
	g.popup(dst, "duel:cards", 0, uiW, uiH, box, func(dst *ebiten.Image) {
		g.window(dst, image.Rect(x, y, x+w, y+h))
		for side := range 2 {
			g.duelCard(dst, v.Fighters[side], side, x+pad+side*(cw+gap), y+pad, cw, ch, s)
		}
	})
}

func (g *Game) duelCard(dst *ebiten.Image, f duelview.Fighter, side, x, y, cw, ch int, s float64) {
	c := good
	if side == 1 {
		c = bad
	}
	rect(dst, float32(x), float32(y), float32(cw), float32(ch), trough)
	rect(dst, float32(x), float32(y), 3, float32(ch), c)
	if f.Portrait != nil {
		blit(dst, f.Portrait, float64(x+8), float64(y+8), float64(ch-16), float64(ch-16), nil)
		strokeRect(dst, float32(x+7), float32(y+7), float32(ch-14), float32(ch-14), 1, goldDim)
	}
	tx := float64(x + ch + 4)
	write(dst, f.Name, g.serif(32*s, 800), tx, float64(y)+float64(ch)/2+11*s, ink)
}
