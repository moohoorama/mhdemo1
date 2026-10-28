package core

import (
	"slices"

	"srpg/internal/content"
)

// waypointStock is how many of every item a game started at a waypoint carries.
const waypointStock = 9

// NewAt starts a game at a waypoint (content.Waypoints) to check the story from there: the story before
// it counts as played on its default path (its officers joined, its grants and flags set, its battles won
// without their bonus objectives), every officer is at the level of the next battle with the learning
// window open for the points that brought, the party owns every item and every piece of equipment, and
// the officers wear the campaign's Loadout.
func NewAt(d *content.Data, seed uint64, node string) (*Engine, error) {
	if !slices.ContainsFunc(d.Waypoints(), func(w content.Waypoint) bool { return w.Node == node }) {
		return nil, fail("UnknownWaypoint")
	}
	e := begin(d, seed)
	for _, c := range d.Chapters {
		for _, id := range c.Nodes {
			if id == node {
				e.arrive(node)
				return e, nil
			}
			e.pass(d.Node(id), c)
		}
	}
	return nil, fail("UnknownWaypoint")
}

// pass plays what a story node leaves behind, without stopping at it.
func (e *Engine) pass(n content.Node, c content.Chapter) {
	switch n.Kind {
	case "set":
		for k, v := range n.Flags {
			e.state.Flags[k] = v
		}
	case "join", "grant":
		e.once(n)
	case "explore": // its events are what the walk sets
		for _, id := range c.Events {
			e.pass(e.data.Node(id), c)
		}
	case "battle":
		stage := e.data.Stages[e.data.StageIndex(n.Stage)]
		e.state.Executed["won-"+stage.ID] = true
		e.state.Executed["reward-"+stage.ID] = true
		for item, k := range stage.Reward {
			e.gain(item, k)
		}
	}
}

// arrive equips the party for the waypoint node and enters it.
func (e *Engine) arrive(node string) {
	level := e.data.Rules.StageLevel(e.data.NextBattleOrder(node))
	for _, id := range keys(e.charIDs) {
		o := e.char(id)
		e.raiseTo(o, level)
		if e.def(o).Playable && e.canLearn(o) {
			e.state.Learning = append(e.state.Learning, id)
		}
	}
	for id := range e.data.Items {
		e.state.Inventory[id] = max(e.state.Inventory[id], waypointStock)
	}
	worn := map[string]bool{}
	for _, id := range keys(e.charIDs) {
		for _, eq := range e.char(id).Equipment {
			worn[eq] = true
		}
	}
	for id := range e.data.Equipment {
		if !worn[id] {
			e.state.Warehouse[id] = max(e.state.Warehouse[id], 1)
		}
	}
	for _, id := range keys(e.charIDs) {
		o := e.char(id)
		for _, eq := range e.data.Campaign.Loadout[id] {
			item := e.data.Equipment[eq]
			if e.state.Warehouse[eq] == 0 || !e.carries(o, item) {
				continue
			}
			if old := o.Equipment[item.Slot]; old != "" {
				e.state.Warehouse[old]++
			}
			e.state.Warehouse[eq]--
			o.Equipment[item.Slot] = eq
		}
	}
	e.state.Node = node
	e.events = nil
	e.enter()
}
