package gui

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"srpg/internal/content"
)

func (f *Field) setDeployment(enabled bool, cells []content.Point) {
	if f.deployment != enabled || len(f.deploymentCells) != len(cells) {
		f.deploymentMaskKey = ""
	}
	f.deployment = enabled
	f.deploymentCells = cells
}
func (f *Field) deploymentBright(x, y float64) bool {
	u, v := f.Cell(x, y)
	return containsCell(f.deploymentCells, u, v)
}
func (f *Field) shadeScenery(dst *ebiten.Image, op *ebiten.DrawImageOptions, x, y float64) {
	if f.deployment && f.sceneryLayer && dst == f.Canvas && !f.deploymentBright(x, y) {
		op.ColorScale.Scale(.45, .45, .45, 1)
	}
}
func (f *Field) drawDeploymentGround() {
	if !f.deployment {
		return
	}
	b := f.Canvas.Bounds()
	key := fmt.Sprintf("%v/%g/%g/%g/%v", b, f.renderX, f.renderY, f.renderZoom, f.deploymentCells)
	if key != f.deploymentMaskKey {
		if f.deploymentMask == nil || f.deploymentMask.Bounds() != b {
			if f.deploymentMask != nil {
				f.deploymentMask.Deallocate()
			}
			f.deploymentMask = ebiten.NewImage(b.Dx(), b.Dy())
		}
		f.deploymentMask.Fill(color.NRGBA{A: 140})
		var path vector.Path
		for _, p := range f.deploymentCells {
			// Keep the mask on the ground; taller scenery and units are shaded separately.
			x, y := f.Center(float64(p.X), float64(p.Y))
			for i, xy := range [][2]float64{{x, y - 16}, {x + 32, y}, {x, y + 16}, {x - 32, y}} {
				rx, ry := f.renderPoint(xy[0], xy[1])
				if i == 0 {
					path.MoveTo(float32(rx), float32(ry))
				} else {
					path.LineTo(float32(rx), float32(ry))
				}
			}
			path.Close()
		}
		vector.FillPath(f.deploymentMask, &path, nil, &vector.DrawPathOptions{ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
		f.deploymentMaskKey = key
	}
	f.Canvas.DrawImage(f.deploymentMask, nil)
}
