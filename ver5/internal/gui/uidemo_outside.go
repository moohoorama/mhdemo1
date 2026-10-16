package gui

func init() { demos["outside"] = demoOutside }

// demoOutside frames the whole first battle like the mockup's capture, at zoom 1, and
// lets the cloud shadows drift.
func demoOutside(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	o := g.field.outside
	g.camX, g.camY = o.cx, o.cy
	return nil
}
