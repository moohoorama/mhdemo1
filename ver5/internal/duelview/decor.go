package duelview

import "github.com/hajimehoshi/ebiten/v2"

// Sprite is an image whose pivot lands on the drawing position.
type Sprite struct {
	Image          *ebiten.Image
	PivotX, PivotY float64
}

// Decor is the arena's scenery, drawn from the game's own image files: rocks and swaying
// grass scattered over the ground, flags behind each army, and dust that bursts where the
// duelists' weapons meet. A nil Decor leaves the arena bare.
type Decor struct {
	Rocks  []Sprite
	Grass  [][]Sprite // each tuft's animation frames
	Banner Sprite
	Flag   Sprite
	Dust   []Sprite
	DustMS []float64
}

const (
	decorScale = 3.0
	grassMS    = 500.0
	cell       = 150 // one scattered tuft or rock per cell at most
)

// hash is a small deterministic mix, so the scenery looks the same on every frame.
func hash(a, b, c int) uint32 {
	h := uint32(a)*73856093 ^ uint32(b)*19349663 ^ uint32(c)*83492791
	h ^= h >> 13
	h *= 0x5bd1e995
	return h ^ h>>15
}

func unit(h uint32) float64 { return float64(h%1000) / 1000 }

// ground returns the scattered rocks and grass of the cw×ch canvas.
func (g *View) ground() []unitDraw {
	d := g.Decor
	var out []unitDraw
	for cy := 0; cy*cell < g.ch; cy++ {
		for cx := 0; cx*cell < g.cw; cx++ {
			h := hash(cx, cy, 1)
			if h%100 >= 62 {
				continue
			}
			x := float64(cx*cell) + unit(h>>3)*cell
			y := float64(cy*cell) + unit(h>>9)*cell
			if h%10 < 3 && len(d.Rocks) > 0 {
				out = append(out, unitDraw{prop: &d.Rocks[int(h>>5)%len(d.Rocks)], scale: decorScale, x: x, y: y})
			} else if len(d.Grass) > 0 {
				frames := d.Grass[int(h>>5)%len(d.Grass)]
				f := (int(g.Clock*1000/grassMS) + int(h>>11)) % len(frames)
				out = append(out, unitDraw{prop: &frames[f], scale: decorScale, x: x, y: y})
			}
		}
	}
	return out
}

// flags is where each army's banners stand, as (col, rank) cells of its formation.
var flags = []struct {
	col, rank int
	command   bool
}{{-8, 1, false}, {8, 1, false}, {0, 2, true}}

func flagCell(col, rank int) bool {
	for _, f := range flags {
		if f.col == col && f.rank == rank {
			return true
		}
	}
	return false
}

// dustPuff is the frame of a dust cloud t seconds after it rose, or false once it has gone.
func (d *Decor) dustPuff(t float64) (*Sprite, bool) {
	ms := t * 1000
	for i, dur := range d.DustMS {
		if ms < dur {
			return &d.Dust[i], true
		}
		ms -= dur
	}
	return nil, false
}

func (d *Decor) dustLength() float64 {
	var ms float64
	for _, v := range d.DustMS {
		ms += v
	}
	return ms / 1000
}

// burst is the dust a clash kicks up at (x, y); k is how much of the clash is left, 1 to 0.
func (g *View) burst(x, y, k float64, critical bool) (unitDraw, bool) {
	d := g.Decor
	if len(d.Dust) == 0 || k <= 0 {
		return unitDraw{}, false
	}
	s, ok := d.dustPuff((1 - k) * d.dustLength())
	if !ok {
		return unitDraw{}, false
	}
	scale := 4.0
	if critical {
		scale = 6
	}
	return unitDraw{prop: s, scale: scale, x: x, y: y, fade: .15}, true
}

// flag is the banner standing in a formation's cell. The ally's turn green, so the two
// armies' flags differ.
func (g *View) flag(side, col, rank int, x, y float64) unitDraw {
	d := g.Decor
	s := &d.Banner
	for _, f := range flags {
		if f.col == col && f.rank == rank && f.command {
			s = &d.Flag
		}
	}
	u := unitDraw{prop: s, scale: 3, x: x, y: y}
	if side == 0 {
		u.hue = 2.1
	}
	return u
}
