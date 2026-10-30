package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
)

func (f *Field) small(x, y int) byte {
	v := f.Map.Visual
	if x < v[0] || y < v[1] || x >= v[0]+v[2] || y >= v[1]+v[3] {
		return '~'
	}
	return f.Map.SmallTiles[y-v[1]][x-v[0]]
}
func (f *Field) eachSmall(fn func(int, int, byte, float64, float64)) {
	v := f.Map.Visual
	for y := v[1]; y < v[1]+v[3]; y++ {
		for x := v[0]; x < v[0]+v[2]; x++ {
			px, py := f.mapCenter((float64(x)+.5)/2-.5, (float64(y)+.5)/2-.5)
			fn(x, y, f.small(x, y), px, py)
		}
	}
}

// shown is small tile (x, y) as it stands now: a pass and an open gate are castle floor under their open gatehouse.
func (f *Field) shown(x, y int) byte {
	t := f.small(x, y)
	if t == 'g' || t == 'm' && f.look(floorDiv(x, 2), floorDiv(y, 2)) == "open" {
		return 'i'
	}
	return t
}

// smallBlock picks the raised-block sprite (wall or cliff) whose open faces look at the
// neighbours outside the block.
func (f *Field) smallBlock(x, y int) string {
	t := f.small(x, y)
	mask := 0
	for i, d := range [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
		if !sameBlock(t, f.small(x+d[0], y+d[1])) {
			mask |= 1 << i
		}
	}
	return blockVariant(t, mask, x, y)
}

// Floors and bridges must be drawn after animated water so wet atlas fragments cannot cover them; bridges go
// after the banks too, which run on under their ends.
func (f *Field) makeNativeFloors() {
	f.eachSmall(func(x, y int, t byte, px, py float64) {
		if t == 'i' || t == 'm' || t == 'g' {
			s := f.A.Structures["floor"]
			f.nativeGround = append(f.nativeGround, drawItem{rect: f.mapRect(s, px, py), draw: func() {
				if f.shown(x, y) == 'i' { // a gate shows its floor once open
					f.blitMap(f.Canvas, s, px, py, nil)
				}
			}})
		}
		if t == 'b' {
			name := "bridge_u"
			if f.small(x, y-1) == 'b' || f.small(x, y+1) == 'b' {
				name = "bridge_v"
			}
			s := f.A.Props[name].Sprite
			at := image.Pt(content.FloorHalf(x), content.FloorHalf(y))
			f.bridges = append(f.bridges, drawItem{rect: f.mapRect(s, px, py), draw: func() {
				if f.shows("bridge-deck") {
					f.nativeProp(f.Canvas, f.bridgeSprite(name, at, s), px, py)
				}
			}})
			f.bridges = append(f.bridges, f.bridgeEdges(x, y, at, px, py)...)
		}
	})
	f.eachGameCell('u', func(x, y int, px, py float64) {
		s := f.A.Props[fmt.Sprintf("bt_knoll_%02d", f.cellMask(x, y, 'u'))].Sprite
		f.nativeGround = append(f.nativeGround, drawItem{rect: f.mapRect(s, px/K, py/K), draw: func() { f.blit(f.Canvas, s, px, py, nil) }})
	})
}

// knollLift is how high a unit on a rolling hill (u) stands: the crest of its picture (tools/worldmap/gen_terrain.py
// KNOLL). Only the unit rises; its cell's marks stay on the ground (markLift).
const knollLift = 7

// eachGameCell calls fn with every game cell of the picture whose small tiles are all glyph t, and its canvas centre.
func (f *Field) eachGameCell(t byte, fn func(x, y int, px, py float64)) {
	v := f.Map.Visual
	for y := v[1] / 2; y < (v[1]+v[3])/2; y++ {
		for x := v[0] / 2; x < (v[0]+v[2])/2; x++ {
			if f.small(2*x, 2*y) == t && f.small(2*x+1, 2*y+1) == t {
				px, py := f.Center(float64(x), float64(y))
				fn(x, y, px, py)
			}
		}
	}
}

