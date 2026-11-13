package gui

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

func init() { demos["party"] = demoParty }

// demoParty opens the party window on a new game's first walk with a few things to carry (the first shop item of
// each kind: an HP and an MP item, a promotion manual, a weapon and an accessory): the officers (frame 160), the
// belongings (220), the equipment alone (280) and a worn piece picked (340).
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
			items := map[string]int{content.ItemHP: 6, content.ItemMP: 3, content.ItemPromote: 1}
			slots := map[string]bool{"weapon": true, "accessory": true}
			d.edit(func(s *core.State) {
				for _, id := range sorted(g.S.Data.Items) {
					if it := g.S.Data.Items[id]; items[it.Effect] > 0 && (it.Price > 0 || it.Effect == content.ItemPromote) {
						s.Inventory[id] += items[it.Effect]
						items[it.Effect] = 0
					}
				}
				for _, id := range sorted(g.S.Data.Equipment) {
					if eq := g.S.Data.Equipment[id]; slots[eq.Slot] && eq.Price > 0 {
						s.Warehouse[id]++
						slots[eq.Slot] = false
					}
				}
			})
		case 130:
			g.openOverlay("party")
		case 145, 160, 175:
			g.party.tab = (f - 145) / 15
		case 190:
			g.party.table = true
		case 205:
			g.party.table, g.party.items = false, true
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
