package gui

import "srpg/internal/session"

func init() { demos["intro"] = demoIntro }

// demoIntro starts a new game and plays the opening scene until 관우 and 장비 are
// introduced.
func demoIntro(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	d.step = func(f int) {
		if g.intro == nil && !g.sceneBusy() && f%30 == 0 {
			g.advanceDialogue()
		}
	}
	return nil
}
