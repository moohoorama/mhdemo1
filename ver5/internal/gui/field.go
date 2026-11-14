package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
	"srpg/internal/scenemap"
)

const margin = 48 // map pixels around the map bounds (sprites reach past the ground)

// K is how many canvas pixels one map pixel covers. The map is drawn K times larger
// than its tiles while units keep their native pixels, so units stand at 1/K of the
// size they had when everything shared one scale, without losing any detail.
const K = 2

// newLayer is a picture as large as the map or the screen. It gets a texture of its own: on ebiten's shared atlas it
// would grow a page up to 16384² that any small picture left on it keeps from being freed.
func newLayer(w, h int) *ebiten.Image {
	return ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
}

// Field draws one battle map. Map pixels: cell (u, v) has its centre at
// ((u - v) * 16, (u + v + 1) * 8) relative to Origin; the canvas is K times that.
type Field struct {
	deployment                   bool
	goals                        []goalShape     // the objective guide's outlines
	RangeBorder                  []content.Point // cells ringed in red: what a picked unit strikes from where it stands
	sceneryLayer                 bool
	deploymentCells              []content.Point
	worldSize                    image.Point
	renderZoom, renderX, renderY float64
	DrawBounds                   image.Rectangle
	nativeWater                  []GroundTile
	nativeItems                  []drawItem
	nativeGround                 []drawItem
	A                            *Assets
	Map                          *MapArt
	OriginX                      float64
	OriginY                      float64
	ground                       [8]*ebiten.Image // per water frame; castle floor baked in
	clip                         *ebiten.Image    // opaque where there is ground
	shadows                      *ebiten.Image
	levels                       []sceneLevel // a format 2 scene map's raised grounds, lowest first (makeScenePlan)
	Canvas                       *ebiten.Image
	ShowExterior                 bool
	exterior                     *ebiten.Image
	outside                      outsideLayers
	night                        nightLayers
	looks                        map[image.Point]string // the look of the state on a cell, by cell
	decor                        *ebiten.Image          // baked ground decoration, see makeScenery
	treeShadow                   *ebiten.Image          // tree shadows while the field is built, then folded into decor
	cliffTop                     *ebiten.Image          // baked decoration of the cliff tops, drawn with each cliff block
	waterFX                      []scenemap.Ripple
	floorTiles                   []floorTile     // a battle map's small tiles far to near, see makeFloors
	bridges                      []drawItem      // bridge decks, over the walls under their ends
	Layers                       map[string]bool // debug: when set, only these layers of a native map are drawn, see shows
	afloat                       *ebiten.Image   // viewport scratch where a boat and its crew are put together
	wading                       *ebiten.Image   // viewport scratch where a unit standing in a ford is put together
	scenePieces                  []scenePiece    // a format 2 scene map's standing layer, see makeScenePlan
	waterChunks                  []waterChunk    // where a format 2 scene map's water plays on its ground, see playWater
	waterFrame                   int             // the water frame the ground shows
	clockMS                      float64

	// beatShadows are the ground shadows of the props a story scene's beats put down (canvas point, shadow):
	// those props are drawn over the field, but their shadows lie on the ground under the units like the map's
	beatShadows []beatShadow
	// groundMarks are pictures laid flat on the ground at canvas points: over the shadows, under everything that
	// stands (the town walk's click ring)
	groundMarks []groundMark
}

var maxBlend = ebiten.Blend{
	BlendFactorSourceRGB: ebiten.BlendFactorOne, BlendFactorSourceAlpha: ebiten.BlendFactorOne,
	BlendFactorDestinationRGB: ebiten.BlendFactorOne, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB: ebiten.BlendOperationMax, BlendOperationAlpha: ebiten.BlendOperationMax,
}

