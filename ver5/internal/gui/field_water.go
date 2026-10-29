package gui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// sunkDepth is how many canvas pixels a battle map's water lies below the ground (docs/plan-naval-water.md).
const sunkDepth = 9

// fordDepth is how far a battle map's ford, a stony bed under shallow water, lies below the ground: about halfway
// down to the water (docs/plan-tile-floors.md).
const fordDepth = 5

// wet reports whether small tile glyph t lets water through: water, and the water under a bridge.
func wet(t byte) bool { return t == '~' || t == 'b' }

// bridgeEdges are the side pieces of bridge small tile (x, y) on a battle map's sunk water: on each edge over open
// water a rail, and on a front edge the deck's side beam and a pier down to the water. A broken bridge has none.
func (f *Field) bridgeEdges(x, y int, at image.Point, px, py float64) []drawItem {
	if !f.Map.SunkWater {
		return nil
	}
	var items []drawItem
	for _, e := range []struct {
		name string
		d    [2]int
	}{{"bridge_rail_nw", [2]int{-1, 0}}, {"bridge_rail_ne", [2]int{0, -1}}, {"bridge_edge_se", [2]int{1, 0}}, {"bridge_edge_sw", [2]int{0, 1}}} {
		if f.small(x+e.d[0], y+e.d[1]) != '~' {
			continue
		}
		s := f.A.Props[e.name].Sprite
		items = append(items, drawItem{rect: f.mapRect(s, px, py), draw: func() {
			if f.look(at.X, at.Y) != "broken" && f.shows("bridge-edge") {
				f.blitMap(f.Canvas, s, px, py, nil)
			}
		}})
	}
	return items
}

// boat is the boat a unit stands in and how much of it shows (0 none, 1 whole): on sunk water, fading in
// with the unit's drop as it steps off the bank.
func (f *Field) boat(u *unitVis) (string, float64) {
	if !f.Map.SunkWater || u.lift >= 0 || u.actor || f.Tile(int(math.Round(u.u)), int(math.Round(u.v))) == 's' {
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
		f.afloat = newLayer(cb.Dx(), cb.Dy())
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
