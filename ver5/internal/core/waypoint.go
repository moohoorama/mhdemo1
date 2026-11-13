package core

import (
	"slices"

	"srpg/internal/content"
)

// waypointStock is how many of every item a game started at a waypoint carries.
const waypointStock = 9

// NewAt starts a game at a waypoint (content.Waypoints) to check the story from there: the story before
// it counts as played on its default path (its officers joined, its grants and flags set, its battles won
// without their bonus objectives), every officer is at the level of the next battle in the class that level
// reaches on the officer's own promotion path, with the learning window open for the points that brought,
// every piece with an Owner the story has handed over is on that officer, and the party owns every item and
// every piece of equipment (the pieces not handed over yet in the warehouse).
// The difficulty is NewGame's.
func NewAt(d *content.Data, seed uint64, node, difficulty string) (*Engine, error) {
	if !slices.ContainsFunc(d.Waypoints(), func(w content.Waypoint) bool { return w.Node == node }) {
		return nil, fail("UnknownWaypoint")
	}
	e, err := beginAt(d, seed, difficulty)
	if err != nil {
		return nil, err
	}
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
	case "join", "grant", "leave":
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
		e.joinAt(o, level)
		if e.def(o).Playable && e.canLearn(o) {
			e.state.Learning = append(e.state.Learning, id)
		}
	}
	for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() { // only what the story has handed over so far
		e.wear(e.char(c.Actor), c.Item)
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
	e.state.Node = node
	e.events = nil
	e.enter()
}
