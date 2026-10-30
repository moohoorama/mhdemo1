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
	level  int             // the ground it stands on (drawItem.level)
	shadow Sprite          // its ground shadow, nil Image for none
	image  *ebiten.Image   // a piece of several pictures drawn once at its Rect, so a frame draws it in one go
	anims  []scenemap.Draw // the looping strips it carries (a campfire's flames), played over it
}

// sceneLevel is a raised ground of the plan: its tops baked once like the ground, and the shadows cast on them this
// frame, laid over the tops and cut to them.
type sceneLevel struct {
	lift    float64
	top     *ebiten.Image     // world size, like the ground
	tops    []image.Rectangle // each top's diamond, canvas pixels
	shadows *ebiten.Image     // render size, like the ground's shadows
}

// makeScenePlan bakes the flat layer into the ground pictures (one per water frame when there is water) and each
// raised level's tops into a picture of its own, and keeps the standing pieces.
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
		g := newLayer(f.worldSize.X, f.worldSize.Y)
		for _, d := range plan.Flat {
			if d.Name == "water" {
				d.Tile = water[i%len(water)]
			}
			f.drawPlan(g, d, ox, oy, nil)
		}
		f.ground[i] = g
	}
	f.clip.DrawImage(f.ground[0], nil)
	f.addScenePieces(plan.Solid, 0)
	for i, l := range plan.Levels {
		lv := sceneLevel{lift: float64(l.Lift), top: newLayer(f.worldSize.X, f.worldSize.Y),
			shadows: newLayer(f.shadows.Bounds().Dx(), f.shadows.Bounds().Dy())}
		for _, d := range l.Flat {
			f.drawPlan(lv.top, d, ox, oy, nil)
		}
		for _, r := range l.Tops {
			lv.tops = append(lv.tops, r.Add(image.Pt(int(ox), int(oy))))
		}
		f.levels = append(f.levels, lv)
		f.addScenePieces(l.Solid, i+1)
	}
	return nil
}

// addScenePieces keeps the standing pieces of one ground of the plan.
func (f *Field) addScenePieces(pieces []scenemap.Piece, level int) {
	ox, oy := f.OriginX*K, f.OriginY*K
	for _, p := range pieces {
		p.Rect = p.Rect.Add(image.Pt(int(ox), int(oy)))
		p.Anchor = [2]float64{p.Anchor[0] + ox, p.Anchor[1] + oy}
		sp := scenePiece{Piece: p, level: level}
		for _, d := range p.Draws {
			if d.Kind == scenemap.DrawAnim {
				sp.anims = append(sp.anims, d)
			}
		}
		for _, l := range p.Lights {
			f.night.props = append(f.night.props, scenemap.Light{X: l.X + ox, Y: l.Y + oy, Radius: l.Radius, Glow: l.Glow})
		}
		if len(p.Draws)-len(sp.anims) > 1 {
			sp.image = ebiten.NewImage(p.Rect.Dx(), p.Rect.Dy())
			for _, d := range p.Draws {
				f.drawPlan(sp.image, d, ox-float64(p.Rect.Min.X), oy-float64(p.Rect.Min.Y), nil)
			}
		}
		if pr, ok := f.A.Props[p.Name]; ok && p.Rank == scenemap.RankObject && !pr.NoShadow {
			sp.shadow = pr.Shadow
		}
		f.scenePieces = append(f.scenePieces, sp)
	}
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
	case scenemap.DrawAnim:
		return // played over its piece every frame by drawStrip
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
	if len(d.Clip) == 0 {
		dst.DrawImage(img, op)
		return
	}
	o, cs := img.Bounds().Min, op.ColorScale
	vs := make([]ebiten.Vertex, len(d.Clip))
	is := make([]uint16, 0, 3*(len(d.Clip)-2))
	for i, p := range d.Clip {
		sx, sy := float64(p[0]-d.X), float64(p[1]-d.Y)
		x, y := op.GeoM.Apply(sx, sy)
		vs[i] = ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: float32(o.X) + float32(sx), SrcY: float32(o.Y) + float32(sy), ColorR: cs.R(), ColorG: cs.G(), ColorB: cs.B(), ColorA: cs.A()}
		if i >= 2 {
			is = append(is, 0, uint16(i-1), uint16(i))
		}
	}
	dst.DrawTriangles(vs, is, img, nil)
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
		if p.shadow.Image != nil {
			sh := p.shadow
			if i, ok := f.animFrame(p.Name, p.Anchor); ok && f.A.Anims[p.Name].Shadows != nil {
				sh = f.A.Anims[p.Name].Shadows[i]
			}
			f.castShadow(float64(p.Lift), sh, p.Anchor[0], p.Anchor[1], &ebiten.DrawImageOptions{Blend: maxBlend})
		}
		ox, oy := f.OriginX*K, f.OriginY*K
		items = append(items, drawItem{y: p.Anchor[1], rect: p.Rect, key: &p.Key, level: p.level, draw: func() {
			if p.image != nil {
				op := &ebiten.DrawImageOptions{}
				f.shadeScenery(f.Canvas, op, p.Anchor[0], p.Anchor[1])
				op.GeoM.Translate(float64(p.Rect.Min.X), float64(p.Rect.Min.Y))
				f.renderOp(f.Canvas, op)
				f.Canvas.DrawImage(p.image, op)
			} else {
				for _, d := range p.Draws {
					op := &ebiten.DrawImageOptions{}
					f.shadeScenery(f.Canvas, op, p.Anchor[0], p.Anchor[1])
					if f.drawAnim(d, p.Anchor, ox, oy, op) {
						continue
					}
					f.drawPlan(f.Canvas, d, ox, oy, op)
				}
			}
			for _, d := range p.anims {
				f.drawStrip(d, ox, oy)
			}
		}})
	}
	return items
}

