package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// Floors and bridges must be drawn after animated water so wet atlas fragments cannot cover them.
func (f *Field) makeNativeFloors() {
	f.eachSmall(func(x, y int, t byte, px, py float64) {
		if t == 'i' {
			s := f.A.Structures["floor"]
			f.nativeGround = append(f.nativeGround, drawItem{rect: f.mapRect(s, px, py), draw: func() { f.blitMap(f.Canvas, s, px, py, nil) }})
		}
		if t == 'b' {
			name := "bridge_u"
			if f.small(x, y-1) == 'b' || f.small(x, y+1) == 'b' {
				name = "bridge_v"
			}
			s := f.A.Props[name].Sprite
			f.nativeGround = append(f.nativeGround, drawItem{rect: f.mapRect(s, px, py), draw: func() { f.nativeProp(f.Canvas, s, px, py) }})
		}
	})
}

func (f *Field) nativeObjects(items *[]drawItem) {
	f.eachSmall(func(u, v int, t byte, x, y float64) {
		if t != 'c' && t != 'h' {
			return
		}
		s := f.A.Structures[f.smallBlock(u, v)]
		*items = append(*items, drawItem{y: y * K, rect: f.mapRect(s, x, y), draw: func() { f.blitMap(f.Canvas, s, x, y, nil) }})
	})
	f.nativeSites(items)
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
			s := f.A.Props[fmt.Sprintf("forest_%d", ((x*7+y*3)%4+4)%4)].Sprite
			px, py := f.Center(float64(x), float64(y))
			*items = append(*items, drawItem{y: py + .1, rect: f.mapRect(s, px/K, py/K), draw: func() { f.blit(f.Canvas, s, px, py, nil) }})
		}
	}
}

// siteSprite names the strongpoint sprite of each one-cell site tile.
var siteSprite = map[byte]string{'k': "camp", 'v': "village", 'o': "castle", 'g': "gate"}

// nativeSites draws one strongpoint per game cell, at the characters' scale like the forests. A cell is a
// site when most of its small tiles are; a gate turns to the castle wall line it sits in.
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
			for _, t := range []byte{'k', 'v', 'o', 'g'} {
				if n := cellHas(x, y, t); n > best {
					site, best = t, n
				}
			}
			if site == 0 {
				continue
			}
			name := siteSprite[site]
			if site == 'g' {
				name = "gate_u"
				if cellHas(x, y-1, 'c') > 0 || cellHas(x, y+1, 'c') > 0 {
					name = "gate_v"
				}
			}
			s := f.A.Sites[name]
			px, py := f.Center(float64(x), float64(y))
			*items = append(*items, drawItem{y: py - .1, rect: f.mapRect(s, px/K, py/K), draw: func() { f.blit(f.Canvas, s, px, py, nil) }})
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
	f.exterior = ebiten.NewImage(w, h)
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
