package gui

import "srpg/internal/core"

func init() { demos["branch"] = demoBranch }

// demoBranch uses an 인수 on the stage's first non-lord ally whose class can be promoted two ways: the item is picked
// and the officer tapped (frame 30), the class picker shows both classes, and the second is chosen at frame 180,
// which plays the promotion.
func demoBranch(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	data := g.S.Data
	snap := g.S.Engine.Snapshot()
	id, item := "", ""
	for _, u := range snap.Units {
		if u.Faction != "ally" || data.Characters[u.Character].Lord || id != "" {
			continue
		}
		for j, c := range snap.Characters {
			if c.ID == u.Character && len(data.Classes[c.Class].Promotes) > 1 {
				tier := data.Classes[c.Class].Tier
				snap.Characters[j].Level = max(c.Level, data.Rules.PromoteLevels[tier])
				id, item = u.ID, insuFor(data, tier)
			}
		}
	}
	if id == "" {
		return nil
	}
	snap.Inventory[item] = 1
	e, err := core.Restore(data, snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.vis = map[string]*unitVis{}
	g.sync(g.S.View())
	d.look(id)
	d.step = func(f int) {
		switch f {
		case 30:
			u := g.unitView(g.S.View(), id)
			g.begin(*u)
			g.ord.stage, g.ord.sub, g.ord.pick = "target", "item", core.Command{Kind: "item", Item: item}
			g.mapClick(u.X, u.Y)
		case 180:
			if len(g.ord.branch) > 1 {
				g.confirm(g.ord.branch[1].cmd)
			}
		}
	}
	return nil
}
