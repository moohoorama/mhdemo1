package gui

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The ground outside the battle area (docs/ui-mockup, idea 1): the same grass, darker and
// strewn with tufts, stones and small groves (map/outside.png), fading into fog away from the map; a shadow
// and a pale rim along the map's edge; cloud shadows drifting over everything. Distances
// are world pixels (zoom 1).
const (
	fogInner, fogOuter = 213.0, 583.0 // fog starts at fogInner, thick by fogOuter
	fogMax             = .85
	vignetteInner      = 270.0
	vignetteOuter      = 640.0
	vignetteMax        = .55
	cloudSpan          = 1400.0 // cloud shadows wrap around this many pixels
)

var (
	fogColor      = color.NRGBA{R: 104, G: 114, B: 98, A: 255}
	vignetteColor = color.NRGBA{R: 10, G: 14, B: 8, A: 255}
)

type cloudShadow struct{ x, y, r, v float64 }

// outsideLayers holds what the outside ground needs between frames.
type outsideLayers struct {
	decor       *ebiten.Image // stones and groves in world space, empty over the map
	tiles       *ebiten.Image // repeating quarter-tile grass with tufts, see outsideTiles
	fog, layer  *ebiten.Image // screen-size scratch layers
	holeFog     *ebiten.Image // radial hole textures, see radialHole
	holeVig     *ebiten.Image
	cloud       *ebiten.Image
	clouds      []cloudShadow
	cx, cy, rad float64 // map centre and half-diagonal in world pixels
}

// mapOutline is the battle area's border in world pixels, corner by corner.
func (f *Field) mapOutline() [4][2]float64 {
	var out [4][2]float64
	for i, uv := range [][2]float64{{-.5, -.5}, {float64(f.Map.W) - .5, -.5}, {float64(f.Map.W) - .5, float64(f.Map.H) - .5}, {-.5, float64(f.Map.H) - .5}} {
		out[i][0], out[i][1] = f.Center(uv[0], uv[1])
	}
	return out
}

func (f *Field) makeOutside() {
	o := &f.outside
	w, h := f.Canvas.Bounds().Dx(), f.Canvas.Bounds().Dy()
	corners := f.mapOutline()
	for _, c := range corners {
		o.cx += c[0] / 4
		o.cy += c[1] / 4
	}
	o.rad = math.Hypot(corners[0][0]-corners[2][0], corners[0][1]-corners[2][1]) / 2
	o.decor = ebiten.NewImage(w, h)
	r := rand.New(rand.NewPCG(uint64(f.Map.W*131+f.Map.H), 7))
	put := func(name string, frame int, x, y float64) {
		sh := f.A.Outside[name]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(x-sh.Anchor[0]), math.Round(y-sh.Anchor[1]))
		o.decor.DrawImage(sh.Frames[frame], op)
	}
	area := float64(w * h)
	o.tiles = outsideTiles(r)
	for range int(area * .00012) { // stones
		put("stone", 0, r.Float64()*float64(w), r.Float64()*float64(h))
	}
	for range max(4, int(area/90000)) { // small groves of dark pines, the pine frames sized 7 to 11
		gx, gy := r.Float64()*float64(w), r.Float64()*float64(h)
		for range 9 {
			x, y, s := gx+(r.Float64()-.5)*64, gy+(r.Float64()-.5)*34, 7+r.Float64()*4
			put("pine", int(math.Round(s))-7, x, y)
		}
	}
	// nothing outside-only may show over the battle area
	var p vector.Path
	for i, c := range corners {
		if i == 0 {
			p.MoveTo(float32(c[0]), float32(c[1]))
		} else {
			p.LineTo(float32(c[0]), float32(c[1]))
		}
	}
	p.Close()
	vector.FillPath(o.decor, &p, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
	o.holeFog = radialHole(func(d float64) float64 { return ramp(d, fogInner, fogOuter, fogMax) / fogMax }, fogOuter*1.05)
	o.holeVig = radialHole(func(d float64) float64 { return ramp(d, vignetteInner, vignetteOuter, vignetteMax) / vignetteMax }, vignetteOuter*1.05)
	o.cloud = softEllipse()
	for i := range 4 {
		o.clouds = append(o.clouds, cloudShadow{x: r.Float64() * cloudSpan, y: o.cy + (r.Float64()-.5)*o.rad*1.6, r: 145 + r.Float64()*90, v: 7 + float64(i%3)*3.5})
	}
}