func NewField(a *Assets, m *MapArt) *Field {
	a = a.grassAssets(m.Grass)
	w, h := (m.Bounds.Dx()+2*margin)*K, (m.Bounds.Dy()+2*margin)*K
	f := &Field{worldSize: image.Pt(w, h), A: a, Map: m, OriginX: float64(margin - m.Bounds.Min.X), OriginY: float64(margin - m.Bounds.Min.Y),
		shadows: newLayer(w, h), Canvas: newLayer(w, h), ShowExterior: m.Scene}
	if !m.SunkWater {
		f.clip = newLayer(w, h) // a battle map draws its floors tile by tile (makeFloors), no map-wide ground
	}
	if m.SceneMap != nil {
		if err := f.makeScenePlan(); err != nil {
			panic(err) // the map loaded, so its plan builds: LoadAssets builds every format 2 map once
		}
		return f
	}
	water := a.Animations["water"].Frames
	for i := range f.ground {
		if m.SunkWater {
			break
		}
		if m.Native && i > 0 {
			f.ground[i] = f.ground[0]
			continue
		}
		g := newLayer(w, h)
		if m.Native {
			f.fillGround(g)
		}
		for _, t := range m.Ground {
			if i == 0 && m.Native {
				for _, id := range t.Layers {
					if id == 0 {
						f.nativeWater = append(f.nativeWater, t)
						break
					}
				}
			}
			for _, id := range t.Layers {
				if id == 0 {
					id = water[i%len(water)]
				}
				f.blitMap(g, a.Tiles[id], f.OriginX+float64(t.X), f.OriginY+float64(t.Y), nil)
			}
		}
		for v, row := range m.Tiles {
			for u, c := range row {
				if c == 'i' && !m.Native {
					x, y := f.mapCenter(float64(u), float64(v))
					f.blitMap(g, a.Structures["floor"], x, y, nil)
				}
			}
		}
		for v, row := range m.Tiles {
			for u, c := range row {
				if c == 'b' && !m.Native {
					x, y := f.Center(float64(u), float64(v))
					f.blit(g, a.Props[f.bridgeName(u, v)].Sprite, x, y, nil)
				}
			}
		}
		f.eachProp(true, func(_ string, s Sprite, x, y float64) { f.blit(g, s, x, y, nil) })
		f.ground[i] = g
	}
	if f.clip != nil {
		f.clip.DrawImage(f.ground[0], nil)
	}
	if m.Native {
		f.makeNativeFloors()
		f.makeFloors()
		f.makeExterior()
		f.makeOutside()
		if a.Scenery != nil {
			f.makeScenery()
		}
		f.nativeObjects(&f.nativeItems)
		if f.treeShadow != nil {
			f.finishScenery()
		}
	}
	f.eachProp(false, func(name string, _ Sprite, x, y float64) {
		_, lights := a.SceneRules.Attach(name, x, y)
		f.night.props = append(f.night.props, lights...)
	})
	return f
}

// eachProp calls fn with every placed prop of one layer at its canvas position; fills repeat
// the prop on every cell of their range.
func (f *Field) eachProp(ground bool, fn func(name string, s Sprite, x, y float64)) {
	for _, p := range f.Map.Props {
		pr, ok := f.A.Props[p.Name]
		if !ok || pr.Ground != ground {
			continue
		}
		if !p.Fill {
			x, y := f.Center(p.U, p.V)
			fn(p.Name, pr.Sprite, x, y)
			continue
		}
		for v := p.V; v <= p.V1; v++ {
			for u := p.U; u <= p.U1; u++ {
				x, y := f.Center(u, v)
				fn(p.Name, pr.Sprite, x, y)
			}
		}
	}
}

// Center is the canvas position of a (fractional) cell's centre.
func (f *Field) Center(u, v float64) (float64, float64) {
	x, y := f.mapCenter(u, v)
	return x * K, y * K
}

func (f *Field) mapCenter(u, v float64) (float64, float64) {
	return f.OriginX + (u-v)*16, f.OriginY + (u+v+1)*8
}

// Cell maps a canvas position back to the cell under it.
func (f *Field) Cell(x, y float64) (int, int) {
	a, b := (x/K-f.OriginX)/16, (y/K-f.OriginY)/8
	return int(math.Floor((a + b) / 2)), int(math.Floor((b - a) / 2))
}

// ToMap converts canvas pixels to map pixels.
func ToMap(x, y float64) (float64, float64) { return x / K, y / K }

// cellPath adds the ground diamond of cell (u, v), raised by lift canvas pixels, to p in render
// pixels.
func (f *Field) cellPath(p *vector.Path, u, v int, lift float64) {
	x, y := f.Center(float64(u), float64(v))
	for i, d := range [][2]float64{{0, -16}, {32, 0}, {0, 16}, {-32, 0}} {
		rx, ry := f.renderPoint(x+d[0], y+d[1]-lift)
		if i == 0 {
			p.MoveTo(float32(rx), float32(ry))
		} else {
			p.LineTo(float32(rx), float32(ry))
		}
	}
	p.Close()
}

