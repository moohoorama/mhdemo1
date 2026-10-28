package gui

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// sunkDepth is how many canvas pixels a battle map's water lies below the ground (docs/plan-naval-water.md).
const sunkDepth = 9

// wet reports whether small tile glyph t lets water through: water, and the water under a bridge.
func wet(t byte) bool { return t == '~' || t == 'b' }

// makeBanks lays the earth faces of a sunk water's banks: on a water small tile's NW and NE edges where the
// neighbour is land, cut along the tile's own lower edge on that side when the land goes on there; and a dark
// line on its SE and SW edges where the land in front hides the lower water. The water texture repeats without
// seams, so water drawn at ground height under these faces looks sunk.
func (f *Field) makeBanks() {
	if !f.Map.SunkWater {
		return
	}
	lip := func(t byte) string {
		if strings.IndexByte("dichm", t) >= 0 {
			return "soil"
		}
		return "grass"
	}
	f.eachSmall(func(x, y int, t byte, px, py float64) {
		if t != '~' {
			return
		}
		var names []string
		for _, e := range []struct {
			side        string
			back, side2 [2]int // the neighbour behind the edge, and the one beside it on the same side
		}{{"nw", [2]int{-1, 0}, [2]int{0, 1}}, {"ne", [2]int{0, -1}, [2]int{1, 0}}} {
			n := f.small(x+e.back[0], y+e.back[1])
			if wet(n) {
				continue
			}
			name := "bank_" + e.side + "_" + lip(n)
			if !wet(f.small(x+e.side2[0], y+e.side2[1])) {
				name += "_cut"
			}
			names = append(names, name)
		}
		if !wet(f.small(x+1, y)) {
			names = append(names, "bank_line_se")
		}
		if !wet(f.small(x, y+1)) {
			names = append(names, "bank_line_sw")
		}
		for _, name := range names {
			s := f.A.Props[name].Sprite
			f.banks = append(f.banks, drawItem{rect: f.mapRect(s, px, py), draw: func() { f.blitMap(f.Canvas, s, px, py, nil) }})
		}
	})
}

// boat is the boat a unit stands in and how much of it shows (0 none, 1 whole): on sunk water, fading in
// with the unit's drop as it steps off the bank.
func (f *Field) boat(u *unitVis) (string, float64) {
	if !f.Map.SunkWater || u.lift >= 0 || u.actor {
		return "", 0
	}
	switch u.dir {
	case "SE", "NW":
		u.boatAxis = "u"
	case "SW", "NE":
		u.boatAxis = "v"
	}
	if u.boatAxis == "" {
		u.boatAxis = "u"
	}
	name := "boat_"
	if u.mounted {
		name = "boat_l_"
	}
	return name + u.boatAxis, min(1, -u.lift/sunkDepth)
}

// pose is the animation and frame a unit is drawn in: afloat it does not walk, the boat carries it.
func (f *Field) pose(u *unitVis) (string, int) {
	if _, k := f.boat(u); k > 0 && u.anim == "walk" {
		return "idle", u.art.FrameIndex("idle", u.animT*1000)
	}
	return u.anim, u.frame()
}

// drawAfloat draws boat, crew and the boat's front rim, rocking a little, then hides what the ground of the land
// cells in front covers (the boat rides lower than they stand) and puts it on the canvas.
func (f *Field) drawAfloat(u *unitVis, s Sprite, boat string, k, x, y float64, op *ebiten.DrawImageOptions) {
	cb := f.Canvas.Bounds()
	if f.afloat == nil || f.afloat.Bounds() != cb {
		if f.afloat != nil {
			f.afloat.Deallocate()
		}
		f.afloat = ebiten.NewImage(cb.Dx(), cb.Dy())
	}
	y += math.Round(math.Sin(f.clockMS/420 + (u.u+u.v)*1.7))
	hull, rim := f.A.Props[boat].Sprite, f.A.Props[boat+"_rim"].Sprite
	b := hull.Image.Bounds()
	x0, y0 := f.renderPoint(x-hull.PivotX, y-hull.PivotY)
	x1, y1 := f.renderPoint(x-hull.PivotX+float64(b.Dx()), y-hull.PivotY+float64(b.Dy()))
	sb := s.Image.Bounds()
	_, sy := f.renderPoint(x, y-3-s.PivotY)
	sx0, _ := f.renderPoint(x-s.PivotX, y)
	sx1, _ := f.renderPoint(x-s.PivotX+float64(sb.Dx()), y)
	area := image.Rect(int(min(x0, sx0))-2, int(min(y0, sy))-2, int(max(x1, sx1))+2, int(y1)+2).Intersect(cb)
	if area.Empty() {
		return
	}
	layer := f.afloat.SubImage(area).(*ebiten.Image)
	layer.Clear()
	hop := &ebiten.DrawImageOptions{}
	hop.ColorScale.ScaleAlpha(float32(k))
	f.blit(f.afloat, hull, x, y, hop)
	f.blit(f.afloat, s, x, y-3, op)
	rop := &ebiten.DrawImageOptions{}
	rop.ColorScale.ScaleAlpha(float32(k))
	f.blit(f.afloat, rim, x, y, rop)
	cu, cv := int(math.Round(u.u)), int(math.Round(u.v))
	var front vector.Path
	for _, d := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
		if t := f.Tile(cu+d[0], cv+d[1]); t != '~' && t != 'b' {
			f.cellPath(&front, cu+d[0], cv+d[1], 0)
		}
	}
	vector.FillPath(layer, &front, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
	cop := &ebiten.DrawImageOptions{}
	cop.GeoM.Translate(float64(area.Min.X), float64(area.Min.Y))
	f.Canvas.DrawImage(layer, cop)
}
