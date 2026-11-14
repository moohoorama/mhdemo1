package gui

import (
	"fmt"
	"slices"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["sortie"] = demoSortie }

// demoSortie jumps to the -node battle node with every officer's level-up learning closed, names the second
// officer of the sortie who is not the lord the first one's deputy (c14-battle: 손건 under 고순) and walks the sortie preparation for cmd/_uidemo-vm: the list with the lord picked (20),
// 고순 with his deputy (50), the last officer called back to wait by a press on his 출전 pill (80), the deputy window (110), the officer's
// trait tab (140), the unit detail on 전투력 (155), 열전 (180) and the deputy's traits (190) and, after 출진, the deployment (215).
func demoSortie(d *Demo) error {
	g := d.g
	if d.node == "" {
		return fmt.Errorf("sortie: pass the battle node to open with -node")
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: d.node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	var lord string
	var others []string
	for _, id := range g.S.View().Deployment {
		if g.S.Data.Characters[id].Lord {
			lord = id
		} else {
			others = append(others, id)
		}
	}
	if len(others) < 2 {
		return fmt.Errorf("sortie: %s has fewer than two officers besides the lord", d.node)
	}
	head := others[0]
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: head, Trait: "deputy_appoint"}})
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "deputy", Actor: head, Target: others[1]}})
	cursorPos = func() (int, int) { return -1000, -1000 }
	d.step = func(f int) {
		o := g.S.View()
		switch f {
		case 50:
			g.selected = head
		case 80: // press the last 출전 pill as a player would, so the row under it cannot steal the press
			for i := len(g.buttons) - 1; i >= 0; i-- {
				if b := g.buttons[i]; b.label == "출전" {
					at := b.rect.Min.Add(b.rect.Size().Div(2))
					cursorPos = func() (int, int) { return at.X, at.Y }
					g.leftClick()
					cursorPos = func() (int, int) { return -1000, -1000 }
					break
				}
			}
		case 110:
			g.selected = head
			g.openOverlay("deputy")
		case 125:
			g.closeOverlay()
		case 140:
			g.party.tab = 1
			g.openOverlay("party")
		case 155:
			g.closeOverlay()
			g.selected = head
			g.openUnit(false)
		case 180:
			g.unitWin.tab = 2
		case 190:
			g.unitWin.tab, g.unitWin.side = 1, 1
		case 200:
			g.closeOverlay()
		case 210:
			if slices.Contains(o.Deployment, lord) {
				g.startSortie()
			}
		}
	}
	return nil
}
