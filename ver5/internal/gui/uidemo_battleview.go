package gui

func init() { demos["battleview"] = demoBattleView }

// demoBattleView opens a deployed battle and frames the entire playable area.
func demoBattleView(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	d.step = func(int) {
		u := float64(g.field.Map.W-1) / 2
		v := float64(g.field.Map.H-1) / 2
		g.setZoomImmediate(g.zoomSteps()[0])
		g.camX, g.camY = g.field.Center(u, v)
	}
	return nil
}
