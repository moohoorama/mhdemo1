package gui

import (
	"fmt"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() {
	demos["insu"] = demoInsu
	demos["gearguide"] = demoGearGuide
}

// demoJump starts at the waypoint node with every learning window closed.
func (d *Demo) demoJump(node string) error {
	g := d.g
	if r := g.request(session.Request{Op: "jump", Seed: 1, Node: node}); !r.OK {
		return fmt.Errorf("jump %s: %s", node, r.Error)
	}
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	cursorPos = func() (int, int) { return -1000, -1000 }
	return nil
}

// firstWalk is the first waypoint that opens on a walk, of the explore file x when x is not "".
func firstWalk(data *content.Data, x string) string {
	for _, w := range data.Waypoints() {
		if n := data.Node(w.Node); n.Kind == "explore" && (x == "" || n.Explore == x) {
			return w.Node
		}
	}
	return ""
}

// pressLabel presses the last button labelled label as a player would.
func (g *Game) pressLabel(label string) {
	for i := len(g.buttons) - 1; i >= 0; i-- {
		if b := g.buttons[i]; b.label == label {
			at := b.rect.Min.Add(b.rect.Size().Div(2))
			cursorPos = func() (int, int) { return at.X, at.Y }
			g.leftClick()
			cursorPos = func() (int, int) { return -1000, -1000 }
			return
		}
	}
}

// demoInsu uses an 인수 outside a battle (-node, default the walk of the gearguide demo's event; a battle node such as
// c14-battle opens the belongings from the sortie preparation's 소지품 at 40): the lord and the first officer a seal
// takes into more than one class (else the lord alone) are raised to the promotion level and the party carries two of
// every seal; the promotion guide rims 관리 in orange (before 60), the belongings show with only what can be used lit
// (60), 소모품 (80), the seal picked (100) and its 승급하기 pressed opens the officer table (115), that officer is
// picked (130), 미리 보기 brings the promotion window with a tab for each class and the traits the class teaches
// (160), the second tab when there are several (190), whose first trait shows its description (220), and 승급하기
// promotes him back on the belongings (260).
func demoInsu(d *Demo) error {
	data := d.g.S.Data
	node := d.node
	if node == "" { // a walk that opens straight on the walk: the smith's
		ev, _ := findSmithyEvent(data)
		node = firstWalk(data, ev.explore)
	}
	if err := d.demoJump(node); err != nil {
		return err
	}
	g := d.g
	lord, who := "", ""
	for _, c := range g.roster(g.S.View()) {
		cl := data.Classes[c.Class]
		if data.Characters[c.Def].Lord {
			lord = c.ID
		}
		if who == "" && cl.Tier < len(data.Rules.PromoteLevels) && len(data.Promotes(c.Def, c.Class)) > 1 {
			who = c.ID
		}
	}
	if who == "" {
		who = lord
	}
	seal := ""
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			c := &s.Characters[i]
			if t := data.Classes[c.Class].Tier; (c.ID == who || c.ID == lord) && t < len(data.Rules.PromoteLevels) {
				c.Level = max(c.Level, data.Rules.PromoteLevels[t])
				if c.ID == who {
					seal = insuFor(data, t)
				}
			}
		}
		for id, it := range data.Items {
			if it.Effect == content.ItemPromote {
				s.Inventory[id] = 2
			}
		}
	})
	d.step = func(f int) {
		switch f {
		case 40:
			if g.S.View().Dialogue.Kind == "battle" {
				g.pressLabel("소지품")
			}
		case 60:
			if g.overlay() == "" {
				g.openOverlay("party")
			}
			g.party.items, g.party.table, g.party.filter = true, false, ""
		case 80:
			g.party.filter = "소모품"
		case 100:
			g.party.item = seal
		case 115:
			g.pressLabel("승급하기")
		case 130:
			g.party.insu.pick = who
		case 160:
			for _, a := range g.insuActs() {
				if a.click != nil {
					a.click()
				}
			}
		case 190:
			if len(g.party.insu.preview) > 1 {
				g.party.insu.at = 1
			}
		case 220:
			if s := g.party.insu; len(s.preview) > 0 {
				if pool := data.Classes[s.preview[s.at].to].Pool; len(pool) > 0 {
					g.modals = append([]*modal{{kind: "trait", trait: pool[0]}}, g.modals...)
				}
			}
		case 240:
			g.closeModal()
		case 260:
			if s := g.party.insu; len(s.preview) > 0 {
				g.useInsu(s.preview[s.at].cmd)
			}
		}
	}
	return nil
}

