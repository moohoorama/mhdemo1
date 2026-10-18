package gui

import (
	"testing"

	"srpg/internal/core"
)

func TestFireCellsAreBurningGroundAndFlameWalls(t *testing.T) {
	o := core.Observation{State: core.State{TileOverrides: []core.TileOverride{{X: 1, Y: 2, Tile: "F"}, {X: 3, Y: 4, Tile: "W"}, {X: 5, Y: 6, Tile: "d"}}}}
	cells, sizes := fireCells(o)
	if len(cells) != 2 || cells[0].X != 1 || cells[1].X != 3 {
		t.Fatalf("cells: %+v", cells)
	}
	if sizes[1] <= sizes[0] {
		t.Fatalf("a wall of flames stands taller than burning ground: %v", sizes)
	}
}
