package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
)

func (f *Field) setDeployment(enabled bool, cells []content.Point) {
	f.deployment = enabled
	f.deploymentCells = cells
}
func (f *Field) deploymentBright(x, y float64) bool {
	u, v := f.Cell(x, y)
	return containsCell(f.deploymentCells, u, v)
}
func (f *Field) shadeScenery(dst *ebiten.Image, op *ebiten.DrawImageOptions, x, y float64) {
	if !f.sceneryLayer || dst != f.Canvas {
		return
	}
	if f.deployment && !f.deploymentBright(x, y) {
		op.ColorScale.Scale(.45, .45, .45, 1)
	}
	if m := f.exteriorShade(x, y); m < 1 {
		op.ColorScale.Scale(float32(m), float32(m), float32(m), 1)
	}
}

// drawDeploymentGround shades the ground everywhere but the cells allies can be placed on;
// taller scenery and units are shaded separately.
func (f *Field) drawDeploymentGround() {
	if !f.deployment {
		return
	}
	b := f.Canvas.Bounds()
	w, h := float32(b.Dx()), float32(b.Dy())
	var p vector.Path
	p.MoveTo(0, 0)
	p.LineTo(w, 0)
	p.LineTo(w, h)
	p.LineTo(0, h)
	p.Close()
	for _, c := range f.deploymentCells {
		f.cellPath(&p, c.X, c.Y, 0)
	}
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(color.NRGBA{A: 140})
	vector.FillPath(f.Canvas, &p, &vector.FillOptions{FillRule: vector.FillRuleEvenOdd}, op)
}
