package gui

import (
	"sort"

	"srpg/internal/session"
)

func init() { demos["popups"] = demoPopups }

// demoPopups walks the windows and screens laid out on the Width×Height design, for
// cmd/_uidemo-vm -sheet: the title (frame 30) with its settings (90), then in battle the
// unit list (180), load slots (240), menu (300), the objective (360) and a trait (420)
// modal, and a new game's sortie preparation (540).
func demoPopups(d *Demo) error {
	g := d.g
	open := func(name string) {
		g.overlay = ""
		g.openOverlay(name)
	}
	traits := make([]string, 0, len(g.S.Data.Traits))
	for id := range g.S.Data.Traits {
		traits = append(traits, id)
	}
	sort.Strings(traits)
	d.step = func(f int) {
		switch f {
		case 60:
			open("settings")
		case 120:
			g.overlay = ""
			if err := d.battle(d.stage); err != nil {
				d.err = err
			}
		case 150:
			open("units")
		case 210:
			open("load")
		case 270:
			open("menu")
		case 330:
			g.overlay = ""
			g.modals = []*modal{{kind: "objective"}}
		case 390:
			if len(traits) > 0 {
				g.modals = []*modal{{kind: "trait", trait: traits[0]}}
			}
		case 450:
			g.modals = nil
			g.request(session.Request{Op: "new", Seed: 1})
		}
	}
	return nil
}
