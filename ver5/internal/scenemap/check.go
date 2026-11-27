package scenemap

import (
	"fmt"
	"math"
)

// Problem is something on a map the editor points out: what, and the small tile where.
type Problem struct {
	U   int    `json:"u"`
	V   int    `json:"v"`
	Msg string `json:"msg"`
}

// Check finds what the rules draw without complaint but a map should not have: a standing object on a wall line
// (in a gate's way), the same object twice at one point, and buildings and other wide objects (scene.json
// footprints at least a tile wide and deep) whose footprints overlap or that stand within two tiles of a wall in front
// of them (greater u or v, seen over the object's own picture). Small objects may stand close together and
// thin ones (a wall, a curtain) along others: those are left alone.
func Check(m *Map, c *Catalog) []Problem {
	type stand struct {
		i              int
		u0, v0, u1, v1 float64
		wide           bool
	}
	var out []Problem
	var stands []stand
	seen := map[Object]bool{}
	for i, ob := range m.Objects {
		if seen[ob] {
			out = append(out, Problem{ob.U, ob.V, fmt.Sprintf("%s가 같은 자리에 둘", ob.Name)})
		}
		seen[ob] = true
		s, ok := c.Sprites[ob.Name]
		if !ok || s.Ground || c.isGate(ob.Name) {
			continue
		}
		f := c.footprint(ob.Name)
		_, listed := c.Rules.Footprints[ob.Name]
		cu := float64(ob.U) + .5 + float64(ob.DX)/32 + float64(ob.DY)/16
		cv := float64(ob.V) + .5 - float64(ob.DX)/32 + float64(ob.DY)/16
		stands = append(stands, stand{i, cu + f[0], cv + f[1], cu + f[0] + f[2], cv + f[1] + f[3], listed && f[2] >= 1 && f[3] >= 1})
	}
	overlap := func(a0, a1, b0, b1 float64) float64 { return math.Min(a1, b1) - math.Max(a0, b0) }
	for x, a := range stands {
		ob := m.Objects[a.i]
		for _, b := range stands[x+1:] {
			if a.wide && b.wide && overlap(a.u0, a.u1, b.u0, b.u1) > 1e-6 && overlap(a.v0, a.v1, b.v0, b.v1) > 1e-6 {
				out = append(out, Problem{ob.U, ob.V, fmt.Sprintf("%s와 %s가 겹침", ob.Name, m.Objects[b.i].Name)})
			}
		}
		if a.wide {
			for _, line := range m.Walls {
				lu0, lu1 := float64(min(line.From[0], line.To[0])), float64(max(line.From[0], line.To[0])+1)
				lv0, lv1 := float64(min(line.From[1], line.To[1])), float64(max(line.From[1], line.To[1])+1)
				const near = 2
				if line.From[1] == line.To[1] && a.v0 < lv1 && lv0-a.v1 <= near && overlap(a.u0, a.u1, lu0, lu1) > 1e-6 ||
					line.From[0] == line.To[0] && a.u0 < lu1 && lu0-a.u1 <= near && overlap(a.v0, a.v1, lv0, lv1) > 1e-6 {
					out = append(out, Problem{ob.U, ob.V, fmt.Sprintf("%s가 앞쪽 성벽과 2칸 이내", ob.Name)})
					break
				}
			}
		}
		// on a wall line: the object's own small tile (where it stands) is one of the line's
		for _, line := range m.Walls {
			u, v := int(math.Floor((a.u0+a.u1)/2)), int(math.Floor((a.v0+a.v1)/2))
			if u >= min(line.From[0], line.To[0]) && u <= max(line.From[0], line.To[0]) &&
				v >= min(line.From[1], line.To[1]) && v <= max(line.From[1], line.To[1]) {
				out = append(out, Problem{ob.U, ob.V, fmt.Sprintf("%s가 성벽 줄 위에 섬", ob.Name)})
				break
			}
		}
	}
	return out
}
