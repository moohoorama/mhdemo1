package duelview

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

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

	waves map[*Sprite]*[flagFrames]Sprite
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
}{{-16, 1, false}, {-8, 1, false}, {8, 1, false}, {16, 1, false}, {0, 2, true}, {0, 5, true}}

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
	frame := (int(g.Clock*1000/flagMS) + int(hash(col, rank, side)>>4)) % flagFrames
	u := unitDraw{prop: &d.wave(s)[frame], scale: 3, x: x, y: y}
	if side == 0 {
		u.hue = 2.1
	}
	return u
}

const (
	flagFrames = 4
	flagMS     = 170.0
	flagPole   = 3 // columns of the pole, which stay put
	flagLift   = 2 // rows added on top for the cloth to rise into
)

// wave is s's flutter: flagFrames copies in which each column of the cloth, more the
// farther it is from the pole, is lifted or dropped a pixel or two along a travelling wave.
func (d *Decor) wave(s *Sprite) *[flagFrames]Sprite {
	if w, ok := d.waves[s]; ok {
		return w
	}
	if d.waves == nil {
		d.waves = map[*Sprite]*[flagFrames]Sprite{}
	}
	b := s.Image.Bounds()
	var out [flagFrames]Sprite
	for f := range out {
		im := ebiten.NewImage(b.Dx(), b.Dy()+2*flagLift)
		for x := 0; x < b.Dx(); x++ {
			dy := 0.0
			if x >= flagPole {
				amp := min(1.6, float64(x-flagPole+1)*.4)
				dy = math.Round(amp * math.Sin(2*math.Pi*float64(f)/flagFrames-float64(x)*.9))
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), flagLift+dy)
			im.DrawImage(s.Image.SubImage(image.Rect(b.Min.X+x, b.Min.Y, b.Min.X+x+1, b.Max.Y)).(*ebiten.Image), op)
		}
		out[f] = Sprite{Image: im, PivotX: s.PivotX, PivotY: s.PivotY + flagLift}
	}
	d.waves[s] = &out
	return &out
}
