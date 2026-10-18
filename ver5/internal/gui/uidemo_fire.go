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
	for dx := 2; dx <= 4; dx++ {
		for dy := -1; dy <= 1; dy++ {
			snap.TileOverrides = append(snap.TileOverrides, core.TileOverride{X: x + dx, Y: y + dy, Tile: "F"})
		}
	}
	for dy := -2; dy <= 2; dy++ {
		snap.TileOverrides = append(snap.TileOverrides, core.TileOverride{X: x + 5, Y: y + dy, Tile: "W"})
	}
	e, err := core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.ensureField(e.Observe())
	g.sync(e.Observe())
	return nil
}
