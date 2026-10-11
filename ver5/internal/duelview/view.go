package duelview

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/duel"
	"srpg/internal/sprite"
)

const width, height = 1280, 850

const PlaybackSpeed = 2.0

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
	Time     float64
	canvas   *ebiten.Image
}

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
func point(q float64) (float64, float64) { return (duel.Actor{Position: q}).ScreenPosition() }

type unitDraw struct {
	art       *sprite.Art
	dir, anim string
	frame     int
	x, y      float64
}

func drawUnit(dst *ebiten.Image, u unitDraw, shadow bool) {
	pivot := u.art.Pivot
	im := u.art.Frame(u.dir, u.anim, u.frame)
	if shadow {
		pivot = u.art.ShadowPivot
		im = u.art.Shadow(u.dir, u.anim, u.frame)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-pivot[0], -pivot[1])
	op.GeoM.Scale(3, 3)
	op.GeoM.Translate(u.x, u.y)
	dst.DrawImage(im, op)
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
	if dst.Bounds().Dx() == width && dst.Bounds().Dy() == height {
		g.draw(dst)
		return
	}
	if g.canvas == nil {
		g.canvas = ebiten.NewImage(width, height)
	}
	g.draw(g.canvas)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(dst.Bounds().Dx())/width, float64(dst.Bounds().Dy())/height)
	dst.DrawImage(g.canvas, op)
}

func (g *View) draw(dst *ebiten.Image) {
	s := g.Story.Sample(g.Time)
	dst.Fill(color.NRGBA{21, 27, 30, 255})
	// A subdued dirt arena keeps existing pixel art and ink-wash portraits legible.
	for y := 0; y < height; y += 4 {
		k := max(0, min(1, float64(y-160)/550))
		box(dst, 0, float64(y), width, 4, color.NRGBA{uint8(74 + 19*k), uint8(74 + 12*k), uint8(63 + 8*k), 255})
	}
	for i := range 160 {
		x := float64((i * 197) % 1280)
		y := float64((i * 113) % height)
		line(dst, x, y, x+7, y+2, 1, color.NRGBA{126, 118, 91, 55})
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
				x := float64(328 + col*44 - rank*44)
				y := float64(282 - col*22 - rank*22)
				if side == 1 {
					x = width - x
					y = 880 - y
				}
				if x < -120 || x > width+120 || y < -160 || y > height+160 {
					continue
				}
				art := g.Fighters[side].Troops
				frame := art.FrameIndex("idle", g.Time*1000+float64((col+24)*63+rank*91))
				units = append(units, unitDraw{art, dir, "idle", frame, x, y})
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
		x, y := a.ScreenPosition()
		if s.ShakeSide < 0 || s.ShakeSide == i {
			x += dx
			y += dy
		}
		art := g.Fighters[i].Art
		frame := art.FrameIndex(a.Animation, a.MS)
		if a.Hold >= 0 {
			frame = min(a.Hold, art.Animations[a.Animation].Frames-1)
		}
		units = append(units, unitDraw{art, a.Facing, a.Animation, frame, x, y})
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
	for i, a := range s.Actors {
		if !a.Visible || a.Speech == "" {
			continue
		}
		x, y := a.ScreenPosition()
		y -= g.Fighters[i].Art.TopOffset() * 3
		f := &text.GoTextFace{Source: g.Font, Size: 17}
		f.SetVariation(text.MustParseTag("wght"), 400)
		w, h := text.Measure(a.Speech, f, 17*1.45)
		nameFace := &text.GoTextFace{Source: g.Font, Size: 16}
		nameFace.SetVariation(text.MustParseTag("wght"), 800)
		nameWidth, _ := text.Measure(g.Fighters[i].Name, nameFace, 16*1.45)
		w = max(80, max(w, nameWidth)+40)
		h = max(80, h+59)
		x = max(12, min(width-w-12, x-28))
		y = max(145, min(610-h, y-h-18))
		g.Bubble.Draw(dst, x, y, w, h)
		textColor := color.NRGBA{40, 37, 34, 255}
		g.labelWeight(dst, g.Fighters[i].Name, x+20, y+14, 16, textColor, 800)
		g.label(dst, a.Speech, x+20, y+41, 17, textColor)
	}
	g.portrait(dst, 0, 24, 70)
	g.portrait(dst, 1, 930, 664)
}
