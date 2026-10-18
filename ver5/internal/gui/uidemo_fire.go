package gui

import "srpg/internal/core"

func init() { demos["fire"] = demoFire }

// demoFire sets a patch of burning ground and a wall of flames on the battlefield, in front of the
// first ally, to see how fire reads on the map.
func demoFire(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	ally := d.firstAlly()
	d.look(ally)
	snap := g.S.Engine.Snapshot()
	var x, y int
	for _, u := range snap.Units {
		if u.ID == ally {
			x, y = u.X, u.Y
		}
	}
	stage := g.S.Engine.Data().Stages[snap.Stage]
	light := func(x, y int, state string) {
		if x >= 0 && y >= 0 && x < stage.Width && y < stage.Height {
			snap.TileStates = append(snap.TileStates, core.TileState{X: x, Y: y, State: state})
		}
	}
	for dx := 2; dx <= 4; dx++ {
		for dy := -1; dy <= 1; dy++ {
			light(x+dx, y+dy, "e")
		}
	}
	for dy := -2; dy <= 2; dy++ {
		light(x+5, y+dy, "F")
	}
	e, err := core.Restore(g.S.Engine.Data(), snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.ensureField(e.Observe())
	g.sync(e.Observe())
	return nil
}
