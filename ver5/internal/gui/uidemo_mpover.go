package gui

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

func init() { demos["mpover"] = demoMPOver }

// demoMPOver has the first ally, its spell points full, drink a spell point drink at frame 20, so they pass the max:
// the map bar's MP line and, once the officer is selected (120), the HUD's 병법치 gauge pulse once every two seconds.
func demoMPOver(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	ally := d.firstAlly()
	d.look(ally)
	full := 0
	for _, u := range g.S.View().UnitViews {
		if u.ID == ally {
			full = u.Stats.MaxMP
		}
	}
	drink := ""
	for _, id := range sorted(g.S.Data.Items) {
		if g.S.Data.Items[id].Effect == content.ItemMP && drink == "" {
			drink = id
		}
	}
	d.edit(func(s *core.State) {
		s.Inventory[drink] = 3
		for i := range s.Units {
			if s.Units[i].ID == ally {
				s.Units[i].MP = full
			}
		}
	})
	d.step = func(f int) {
		switch f {
		case 20:
			g.command(core.Command{Kind: "item", Actor: ally, Target: ally, Item: drink})
		case 120:
			g.hudBase, g.selected = "units", ally
		}
	}
	return nil
}
