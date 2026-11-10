package gui

import "srpg/internal/core"

func init() { demos["meritresult"] = demoMeritResult }

// demoMeritResult shows the victory window of 동탁 추격전 with merit set by hand: the podium comes in third,
// second, first, then the rest and the spoils.
func demoMeritResult(d *Demo) error {
	if err := d.battle("B07"); err != nil {
		return err
	}
	g := d.g
	snap := g.S.Engine.Snapshot()
	snap.Phase, snap.Result = "result", "victory"
	m := &core.Merit{XP: map[string]int{}, From: map[string]int{}, Spoils: map[string]int{"money": 600, "tangni_armor": 1, "bean": 2}}
	xp := []int{9200, 14100, 6400, 3300, 2100, 800}
	n := 0
	for _, u := range snap.Units {
		if u.Faction == "ally" && !u.AI {
			m.XP[u.Character] = xp[n%len(xp)]
			for _, c := range snap.Characters {
				if c.ID == u.Character {
					m.From[c.ID] = c.Level - xp[n%len(xp)]/10000
				}
			}
			n++
		}
	}
	snap.Merit = m
	e, err := core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.vis = map[string]*unitVis{}
	g.sync(g.S.View())
	return nil
}