// cellMask has bit i set when the game cell next to (x, y) at side i (-u, -v, +u, +v) is glyph t too.
func (f *Field) cellMask(x, y int, t byte) int {
	mask := 0
	for i, d := range [4][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
		if f.small(2*(x+d[0]), 2*(y+d[1])) == t {
			mask |= 1 << i
		}
	}
	return mask
}

// nativeMountains stands a mountain on every mountain cell (n): bare rock, a big one where mountains ring it.
func (f *Field) nativeMountains(items *[]drawItem) {
	f.eachGameCell('n', func(x, y int, px, py float64) {
		h := ((x*7+y*13+x*y*3)%97 + 97) % 97
		name := fmt.Sprintf("wm_mtn_rock_%d", h%3)
		if f.cellMask(x, y, 'n') == 15 && h%3 == 0 {
			name = fmt.Sprintf("wm_mtn_big_%d", h%2)
		}
		s := f.A.Props[name].Sprite
		*items = append(*items, drawItem{y: py, rect: f.mapRect(s, px/K, py/K), draw: func() { f.blit(f.Canvas, s, px, py, nil) }})
	})
}

func (f *Field) nativeObjects(items *[]drawItem) {
	f.eachSmall(func(u, v int, t byte, x, y float64) {
		if t != 'c' && t != 'h' && t != 'm' && t != 'g' {
			return
		}
		s := f.A.Structures[f.smallBlock(u, v)]
		if t != 'm' && t != 'g' {
			top := t == 'h'
			*items = append(*items, drawItem{y: y * K, rect: f.mapRect(s, x, y), hides: t == 'c', draw: func() {
				f.blitMap(f.Canvas, s, x, y, nil)
				if top {
					f.drawCliffTop(x*K, y*K)
				}
			}})
			return
		}
		closed, open := f.gateSprite(u, v, s, "closed"), f.gateSprite(u, v, s, "open")
		*items = append(*items, drawItem{y: y * K, rect: f.mapRect(closed, x, y).Union(f.mapRect(open, x, y)), hides: true, draw: func() {
			if f.shown(u, v) == 'm' {
				f.blitMap(f.Canvas, closed, x, y, nil)
			} else {
				f.blitMap(f.Canvas, open, x, y, nil)
			}
		}})
	})
	f.nativeSites(items)
	f.nativeMountains(items)
	f.nativeFences(items)
	if f.A.Scenery != nil {
		f.nativeForest(items)
		return
	}
	// Preserve the number and native size of forest clusters, including exterior forests.
	v := f.Map.Visual
	for y := v[1] / 2; y < (v[1]+v[3])/2; y++ {
		for x := v[0] / 2; x < (v[0]+v[2])/2; x++ {

			forest := false
			for j := 0; j < 2; j++ {
				for i := 0; i < 2; i++ {
					forest = forest || f.small(2*x+i, 2*y+j) == 'f'
				}
			}
			if !forest {
				continue
			}
			n := ((x*7+y*3)%4 + 4) % 4
			s := f.A.Props[fmt.Sprintf("forest_%d", n)].Sprite
			px, py := f.Center(float64(x), float64(y))
			at := image.Pt(x, y)
			*items = append(*items, drawItem{y: py + .1, rect: f.mapRect(s, px/K, py/K), draw: func() {
				if burnt, ok := f.A.Props[fmt.Sprintf("burnt_forest_%d", n)]; ok && f.look(at.X, at.Y) == "soot" {
					f.blit(f.Canvas, burnt.Sprite, px, py, nil)
					return
				}
				f.blit(f.Canvas, s, px, py, nil)
			}})
		}
	}
}

// siteSprite names the strongpoint sprite of each one-cell site tile.
var siteSprite = map[byte]string{'k': "camp", 'v': "village", 'o': "castle"}

// burntSite names the picture a site burns down to; the castle does not burn.
var burntSite = map[byte]string{'k': "burnt_camp", 'v': "burnt_village"}