// outsideTiles is a seamless 16×32 block of quarter-tile diamonds (32×16 world pixels, so
// four meet in each map cell) in four dull greens, with grass tufts.
func outsideTiles(r *rand.Rand) *ebiten.Image {
	const w, h = 512, 256
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	shades := []color.NRGBA{{R: 58, G: 76, B: 44, A: 255}, {R: 63, G: 82, B: 47, A: 255}, {R: 54, G: 71, B: 42, A: 255}, {R: 66, G: 86, B: 49, A: 255}}
	for j := range 32 {
		for i := range 16 {
			c := shades[r.IntN(len(shades))]
			cx, cy := i*32+(j&1)*16, j*8
			for dy := -8; dy < 8; dy++ {
				half := 16 - int(math.Abs(float64(dy)+.5)*2)
				for dx := -half; dx < half; dx++ {
					img.SetNRGBA(((cx+dx)%w+w)%w, ((cy+dy)%h+h)%h, c)
				}
			}
		}
	}
	for range 110 {
		x, y := r.IntN(w), r.IntN(h)
		c := color.NRGBA{R: 34, G: 48, B: 26, A: 255}
		if r.Float64() < .5 {
			c = color.NRGBA{R: 42, G: 58, B: 31, A: 255}
		}
		for _, d := range [][3]int{{0, 0, 2}, {2, -1, 3}, {4, 0, 2}} {
			for k := range d[2] {
				img.SetNRGBA((x+d[0])%w, ((y+d[1]+k)%h+h)%h, c)
			}
		}
	}
	return ebiten.NewImageFromImage(img)
}

// ramp is the mockup's two-stop gradient: 0 inside lo, 35% of top at 60% of the way,
// top at hi and beyond.
func ramp(d, lo, hi, top float64) float64 {
	k := (d - lo) / (hi - lo)
	switch {
	case k <= 0:
		return 0
	case k >= 1:
		return top
	case k < .6:
		return top * .41 * k / .6
	}
	return top * (.41 + .59*(k-.6)/.4)
}

// radialHole is a texture whose alpha at distance d (texture radius = reach) is
// 1 - layer(d); drawn with BlendDestinationOut over a full layer it leaves layer(d).
func radialHole(layer func(d float64) float64, reach float64) *ebiten.Image {
	const n = 256
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			d := math.Hypot(float64(x)+.5-n/2, float64(y)+.5-n/2) / (n / 2) * reach
			a := 1 - layer(d)
			if d >= reach {
				a = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * max(0, min(1, a)))})
		}
	}
	return ebiten.NewImageFromImage(img)
}

func softEllipse() *ebiten.Image {
	const n = 128
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			d := math.Hypot(float64(x)+.5-n/2, float64(y)+.5-n/2) / (n / 2)
			img.SetNRGBA(x, y, color.NRGBA{R: 8, G: 16, B: 6, A: uint8(255 * .28 * max(0, 1-d))})
		}
	}
	return ebiten.NewImageFromImage(img)
}

