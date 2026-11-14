package gui

import (
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pictures cut from one sheet can be drawn in one call (batch.draw), so the sheets remember what was cut from them.
var (
	sheetsMu sync.Mutex
	sheetOf  = map[*ebiten.Image]*ebiten.Image{}
)

// cut is the r part of sheet, remembered as cut from it.
func cut(sheet *ebiten.Image, r image.Rectangle) *ebiten.Image {
	im := sheet.SubImage(r).(*ebiten.Image)
	sheetsMu.Lock()
	sheetOf[im] = sheet
	sheetsMu.Unlock()
	return im
}

// sheetFor is the sheet im was cut from, or im itself.
func sheetFor(im *ebiten.Image) *ebiten.Image {
	sheetsMu.Lock()
	defer sheetsMu.Unlock()
	if s, ok := sheetOf[im]; ok {
		return s
	}
	return im
}

// batch gathers pictures drawn over one image and sends each run cut from the same sheet in one call; it keeps
// its vertices between frames, so they are not allocated again.
type batch struct {
	vs    []ebiten.Vertex
	is    []uint32
	dst   *ebiten.Image
	sheet *ebiten.Image
	opts  ebiten.DrawTrianglesOptions
}

var noTint = [4]float32{1, 1, 1, 1}

// begin starts drawing over dst with blend.
func (b *batch) begin(dst *ebiten.Image, blend ebiten.Blend) {
	b.dst, b.sheet, b.opts = dst, nil, ebiten.DrawTrianglesOptions{Blend: blend}
	b.vs, b.is = b.vs[:0], b.is[:0]
}

// use makes the next pictures come from sheet, sending what was gathered from another.
func (b *batch) use(sheet *ebiten.Image) {
	if sheet != b.sheet {
		b.flush()
		b.sheet = sheet
	}
}

// quad lays the src part of the sheet in use on the destination box (x0, y0)–(x1, y1), in colour scale c.
func (b *batch) quad(x0, y0, x1, y1 float64, src image.Rectangle, c [4]float32) {
	n := uint32(len(b.vs))
	for _, v := range [4][4]float64{{x0, y0, float64(src.Min.X), float64(src.Min.Y)}, {x1, y0, float64(src.Max.X), float64(src.Min.Y)},
		{x0, y1, float64(src.Min.X), float64(src.Max.Y)}, {x1, y1, float64(src.Max.X), float64(src.Max.Y)}} {
		b.vs = append(b.vs, ebiten.Vertex{DstX: float32(v[0]), DstY: float32(v[1]), SrcX: float32(v[2]), SrcY: float32(v[3]), ColorR: c[0], ColorG: c[1], ColorB: c[2], ColorA: c[3]})
	}
	b.is = append(b.is, n, n+1, n+2, n+1, n+3, n+2)
}

// flush sends what was gathered.
func (b *batch) flush() {
	if len(b.is) > 0 {
		b.dst.DrawTriangles32(b.vs, b.is, b.sheet, &b.opts)
	}
	b.vs, b.is = b.vs[:0], b.is[:0]
}

// draw draws the pictures of ps that meet view over dst, each moved by geo (a scale and a shift).
func (b *batch) draw(dst *ebiten.Image, ps []placed, view image.Rectangle, geo ebiten.GeoM, blend ebiten.Blend) {
	b.begin(dst, blend)
	for _, p := range ps {
		if !view.Empty() && !p.rect.Overlaps(view) {
			continue
		}
		b.use(p.sheet)
		x0, y0 := geo.Apply(float64(p.rect.Min.X), float64(p.rect.Min.Y))
		x1, y1 := geo.Apply(float64(p.rect.Max.X), float64(p.rect.Max.Y))
		b.quad(x0, y0, x1, y1, p.img.Bounds(), noTint)
	}
	b.flush()
}
