package gui

import "srpg/internal/session"

func init() { demos["dialogue"] = demoDialogue }

// demoDialogue starts a new game and steps through the opening story lines, so the
// scroll opens, types, and springs to each next line's width.
func demoDialogue(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	d.step = func(f int) {
		if f > 0 && f%150 == 0 {
			g.advanceDialogue()
			g.advanceDialogue()
		}
	}
	return nil
}
