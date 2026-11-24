package maplight

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"

	"srpg/internal/scenemap"
)

const (
	margin = 48 // map pixels around the map bounds, as in the game's field
	scale  = 2  // canvas pixels per map pixel (the game's K)
)

// Frame is where a map's ground lies on its canvas.
type Frame struct {
	W, H   int // canvas size
	OX, OY int // canvas pixel of the map's origin
}

// FrameOf is the canvas the game gives a map.
func FrameOf(m *scenemap.Map) Frame {
	b := m.Bounds()
	return Frame{W: (b[2] - b[0] + 2*margin) * scale, H: (b[3] - b[1] + 2*margin) * scale,
		OX: (margin - b[0]) * scale, OY: (margin - b[1]) * scale}
}

// Cell is the canvas pixel of the centre of small tile (u, v).
func (f Frame) Cell(u, v int) (x, y int) { return f.OX + 16*(u-v), f.OY + 8*(u+v+1) }

// Render draws the whole canvas of a map: ground, then everything that stands, ground by ground.
func (a *Atlas) Render(m *scenemap.Map) (*image.RGBA, Frame, error) {
	plan, err := scenemap.Build(m, a.Catalog, scenemap.Options{Decor: true})
	if err != nil {
		return nil, Frame{}, err
	}
	art := a.withGrass(m.Grass)
	fr := FrameOf(m)
	dst := image.NewRGBA(image.Rect(0, 0, fr.W, fr.H))
	r := &renderer{a: art, dst: dst, ox: fr.OX, oy: fr.OY, flood: a.Catalog.Rules.Flood}
	for _, d := range plan.Flat {
		r.draw(d)
	}
	r.shadows(plan.Solid, nil)
	r.pieces(plan.Solid)
	for _, l := range plan.Levels {
		top := image.NewRGBA(dst.Rect)
		sub := &renderer{a: art, dst: top, ox: fr.OX, oy: fr.OY, flood: r.flood}
		for _, d := range l.Flat {
			sub.draw(d)
		}
		draw.Draw(dst, dst.Rect, top, image.Point{}, draw.Over)
		r.shadows(l.Solid, top)
		r.pieces(l.Solid)
	}
	return dst, fr, nil
}

type renderer struct {
	a      *Atlas
	dst    *image.RGBA
	ox, oy int
	flood  scenemap.Flood
	err    error
}

// shadows casts the ground shadows of the objects of one ground in one layer and lays it on the canvas; on a raised
// ground only where its top is.
func (r *renderer) shadows(ps []scenemap.Piece, top *image.RGBA) {
	layer := image.NewRGBA(r.dst.Rect)
	cast := false
	for _, p := range ps {
		if p.Rank != scenemap.RankObject || slices.ContainsFunc(p.Draws, func(d scenemap.Draw) bool { return d.Kind == scenemap.DrawWaterline }) {
			continue
		}
		sh := r.a.shadowOf(p.Name)
		if sh == nil {
			continue
		}
		x, y := math.Round(p.Anchor[0]+float64(r.ox)-sh.PivotX), math.Round(p.Anchor[1]+float64(r.oy)-sh.PivotY)
		maxBlit(layer, sh.Image, int(x), int(y))
		cast = true
	}
	if !cast {
		return
	}
	if top != nil {
		for i := 0; i < len(layer.Pix); i += 4 {
			a := uint32(top.Pix[i+3])
			for k := 0; k < 4; k++ {
				layer.Pix[i+k] = uint8(uint32(layer.Pix[i+k]) * a / 255)
			}
		}
		draw.Draw(r.dst, r.dst.Rect, layer, image.Point{}, draw.Over)
		return
	}
	laySourceAtop(r.dst, layer)
}

func (r *renderer) pieces(ps []scenemap.Piece) {
	for _, p := range ps {
		for _, d := range p.Draws {
			r.draw(d)
		}
	}
}

