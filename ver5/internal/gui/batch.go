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

// batch keeps the vertices of draw between frames, so they are not allocated again.
type batch struct {
	vs []ebiten.Vertex
	is []uint32
}

// draw draws the pictures of ps that meet view over dst, each moved by geo (a scale and a shift), in one call
// for each run of pictures cut from the same sheet.
func (b *batch) draw(dst *ebiten.Image, ps []placed, view image.Rectangle, geo ebiten.GeoM, blend ebiten.Blend) {
	var sheet *ebiten.Image
	flush := func() {
		if len(b.is) > 0 {
			dst.DrawTriangles32(b.vs, b.is, sheet, &ebiten.DrawTrianglesOptions{Blend: blend})
		}
		b.vs, b.is = b.vs[:0], b.is[:0]
	}
	for _, p := range ps {
		if !view.Empty() && !p.rect.Overlaps(view) {
			continue
		}
		if p.sheet != sheet {
			flush()
			sheet = p.sheet
		}
		src := p.img.Bounds()
		x0, y0 := geo.Apply(float64(p.rect.Min.X), float64(p.rect.Min.Y))
		x1, y1 := geo.Apply(float64(p.rect.Max.X), float64(p.rect.Max.Y))
		n := uint32(len(b.vs))
		for _, c := range [4][4]float64{{x0, y0, float64(src.Min.X), float64(src.Min.Y)}, {x1, y0, float64(src.Max.X), float64(src.Min.Y)},
			{x0, y1, float64(src.Min.X), float64(src.Max.Y)}, {x1, y1, float64(src.Max.X), float64(src.Max.Y)}} {
			b.vs = append(b.vs, ebiten.Vertex{DstX: float32(c[0]), DstY: float32(c[1]), SrcX: float32(c[2]), SrcY: float32(c[3]), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
		}
		b.is = append(b.is, n, n+1, n+2, n+1, n+3, n+2)
	}
	flush()
}
