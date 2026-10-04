package gui

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
)

const margin = 48 // map pixels around the map bounds (sprites reach past the ground)

// K is how many canvas pixels one map pixel covers. The map is drawn K times larger
// than its tiles while units keep their native pixels, so units stand at 1/K of the
// size they had when everything shared one scale, without losing any detail.
const K = 2

// Field draws one battle map. Map pixels: cell (u, v) has its centre at
// ((u - v) * 16, (u + v + 1) * 8) relative to Origin; the canvas is K times that.
type Field struct {
	A       *Assets
	Map     *MapArt
	OriginX float64
	OriginY float64
	ground  [8]*ebiten.Image // per water frame; castle floor baked in
	clip    *ebiten.Image    // opaque where there is ground
	shadows *ebiten.Image
	Canvas  *ebiten.Image
	diamond *ebiten.Image
}

var maxBlend = ebiten.Blend{
	BlendFactorSourceRGB: ebiten.BlendFactorOne, BlendFactorSourceAlpha: ebiten.BlendFactorOne,
	BlendFactorDestinationRGB: ebiten.BlendFactorOne, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB: ebiten.BlendOperationMax, BlendOperationAlpha: ebiten.BlendOperationMax,
}

func NewField(a *Assets, m *MapArt) *Field {
	w, h := (m.Bounds.Dx()+2*margin)*K, (m.Bounds.Dy()+2*margin)*K
	f := &Field{A: a, Map: m, OriginX: float64(margin - m.Bounds.Min.X), OriginY: float64(margin - m.Bounds.Min.Y),
		clip: ebiten.NewImage(w, h), shadows: ebiten.NewImage(w, h), Canvas: ebiten.NewImage(w, h)}
	water := a.Animations["water"].Frames
	for i := range f.ground {
		g := ebiten.NewImage(w, h)
		for _, t := range m.Ground {
			for _, id := range t.Layers {
				if id == 0 {
					id = water[i%len(water)]
				}
				f.blitMap(g, a.Tiles[id], f.OriginX+float64(t.X), f.OriginY+float64(t.Y), nil)
			}
		}
		for v, row := range m.Tiles {
			for u, c := range row {
				if c == 'i' {
					x, y := f.mapCenter(float64(u), float64(v))
					f.blitMap(g, a.Structures["floor"], x, y, nil)
				}
			}
		}
		f.ground[i] = g
	}
	f.clip.DrawImage(f.ground[0], nil)
	f.diamond = ebiten.NewImage(32, 16)
	for y := 0; y < 16; y++ {
		half := 16 - int(math.Abs(float64(y)+.5-8)*2)
		vector.FillRect(f.diamond, float32(16-half), float32(y), float32(2*half), 1, color.White, false)
	}
	return f
}

// Center is the canvas position of a (fractional) cell's centre.
func (f *Field) Center(u, v float64) (float64, float64) {
	x, y := f.mapCenter(u, v)
	return x * K, y * K
}

func (f *Field) mapCenter(u, v float64) (float64, float64) {
	return f.OriginX + (u-v)*16, f.OriginY + (u+v+1)*8
}

// Cell maps a canvas position back to the cell under it.
func (f *Field) Cell(x, y float64) (int, int) {
	a, b := (x/K-f.OriginX)/16, (y/K-f.OriginY)/8
	return int(math.Floor((a + b) / 2)), int(math.Floor((b - a) / 2))
}

// ToMap converts canvas pixels to map pixels.
func ToMap(x, y float64) (float64, float64) { return x / K, y / K }

func (f *Field) Inside(u, v int) bool { return u >= 0 && v >= 0 && u < f.Map.W && v < f.Map.H }

func (f *Field) Tile(u, v int) byte {
	if !f.Inside(u, v) {
		return '~'
	}
	return f.Map.Tiles[v][u]
}

// Lift is how many canvas pixels a cell's surface stands above the ground (castle walls).
func (f *Field) Lift(u, v int) float64 {
	if f.Tile(u, v) == 'c' {
		return float64(f.A.Rise * K)
	}
	return 0
}

func (f *Field) blit(dst *ebiten.Image, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	op.GeoM.Translate(math.Round(x-s.PivotX), math.Round(y-s.PivotY))
	dst.DrawImage(s.Image, op)
}

// blitMap draws map art (tiles, scenery, structures) at a map-pixel position, K times larger.
func (f *Field) blitMap(dst *ebiten.Image, s Sprite, x, y float64, op *ebiten.DrawImageOptions) {
	if op == nil {
		op = &ebiten.DrawImageOptions{}
	}
	op.GeoM.Translate(math.Round(x-s.PivotX), math.Round(y-s.PivotY))
	op.GeoM.Scale(K, K)
	dst.DrawImage(s.Image, op)
}

// Highlight is a tinted diamond on a cell (move range, targets, cursor).
type Highlight struct {
	U, V  int
	Color color.RGBA
}

type drawItem struct {
	y    float64
	kind int // 0 scenery, 1 unit
	draw func()
}

