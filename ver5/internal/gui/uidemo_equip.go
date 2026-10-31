package gui

import (
	"os"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["equip"] = demoEquip }

// demoEquip walks the equipment window on a new game's sortie preparation with every piece in
// the warehouse: the lord's slot cards (frame 100), the picker of slot EQUIP_SLOT (default
// weapon, 160), the first piece there with a trait picked (200), the list scrolled (240), that trait's description (280) and the
// cards after 확인 (330).
func demoEquip(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	pick, slot := "", "weapon"
	if s := os.Getenv("EQUIP_SLOT"); s != "" {
		slot = s
	}
	d.step = func(f int) {
		switch f {
		case 30:
			for i := 0; i < 200 && g.S.View().Dialogue.Kind != "battle"; i++ {
				g.command(core.Command{Kind: "next"})
			}
			for _, c := range g.roster(g.S.View()) {
				if g.S.Data.Characters[c.Def].Lord {
					g.selected = c.ID
				}
			}
			d.edit(func(s *core.State) {
				if s.Explore != "" {
					s.Node = s.Explore // past the walk's opening event, where pieces change
				}
				for id := range g.S.Data.Equipment {
					s.Warehouse[id]++
				}
				for _, c := range s.Characters {
					if c.ID == g.selected {
						s.Warehouse[c.Equipment["weapon"]]++
					}
				}
			})
		case 60:
			g.openOverlay("equipment")
		case 120:
			g.openPicker(slot)
		case 170:
			for _, id := range sorted(g.S.View().Warehouse) {
				eq := g.S.Data.Equipment[id]
				if pick == "" && eq.Slot == slot && len(eq.Traits) > 0 && g.itemEffect(g.selected, slot, id).err == "" {
					pick = id
				}
			}
			g.equip.pick = pick
		case 210:
			g.equip.list.top = 5
		case 250:
			if pick != "" {
				g.equip.info = equipChip{"trait", g.S.Data.Equipment[pick].Traits[0]}
			}
		case 290:
			g.equip.info = equipChip{}
			if pick != "" {
				g.command(core.Command{Kind: "equip", Actor: g.selected, Item: pick})
			}
			g.equip.slot = ""
		}
	}
	return nil
}