// shows reports whether debug layer name is drawn: always, unless Layers picks some.
func (f *Field) shows(name string) bool { return f.Layers == nil || f.Layers[name] }

func (f *Field) Inside(u, v int) bool { return u >= 0 && v >= 0 && u < f.Map.W && v < f.Map.H }

func (f *Field) Tile(u, v int) byte {
	if !f.Inside(u, v) {
		return '~'
	}
	return f.Map.Tiles[v][u]
}

// Lift is how many canvas pixels a cell's surface stands above the ground (castle walls).
func (f *Field) Lift(u, v int) float64 {
	if t := f.Tile(u, v); t == 'c' || t == 'm' && f.look(u, v) != "open" {
		return float64(f.A.Rise) * f.artScale()
	} else if t == '~' && f.Map.SunkWater {
		return -sunkDepth
	} else if t == 's' && f.Map.SunkWater {
		return -fordDepth
	} else if t == 'u' && f.Map.Native {
		return knollLift
	}
	return 0
}

// markLift is how high a cell's marks (move range, goals) are drawn: on a wall's top, but on the flat ground over
// sunk water and over a rolling hill, whose heights are only in the picture: the cell is chosen on the ground.
func (f *Field) markLift(u, v int) float64 {
	if f.Tile(u, v) == 'u' {
		return 0
	}
	return max(0, f.Lift(u, v))
}

func (f *Field) blit(dst *ebiten.Image, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	f.shadeScenery(dst, op, x, y)
	op.GeoM.Translate(math.Round(x-s.PivotX), math.Round(y-s.PivotY))
	f.renderOp(dst, op)
	dst.DrawImage(s.Image, op)
}

// blitMap draws map art (tiles, scenery, structures) at a map-pixel position, K times larger.
func (f *Field) artScale() float64 {
	if f.Map.Native {
		return 1
	}
	return K
}
func (f *Field) blitMap(dst *ebiten.Image, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	f.shadeScenery(dst, op, x*K, y*K)
	k := f.artScale()
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(math.Round(x*K-s.PivotX*k), math.Round(y*K-s.PivotY*k))
	f.renderOp(dst, op)
	dst.DrawImage(s.Image, op)
}

// blitK draws a prop k times its size, its pivot on (x, y) of the canvas.
func (f *Field) blitK(dst *ebiten.Image, s Sprite, x, y, k float64) {
	op := &ebiten.DrawImageOptions{}
	f.shadeScenery(dst, op, x, y)
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(math.Round(x-s.PivotX*k), math.Round(y-s.PivotY*k))
	f.renderOp(dst, op)
	dst.DrawImage(s.Image, op)
}

// Highlight is a tinted diamond on a cell (move range, targets, cursor).
type Highlight struct {
	U, V  int
	Color color.RGBA
}

type drawItem struct {
	y     float64
	kind  int // 0 scenery, 1 unit
	draw  func()
	rect  image.Rectangle   // canvas area the sprite's image covers
	ghost func(float32)     // units: redraw at this opacity when scenery drawn later covers it
	hides bool              // scenery a unit behind it shows through faintly (castle walls and gates)
	low   bool              // scenery that only hides the feet of a unit behind it, which stays hidden (a parapet)
	key   *scenemap.Key     // a format 2 scene map's sort key; nil: made from y and rect (orderScene)
	level int               // the ground a format 2 scene map's item stands on: 0, or 1 + its index in levels
	tops  []image.Rectangle // a raised level's tops, drawn as one item: the areas that hide what lies behind
}

func (it drawItem) covers(r image.Rectangle) bool {
	if it.tops == nil {
		return it.rect.Overlaps(r)
	}
	for _, t := range it.tops {
		if t.Overlaps(r) {
			return true
		}
	}
	return false
}

// mapRect is the canvas area blitMap covers with s at map position (x, y).
func (f *Field) mapRect(s Sprite, x, y float64) image.Rectangle {
	b := s.Image.Bounds()
	k := f.artScale()
	rx, ry := int(math.Round(x*K-s.PivotX*k)), int(math.Round(y*K-s.PivotY*k))
	return image.Rect(rx, ry, rx+int(float64(b.Dx())*k), ry+int(float64(b.Dy())*k))
}

