package gui

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
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
	decor       []placed      // stones and groves, cut off the battle area with the rest of the outside
	tiles       *ebiten.Image // repeating quarter-tile grass with tufts, see outsideTiles
	layer       *ebiten.Image // screen-size scratch layer
	haze        *ebiten.Image // the fog under the vignette round the map's centre, see hazeTexture
	drop, rim   *ebiten.Image // the border's bands across, see bandTexture
	cloud       *ebiten.Image
	clouds      []cloudShadow
	cx, cy, rad float64 // map centre and half-diagonal in world pixels
}

// hazeReach is how far from the map's centre the haze texture reaches; beyond it the haze is even.
const hazeReach = vignetteOuter * 1.05

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
	f.dimOutside = true
	w, h := f.Canvas.Bounds().Dx(), f.Canvas.Bounds().Dy()
	corners := f.mapOutline()
	for _, c := range corners {
		o.cx += c[0] / 4
		o.cy += c[1] / 4
	}
	o.rad = math.Hypot(corners[0][0]-corners[2][0], corners[0][1]-corners[2][1]) / 2
	r := rand.New(rand.NewPCG(uint64(f.Map.W*131+f.Map.H), 7))
	put := func(name string, frame int, x, y float64) {
		sh := f.A.Outside[name]
		o.decor = append(o.decor, placeAt(Sprite{Image: sh.Frames[frame], PivotX: sh.Anchor[0], PivotY: sh.Anchor[1]}, x, y))
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
	o.haze = hazeTexture()
	o.drop, o.rim = dropBand(), rimBand()
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

// haze is the vignette laid over the fog at distance d from the map's centre: both deepen with d, so
// one picture holds the two.
func haze(d float64) color.NRGBA {
	fa, va := ramp(d, fogInner, fogOuter, fogMax), ramp(d, vignetteInner, vignetteOuter, vignetteMax)
	a := va + fa*(1-va)
	if a == 0 {
		return color.NRGBA{}
	}
	mix := func(v, f uint8) uint8 { return uint8(math.Round((float64(v)*va + float64(f)*fa*(1-va)) / a)) }
	return color.NRGBA{R: mix(vignetteColor.R, fogColor.R), G: mix(vignetteColor.G, fogColor.G), B: mix(vignetteColor.B, fogColor.B), A: uint8(math.Round(255 * a))}
}

// hazeTexture is haze round the centre of a square hazeReach from it to each side; its corners are
// already the even haze beyond.
func hazeTexture() *ebiten.Image {
	const n = 256
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			img.SetNRGBA(x, y, haze(math.Hypot(float64(x)+.5-n/2, float64(y)+.5-n/2)/(n/2)*hazeReach))
		}
	}
	return ebiten.NewImageFromImage(img)
}

// A band is the border's cross-section: a one pixel wide column of alpha, stretched across a strip
// along each side of the battle area (see drawBorder).
const bandTexels = 18

// dropBand is the dark drop just outside the edge: three soft strokes 3, 6 and 9 world px wide laid
// on each other, the narrowest darkest.
func dropBand() *ebiten.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 1, bandTexels))
	for j := range bandTexels {
		t := math.Abs((float64(j)+.5)/bandTexels-.5) * 9 // world px from the band's middle
		clear := 1.0
		for i, a := range []float64{60, 50, 40} {
			if t < float64(i+1)*1.5 {
				clear *= 1 - a/255
			}
		}
		img.SetNRGBA(0, j, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(math.Round(255 * (1 - clear)))})
	}
	return ebiten.NewImageFromImage(img)
}

// rimBand is the pale rim: solid in the middle, softening over the outer fifth on each side.
func rimBand() *ebiten.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 1, bandTexels))
	for j := range bandTexels {
		t := math.Abs((float64(j)+.5)/bandTexels-.5) * 2
		img.SetNRGBA(0, j, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(math.Round(255 * min(1, (1-t)/.4)))})
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
	if o.tiles == nil || f.renderZoom <= 0 {
		return
	}
	c := f.Canvas
	w, h := c.Bounds().Dx(), c.Bounds().Dy()
	if o.layer == nil || o.layer.Bounds().Dx() != w || o.layer.Bounds().Dy() != h {
		if o.layer != nil {
			o.layer.Deallocate()
		}
		o.layer = newLayer(w, h)
	}
	z := f.renderZoom
	// on one screen-size layer: quarter-tile grass on the grid of the map's cells, the decor, and the
	// haze over them; the battle area is cut out of it once, and it is laid on the canvas once
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
	for _, p := range o.decor {
		if !f.DrawBounds.Empty() && !p.rect.Overlaps(f.DrawBounds) {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(p.rect.Min.X)-f.renderX, float64(p.rect.Min.Y)-f.renderY)
		op.GeoM.Scale(z, z)
		o.layer.DrawImage(p.img, op)
	}
	f.drawHaze(o.layer)
	f.cutBattleArea(o.layer)
	c.DrawImage(o.layer, nil)
	// a dark drop just outside the edge, then a faintly pulsing pale rim on it, both off the tiles so the
	// edge reads as the battle area's outline on every side
	f.drawBorder(c, o.drop, 0, 9*z, color.NRGBA{A: 255})
	pulse := .35 + .08*math.Sin(clockMS/1000*1.6)
	f.drawBorder(c, o.rim, 0, math.Max(1, 1.1*z)/.8, color.NRGBA{R: 240, G: 225, B: 170, A: uint8(255 * pulse)})
}

