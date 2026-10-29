package gui

func init() { demos["settings"] = demoSettings }

// demoSettings opens the settings window over the start screen.
func demoSettings(d *Demo) error {
	d.g.openOverlay("settings")
	return nil
}
