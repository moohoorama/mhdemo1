package gui

import (
	"srpg/internal/content"
	"srpg/internal/mapedit"
)

func init() { demos["mapfx"] = demoMapFX }

// demoMapFX paints what the battle map's pictures can show onto the first stage's map and plays it: a gate
// in the castle wall, a fenced yard, burning and burnt forest, village and ground, flames on a wall, and a
// bridge whole, damaged and broken. Coordinates are those of the first stage's map.
func demoMapFX(d *Demo) error {
	err := d.battleWith(d.stage, func(doc *mapedit.Document) {
		data := d.g.S.Data
		paint := func(x, y int, layer, glyph byte) { doc.Paint(data, x, y, layer, glyph) }
		const (
			ground, structure, state = content.LayerGround, content.LayerStructure, content.LayerState
		)
		paint(3, 4, ground, 'm') // a gate in the north wall, and one in the west wall
		paint(1, 7, ground, 'm')
		for x := 8; x <= 13; x++ { // a fenced yard with a gap to the south
			for y := 6; y <= 11; y++ {
				if (x == 8 || x == 13 || y == 6 || y == 11) && !(y == 11 && (x == 10 || x == 11)) {
					paint(x, y, structure, 'z')
				}
			}
		}
		paint(0, 0, state, 's') // forest: burnt, burning, glowing
		paint(1, 0, state, 'F')
		paint(2, 0, state, 'e')
		paint(14, 10, state, 's') // village
		paint(15, 10, state, 'F')
		paint(16, 10, state, 'e')
		paint(14, 11, state, 'F')
		for x := 4; x <= 6; x++ { // scorched ground
			for y := 11; y <= 12; y++ {
				paint(x, y, state, 's')
			}
		}
		paint(2, 4, state, 'F') // fire on the wall
		paint(5, 4, state, 'e')
		for _, y := range []int{2, 8, 11} { // three bridges over the river
			paint(21, y, ground, 'w')
			paint(22, y, ground, 'w')
			paint(21, y, structure, 'b')
			paint(22, y, structure, 'b')
		}
		paint(21, 8, state, 'd')
		paint(22, 8, state, 'd')
		paint(21, 11, state, 'x')
		paint(22, 11, state, 'x')
	})
	if err != nil {
		return err
	}
	g := d.g
	// the camera visits each group of features for 60 frames: the overview, the gates, the burning forest, the
	// village, the wall fire, the bridges
	sights := [][3]float64{{11, 7, 1.4}, {3, 6, 3}, {1, 1, 3}, {15, 11, 3}, {4, 4, 3}, {21.5, 7, 2.2}}
	d.step = func(frame int) {
		at := sights[min(frame/60, len(sights)-1)]
		g.zoom = at[2]
		g.camX, g.camY = g.field.Center(at[0], at[1])
	}
	return nil
}