// siteVariants is how many pictures a site has, named <name>_0, _1, ...: a cell takes picture (x + 2y) mod n, so
// with four pictures its eight neighbours all take another one than it does.
var siteVariants = map[byte]int{'v': 4}

// nativeSites draws one strongpoint per game cell, at the characters' scale like the forests. A cell is a
// site when most of its small tiles are.
func (f *Field) nativeSites(items *[]drawItem) {
	cellHas := func(x, y int, t byte) int {
		n := 0
		for j := 0; j < 2; j++ {
			for i := 0; i < 2; i++ {
				if f.small(2*x+i, 2*y+j) == t {
					n++
				}
			}
		}
		return n
	}
	v := f.Map.Visual
	for y := v[1] / 2; y < (v[1]+v[3])/2; y++ {
		for x := v[0] / 2; x < (v[0]+v[2])/2; x++ {
			var site byte
			best := 0
			for _, t := range []byte{'k', 'v', 'o'} {
				if n := cellHas(x, y, t); n > best {
					site, best = t, n
				}
			}
			if site == 0 {
				continue
			}
			name, burntName := siteSprite[site], burntSite[site]
			s := f.A.Sites[name]
			if n := siteVariants[site]; n > 0 {
				k := fmt.Sprintf("_%d", ((x+2*y)%n+n)%n)
				name, burntName = name+k, burntName+k
			}
			if p, ok := f.A.Props[name]; ok { // the pixel-kit picture replaces sites.png's
				s = p.Sprite
			}
			px, py := f.Center(float64(x), float64(y))
			at := image.Pt(x, y)
			still := func() { f.blit(f.Canvas, s, px, py, nil) }
			if f.A.Scenery != nil {
				if life, ok := f.A.Scenery.sites[name]; ok {
					still = func() { f.drawSite(s, life, px, py, x, y) }
				}
			}
			*items = append(*items, drawItem{y: py - .1, rect: f.mapRect(s, px/K, py/K), draw: func() {
				if burnt, ok := f.A.Props[burntName]; ok && f.look(at.X, at.Y) == "soot" {
					f.blit(f.Canvas, burnt.Sprite, px, py, nil)
					return
				}
				still()
			}})
		}
	}
}

func (f *Field) drawNativeWater(dst *ebiten.Image, ms float64) {
	water := f.A.Animations["water"]
	frame := water.Frames[int(ms/float64(water.MS))%len(water.Frames)]
	for _, t := range f.nativeWater {
		if !f.DrawBounds.Empty() && !f.mapRect(f.A.Tiles[frame], f.OriginX+float64(t.X), f.OriginY+float64(t.Y)).Overlaps(f.DrawBounds) {
			continue
		}
		for _, id := range t.Layers {
			if id == 0 {
				id = frame
			}
			f.blitMap(dst, f.A.Tiles[id], f.OriginX+float64(t.X), f.OriginY+float64(t.Y), nil)
		}
	}
}
func (f *Field) makeExterior() {
	w, h := f.Canvas.Bounds().Dx(), f.Canvas.Bounds().Dy()
	f.exterior = newLayer(w, h)
	f.exterior.Fill(color.NRGBA{A: 140})
	var p vector.Path
	// Grid vertices differ from cell centers by half a cell along both axes.
	for i, uv := range [][2]float64{{-.5, -.5}, {float64(f.Map.W) - .5, -.5}, {float64(f.Map.W) - .5, float64(f.Map.H) - .5}, {-.5, float64(f.Map.H) - .5}} {
		x, y := f.Center(uv[0], uv[1])
		if i == 0 {
			p.MoveTo(float32(x), float32(y))
		} else {
			p.LineTo(float32(x), float32(y))
		}
	}
	p.Close()
	vector.FillPath(f.exterior, &p, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
}
func colorScaleWhite() ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleWithColor(color.White)
	return c
}

