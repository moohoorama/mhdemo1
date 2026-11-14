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
	Side    bool   // the event is a side event
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

// AreaTime is the time of day in the hero's area now: the first of its Times whose If holds, else its Time.
func (e *Engine) AreaTime() string {
	a := e.area()
	for _, t := range a.Times {
		if e.holdsAll(t.If) {
			return t.Time
		}
	}
	return a.Time
}

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
				v.Do, v.Event, v.Side = "event", ev.Node, ev.Side
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
// another area only through an open exit; with Option empty the hero goes through the open exit at (X, Y) that leads
// into an event, which starts; talk: talking to Target starts their event; sortie: the confidant
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
		if c.Option == "" {
			return e.leaveBy([2]int{c.X, c.Y}, c.Target)
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
		shop := e.area().People[c.Target].Shop
		price = e.data.Ask(shop, c.Item)
		if !slices.Contains(e.data.Shops[shop].Items, c.Item) {
			return fail("NotSold")
		}
		if e.state.Money < price*c.Count {
			return fail("NotEnoughMoney")
		}
		e.state.Money -= price * c.Count
		e.gain(c.Item, c.Count)
		e.emit("buy", "", c.Item, c.Count)
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

// leaveBy sends the hero through the open exit at small tile at that leads into an event, and starts the event.
func (e *Engine) leaveBy(at [2]int, face string) error {
	for _, x := range e.area().Exits {
		if x.Node != "" && x.At == at && e.ExitOpen(x) {
			e.state.Spot.At, e.state.Spot.Face = at, face
			e.state.Node = x.Node
			e.enter()
			return nil
		}
	}
	return fail("ExitShut")
}

var faces = map[string]bool{"NE": true, "NW": true, "SE": true, "SW": true}

// Goal is someone the story waits on in the walk, reachable now through open exits: in Area, by key Person, or an
// exit of Area that leads into an event (Person empty). First is the exit of the hero's area that leads toward them,
// -1 when they stand in it. Side marks a side event, Sortie the confidant (a goal only while no main event is due).
type Goal struct {
	Area, Person string
	First        int
	Side, Sortie bool
	path         []Command
	do           Command // what reaching the goal does: talk, sortie or the walk through the exit
}

// Goals are the events due anywhere the hero can walk to now, nearest first, then the confidant's sortie when no
// main event is due. Nil outside a walk.
func (e *Engine) Goals() []Goal {
	if !e.Walking() {
		return nil
	}
	x, start := e.ExploreFile(), e.state.Spot.Area
	prev, first := map[string]string{start: ""}, map[string]int{start: -1}
	queue := []string{start}
	var events, sorties []Goal
	main := false
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		t := fromState(e.data, e.Snapshot())
		t.state.Spot = content.Spot{Area: id, Face: "SE"}
		var path []Command
		for at := id; at != start; at = prev[at] {
			s := x.Arrival(at, prev[at])
			path = append([]Command{{Kind: "walk", Option: at, X: s.At[0], Y: s.At[1], Target: s.Face}}, path...)
		}
		for _, v := range t.Visitors() {
			g := Goal{Area: id, Person: v.Key, First: first[id], Side: v.Side, path: path, do: Command{Kind: "talk", Target: v.Key}}
			switch v.Do {
			case "event":
				events, main = append(events, g), main || !v.Side
			case "sortie":
				g.Sortie, g.do.Kind = true, "sortie"
				sorties = append(sorties, g)
			}
		}
		open := t.OpenExits()
		for i, ex := range x.Areas[id].Exits {
			if open[i] && ex.Node != "" {
				g := Goal{Area: id, First: first[id], path: path, do: Command{Kind: "walk", X: ex.At[0], Y: ex.At[1], Target: "NE"}}
				if id == start {
					g.First = i
				}
				events, main = append(events, g), true
			}
		}
		for i, ex := range x.Areas[id].Exits {
			if _, seen := prev[ex.To]; open[i] && ex.To != "" && !seen {
				prev[ex.To] = id
				first[ex.To] = first[id]
				if id == start {
					first[ex.To] = i
				}
				queue = append(queue, ex.To)
			}
		}
	}
	if !main {
		events = append(events, sorties...)
	}
	return events
}

// ExploreNext is what a player who wants the story to go on does next on the walk: walk to the nearest main event and
// talk, else to the confidant and sortie, else to the nearest side event. Nil when the walk is stuck (or the story is
// not walking). Each walk command stands at the arrival place of the area it enters.
func (e *Engine) ExploreNext() []Command {
	goals := e.Goals()
	pick := -1
	for i, g := range goals {
		if !g.Side && pick < 0 {
			pick = i
		}
	}
	if pick < 0 && len(goals) > 0 {
		pick = 0
	}
	if pick < 0 {
		return nil
	}
	g := goals[pick]
	return append(append([]Command{}, g.path...), g.do)
}
