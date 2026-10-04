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
}

// Step is one applied command and the events it produced.
type Step struct {
	Command core.Command
	Events  []core.Event
}

// barrierKinds are events shown over the whole field.
var barrierKinds = map[string]bool{"duel": true, "victory": true, "turn": true}

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

// Claims are the units and cells the action touches: its actor and the cell it starts on,
// every cell of its walk, and every unit an event names (targets, counter-attackers,
// retreats) together with that unit's cell.
func (a *Action) Claims() []string {
	set := map[string]bool{}
	unit := func(id string) {
		if id == "" {
			return
		}
		set["unit:"+id] = true
		if p, ok := a.Before[id]; ok {
			set[cell(p)] = true
		}
	}
	unit(a.Actor)
	for _, s := range a.Steps {
		for _, e := range s.Events {
			unit(e.Actor)
			unit(e.Target)
			for _, p := range e.Path {
				set[cell(p)] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func cell(p content.Point) string { return fmt.Sprintf("cell:%d,%d", p.X, p.Y) }

// Link fills After for actions in resolution order using the claim table: each claim
// remembers only its last claimant, so the precedence edges stay minimal and acyclic.
func Link(actions []*Action) {
	claims := map[string]int{}
	barrier := -1
	for i, a := range actions {
		after := map[int]bool{}
		if barrier >= 0 {
			after[barrier] = true
		}
		if a.Barrier {
			for j := barrier + 1; j < i; j++ {
				after[j] = true
			}
			barrier = i
			claims = map[string]int{}
		} else {
			for _, k := range a.Claims() {
				if j, ok := claims[k]; ok {
					after[j] = true
				}
				claims[k] = i
			}
		}
		a.After = a.After[:0]
		for j := range after {
			a.After = append(a.After, j)
		}
		sort.Ints(a.After)
	}
}
