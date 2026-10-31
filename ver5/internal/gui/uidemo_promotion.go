package gui

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

func init() { demos["promotion"] = demoPromotion }

// insuFor is the 인수 that promotes from a class of the given tier.
func insuFor(data *content.Data, tier int) string {
	for _, id := range sorted(data.Items) {
		if it := data.Items[id]; it.Effect == content.ItemPromote && it.Tier == tier+1 {
			return id
		}
	}
	return ""
}

// demoPromotion promotes, with its 인수, the first officer of the party who has no sprite of his own and so wears
// his class's (번궁, 무도가 → 권사) on the stage's first non-lord ally's cell (from frame 30): the seal rises over him,
// he whitens while the gold rings close in, takes the new class's look in the pillar of light and the promotion
// window opens when the replay ends (about 4 s).
func demoPromotion(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	data := g.S.Data
	snap := g.S.Engine.Snapshot()
	who, item := -1, ""
	for i, c := range snap.Characters {
		def := data.Characters[c.Def]
		if who >= 0 || c.Temp || def.Lord || def.Look.Sprite != "" || len(data.Classes[c.Class].Promotes) == 0 {
			continue
		}
		who, item = i, insuFor(data, data.Classes[c.Class].Tier)
	}
	if who < 0 {
		return nil
	}
	c := &snap.Characters[who]
	c.Level = max(c.Level, data.Rules.PromoteLevels[data.Classes[c.Class].Tier])
	for i, u := range snap.Units {
		if u.Faction != "ally" || data.Characters[u.Character].Lord {
			continue
		}
		for k, dep := range snap.Deployment {
			if dep == u.ID {
				snap.Deployment[k] = c.ID
			}
		}
		snap.Units[i].ID, snap.Units[i].Character, snap.Units[i].Acted, snap.Units[i].HP = c.ID, c.ID, false, 1
		break
	}
	snap.Inventory[item] = 1
	e, err := core.Restore(data, snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.vis = map[string]*unitVis{}
	g.sync(g.S.View())
	d.look(c.ID)
	d.step = func(f int) {
		if f == 30 {
			g.command(core.Command{Kind: "item", Actor: c.ID, Target: c.ID, Item: item})
		}
	}
	return nil
}
