package gui

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Scenery is the soft ground decoration and the pine art that make battle maps vary (graphics/map/scenery.png,
// drawn by assets_candidate/tools/gen_scenery.py). Ground tiles stay as they are; decoration is stamped on
// top, at places derived from the map and cell coordinates, so nothing is stored in the map files. Trees keep
// the ink outline of everything that matters in a battle; decoration has none.
type Scenery struct {
	sprites map[string]Sprite
	groups  map[string][]string
	scatter map[string]map[string]float64 // surface -> group -> expected stamps per small tile
	anim    map[string][]Sprite           // looping effects: frames in order
	sites   map[string]siteLife           // by strongpoint sprite name (camp, village, castle)
	pines   []pineArt
}

// siteLife brings a strongpoint to life: an optional replacement for its still picture (the camp without its flag
// cloth) and looping overlays drawn on it (the flag, chimney smoke, a pennant). at is where an overlay's pivot lies
// from the strongpoint's own pivot.
type siteLife struct {
	base     *Sprite
	overlays []siteOverlay
}

type siteOverlay struct {
	frames []Sprite
	dx, dy float64
	ms     float64
	phase  int
	chance float64 // share of this kind of strongpoint that has the overlay (0 means all)
}

// pineArt is one pine with its four sway frames. weight is how often it is chosen inside a forest and at its edge.
type pineArt struct {
	frames [4]Sprite
	shadow Sprite
	weight [2]float64
}

const swayMS = 300.0 // one sway frame

func loadScenery(dir string) (*Scenery, error) {
	b, err := os.ReadFile(filepath.Join(dir, "map", "scenery.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var meta struct {
		Image   string
		Sprites map[string]struct {
			Rect  [4]int
			Pivot [2]float64
		}
		Groups  map[string][]string
		Scatter map[string]map[string]float64
		Anim    map[string][]string
		Sites   map[string]struct {
			Base     string
			Overlays []struct {
				Anim   string
				At     [2]float64
				MS     float64
				Phase  int
				Chance float64
			}
		}
		Pines []struct {
			Frames []string
			Shadow string
			Weight [2]float64
		}
	}
	if err = json.Unmarshal(b, &meta); err != nil {
		return nil, fmt.Errorf("scenery.json: %w", err)
	}
	pixels, err := loadRGBA(filepath.Join(dir, meta.Image))
	if err != nil {
		return nil, err
	}
	sheet := ebiten.NewImageFromImage(pixels)
	s := &Scenery{sprites: map[string]Sprite{}, groups: meta.Groups, scatter: meta.Scatter, anim: map[string][]Sprite{}}
	for name, v := range meta.Sprites {
		r := image.Rect(v.Rect[0], v.Rect[1], v.Rect[0]+v.Rect[2], v.Rect[1]+v.Rect[3])
		s.sprites[name] = Sprite{sheet.SubImage(r).(*ebiten.Image), v.Pivot[0], v.Pivot[1]}
	}
	for kind, names := range meta.Anim {
		for _, name := range names {
			s.anim[kind] = append(s.anim[kind], s.sprites[name])
		}
	}
	s.sites = map[string]siteLife{}
	for name, v := range meta.Sites {
		var life siteLife
		if v.Base != "" {
			base := s.sprites[v.Base]
			life.base = &base
		}
		for _, o := range v.Overlays {
			life.overlays = append(life.overlays, siteOverlay{s.anim[o.Anim], o.At[0], o.At[1], o.MS, o.Phase, o.Chance})
		}
		s.sites[name] = life
	}
	for _, p := range meta.Pines {
		if len(p.Frames) != 4 {
			return nil, fmt.Errorf("scenery.json: pine needs 4 frames, has %d", len(p.Frames))
		}
		art := pineArt{shadow: s.sprites[p.Shadow], weight: p.Weight}
		for i, name := range p.Frames {
			art.frames[i] = s.sprites[name]
		}
		s.pines = append(s.pines, art)
	}
	return s, nil
}

// sceneSeed makes the hash streams of one map differ from another's.
type sceneSeed uint32

func newSceneSeed(id string) sceneSeed {
	h := fnv.New32a()
	h.Write([]byte(id))
	return sceneSeed(h.Sum32())
}

func (s sceneSeed) hash(a, b, c int) uint32 {
	h := uint32(s) ^ uint32(a)*0x9E3779B1 ^ uint32(b)*0x85EBCA6B ^ uint32(c)*0xC2B2AE35
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}

// unit is a hash stream value in [0, 1).
func (s sceneSeed) unit(a, b, c int) float64 { return float64(s.hash(a, b, c)>>8) / (1 << 24) }

// noise is smooth value noise in [0, 1): neighbouring places get alike values, so decoration gathers in patches.
func (s sceneSeed) noise(x, y float64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := x-ix, y-iy
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	at := func(dx, dy int) float64 { return s.unit(int(ix)+dx, int(iy)+dy, 77) }
	top := at(0, 0)*(1-fx) + at(1, 0)*fx
	bottom := at(0, 1)*(1-fx) + at(1, 1)*fx
	return top*(1-fy) + bottom*fy
}

// pick chooses an index by weights from a unit value.
func pick(weights []float64, u float64) int {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	u *= total
	for i, w := range weights {
		if u < w {
			return i
		}
		u -= w
	}
	return len(weights) - 1
}

// sceneSurface names the decoration set a small tile's glyph takes. Everything that is not open ground gets none.
func sceneSurface(t byte) string {
	switch t {
	case '.':
		return "grass"
	case 'd':
		return "waste"
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

// sceneUnderlay lists the groups drawn first (big soft patches); the rest lie over them.
var sceneUnderlay = map[string]bool{"patch": true, "tone": true, "stain": true, "shade": true}

// sceneClumped lists the groups that gather in a few places instead of spreading evenly.
var sceneClumped = map[string]bool{"flower": true, "mushroom": true, "bush": true}

// sceneKeepOff lists the glyphs ground decoration stays a tile away from: water, walls, cliffs, bridges and the
// strongpoints, whose edges and sprites decoration would run into.
const sceneKeepOff = "~chbvkog"

// makeScenery bakes the ground decoration into one picture, drawn once per frame over the ground and floors; the
// cliff tops go into a second picture drawn with their blocks.
// Tree shadows are collected apart while the trees are placed and folded into it by finishScenery.
func (f *Field) makeScenery() {
	sc := f.A.Scenery
	w, h := f.Canvas.Bounds().Dx(), f.Canvas.Bounds().Dy()
	f.decor = ebiten.NewImage(w, h)
	f.treeShadow = ebiten.NewImage(w, h)
	seed := newSceneSeed(f.Map.ID)
	type stamp struct {
		s     Sprite
		x, y  float64
		layer int
		top   bool // on a cliff top: drawn with the cliff block instead of with the ground
	}
	var stamps []stamp
	add := func(name string, x, y float64, layer int, top bool) {
		stamps = append(stamps, stamp{sc.sprites[name], x, y, layer, top})
	}
	groupNames := map[string][]string{} // surface -> sorted groups, so the hash streams are stable
	for surface, groups := range sc.scatter {
		for g := range groups {
			groupNames[surface] = append(groupNames[surface], g)
		}
		slices.Sort(groupNames[surface])
	}
	floored := map[[2]int]bool{} // a story room's boards, mats and carpets lie over plain ground: no grass under them
	for _, p := range f.Map.Props {
		if strings.HasPrefix(p.Name, "floor_") && !p.Fill {
			u, v := int(math.Round(p.U*2)), int(math.Round(p.V*2))
			for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				floored[[2]int{u + d[0], v + d[1]}] = true
			}
		}
	}
	for y := range 2 * f.Map.H {
		for x := range 2 * f.Map.W {
			t := f.small(x, y)
			surface := sceneSurface(t)
			if surface == "" || floored[[2]int{x, y}] {
				continue
			}
			if surface != "forest" && surface != "cliff" && surface != "water" {
				blocked := false
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					blocked = blocked || slices.Contains([]byte(sceneKeepOff), f.small(x+d[0], y+d[1]))
				}
				if blocked {
					continue
				}
			}
			mx, my := f.mapCenter((float64(x)+.5)/2-.5, (float64(y)+.5)/2-.5)
			cx, cy := mx*K, my*K
			if surface == "water" {
				f.sceneWater(seed, x, y, cx, cy, func(name string, x, y float64, layer int) { add(name, x, y, layer, false) })
				continue
			}
			spread, top := .8, false
			if surface == "cliff" {
				spread, top = .6, true
				cy -= float64(f.A.CliffRise)
			}
			patch := seed.noise(float64(x)/5, float64(y)/5)
			for gi, g := range groupNames[surface] {
				names := sc.groups[surface+"."+g]
				if len(names) == 0 {
					continue
				}
				mod := .4 + 1.2*patch
				if sceneClumped[g] {
					mod = 3 * patch * patch
				}
				lambda := sc.scatter[surface][g] * mod
				n := int(lambda)
				if seed.unit(x, y, 100+gi) < lambda-float64(n) {
					n++
				}
				for k := range n {
					a := (seed.unit(x, y, 200+gi*8+k)*2 - 1) * spread
					b := (seed.unit(x, y, 300+gi*8+k)*2 - 1) * spread
					name := names[seed.hash(x, y, 400+gi*8+k)%uint32(len(names))]
					layer := 1
					if sceneUnderlay[g] {
						layer = 0
					}
					add(name, cx+(a-b)*8, cy+(a+b)*4, layer, top)
				}
			}
		}
	}
	slices.SortStableFunc(stamps, func(p, q stamp) int {
		if p.layer != q.layer {
			return p.layer - q.layer
		}
		return int(math.Round(p.y - q.y))
	})
	for _, st := range stamps {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(st.x-st.s.PivotX), math.Round(st.y-st.s.PivotY))
		dst := f.decor
		if st.top {
			if f.cliffTop == nil {
				f.cliffTop = ebiten.NewImage(w, h)
			}
			dst = f.cliffTop
		}
		dst.DrawImage(st.s.Image, op)
	}
}

// waterFX is a looping effect on the water (a twinkle, a ripple) at a canvas position.
type waterFX struct {
	x, y  float64
	kind  string
	phase int
	lag   float64
}

// sceneFXMS is how long one frame of each looping water effect shows.
var sceneFXMS = map[string]float64{"glint": 260, "ripple": 380}

// sceneWater decorates one water small tile: foam along the edges that meet land, reeds in the shallows there,
// lily pads and tone patches (deeper, shallower) elsewhere, and the looping twinkles and ripples. Tiles beside a
// bridge are left alone.
func (f *Field) sceneWater(seed sceneSeed, x, y int, cx, cy float64, add func(name string, x, y float64, layer int)) {
	sc := f.A.Scenery
	nb := [4]byte{f.small(x+1, y), f.small(x-1, y), f.small(x, y+1), f.small(x, y-1)} // SE, NW, SW, NE
	if slices.Contains(nb[:], 'b') {
		return
	}
	sp := sc.scatter["water"]
	patch := seed.noise(float64(x)/5, float64(y)/5)
	roll := func(lambda float64, salt int) int {
		n := int(lambda)
		if seed.unit(x, y, salt) < lambda-float64(n) {
			n++
		}
		return n
	}
	name := func(group string, salt int) string {
		names := sc.groups["water."+group]
		return names[seed.hash(x, y, salt)%uint32(len(names))]
	}
	mids := [4][2]float64{{8, 4}, {-8, -4}, {-8, 4}, {8, -4}} // where the edge to each neighbour lies from the centre
	var land []int
	for i, g := range nb {
		if g == '~' {
			continue
		}
		land = append(land, i)
		group, dir := "foam_up", [2]float64{8, -4}
		if i >= 2 {
			group, dir = "foam_down", [2]float64{8, 4}
		}
		for k, p := range [2]float64{sp["foam"], sp["foam2"]} {
			if seed.unit(x, y, 500+i*4+k) < p {
				t := (seed.unit(x, y, 520+i*4+k)*2 - 1) * .6
				add(name(group, 540+i*4+k), cx+mids[i][0]*.72+t*dir[0], cy+mids[i][1]*.72+t*dir[1], 1)
			}
		}
	}
	spot := func(salt int, spread float64) (float64, float64) {
		a := (seed.unit(x, y, salt)*2 - 1) * spread
		b := (seed.unit(x, y, salt+1)*2 - 1) * spread
		return cx + (a-b)*8, cy + (a+b)*4
	}
	if len(land) > 0 {
		for k := range roll(sp["reed"]*3*patch*patch, 600) {
			i := land[int(seed.hash(x, y, 610+k)%uint32(len(land)))]
			t := (seed.unit(x, y, 620+k)*2 - 1) * .5
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
			f.waterFX = append(f.waterFX, waterFX{px, py, kind, int(seed.hash(x, y, 740+k) % 4), float64(seed.hash(x, y, 750+k) % 400)})
		}
	}
}

// drawWaterFX draws the looping twinkles and ripples that are in view over the water.
func (f *Field) drawWaterFX() {
	for _, e := range f.waterFX {
		if !f.DrawBounds.Empty() && !image.Pt(int(e.x), int(e.y)).In(f.DrawBounds) {
			continue
		}
		frames := f.A.Scenery.anim[e.kind]
		f.blit(f.Canvas, frames[(int((f.clockMS+e.lag)/sceneFXMS[e.kind])+e.phase)%len(frames)], e.x, e.y, nil)
	}
}

// drawCliffTop draws the decoration of the cliff block whose ground cell centre is (cx, cy), on top of that block.
func (f *Field) drawCliffTop(cx, cy float64) {
	if f.cliffTop == nil {
		return
	}
	top := cy - float64(f.A.CliffRise)
	r := image.Rect(int(cx)-16, int(top)-17, int(cx)+16, int(top)+9)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	f.shadeScenery(f.Canvas, op, cx, cy)
	f.renderOp(f.Canvas, op)
	f.Canvas.DrawImage(f.cliffTop.SubImage(r).(*ebiten.Image), op)
}

// nativeForest fills every forest cell with several pines of different heights, tones and positions instead of one
// repeated cluster. Each pine is its own depth-sorted item, so a unit among the trees has some of them behind it
// and some in front, and sways on its own phase and lag so the wind seems to pass through. A burnt cell shows the
// burnt forest picture instead of its pines.
func (f *Field) nativeForest(items *[]drawItem) {
	sc := f.A.Scenery
	seed := newSceneSeed(f.Map.ID)
	v := f.Map.Visual
	forest := func(x, y int) bool {
		for j := range 2 {
			for i := range 2 {
				if f.small(2*x+i, 2*y+j) == 'f' {
					return true
				}
			}
		}
		return false
	}
	for y := v[1] / 2; y < (v[1]+v[3])/2; y++ {
		for x := v[0] / 2; x < (v[0]+v[2])/2; x++ {
			if !forest(x, y) {
				continue
			}
			near := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if (dx != 0 || dy != 0) && forest(x+dx, y+dy) {
						near++
					}
				}
			}
			edge := 1 - float64(near)/8
			count := 4 + int(math.Round(3*(1-edge))) + int(seed.hash(x, y, 1)%2)
			slots := [9][2]float64{}
			for i := range slots {
				slots[i] = [2]float64{float64(i%3-1) * .55, float64(i/3-1) * .55}
			}
			for i := len(slots) - 1; i > 0; i-- {
				j := int(seed.hash(x, y, 10+i) % uint32(i+1))
				slots[i], slots[j] = slots[j], slots[i]
			}
			cx, cy := f.Center(float64(x), float64(y))
			at := image.Pt(x, y)
			if burnt, ok := f.A.Props[fmt.Sprintf("burnt_forest_%d", ((x*7+y*3)%4+4)%4)]; ok {
				*items = append(*items, drawItem{y: cy + .1, rect: f.mapRect(burnt.Sprite, cx/K, cy/K), draw: func() {
					if f.look(at.X, at.Y) == "soot" {
						f.blit(f.Canvas, burnt.Sprite, cx, cy, nil)
					}
				}})
			}
			for i := range min(count, len(slots)) {
				a := slots[i][0] + (seed.unit(x, y, 20+i)-.5)*.4
				b := slots[i][1] + (seed.unit(x, y, 40+i)-.5)*.4
				px, py := cx+(a-b)*16, cy+(a+b)*8
				weights := make([]float64, len(sc.pines))
				for k, p := range sc.pines {
					weights[k] = p.weight[0]*(1-edge) + p.weight[1]*edge
				}
				pine := sc.pines[pick(weights, seed.unit(x, y, 60+i))]
				phase := int(math.Floor(px*.012 + py*.006))
				lag := float64(seed.hash(x, y, 80+i) % uint32(swayMS)) // so the trees do not all change frame at the same instant
				s0 := pine.frames[0]
				b0 := s0.Image.Bounds()
				rx, ry := int(math.Round(px-s0.PivotX)), int(math.Round(py-s0.PivotY))
				*items = append(*items, drawItem{y: py, rect: image.Rect(rx, ry, rx+b0.Dx(), ry+b0.Dy()), draw: func() {
					if f.look(at.X, at.Y) == "soot" {
						return
					}
					f.blit(f.Canvas, pine.frames[(int((f.clockMS+lag)/swayMS)+phase)&3], px, py, nil)
				}})
				op := &ebiten.DrawImageOptions{Blend: maxBlend}
				op.GeoM.Translate(math.Round(px-pine.shadow.PivotX), math.Round(py-pine.shadow.PivotY))
				f.treeShadow.DrawImage(pine.shadow.Image, op)
			}
		}
	}
}

