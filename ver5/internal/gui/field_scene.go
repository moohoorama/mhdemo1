package gui

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/scenemap"
)

// A scene map of format 2 (MapArt.SceneMap) is drawn from its plan (internal/scenemap), the list the scene viewer
// draws too: the flat layer is baked into the ground once, the standing pieces are sorted with the people every
// frame by scenemap.Order.

// scenePiece is a standing piece of the plan in canvas pixels.
type scenePiece struct {
	scenemap.Piece
	shadow int           // width of the ground shadow under it, 0 for none
	image  *ebiten.Image // a piece of several pictures (a raised block) drawn once at its Rect, so a frame draws it in one go
}

// makeScenePlan bakes the flat layer into the ground pictures (one per water frame when there is water) and keeps
// the standing pieces.
func (f *Field) makeScenePlan() error {
	plan, err := scenemap.Build(f.Map.SceneMap, f.A.SceneRules, scenemap.Options{Decor: true})
	if err != nil {
		return err
	}
	ox, oy := f.OriginX*K, f.OriginY*K
	water := f.A.Animations["water"].Frames
	animated := false
	for _, d := range plan.Flat {
		animated = animated || d.Name == "water"
	}
	for i := range f.ground {
		if i > 0 && !animated {
			f.ground[i] = f.ground[0]
			continue
		}
		g := ebiten.NewImage(f.worldSize.X, f.worldSize.Y)
		for _, d := range plan.Flat {
			if d.Name == "water" {
				d.Tile = water[i%len(water)]
			}
			f.drawPlan(g, d, ox, oy, nil)
		}
		f.ground[i] = g
	}
	f.clip.DrawImage(f.ground[0], nil)
	for _, p := range plan.Solid {
		p.Rect = p.Rect.Add(image.Pt(int(ox), int(oy)))
		p.Anchor = [2]float64{p.Anchor[0] + ox, p.Anchor[1] + oy}
		sp := scenePiece{Piece: p}
		if len(p.Draws) > 1 {
			sp.image = ebiten.NewImage(p.Rect.Dx(), p.Rect.Dy())
			for _, d := range p.Draws {
				f.drawPlan(sp.image, d, ox-float64(p.Rect.Min.X), oy-float64(p.Rect.Min.Y), nil)
			}
		}
		if pr, ok := f.A.Props[p.Name]; ok && p.Rank == scenemap.RankObject && !pr.NoShadow {
			sp.shadow = pr.Image.Bounds().Dx()
		}
		f.scenePieces = append(f.scenePieces, sp)
	}
	return nil
}

// drawPlan draws one picture of a plan at canvas offset (ox, oy) of dst; op carries the caller's colour.
func (f *Field) drawPlan(dst *ebiten.Image, d scenemap.Draw, ox, oy float64, op *ebiten.DrawImageOptions) {
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	var img *ebiten.Image
	switch d.Kind {
	case scenemap.DrawTile:
		img = f.A.Tiles[d.Tile].Image
	case scenemap.DrawSprite:
		switch d.Sheet {
		case "scenery":
			img = f.A.Scenery.sprites[d.Name].Image
		case "structures":
			img = f.A.Structures[d.Name].Image
		default:
			img = f.A.Props[d.Name].Image
		}
	case scenemap.DrawFill:
		img = f.A.diamond(d.Color)
		op.GeoM.Translate(-32, 0)
	case scenemap.DrawLines:
		z := float32(1)
		if f.renderZoom > 0 && dst == f.Canvas {
			z = float32(f.renderZoom)
		}
		for _, l := range d.Lines {
			x0, y0, x1, y1 := ox+float64(l[0]), oy+float64(l[1]), ox+float64(l[2]), oy+float64(l[3])
			if dst == f.Canvas {
				x0, y0 = f.renderPoint(x0, y0)
				x1, y1 = f.renderPoint(x1, y1)
			}
			vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), z, hexColor(d.Color), false)
		}
		return
	}
	op.GeoM.Translate(ox+float64(d.X), oy+float64(d.Y))
	f.renderOp(dst, op)
	dst.DrawImage(img, op)
}