// drawHaze lays the haze over dst as one 3×3 mesh: the texture's square round the map's centre and
// eight quads of even haze around it sharing its edge vertices, so no pixel along the seam is covered twice
// or missed. The square's outer half texel is left to the quads, so the linear filter never reads
// past the texture's edge (where it would fade to transparent).
func (f *Field) drawHaze(dst *ebiten.Image) {
	o := &f.outside
	mx, my := f.renderPoint(o.cx, o.cy)
	const n = 256
	r := hazeReach * f.renderZoom * (1 - 1.0/n) // to the outer texels' centres
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	xs := [4]float64{min(0, mx-r), mx - r, mx + r, max(w, mx+r)}
	ys := [4]float64{min(0, my-r), my - r, my + r, max(h, my+r)}
	var vs []ebiten.Vertex
	var is []uint16
	for j := range 3 {
		for i := range 3 {
			// the middle quad samples the texture between its outer texels' centres; the others sit on
			// a corner texel, all of which are the even haze
			sx := [2]float32{.5, .5}
			sy := [2]float32{.5, .5}
			if i == 1 && j == 1 {
				sx, sy = [2]float32{.5, n - .5}, [2]float32{.5, n - .5}
			}
			at := uint16(len(vs))
			for _, c := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
				vs = append(vs, ebiten.Vertex{DstX: float32(xs[i+c[0]]), DstY: float32(ys[j+c[1]]), SrcX: sx[c[0]], SrcY: sy[c[1]], ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
			}
			is = append(is, at, at+1, at+2, at, at+2, at+3)
		}
	}
	dst.DrawTriangles(vs, is, o.haze, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}

// cutBattleArea clears the battle area out of a canvas-size dst: two triangles, no path to fill.
func (f *Field) cutBattleArea(dst *ebiten.Image) {
	var vs [4]ebiten.Vertex
	for i, c := range f.mapOutline() {
		x, y := f.renderPoint(c[0], c[1])
		vs[i] = ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: 1.5, SrcY: 1.5, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	}
	dst.DrawTriangles(vs[:], []uint16{0, 1, 2, 0, 2, 3}, whitePixel(), &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendDestinationOut})
}

// drawBorder stretches band across a strip width canvas px wide along each side of the battle area,
// off canvas px out from it, tinted c. Neighbouring strips meet on the corners' bisectors, so none
// overlaps another.
func (f *Field) drawBorder(dst, band *ebiten.Image, off, width float64, c color.NRGBA) {
	var p [4][2]float64
	var mid [2]float64
	for i, q := range f.mapOutline() {
		p[i][0], p[i][1] = f.renderPoint(q[0], q[1])
		mid[0], mid[1] = mid[0]+p[i][0]/4, mid[1]+p[i][1]/4
	}
	// each side's outward normal, then each corner's offset to where lines t px out from its two sides cross
	var n [4][2]float64
	for i := range 4 {
		a, b := p[i], p[(i+1)%4]
		l := math.Hypot(b[0]-a[0], b[1]-a[1])
		n[i] = [2]float64{(b[1] - a[1]) / l, -(b[0] - a[0]) / l}
		if n[i][0]*((a[0]+b[0])/2-mid[0])+n[i][1]*((a[1]+b[1])/2-mid[1]) < 0 {
			n[i] = [2]float64{-n[i][0], -n[i][1]}
		}
	}
	var out, in [4][2]float64
	for i := range 4 {
		m, q := n[(i+3)%4], n[i]
		at := func(t float64) [2]float64 {
			k := t / (1 + m[0]*q[0] + m[1]*q[1])
			return [2]float64{p[i][0] + (m[0]+q[0])*k, p[i][1] + (m[1]+q[1])*k}
		}
		out[i], in[i] = at(off+width), at(off)
	}
	r, g, b, a := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	vertex := func(q [2]float64, v float32) ebiten.Vertex {
		return ebiten.Vertex{DstX: float32(q[0]), DstY: float32(q[1]), SrcX: .5, SrcY: v, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	}
	vs := make([]ebiten.Vertex, 0, 16)
	is := make([]uint16, 0, 24)
	for i := range 4 {
		j := (i + 1) % 4
		at := uint16(len(vs))
		vs = append(vs, vertex(out[i], 0), vertex(out[j], 0), vertex(in[j], bandTexels), vertex(in[i], bandTexels))
		is = append(is, at, at+1, at+2, at, at+2, at+3)
	}
	dst.DrawTriangles(vs, is, band, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}

// exteriorDim is how bright scenery just outside the battle area stays, like the ground there.
const exteriorDim = .55

// exteriorShade is the brightness factor for scenery standing at canvas position (x, y), its foot: the whole
// sprite takes it, so a tree on the border is not cut in two by the edge. Scenery inside the battle area is
// untouched; outside it is dimmed like the ground, darker toward the edge of the picture (the vignette's curve).
func (f *Field) exteriorShade(x, y float64) float64 {
	if !f.dimOutside || f.ShowExterior {
		return 1
	}
	if u, v := f.Cell(x, y); f.Inside(u, v) {
		return 1
	}
	o := &f.outside
	return exteriorDim * (1 - ramp(math.Hypot(x-o.cx, y-o.cy), vignetteInner, vignetteOuter, vignetteMax))
}

// edgeShade is the brightness of a unit at fractional cell (u, v) walking off the battle area: 1 inside,
// falling smoothly to the ground's exteriorDim over the first cell past the edge.
func (f *Field) edgeShade(u, v float64) float64 {
	if !f.dimOutside || f.ShowExterior {
		return 1
	}
	d := max(-.5-u, -.5-v, u-(float64(f.Map.W)-.5), v-(float64(f.Map.H)-.5))
	return 1 - (1-exteriorDim)*math.Min(1, math.Max(0, d))
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
