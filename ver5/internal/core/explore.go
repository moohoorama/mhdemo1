package core

import (
	"slices"
	"srpg/internal/content"
)

// gain gives what a grant or a reward names: money, an equipment to the warehouse, an item to the inventory.
func (e *Engine) gain(id string, n int) {
	if id == content.Money {
		e.state.Money += n
	} else if _, ok := e.data.Equipment[id]; ok {
		e.state.Warehouse[id] += n
	} else {
		e.state.Inventory[id] += n
	}
}

// Visitor is someone standing in the hero's area of the walk, and what talking to them does: "event" starts an
// event, "sortie" asks whether to sortie, "shop" opens a shop, "talk" only says their lines.
type Visitor struct {
	Key, Do string
	Event   string // the event's first node
}

// Walking reports whether the story waits at an explore node for the hero to walk.
func (e *Engine) Walking() bool {
	return e.state.Phase == "scenario" && e.data.Node(e.state.Node).Kind == "explore"
}

// ExploreFile is the explore file of the walk the hero is in (also during its events); nil outside one.
func (e *Engine) ExploreFile() *content.Explore {
	if e.state.Explore == "" {
		return nil
	}
	return e.data.Explores[e.data.Node(e.state.Explore).Explore]
}

func (e *Engine) area() *content.ExploreArea { return e.ExploreFile().Areas[e.state.Spot.Area] }

// Visitors are the people standing in the hero's area now, by key: also during one of the walk's events, where
// nobody can be talked to.
func (e *Engine) Visitors() []Visitor {
	if e.state.Explore == "" || e.state.Phase != "scenario" {
		return nil
	}
	a := e.area()
	var out []Visitor
	for _, key := range keys(a.People) {
		p := a.People[key]
		if !e.holdsAll(p.If) {
			continue
		}
		v := Visitor{Key: key, Do: "talk"}
		for _, ev := range p.Event {
			if e.holdsAll(ev.If) {
				v.Do, v.Event = "event", ev.Node
				break
			}
		}
		if v.Do == "talk" && p.Sortie != nil && e.holdsAll(p.Sortie.If) {
			v.Do = "sortie"
		} else if v.Do == "talk" && p.Shop != "" {
			v.Do = "shop"
		}
		out = append(out, v)
	}
	return out
}

// OpenExits tells for each exit of the hero's area whether it is open now.
func (e *Engine) OpenExits() []bool {
	if !e.Walking() {
		return nil
	}
	var out []bool
	for _, x := range e.area().Exits {
		out = append(out, e.ExitOpen(x))
	}
	return out
}

// ExitOpen reports whether an exit of the walk is open now.
func (e *Engine) ExitOpen(x content.ExploreExit) bool { return e.holdsAll(x.If) }

func (e *Engine) visitor(key string) (Visitor, bool) {
	for _, v := range e.Visitors() {
		if v.Key == key {
			return v, true
		}
	}
	return Visitor{}, false
}

// explore runs a command of the walk. walk: the hero stands at small tile (X, Y) of area Option facing Target,
// another area only through an open exit; talk: talking to Target starts their event; sortie: the confidant
// Target sends the party on with the story; buy, sell: Count of Item at the shop of the person Target.
func (e *Engine) explore(c Command) error {
	if !e.Walking() {
		return fail("WrongPhase")
	}
	switch c.Kind {
	case "walk":
		if !faces[c.Target] {
			return fail("InvalidFace")
		}
		if c.Option != e.state.Spot.Area {
			open := e.OpenExits()
			ok := false
			for i, x := range e.area().Exits {
				ok = ok || open[i] && x.To == c.Option
			}
			if !ok {
				return fail("ExitShut")
			}
		}
		e.state.Spot = content.Spot{Area: c.Option, At: [2]int{c.X, c.Y}, Face: c.Target}
		return nil
	case "talk":
		v, ok := e.visitor(c.Target)
		if !ok || v.Do != "event" {
			return fail("NothingToSay")
		}
		e.state.Node = v.Event
		e.enter()
		return nil
	case "sortie":
		v, ok := e.visitor(c.Target)
		if !ok || v.Do != "sortie" {
			return fail("NoSortie")
		}
		next := e.data.After(e.state.Explore)
		e.state.Explore, e.state.Spot = "", content.Spot{}
		e.state.Node = next
		e.enter()
		return nil
	}
	v, ok := e.visitor(c.Target)
	if !ok || v.Do != "shop" {
		return fail("NoShop")
	}
	price := e.data.Price(c.Item)
	if c.Count < 1 || price <= 0 {
		return fail("InvalidItem")
	}
	if c.Kind == "buy" {
		if !slices.Contains(e.data.Shops[e.area().People[c.Target].Shop].Items, c.Item) {
			return fail("NotSold")
		}
		if e.state.Money < price*c.Count {
			return fail("NotEnoughMoney")
		}
		e.state.Money -= price * c.Count
		e.gain(c.Item, c.Count)
		return nil
	}
	store := e.state.Inventory
	if _, ok := e.data.Equipment[c.Item]; ok {
		store = e.state.Warehouse
	}
	if store[c.Item] < c.Count {
		return fail("NotOwned")
	}
	store[c.Item] -= c.Count
	if store[c.Item] == 0 {
		delete(store, c.Item)
	}
	e.state.Money += price / 2 * c.Count
	return nil
}

var faces = map[string]bool{"NE": true, "NW": true, "SE": true, "SW": true}

// ExploreNext is what a player who wants the story to go on does next on the walk: walk to the nearest area where
// an event is due and talk, else to the confidant and sortie. Nil when the walk is stuck (or the story is not
// walking). Each walk command stands at the arrival place of the area it enters.
func (e *Engine) ExploreNext() []Command {
	if !e.Walking() {
		return nil
	}
	x, start := e.ExploreFile(), e.state.Spot.Area
	for _, want := range []string{"event", "sortie"} {
		prev := map[string]string{start: ""}
		queue := []string{start}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			t := fromState(e.data, e.Snapshot())
			t.state.Spot = content.Spot{Area: id, Face: "SE"}
			for _, v := range t.Visitors() {
				if v.Do != want {
					continue
				}
				var path []Command
				for at := id; at != start; at = prev[at] {
					s := x.Arrival(at, prev[at])
					path = append([]Command{{Kind: "walk", Option: at, X: s.At[0], Y: s.At[1], Target: s.Face}}, path...)
				}
				kind := "talk"
				if want == "sortie" {
					kind = "sortie"
				}
				return append(path, Command{Kind: kind, Target: v.Key})
			}
			open := t.OpenExits()
			for i, ex := range x.Areas[id].Exits {
				if _, seen := prev[ex.To]; open[i] && !seen {
					prev[ex.To] = id
					queue = append(queue, ex.To)
				}
			}
		}
	}
	return nil
}