// castShadow lays a shadow cast on ground lifted this high on that ground's shadow layer.
func (f *Field) castShadow(lift float64, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	dst := f.shadows
	if l := f.levelOf(lift); l > 0 {
		dst = f.levels[l-1].shadows
	}
	f.blit(dst, s, x, y, op)
}

// levelOf is the ground (drawItem.level) lifted this high.
func (f *Field) levelOf(lift float64) int {
	for i, l := range f.levels {
		if l.lift == lift {
			return i + 1
		}
	}
	return 0
}

func (f *Field) levelShadows(img *ebiten.Image) bool {
	for _, l := range f.levels {
		if l.shadows == img {
			return true
		}
	}
	return false
}

// drawSceneLevels draws a format 2 map's standing layer ground by ground: what stands on the ground, then for each
// raised level its tops, the shadows on them and what stands on it. It returns the items in the order drawn, each
// level's tops among them as scenery, so what they hide shows through.
func (f *Field) drawSceneLevels(items []drawItem) []drawItem {
	out := make([]drawItem, 0, len(items)+len(f.levels))
	for i := 0; i <= len(f.levels); i++ {
		if i > 0 {
			l := &f.levels[i-1]
			op := &ebiten.DrawImageOptions{}
			f.renderOp(f.Canvas, op)
			f.Canvas.DrawImage(l.top, op)
			clipOp := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn}
			f.renderOp(l.shadows, clipOp)
			l.shadows.DrawImage(l.top, clipOp)
			f.Canvas.DrawImage(l.shadows, nil)
			out = append(out, drawItem{tops: l.tops, level: i})
		}
		var ground []drawItem
		for _, it := range items {
			if it.level == i {
				ground = append(ground, it)
			}
		}
		for _, it := range f.orderScene(ground) {
			f.sceneryLayer = it.kind == 0
			it.draw()
			out = append(out, it)
		}
	}
	return out
}

// drawAnim draws a prop that has a looping strip of its name in Assets.Anims (a tree's leaves, the incense smoke) at
// its current frame. It reports false for anything else.
func (f *Field) drawAnim(d scenemap.Draw, anchor [2]float64, ox, oy float64, op *ebiten.DrawImageOptions) bool {
	if d.Kind != scenemap.DrawSprite || d.Sheet == "scenery" || d.Sheet == "structures" {
		return false
	}
	i, ok := f.animFrame(d.Name, anchor)
	if !ok {
		return false
	}
	s := f.A.Anims[d.Name].Frames[i]
	pr := f.A.Props[d.Name]
	op.GeoM.Translate(ox+float64(d.X)+pr.PivotX-s.PivotX, oy+float64(d.Y)+pr.PivotY-s.PivotY)
	f.renderOp(f.Canvas, op)
	f.Canvas.DrawImage(s.Image, op)
	return true
}

// drawStrip draws a looping strip an object carries (scenemap.DrawAnim, top left at canvas offset (ox, oy)) at its
// current frame; each starts at its own frame and time, so two flames of a fire do not flicker in step.
func (f *Field) drawStrip(d scenemap.Draw, ox, oy float64) {
	i, ok := f.animFrame(d.Name, [2]float64{float64(d.X), float64(d.Y)})
	if !ok {
		return
	}
	s := f.A.Anims[d.Name].Frames[i]
	op := &ebiten.DrawImageOptions{}
	f.shadeScenery(f.Canvas, op, ox+float64(d.X)+s.PivotX, oy+float64(d.Y)+s.PivotY)
	op.GeoM.Translate(ox+float64(d.X), oy+float64(d.Y))
	f.renderOp(f.Canvas, op)
	f.Canvas.DrawImage(s.Image, op)
}

// animFrame is the frame of the looping strip of prop name standing at anchor; each prop starts at its own frame and
// time so a grove does not move in step.
func (f *Field) animFrame(name string, anchor [2]float64) (int, bool) {
	strip, ok := f.A.Anims[name]
	if !ok || strip.MS <= 0 {
		return 0, false
	}
	ax, ay := int(anchor[0]), int(anchor[1])
	lag := hash3(ax, ay, 7) * float64(strip.MS)
	n := len(strip.Frames)
	return (int((f.clockMS+lag)/float64(strip.MS)) + int(hash3(ax, ay, 8)*float64(n))) % n, true
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
