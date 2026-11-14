package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/scenemap"
)

// Scenery is the soft ground decoration and the pine art that make battle maps vary (graphics/map/scenery.png,
// drawn by assets_candidate/tools/gen_scenery.py). Ground tiles stay as they are; decoration is stamped on
// top, at places derived from the map and cell coordinates, so nothing is stored in the map files. Trees keep
// the ink outline of everything that matters in a battle; decoration has none.
type Scenery struct {
	sprites map[string]Sprite
	anim    map[string]FrameStrip // looping strips: the water's glints and ripples, flags, smoke, pine sway
	sites   map[string]siteLife   // by strongpoint sprite name (camp, village, castle)
	pines   []pineArt
}

// siteLife brings a strongpoint to life: looping overlays drawn on it (the flag, chimney smoke, a pennant). at is
// where an overlay's pivot lies from the strongpoint's own pivot.
type siteLife struct {
	overlays []siteOverlay
}

type siteOverlay struct {
	strip  FrameStrip
	dx, dy float64
	phase  int
	chance float64 // share of this kind of strongpoint that has the overlay (0 means all)
}

// pineArt is one pine with its sway strip. weight is how often it is chosen inside a forest and at its edge.
type pineArt struct {
	sway   FrameStrip
	shadow Sprite
	weight [2]float64
}

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
		Anim map[string]struct {
			Frames []string
			MS     int
		}
		Sites map[string]struct {
			Overlays []struct {
				Anim   string
				At     [2]float64
				Phase  int
				Chance float64
			}
		}
		Pines []struct {
			Anim   string
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
	s := &Scenery{sprites: map[string]Sprite{}, anim: map[string]FrameStrip{}}
	for name, v := range meta.Sprites {
		r := image.Rect(v.Rect[0], v.Rect[1], v.Rect[0]+v.Rect[2], v.Rect[1]+v.Rect[3])
		s.sprites[name] = Sprite{sheet.SubImage(r).(*ebiten.Image), v.Pivot[0], v.Pivot[1]}
	}
	for kind, a := range meta.Anim {
		if a.MS <= 0 || len(a.Frames) == 0 {
			return nil, fmt.Errorf("scenery.json: anim %s needs frames and ms", kind)
		}
		strip := FrameStrip{MS: a.MS}
		for _, name := range a.Frames {
			strip.Frames = append(strip.Frames, s.sprites[name])
		}
		s.anim[kind] = strip
	}
	s.sites = map[string]siteLife{}
	for name, v := range meta.Sites {
		var life siteLife
		for _, o := range v.Overlays {
			life.overlays = append(life.overlays, siteOverlay{s.anim[o.Anim], o.At[0], o.At[1], o.Phase, o.Chance})
		}
		s.sites[name] = life
	}
	for _, p := range meta.Pines {
		sway, ok := s.anim[p.Anim]
		if !ok {
			return nil, fmt.Errorf("scenery.json: pine anim %q is missing", p.Anim)
		}
		s.pines = append(s.pines, pineArt{sway: sway, shadow: s.sprites[p.Shadow], weight: p.Weight})
	}
	return s, nil
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

// makeScenery places the ground decoration (scenemap.Scenery.Decorate), drawn at each frame over the ground and
// floors; the cliff tops are kept apart and drawn with their blocks. Tree shadows are collected while the trees
// are placed (nativeForest).
func (f *Field) makeScenery() {
	floored := map[[2]int]bool{} // an old room map's boards, mats and carpets lie over plain ground: no grass under them
	for _, p := range f.Map.Props {
		if strings.HasPrefix(p.Name, "floor_") && !p.Fill {
			u, v := int(math.Round(p.U*2)), int(math.Round(p.V*2))
			for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				floored[[2]int{u + d[0], v + d[1]}] = true
			}
		}
	}
	glyph := func(x, y int) byte {
		if floored[[2]int{x, y}] {
			return 0
		}
		return f.small(x, y)
	}
	stamps, ripples := f.A.SceneRules.Scenery.Decorate(f.Map.ID, glyph, 0, 0, 2*f.Map.W, 2*f.Map.H, func(x, y int) (float64, float64) {
		mx, my := f.mapCenter((float64(x)+.5)/2-.5, (float64(y)+.5)/2-.5)
		return mx * K, my * K
	}, float64(f.A.CliffRise))
	if f.Map.SunkWater { // lowered with the floor they lie on (docs/plan-tile-floors.md)
		stamps = slices.DeleteFunc(stamps, func(st scenemap.Stamp) bool { return !f.lowerOnFloor(&st.X, &st.Y) })
		ripples = slices.DeleteFunc(ripples, func(r scenemap.Ripple) bool { return !f.lowerOnFloor(&r.X, &r.Y) })
	}
	f.waterFX = ripples
	for _, st := range stamps {
		p := placeAt(f.A.Scenery.sprites[st.Name], st.X, st.Y)
		if st.Top {
			f.cliffTops = append(f.cliffTops, p)
		} else {
			f.decor = append(f.decor, p)
		}
	}
}

