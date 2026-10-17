package gui

import "srpg/internal/session"

func init() { demos["uiscale"] = demoUIScale }

// uiScaleBattleAt is the frame demoUIScale leaves the opening dialogue for the battle.
const uiScaleBattleAt = 480

// demoUIScale walks the screens a UI size change has to fit, for cmd/_uidemo-vm -uiscale and
// -sheet: a new game's opening (장비's first-appearance card around frame 260, the dialogue
// scroll around 380), then from frame 480 the battle HUD (top bar, dock, log, minimap, detail panel) with
// the command menu and its skill list open (around 720).
func demoUIScale(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	var menu func(int)
	var step func(int)
	step = func(f int) {
		switch {
		case f < uiScaleBattleAt:
			if f > 0 && f%150 == 0 {
				g.advanceDialogue()
				g.advanceDialogue()
			}
		case f == uiScaleBattleAt:
			if err := demoMenu(d); err != nil {
				d.err = err
				return
			}
			menu, d.step = d.step, step
			menu(0)
		default:
			menu(f - uiScaleBattleAt)
			if f == uiScaleBattleAt+21 {
				g.selected = g.ord.actor // the detail panel too
			}
		}
	}
	d.step = step
	return nil
}
