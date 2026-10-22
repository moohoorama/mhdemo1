package gui

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

func init() { demos["flee"] = demoFlee }

// demoFlee turns four AI allies into fleeing units a few cells from the east edge and plays their
// turn (-stage B04 has AI allies): each walks to the goal on the edge and walks off the battle area, darkening and fading.
func demoFlee(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	goal := content.Point{X: g.field.Map.W - 1, Y: 1}
	d.step = func(f int) {
		switch f {
		case 1:
			d.edit(func(s *core.State) {
				n := 0
				for j := range s.Units {
					u := &s.Units[j]
					if !u.AI || u.Faction != "ally" || u.HP <= 0 {
						continue
					}
					if n >= 4 { // the rest of the army stands still
						u.Moved, u.Acted, u.Done = true, true, true
						continue
					}
					u.Policy, u.Goal = "flee", &goal
					u.X, u.Y = goal.X-3+n%2, goal.Y+n
					s.Turn, s.Army = "support", u.Color
					n++
				}
			})
			g.camX, g.camY = g.field.Center(float64(goal.X-1), float64(goal.Y+1))
		case 20:
			g.resolvePhase()
		}
	}
	return nil
}
