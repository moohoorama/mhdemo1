package gui

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// Surge is a flood baked by tools/floodsim.py (assets/graphics/effects/flood/<name>): a height field of water
// simulated on one battle map's ground, drawn frame by frame as iso pixel art, with the dam of voxel planks and earth
// it bursts out of (the gorge's cliffs are the map's own tiles) painted the same way. Each frame is cut into bands, one per diagonal of
// cells, which the field draws among its units by depth; with them come how deep a unit standing at each spot is
// under the water and the spray thrown up where the front breaks, drawn over everything. Positions are canvas
// pixels from the map's origin, so a surge fits the map it was simulated on. It plays as two shots, the dam and
// then the place the water strikes, which lie too far apart on the map to show together.
//
// A frame's bands are kept as palette indices, deflated, and become pictures only while the frame shows: the
// flood is hundreds of frames of a few thousand dots, which as one sheet would stay in video memory the whole time.
type Surge struct {
	ms           []int
	bands        [][]surgeBand // per frame, the bands' places and sizes; img is set only for the frame showing
	blobs        [][]byte      // per frame, every band's indices (255: clear) deflated
	pal          []color.RGBA
	showing      int // the frame whose bands have pictures, or -1
	sprayBlobs   [][]byte
	sprayShowing int // the spray frame that has its picture, or -1
	scale        float64
	depth        surgeDepth
	cut          float64 // seconds in, the shot changes from the dam to where the water strikes
	spray        []surgeBand
	sprayT       float64
	sprayS       float64 // seconds a spray frame shows
}

type surgeBand struct {
	d    int // the band's cells' u + v; rock's bands are numbered far lower, so they lie under every unit
	x, y float64
	w, h int
	img  *ebiten.Image
}

// show makes pictures of frame k's bands and drops the pictures of the frame before it.
func (s *Surge) show(k int) []surgeBand {
	if s.showing != k {
		if s.showing >= 0 {
			dropPictures(s.bands[s.showing])
		}
		s.showing = -1
		if s.paint(s.blobs[k], s.bands[k]) {
			s.showing = k
		}
	}
	if s.showing != k {
		return nil
	}
	return s.bands[k]
}

// showSpray is the same for the spray frame i.
func (s *Surge) showSpray(i int) *surgeBand {
	if s.sprayShowing != i {
		if s.sprayShowing >= 0 {
			dropPictures(s.spray[s.sprayShowing : s.sprayShowing+1])
		}
		s.sprayShowing = -1
		if s.paint(s.sprayBlobs[i], s.spray[i:i+1]) {
			s.sprayShowing = i
		}
	}
	if s.sprayShowing != i {
		return nil
	}
	return &s.spray[i]
}

func dropPictures(bands []surgeBand) {
	for i := range bands {
		if bands[i].img != nil {
			bands[i].img.Dispose()
			bands[i].img = nil
		}
	}
}

// paint inflates a frame's indices and gives each of its bands a picture.
func (s *Surge) paint(blob []byte, bands []surgeBand) bool {
	zr, err := zlib.NewReader(bytes.NewReader(blob))
	if err != nil {
		return false
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return false
	}
	for i := range bands {
		b := &bands[i]
		if len(raw) < b.w*b.h {
			dropPictures(bands)
			return false
		}
		rgba := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
		for n, c := range raw[:b.w*b.h] {
			if int(c) < len(s.pal) {
				p := s.pal[c]
				rgba.Pix[n*4], rgba.Pix[n*4+1], rgba.Pix[n*4+2], rgba.Pix[n*4+3] = p.R, p.G, p.B, p.A
			}
		}
		raw = raw[b.w*b.h:]
		b.img = ebiten.NewImageFromImage(rgba)
	}
	return true
}

// surgeDepth is the depth sheet: frame k's grid of w x h spots, cell world units apart from (x, y), lies at column
// k mod cols, row k / cols, in per-ths of a pixel.
type surgeDepth struct {
	pix                  *image.RGBA
	cell, per, x, y      float64
	w, h, cols, nFrames  int
	frameW, frameH, rows int
}