type beatShadow struct {
	x, y float64
	s    Sprite
}

type groundMark struct {
	s    Sprite
	x, y float64
}

// fieldGround is the colour around a map, beyond its margin.
var fieldGround = color.RGBA{30, 38, 46, 255}

// Draw renders the field and returns nothing; units come from the replay's visual state.
func (f *Field) Draw(clockMS float64, units []*unitVis, marks []Highlight) {
	c := f.Canvas
	f.sceneryLayer = false
	f.clockMS = clockMS
	if f.Map.Scene || f.Layers != nil {
		c.Clear() // a story scene: what lies past the picture is the scene's backdrop (or the ground colour)
	} else {
		c.Fill(fieldGround)
	}
	water := f.A.Animations["water"]
	if f.Map.Native && f.renderZoom > 0 && f.shows("background") {
		f.renderBackground(c)
	}
	if f.shows("ground") {
		if f.Map.SunkWater {
			f.drawFloors(c, clockMS)
		} else {
			frame := int(clockMS/float64(water.MS)) % len(f.ground)
			f.playWater(frame)
			groundOp := &ebiten.DrawImageOptions{}
			f.renderOp(c, groundOp)
			c.DrawImage(f.ground[frame], groundOp)
		}
	}
	if f.Map.Native {
		if f.shows("water") && !f.Map.SunkWater {
			f.drawNativeWater(c, clockMS)
		}
		for _, it := range f.nativeGround {
			if (f.DrawBounds.Empty() || it.rect.Overlaps(f.DrawBounds)) && f.shows("floors") {
				it.draw()
			}
		}
		if f.shows("walls") {
			f.drawWalls(c)
		}
		if f.decor != nil {
			if f.shows("decor") {
				f.drawBaked(c, f.decor)
			}
			if f.shows("waterfx") {
				f.drawWaterFX()
			}
		}
		for _, it := range f.bridges {
			if f.DrawBounds.Empty() || it.rect.Overlaps(f.DrawBounds) {
				it.draw()
			}
		}
	}
	if !f.shows("objects") {
		return
	}
	// Shade the out-of-battle ground only; scenery and units are drawn afterward.
	if f.exterior != nil && !f.ShowExterior {
		op := &ebiten.DrawImageOptions{}
		f.renderOp(c, op)
		c.DrawImage(f.exterior, op)
		f.drawOutsideGround(clockMS)
	}
	if f.Map.Native {
		f.drawStateGround(c)
	}
	f.drawDeploymentGround()
	// highlights of one colour share a path, so neighbouring cells meet without seams
	paths := map[color.RGBA]*vector.Path{}
	var colors []color.RGBA
	for _, m := range marks {
		p := paths[m.Color]
		if p == nil {
			p = &vector.Path{}
			paths[m.Color] = p
			colors = append(colors, m.Color)
		}
		f.cellPath(p, m.U, m.V, f.markLift(m.U, m.V)) // sunk water, a hill: the cell's footprint, where a click lands
	}
	for _, col := range colors {
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleWithColor(color.NRGBA(col)) // Highlight colors are straight alpha
		vector.FillPath(c, paths[col], nil, op)
	}
	f.drawRangeBorder(c)
	f.drawGoals(c, 1)
	f.shadows.Clear()
	for _, l := range f.levels {
		l.shadows.Clear()
	}
	items := []drawItem{}
	for _, d := range f.Map.Decorations {
		sid := f.A.TileFrame(d.Animation, clockMS+float64(d.Phase)*125)
		x, y := f.OriginX+float64(d.X), f.OriginY+float64(d.Y)
		if s, ok := f.A.TileShadow[sid]; ok {
			f.blitMap(f.shadows, s, x, y, &ebiten.DrawImageOptions{Blend: maxBlend})
		}
		s := f.A.Tiles[sid]
		if !f.DrawBounds.Empty() && !f.mapRect(s, x, y).Overlaps(f.DrawBounds) {
			continue
		}
		items = append(items, drawItem{y: y * K, draw: func() { f.blitMap(c, s, x, y, nil) }, rect: f.mapRect(s, x, y)})
	}
	if f.Map.SceneMap != nil {
		items = append(items, f.sceneItems()...)
	}
	if f.Map.Native {
		for _, it := range append(append([]drawItem{}, f.nativeItems...), f.stateItems(clockMS)...) {
			if f.DrawBounds.Empty() || it.rect.Empty() || it.rect.Overlaps(f.DrawBounds) {
				items = append(items, it)
			}
		}
	}
	for v, row := range f.Map.Tiles {
		for u, t := range row {
			if f.Map.Native {
				continue
			}
			x, y := f.mapCenter(float64(u), float64(v))
			switch t {
			case 'v':
				s := f.A.Structures["village"]
				items = append(items, drawItem{y: y*K - .1, draw: func() { f.blitMap(c, s, x, y, nil) }, rect: f.mapRect(s, x, y)})
			case 'f':
				if !f.Map.Deformed {
					break // scenario stages keep the big trees
				}
				s := f.A.Props["forest_"+string(rune('0'+(u*7+v*3)%4))].Sprite // pines at the characters' scale, drawn after a unit on the cell so it stands among the trees
				cx, cy := x*K, y*K
				b := s.Image.Bounds()
				rx, ry := int(math.Round(cx-s.PivotX)), int(math.Round(cy-s.PivotY))
				items = append(items, drawItem{y: cy + .1, draw: func() { f.blit(c, s, cx, cy, nil) }, rect: image.Rect(rx, ry, rx+b.Dx(), ry+b.Dy())})
			case 'c', 'h':
				s := f.A.Structures[blockSprite(f, u, v)]
				items = append(items, drawItem{y: y * K, draw: func() { f.blitMap(c, s, x, y, nil) }, rect: f.mapRect(s, x, y)})
			case 'k': // 주둔지: a tent on its cell
				s := f.A.Props[[]string{"tent", "tent_round"}[(u+v)%2]].Sprite
				cx, cy := x*K, y*K
				b := s.Image.Bounds()
				rx, ry := int(math.Round(cx-s.PivotX)), int(math.Round(cy-s.PivotY))
				items = append(items, drawItem{y: cy - .1, draw: func() { f.blit(c, s, cx, cy, nil) }, rect: image.Rect(rx, ry, rx+b.Dx(), ry+b.Dy())})
			case 'g':
				s := f.A.Props[gateName(f, u, v)].Sprite
				cx, cy := x*K, y*K
				b := s.Image.Bounds()
				rx, ry := int(math.Round(cx-s.PivotX)), int(math.Round(cy-s.PivotY))
				items = append(items, drawItem{y: cy, draw: func() { f.blit(c, s, cx, cy, nil) }, rect: image.Rect(rx, ry, rx+b.Dx(), ry+b.Dy())})
			}
		}
	}
	f.eachProp(false, func(name string, s Sprite, x, y float64) {
		b := s.Image.Bounds()
		rx, ry := int(math.Round(x-s.PivotX)), int(math.Round(y-s.PivotY))
		anims, _ := f.A.SceneRules.Attach(name, x, y)
		rect := image.Rect(rx, ry, rx+b.Dx(), ry+b.Dy())
		for _, d := range anims {
			rect = rect.Union(f.A.Anims[d.Name].Frames[0].Image.Bounds().Add(image.Pt(d.X, d.Y)))
		}
		if !f.DrawBounds.Empty() && !rect.Overlaps(f.DrawBounds) {
			return
		}
		items = append(items, drawItem{y: y, draw: func() {
			f.blit(c, s, x, y, nil)
			for _, d := range anims {
				f.drawStrip(d, 0, 0)
			}
		}, rect: rect})
	})
	for _, b := range f.beatShadows {
		f.blit(f.shadows, b.s, b.x, b.y, &ebiten.DrawImageOptions{Blend: maxBlend})
	}
	for _, u := range units {
		if u.gone {
			continue
		}
		x, y := f.Center(u.u, u.v)
		x += u.shakeOffset()
		ground := f.groundLift(u.u, u.v)
		lift := u.lift + ground
		f.updateWade(u)
		if u.visible() {
			anim, frame := f.pose(u)
			sh := Sprite{u.art.Shadow(u.dir, anim, frame), u.art.ShadowPivot[0], u.art.ShadowPivot[1]}
			sop := &ebiten.DrawImageOptions{Blend: maxBlend}
			sop.ColorScale.ScaleAlpha(float32((1 - u.leaving()) * (1 - .7*u.wade)))
			f.castShadow(ground, sh, x, y-lift, sop)
			if boat, k := f.boat(u); k > 0 {
				bop := &ebiten.DrawImageOptions{Blend: maxBlend}
				bop.ColorScale.ScaleAlpha(float32(k * (1 - u.leaving())))
				f.castShadow(ground, f.A.Props[boat].Shadow, x, y-lift, bop)
			}
		}
		unit := u
		b := u.art.Frame(u.dir, u.anim, u.frame()).Bounds()
		rx, ry := int(math.Round(x-u.art.Pivot[0])), int(math.Round(y-lift-u.art.Pivot[1]))
		it := drawItem{y: y, kind: 1, draw: func() { f.drawUnit(unit, x, y-lift, 0) },
			rect: image.Rect(rx, ry, rx+b.Dx(), ry+b.Dy()), ghost: func(a float32) { f.drawUnit(unit, x, y-lift, a) }}
		if f.Map.SceneMap != nil {
			k := f.sceneKey(x, y, ground, scenemap.RankPerson, it.rect)
			it.key, it.level = &k, f.levelOf(ground)
		}
		items = append(items, it)
	}
	// one layer so overlapping shadows do not darken each other, kept on the ground
	if f.clip != nil {
		clipOp := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn}
		f.renderOp(f.shadows, clipOp)
		f.shadows.DrawImage(f.clip, clipOp)
	}
	c.DrawImage(f.shadows, nil)
	for _, m := range f.groundMarks {
		f.blit(c, m.s, m.x, m.y, nil)
	}
	if f.Map.SceneMap != nil {
		items = f.drawSceneLevels(items)
	} else {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].y != items[j].y {
				return items[i].y < items[j].y
			}
			return items[i].kind < items[j].kind
		})
		for _, it := range items {
			f.sceneryLayer = it.kind == 0
			it.draw()
		}
	}
	f.sceneryLayer = false
	// a unit behind a tree or house shows through it at half opacity, behind a castle wall or gate
	// faintly; where nothing covers it the ghost lands on its own pixels and changes nothing
	for i, it := range items {
		if it.kind != 1 {
			continue
		}
		var alpha float32
		for _, later := range items[i+1:] {
			if later.kind == 0 && !later.low && later.covers(it.rect) {
				if later.hides {
					alpha = .3
					break
				}
				alpha = .5
			}
		}
		if alpha > 0 {
			it.ghost(alpha)
		}
	}
	// the guide again, faint, through a castle on its cell or a horse standing on it
	f.drawGoals(c, goalThrough)
	if !f.ShowExterior {
		f.drawCloudShadows(clockMS)
	}
	f.drawNight(clockMS, units)
	for _, u := range units {
		x, y := f.Center(u.u, u.v)
		if f.night.on && !u.gone && u.visible() {
			f.unitBar(u, x+u.shakeOffset(), y-u.lift)
		}
		f.drawUnitStatuses(u, x+u.shakeOffset(), y-u.lift, clockMS)
	}
}

