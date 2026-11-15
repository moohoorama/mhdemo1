// Package replay orders resolved actions for claim replay (design.md 0.9): a faction's
// phase is resolved one unit at a time by the core, then shown with independent actions
// overlapping. An action waits only for the last earlier action that claimed one of the
// same units or cells, so the replay shows exactly the sequentially resolved result.
package replay

import (
	"fmt"
	"sort"

	"srpg/internal/content"
	"srpg/internal/core"
)

// Action is one unit's resolved turn (all of its commands) or a barrier such as ending
// the faction, a duel or the battle result, which waits for everything before it and
// holds back everything after it.
type Action struct {
	Actor string
	Steps []Step
	// Before holds every living unit's cell before the action; claims are taken from it.
	Before  map[string]content.Point
	Barrier bool
	// After lists the indices of the actions that must be shown completely first.
	After []int
	// Trail lists the actions whose walk this one follows over the same cells: it may start once
	// the other has been playing that many walking steps, so a file of units crosses a bridge
	// a step or two apart instead of one by one.
	Trail []Lead
}

// Lead is an earlier action and how many walking steps after its start a trailing one may start.
type Lead struct {
	Index int
	Steps float64
}

// Step is one applied command and the events it produced.
type Step struct {
	Command core.Command
	Events  []core.Event
}

// barrierKinds are events shown over the whole field.
var barrierKinds = map[string]bool{"speech": true, "unit-arrival": true, "duel": true, "cutscene": true, "victory": true, "turn": true}

// New starts an action in the observed state; Add then records each applied command.
func New(o core.Observation, actor string) *Action {
	a := &Action{Actor: actor, Before: map[string]content.Point{}}
	for _, u := range o.UnitViews {
		if u.HP > 0 {
			a.Before[u.ID] = content.Point{X: u.X, Y: u.Y}
		}
	}
	return a
}

// Add appends a command and the events it produced.
func (a *Action) Add(c core.Command, events []core.Event) {
	a.Steps = append(a.Steps, Step{c, events})
	if c.Kind == "end" || c.Actor == "" {
		a.Barrier = true
	}
	for _, e := range events {
		if barrierKinds[e.Kind] {
			a.Barrier = true
		}
	}
}

// Claim is a unit or cell an action touches. A cell its actor only walks over is held from In to
// Out, in walking steps from the action's start (In -1 for the cell it starts on, Out -1 for the cell
// it stays on: there it holds from its arrival on); any other claim is held for the whole action.
type Claim struct {
	Key     string
	Walk    bool
	In, Out float64
}

// Claims are the units and cells the action touches: its actor and the cell it starts on,
// every cell of its walk, every unit an event names (targets, counter-attackers, retreats)
// together with that unit's cell, and every cell an event lists (a blow's area among them).
// A walk that opens the action claims its cells for the time the actor is on them; the rest
// is claimed whole.
func (a *Action) Claims() []Claim {
	set := map[string]Claim{}
	whole := func(k string) { set[k] = Claim{Key: k} }
	unit := func(id string) {
		if id == "" {
			return
		}
		whole("unit:" + id)
		if p, ok := a.Before[id]; ok {
			whole(cell(p))
		}
	}
	unit(a.Actor)
	walk := a.walk()
	for i, s := range a.Steps {
		for _, e := range s.Events {
			unit(e.Actor)
			unit(e.Target)
			if i == 0 && walk != nil && e.Kind == "move" && e.Actor == a.Actor {
				continue
			}
			for _, p := range e.Path {
				whole(cell(p))
			}
		}
	}
	// a step to walk[i] starts at i-1 and ends at i: the actor is on the cell from i-1 until it
	// has stepped off it at i+1
	for i, p := range walk {
		c := Claim{Key: cell(p), Walk: true, In: float64(i - 1), Out: float64(i + 1)}
		if i == len(walk)-1 {
			c.Out = -1
		}
		if old, ok := set[c.Key]; !ok || old.Walk || i == 0 { // the start cell was claimed whole for the actor
			set[c.Key] = c
		}
	}
	out := make([]Claim, 0, len(set))
	for _, c := range set {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// walk is the path the actor walks at the very start of the action (its first command a move), or nil.
func (a *Action) walk() []content.Point {
	if len(a.Steps) == 0 || a.Steps[0].Command.Kind != "move" {
		return nil
	}
	for _, e := range a.Steps[0].Events {
		if e.Kind == "move" && e.Actor == a.Actor && len(e.Path) > 1 {
			return e.Path
		}
	}
	return nil
}

func cell(p content.Point) string { return fmt.Sprintf("cell:%d,%d", p.X, p.Y) }

// Link fills After and Trail for actions in resolution order using the claim table: each claim
// remembers only its last claimant, so the precedence edges stay minimal and acyclic. Where both
// only walk over a cell, the later action trails the earlier one instead of waiting for all of it:
// it reaches the cell once the earlier one has stepped off it (or, where that one stays, has
// arrived; walking through a standing unit is what a one-by-one replay shows too).
func Link(actions []*Action) {
	claims := map[string]Claim{}
	owner := map[string]int{}
	barrier := -1
	for i, a := range actions {
		after := map[int]bool{}
		trail := map[int]float64{}
		if barrier >= 0 {
			after[barrier] = true
		}
		if a.Barrier {
			for j := barrier + 1; j < i; j++ {
				after[j] = true
			}
			barrier = i
			claims, owner = map[string]Claim{}, map[string]int{}
		} else {
			for _, c := range a.Claims() {
				if j, ok := owner[c.Key]; ok {
					prev := claims[c.Key]
					switch {
					case !c.Walk || !prev.Walk:
						after[j] = true
					case c.In >= 0: // a walk from the cell it stands on was passed through by the earlier one
						left := prev.Out
						if left < 0 {
							left = prev.In + 1
						}
						trail[j] = max(trail[j], left-c.In, 0)
					}
				}
				claims[c.Key], owner[c.Key] = c, i
			}
		}
		a.After = a.After[:0]
		for j := range after {
			a.After = append(a.After, j)
		}
		sort.Ints(a.After)
		a.Trail = a.Trail[:0]
		for j, n := range trail {
			if !after[j] {
				a.Trail = append(a.Trail, Lead{j, n})
			}
		}
		sort.Slice(a.Trail, func(x, y int) bool { return a.Trail[x].Index < a.Trail[y].Index })
	}
}
