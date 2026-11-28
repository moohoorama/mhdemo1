package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
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

// makeScenePlan keeps the flat layer, drawn at each frame (drawSceneFlat), bakes each raised level's tops into a
// picture of its own, and keeps the standing pieces.
func (f *Field) makeScenePlan() error {
	plan, err := scenemap.Build(f.Map.SceneMap, f.A.SceneRules, scenemap.Options{Decor: true})
	if err != nil {
		return err
	}
	ox, oy := f.OriginX*K, f.OriginY*K
	water := f.A.Animations["water"].Frames
	for _, d := range f.makeSceneGround(plan.Flat, ox, oy) {
		isWater := d.Name == "water"
		if isWater {
			d.Tile = water[0]
		}
		img, at := f.planImage(d)
		if img == nil {
			continue
		}
		fl := sceneFlat{sheet: sheetFor(img), src: img.Bounds(), rect: f.planRect(d, ox, oy), color: noTint, water: isWater}
		if d.Tint != "" { // a picture in greys takes its colour (scenemap.Tint)
			var cs ebiten.ColorScale
			cs.ScaleWithColor(hexColor(d.Tint))
			fl.color = [4]float32{cs.R(), cs.G(), cs.B(), cs.A()}
		}
		for _, p := range d.Clip {
			fl.clip = append(fl.clip, image.Pt(p[0]-d.X-at.X, p[1]-d.Y-at.Y))
		}
		f.sceneFlat = append(f.sceneFlat, fl)
	}
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

// makeSceneGround puts the four 16 x 8 quarters of each small tile of the ground (the plan's flat layer opens with
// them) together: the few different small tiles of a map (a few dozen) are drawn once on a sheet of their own, one
// picture per water frame for one holding water, and each small tile is kept as one flat picture. A quarter of the
// pictures to draw at each frame, all from one sheet; a game cell's (64 x 32) would draw a quarter again but take
// several times the sheet where water runs. It returns the rest of the flat layer.
func (f *Field) makeSceneGround(flat []scenemap.Draw, ox, oy float64) []scenemap.Draw {
	n := 0
	for n < len(flat) && f.isQuarter(flat[n]) {
		n++
	}
	if n == 0 {
		return flat
	}
	type piece struct {
		at   image.Point // from the small tile's top vertex
		tile int         // -1: water
	}
	type spot struct {
		origin image.Point
		pieces []piece
	}
	var order []image.Point
	spots := map[image.Point]*spot{}
	for _, d := range flat[:n] {
		s := f.A.Tiles[d.Tile]
		a, b := (d.X+int(s.PivotX))/8, (d.Y+int(s.PivotY))/4 // the quarter's top vertex: a = 2(u-v)+(i-j), b = 2(u+v)+(i+j)
		k := image.Pt(floorDiv(a+b, 4), floorDiv(b-a, 4))    // the small tile (u, v)
		sp := spots[k]
		if sp == nil {
			sp = &spot{origin: image.Pt(16*(k.X-k.Y), 8*(k.X+k.Y))}
			spots[k] = sp
			order = append(order, k)
		}
		t := d.Tile
		if d.Name == "water" {
			t = -1
		}
		sp.pieces = append(sp.pieces, piece{at: image.Pt(d.X, d.Y).Sub(sp.origin), tile: t})
	}
	type combo struct {
		box    image.Rectangle // from the small tile's top vertex
		frames []image.Rectangle
	}
	water := f.A.Animations["water"].Frames
	combos := map[string]*combo{}
	var keys []string
	keyOf := func(ps []piece) string { return fmt.Sprint(ps) }
	for _, k := range order {
		sp := spots[k]
		key := keyOf(sp.pieces)
		if combos[key] != nil {
			continue
		}
		c := &combo{}
		wet := false
		for _, p := range sp.pieces {
			c.box = c.box.Union(image.Rectangle{Min: p.at, Max: p.at.Add(image.Pt(16, 8))})
			wet = wet || p.tile < 0
		}
		c.frames = make([]image.Rectangle, 1)
		if wet {
			c.frames = make([]image.Rectangle, len(water))
		}
		combos[key] = c
		keys = append(keys, key)
	}
	// shelves on a sheet 1024 wide
	const sheetW = 1024
	x, y, row := 0, 0, 0
	for _, key := range keys {
		c := combos[key]
		for i := range c.frames {
			w, h := c.box.Dx(), c.box.Dy()
			if x+w > sheetW {
				x, y, row = 0, y+row, 0
			}
			c.frames[i] = image.Rect(x, y, x+w, y+h)
			x, row = x+w, max(row, h)
		}
	}
	f.groundSheet = ebiten.NewImage(sheetW, y+row)
	drawn := map[string]bool{}
	for _, k := range order {
		sp := spots[k]
		key := keyOf(sp.pieces)
		c := combos[key]
		if !drawn[key] {
			drawn[key] = true
			for i, r := range c.frames {
				for _, p := range sp.pieces {
					t := p.tile
					if t < 0 {
						t = water[i%len(water)]
					}
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(r.Min.X+p.at.X-c.box.Min.X), float64(r.Min.Y+p.at.Y-c.box.Min.Y))
					f.groundSheet.DrawImage(f.A.Tiles[t].Image, op)
				}
			}
		}
		fl := sceneFlat{sheet: f.groundSheet, src: c.frames[0], rect: c.box.Add(sp.origin).Add(image.Pt(int(ox), int(oy))), color: noTint}
		if len(c.frames) > 1 {
			fl.frames = c.frames
		}
		f.sceneFlat = append(f.sceneFlat, fl)
	}
	return flat[n:]
}