// lowerOnFloor moves canvas point (x, y), placed at ground height, down by the depth of the battle map small tile
// under it, and reports whether it still shows there: false where a nearer, higher floor covers the lowered point.
func (f *Field) lowerOnFloor(x, y *float64) bool {
	at := func(x, y float64) (int, int) { // the small tile whose ground diamond holds canvas point (x, y)
		a, b := (x/K-f.OriginX)/8, (y/K-f.OriginY-4)/4 // x - y and x + y in small tiles
		return int(math.Floor((a+b)/2 + .5)), int(math.Floor((b-a)/2 + .5))
	}
	sx, sy := at(*x, *y)
	d := depthOf(f.small(sx, sy))
	*y += float64(d)
	for _, dd := range []int{0, fordDepth} {
		if dd >= d {
			break
		}
		if nx, ny := at(*x, *y-float64(dd)); depthOf(f.small(nx, ny)) == dd && nx+ny > sx+sy {
			return false
		}
	}
	return true
}

// drawWaterFX draws the looping twinkles and ripples that are in view over the water.
func (f *Field) drawWaterFX() {
	for _, e := range f.waterFX {
		if !f.DrawBounds.Empty() && !image.Pt(int(e.X), int(e.Y)).In(f.DrawBounds) {
			continue
		}
		strip := f.A.Scenery.anim[e.Kind]
		f.blit(f.Canvas, strip.Frames[(int((f.clockMS+e.Lag)/float64(strip.MS))+e.Phase)%len(strip.Frames)], e.X, e.Y, nil)
	}
}

// drawCliffTop draws the decoration of the cliff block whose ground cell centre is (cx, cy), on top of that block:
// the part of each cliff top stamp inside the block's box.
func (f *Field) drawCliffTop(cx, cy float64) {
	if len(f.cliffTops) == 0 {
		return
	}
	top := cy - float64(f.A.CliffRise)
	r := image.Rect(int(cx)-16, int(top)-17, int(cx)+16, int(top)+9)
	ps, ok := f.cliffTopsAt[r]
	if !ok {
		for _, p := range f.cliffTops {
			if part := p.rect.Intersect(r); !part.Empty() {
				ps = append(ps, placed{img: p.img.SubImage(part.Sub(p.rect.Min).Add(p.img.Bounds().Min)).(*ebiten.Image), rect: part})
			}
		}
		if f.cliffTopsAt == nil {
			f.cliffTopsAt = map[image.Rectangle][]placed{}
		}
		f.cliffTopsAt[r] = ps
	}
	for _, p := range ps {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(p.rect.Min.X), float64(p.rect.Min.Y))
		f.shadeScenery(f.Canvas, op, cx, cy)
		f.renderOp(f.Canvas, op)
		f.Canvas.DrawImage(p.img, op)
	}
}