// finishScenery lays the tree shadows over the baked decoration, so they cost nothing per frame.
func (f *Field) finishScenery() {
	f.decor.DrawImage(f.treeShadow, nil)
	f.treeShadow.Deallocate()
	f.treeShadow = nil
}

// drawBaked draws a world-size picture over dst, only the part in view once the camera is known.
func (f *Field) drawBaked(dst, img *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	if r := f.DrawBounds.Intersect(img.Bounds()); !f.DrawBounds.Empty() {
		if r.Empty() {
			return
		}
		img = img.SubImage(r).(*ebiten.Image)
		op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	}
	f.renderOp(dst, op)
	dst.DrawImage(img, op)
}

// drawSite draws a strongpoint with its life: the still picture (or its replacement) and the looping overlays.
// u and v are its cell, which decides where it smokes and how its phase lags.
func (f *Field) drawSite(still Sprite, life siteLife, x, y float64, u, v int) {
	if life.base != nil {
		still = *life.base
	}
	f.blit(f.Canvas, still, x, y, nil)
	seed := newSceneSeed(f.Map.ID)
	// the same wind as the trees: the phase moves across the map, with a little spread
	wave := math.Floor(x*.012+y*.006) + float64(seed.hash(u, v, 9)%120)/60
	for i, o := range life.overlays {
		if o.chance > 0 && seed.unit(u, v, 11+i) >= o.chance {
			continue
		}
		frames := o.frames
		f.blit(f.Canvas, frames[(int(f.clockMS/o.ms+wave)+o.phase)%len(frames)], x+o.dx, y+o.dy, nil)
	}
}