// isQuarter reports whether d is a plain 16 x 8 quarter of a small tile's ground.
func (f *Field) isQuarter(d scenemap.Draw) bool {
	return d.Kind == scenemap.DrawTile && d.Clip == nil && d.Tint == "" && f.A.Tiles[d.Tile].Image.Bounds().Size() == image.Pt(16, 8)
}

// sceneFlat is a picture of a scene map's flat layer: the src part of sheet laid on rect of the canvas, in colour
// scale color, cut to the polygon clip (picture pixels from rect's corner) when it has one. A water tile shows the
// water frame of the moment instead of src.
type sceneFlat struct {
	sheet  *ebiten.Image
	src    image.Rectangle
	rect   image.Rectangle
	color  [4]float32
	clip   []image.Point
	water  bool              // a water tile: the water frame of the moment instead of src
	frames []image.Rectangle // ground put together with water (makeSceneGround): its src at each water frame
}

// drawSceneFlat draws the flat pictures in view over dst, the water at water frame frame, in one call for each run
// of pictures from one sheet. The canvas is clear under them, so the shadows can be laid on what they cover alone.
func (f *Field) drawSceneFlat(dst *ebiten.Image, frame int) {
	tiles := f.A.Animations["water"].Frames
	w := f.A.Tiles[tiles[frame%len(tiles)]].Image
	wSheet, wSrc := sheetFor(w), w.Bounds()
	op := &ebiten.DrawImageOptions{}
	f.renderOp(dst, op)
	geo := op.GeoM
	// At a zoom between whole steps a pixel can sample right on the border of two texels; nudged alike, every
	// picture takes the same side there, so a tile and the one it lies over never part and show a seam.
	geo.Translate(1.0/512, 1.0/512)
	b := &f.batch
	b.begin(dst, ebiten.BlendSourceOver)
	for i := range f.sceneFlat {
		fl := &f.sceneFlat[i]
		if !f.DrawBounds.Empty() && !fl.rect.Overlaps(f.DrawBounds) {
			continue
		}
		sheet, src := fl.sheet, fl.src
		if fl.water {
			sheet, src = wSheet, wSrc
		} else if fl.frames != nil {
			src = fl.frames[frame%len(fl.frames)]
		}
		b.use(sheet)
		if fl.clip == nil {
			x0, y0 := geo.Apply(float64(fl.rect.Min.X), float64(fl.rect.Min.Y))
			x1, y1 := geo.Apply(float64(fl.rect.Max.X), float64(fl.rect.Max.Y))
			b.quad(x0, y0, x1, y1, src, fl.color)
			continue
		}
		n := uint32(len(b.vs))
		for _, p := range fl.clip {
			x, y := geo.Apply(float64(fl.rect.Min.X+p.X), float64(fl.rect.Min.Y+p.Y))
			b.vs = append(b.vs, ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: float32(src.Min.X + p.X), SrcY: float32(src.Min.Y + p.Y),
				ColorR: fl.color[0], ColorG: fl.color[1], ColorB: fl.color[2], ColorA: fl.color[3]})
		}
		for j := 2; j < len(fl.clip); j++ { // a fan from the first corner: the clip is convex
			b.is = append(b.is, n, n+uint32(j-1), n+uint32(j))
		}
	}
	b.flush()
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
		sunk := slices.ContainsFunc(p.Draws, func(d scenemap.Draw) bool { return d.Kind == scenemap.DrawWaterline })
		pr, ok := f.A.Props[p.Name]
		casts := p.Rank == scenemap.RankObject || p.Rank == scenemap.RankWall && pr.Masked
		if ok && casts && !pr.NoShadow && !sunk { // the flood takes no shadow
			sp.shadow = pr.Shadow
		}
		f.scenePieces = append(f.scenePieces, sp)
	}
}