// nativeForest fills every forest cell with several pines of different heights, tones and positions instead of one
// repeated cluster. Each pine is its own depth-sorted item, so a unit among the trees has some of them behind it
// and some in front, and sways on its own phase and lag so the wind seems to pass through. A burnt cell shows the
// burnt forest picture instead of its pines.
func (f *Field) nativeForest(items *[]drawItem) {
	sc := f.A.Scenery
	seed := scenemap.NewSeed(f.Map.ID)
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
			count := 4 + int(math.Round(3*(1-edge))) + int(seed.Hash(x, y, 1)%2)
			slots := [9][2]float64{}
			for i := range slots {
				slots[i] = [2]float64{float64(i%3-1) * .55, float64(i/3-1) * .55}
			}
			for i := len(slots) - 1; i > 0; i-- {
				j := int(seed.Hash(x, y, 10+i) % uint32(i+1))
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
				a := slots[i][0] + (seed.Unit(x, y, 20+i)-.5)*.4
				b := slots[i][1] + (seed.Unit(x, y, 40+i)-.5)*.4
				px, py := cx+(a-b)*16, cy+(a+b)*8
				weights := make([]float64, len(sc.pines))
				for k, p := range sc.pines {
					weights[k] = p.weight[0]*(1-edge) + p.weight[1]*edge
				}
				pine := sc.pines[pick(weights, seed.Unit(x, y, 60+i))]
				phase := int(math.Floor(px*.012 + py*.006))
				sway := pine.sway
				lag := float64(seed.Hash(x, y, 80+i) % uint32(sway.MS)) // so the trees do not all change frame at the same instant
				s0 := sway.Frames[0]
				b0 := s0.Image.Bounds()
				rx, ry := int(math.Round(px-s0.PivotX)), int(math.Round(py-s0.PivotY))
				*items = append(*items, drawItem{y: py, rect: image.Rect(rx, ry, rx+b0.Dx(), ry+b0.Dy()), draw: func() {
					if f.look(at.X, at.Y) == "soot" {
						return
					}
					f.blit(f.Canvas, sway.Frames[(int((f.clockMS+lag)/float64(sway.MS))+phase)%len(sway.Frames)], px, py, nil)
				}})
				f.treeShadows = append(f.treeShadows, placeAt(pine.shadow, px, py))
			}
		}
	}
}

// placed is a picture put down on the canvas, its corner on a whole pixel.
type placed struct {
	img  *ebiten.Image
	rect image.Rectangle // canvas pixels
}

// placeAt puts s with its pivot at canvas point (x, y).
func placeAt(s Sprite, x, y float64) placed {
	at := image.Pt(int(math.Round(x-s.PivotX)), int(math.Round(y-s.PivotY)))
	return placed{img: s.Image, rect: image.Rectangle{Min: at, Max: at.Add(s.Image.Bounds().Size())}}
}

// drawPlaced draws the pictures of ps that are in view over dst.
func (f *Field) drawPlaced(dst *ebiten.Image, ps []placed, blend ebiten.Blend) {
	for _, p := range ps {
		if !f.DrawBounds.Empty() && !p.rect.Overlaps(f.DrawBounds) {
			continue
		}
		op := &ebiten.DrawImageOptions{Blend: blend}
		op.GeoM.Translate(float64(p.rect.Min.X), float64(p.rect.Min.Y))
		f.renderOp(dst, op)
		dst.DrawImage(p.img, op)
	}
}

// drawDecor draws the ground decoration, then the tree shadows over it: those meet on the shadow layer first
// (taking the darker where they overlap), which is free until the units' shadows are cast later in the frame.
func (f *Field) drawDecor(c *ebiten.Image) {
	f.drawPlaced(c, f.decor, ebiten.BlendSourceOver)
	if len(f.treeShadows) == 0 {
		return
	}
	f.shadows.Clear()
	f.drawPlaced(f.shadows, f.treeShadows, maxBlend)
	c.DrawImage(f.shadows, nil)
}

// drawSite draws a strongpoint with its life: the still picture and the looping overlays. u and v are its cell,
// which decides where it smokes and how its phase lags.
func (f *Field) drawSite(still Sprite, life siteLife, x, y float64, u, v int) {
	f.blit(f.Canvas, still, x, y, nil)
	seed := scenemap.NewSeed(f.Map.ID)
	// the same wind as the trees: the phase moves across the map, with a little spread
	wave := math.Floor(x*.012+y*.006) + float64(seed.Hash(u, v, 9)%120)/60
	for i, o := range life.overlays {
		if o.chance > 0 && seed.Unit(u, v, 11+i) >= o.chance {
			continue
		}
		frames := o.strip.Frames
		f.blit(f.Canvas, frames[(int(f.clockMS/float64(o.strip.MS)+wave)+o.phase)%len(frames)], x+o.dx, y+o.dy, nil)
	}
}