// diamond is a game cell's diamond (64 x 32, scenemap.DiamondRow) filled with a colour.
func (a *Assets) diamond(hex string) *ebiten.Image {
	if a.diamonds == nil {
		a.diamonds = map[string]*ebiten.Image{}
	}
	if img, ok := a.diamonds[hex]; ok {
		return img
	}
	img := ebiten.NewImage(64, 32)
	c := hexColor(hex)
	for k := range 32 {
		x0, x1 := scenemap.DiamondRow(32, k)
		vector.FillRect(img, float32(x0), float32(k), float32(x1-x0), 1, c, false)
	}
	a.diamonds[hex] = img
	return img
}

func hexColor(hex string) color.RGBA {
	var c color.RGBA
	c.A = 255
	for i, p := range []*uint8{&c.R, &c.G, &c.B} {
		v := 0
		for _, ch := range hex[1+2*i : 3+2*i] {
			v *= 16
			switch {
			case ch >= '0' && ch <= '9':
				v += int(ch - '0')
			case ch >= 'a' && ch <= 'f':
				v += int(ch-'a') + 10
			case ch >= 'A' && ch <= 'F':
				v += int(ch-'A') + 10
			}
		}
		*p = uint8(v)
	}
	return c
}

// sceneItems are the standing pieces in view, with their ground shadows laid.
func (f *Field) sceneItems() []drawItem {
	var items []drawItem
	for i := range f.scenePieces {
		p := &f.scenePieces[i]
		if !f.DrawBounds.Empty() && !p.Rect.Overlaps(f.DrawBounds) {
			continue
		}
		if p.shadow > 0 {
			f.blit(f.shadows, propShadow(p.shadow), p.Anchor[0], p.Anchor[1], &ebiten.DrawImageOptions{Blend: maxBlend})
		}
		ox, oy := f.OriginX*K, f.OriginY*K
		items = append(items, drawItem{y: p.Anchor[1], rect: p.Rect, key: &p.Key, draw: func() {
			if p.image != nil {
				op := &ebiten.DrawImageOptions{}
				f.shadeScenery(f.Canvas, op, p.Anchor[0], p.Anchor[1])
				op.GeoM.Translate(float64(p.Rect.Min.X), float64(p.Rect.Min.Y))
				f.renderOp(f.Canvas, op)
				f.Canvas.DrawImage(p.image, op)
				return
			}
			for _, d := range p.Draws {
				op := &ebiten.DrawImageOptions{}
				f.shadeScenery(f.Canvas, op, p.Anchor[0], p.Anchor[1])
				f.drawPlan(f.Canvas, d, ox, oy, op)
			}
		}})
	}
	return items
}

// groundLift is how high the ground under a point of game cell coordinates stands on a format 2 map.
func (f *Field) groundLift(u, v float64) float64 {
	m := f.Map.SceneMap
	if m == nil {
		return 0
	}
	step := m.Step(int(math.Floor(2*u+1)), int(math.Floor(2*v+1)))
	if step == 0 {
		return 0
	}
	return float64(f.A.SceneRules.Rules.Heights[strconv.Itoa(step)].Lift)
}

// sceneKey is the standing-layer key of an item drawn without one (a unit, a flame): the small tile round the
// ground point under (x, y), its canvas anchor.
func (f *Field) sceneKey(x, y, lift float64, rank int, rect image.Rectangle) scenemap.Key {
	a, b := (x-f.OriginX*K)/16, (y-f.OriginY*K)/8 // cu - cv and cu + cv
	cu, cv := (a+b)/2, (b-a)/2
	return scenemap.Key{Box: [4]float64{cu - .5, cv - .5, cu + .5, cv + .5}, Lift: int(lift), Rank: rank, X: x, Rect: rect}
}

// orderScene puts the standing layer of a format 2 map in drawing order.
func (f *Field) orderScene(items []drawItem) []drawItem {
	keys := make([]scenemap.Key, len(items))
	for i, it := range items {
		if it.key != nil {
			keys[i] = *it.key
			continue
		}
		rank := scenemap.RankObject
		if it.kind == 1 {
			rank = scenemap.RankPerson
		}
		keys[i] = f.sceneKey(float64(it.rect.Min.X+it.rect.Max.X)/2, it.y, 0, rank, it.rect)
	}
	out := make([]drawItem, 0, len(items))
	for _, i := range scenemap.Order(keys) {
		out = append(out, items[i])
	}
	return out
}
