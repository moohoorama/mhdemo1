package gui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
)

// A map's grass can lean toward its climate (content.GrassTint): the field draws from copies of the ground tiles
// and the ground decoration in which only the green pixels turned. Soil, water, stone and flowers stay as drawn.

// grassShaderSrc turns the hue of green pixels by Hue (a turn is 1) and scales their saturation and value; the turn
// fades out toward the edges of the green band, so grass running into soil or water shows no seam.
const grassShaderSrc = `//kage:unit pixels
package main

var Hue float
var Sat float
var Val float

func rgb2hsv(c vec3) vec3 {
	k := vec4(0.0, -1.0/3.0, 2.0/3.0, -1.0)
	p := mix(vec4(c.bg, k.wz), vec4(c.gb, k.xy), step(c.b, c.g))
	q := mix(vec4(p.xyw, c.r), vec4(c.r, p.yzx), step(p.x, c.r))
	d := q.x - min(q.w, q.y)
	e := 1.0e-10
	return vec3(abs(q.z+(q.w-q.y)/(6.0*d+e)), d/(q.x+e), q.x)
}

func hsv2rgb(c vec3) vec3 {
	k := vec4(1.0, 2.0/3.0, 1.0/3.0, 3.0)
	p := abs(fract(c.xxx+k.xyz)*6.0 - k.www)
	return c.z * mix(k.xxx, clamp(p-k.xxx, 0.0, 1.0), c.y)
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a == 0.0 {
		return c
	}
	rgb := c.rgb / c.a
	hsv := rgb2hsv(rgb)
	// green: hue 50° to 170°, with a soft 20° edge; grey and near-black pixels are left alone
	w := smoothstep(50.0/360.0, 70.0/360.0, hsv.x) * (1.0 - smoothstep(150.0/360.0, 170.0/360.0, hsv.x))
	w *= smoothstep(0.08, 0.2, hsv.y)
	turned := vec3(fract(hsv.x+Hue), clamp(hsv.y*Sat, 0.0, 1.0), clamp(hsv.z*Val, 0.0, 1.0))
	out := mix(rgb, hsv2rgb(turned), w)
	return vec4(out*c.a, c.a)
}
`

var grassShader *ebiten.Shader

// grassAssets is a with the map's grass: the same assets when the map keeps the tileset's, else a copy whose
// ground tiles and ground decoration are turned, made once per tint. Whole sheets are turned and the pictures cut
// again from them, so the turned pictures of one sheet still draw in one call (batch).
func (a *Assets) grassAssets(t *content.GrassTint) *Assets {
	if t == nil || *t == (content.GrassTint{}) {
		return a
	}
	if c, ok := a.grassTints[*t]; ok {
		return c
	}
	if grassShader == nil {
		s, err := ebiten.NewShader([]byte(grassShaderSrc))
		if err != nil {
			panic(err)
		}
		grassShader = s
	}
	sat, val := t.Sat, t.Val
	if sat == 0 {
		sat = 1
	}
	if val == 0 {
		val = 1
	}
	uniforms := map[string]any{"Hue": float32(t.Hue / 360), "Sat": float32(sat), "Val": float32(val)}
	turn := func(img *ebiten.Image) *ebiten.Image {
		if img == nil {
			return nil
		}
		b := img.Bounds()
		out := ebiten.NewImage(b.Dx(), b.Dy())
		op := &ebiten.DrawRectShaderOptions{Uniforms: uniforms, Images: [4]*ebiten.Image{img}}
		out.DrawRectShader(b.Dx(), b.Dy(), grassShader, op)
		return out
	}
	turned := map[*ebiten.Image]*ebiten.Image{}
	recut := func(img *ebiten.Image) *ebiten.Image {
		if img == nil {
			return nil
		}
		sheet := sheetFor(img)
		t, ok := turned[sheet]
		if !ok {
			t = turn(sheet)
			turned[sheet] = t
		}
		if sheet == img {
			return t
		}
		return cut(t, img.Bounds())
	}
	c := *a
	c.groundPattern, c.grassTints = nil, nil
	c.Tiles = make(map[int]Sprite, len(a.Tiles))
	for id, s := range a.Tiles {
		s.Image = recut(s.Image)
		c.Tiles[id] = s
	}
	if a.Scenery != nil {
		sc := *a.Scenery
		sc.sprites = make(map[string]Sprite, len(a.Scenery.sprites))
		for name, s := range a.Scenery.sprites {
			s.Image = recut(s.Image)
			sc.sprites[name] = s
		}
		c.Scenery = &sc
	}
	if a.grassTints == nil {
		a.grassTints = map[content.GrassTint]*Assets{}
	}
	a.grassTints[*t] = &c
	return &c
}
