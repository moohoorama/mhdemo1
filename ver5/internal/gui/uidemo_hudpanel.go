package gui

import "srpg/internal/core"

func init() { demos["hudpanel"] = demoHUDPanel }

// demoHUDPanel walks the battle HUD panel: the minimap (frame 20), the log (60) and the other
// tab (100), an officer selected (140) with its tabs, a peek at the minimap while selected (260),
// back to the base tab on letting go (300), and the enemy's turn (340).
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
			g.hudBase = "misc"
		case 140:
			g.selected = ally
			g.detailTab = 0
		case 180:
			g.detailTab = 1
		case 220:
			g.detailTab = 3
		case 260:
			g.hudPeek = "minimap"
		case 300:
			g.selected, g.hudPeek = "", ""
		case 340:
			g.S.Engine.Apply(core.Command{Kind: "end"})
		}
	}
	return nil
}
