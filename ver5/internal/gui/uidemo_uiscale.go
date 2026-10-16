package gui

func init() { demos["uiscale"] = demoUIScale }

// demoUIScale plays the menu demo, then steps the UI size through 80, 100, 130 and 160%
// with the skill list open.
func demoUIScale(d *Demo) error {
	if err := demoMenu(d); err != nil {
		return err
	}
	menu := d.step
	d.step = func(f int) {
		if f <= 240 {
			menu(f)
		}
		switch f {
		case 250:
			d.g.S.Settings.UIScale = 80
		case 280:
			d.g.S.Settings.UIScale = 100
		case 310:
			d.g.S.Settings.UIScale = 130
		case 340:
			d.g.S.Settings.UIScale = 160
		}
	}
	return nil
}
