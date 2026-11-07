package gui

import (
	"slices"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["sortie"] = demoSortie }

// demoSortie jumps to the -node battle node (default c14-battle) with every officer's level-up learning closed,
// names 손건 고순's deputy and walks the sortie preparation for cmd/_uidemo-vm: the list with the lord picked (20),
// 고순 with his deputy (50), the last officer called back to wait (80), the deputy window (110), the officer's
// trait tab (140), the unit detail on 전투력 (155), 열전 (180) and the deputy's traits (190) and, after 출진, the deployment (215).
func demoSortie(d *Demo) error {
	g := d.g
	node := d.node
	if node == "" {
		node = "c14-battle"
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: "gaoshun", Trait: "deputy_appoint"}})
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "deputy", Actor: "gaoshun", Target: "sunqian"}})
	cursorPos = func() (int, int) { return -1000, -1000 }
	d.step = func(f int) {
		o := g.S.View()
		switch f {
		case 50:
			g.selected = "gaoshun"
		case 80:
			if n := len(o.Deployment); n > 0 && !g.sortieLocked(o.Deployment[n-1]) {
				g.togglePrep(o.Deployment[n-1])
			}
		case 110:
			g.selected = "gaoshun"
			g.openOverlay("deputy")
		case 125:
			g.closeOverlay()
		case 140:
			g.party.tab = 1
			g.openOverlay("party")
		case 155:
			g.closeOverlay()
			g.selected = "gaoshun"
			g.openUnit(false)
		case 180:
			g.unitWin.tab = 2
		case 190:
			g.unitWin.tab, g.unitWin.side = 1, 1
		case 200:
			g.closeOverlay()
		case 210:
			if slices.Contains(o.Deployment, "liubei") {
				g.startSortie()
			}
		}
	}
	return nil
}
