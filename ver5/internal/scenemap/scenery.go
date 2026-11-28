package scenemap

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"slices"
)

// Scenery is the soft ground decoration (graphics/map/scenery.png, drawn by assets_candidate/tools/gen_scenery.py):
// tufts, flowers, pebbles, cracks, foam and reeds stamped at places derived from the map and the small tile, so
// nothing is stored in the map files. Battle maps and scene maps scatter it the same way (Decorate).
type Scenery struct {
	Sprites map[string]Sprite // on the "scenery" sheet
	Groups  map[string][]string
	Scatter map[string]map[string]float64 // surface -> group -> expected stamps per small tile
}

func parseScenery(data []byte) (*Scenery, error) {
	var meta struct {
		Sprites map[string]struct {
			Rect  [4]int
			Pivot [2]int
		}
		Groups  map[string][]string
		Scatter map[string]map[string]float64
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("scenery.json: %w", err)
	}
	s := &Scenery{Sprites: map[string]Sprite{}, Groups: meta.Groups, Scatter: meta.Scatter}
	for name, v := range meta.Sprites {
		s.Sprites[name] = Sprite{Sheet: "scenery", Rect: v.Rect, Pivot: v.Pivot}
	}
	return s, nil
}

// Seed makes the hash streams of one map differ from another's.
type Seed uint32

func NewSeed(id string) Seed {
	h := fnv.New32a()
	h.Write([]byte(id))
	return Seed(h.Sum32())
}

func (s Seed) Hash(a, b, c int) uint32 {
	h := uint32(s) ^ uint32(a)*0x9E3779B1 ^ uint32(b)*0x85EBCA6B ^ uint32(c)*0xC2B2AE35
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}

// Unit is a hash stream value in [0, 1).
func (s Seed) Unit(a, b, c int) float64 { return float64(s.Hash(a, b, c)>>8) / (1 << 24) }

// Noise is smooth value noise in [0, 1): neighbouring places get alike values, so decoration gathers in patches.
func (s Seed) Noise(x, y float64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := x-ix, y-iy
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	at := func(dx, dy int) float64 { return s.Unit(int(ix)+dx, int(iy)+dy, 77) }
	top := at(0, 0)*(1-fx) + at(1, 0)*fx
	bottom := at(0, 1)*(1-fx) + at(1, 1)*fx
	return top*(1-fy) + bottom*fy
}

// surface names the decoration set a small tile's glyph takes. Everything that is not open ground gets none.
func surface(t byte) string {
	switch t {
	case '.':
		return "grass"
	case 'd':
		return "waste"
	case 's':
		return "ford"
	case 'l':
		return "plain"
	case 'i':
		return "floor"
	case 'f':
		return "forest"
	case 'h':
		return "cliff"
	case '~':
		return "water"
	}
	return ""
}

// underlay lists the groups drawn first (big soft patches); the rest lie over them.
var underlay = map[string]bool{"patch": true, "tone": true, "stain": true, "shade": true}

// clumped lists the groups that gather in a few places instead of spreading evenly.
var clumped = map[string]bool{"flower": true, "mushroom": true, "bush": true, "marsh": true}

// keepOff lists the glyphs ground decoration stays a tile away from: water, walls, cliffs, bridges and the
// strongpoints, whose edges and sprites decoration would run into.
const keepOff = "~chbvkog"

// Stamp is one decoration picture at a canvas point; Layer 0 lies under layer 1; Top stamps lie on a cliff block's
// top and are drawn with it.
type Stamp struct {
	Name  string
	X, Y  float64
	Layer int
	Top   bool
}

// Ripple is a looping effect on the water (a twinkle, a ripple) at a canvas point.
type Ripple struct {
	X, Y  float64
	Kind  string
	Phase int
	Lag   float64
}

