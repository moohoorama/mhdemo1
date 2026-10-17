package gui

import "srpg/internal/core"

func init() { demos["hudpanel"] = demoHUDPanel }

// demoHUDPanel walks the battle HUD panel: the minimap (frame 20), the log (60), the unit list
// (100) sorted by orders (130), its stats view by attack (160), the biography view of all
// units (200) and the other tab (240), an officer selected (280) with its tabs, a peek at the
// minimap while selected (400), back to the base tab on letting go (440), and the enemy's turn (480).
func demoHUDPanel(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	ally := d.firstAlly()
	d.look(ally)
	g.logPending = []string{"유비 → 장각: 일반 공격", "유비 → 장각: 피해 320", "장각 → 유비: 반격"}
	d.step = func(f int) {
		switch f {
		case 1, 2, 3:
			g.publishBattleLog()
		case 60:
			g.hudBase = "log"
		case 100:
			g.hudBase = "units"
		case 130:
			g.unitOrder = g.unitOrder.cycle("order").cycle("order")
		case 160:
			g.unitKind, g.unitOrder = 1, unitSort{"attack", 2}
		case 200:
			g.unitKind, g.unitFilter, g.unitOrder = 2, "all", unitSort{"attr무력", 2}
		case 240:
			g.hudBase = "misc"
		case 280:
			g.hudBase, g.selected = "units", ally
			g.detailTab = 0
		case 320:
			g.detailTab = 1
		case 360:
			g.detailTab = 3
		case 400:
			g.hudPeek = "minimap"
		case 440:
			g.selected, g.hudPeek = "", ""
		case 480:
			g.S.Engine.Apply(core.Command{Kind: "end"})
		}
	}
	return nil
}
