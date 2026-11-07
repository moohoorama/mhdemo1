package gui

import "srpg/internal/core"

func init() { demos["hudpanel"] = demoHUDPanel }

// demoHUDPanel walks the battle HUD panel: the minimap (frame 20), the log (60), the unit list
// (100) sorted by orders (130), all units by troops (160), the roster window (190) with its stats view
// by attack (205) and the biography view of all units (220), and the other tab (240), an officer selected (280) in short, its detail opened to look at (320), a peek at the
// minimap while selected (400), back to the base tab on letting go (440), the brief card over a
// unit (480) and over bare ground (520), the objective window (600), the panel opaque under the pointer (700), folded (740), faded again and unfolded (800), and the enemy's turn (860).
func demoHUDPanel(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	ally := d.firstAlly()
	d.look(ally)
	g.logPending = []string{"유비 → 장각: 일반 공격", "유비 → 장각: 피해 320", "장각 → 유비: 반격"}
	var u core.UnitView
	for _, v := range g.S.View().UnitViews {
		if v.ID == ally {
			u = v
		}
	}
	// at points the cursor at a cell
	at := func(cu, cv int) {
		g.hover = hoverCell{cu, cv, true}
		x, y := g.field.Center(float64(cu), float64(cv))
		sx, sy := worldToUI(g.toScreen(x, y))
		cursorPos = func() (int, int) { return int(sx), int(sy) }
	}
	d.step = func(f int) {
		switch f {
		case 1, 2, 3:
			g.publishBattleLog()
		case 60:
			g.hudBase = "log"
		case 100:
			g.hudBase = "units"
		case 130:
			g.unitsList.order = tableSort{"order", 2}
		case 160:
			g.unitsList.filter, g.unitsList.order = "all", tableSort{"hp", 2}
		case 190:
			g.overlays = []string{"roster"}
		case 205:
			g.rosterList.kind, g.rosterList.order = 1, tableSort{"attack", 2}
		case 220:
			g.rosterList.kind, g.rosterList.filter, g.rosterList.order = 2, "all", tableSort{"attr무력", 2}
		case 235:
			g.closeOverlays()
		case 240:
			g.hudBase = "misc"
		case 280:
			g.hudBase, g.selected = "units", ally
		case 320:
			g.openUnit(true)
		case 360:
			g.closeOverlay()
		case 400:
			g.hudPeek = "minimap"
		case 440:
			g.selected, g.hudPeek = "", ""
		case 480:
			at(u.X, u.Y)
		case 520:
			at(u.X, u.Y+3)
		case 600:
			g.modals = []*modal{{kind: "objective", manual: true}}
		case 660:
			g.modals = nil
		case 700:
			g.hover = hoverCell{}
			cursorPos = func() (int, int) { return uiW - 150, 200 }
		case 740:
			g.hudFolded = true
		case 800:
			cursorPos = func() (int, int) { return -100, -100 }
			g.hudFolded = false
		case 860:
			cursorPos, g.hover = func() (int, int) { return -100, -100 }, hoverCell{}
			g.S.Engine.Apply(core.Command{Kind: "end"})
		}
	}
	return nil
}
