package duelview

import (
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/duel"
	"srpg/internal/sprite"
)

const width, height = 1280, 850

const PlaybackSpeed = 2.0

// Params tunes the transition between battle and duel screens. Entering first scrolls
// the meeting point to the screen centre while the battle UI fades out (PanTime), then
// zooms the battle in by ZoomFactor as the duel's battlefield grows from StartScale to full
// size (ZoomTime), and last fades the duel's own UI (portraits, speech) in (UITime).
// Leaving plays the same in reverse.
type Params struct {
	PanTime, ZoomTime, UITime, ZoomFactor, StartScale float64
}

var Transition = Params{PanTime: .6, ZoomTime: .5, UITime: .4, ZoomFactor: 5, StartScale: .06}

func (p Params) Intro() float64 { return p.PanTime + p.ZoomTime + p.UITime }

var (
	ink  = color.NRGBA{239, 226, 195, 255}
	gold = color.NRGBA{219, 177, 96, 255}
	blue = color.NRGBA{91, 174, 216, 255}
	red  = color.NRGBA{221, 99, 79, 255}
)

type Fighter struct {
	Name, Class string
	Art, Troops *sprite.Art
	Portrait    *ebiten.Image
}

type View struct {
	Font     *text.GoTextFaceSource
	Bubble   *Bubble
	Fighters [4]Fighter
	Story    duel.Story
	// Axis is the direction from the ally to the enemy on screen: SE (default), SW, NE or NW.
	Axis string
	// Tint is the arena's ground colour; the zero value keeps the default dirt.
	Tint color.NRGBA
	// Staged views follow the battle transition; others (the gallery) always show everything.
	Staged bool
	// Focus is where the duelists meet, in battle canvas coordinates, for the camera.
	Focus [2]float64
	Time  float64
	// Clock is real seconds since the duel began; Time follows it between the two transitions.
	Clock   float64
	canvas  *ebiten.Image
	uiLayer *ebiten.Image
}

// Length is the real duration of the whole presentation, transitions included.
func (g *View) Length() float64 {
	return g.Story.Duration()/PlaybackSpeed + 2*Transition.Intro()
}

// Advance moves the real clock and the story clock it drives.
func (g *View) Advance(dt float64) {
	g.Clock = max(0, min(g.Length(), g.Clock+dt))
	g.Time = max(0, min(g.Story.Duration(), (g.Clock-Transition.Intro())*PlaybackSpeed))
}

// Finish jumps to the start of the closing transition.
func (g *View) Finish() {
	g.Clock = max(g.Clock, g.Length()-Transition.Intro())
	g.Time = g.Story.Duration()
}

func ramp(p float64) float64 { return smooth(max(0, min(1, p))) }

func smooth(p float64) float64 { return p * p * (3 - 2*p) }

// Phase is how far the camera has scrolled to the meeting point (pan), how far the battle
// has zoomed into the duel (zoom) and how visible the duel's own UI is (ui), all 0..1. Pan
// runs first when entering and last when leaving.
func (g *View) Phase() (pan, zoom, ui float64) {
	if !g.Staged {
		return 1, 1, 1
	}
	t := max(0, min(g.Clock, g.Length()-g.Clock))
	p := Transition
	return ramp(t / p.PanTime), ramp((t - p.PanTime) / p.ZoomTime), ramp((t - p.PanTime - p.ZoomTime) / p.UITime)
}

// Cover is 1 once the duel fills the screen and the battle need not be drawn.
func (g *View) Cover() float64 {
	_, zoom, _ := g.Phase()
	return zoom
}

// Compose draws the battle map and its UI with the duel growing out of the screen centre.
// battle and battleUI are full-screen layers the same size as dst; the camera is already
// scrolled to the meeting point.
func (g *View) Compose(dst, battle, battleUI *ebiten.Image) {
	pan, zoom, ui := g.Phase()
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	s := w / width
	if g.canvas == nil {
		g.canvas = ebiten.NewImage(width, height)
	}
	g.render(g.canvas, ui)
	k := math.Pow(max(1, Transition.ZoomFactor), zoom)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(w/2, h/2)
	dst.DrawImage(battle, op)
	if pan < 1 {
		op = &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(float32(1 - pan))
		dst.DrawImage(battleUI, op)
	}
	// Zoom is geometric so growth reads as steady rather than bursting at the end.
	k = math.Pow(Transition.StartScale, 1-zoom) * s
	op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-width/2, -height/2)
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(w/2, h/2)
	op.ColorScale.ScaleAlpha(float32(smooth(zoom)))
	dst.DrawImage(g.canvas, op)
}

func shade(c color.NRGBA, k float64) color.NRGBA {
	f := func(v uint8) uint8 { return uint8(min(255, float64(v)*k)) }
	return color.NRGBA{f(c.R), f(c.G), f(c.B), 255}
}