// drawOutsideGround lays the decor, fog and vignette over the outside ground and outlines
// the battle area. It runs after the ground and before scenery and units.
func (f *Field) drawOutsideGround(clockMS float64) {
	o := &f.outside
	if o.decor == nil || f.renderZoom <= 0 {
		return
	}
	c := f.Canvas
	w, h := c.Bounds().Dx(), c.Bounds().Dy()
	if o.layer == nil || o.layer.Bounds().Dx() != w || o.layer.Bounds().Dy() != h {
		o.layer = ebiten.NewImage(w, h)
	}
	z := f.renderZoom
	mx, my := f.renderPoint(o.cx, o.cy)
	outline := f.renderOutline()
	// quarter-tile grass on the grid of the map's cells, outside the map only
	o.layer.Clear()
	ax, ay := f.renderPoint(f.Center(-.25, -.25))
	tw, th := 512*z, 256*z
	for y := math.Mod(ay, th) - th; y < float64(h); y += th {
		for x := math.Mod(ax, tw) - tw; x < float64(w); x += tw {
			top := &ebiten.DrawImageOptions{}
			top.GeoM.Scale(z, z)
			top.GeoM.Translate(x, y)
			o.layer.DrawImage(o.tiles, top)
		}
	}
	vector.FillPath(o.layer, outline, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
	c.DrawImage(o.layer, nil)
	op := &ebiten.DrawImageOptions{}
	f.renderOp(c, op)
	c.DrawImage(o.decor, op)
	for _, l := range []struct {
		hole  *ebiten.Image
		reach float64
		c     color.NRGBA
		top   float64
	}{{o.holeFog, fogOuter * 1.05, fogColor, fogMax}, {o.holeVig, vignetteOuter * 1.05, vignetteColor, vignetteMax}} {
		o.layer.Clear()
		lc := l.c
		lc.A = uint8(255 * l.top)
		o.layer.Fill(lc)
		hop := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationOut}
		s := l.reach * z * 2 / 256
		hop.GeoM.Scale(s, s)
		hop.GeoM.Translate(mx-l.reach*z, my-l.reach*z)
		o.layer.DrawImage(l.hole, hop)
		vector.FillPath(o.layer, outline, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
		c.DrawImage(o.layer, nil)
	}
	// a dark drop just outside the edge, then a faintly pulsing pale rim on it
	for i := 3; i >= 1; i-- {
		so := &vector.StrokeOptions{Width: float32(float64(i) * 3 * z), LineJoin: vector.LineJoinRound}
		dop := &vector.DrawPathOptions{AntiAlias: true}
		dop.ColorScale.ScaleWithColor(color.NRGBA{A: uint8(40 + 10*(3-i))})
		var shifted vector.Path
		ao := &vector.AddPathOptions{}
		ao.GeoM.Translate(0, 4*z)
		shifted.AddPath(outline, ao)
		vector.StrokePath(c, &shifted, so, dop)
	}
	pulse := .35 + .08*math.Sin(clockMS/1000*1.6)
	rop := &vector.DrawPathOptions{AntiAlias: true}
	rop.ColorScale.ScaleWithColor(color.NRGBA{R: 240, G: 225, B: 170, A: uint8(255 * pulse)})
	vector.StrokePath(c, outline, &vector.StrokeOptions{Width: float32(math.Max(1, 1.1*z))}, rop)
}

// exteriorDim is how bright scenery just outside the battle area stays, like the ground there.
const exteriorDim = .55

// exteriorShade is the brightness factor for scenery standing at canvas position (x, y), its foot: the whole
// sprite takes it, so a tree on the border is not cut in two by the edge. Scenery inside the battle area is
// untouched; outside it is dimmed like the ground, darker toward the edge of the picture (the vignette's curve).
func (f *Field) exteriorShade(x, y float64) float64 {
	if f.exterior == nil || f.ShowExterior {
		return 1
	}
	if u, v := f.Cell(x, y); f.Inside(u, v) {
		return 1
	}
	o := &f.outside
	return exteriorDim * (1 - ramp(math.Hypot(x-o.cx, y-o.cy), vignetteInner, vignetteOuter, vignetteMax))
}

// renderOutline is the battle area's border in canvas pixels.
func (f *Field) renderOutline() *vector.Path {
	var p vector.Path
	for i, c := range f.mapOutline() {
		x, y := f.renderPoint(c[0], c[1])
		if i == 0 {
			p.MoveTo(float32(x), float32(y))
		} else {
			p.LineTo(float32(x), float32(y))
		}
	}
	p.Close()
	return &p
}

// drawCloudShadows drifts soft shadows over the whole field, units included.
func (f *Field) drawCloudShadows(clockMS float64) {
	o := &f.outside
	if o.cloud == nil || f.renderZoom <= 0 {
		return
	}
	t := clockMS / 1000
	for _, s := range o.clouds {
		x := o.cx - cloudSpan/2 + math.Mod(s.x+t*s.v, cloudSpan)
		cx, cy := f.renderPoint(x, s.y)
		op := &ebiten.DrawImageOptions{}
		k := s.r * f.renderZoom * 2 / 128
		op.GeoM.Scale(k, k*.5)
		op.GeoM.Translate(cx-s.r*f.renderZoom, cy-s.r*f.renderZoom*.5)
		f.Canvas.DrawImage(o.cloud, op)
	}
}
