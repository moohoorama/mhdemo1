package gui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"srpg/internal/content"
	"srpg/internal/core"
)

// goalShape is one outline of the objective guide on the ground: a diamond under a unit (u, v may lie
// between cells while it walks) or the border of an area's cells. A win is the size of its cell, a
// loss smaller, a duel (dashed, in its group's colour) and a bonus goal (grey) between the two.
type goalShape struct {
	u, v              float64
	cells             []content.Point
	win, sub, pending bool
	duel              int // 1 + the duel group; 0 for a win or loss mark
}

var (
	goalWin  = color.RGBA{90, 235, 110, 255}
	goalLose = color.RGBA{245, 64, 52, 255}
	goalSub  = color.RGBA{196, 198, 204, 255}
	// duelColors stay away from the win green and the loss red; past five groups they repeat.
	duelColors = []color.RGBA{{250, 215, 60, 255}, {80, 185, 250, 255}, {180, 115, 245, 255}, {250, 125, 200, 255}, {240, 240, 235, 255}}
)

func duelColor(group int) color.RGBA { return duelColors[group%len(duelColors)] }

func (f *Field) setGoals(shapes []goalShape) { f.goals = shapes }

// goalShapes places the stage's goal marks: units where they are drawn, areas on their cells.
func (g *Game) goalShapes(o core.Observation) []goalShape {
	if g.S.Settings.NoGoalGuide || o.Phase != "battle" && o.Phase != "deployment" {
		return nil
	}
	var out []goalShape
	at := func(s goalShape, unit string) {
		if unit != "" {
			vis := g.vis[unit]
			if vis == nil || vis.gone || vis.dying >= 0 {
				return
			}
			s.u, s.v = vis.u, vis.v
		}
		out = append(out, s)
	}
	for _, m := range g.S.Engine.GoalMarks() {
		at(goalShape{cells: m.Cells, win: m.Win, pending: m.Pending}, m.Unit)
	}
	for _, m := range g.S.Engine.SubMarks() {
		at(goalShape{cells: m.Cells, sub: true, pending: m.Pending}, m.Unit)
	}
	for _, m := range g.S.Engine.DuelMarks() {
		at(goalShape{duel: 1 + m.Group}, m.Unit)
	}
	return out
}

// goalThrough is how strongly the guide shows again over scenery and units.
const goalThrough = .55

// drawGoals strokes the objective guide at alpha: over the ground tints under scenery and units, then
// faintly over them. A winning mark is the size of its cell, a losing one smaller, so a unit carrying both shows the two.
func (f *Field) drawGoals(c *ebiten.Image, alpha float64) {
	z := 1.0
	if f.renderZoom > 0 {
		z = f.renderZoom
	}
	pulse := .8 + .2*math.Sin(f.clockMS/1000*math.Pi*1.2)
	for _, s := range f.goals {
		col, k := goalLose, .62
		switch {
		case s.duel > 0:
			col, k = duelColor(s.duel-1), .77
		case s.win:
			col, k = goalWin, .92
		case s.sub:
			col, k = goalSub, .77
		}
		a := pulse
		if s.pending {
			a = .35
		}
		p := &vector.Path{}
		switch {
		case s.cells != nil:
			f.areaBorder(p, s.cells, k)
		case s.duel > 0:
			f.dashedDiamond(p, s.u, s.v, k, f.markLift(int(math.Round(s.u)), int(math.Round(s.v))))
		default:
			f.diamond(p, s.u, s.v, k, f.markLift(int(math.Round(s.u)), int(math.Round(s.v))))
		}
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(col)
		op.ColorScale.ScaleAlpha(float32(a * alpha))
		vector.StrokePath(c, p, &vector.StrokeOptions{Width: float32(3 * z), LineJoin: vector.LineJoinRound}, op)
	}
}

// cellCorners are a cell diamond's top, right, bottom and left corners from its centre; the edge from
// corner i to i+1 is shared with the neighbour at cellSides[i].
var (
	cellCorners = [4][2]float64{{0, -16}, {32, 0}, {0, 16}, {-32, 0}}
	cellSides   = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	// cellCornersUV are the same corners in cells
	cellCornersUV = [4][2]float64{{-.5, -.5}, {.5, -.5}, {.5, .5}, {-.5, .5}}
)