// picture is the image a draw of a plan lays and where its top left sits from the draw's point.
func (r *renderer) picture(d scenemap.Draw) (*image.RGBA, image.Point) {
	a := r.a
	switch d.Kind {
	case scenemap.DrawTile:
		if d.Name == "water" && len(a.Water) > 0 {
			return a.Tiles[a.Water[0]].Image, image.Point{}
		}
		return a.Tiles[d.Tile].Image, image.Point{}
	case scenemap.DrawSprite:
		switch d.Sheet {
		case "scenery":
			return a.Scenery[d.Name].Image, image.Point{}
		case "structures":
			return a.Structures[d.Name].Image, image.Point{}
		}
		if strip := a.Anims[d.Name]; len(strip) > 0 {
			p, s := a.Props[d.Name], strip[0]
			return s.Image, image.Pt(int(p.PivotX-s.PivotX), int(p.PivotY-s.PivotY))
		}
		return a.Props[d.Name].Image, image.Point{}
	case scenemap.DrawAnim:
		if strip := a.Anims[d.Name]; len(strip) > 0 {
			return strip[0].Image, image.Point{}
		}
	case scenemap.DrawFill:
		return diamond(hexColor(d.Color)), image.Pt(-32, 0)
	case scenemap.DrawFlood:
		return r.floodTile(d.Tile), image.Pt(-16, 0)
	case scenemap.DrawWaterline:
		return r.waterline(d.Line)
	}
	return nil, image.Point{}
}

func (r *renderer) draw(d scenemap.Draw) {
	img, at := r.picture(d)
	if img == nil {
		return
	}
	x, y := r.ox+d.X+at.X, r.oy+d.Y+at.Y
	if d.Tint == "" && len(d.Clip) == 0 {
		draw.Draw(r.dst, image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+img.Bounds().Dx(), y+img.Bounds().Dy())}, img, image.Point{}, draw.Over)
		return
	}
	var tint *[3]float64
	if d.Tint != "" {
		c := hexColor(d.Tint)
		tint = &[3]float64{float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255}
	}
	var clip [][2]float64
	for _, p := range d.Clip {
		clip = append(clip, [2]float64{float64(p[0] - d.X - at.X), float64(p[1] - d.Y - at.Y)})
	}
	b := img.Bounds()
	for sy := 0; sy < b.Dy(); sy++ {
		for sx := 0; sx < b.Dx(); sx++ {
			if clip != nil && !inside(clip, float64(sx)+.5, float64(sy)+.5) {
				continue
			}
			si := img.PixOffset(b.Min.X+sx, b.Min.Y+sy)
			sa := uint32(img.Pix[si+3])
			if sa == 0 {
				continue
			}
			px, py := x+sx, y+sy
			if !image.Pt(px, py).In(r.dst.Rect) {
				continue
			}
			sr, sg, sb := uint32(img.Pix[si]), uint32(img.Pix[si+1]), uint32(img.Pix[si+2])
			if tint != nil {
				sr, sg, sb = uint32(float64(sr)*tint[0]+.5), uint32(float64(sg)*tint[1]+.5), uint32(float64(sb)*tint[2]+.5)
			}
			di := r.dst.PixOffset(px, py)
			inv := 255 - sa
			r.dst.Pix[di] = uint8(sr + uint32(r.dst.Pix[di])*inv/255)
			r.dst.Pix[di+1] = uint8(sg + uint32(r.dst.Pix[di+1])*inv/255)
			r.dst.Pix[di+2] = uint8(sb + uint32(r.dst.Pix[di+2])*inv/255)
			r.dst.Pix[di+3] = uint8(sa + uint32(r.dst.Pix[di+3])*inv/255)
		}
	}
}

// inside reports whether (x, y) lies in the convex polygon.
func inside(poly [][2]float64, x, y float64) bool {
	sign := 0.0
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		c := (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
		if c == 0 {
			continue
		}
		if sign == 0 {
			sign = c
		} else if sign*c < 0 {
			return false
		}
	}
	return true
}

func hexColor(hex string) color.RGBA {
	c := color.RGBA{A: 255}
	for i, p := range []*uint8{&c.R, &c.G, &c.B} {
		var v int
		fmt.Sscanf(hex[1+2*i:3+2*i], "%x", &v)
		*p = uint8(v)
	}
	return c
}