func lerp8(a, b uint8, k float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*k) }

func box(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}
func line(dst *ebiten.Image, x, y, x1, y1, w float64, c color.Color) {
	vector.StrokeLine(dst, float32(x), float32(y), float32(x1), float32(y1), float32(w), c, true)
}
func (g *View) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	g.labelWeight(dst, s, x, y, size, c, 400)
}

func (g *View) labelWeight(dst *ebiten.Image, s string, x, y, size float64, c color.Color, weight float32) {
	f := &text.GoTextFace{Source: g.Font, Size: size}
	f.SetVariation(text.MustParseTag("wght"), weight)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.45
	text.Draw(dst, s, f, op)
}

// at mirrors a default-axis (ally upper left, enemy lower right) arena point onto Axis.
func (g *View) at(x, y float64) (float64, float64) {
	if g.flipX() {
		x = width - x
	}
	if g.flipY() {
		y = 880 - y
	}
	return x, y
}

func (g *View) flipX() bool { return strings.HasSuffix(g.Axis, "W") }
func (g *View) flipY() bool { return strings.HasPrefix(g.Axis, "N") }

func (g *View) facing(dir string) string {
	if g.flipX() {
		dir = strings.NewReplacer("E", "W", "W", "E").Replace(dir)
	}
	if g.flipY() {
		dir = strings.NewReplacer("N", "S", "S", "N").Replace(dir)
	}
	return dir
}

func point(q float64) (float64, float64) { return (duel.Actor{Position: q}).ScreenPosition() }

type unitDraw struct {
	art       *sprite.Art
	dir, anim string
	frame     int
	x, y      float64
	fade      float64
	gray      float64
}

func drawUnit(dst *ebiten.Image, u unitDraw, shadow bool) {
	pivot := u.art.Pivot
	im := u.art.Frame(u.dir, u.anim, u.frame)
	if shadow {
		pivot = u.art.ShadowPivot
		im = u.art.Shadow(u.dir, u.anim, u.frame)
	}
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(-pivot[0], -pivot[1])
	op.GeoM.Scale(3, 3)
	op.GeoM.Translate(u.x, u.y)
	var cm colorm.ColorM
	if u.gray > 0 && !shadow {
		cm.ChangeHSV(0, 1-u.gray, 1)
	}
	cm.Scale(1, 1, 1, 1-u.fade)
	colorm.DrawImage(dst, im, cm, op)
}

func (g *View) portrait(dst *ebiten.Image, i int, x, y float64) {
	f := g.Fighters[i]
	c := blue
	if i == 1 {
		c = red
	}
	box(dst, x, y, 326, 128, color.NRGBA{27, 30, 31, 245})
	box(dst, x, y, 3, 128, c)
	if f.Portrait == nil {
		g.labelWeight(dst, f.Name, x+132, y+38, 28, ink, 800)
		g.label(dst, f.Class, x+132, y+84, 16, gold)
		return
	}
	b := f.Portrait.Bounds()
	op := &ebiten.DrawImageOptions{}
	k := 112 / float64(max(b.Dx(), b.Dy()))
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(x+8+(112-float64(b.Dx())*k)/2, y+8)
	dst.DrawImage(f.Portrait, op)
	g.label(dst, []string{"아군 장수", "적군 장수"}[i], x+132, y+12, 14, c)
	g.labelWeight(dst, f.Name, x+132, y+38, 28, ink, 800)
	g.label(dst, f.Class, x+132, y+84, 16, gold)
}

func (g *View) Draw(dst *ebiten.Image) {
	_, _, ui := g.Phase()
	if dst.Bounds().Dx() == width && dst.Bounds().Dy() == height {
		g.render(dst, ui)
		return
	}
	if g.canvas == nil {
		g.canvas = ebiten.NewImage(width, height)
	}
	g.render(g.canvas, ui)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(dst.Bounds().Dx())/width, float64(dst.Bounds().Dy())/height)
	dst.DrawImage(g.canvas, op)
}

// render draws the battlefield, then the duel's UI at the given opacity.
func (g *View) render(dst *ebiten.Image, ui float64) {
	s := g.draw(dst)
	switch {
	case ui >= 1:
		g.drawUI(dst, s)
	case ui > 0:
		if g.uiLayer == nil {
			g.uiLayer = ebiten.NewImage(width, height)
		}
		g.uiLayer.Clear()
		g.drawUI(g.uiLayer, s)
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(float32(ui))
		dst.DrawImage(g.uiLayer, op)
	}
}

