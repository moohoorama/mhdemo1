package gui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
)

// fireCell is a cell with fire on it and the size of the blaze.
type fireCell struct {
	X, Y int
	Size float32
}

// fireCells are the cells that burn (embers, flames), with the size of each blaze.
func fireCells(data *content.Data, o core.Observation) (cells []fireCell) {
	if o.Map == nil {
		return nil
	}
	for y := 0; y < o.Map.Height; y++ {
		for x := 0; x < o.Map.Width; x++ {
			if s := data.States[string(o.Map.Cell(x, y).State)]; s.Blaze > 0 {
				cells = append(cells, fireCell{x, y, float32(s.Blaze)})
			}
		}
	}
	return cells
}

// drawFlames puts fire on the burning cells: small flames on burning ground, a tall wall of fire.
func (g *Game) drawFlames(dst *ebiten.Image, o core.Observation) {
	z := float32(g.zoom)
	for _, c := range fireCells(g.S.Data, o) {
		x, y := g.field.Center(float64(c.X), float64(c.Y))
		sx, sy := g.toScreen(x, y)
		flames(dst, float32(sx), float32(sy), g.clockMS/1000+float64(c.X*7+c.Y*13)*.37, z, c.Size)
	}
}
