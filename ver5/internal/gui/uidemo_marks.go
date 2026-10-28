package gui

import (
	"srpg/internal/content"
	"srpg/internal/mapedit"
)

func init() { demos["marks"] = demoMarks }

// demoMarks lays the marks critical spells leave (embers, wet ground, rubble) on the first stage's map, each in
// a cross as a cross-shaped spell leaves it, one row on grass and one on wasteland, and visits them up close.
func demoMarks(d *Demo) error {
	err := d.battleWith(d.stage, func(doc *mapedit.Document) {
		data := d.g.S.Data
		for i, st := range []byte{'e', 'w', 'r'} {
			for row, ground := range []byte{'g', 'd'} {
				for _, o := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
					x, y := 4+i*4+o[0], 4+row*4+o[1]
					doc.Paint(data, x, y, content.LayerStructure, content.NoLayer)
					doc.Paint(data, x, y, content.LayerGround, ground)
					doc.Paint(data, x, y, content.LayerState, st)
				}
			}
		}
	})
	if err != nil {
		return err
	}
	g := d.g
	sights := [][3]float64{{8, 6, 2}, {4, 4, 3.5}, {8, 4, 3.5}, {12, 4, 3.5}, {4, 8, 3.5}, {8, 8, 3.5}, {12, 8, 3.5}}
	d.step = func(frame int) {
		at := sights[min(frame/30, len(sights)-1)]
		g.zoom = at[2]
		g.camX, g.camY = g.field.Center(at[0], at[1])
	}
	return nil
}
