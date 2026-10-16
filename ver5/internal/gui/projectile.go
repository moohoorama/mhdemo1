package gui

import (
	"encoding/json"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/sprite"
)

// Projectiles is the hand-drawn projectile sheet, assets/graphics/effects/projectiles.{png,json}
// (built by tools/build_projectiles.py): the flying stone and its landing dust, at the
// characters' pixel scale.
type Projectiles map[string]Shot

// Shot is one projectile sprite: its frames, the anchor (sprite pixels) that lands on the
// drawing position, and per-frame milliseconds for a one-shot animation.
type Shot struct {
	Anchor [2]float64
	Frames []*ebiten.Image
	MS     []int
}

func LoadProjectiles(dir string) (Projectiles, error) {
	b, err := os.ReadFile(filepath.Join(dir, "effects", "projectiles.json"))
	if err != nil {
		return nil, err
	}
	var index map[string]struct {
		Anchor [2]float64
		Frames [][4]int
		MS     []int
	}
	if err := json.Unmarshal(b, &index); err != nil {
		return nil, err
	}
	pix, err := loadRGBA(filepath.Join(dir, "effects", "projectiles.png"))
	if err != nil {
		return nil, err
	}
	sheet := ebiten.NewImageFromImage(pix)
	p := Projectiles{}
	for name, s := range index {
		shot := Shot{Anchor: s.Anchor, MS: s.MS}
		for _, r := range s.Frames {
			shot.Frames = append(shot.Frames, sheet.SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])).(*ebiten.Image))
		}
		p[name] = shot
	}
	return p, nil
}

// frameAt is the frame of a one-shot animation t seconds in, or -1 once it has finished.
func (s Shot) frameAt(t float64) int {
	ms := t * 1000
	for i, d := range s.MS {
		if ms < float64(d) {
			return i
		}
		ms -= float64(d)
	}
	return -1
}

// length is a one-shot animation's duration in seconds.
func (s Shot) length() float64 {
	total := 0
	for _, d := range s.MS {
		total += d
	}
	return float64(total) / 1000
}

// draw blits frame i with its anchor at canvas point (x, y), snapped to the canvas pixel grid
// so the dots stay crisp at any zoom.
func (s Shot) draw(dst *ebiten.Image, i int, x, y float64, toScreen func(x, y float64) (float64, float64), zoom float64) {
	px, py := toScreen(math.Round(x-s.Anchor[0]), math.Round(y-s.Anchor[1]))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(math.Round(px), math.Round(py))
	drawScaled(dst, s.Frames[i], op)
}

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
func (l *Lob) Done(p Projectiles) bool { return l.T >= l.Dur+p.dust(l.Kind).length() }

func (p Projectiles) dust(kind string) Shot {
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
func (l *Lob) Draw(dst *ebiten.Image, p Projectiles, toScreen func(x, y float64) (float64, float64), zoom float64) {
	if l.landed {
		d := p.dust(l.Kind)
		if i := d.frameAt(l.T - l.Dur); i >= 0 {
			d.draw(dst, i, l.TX, l.TY, toScreen, zoom)
		}
		return
	}
	gx, gy, h := l.at(math.Min(1, l.T/l.Dur))
	k := h / math.Max(1, l.maxH)
	sx, sy := toScreen(gx, gy)
	rx := (5 - 2*k) * zoom
	shadowEllipse(dst, float32(sx), float32(sy), float32(rx), float32(rx*.5), color.NRGBA{20, 16, 12, 255}, .6-.25*k)
	stone := p[l.Kind]
	if n := len(stone.Frames); n > 0 {
		i := int(l.T/lobFrameTime) * l.spinDir
		stone.draw(dst, (i%n+n)%n, gx, gy-h, toScreen, zoom)
	}
}

func ellipsePath(p *vector.Path, x, y, rx, ry float32) {
	for i := 0; i <= 24; i++ {
		t := float64(i) / 24 * 2 * math.Pi
		px, py := x+rx*float32(math.Cos(t)), y+ry*float32(math.Sin(t))
		if i == 0 {
			p.MoveTo(px, py)
		} else {
			p.LineTo(px, py)
		}
	}
	p.Close()
}

// shadowEllipse fills a soft flat shadow (a darker core inside a faint rim).
func shadowEllipse(dst *ebiten.Image, x, y, rx, ry float32, c color.NRGBA, a float64) {
	for i, k := range []float32{1, .6} {
		var p vector.Path
		ellipsePath(&p, x, y, rx*k, ry*k)
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(with8(c, a*[]float64{.55, .6}[i]))
		fillPath(dst, &p, op)
	}
}
