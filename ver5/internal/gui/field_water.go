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

// afloat is the boat a unit is in: its sprite name, how much of it shows (0 none, 1 whole) and where it lies relative
// to the unit's own draw point (nonzero only while the unit steps between the bank and the water).
type afloat struct {
	name   string
	k      float64
	dx, dy float64
	cu, cv int  // the cell the boat lies in
	walks  bool // the unit steps over land at this moment: it walks rather than stands in the boat
}

// boat is the boat a unit stands in, on sunk water. Afloat the unit does not walk and the boat carries it, whole from
// the moment the unit steps in. A step between the bank and the water leaves the boat where the water is, shown
// whole while the unit crosses to it or from it (the unit walks over the land half of that step and stands in the
// boat over the water half); the boat sinks out of sight once the unit has left it. A step onto a ford has none.
func (f *Field) boat(u *unitVis) afloat {
	if !f.Map.SunkWater || u.actor {
		return afloat{}
	}
	water := func(u, v float64) bool { return f.Tile(int(math.Round(u)), int(math.Round(v))) == '~' }
	out := afloat{k: 1, cu: int(math.Round(u.u)), cv: int(math.Round(u.v))}
	if w := u.walk; w != nil {
		from, to := water(w.fu, w.fv), water(w.tu, w.tv)
		switch {
		case !from && !to:
			return afloat{}
		case from != to:
			du, dv := w.tu-w.fu, w.tv-w.fv
			p := math.Max(0, math.Min(1, ((u.u-w.fu)*du+(u.v-w.fv)*dv)/(du*du+dv*dv)))
			wu, wv := w.fu, w.fv
			out.walks = p < .5 && to || p >= .5 && from
			if to {
				wu, wv = w.tu, w.tv
				out.k = math.Min(1, .25+p*3) // lies ready at the water and shows quickly as the unit sets off
			} else {
				out.k = math.Min(1, (1-p)*5/2) // sinks once the unit is well clear of it
			}
			out.cu, out.cv = int(math.Round(wu)), int(math.Round(wv))
			cx, cy := f.Center(u.u, u.v)
			bx, by := f.Center(wu, wv)
			out.dx, out.dy = bx-cx, (by-f.Lift(int(math.Round(wu)), int(math.Round(wv))))-(cy-u.lift)
		}
	} else if !water(u.u, u.v) || u.lift >= 0 {
		return afloat{}
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
	out.name = "boat_"
	if u.mounted {
		out.name = "boat_l_"
	}
	out.name += u.boatAxis
	return out
}

// pose is the animation and frame a unit is drawn in: afloat it does not walk, the boat carries it.
func (f *Field) pose(u *unitVis) (string, int) {
	if b := f.boat(u); b.k > 0 && !b.walks && u.anim == "walk" {
		return "idle", u.art.FrameIndex("idle", u.animT*1000)
	}
	return u.anim, u.frame()
}

// drawAfloat draws boat, crew and the boat's front rim, rocking a little, then hides what the ground of the land
// cells in front covers (the boat rides lower than they stand) and puts it on the canvas. A unit stepping over land
// toward or from its boat is drawn over the canvas afterwards, so that the ground cut away does not take it too.
func (f *Field) drawAfloat(u *unitVis, s Sprite, bt afloat, x, y float64, op *ebiten.DrawImageOptions) {
	cb := f.Canvas.Bounds()
	if f.afloat == nil || f.afloat.Bounds() != cb {
		if f.afloat != nil {
			f.afloat.Deallocate()
		}
		f.afloat = newLayer(cb.Dx(), cb.Dy())
	}
	rock := math.Round(math.Sin(f.clockMS/420 + (u.u+u.v)*1.7))
	bx, by := x+bt.dx, y+bt.dy+rock
	y += rock
	hull, rim := f.A.Props[bt.name].Sprite, f.A.Props[bt.name+"_rim"].Sprite
	b := hull.Image.Bounds()
	x0, y0 := f.renderPoint(bx-hull.PivotX, by-hull.PivotY)
	x1, y1 := f.renderPoint(bx-hull.PivotX+float64(b.Dx()), by-hull.PivotY+float64(b.Dy()))
	sb := s.Image.Bounds()
	_, sy := f.renderPoint(x, y-3-s.PivotY)
	sx0, _ := f.renderPoint(x-s.PivotX, y)
	sx1, _ := f.renderPoint(x-s.PivotX+float64(sb.Dx()), y)
	area := image.Rect(int(min(x0, sx0))-2, int(min(y0, sy))-2, int(max(x1, sx1))+2, int(max(y1, 0))+2).Intersect(cb)
	if area.Empty() {
		return
	}
	layer := f.afloat.SubImage(area).(*ebiten.Image)
	layer.Clear()
	hop := &ebiten.DrawImageOptions{}
	hop.ColorScale.ScaleAlpha(float32(bt.k))
	f.blit(f.afloat, hull, bx, by, hop)
	if !bt.walks {
		f.blit(f.afloat, s, x, y-3, op)
	}
	rop := &ebiten.DrawImageOptions{}
	rop.ColorScale.ScaleAlpha(float32(bt.k))
	f.blit(f.afloat, rim, bx, by, rop)
	var front vector.Path
	for _, d := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
		if t := f.Tile(bt.cu+d[0], bt.cv+d[1]); t != '~' && t != 'b' {
			f.cellPath(&front, bt.cu+d[0], bt.cv+d[1], 0)
		}
	}
	vector.FillPath(layer, &front, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
	cop := &ebiten.DrawImageOptions{}
	cop.GeoM.Translate(float64(area.Min.X), float64(area.Min.Y))
	f.Canvas.DrawImage(layer, cop)
	if bt.walks {
		f.blit(f.Canvas, s, x, y, op)
	}
}