// smithyEvent is the first event of a walk that hands over a piece with an Owner: its explore file, area, the person
// who starts it and what must hold for it, and the pieces.
type smithyEvent struct {
	explore, area, person string
	needs, pieces         []string
}

func findSmithyEvent(data *content.Data) (smithyEvent, bool) {
	for _, xid := range sorted(data.Explores) {
		x := data.Explores[xid]
		for _, aid := range sorted(x.Areas) {
			for _, key := range sorted(x.Areas[aid].People) {
				for _, ev := range x.Areas[aid].People[key].Event {
					var pieces []string
					for id := ev.Node; id != ""; id = data.After(id) {
						n := data.Node(id)
						if n.Kind == "back" {
							break
						}
						if n.Kind == "grant" && data.Equipment[n.Item].Owner != "" {
							pieces = append(pieces, n.Item)
						}
					}
					if len(pieces) > 0 {
						return smithyEvent{xid, aid, key, ev.If, pieces}, true
					}
				}
			}
		}
	}
	return smithyEvent{}, false
}

// demoGearGuide plays the first walk event that hands over pieces with an Owner (chapter 2's smith and his two
// weapons) at its walk's waypoint, the pieces first taken out of the warehouse, and follows the green guide to
// wearing them: 관리 (90), the lord's row picked and the owner's row lit (130), the 장비 tab (160), the slot's 바꾸기
// (190), the piece in the picker (230), the button that puts it on (260), 닫기 for the next owner's piece (290), that
// owner's row (320), and out of the area, no guide (380).
func demoGearGuide(d *Demo) error {
	data := d.g.S.Data
	ev, ok := findSmithyEvent(data)
	if !ok {
		return fmt.Errorf("gearguide demo: no walk event hands over an owned piece")
	}
	if err := d.demoJump(firstWalk(data, ev.explore)); err != nil {
		return err
	}
	g := d.g
	d.edit(func(s *core.State) {
		for _, c := range ev.needs {
			if f, ok := strings.CutPrefix(c, "flag:"); ok {
				s.Flags[f] = 1
			} else if f, ok := strings.CutPrefix(c, "!flag:"); ok {
				delete(s.Flags, f)
			}
		}
		s.Spot = data.Explores[ev.explore].Arrival(ev.area, "")
		for _, id := range ev.pieces {
			s.Warehouse[id] = 0
		}
	})
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "talk", Target: ev.person}})
	for i := 0; !g.S.Engine.Walking() && i < 50; i++ {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
	}
	g.modals, g.held = nil, nil
	p := &g.party
	lord := ""
	for _, c := range g.roster(g.S.View()) {
		if data.Characters[c.Def].Lord {
			lord = c.ID
		}
	}
	d.step = func(f int) {
		piece, owner, slot, _ := g.gearStep()
		switch f {
		case 100:
			p.items, p.table, p.tab = false, false, 0
			g.selected = lord
			g.openOverlay("party")
		case 140:
			g.selected = owner
		case 170:
			p.tab = 2
		case 200:
			g.openOverlay("equipment")
			g.openPicker(slot)
		case 240:
			g.equip.pick = piece
		case 270:
			g.command(core.Command{Kind: "equip", Actor: owner, Item: piece})
			g.equip.pick = equipCurrent
		case 300:
			g.closeOverlay()
		case 340:
			g.closeOverlays()
			d.edit(func(s *core.State) {
				s.Spot = g.S.Engine.ExploreFile().Arrival(g.S.Engine.ExploreFile().Start.Area, "")
			})
		}
	}
	return nil
}
