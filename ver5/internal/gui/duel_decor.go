package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/duelview"
)

// duelDecor gathers the duel arena's scenery from the map's own image files: the tileset's
// rocks and grass, the props' banners and the projectile sheet's dust.
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

func (g *Game) drawDuelCards(dst *ebiten.Image, v *duelview.View) {
	w, h := 2*cardW+cardGap+2*cardPad, cardH+2*cardPad
	x, y := uiW/2-w/2, uiH-h-48
	box := image.Rect(x-24, y-24, x+w+24, y+h+24)
	g.popup(dst, "duel:cards", 0, uiW, uiH, box, func(dst *ebiten.Image) {
		window(dst, image.Rect(x, y, x+w, y+h))
		for side := range 2 {
			g.duelCard(dst, v.Fighters[side], side, x+cardPad+side*(cardW+cardGap), y+cardPad)
		}
	})
}

func (g *Game) duelCard(dst *ebiten.Image, f duelview.Fighter, side, x, y int) {
	tag, c := "아군 장수", good
	if side == 1 {
		tag, c = "적군 장수", bad
	}
	rect(dst, float32(x), float32(y), cardW, cardH, trough)
	rect(dst, float32(x), float32(y), 3, cardH, c)
	if f.Portrait != nil {
		blit(dst, f.Portrait, float64(x+8), float64(y+8), cardH-16, cardH-16, nil)
		strokeRect(dst, float32(x+7), float32(y+7), cardH-14, cardH-14, 1, goldDim)
	}
	tx := float64(x + cardH + 4)
	g.label(dst, tag, tx, float64(y+18), 14, c)
	write(dst, f.Name, g.serif(32, 800), tx, float64(y+46), ink)
}
