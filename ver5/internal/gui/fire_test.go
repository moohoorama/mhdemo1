package gui

import (
	"testing"

	"srpg/internal/content"
	"srpg/internal/core"
)

func TestFireCellsAreEmbersAndFlames(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	st := content.Stage{Width: 3, Height: 1, Tiles: []string{"g-eg-Fg-s"}}
	cells := fireCells(d, core.Observation{Map: &st})
	if len(cells) != 2 || cells[0].X != 0 || cells[1].X != 1 {
		t.Fatalf("cells: %+v", cells)
	}
	if cells[1].Size <= cells[0].Size {
		t.Fatalf("flames stand taller than embers: %+v", cells)
	}
}
