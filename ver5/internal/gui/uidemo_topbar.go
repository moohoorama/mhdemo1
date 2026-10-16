package gui

import "srpg/internal/core"

func init() { demos["topbar"] = demoTopBar }

// demoTopBar unrolls the objectives, selects each officer in turn and marks them done,
// then hands the turn to the enemy and starts the next round.
func demoTopBar(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	allies := []string{}
	for _, u := range g.S.Engine.Observe().UnitViews {
		if u.Faction == "ally" && !u.AI {
			allies = append(allies, u.ID)
		}
	}
	d.step = func(f int) {
		for i, id := range allies {
			switch f {
			case 70 + i*60:
				g.selected = id
			case 110 + i*60:
				d.edit(func(s *core.State) {
					for j := range s.Units {
						if s.Units[j].ID == id {
							s.Units[j].Moved, s.Units[j].Acted, s.Units[j].Done = true, true, true
						}
					}
				})
				g.selected = ""
			}
		}
		switch f {
		case 70 + len(allies)*60 + 30:
			d.edit(func(s *core.State) { s.Turn = "enemy" })
		case 70 + len(allies)*60 + 90:
			d.edit(func(s *core.State) {
				s.Turn, s.Round = "ally", s.Round+1
				for j := range s.Units {
					s.Units[j].Moved, s.Units[j].Acted, s.Units[j].Done = false, false, false
				}
			})
		}
	}
	return nil
}
