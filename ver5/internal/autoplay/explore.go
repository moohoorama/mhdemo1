package autoplay

import (
	"math"
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

// Explore visits reachable shops for promotions and supplies before following the story.
func (g *Genome) Explore(d *content.Data, e *core.Engine) []core.Command {
	x := e.ExploreFile()
	start := e.Observe().Spot.Area
	type visit struct {
		area string
		path []core.Command
	}
	queue := []visit{{area: start}}
	seen := map[string]bool{start: true}
	var best []core.Command
	bestScore := 0.0
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		t, err := core.Restore(d, e.Snapshot())
		if err != nil {
			continue
		}
		for _, c := range v.path {
			if _, err = t.Apply(c); err != nil {
				break
			}
		}
		if err != nil {
			continue
		}
		for _, who := range t.Visitors() {
			if who.Do != "shop" {
				continue
			}
			shop := x.Areas[v.area].People[who.Key].Shop
			if c, score := g.purchase(d, t, shop); score > bestScore {
				c.Target = who.Key
				best, bestScore = append(slices.Clone(v.path), c), score
			}
		}
		for i, open := range t.OpenExits() {
			to := x.Areas[v.area].Exits[i].To
			if !open || to == "" || seen[to] {
				continue
			}
			seen[to] = true
			at := x.Arrival(to, v.area)
			path := append(slices.Clone(v.path), core.Command{Kind: "walk", Option: to, X: at.At[0], Y: at.At[1], Target: at.Face})
			queue = append(queue, visit{to, path})
		}
	}
	if len(best) > 0 {
		return best
	}
	return e.ExploreNext()
}

func (g *Genome) purchase(d *content.Data, e *core.Engine, shop string) (core.Command, float64) {
	o := e.Observe()
	party := Officers(d, e)
	party = party[:min(8, len(party))]
	hp, mp := 0, 0
	for _, c := range party {
		st, err := e.CharacterStats(c.ID)
		if err == nil {
			hp += st.MaxHP
			mp += st.MaxMP
		}
	}
	stockHP, stockMP := 0, 0
	for id, n := range o.Inventory {
		it := d.Items[id]
		if it.Effect == "hp" {
			stockHP += n * it.Amount
		} else if it.Effect == "mp" {
			stockMP += n * it.Amount
		}
	}
	var best core.Command
	bestScore := 0.0
	for _, id := range d.Shops[shop].Items {
		it := d.Items[id]
		price := d.Ask(shop, id)
		if price <= 0 || o.Money < price {
			continue
		}
		score, count := 0.0, 0
		switch it.Effect {
		case content.ItemPromote:
			for _, c := range party {
				cl := d.Classes[c.Class]
				if it.Tier == cl.Tier+1 && cl.Tier < len(d.Rules.PromoteLevels) && c.Level >= d.Rules.PromoteLevels[cl.Tier] && len(d.Promotes(c.Def, c.Class)) > 0 {
					count++
				}
			}
			count -= o.Inventory[id]
			score = 100
		case "hp", "mp":
			need, average := hp-stockHP, float64(hp)/float64(max(1, len(party)))
			priority := 10.0
			if it.Effect == "mp" {
				need, average, priority = mp/2-stockMP, float64(mp)/float64(max(1, len(party))), 1
			}
			if it.Amount > 0 {
				count = int(math.Ceil(float64(need) / float64(it.Amount)))
				useful := min(float64(it.Amount), average/2)
				score = priority * useful / float64(price) * min(1, float64(it.Amount)/max(1, average/2))
			}
		}
		count = min(count, o.Money/price)
		if count > 0 && score > bestScore {
			best, bestScore = core.Command{Kind: "buy", Item: id, Count: count}, score
		}
	}
	return best, bestScore
}

func (r *RulePlayer) Explore(e *core.Engine) []core.Command   { return r.G.Explore(r.D, e) }
func (s *Searcher) Explore(e *core.Engine) []core.Command     { return s.G.Explore(s.D, e) }
func (v *ValuePlayer) Explore(e *core.Engine) []core.Command  { return v.G.Explore(v.D, e) }
func (p *PolicyPlayer) Explore(e *core.Engine) []core.Command { return p.G.Explore(p.D, e) }
