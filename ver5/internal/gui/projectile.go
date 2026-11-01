package gui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/sprite"
)

// ReleasePoints finds where a thrown stone leaves a siege engine in every facing: the tip of the
// throwing arm (its bucket, or a stone drawn in it) in one frame of an animation, taken as the
// centre of the topmost few rows of the sprite that are not attack effect, as an offset from the
// sprite pivot in sprite pixels (y < 0 is up). The frame should show the arm about upright.
func ReleasePoints(s *sprite.Sheet, anim string, frame int) map[string][2]float64 {
	const depth = 4 // rows below the top that count as the tip
	mark := color.RGBA{255, 0, 255, 255}
	img := s.Render(sprite.Palette{"effect": {mark, mark, mark}})
	w, h := s.Cell[0], s.Cell[1]
	col := s.Animations[anim].FirstColumn + frame
	out := map[string][2]float64{}
	for row, dir := range s.Rows {
		x0, y0 := col*w, row*h
		solid := func(x, y int) bool {
			c := img.RGBAAt(x0+x, y0+y)
			return c.A == 255 && c != mark
		}
		top := -1
		for y := 0; y < h && top < 0; y++ {
			for x := 0; x < w; x++ {
				if solid(x, y) {
					top = y
					break
				}
			}
		}
		if top < 0 {
			continue
		}
		var sx, sy, n float64
		for y := top; y < min(h, top+depth); y++ {
			for x := 0; x < w; x++ {
				if solid(x, y) {
					sx, sy, n = sx+float64(x)+.5, sy+float64(y)+.5, n+1
				}
			}
		}
		out[dir] = [2]float64{sx/n - float64(s.Pivot[0]), sy/n - float64(s.Pivot[1])}
	}
	return out
}

const (
	lobApexBase    = 18.0 // canvas px the arc climbs over its chord at the shortest range...
	lobApexPerCell = 13.0 // ...and this much more for every cell of distance
	lobTimeBase    = .38  // flight seconds at zero range...
	lobTimePerCell = .1   // ...plus this per cell
	lobHang        = .3   // 0: constant ground speed; up to 1: slows down over the apex
	lobFrameTime   = 1.0 / 14
)

// Lob is a stone thrown along a parabola. It flies over the ground from (GX, GY), where it
// leaves Rise canvas pixels above the ground, to (TX, TY) on the ground; positions are field
// canvas pixels, like Field.Center. The arc's height and the flight time grow with distance.
// Its ground shadow moves under it, and its sheet's "<Kind>-dust" or "dust" frames play where it lands.
type Lob struct {
	Kind         string // projectile sheet name
	GX, GY, Rise float64
	TX, TY       float64
	Apex, Dur, T float64
	OnLand       func() // called once when the stone hits the ground (start the hit there)
	spinDir      int
	landed       bool
	maxH         float64
}

// isoCells is the length in cells of a canvas-pixel offset on the ground (cells step (±32, 16)).
func isoCells(dx, dy float64) float64 {
	a, b := dx/(16*K), dy/(8*K)
	return math.Hypot((a+b)/2, (b-a)/2)
}

func NewLob(kind string, gx, gy, rise, tx, ty float64) *Lob {
	d := isoCells(tx-gx, ty-gy)
	l := &Lob{Kind: kind, GX: gx, GY: gy, Rise: rise, TX: tx, TY: ty,
		Apex: lobApexBase + lobApexPerCell*d, Dur: lobTimeBase + lobTimePerCell*d, spinDir: 1}
	if tx < gx {
		l.spinDir = -1 // tumbles forward: clockwise when flying right
	}
	for i := 0; i <= 32; i++ {
		_, _, h := l.at(float64(i) / 32)
		l.maxH = math.Max(l.maxH, h)
	}
	return l
}

// at is the ground point and the height above it at flight time fraction k.
func (l *Lob) at(k float64) (x, y, h float64) {
	s := k + lobHang*math.Sin(2*math.Pi*k)/(2*math.Pi)
	return l.GX + (l.TX-l.GX)*s, l.GY + (l.TY-l.GY)*s, l.Rise*(1-s) + 4*l.Apex*s*(1-s)
}

// Pos is where the stone is drawn now and its height above its shadow (canvas pixels).
func (l *Lob) Pos() (x, y, h float64) {
	x, y, h = l.at(math.Min(1, l.T/l.Dur))
	return x, y - h, h
}

func (l *Lob) Landed() bool { return l.landed }

