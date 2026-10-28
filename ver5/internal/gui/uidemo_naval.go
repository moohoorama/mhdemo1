package gui

import (
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/mapedit"
)

func init() { demos["naval"] = demoNaval }

// navalRows is the river demoNaval paints over the stage's map: water runs from the north-east edge to the south-west one.
var navalRows = []string{
	"gggggggggwwwwggggggg", "ggtggggggwwwwwgggggg", "gggggggggwwwwwgggggg", "ggggggggwwwwwggggtgg",
	"ggggggggwwwwwggggggg", "gggggggwwwwwgggggggg", "ggggggwwwwwwwggggggg", "ggggggwwwwwwwggggggg",
	"gggggggwwwwwgggggggg", "ggtggggwwwwwgggggggg", "ggggggwwwwwggggggggg", "gggggwwwwwwggggtgggg",
	"gggggwwwwwgggggggggg", "ggggwwwwwggggggggggg", "ggggwwwwwggggggggggg", "gggwwwwwgggggggggggg",
}

// demoNaval plays a stage (-stage, else the largest map) as a battle with boats (docs/plan-naval-water.md) on a river: the allies wait on the east
// bank and in boats on the water, the enemies come from the west bank; the allied phase ends and the AI armies
// cross. Shows the sunk water and its banks, boats with foot soldiers and riders, and boats gliding.
func demoNaval(d *Demo) error {
	allies := []content.Point{{X: 13, Y: 5}, {X: 10, Y: 6}, {X: 13, Y: 8}, {X: 9, Y: 7}, {X: 14, Y: 6}, {X: 11, Y: 4}}
	stage, data := d.stage, d.g.S.Data
	if data.StageIndex(stage) < 0 {
		for _, st := range data.Stages {
			if i := data.StageIndex(stage); i < 0 || st.Width*st.Height > data.Stages[i].Width*data.Stages[i].Height {
				stage = st.ID
			}
		}
	}
	err := d.battleWith(stage, func(doc *mapedit.Document) {
		data := d.g.S.Data
		for y := 0; y < doc.Play.H; y++ {
			for x := 0; x < doc.Play.W; x++ {
				c := byte('g')
				if y < len(navalRows) && x < len(navalRows[y]) {
					c = navalRows[y][x]
				}
				doc.Paint(data, x, y, content.LayerStructure, content.NoLayer)
				doc.Paint(data, x, y, content.LayerState, content.NoLayer)
				if c == 't' {
					doc.Paint(data, x, y, content.LayerGround, 'g')
					doc.Paint(data, x, y, content.LayerStructure, 't')
					continue
				}
				doc.Paint(data, x, y, content.LayerGround, c)
			}
		}
		doc.Stage.Boats, doc.Stage.Events, doc.Stage.Forced = true, nil, true
		doc.Stage.Allies = allies
		west := []content.Point{{X: 4, Y: 6}, {X: 3, Y: 7}, {X: 5, Y: 8}, {X: 4, Y: 4}, {X: 2, Y: 9}, {X: 6, Y: 5}, {X: 3, Y: 11}, {X: 1, Y: 6}}
		var enemies []content.Spawn
		for i, sp := range doc.Stage.Enemies {
			if i == len(west) {
				break
			}
			sp.X, sp.Y, sp.Facing, sp.Policy, sp.Goal, sp.GoalArea = west[i].X, west[i].Y, "NE", "assault", nil, 0
			enemies = append(enemies, sp)
		}
		doc.Stage.Enemies = enemies
	})
	if err != nil {
		return err
	}
	g := d.g
	d.step = func(f int) {
		switch f {
		case 1:
			g.zoom = 2
			g.camX, g.camY = g.field.Center(9, 6)
		case 90:
			g.command(core.Command{Kind: "end"})
		}
		if o := g.S.View(); f > 90 && o.Phase == "battle" && o.Turn != "ally" && !g.busy() {
			g.resolvePhase()
		}
	}
	return nil
}
