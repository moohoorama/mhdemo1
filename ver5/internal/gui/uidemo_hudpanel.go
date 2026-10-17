package gui

func init() { demos["hudpanel"] = demoHUDPanel }

// demoHUDPanel selects an ally and walks the unit panel's tabs.
func demoHUDPanel(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.selected = d.firstAlly()
	d.look(g.selected)
	d.step = func(f int) {
		if f > 0 && f%30 == 0 {
			g.detailTab, g.detailPage = f/30%4, 0
		}
	}
	return nil
}