// Done is true once the landing dust has played, given the sheet it is drawn from.
func (l *Lob) Done(p Shots) bool { return l.T >= l.Dur+p.dust(l.Kind).length() }

func (p Shots) dust(kind string) Shot {
	if s, ok := p[kind+"-dust"]; ok {
		return s
	}
	return p["dust"]
}

func (l *Lob) Update(dt float64) {
	l.T += dt
	if l.landed || l.T < l.Dur {
		return
	}
	l.landed = true
	if l.OnLand != nil {
		l.OnLand()
	}
}

// Draw paints the shadow and the stone, then the landing dust. toScreen maps field canvas
// pixels to logical screen pixels (Game.toScreen) and zoom is the canvas scale.
func (l *Lob) Draw(dst *ebiten.Image, p Shots, toScreen func(x, y float64) (float64, float64), zoom float64) {
	if l.landed {
		d := p.dust(l.Kind)
		if i := d.frameAt(l.T - l.Dur); i >= 0 {
			d.draw(dst, i, l.TX, l.TY, toScreen, zoom)
		}
		return
	}
	gx, gy, h := l.at(math.Min(1, l.T/l.Dur))
	k := h / math.Max(1, l.maxH)
	if sh := p["shadow"]; len(sh.Frames) > 0 {
		sh.drawAlpha(dst, min(len(sh.Frames)-1, int(k*float64(len(sh.Frames)))), gx, gy, toScreen, zoom, .6-.25*k)
	}
	stone := p[l.Kind]
	if n := len(stone.Frames); n > 0 {
		i := int(l.T/lobFrameTime) * l.spinDir
		stone.draw(dst, (i%n+n)%n, gx, gy-h, toScreen, zoom)
	}
}

// NewDust is a landing dust cloud alone, at canvas point (x, y), as if a stone of kind had
// just hit there.
func NewDust(kind string, x, y float64) *Lob {
	return &Lob{Kind: kind, TX: x, TY: y, landed: true}
}

// releaseFrame is the attack frame in which a thrower's stone leaves its bucket.
const releaseFrame = 2

// release is where a unit's thrown stone leaves its sprite (offset from the pivot) in its facing.
func (g *Game) release(u *unitVis) [2]float64 {
	if g.releases == nil {
		g.releases = map[string]map[string][2]float64{}
	}
	points, ok := g.releases[u.art.ID]
	if !ok {
		if s := g.assets.Lib.Sheet(u.art.ID); s != nil {
			points = ReleasePoints(s, "attack", releaseFrame)
		}
		g.releases[u.art.ID] = points
	}
	if p, ok := points[u.dir]; ok {
		return p
	}
	return [2]float64{0, -u.top}
}

// lobAt is the stone a thrower at (u, v) facing dir sends to the target cell's ground.
func (g *Game) lobAt(thrower *unitVis, dir string, u, v float64, target *unitVis) *Lob {
	keep := thrower.dir
	thrower.dir = dir
	r := g.release(thrower)
	thrower.dir = keep
	fx, fy := g.field.Center(u, v)
	tx, ty := g.field.Center(target.u, target.v)
	return NewLob("stone", fx+r[0], fy-thrower.lift, -r[1], tx, ty-target.lift)
}

// lobFlight is how long a stone from thrower to target will fly, judged before the throw
// from where both stand when the action ends (the thrower may still walk there first).
func (g *Game) lobFlight(thrower, target *unitVis) float64 {
	u, v := thrower.u, thrower.v
	for _, w := range g.player.final.UnitViews {
		if w.ID == thrower.id {
			u, v = float64(w.X), float64(w.Y)
		}
	}
	dir := thrower.dir
	if u != target.u || v != target.v {
		dir = facing(target.u-u, target.v-v)
	}
	return g.lobAt(thrower, dir, u, v, target).Dur
}

// throw starts a stone's flight from the thrower's bucket; it lands after flight seconds,
// when the blow's beat shows the hit.
func (g *Game) throw(thrower, target *unitVis, flight float64) {
	l := g.lobAt(thrower, thrower.dir, thrower.u, thrower.v, target)
	l.Dur = flight
	g.lobs = append(g.lobs, l)
}

// dust raises a stone's landing dust under a unit the blow reaches beside its landing cell.
func (g *Game) dust(u *unitVis) { g.dustAt(u.u, u.v, u.lift) }

// dustAt raises the landing dust on the ground at the (fractional) cell (u, v) lifted by lift.
func (g *Game) dustAt(u, v, lift float64) {
	x, y := g.field.Center(u, v)
	g.lobs = append(g.lobs, NewDust("stone", x, y-lift))
}
