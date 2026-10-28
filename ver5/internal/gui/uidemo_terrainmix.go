package gui

import (
	"srpg/internal/content"
	"srpg/internal/mapedit"
)

func init() { demos["terrainmix"] = demoTerrainMix }

// terrainPatches are 6×6-cell samples, row by row: a background ground, a blob of another, and what else stands there.
var terrainPatches = []struct{ bg, blob, extra byte }{
	{'w', 's', 0}, {'g', 's', 0}, {'d', 's', 0}, {'l', 's', 0},
	{'g', 'l', 0}, {'d', 'l', 0}, {'w', 'l', 0}, {'s', 'l', 0},
	{'i', 's', 0}, {'i', 'l', 0}, {'s', 'g', 'c'}, {'l', 'g', 'c'},
	{'s', 'g', 'h'}, {'l', 'g', 'h'}, {'s', 'g', 't'}, {'l', 'g', 't'},
	{'l', 'g', 'v'}, {'l', 'g', 'z'}, {'w', 0, 'R'}, {'d', 0, 'P'},
}

// demoTerrainMix paints the ford (s) and the plain (l) against every other ground on the stage's map grown to 26×31 and visits
// each sample: a trial of the two new grounds, the armies out of the way in the bottom row.
func demoTerrainMix(d *Demo) error {
	err := d.battleWith(d.stage, func(doc *mapedit.Document) {
		data := d.g.S.Data
		if err := doc.Resize(mapedit.Rect{X: 0, Y: 0, W: 26, H: 31}); err != nil {
			panic(err)
		}
		// the armies out of the way in the bottom row, no story
		doc.Stage.Events = nil
		for i := range doc.Stage.Party {
			doc.Stage.Party[i].X, doc.Stage.Party[i].Y = 7+i, 30
		}
		for i := range doc.Stage.Enemies {
			doc.Stage.Enemies[i].X, doc.Stage.Enemies[i].Y = 25-i, 30
		}
		doc.Stage.Allies = []content.Point{{X: 1, Y: 30}, {X: 2, Y: 30}, {X: 3, Y: 30}, {X: 4, Y: 30}, {X: 5, Y: 30}, {X: 6, Y: 30}}
		paint := func(x, y int, layer, glyph byte) { doc.Paint(data, x, y, layer, glyph) }
		G, S := byte(content.LayerGround), byte(content.LayerStructure)
		for y := 0; y < doc.Play.H; y++ {
			for x := 0; x < doc.Play.W; x++ {
				paint(x, y, S, '-')
				paint(x, y, byte(content.LayerState), '-')
			}
		}
		for n, p := range terrainPatches {
			x0, y0 := n%4*6+1, n/4*6
			for y := 0; y < 6; y++ {
				for x := 0; x < 6; x++ {
					c := p.bg
					dx, dy := float64(x)-2.5, float64(y)-2.5
					if p.blob != 0 && dx*dx+dy*dy+float64((x*7+y*3)%3)*.6 <= 4.6 {
						c = p.blob
					}
					switch p.extra {
					case 'R': // strips from the river to the soil
						c = []byte{'w', 'w', 's', 'g', 'l', 'd'}[(x+y/2)%6]
					case 'P':
						c = []byte{'d', 'l', 'g', 's', 'w', 'w'}[min(5, (x*2+y)/2)]
					}
					paint(x0+x, y0+y, G, c)
					switch {
					case p.extra == 'c' && y == 3:
						paint(x0+x, y0+y, G, 'c')
					case p.extra == 'h' && x >= 2 && x <= 3 && y >= 1 && y <= 2:
						paint(x0+x, y0+y, G, 'h')
					case p.extra == 't' && (x+2*y)%3 == 0:
						paint(x0+x, y0+y, S, 't')
					case p.extra == 'v' && x == 1 && y == 1, p.extra == 'v' && x == 4 && y == 4:
						paint(x0+x, y0+y, S, 'v')
					case p.extra == 'z' && (y == 0 || x == 5) && y < 5:
						paint(x0+x, y0+y, S, 'z')
					}
				}
			}
		}
	})
	if err != nil {
		return err
	}
	g := d.g
	d.step = func(frame int) {
		n := frame / 10
		if n >= len(terrainPatches) {
			g.zoom = .9
			g.camX, g.camY = g.field.Center(13, 15)
			return
		}
		g.zoom = 2.2
		g.camX, g.camY = g.field.Center(float64(n%4*6+1)+2.5, float64(n/4*6)+2.5)
	}
	return nil
}
