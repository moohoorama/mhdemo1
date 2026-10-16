package gui

func init() { demos["tagfocus"] = demoTagFocus }

// demoTagFocus starts away from the allies and presses each name tag in turn: the camera
// glides to the officer and selects it.
func demoTagFocus(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	size := g.field.Size()
	g.camX, g.camY = float64(size.X)*.8, float64(size.Y)*.8
	names := []string{}
	for _, u := range g.S.Engine.Observe().UnitViews {
		if u.Faction == "ally" && !u.AI {
			names = append(names, g.name(u.ID))
		}
	}
	d.step = func(f int) {
		for i, n := range names {
			if f == 30+i*70 {
				for _, b := range g.buttons {
					if b.label == n && b.rect.Min.Y == 0 {
						b.click()
					}
				}
			}
		}
	}
	return nil
}
