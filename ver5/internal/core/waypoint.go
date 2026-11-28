package core

import (
	"slices"

	"srpg/internal/content"
)

// waypointStock is how many of every item a game started at a waypoint carries.
const waypointStock = 9

// Start is what a game started at a waypoint is told about the story before it.
type Start struct {
	Flags []string // the battle flags (content.StoryFlags) the battles before the waypoint left
	Level int      // the level every officer is raised to; none is the level of the next battle
}

// NewAt starts a game at a waypoint as NewAtWith does with nothing told: no battle left a flag.
func NewAt(d *content.Data, seed uint64, node, difficulty string) (*Engine, error) {
	return NewAtWith(d, seed, node, difficulty, Start{})
}

// NewAtWith starts a game at a waypoint (content.Waypoints) to check the story from there: the story before
// it counts as played along the path its branches take (its officers joined, its grants and flags set, its
// battles won with the bonus flags start names and no others), every officer is at the level of the next
// battle (or start's) in the class that level reaches on the officer's own promotion path, with the learning
// window open for the points that brought, every piece with an Owner the story has handed over is on that
// officer, and the party owns every item and every piece of equipment (the pieces not handed over yet in
// the warehouse). The difficulty is NewGame's.
func NewAtWith(d *content.Data, seed uint64, node, difficulty string, start Start) (*Engine, error) {
	if !slices.ContainsFunc(d.Waypoints(), func(w content.Waypoint) bool { return w.Node == node }) {
		return nil, fail("UnknownWaypoint")
	}
	if start.Level < 0 || start.Level > d.Rules.MaxLevel {
		return nil, fail("BadLevel")
	}
	e, err := beginAt(d, seed, difficulty)
	if err != nil {
		return nil, err
	}
	left := map[string][]string{} // stage → the flags it leaves
	for _, f := range d.StoryFlags() {
		if slices.Contains(start.Flags, f.Flag) {
			left[f.Stage] = append(left[f.Stage], f.Flag)
		}
	}
	leads := d.Leads(node)
	for ci := 0; ci < len(d.Chapters); ci++ {
		c := d.Chapters[ci]
		at := slices.Index(c.Nodes, node)
		for j := 0; j < len(c.Nodes); {
			if j == at {
				e.arrive(node, start.Level)
				return e, nil
			}
			n := d.Node(c.Nodes[j])
			e.pass(n, c)
			if n.Kind == "battle" {
				for _, f := range left[n.Stage] {
					e.state.Flags[f] = 1
				}
			}
			next := e.toward(n, leads)
			if ni := d.ChapterIndex(next); ni > ci && slices.Contains(d.Chapters[ni].Nodes, next) {
				// a branch into another telling's chapter (25장 → 25v장): the chapters between are not played
				ci, c, at = ni, d.Chapters[ni], slices.Index(d.Chapters[ni].Nodes, node)
				j = slices.Index(c.Nodes, next)
				continue
			}
			// the nodes a branch or a jump goes over are not played; one that goes nowhere ahead reads on
			k := slices.Index(c.Nodes, next)
			if k <= j {
				k = j + 1
			}
			if at > j && at < k { // the waypoint is on an arm the story does not take: start there all the same
				k = at
			}
			j = k
		}
	}
	return nil, fail("UnknownWaypoint")
}

// toward is where the walk to a waypoint goes after n: where the story goes (passed), unless that can no longer
// come to the waypoint and another way out of n can (leads, content.Leads) — a jump into 가상 25v장 takes 25장's
// branch to it even though no 「느린 길」 was chosen; the chapter's own nodes then set its flags.
func (e *Engine) toward(n content.Node, leads map[string]bool) string {
	next := e.passed(n)
	if next == "" && n.Kind == "choice" { // a choice reads on, unless its next node goes away from the waypoint
		next = e.data.After(n.ID)
	}
	if leads[next] {
		return next
	}
	for _, s := range e.data.Successors(n) {
		if leads[s] {
			return s
		}
	}
	return e.passed(n)
}

// passed is the node the story goes to after a node it passes, as far as that is known without the player.
func (e *Engine) passed(n content.Node) string {
	switch n.Kind {
	case "choice", "end":
		return ""
	case "branch":
		for _, c := range n.Cases {
			if e.holdsAll(c.If) {
				return c.Next
			}
		}
		if n.Else != "" {
			return n.Else
		}
	}
	return e.data.After(n.ID)
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
		for _, ev := range stage.Events { // officers who join on the field (조운 at B19, 관우's party at B20)
			for _, a := range ev.Do {
				if a.Kind != "join" {
					continue
				}
				for _, sp := range a.Spawns {
					if e.char(sp.Character) == nil {
						e.welcome(sp.Character, stage.Order)
					}
					if !contains(e.state.Deployment, sp.Character) {
						e.state.Deployment = append(e.state.Deployment, sp.Character)
					}
				}
			}
		}
	}
}

// arrive equips the party for the waypoint node and enters it, at the level given or else the next battle's.
func (e *Engine) arrive(node string, level int) {
	if level == 0 {
		level = e.data.Rules.StageLevel(e.data.NextBattleOrder(node))
	}
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

// Reached is the last waypoint (content.Waypoints) the story arrived at: a chapter's start when a node of a chapter
// other than the one from stood in was entered (a branch may go into the middle of a chapter, 25장 → 25v장, so
// the first node of it need not be passed), or a battle entered. from is the node the engine stood at before
// the last command, empty for a game just begun. It is empty when the story arrived at none, and a node
// re-entered inside its own chapter (an explore returning from a talk) is not an arrival. Read it right after
// the command or construction that told Entered.
func (e *Engine) Reached(from string) string {
	chapter := -1
	if from != "" {
		chapter = e.data.ChapterIndex(from)
	}
	out := ""
	for _, id := range e.entered {
		if ci := e.data.ChapterIndex(id); ci >= 0 && ci != chapter {
			out, chapter = e.data.Chapters[ci].Nodes[0], ci
		}
		if id != from && e.data.Node(id).Kind == "battle" {
			out = id
		}
	}
	return out
}

// Route is the arm of its fork (content.Fork) the game is on at a waypoint, told by the flags and the party it has
// now: the waypoint is kept once per arm. It is empty when the waypoint is on no fork.
func (e *Engine) Route(node string) string {
	f := e.data.ForkAt(node)
	if f == nil {
		return ""
	}
	for _, c := range f.Cases {
		if e.holdsAll(c.If) {
			return c.Next
		}
	}
	return f.Else
}
