package gui

func init() { demos["goals"] = demoGoals }

// demoGoals shows the objective guide on a stage, the camera on the middle of its marks, and switches it
// off at frame 150.
func demoGoals(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	d.step = func(f int) {
		var su, sv, n float64
		for _, s := range g.goalShapes(g.view()) {
			for _, c := range s.cells {
				s.u, s.v = float64(c.X), float64(c.Y)
			}
			su, sv, n = su+s.u, sv+s.v, n+1
		}
		g.zoom = 2
		if n > 0 {
			g.camX, g.camY = g.field.Center(su/n, sv/n)
		}
		if f == 150 {
			g.S.Settings.NoGoalGuide = true
		}
	}
	return nil
}