func (f *Field) diamond(p *vector.Path, u, v, k, lift float64) {
	x, y := f.Center(u, v)
	for i, d := range cellCorners {
		rx, ry := f.renderPoint(x+d[0]*k, y+d[1]*k-lift)
		if i == 0 {
			p.MoveTo(float32(rx), float32(ry))
		} else {
			p.LineTo(float32(rx), float32(ry))
		}
	}
	p.Close()
}

// dashedDiamond is diamond in dashes, each edge cut into the same number of them so the corners stay drawn.
func (f *Field) dashedDiamond(p *vector.Path, u, v, k, lift float64) {
	x, y := f.Center(u, v)
	const dashes, half = 2, .3 // long dashes: short ones break into dots a horse's legs hide
	for i := range cellCorners {
		a, b := cellCorners[i], cellCorners[(i+1)%4]
		for j := range dashes {
			t0, t1 := (float64(j)-half)/dashes, (float64(j)+half)/dashes
			for _, t := range [][2]float64{{max(0, t0), t1}, {1 + t0, 1}} {
				if t[0] >= t[1] {
					continue
				}
				ax, ay := f.renderPoint(x+(a[0]+(b[0]-a[0])*t[0])*k, y+(a[1]+(b[1]-a[1])*t[0])*k-lift)
				bx, by := f.renderPoint(x+(a[0]+(b[0]-a[0])*t[1])*k, y+(a[1]+(b[1]-a[1])*t[1])*k-lift)
				p.MoveTo(float32(ax), float32(ay))
				p.LineTo(float32(bx), float32(by))
			}
		}
	}
}

// areaBorder traces the edges of the cells that no other cell of the area shares, drawn inside the area by
// half of what k takes off a unit's diamond, so a win and a loss on the same area show as two rings.
func (f *Field) areaBorder(p *vector.Path, cells []content.Point, k float64) {
	d := (1 - k) / 2
	in := func(c content.Point, n [2]int) bool { return containsCell(cells, c.X+n[0], c.Y+n[1]) }
	// end moves a boundary corner along the side m it meets: in at an outer corner, out where the border turns inward
	end := func(c content.Point, n, m [2]int) float64 {
		switch {
		case !in(c, m):
			return -d
		case in(c, [2]int{m[0] + n[0], m[1] + n[1]}):
			return d
		}
		return 0
	}
	for _, c := range cells {
		lift := f.markLift(c.X, c.Y)
		pt := func(du, dv float64) (float32, float32) {
			x, y := f.Center(float64(c.X)+du, float64(c.Y)+dv)
			rx, ry := f.renderPoint(x, y-lift)
			return float32(rx), float32(ry)
		}
		for i, n := range cellSides {
			if in(c, n) {
				continue
			}
			ma, mb := cellSides[(i+3)%4], cellSides[(i+1)%4]
			a, b := cellCornersUV[i], cellCornersUV[(i+1)%4]
			ea, eb := end(c, n, ma), end(c, n, mb)
			ax, ay := pt(a[0]-float64(n[0])*d+float64(ma[0])*ea, a[1]-float64(n[1])*d+float64(ma[1])*ea)
			bx, by := pt(b[0]-float64(n[0])*d+float64(mb[0])*eb, b[1]-float64(n[1])*d+float64(mb[1])*eb)
			p.MoveTo(ax, ay)
			p.LineTo(bx, by)
		}
	}
}

// duelSwatch is a duel guide's dashed diamond in small, centred on (x, y), before its line of the briefing.
func duelSwatch(dst *ebiten.Image, x, y float64, col color.RGBA) {
	p := &vector.Path{}
	corners := [4][2]float64{{0, -6}, {12, 0}, {0, 6}, {-12, 0}}
	for i, a := range corners {
		b := corners[(i+1)%4]
		for _, t := range [][2]float64{{0, .3}, {.7, 1}} {
			p.MoveTo(float32(x+a[0]+(b[0]-a[0])*t[0]), float32(y+a[1]+(b[1]-a[1])*t[0]))
			p.LineTo(float32(x+a[0]+(b[0]-a[0])*t[1]), float32(y+a[1]+(b[1]-a[1])*t[1]))
		}
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(col)
	strokePath(dst, p, vector.StrokeOptions{Width: 2.5}, op)
}
