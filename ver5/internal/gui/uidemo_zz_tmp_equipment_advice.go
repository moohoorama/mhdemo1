package gui

import "srpg/internal/core"

func init() { demos["equipment-advice"] = demoEquipmentAdvice }

// demoEquipmentAdvice visits the sortie list, its gear tab and picker, then the party list before and after equipping.
func demoEquipmentAdvice(d *Demo) error {
	if err := d.demoJump("c14-battle"); err != nil {
		return err
	}
	const actor, piece = "liubei", "fire_manual"
	g := d.g
	g.selected = actor
	d.edit(func(s *core.State) {
		s.Warehouse = map[string]int{piece: 1}
		for _, c := range s.Characters {
			if c.ID == actor {
				delete(c.Equipment, "accessory")
			}
		}
	})
	d.step = func(f int) {
		switch f {
		case 30:
			g.party.sortieView = 3
		case 60:
			g.openOverlay("equipment")
			g.openPicker("accessory")
		case 70:
			g.equip.pick = piece
		case 90:
			g.closeOverlay()
			g.openOverlay("party")
		case 120:
			g.command(core.Command{Kind: "equip", Actor: actor, Item: piece})
		}
	}
	return d.err
}
