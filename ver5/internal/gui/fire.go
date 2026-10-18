package gui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// fireCells are the cells an event has set on fire (burning ground, flames), with the size of the blaze.
func fireCells(o core.Observation) (cells []core.TileOverride, sizes []float32) {
	for _, t := range o.TileOverrides {
		switch t.Tile {
		case "F":
			cells, sizes = append(cells, t), append(sizes, .8)
		case "W":
			cells, sizes = append(cells, t), append(sizes, 1.4)
		}
	}
	return cells, sizes
}

// drawFlames puts fire on the burning cells: small flames on burning ground, a tall wall of fire.
func (g *Game) drawFlames(dst *ebiten.Image, o core.Observation) {
	cells, sizes := fireCells(o)
	z := float32(g.zoom)
	for i, c := range cells {
		x, y := g.field.Center(float64(c.X), float64(c.Y))
		sx, sy := g.toScreen(x, y)
		flames(dst, float32(sx), float32(sy), g.clockMS/1000+float64(c.X*7+c.Y*13)*.37, z, sizes[i])
	}
}
