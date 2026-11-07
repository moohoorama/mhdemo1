package gui

import "maps"

func init() {
	demos["zz-julu"] = func(d *Demo) error {
		g := d.g
		g.S.Data.Characters = maps.Clone(g.S.Data.Characters)
		for id, c := range g.S.Data.Characters {
			if c.Playable {
				c.Level = 15
				g.S.Data.Characters[id] = c
			}
		}
		if err := d.battle("B09"); err != nil {
			return err
		}
		held := 0
		d.step = func(f int) {
			// a level-up card stays a second on screen, then goes as a click would take it
			if len(g.modals) > 0 {
				if held++; held > 60 {
					g.closeModal()
					held = 0
				}
				return
			}
			if f == 1 {
				g.autoplay, g.speed = true, 4
			}
			// the editor's trial battle is not the "playing" screen whose loop runs autoplay
			if f%12 == 0 && g.overlay() == "" && len(g.modals) == 0 {
				g.assist()
			}
		}
		return nil
	}
}
