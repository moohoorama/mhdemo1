package gui

func init() { demos["turn"] = demoTurn }

// demoTurn plays the allied phase banner, then the enemy one.
func demoTurn(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	d.step = func(f int) {
		switch f {
		case 6:
			d.g.showTurn(8, "ally")
		case 180:
			d.g.showTurn(7, "enemy")
		}
	}
	return nil
}