func (g *View) draw(dst *ebiten.Image) duel.State {
	s := g.Story.Sample(g.Time)
	dst.Fill(color.NRGBA{21, 27, 30, 255})
	// A subdued ground keeps existing pixel art and ink-wash portraits legible.
	far, near := color.NRGBA{74, 74, 63, 255}, color.NRGBA{93, 86, 71, 255}
	speck := color.NRGBA{126, 118, 91, 55}
	if g.Tint.A != 0 {
		far, near = shade(g.Tint, .8), g.Tint
		speck = shade(g.Tint, 1.35)
		speck.A = 55
	}
	for y := 0; y < height; y += 4 {
		k := max(0, min(1, float64(y-160)/550))
		box(dst, 0, float64(y), width, 4, color.NRGBA{lerp8(far.R, near.R, k), lerp8(far.G, near.G, k), lerp8(far.B, near.B, k), 255})
	}
	for i := range 160 {
		x := float64((i * 197) % 1280)
		y := float64((i * 113) % height)
		line(dst, x, y, x+7, y+2, 1, speck)
	}
	var units []unitDraw
	for side := range 2 {
		dir := "SE"
		if side == 1 {
			dir = "NW"
		}
		for rank := range 24 {
			for col := -24; col <= 24; col++ {
				// Ranks run across the duel axis; depth extends behind the front rank.
				x := float64(168 + col*44 - rank*44)
				y := float64(202 - col*22 - rank*22)
				if side == 1 {
					x = width - x
					y = 880 - y
				}
				x, y = g.at(x, y)
				if x < -120 || x > width+120 || y < -160 || y > height+160 {
					continue
				}
				art := g.Fighters[side].Troops
				frame := art.FrameIndex("idle", g.Time*1000+float64((col+24)*63+rank*91))
				units = append(units, unitDraw{art: art, dir: g.facing(dir), anim: "idle", frame: frame, x: x, y: y})
			}
		}
	}
	shake := s.Impact * 5
	if !s.Critical {
		shake *= .35
	}
	dx := math.Sin(g.Time*160) * shake
	dy := math.Cos(g.Time*137) * shake
	for i, a := range s.Actors {
		if !a.Visible {
			continue
		}
		x, y := g.at(a.ScreenPosition())
		if s.ShakeSide < 0 || s.ShakeSide == i {
			x += dx
			y += dy
		}
		art := g.Fighters[i].Art
		frame := art.FrameIndex(a.Animation, a.MS)
		if a.Hold >= 0 {
			frame = min(a.Hold, art.Animations[a.Animation].Frames-1)
		}
		units = append(units, unitDraw{art, g.facing(a.Facing), a.Animation, frame, x, y, a.Fade, a.Gray})
	}
	sort.SliceStable(units, func(i, j int) bool { return units[i].y < units[j].y })
	for _, u := range units {
		drawUnit(dst, u, true)
	}
	for _, u := range units {
		drawUnit(dst, u, false)
	}
	if s.Impact > 0 {
		def := 1
		if s.Attacker == 1 {
			def = 0
		}
		q := (s.Actors[s.Attacker].Position + s.Actors[def].Position) / 2
		x, y := point(q)
		lane := (s.Actors[s.Attacker].Lane + s.Actors[def].Lane) / 2
		x -= lane * .45
		y += lane * .89
		x, y = g.at(x, y)
		y -= 58
		r := 36.0
		if s.Critical {
			r = 100
		}
		c := color.NRGBA{255, 219, 136, uint8(255 * s.Impact)}
		for i := range 12 {
			a := float64(i) * math.Pi / 6
			line(dst, x+math.Cos(a)*8, y+math.Sin(a)*8, x+math.Cos(a)*r, y+math.Sin(a)*r, 2+s.Impact*3, c)
		}
		vector.StrokeCircle(dst, float32(x), float32(y), float32(r*(1-s.Impact)+8), 2, c, true)
	}
	if s.Flash > 0 {
		box(dst, 0, 0, width, height, color.NRGBA{255, 255, 245, uint8(s.Flash * 225)})
	}
	return s
}

// drawUI is the duel's own interface: speech bubbles and fighter portraits.
func (g *View) drawUI(dst *ebiten.Image, s duel.State) {
	for i, a := range s.Actors {
		if !a.Visible || a.Speech == "" {
			continue
		}
		x, y := g.at(a.ScreenPosition())
		y -= g.Fighters[i].Art.TopOffset() * 3
		w, h := g.Bubble.Size(g.Fighters[i].Name, a.Speech)
		x = max(12, min(width-w-12, x-28))
		y = max(145, min(610-h, y-h))
		g.Bubble.DrawText(dst, g.Fighters[i].Name, a.Speech, x, y, 1)
	}
	allyX, allyY, enemyX, enemyY := 24.0, 70.0, 930.0, 664.0
	if g.flipX() {
		allyX, enemyX = enemyX, allyX
	}
	if g.flipY() {
		allyY, enemyY = enemyY, allyY
	}
	g.portrait(dst, 0, allyX, allyY)
	g.portrait(dst, 1, enemyX, enemyY)
}