// Decorate scatters the decoration over small tiles [x0, x1) x [y0, y1). glyph gives a small tile's glyph ('~' off
// the map), center its canvas centre; cliff-top stamps stand rise pixels up. Stamps come sorted: layer, then y.
func (s *Scenery) Decorate(id string, glyph func(x, y int) byte, x0, y0, x1, y1 int, center func(x, y int) (float64, float64),
	rise float64) ([]Stamp, []Ripple) {
	seed := NewSeed(id)
	var stamps []Stamp
	var ripples []Ripple
	groupNames := map[string][]string{} // surface -> sorted groups, so the hash streams are stable
	for surface, groups := range s.Scatter {
		for g := range groups {
			groupNames[surface] = append(groupNames[surface], g)
		}
		slices.Sort(groupNames[surface])
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			surf := surface(glyph(x, y))
			if surf == "" {
				continue
			}
			if surf != "forest" && surf != "cliff" && surf != "water" {
				blocked := false
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					blocked = blocked || slices.Contains([]byte(keepOff), glyph(x+d[0], y+d[1]))
				}
				if blocked {
					continue
				}
			}
			cx, cy := center(x, y)
			if surf == "water" {
				ripples = s.water(seed, glyph, x, y, cx, cy, &stamps, ripples)
				continue
			}
			spread, top := .8, false
			if surf == "cliff" {
				spread, top = .6, true
				cy -= rise
			}
			patch := seed.Noise(float64(x)/5, float64(y)/5)
			for gi, g := range groupNames[surf] {
				names := s.Groups[surf+"."+g]
				if len(names) == 0 {
					continue
				}
				mod := .4 + 1.2*patch
				if clumped[g] {
					mod = 3 * patch * patch
				}
				lambda := s.Scatter[surf][g] * mod
				n := int(lambda)
				if seed.Unit(x, y, 100+gi) < lambda-float64(n) {
					n++
				}
				for k := range n {
					a := (seed.Unit(x, y, 200+gi*8+k)*2 - 1) * spread
					b := (seed.Unit(x, y, 300+gi*8+k)*2 - 1) * spread
					name := names[seed.Hash(x, y, 400+gi*8+k)%uint32(len(names))]
					layer := 1
					if underlay[g] {
						layer = 0
					}
					stamps = append(stamps, Stamp{name, cx + (a-b)*8, cy + (a+b)*4, layer, top})
				}
			}
		}
	}
	slices.SortStableFunc(stamps, func(p, q Stamp) int {
		if p.Layer != q.Layer {
			return p.Layer - q.Layer
		}
		return int(math.Round(p.Y - q.Y))
	})
	return stamps, ripples
}

// water decorates one water small tile: foam along the edges that meet land, reeds in the shallows there, lily pads
// and tone patches (deeper, shallower) elsewhere, and the looping twinkles and ripples. Tiles beside a bridge are
// left alone.
func (s *Scenery) water(seed Seed, glyph func(x, y int) byte, x, y int, cx, cy float64, stamps *[]Stamp, ripples []Ripple) []Ripple {
	nb := [4]byte{glyph(x+1, y), glyph(x-1, y), glyph(x, y+1), glyph(x, y-1)} // SE, NW, SW, NE
	if slices.Contains(nb[:], 'b') {
		return ripples
	}
	add := func(name string, px, py float64, layer int) {
		*stamps = append(*stamps, Stamp{name, px, py, layer, false})
	}
	sp := s.Scatter["water"]
	patch := seed.Noise(float64(x)/5, float64(y)/5)
	roll := func(lambda float64, salt int) int {
		n := int(lambda)
		if seed.Unit(x, y, salt) < lambda-float64(n) {
			n++
		}
		return n
	}
	name := func(group string, salt int) string {
		names := s.Groups["water."+group]
		return names[seed.Hash(x, y, salt)%uint32(len(names))]
	}
	mids := [4][2]float64{{8, 4}, {-8, -4}, {-8, 4}, {8, -4}} // where the edge to each neighbour lies from the centre
	var land []int
	for i, g := range nb {
		if g == '~' || g == 'S' { // a scene map's ford (scene.json "ford") is the same river, only shallow
			continue
		}
		land = append(land, i)
		group, dir := "foam_up", [2]float64{8, -4}
		if i >= 2 {
			group, dir = "foam_down", [2]float64{8, 4}
		}
		for k, p := range [2]float64{sp["foam"], sp["foam2"]} {
			if seed.Unit(x, y, 500+i*4+k) < p {
				t := (seed.Unit(x, y, 520+i*4+k)*2 - 1) * .6
				add(name(group, 540+i*4+k), cx+mids[i][0]*.72+t*dir[0], cy+mids[i][1]*.72+t*dir[1], 1)
			}
		}
	}
	spot := func(salt int, spread float64) (float64, float64) {
		a := (seed.Unit(x, y, salt)*2 - 1) * spread
		b := (seed.Unit(x, y, salt+1)*2 - 1) * spread
		return cx + (a-b)*8, cy + (a+b)*4
	}
	if len(land) > 0 {
		for k := range roll(sp["reed"]*3*patch*patch, 600) {
			i := land[int(seed.Hash(x, y, 610+k)%uint32(len(land)))]
			t := (seed.Unit(x, y, 620+k)*2 - 1) * .5
			add(name("reed", 630+k), cx+mids[i][0]*.62+t*8, cy+mids[i][1]*.62+t*(-4+8*float64(i/2)), 1)
		}
	} else {
		for k := range roll(sp["lily"]*3*patch*patch, 650) {
			px, py := spot(660+2*k, .7)
			add(name("lily", 670+k), px, py, 1)
		}
	}
	for k := range roll(sp["tone"]*(.4+1.2*patch), 680) {
		px, py := spot(690+2*k, .8)
		add(name("tone", 700+k), px, py, 0)
	}
	for _, kind := range []string{"glint", "ripple"} {
		for k := range roll(sp[kind], 710+len(kind)) {
			px, py := spot(720+2*k+len(kind)*10, .85)
			ripples = append(ripples, Ripple{px, py, kind, int(seed.Hash(x, y, 740+k) % 4), float64(seed.Hash(x, y, 750+k) % 400)})
		}
	}
	return ripples
}
