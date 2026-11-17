package gui

import (
	"fmt"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["officercards"] = demoOfficerCards }

// demoOfficerCards jumps to the -node battle node (c14-battle by default), dresses the lord in the gear his class
// carries with the most grades and traits in every slot and shows, in the sortie preparation, his 열전 (frame 20)
// and 장비 (40) tabs, then the same two tabs in the officers' window (60, 80), for checking the cards on a phone.
func demoOfficerCards(d *Demo) error {
	g := d.g
	if d.node == "" {
		d.node = "c14-battle"
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: d.node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	pick := ""
	for _, id := range g.S.View().Deployment {
		if g.S.Data.Characters[id].Lord {
			pick = id
		}
	}
	if pick == "" {
		return fmt.Errorf("officercards: %s deploys no lord", d.node)
	}
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			c := &s.Characters[i]
			if c.ID != pick {
				continue
			}
			best := map[string]int{}
			for _, id := range sorted(g.S.Data.Equipment) {
				eq := g.S.Data.Equipment[id]
				if !g.S.Data.Carries(g.S.Data.Classes[c.Class], nil, eq) {
					continue
				}
				n := 2 * len(eq.Traits)
				for _, v := range eq.Bonus {
					if v != 0 {
						n++
					}
				}
				if n > best[eq.Slot] {
					best[eq.Slot], c.Equipment[eq.Slot] = n, id
				}
			}
		}
	})
	cursorPos = func() (int, int) { return -1000, -1000 }
	d.step = func(f int) {
		switch f {
		case 10:
			g.selected, g.party.sortieTab = pick, 0
		case 30:
			g.party.sortieTab = 2
		case 50:
			g.party.tab = 0
			g.openOverlay("party")
		case 70:
			g.party.tab = 2
		}
	}
	return nil
}
