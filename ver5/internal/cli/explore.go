package cli

import (
	"fmt"
	"maps"
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

// exploreScreen is the walk: the area, the people to talk to, the signs to read and the open ways out. A shop
// and the confidant's question are screens of their own (p.screen "shop", "sortie" for the person p.actor).
func (p *player) exploreScreen(o core.Observation) {
	e := p.s.Engine
	x := e.ExploreFile()
	a := x.Areas[o.Spot.Area]
	switch p.screen {
	case "shop":
		p.shopScreen(o, a.People[p.actor])
		return
	case "sortie":
		p.printf("\n%s: %s\n", content.PersonName(p.actor), a.People[p.actor].Sortie.Ask)
		p.add("예", func() {
			p.screen = ""
			p.command(core.Command{Kind: "sortie", Target: p.actor})
		})
		p.add("아니오", func() { p.screen = "" })
		return
	}
	p.printf("\n== %s · 돈 %d ==\n", a.Name, o.Money)
	for _, v := range e.Visitors() {
		name, person := content.PersonName(v.Key), a.People[v.Key]
		switch v.Do {
		case "event":
			p.add(name+"에게 말 걸기 ★", func() { p.command(core.Command{Kind: "talk", Target: v.Key}) })
		case "sortie":
			p.add(name+"에게 말 걸기 (출진)", func() { p.screen, p.actor = "sortie", v.Key })
		case "shop":
			p.add(name+"에게 말 걸기 (상점)", func() { p.screen, p.actor = "shop", v.Key })
		default:
			if len(person.Talk) > 0 {
				p.add(name+"에게 말 걸기", func() { p.lines(person.Talk) })
			}
		}
	}
	for _, s := range a.Signs {
		p.add(s.Name+" 살펴보기", func() { p.lines(s.Talk) })
	}
	open := e.OpenExits()
	for i, ex := range a.Exits {
		if !open[i] {
			if len(ex.Shut) > 0 {
				p.add(ex.Name+" (닫힘)", func() { p.lines(ex.Shut) })
			}
			continue
		}
		p.add("→ "+ex.Name, func() {
			if ex.Node != "" {
				p.command(core.Command{Kind: "walk", X: ex.At[0], Y: ex.At[1], Target: o.Spot.Face})
				return
			}
			s := x.Arrival(ex.To, o.Spot.Area)
			p.command(core.Command{Kind: "walk", Option: ex.To, X: s.At[0], Y: s.At[1], Target: s.Face})
		})
	}
}

func (p *player) lines(talk []content.Line) {
	for _, l := range talk {
		if l[0] == "" {
			p.printf("  %s\n", l[1])
		} else if c, ok := p.d.Characters[l[0]]; ok {
			p.printf("%s: %s\n", c.Name, l[1])
		} else {
			p.printf("%s: %s\n", l[0], l[1])
		}
	}
}

func (p *player) shopScreen(o core.Observation, person *content.ExplorePerson) {
	shop := p.d.Shops[person.Shop]
	p.printf("\n== %s · 돈 %d (0: 나가기) ==\n", shop.Name, o.Money)
	trade := func(kind, item string) func() {
		return func() { p.command(core.Command{Kind: kind, Target: p.actor, Item: item, Count: 1}) }
	}
	for _, id := range shop.Items {
		p.add(fmt.Sprintf("사기: %s %d", p.itemName(id), p.d.Ask(person.Shop, id)), trade("buy", id))
	}
	for _, store := range []map[string]int{o.Inventory, o.Warehouse} {
		for _, id := range slices.Sorted(maps.Keys(store)) {
			if price := p.d.Price(id); price > 0 && store[id] > 0 {
				p.add(fmt.Sprintf("팔기: %s ×%d %d", p.itemName(id), store[id], price/2), trade("sell", id))
			}
		}
	}
	p.add("나가기", func() { p.screen = "" })
}

func (p *player) itemName(id string) string {
	if e, ok := p.d.Equipment[id]; ok {
		return e.Name
	}
	return p.d.Items[id].Name
}