// gateName turns a gate along its wall line: along u when a wall or gate continues on the u sides.
func gateName(f *Field, u, v int) string {
	wall := func(t byte) bool { return t == 'c' || t == 'g' }
	if wall(f.Tile(u-1, v)) || wall(f.Tile(u+1, v)) {
		return "gate_u"
	}
	return "gate_v"
}

// bridgeName lays a bridge cell across the water: along the axis whose run of bridge cells
// ends on land at both ends; when both or neither do, along the longer run (u on a tie).
func (f *Field) bridgeName(u, v int) string {
	span := func(du, dv int) (int, bool) {
		n, k0, k1 := 0, 1, 1
		for ; f.Tile(u+k1*du, v+k1*dv) == 'b'; k1++ {
			n++
		}
		for ; f.Tile(u-k0*du, v-k0*dv) == 'b'; k0++ {
			n++
		}
		return n, f.Tile(u+k1*du, v+k1*dv) != '~' && f.Tile(u-k0*du, v-k0*dv) != '~'
	}
	nu, lu := span(1, 0)
	nv, lv := span(0, 1)
	if lu != lv {
		if lu {
			return "bridge_u"
		}
		return "bridge_v"
	}
	if nv > nu {
		return "bridge_v"
	}
	return "bridge_u"
}

