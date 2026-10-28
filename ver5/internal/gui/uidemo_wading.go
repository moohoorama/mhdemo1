package gui

import (
	"maps"
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/mapedit"
)

func init() { demos["wading"] = demoWading }

// demoWading walks a foot officer and a mounted one of the stage from the grass into a ford, out and in again, then
// visits a row of foot soldiers and a row of riders standing in the ford, one to each of the eight facings, on the
// stage's map grown to 24×18: the wading overlay (field_wade.go). Frames 80-700 are the walk, 740-931 the close-ups.
func demoWading(d *Demo) error {
	data := d.g.S.Data
	// a foot and a mounted officer of the stage's own for the walk; for the rows, the troop templates the stages field
	// most that move like the foot officer and that ride
	mounted := func(class string) bool { return slices.Contains(data.Classes[class].Tags, content.TagMounted) }
	var walkers [2]string
	st := data.Stages[data.StageIndex(d.stage)]
	for _, id := range st.Required {
		k := 0
		if mounted(data.Characters[id].Class) {
			k = 1
		}
		if walkers[k] == "" {
			walkers[k] = id
		}
	}
	fielded := map[string]int{}
	for _, st := range data.Stages {
		for _, sp := range append(slices.Clone(st.Enemies), st.Party...) {
			fielded[sp.Character]++
		}
	}
	footTerrain := data.Classes[data.Characters[walkers[0]].Class].Terrain
	var foot, horse string
	for _, id := range slices.Sorted(maps.Keys(data.Characters)) {
		c := data.Characters[id]
		cl := data.Classes[c.Class]
		switch {
		case !c.Template || cl.Attack == "ranged":
		case mounted(c.Class) && fielded[id] > fielded[horse]:
			horse = id
		case !mounted(c.Class) && cl.Terrain == footTerrain && fielded[id] > fielded[foot]:
			foot = id
		}
	}
	err := d.battleWith(d.stage, func(doc *mapedit.Document) {
		if err := doc.Resize(mapedit.Rect{X: 0, Y: 0, W: 24, H: 18}); err != nil {
			panic(err)
		}
		G, S, T := byte(content.LayerGround), byte(content.LayerStructure), byte(content.LayerState)
		for y := 0; y < doc.Play.H; y++ {
			for x := 0; x < doc.Play.W; x++ {
				c := byte('g')
				switch {
				case y <= 1 && x >= 2 && x <= 21:
					c = 'w'
				case y >= 2 && y <= 9 && x >= 2 && x <= 21:
					c = 's'
				}
				doc.Paint(data, x, y, S, '-')
				doc.Paint(data, x, y, T, '-')
				doc.Paint(data, x, y, G, c)
			}
		}
		// the stage's own armies, which its victory and defeat name, out of the way in the bottom row
		for i := range doc.Stage.Party {
			doc.Stage.Party[i].X, doc.Stage.Party[i].Y = 23-i, 17
		}
		for i := range doc.Stage.Enemies {
			doc.Stage.Enemies[i].X, doc.Stage.Enemies[i].Y = 23-len(doc.Stage.Party)-i, 17
		}
		for i := range directions {
			doc.Stage.Enemies = append(doc.Stage.Enemies,
				content.Spawn{Character: foot, X: 3 + 2*i, Y: 4, Facing: "SE", Policy: "hold", Group: "wade-foot"},
				content.Spawn{Character: horse, X: 3 + 2*i, Y: 7, Facing: "SE", Policy: "hold", Group: "wade-horse"})
		}
		doc.Stage.Allies = []content.Point{{X: 4, Y: 14}, {X: 10, Y: 14}, {X: 14, Y: 14}, {X: 16, Y: 14}, {X: 18, Y: 14}, {X: 20, Y: 14}}
		var into, out []content.Step
		for k, id := range walkers {
			if id != "" {
				into = append(into, content.Step{Kind: "move", Unit: id, To: &content.Point{X: 4 + 6*k, Y: 5}})
				out = append(out, content.Step{Kind: "move", Unit: id, To: &content.Point{X: 4 + 6*k, Y: 14}})
			}
		}
		rest := []content.Step{{Kind: "wait", Seconds: 1}}
		doc.Stage.Events = []content.BattleEvent{{ID: "wade", If: []string{"turnstart:support"}, Do: []content.Action{{Kind: "cutscene",
			Steps: [][]content.Step{into, rest, out, rest, into, rest}}}}}
	})
	if err != nil {
		return err
	}
	g := d.g
	d.step = func(frame int) {
		if frame == 5 { // the cutscene runs as the allied armies' turn starts
			g.command(core.Command{Kind: "end"})
		}
		for _, u := range g.vis {
			if !u.ally && u.v >= 3 && u.v <= 8 {
				u.dir = directions[int(u.u-3)/2%8]
			}
		}
		if n := (frame - 740) / 12; n >= 0 && n < 16 { // each soldier close up: the foot row, then the riders
			g.zoom = 5
			g.camX, g.camY = g.field.Center(float64(3+2*(n%8)), float64(4+3*(n/8))-.3)
			return
		}
		g.zoom = 3
		g.camX, g.camY = g.field.Center(7, 9)
	}
	return nil
}
