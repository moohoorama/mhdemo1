package gui

import "srpg/internal/core"

func init() { demos["party"] = demoParty }

// demoParty opens the party window on a new game's first walk with a few things to carry: the officers (frame
// 160), the belongings (220), the equipment alone (280) and a worn piece picked (340).
func demoParty(d *Demo) error {
	d.stage = "exit:주점"
	if err := demoExplore(d); err != nil {
		return err
	}
	g, walk := d.g, d.step
	d.step = func(f int) {
		if f < 100 {
			walk(f)
			return
		}
		switch f {
		case 100:
			d.edit(func(s *core.State) {
				s.Inventory["bean"] += 6
				s.Inventory["liquor"] += 3
				s.Inventory["promote_heavy_infantry"]++
				s.Warehouse["iron_sword"]++
				s.Warehouse["war_horse"]++
			})
		case 130:
			g.openOverlay("party")
		case 190:
			g.party.items = true
		case 250:
			g.party.filter = "무기"
		case 310:
			for _, b := range g.belongings(g.S.View()) {
				if len(b.wearer) > 0 && g.belongingKind(b.id) == "무기" {
					g.party.item = b.id
				}
			}
		}
	}
	return nil
}
