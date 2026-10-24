package scenemap

import (
	"math"
	"strings"
)

// Walk is which small tiles of a map a person can stand on and step through.
type Walk struct {
	Origin [2]int
	W, H   int
	open   []bool
}

// Open reports whether small tile (u, v) can be walked.
func (w *Walk) Open(u, v int) bool {
	x, y := u-w.Origin[0], v-w.Origin[1]
	return x >= 0 && y >= 0 && x < w.W && y < w.H && w.open[y*w.W+x]
}

// Walkable marks the small tiles a person can walk: on ground whose glyph is walkable (scene.json "walkable") at
// height 0; not under a standing object, which covers the tiles its footprint overlaps by half a tile (or half its
// own size, when smaller) along both axes; not on a wall line, except through a gate's passage (scene.json
// gate_passage: the small tiles from the gate's point along the line).
func Walkable(m *Map, c *Catalog) *Walk {
	w := &Walk{Origin: m.Origin, W: m.W(), H: m.H(), open: make([]bool, m.W()*m.H())}
	set := func(u, v int, open bool) {
		if x, y := u-w.Origin[0], v-w.Origin[1]; x >= 0 && y >= 0 && x < w.W && y < w.H {
			w.open[y*w.W+x] = open
		}
	}
	for y := range w.H {
		for x := range w.W {
			u, v := x+w.Origin[0], y+w.Origin[1]
			set(u, v, m.Step(u, v) == 0 && strings.IndexByte(c.Rules.Walkable, m.Glyph(u, v)) >= 0)
		}
	}
	for _, ob := range m.Objects {
		s, ok := c.Sprites[ob.Name]
		if !ok || s.Ground || c.isGate(ob.Name) {
			continue
		}
		f := c.footprint(ob.Name)
		cu := float64(ob.U) + .5 + float64(ob.DX)/32 + float64(ob.DY)/16
		cv := float64(ob.V) + .5 - float64(ob.DX)/32 + float64(ob.DY)/16
		u0, v0, u1, v1 := cu+f[0], cv+f[1], cu+f[0]+f[2], cv+f[1]+f[3]
		needU, needV := math.Min(.5, f[2]/2), math.Min(.5, f[3]/2)
		for v := int(math.Floor(v0)); v < int(math.Ceil(v1)); v++ {
			for u := int(math.Floor(u0)); u < int(math.Ceil(u1)); u++ {
				ou := math.Min(u1, float64(u+1)) - math.Max(u0, float64(u))
				ov := math.Min(v1, float64(v+1)) - math.Max(v0, float64(v))
				if ou >= needU-1e-9 && ov >= needV-1e-9 {
					set(u, v, false)
				}
			}
		}
	}
	for _, line := range m.Walls {
		alongU := line.From[0] != line.To[0]
		for t := min(line.From[0]+line.From[1], line.To[0]+line.To[1]); t <= max(line.From[0]+line.From[1], line.To[0]+line.To[1]); t++ {
			u, v := line.From[0], t-line.From[0]
			if alongU {
				u, v = t-line.From[1], line.From[1]
			}
			set(u, v, false)
		}
		for _, ob := range m.Objects {
			if !c.isGate(ob.Name) {
				continue
			}
			if alongU && ob.V == line.From[1] || !alongU && ob.U == line.From[0] {
				for d := c.Rules.GatePassage[0]; d <= c.Rules.GatePassage[1]; d++ {
					if alongU {
						set(ob.U+d, ob.V, m.Step(ob.U+d, ob.V) == 0)
					} else {
						set(ob.U, ob.V+d, m.Step(ob.U, ob.V+d) == 0)
					}
				}
			}
		}
	}
	return w
}