func (f *Field) nativeProp(dst *ebiten.Image, s Sprite, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	f.shadeScenery(dst, op, x*K, y*K)
	op.GeoM.Scale(.5, .5)
	op.GeoM.Translate(x*K-s.PivotX*.5, y*K-s.PivotY*.5)
	f.renderOp(dst, op)
	dst.DrawImage(s.Image, op)
}
func (a *Assets) pattern() *ebiten.Image {
	if a.groundPattern != nil {
		return a.groundPattern
	}
	im := ebiten.NewImage(64, 32)
	s := a.Tiles[23]
	for y := -8; y < 40; y += 4 {
		for x := -8; x < 80; x += 16 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x+(y/4&1)*8)-s.PivotX, float64(y)-s.PivotY)
			im.DrawImage(s.Image, op)
		}
	}
	a.groundPattern = im
	return im
}
func (f *Field) fillGround(dst *ebiten.Image) {
	pattern := f.A.pattern()
	for y := 0; y < dst.Bounds().Dy(); y += 32 {
		for x := 0; x < dst.Bounds().Dx(); x += 64 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			dst.DrawImage(pattern, op)
		}
	}
}
func (f *Field) background(dst *ebiten.Image, r image.Rectangle, z, ox, oy float64, bright bool) {
	p := f.A.pattern()
	w, h := 64*z, 32*z
	startX := float64(r.Min.X) + math.Mod(ox-float64(r.Min.X), w) - w
	startY := float64(r.Min.Y) + math.Mod(oy-float64(r.Min.Y), h) - h
	for y := startY; y < float64(r.Max.Y); y += h {
		for x := startX; x < float64(r.Max.X); x += w {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(x, y)
			if !bright {
				op.ColorScale.Scale(.45, .45, .45, 1)
			}
			drawScaled(dst, p, op)
		}
	}
}

// bridgeSprite is the bridge a cell shows: whole, damaged or broken by its state.
func (f *Field) bridgeSprite(name string, at image.Point, whole Sprite) Sprite {
	switch f.look(at.X, at.Y) {
	case "damaged", "broken":
		if p, ok := f.A.Props[name+"_"+f.look(at.X, at.Y)]; ok {
			return p.Sprite
		}
	}
	return whole
}

// gateSprite is small tile (u, v)'s piece of the gatehouse over its game cell in a picture set ("closed", "open"), turned to
// the wall line the cell sits in. A piece is named by its place in the cell, du + 2dv.
func (f *Field) gateSprite(u, v int, wall Sprite, set string) Sprite {
	isWall := func(t byte) bool { return t == 'c' || t == 'm' || t == 'g' }
	x, y := 2*floorDiv(u, 2), 2*floorDiv(v, 2)
	axis := "v"
	if isWall(f.small(x-1, y)) || isWall(f.small(x+2, y)) {
		axis = "u"
	}
	name := fmt.Sprintf("gatehouse_%s_%s_%d", set, axis, u-x+2*(v-y))
	if p, ok := f.A.Props[name]; ok {
		return p.Sprite
	}
	return wall
}

// nativeFences puts a fence on each fenced game cell, joined to the fences next to it.
func (f *Field) nativeFences(items *[]drawItem) {
	v := f.Map.Visual
	fence := func(x, y int) bool { return f.small(2*x, 2*y) == 'z' }
	for y := v[1] / 2; y < (v[1]+v[3])/2; y++ {
		for x := v[0] / 2; x < (v[0]+v[2])/2; x++ {
			if !fence(x, y) {
				continue
			}
			mask := 0
			for i, d := range [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} { // nw, ne, se, sw
				if fence(x+d[0], y+d[1]) {
					mask |= 1 << i
				}
			}
			s, ok := f.A.Props[fmt.Sprintf("fence_%02d", mask)]
			if !ok {
				continue
			}
			px, py := f.Center(float64(x), float64(y))
			sprite := s.Sprite
			*items = append(*items, drawItem{y: py, rect: f.mapRect(sprite, px/K, py/K), draw: func() { f.blit(f.Canvas, sprite, px, py, nil) }})
		}
	}
}

// floorDiv is a / b rounded down, so small tiles left of or above the origin fall in the right cell.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
