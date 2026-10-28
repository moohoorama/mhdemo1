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
	gate   []bool // the small tiles of a gate's passage
}

func (w *Walk) index(u, v int) (int, bool) {
	x, y := u-w.Origin[0], v-w.Origin[1]
	return y*w.W + x, x >= 0 && y >= 0 && x < w.W && y < w.H
}

// Open reports whether small tile (u, v) can be walked.
func (w *Walk) Open(u, v int) bool {
	i, ok := w.index(u, v)
	return ok && w.open[i]
}

// Within keeps open only the small tiles reachable from (u, v) without going through a gate, and the gates'
// passages themselves: a gate becomes a way out of the walls (an explore exit), not a way to the ground beyond.
// Nothing changes when (u, v) cannot be walked.
func (w *Walk) Within(u, v int) {
	start, ok := w.index(u, v)
	if !ok || !w.open[start] || w.gate[start] {
		return
	}
	seen := make([]bool, len(w.open))
	seen[start] = true
	for queue := []int{start}; len(queue) > 0; queue = queue[1:] {
		i := queue[0]
		x, y := i%w.W, i/w.W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			j, ok := w.index(x+d[0]+w.Origin[0], y+d[1]+w.Origin[1])
			if ok && !seen[j] && w.open[j] && !w.gate[j] {
				seen[j] = true
				queue = append(queue, j)
			}
		}
	}
	for i := range w.open {
		w.open[i] = w.open[i] && (seen[i] || w.gate[i])
	}
}

// Walkable marks the small tiles a person can walk: on ground whose glyph is walkable (scene.json "walkable") at
// height 0; not under a standing object, which covers the tiles its footprint overlaps by half a tile (or half its
// own size, when smaller) along both axes; not on a wall line, except through a gate's passage (scene.json
// gate_passage: the small tiles from the gate's point along the line).
func Walkable(m *Map, c *Catalog) *Walk {
	w := &Walk{Origin: m.Origin, W: m.W(), H: m.H(), open: make([]bool, m.W()*m.H()), gate: make([]bool, m.W()*m.H())}
	set := func(u, v int, open bool) {
		if i, ok := w.index(u, v); ok {
			w.open[i] = open
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
					u, v := ob.U, ob.V+d
					if alongU {
						u, v = ob.U+d, ob.V
					}
					set(u, v, m.Step(u, v) == 0)
					if i, ok := w.index(u, v); ok {
						w.gate[i] = true
					}
				}
			}
		}
	}
	return w
}
