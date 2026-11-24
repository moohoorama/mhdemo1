package maplight

import (
	"image"
	"math"

	"srpg/internal/content"
)

// withGrass is the atlas with a map's grass: the ground tiles and the ground decoration with their green pixels
// turned toward the climate (the game's grass shader, internal/gui/field_grass.go), made once per tint.
func (a *Atlas) withGrass(t *content.GrassTint) *Atlas {
	if t == nil || *t == (content.GrassTint{}) {
		return a
	}
	a.cache.mu.Lock()
	defer a.cache.mu.Unlock()
	if c, ok := a.cache.tinted[*t]; ok {
		return c
	}
	sat, val := t.Sat, t.Val
	if sat == 0 {
		sat = 1
	}
	if val == 0 {
		val = 1
	}
	turn := func(s Sprite) Sprite {
		s.Image = turnGreen(s.Image, t.Hue/360, sat, val)
		return s
	}
	c := *a
	c.Tiles = make(map[int]Sprite, len(a.Tiles))
	for id, s := range a.Tiles {
		c.Tiles[id] = turn(s)
	}
	c.Scenery = make(map[string]Sprite, len(a.Scenery))
	for name, s := range a.Scenery {
		c.Scenery[name] = turn(s)
	}
	a.cache.tinted[*t] = &c
	return &c
}

func smoothstep(e0, e1, x float64) float64 {
	t := math.Min(math.Max((x-e0)/(e1-e0), 0), 1)
	return t * t * (3 - 2*t)
}

func rgb2hsv(r, g, b float64) (h, s, v float64) {
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	v = mx
	s = d / (mx + 1e-10)
	switch {
	case d == 0:
		h = 0
	case mx == r:
		h = math.Mod((g-b)/d+6, 6) / 6
	case mx == g:
		h = ((b-r)/d + 2) / 6
	default:
		h = ((r-g)/d + 4) / 6
	}
	return
}

func hsv2rgb(h, s, v float64) (float64, float64, float64) {
	ch := func(n float64) float64 {
		k := math.Mod(n+h*6, 6)
		return v - v*s*math.Max(0, math.Min(math.Min(k, 4-k), 1))
	}
	return ch(5), ch(3), ch(1)
}

// turnGreen turns the hue of the green pixels (hue 50° to 170°, soft 20° edges, grey and near-black left alone).
func turnGreen(src *image.RGBA, hue, sat, val float64) *image.RGBA {
	out := image.NewRGBA(src.Bounds())
	for i := 0; i < len(src.Pix); i += 4 {
		a := float64(src.Pix[i+3])
		if a == 0 {
			continue
		}
		r, g, b := float64(src.Pix[i])/a, float64(src.Pix[i+1])/a, float64(src.Pix[i+2])/a
		h, s, v := rgb2hsv(r, g, b)
		w := smoothstep(50.0/360, 70.0/360, h) * (1 - smoothstep(150.0/360, 170.0/360, h)) * smoothstep(0.08, 0.2, s)
		tr, tg, tb := hsv2rgb(math.Mod(h+hue+1, 1), math.Min(math.Max(s*sat, 0), 1), math.Min(math.Max(v*val, 0), 1))
		put := func(o int, orig, turned float64) {
			out.Pix[i+o] = uint8(math.Round((orig + (turned-orig)*w) * a))
		}
		put(0, r, tr)
		put(1, g, tg)
		put(2, b, tb)
		out.Pix[i+3] = src.Pix[i+3]
	}
	return out
}