// Draw renders the field and returns nothing; units come from the replay's visual state.
func (f *Field) Draw(clockMS float64, units []*unitVis, marks []Highlight) {
	c := f.Canvas
	c.Fill(color.RGBA{30, 38, 46, 255})
	water := f.A.Animations["water"]
	c.DrawImage(f.ground[int(clockMS/float64(water.MS))%len(f.ground)], nil)
	for _, m := range marks {
		x, y := f.mapCenter(float64(m.U), float64(m.V))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x-16, y-8-f.Lift(m.U, m.V)/K)
		op.GeoM.Scale(K, K)
		a := float32(m.Color.A) / 255 // Highlight colors are straight alpha
		op.ColorScale.Scale(float32(m.Color.R)/255*a, float32(m.Color.G)/255*a, float32(m.Color.B)/255*a, a)
		c.DrawImage(f.diamond, op)
	}
	f.shadows.Clear()
	items := []drawItem{}
	for _, d := range f.Map.Decorations {
		sid := f.A.TileFrame(d.Animation, clockMS+float64(d.Phase)*125)
		x, y := f.OriginX+float64(d.X), f.OriginY+float64(d.Y)
		if s, ok := f.A.TileShadow[sid]; ok {
			f.blitMap(f.shadows, s, x, y, &ebiten.DrawImageOptions{Blend: maxBlend})
		}
		s := f.A.Tiles[sid]
		items = append(items, drawItem{y * K, 0, func() { f.blitMap(c, s, x, y, nil) }})
	}
	for v, row := range f.Map.Tiles {
		for u, t := range row {
			x, y := f.mapCenter(float64(u), float64(v))
			switch t {
			case 'v':
				items = append(items, drawItem{y*K - .1, 0, func() { f.blitMap(c, f.A.Structures["village"], x, y, nil) }})
			case 'c':
				s := f.A.Structures[wallName(f, u, v)]
				items = append(items, drawItem{y * K, 0, func() { f.blitMap(c, s, x, y, nil) }})
			}
		}
	}
	for _, u := range units {
		if u.gone {
			continue
		}
		x, y := f.Center(u.u, u.v)
		x += u.shakeOffset()
		lift := u.lift
		if u.visible() {
			sh := Sprite{u.art.Shadow(u.dir, u.anim, u.frame()), u.art.ShadowPivot[0], u.art.ShadowPivot[1]}
			f.blit(f.shadows, sh, x, y-lift, &ebiten.DrawImageOptions{Blend: maxBlend})
		}
		unit := u
		items = append(items, drawItem{y, 1, func() { f.drawUnit(unit, x, y-lift) }})
	}
	// one layer so overlapping shadows do not darken each other, kept on the ground
	f.shadows.DrawImage(f.clip, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	c.DrawImage(f.shadows, nil)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].y != items[j].y {
			return items[i].y < items[j].y
		}
		return items[i].kind < items[j].kind
	})
	for _, it := range items {
		it.draw()
	}
}

// wallName picks the wall sprite whose parapets face the cell's non-wall neighbours.
func wallName(f *Field, u, v int) string {
	mask := 0
	for i, d := range []content.Point{{X: -1}, {Y: -1}, {X: 1}, {Y: 1}} { // nw, ne, se, sw
		if f.Tile(u+d.X, v+d.Y) != 'c' {
			mask |= 1 << i
		}
	}
	return "wall_" + string(rune('0'+mask/10)) + string(rune('0'+mask%10))
}

func (f *Field) drawUnit(u *unitVis, x, y float64) {
	if !u.visible() {
		return
	}
	op := &ebiten.DrawImageOptions{}
	if u.greyed {
		op.ColorScale.Scale(.62, .62, .62, 1)
	}
	if u.flash > 0 {
		k := float32(1 + u.flash*1.6)
		op.ColorScale.Scale(k, k, k, 1)
	}
	f.blit(f.Canvas, Sprite{u.art.Frame(u.faction, u.dir, u.anim, u.frame()), u.art.Pivot[0], u.art.Pivot[1]}, x, y, op)
	if u.shownHP <= 0 || u.actor {
		return
	}
	top := float32(math.Round(y - u.top - 4))
	bx := float32(math.Round(x))
	vector.FillRect(f.Canvas, bx-9, top, 18, 3, color.RGBA{20, 20, 24, 255}, false)
	k := float32(u.shownHP) / float32(max(1, u.maxHP))
	bar := color.RGBA{96, 214, 110, 255}
	if k <= .25 {
		bar = color.RGBA{226, 72, 60, 255}
	} else if k <= .5 {
		bar = color.RGBA{236, 184, 64, 255}
	}
	vector.FillRect(f.Canvas, bx-8, top+1, float32(math.Max(1, math.Round(float64(16*k)))), 1, bar, false)
	team := color.RGBA{86, 160, 236, 255}
	if !u.ally {
		team = color.RGBA{232, 84, 70, 255}
	}
	vector.FillRect(f.Canvas, bx-11, top, 2, 3, team, false)
}

// Bounds of the canvas, for the camera.
func (f *Field) Size() image.Point { return f.Canvas.Bounds().Size() }