// sameBlock reports whether neighbour n continues the block of tile t: the wall runs on into a gate, shut, open or
// a pass, whose gatehouse stands to the wall's height at both ends.
func sameBlock(t, n byte) bool {
	wall := func(t byte) bool { return t == 'c' || t == 'm' || t == 'g' }
	return n == t || wall(t) && wall(n)
}

// blockSprite picks the wall or cliff sprite whose open faces look at the cell's outside neighbours.
func blockSprite(f *Field, u, v int) string {
	t := f.Tile(u, v)
	mask := 0
	for i, d := range []content.Point{{X: -1}, {Y: -1}, {X: 1}, {Y: 1}} { // nw, ne, se, sw
		if !sameBlock(t, f.Tile(u+d.X, v+d.Y)) {
			mask |= 1 << i
		}
	}
	return blockVariant(t, mask, u, v)
}

// blockVariant names the sprite; cliffs come in variants picked by position so long faces do not repeat.
func blockVariant(t byte, mask, x, y int) string {
	if t == 'h' {
		return fmt.Sprintf("cliff_%02d_%d", mask, ((x*7+y*3)%3+3)%3)
	}
	return fmt.Sprintf("wall_%02d", mask)
}

// drawUnit draws u at canvas point (x, y); a ghost (opacity above 0) is the unit redrawn faintly over what covers it.
func (f *Field) drawUnit(u *unitVis, x, y float64, ghost float32) {
	if !u.visible() {
		return
	}
	op := &ebiten.DrawImageOptions{}
	if ghost > 0 {
		op.ColorScale.ScaleAlpha(ghost)
	}
	if u.status["poison"].Turns > 0 {
		op.ColorScale.Scale(.82, .95, .76, 1)
	}
	if u.status["weak"].Turns > 0 {
		op.ColorScale.Scale(.7, .7, .8, 1)
	}
	if f.deployment && !u.ally {
		op.ColorScale.Scale(.45, .45, .45, 1)
	}
	if u.greyed {
		op.ColorScale.Scale(.62, .62, .62, 1)
	}
	if u.flash > 0 {
		k := float32(1 + u.flash*1.6)
		op.ColorScale.Scale(k, k, k, 1)
	}
	if k := u.leaving(); k > 0 {
		b := float32(f.edgeShade(u.u, u.v))
		op.ColorScale.Scale(b, b, b, 1)
		op.ColorScale.ScaleAlpha(float32(1 - k*k))
	}
	anim, frame := f.pose(u)
	s := Sprite{u.art.Frame(u.dir, anim, frame), u.art.Pivot[0], u.art.Pivot[1]}
	if boat, k := f.boat(u); k > 0 {
		f.drawAfloat(u, s, boat, k, x, y, op)
	} else if u.wade > .01 {
		f.drawWading(u, s, x, y, op)
	} else {
		f.blit(f.Canvas, s, x, y, op)
	}
	if k := .9 * u.charge; k > 0 && ghost == 0 { // gathering a critical blow: a white silhouette fades in over the sprite
		var cm colorm.ColorM
		cm.Scale(0, 0, 0, k)
		cm.Translate(1, 1, 1, 0)
		wop := &colorm.DrawImageOptions{}
		wop.GeoM.Translate(math.Round(x-s.PivotX), math.Round(y-s.PivotY))
		if f.renderZoom > 0 {
			wop.GeoM.Translate(-f.renderX, -f.renderY)
			wop.GeoM.Scale(f.renderZoom, f.renderZoom)
		}
		colorm.DrawImage(f.Canvas, s.Image, cm, wop)
	}
	if !f.night.on { // at night the bars go over the dark, after everything else
		f.unitBar(u, x, y)
	}
}

// unitBar draws u's bar under its feet at canvas point (x, y).
func (f *Field) unitBar(u *unitVis, x, y float64) {
	if u.shownHP <= 0 || u.actor {
		return
	}
	rx, ry := f.renderPoint(x, y+unitBarDrop)
	z := float32(1)
	if f.renderZoom > 0 {
		z = float32(f.renderZoom)
	}
	f.drawUnitBar(u, float32(math.Round(rx)), float32(math.Round(ry)), z)
}

// Bounds of the canvas, for the camera.
func (f *Field) Size() image.Point { return f.worldSize }
