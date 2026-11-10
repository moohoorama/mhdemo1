package gui

import (
	"slices"

	"srpg/internal/core"
)

func init() { demos["branchdeputy"] = demoBranchDeputy }

// demoBranchDeputy gives the stage's first non-lord ally a deputy from the party's reserve, both of the same tier and
// high enough to promote, then uses the tier's 인수 on the officer (frame 30): the picker shows the officer's and the
// deputy's classes side by side, and the deputy's first card is chosen at frame 180, which plays its promotion.
func demoBranchDeputy(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	data := g.S.Data
	snap := g.S.Engine.Snapshot()
	field := map[string]bool{}
	for _, u := range snap.Units {
		field[u.Character] = true
	}
	lead, deputy := -1, -1
	for _, u := range snap.Units {
		if u.Faction != "ally" || u.AI || data.Characters[u.Character].Lord || lead >= 0 {
			continue
		}
		for j, c := range snap.Characters {
			if c.ID == u.Character && len(data.Classes[c.Class].Promotes) > 0 {
				lead = j
			}
		}
	}
	if lead < 0 {
		return nil
	}
	tier := data.Classes[snap.Characters[lead].Class].Tier
	for j, c := range snap.Characters {
		def := data.Characters[c.Def]
		if deputy < 0 && !field[c.ID] && def.Playable && !def.Lord && !c.Dead && c.Deputy == "" &&
			data.Classes[c.Class].Tier == tier && len(data.Classes[c.Class].Promotes) > 0 {
			deputy = j
		}
	}
	if deputy < 0 {
		return nil
	}
	for _, j := range []int{lead, deputy} {
		snap.Characters[j].Level = max(snap.Characters[j].Level, data.Rules.PromoteLevels[tier])
	}
	o := &snap.Characters[lead]
	if !slices.Contains(o.Learned, "deputy_appoint") {
		o.Learned = append(o.Learned, "deputy_appoint")
	}
	o.Deputy = snap.Characters[deputy].ID
	snap.Characters[deputy].Equipment = map[string]string{}
	item := insuFor(data, tier)
	snap.Inventory[item] = 1
	e, err := core.Restore(data, snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.vis = map[string]*unitVis{}
	g.sync(g.S.View())
	id := o.ID
	d.look(id)
	d.step = func(f int) {
		switch f {
		case 30:
			u := g.unitView(g.S.View(), id)
			g.begin(*u)
			g.ord.stage, g.ord.sub, g.ord.pick = "target", "item", core.Command{Kind: "item", Item: item}
			g.mapClick(u.X, u.Y)
		case 180:
			for _, p := range g.ord.branch {
				if p.id != id {
					g.confirm(p.cmd)
					return
				}
			}
		}
	}
	return nil
}
