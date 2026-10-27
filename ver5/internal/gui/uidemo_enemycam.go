package gui

import "srpg/internal/core"

func init() { demos["enemycam"] = demoEnemyCam }

// demoEnemyCam hands the turn to the enemy with the camera on the allies and plays the
// enemy phase; the camera follows the middle of each group that moves together.
func demoEnemyCam(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	d.step = func(f int) {
		switch f {
		case 1:
			for _, u := range g.S.View().UnitViews {
				if u.Faction == "ally" {
					d.look(u.ID)
					break
				}
			}
		case 20:
			d.edit(func(s *core.State) {
				s.Turn = "enemy"
				for j := range s.Units {
					if s.Units[j].Faction == "ally" {
						s.Units[j].Moved, s.Units[j].Acted, s.Units[j].Done = true, true, true
					}
				}
			})
			g.resolvePhase()
		}
	}
	return nil
}
