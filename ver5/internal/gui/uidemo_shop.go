package gui

import (
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["shop"] = demoShop }

// demoShop starts at chapter 1's walk and opens its tools merchant's shop with money to spend and twelve of an item to sell,
// the opening event skipped and the hero put in the merchant's area; from the frame the shop opens: the buy table (+20), a piece picked (+40) and five of it chosen (+60), bought (+80),
// the sell tab sorted by count (+100), the beans picked at their most (+120) and sold (+140).
func demoShop(d *Demo) error {
	if d.stage == "" {
		d.stage = "start:c01-walk"
	}
	if err := demoExplore(d); err != nil {
		return err
	}
	g, walk, open := d.g, d.step, 0
	d.step = func(f int) {
		if open == 0 {
			if g.S.Engine.ExploreFile() == nil || f < 60 {
				walk(f)
				return
			}
			open = f
			d.edit(func(s *core.State) {
				s.Node, s.Spot.Area = s.Explore, "tools" // past the walk's opening event
				s.Money = 3000
				for _, id := range sorted(g.S.Data.Items) {
					if g.S.Data.Items[id].Price > 0 {
						s.Inventory[id] += 12
						break
					}
				}
			})
			g.shopFrom = "상인"
			g.openOverlay("shop")
		}
		v := &g.shopView
		pickAt := func(i int) {
			rows := g.shopRows(g.S.View(), g.S.Data.Shops[g.S.Engine.ExploreFile().Areas[g.S.View().Spot.Area].People[g.shopFrom].Shop].Items, v.sell)
			sortRows(rows, g.goodsColumns(goodsNumber("count", "", func(r *goodsRow) int { return r.count })), v.list.order, func(r *goodsRow) bool { return r.ok })
			if len(rows) > i {
				v.pick, v.n = rows[i].id, 1
			}
		}
		trade := func(kind string) {
			g.request(session.Request{Op: "command", Command: core.Command{Kind: kind, Target: g.shopFrom, Item: v.pick, Count: v.n}})
			v.n = 1
		}
		switch f - open {
		case 40:
			pickAt(0)
		case 60:
			v.n = 5
		case 80:
			trade("buy")
		case 100:
			*v = shopView{of: v.of, sell: true, list: tableList{order: tableSort{"count", 2}}}
		case 120:
			pickAt(0)
			v.n = 1 << 20
		case 140:
			trade("sell")
		}
	}
	return nil
}