// diamond is a game cell's diamond (64 x 32) filled with a colour.
func diamond(c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for k := range 32 {
		x0, x1 := scenemap.DiamondRow(32, k)
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, k, c)
		}
	}
	return img
}

func hashInt(a, b int) uint32 {
	h := uint32(a)*374761393 + uint32(b)*668265263
	h = (h ^ h>>13) * 1274126177
	return h ^ h>>16
}

func rippled(x, y, variant int) int {
	for k := range 4 {
		h := hashInt(variant*7+k, 31)
		rx, ry, n := 4+int(h%22), 3+int(h/22%10), 2+int(h/220%3)
		switch {
		case y == ry && x >= rx && x < rx+n:
			return 1
		case y == ry+1 && x > rx && x <= rx+n:
			return 2
		}
	}
	return 0
}

func premul(v uint8, alpha float64) uint8 { return uint8(float64(v)*alpha + .5) }

func mixRGBA(a, b color.RGBA, k float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-k) + float64(y)*k + .5) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// floodTile is the flood water over one small tile (32 x 16) by its scenemap.DrawFlood tile.
func (r *renderer) floodTile(tile int) *image.RGBA {
	water, foam := hexColor(r.flood.Color), hexColor(r.flood.Foam)
	sides, variant := tile&15, tile>>4
	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for y := range 16 {
		for x := range 32 {
			cx, cy := float64(x)+.5-16, float64(y)+.5-8
			du, dv := cx/32+cy/16, cy/16-cx/32
			if math.Abs(du) > .5 || math.Abs(dv) > .5 {
				continue
			}
			shore := 1.0
			for bit, d := range []float64{dv + .5, .5 - du, .5 - dv, du + .5} {
				if sides&(1<<bit) != 0 {
					shore = math.Min(shore, d)
				}
			}
			c, alpha := water, r.flood.Alpha
			switch {
			case shore < .09:
				c, alpha = mixRGBA(water, foam, .6), .7
			case shore < .4:
				alpha *= .55 + .45*shore/.4
			}
			if shore >= .09 {
				switch rippled(x, y, variant) {
				case 1:
					c = mixRGBA(c, foam, .45)
				case 2:
					c = mixRGBA(c, color.RGBA{A: 255}, .18)
				}
			}
			img.SetRGBA(x, y, color.RGBA{premul(c.R, alpha), premul(c.G, alpha), premul(c.B, alpha), uint8(alpha*255 + .5)})
		}
	}
	return img
}

// waterline is the foam line along where the flood meets something standing in it, and where its top left sits.
func (r *renderer) waterline(pts [][2]int) (*image.RGBA, image.Point) {
	if len(pts) != 3 {
		return nil, image.Point{}
	}
	w, s, e := pts[0], pts[1], pts[2]
	x0, y0 := w[0], min(w[1], e[1])-1
	width, height := e[0]-x0+1, s[1]-y0+2
	foam := hexColor(r.flood.Foam)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	set := func(x, y int, alpha float64) {
		if x >= 0 && y >= 0 && x < width && y < height {
			img.SetRGBA(x, y, color.RGBA{premul(foam.R, alpha), premul(foam.G, alpha), premul(foam.B, alpha), uint8(alpha*255 + .5)})
		}
	}
	for x := w[0]; x <= e[0]; x++ {
		var y float64
		if x <= s[0] {
			y = float64(w[1]) + float64(s[1]-w[1])*float64(x-w[0])/math.Max(1, float64(s[0]-w[0]))
		} else {
			y = float64(s[1]) + float64(e[1]-s[1])*float64(x-s[0])/math.Max(1, float64(e[0]-s[0]))
		}
		set(x-x0, int(math.Round(y))-y0, .85)
		if (x/2)%2 == 0 {
			set(x-x0, int(math.Round(y))-y0+1, .4)
		}
	}
	return img, image.Pt(x0, y0)
}
