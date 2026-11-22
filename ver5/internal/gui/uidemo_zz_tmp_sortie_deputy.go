package gui

import (
	"fmt"
	"slices"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["sortie-deputy"] = demoSortieDeputy }

// demoSortieDeputy jumps to the -node battle node (c20-battle has the party of the phone report: 장비, 유비, 손건,
// 미축, 번궁, 번아 out, 진도, 고순 waiting), spends everyone's trait points, teaches 장비 (or the first officer out besides the lord) to name a
// deputy, names 미방 (or the next officer) and walks the sortie's 부관 tab for cmd/_uidemo-vm: the table by weight
// with the deputy now on top in blue (1), 진도 picked in gold (40), sorted by 매력 (60), every trait of the pick
// (75), a trait's description (90), the deputy's own row picked in the list (120, basic at 124), the lead's 장비 (130), 특성 (136)
// and 열전 (142) tabs, the deputy window (150) and the officers' window with the deputy plates (190).
func demoSortieDeputy(d *Demo) error {
	g := d.g
	if d.node == "" {
		return fmt.Errorf("sortie-deputy: pass the battle node to open with -node")
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: d.node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	o := g.S.View()
	var others []string
	for _, id := range o.Deployment {
		if !g.S.Data.Characters[id].Lord {
			others = append(others, id)
		}
	}
	if len(others) < 2 {
		return fmt.Errorf("sortie-deputy: %s has fewer than two officers besides the lord", d.node)
	}
	lead := others[0]
	dep := others[len(others)-1]
	for _, c := range o.Characters { // a deputy from the bench when there is one, as in a real preparation
		if c.ID != lead && !c.Dead && !g.S.Data.Characters[c.ID].Lord && !slices.Contains(o.Deployment, c.ID) {
			dep = c.ID
			break
		}
	}
	look := ""
	for _, c := range o.Characters {
		if c.ID != lead && c.ID != dep && !c.Dead && !g.S.Data.Characters[c.ID].Lord {
			look = c.ID
		}
	}
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: lead, Trait: "deputy_appoint"}})
	if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "deputy", Actor: lead, Target: dep}}); !r.OK {
		return fmt.Errorf("sortie-deputy: naming %s the deputy of %s failed", dep, lead)
	}
	for _, c := range g.roster(g.S.View()) { // spend the points as a player would, so the deputies bring traits
		for n := 0; n < 6; n++ {
			learnable := g.S.Engine.Learnable(c.ID)
			if len(learnable) == 0 {
				break
			}
			g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: c.ID, Trait: learnable[0]}})
		}
	}
	cursorPos = func() (int, int) { return -1000, -1000 }
	d.step = func(f int) {
		p := &g.party
		switch f {
		case 1:
			g.selected, p.sortieView = lead, 5
		case 40:
			g.deputyPick = look
		case 60:
			p.deputies.order = tableSort{"charm", 2}
		case 75:
			p.depAll = true
		case 90:
			p.depAll = false
			if ts := g.S.Engine.DeputyTraits(g.deputyPick); len(ts) > 0 {
				g.deputyInfo = equipChip{"deputy", ts[0]}
			}
		case 101:
			g.deputyInfo = equipChip{}
		case 110:
			g.selected, p.sortieView = lead, 1
		case 114:
			o := g.S.View()
			if c := g.officer(o, "zhaoyun"); c != nil && g.mountOf(*c) == "baima" && !slices.Contains(o.Deployment, c.ID) && g.servedBy(o, c.ID) == "" {
				ids := slices.Clone(o.Deployment)
				for i, id := range ids {
					if id != lead && !g.sortieLocked(id) {
						ids[i] = c.ID
						g.request(session.Request{Op: "command", Command: core.Command{Kind: "deploy", Deployment: ids}})
						break
					}
				}
			}
		case 120:
			g.selected, p.sortieView = dep, 2
		case 124:
			p.sortieView = 0
		case 130:
			g.selected, p.sortieView = lead, 3
		case 136:
			p.sortieView = 4
		case 142:
			p.sortieView = 2
		case 150:
			g.selected = lead
			g.openOverlay("deputy")
			g.deputyPick = look
		case 190:
			g.closeOverlay()
			p.items, p.table = false, false
			g.selected = dep
			g.openOverlay("party")
		}
	}
	return nil
}
