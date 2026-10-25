package gui

func init() { demos["title"] = demoTitle }

// demoTitle stays on the start screen, where the dragon pattern drifts.
func demoTitle(d *Demo) error { return nil }
