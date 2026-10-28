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
// between cells while it walks) or the border of an area's cells.
type goalShape struct {
	u, v         float64
	cells        []content.Point
	win, pending bool
}

var (
	goalWin  = color.RGBA{90, 235, 110, 255}
	goalLose = color.RGBA{245, 64, 52, 255}
)

func (f *Field) setGoals(shapes []goalShape) { f.goals = shapes }

// goalShapes places the stage's goal marks: units where they are drawn, areas on their cells.
func (g *Game) goalShapes(o core.Observation) []goalShape {
	if g.S.Settings.NoGoalGuide || g.S.Engine == nil || o.Phase != "battle" && o.Phase != "deployment" {
		return nil
	}
	var out []goalShape
	for _, m := range g.S.Engine.GoalMarks() {
		s := goalShape{cells: m.Cells, win: m.Win, pending: m.Pending}
		if m.Unit != "" {
			vis := g.vis[m.Unit]
			if vis == nil || vis.gone || vis.dying >= 0 {
				continue
			}
			s.u, s.v = vis.u, vis.v
		}
		out = append(out, s)
	}
	return out
}

// drawGoals strokes the objective guide over the ground tints, under scenery and units: a winning
// mark the size of its cell, a losing one smaller, so a unit carrying both shows the two.
func (f *Field) drawGoals(c *ebiten.Image) {
	z := 1.0
	if f.renderZoom > 0 {
		z = f.renderZoom
	}
	pulse := .8 + .2*math.Sin(f.clockMS/1000*math.Pi*1.2)
	for _, s := range f.goals {
		col, k := goalLose, .62
		if s.win {
			col, k = goalWin, .92
		}
		a := pulse
		if s.pending {
			a = .35
		}
		p := &vector.Path{}
		if s.cells != nil {
			f.areaBorder(p, s.cells)
		} else {
			f.diamond(p, s.u, s.v, k, max(0, f.Lift(int(math.Round(s.u)), int(math.Round(s.v)))))
		}
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(col)
		op.ColorScale.ScaleAlpha(float32(a))
		vector.StrokePath(c, p, &vector.StrokeOptions{Width: float32(3 * z), LineJoin: vector.LineJoinRound}, op)
	}
}

// cellCorners are a cell diamond's top, right, bottom and left corners from its centre; the edge from
// corner i to i+1 is shared with the neighbour at cellSides[i].
var (
	cellCorners = [4][2]float64{{0, -16}, {32, 0}, {0, 16}, {-32, 0}}
	cellSides   = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
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

// areaBorder traces the edges of the cells that no other cell of the area shares.
func (f *Field) areaBorder(p *vector.Path, cells []content.Point) {
	for _, c := range cells {
		x, y := f.Center(float64(c.X), float64(c.Y))
		lift := max(0, f.Lift(c.X, c.Y))
		for i, n := range cellSides {
			if containsCell(cells, c.X+n[0], c.Y+n[1]) {
				continue
			}
			a, b := cellCorners[i], cellCorners[(i+1)%4]
			ax, ay := f.renderPoint(x+a[0], y+a[1]-lift)
			bx, by := f.renderPoint(x+b[0], y+b[1]-lift)
			p.MoveTo(float32(ax), float32(ay))
			p.LineTo(float32(bx), float32(by))
		}
	}
}