// planImage is the picture a draw of a plan lays and where its top left sits from the draw's point; nil for a
// looping strip, played over its piece every frame by drawStrip.
func (f *Field) planImage(d scenemap.Draw) (*ebiten.Image, image.Point) {
	switch d.Kind {
	case scenemap.DrawTile:
		return f.A.Tiles[d.Tile].Image, image.Point{}
	case scenemap.DrawSprite:
		switch d.Sheet {
		case "scenery":
			return f.A.Scenery.sprites[d.Name].Image, image.Point{}
		case "structures":
			return f.A.Structures[d.Name].Image, image.Point{}
		default:
			return f.A.Props[d.Name].Image, image.Point{}
		}
	case scenemap.DrawFill:
		return f.A.diamond(d.Color), image.Pt(-32, 0)
	case scenemap.DrawFlood:
		return f.A.floodTile(d.Tile), image.Pt(-16, 0)
	case scenemap.DrawWaterline:
		return f.A.waterline(d.Line)
	}
	return nil, image.Point{}
}

// planRect is where a draw of a plan lays its picture at canvas offset (ox, oy).
func (f *Field) planRect(d scenemap.Draw, ox, oy float64) image.Rectangle {
	img, at := f.planImage(d)
	if img == nil {
		return image.Rectangle{}
	}
	r := image.Rectangle{Max: img.Bounds().Size()}
	return r.Add(at.Add(image.Pt(int(ox)+d.X, int(oy)+d.Y)))
}

// drawPlan draws one picture of a plan at canvas offset (ox, oy) of dst; op carries the caller's colour.
func (f *Field) drawPlan(dst *ebiten.Image, d scenemap.Draw, ox, oy float64, op *ebiten.DrawImageOptions) {
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	img, at := f.planImage(d)
	if img == nil {
		return
	}
	op.GeoM.Translate(ox+float64(d.X+at.X), oy+float64(d.Y+at.Y))
	if d.Tint != "" { // a picture in greys takes its colour (scenemap.Tint)
		op.ColorScale.ScaleWithColor(hexColor(d.Tint))
	}
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
		items = append(items, drawItem{y: p.Anchor[1], rect: p.Rect, key: &p.Key, level: p.level, low: p.Rank == scenemap.RankRim, draw: func() {
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