// Surge reads flood name the first time it is asked for.
func (a *Assets) Surge(name string) (*Surge, error) {
	if s, ok := a.surges[name]; ok {
		return s, nil
	}
	base := filepath.Join(a.dir, "effects", "flood", name)
	b, err := os.ReadFile(base + ".json")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Scale float64
		Cut   float64
		MS    []int
		Bands [][][5]int
		Pack  struct {
			Frames [][2]int
			Pal    [][4]uint8
		}
		Depth struct {
			Cell, Per, X, Y float64
			W, H, Cols      int
		}
		Spray struct {
			At     float64
			MS     int
			Frames [][4]int
		}
	}
	if err = json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	packed, err := os.ReadFile(base + ".idx")
	if err != nil {
		return nil, err
	}
	s := &Surge{ms: doc.MS, scale: doc.Scale, cut: doc.Cut, sprayT: doc.Spray.At, sprayS: float64(doc.Spray.MS) / 1000, showing: -1, sprayShowing: -1}
	for _, c := range doc.Pack.Pal {
		s.pal = append(s.pal, color.RGBA{c[0], c[1], c[2], c[3]})
	}
	if len(doc.Bands) != len(doc.MS) || len(doc.Pack.Frames) != len(doc.MS)+len(doc.Spray.Frames) {
		return nil, fmt.Errorf("%s: %d frames of bands, %d spray, %d packed, %d timed", base, len(doc.Bands), len(doc.Spray.Frames), len(doc.Pack.Frames), len(doc.MS))
	}
	for k, r := range doc.Pack.Frames {
		if r[0] < 0 || r[1] < 0 || r[0]+r[1] > len(packed) {
			return nil, fmt.Errorf("%s: packed frame %d lies outside the file", base, k)
		}
		if k < len(doc.MS) {
			s.blobs = append(s.blobs, packed[r[0]:r[0]+r[1]])
		} else {
			s.sprayBlobs = append(s.sprayBlobs, packed[r[0]:r[0]+r[1]])
		}
	}
	for _, frame := range doc.Bands {
		var bands []surgeBand
		for _, r := range frame {
			bands = append(bands, surgeBand{d: r[0], x: float64(r[1]), y: float64(r[2]), w: r[3], h: r[4]})
		}
		s.bands = append(s.bands, bands)
	}
	for _, r := range doc.Spray.Frames {
		s.spray = append(s.spray, surgeBand{x: float64(r[0]), y: float64(r[1]), w: r[2], h: r[3]})
	}
	d := doc.Depth
	s.depth = surgeDepth{cell: d.Cell, per: d.Per, x: d.X, y: d.Y, w: d.W, h: d.H, cols: d.Cols}
	if s.depth.pix, err = loadRGBA(base + "_depth.png"); err != nil {
		return nil, err
	}
	if a.surges == nil {
		a.surges = map[string]*Surge{}
	}
	a.surges[name] = s
	return s, nil
}

// frameAt is the frame showing t seconds in; the last one stays.
func (s *Surge) frameAt(t float64) int {
	ms := t * 1000
	for i, d := range s.ms {
		if ms < float64(d) {
			return i
		}
		ms -= float64(d)
	}
	return len(s.ms) - 1
}

// Cut is when, in seconds, the surge's shot changes from the dam to where the water strikes.
func (s *Surge) Cut() float64 { return s.cut }

// Length is how long the surge plays, in seconds.
func (s *Surge) Length() float64 {
	total := 0
	for _, d := range s.ms {
		total += d
	}
	return float64(total) / 1000
}

// surgeDepth is how deep, in canvas pixels, a unit standing at cell point (u, v) is under the flood now.
func (f *Field) surgeDepth(u, v float64) float64 {
	s := f.Surge
	if s == nil || s.depth.pix == nil {
		return 0
	}
	d := s.depth
	i, j := int(((u+.5)*32-d.x)/d.cell), int(((v+.5)*32-d.y)/d.cell)
	if i < 0 || j < 0 || i >= d.w || j >= d.h {
		return 0
	}
	k := s.frameAt(f.SurgeT)
	return float64(d.pix.RGBAAt(k%d.cols*d.w+i, k/d.cols*d.h+j).R) / d.per
}

// surgeItems are the flood's pieces at SurgeT for the field's depth-sorted drawing: a band goes in with the units
// of its diagonal (before them: the water of a unit's own cell lies under it, the next band's over its legs), the
// spray last.
func (f *Field) surgeItems() []drawItem {
	s := f.Surge
	if s == nil || len(s.bands) == 0 {
		return nil
	}
	ox, oy := f.OriginX*K, f.OriginY*K
	k := s.frameAt(f.SurgeT)
	piece := func(b surgeBand, y, scale float64) drawItem {
		x0, y0 := ox+b.x, oy+b.y
		size := b.img.Bounds().Size()
		return drawItem{y: y, low: true, rect: image.Rect(int(x0), int(y0), int(x0+float64(size.X)*scale), int(y0+float64(size.Y)*scale)), draw: func() {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(math.Round(x0), math.Round(y0))
			f.renderOp(f.Canvas, op)
			f.Canvas.DrawImage(b.img, op)
		}}
	}
	var items []drawItem
	for _, b := range s.show(k) {
		_, y := f.Center(float64(b.d), 0) // computed as a unit's is, so the band sorts before the units of its diagonal
		items = append(items, piece(b, y, s.scale))
	}
	if i := int(math.Floor((f.SurgeT - s.sprayT) / s.sprayS)); f.SurgeT >= s.sprayT && i < len(s.spray) {
		if b := s.showSpray(i); b != nil {
			items = append(items, piece(*b, math.Inf(1), 1)) // the spray is baked at the canvas's own scale
		}
	}
	return items
}
