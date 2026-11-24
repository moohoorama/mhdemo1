package scenegraph

import (
	"fmt"
	"slices"
	"strings"

	"srpg/internal/content"
)

// Step is one thing the player does, in the order the walk's rules allow it: follow mode.
type Step struct {
	N      int      `json:"n"`
	Vertex string   `json:"vertex"`
	Kind   string   `json:"kind"`
	Title  string   `json:"title"`
	Where  string   `json:"where,omitempty"` // the area it starts in
	Via    string   `json:"via,omitempty"`   // who is talked to, or which exit
	Needs  []string `json:"needs,omitempty"`
	Sets   []string `json:"sets,omitempty"`
	Side   bool     `json:"side,omitempty"` // a side event: the sortie does not wait for it
}

// follow plays the graph the way the engine's rules let the hero move, without fighting: every exit and talk whose
// conditions hold is open, the main events come before the sortie, an event's flags open what they open. The order
// it takes is the follow mode's step list; a point where nothing is left to do before the sortie is a soft lock.
func (b *builder) follow() {
	g := b.g
	if len(g.Vertices) == 0 {
		return
	}
	byID := map[string]Vertex{}
	for _, v := range g.Vertices {
		byID[v.ID] = v
	}
	out := map[string][]Edge{}
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e)
	}
	flags := map[string]int{}
	for _, f := range b.opt.Assume {
		flags[f] = 1
	}
	holds := func(conds []string) bool {
		for _, s := range conds {
			c, err := content.ParseCond(s)
			if err != nil {
				return false
			}
			if !b.condHolds(c, flags) {
				return false
			}
		}
		return true
	}
	done := map[string]bool{}
	roster := RosterBefore(b.d, g.Start)
	var steps []Step
	step := func(vid string, e *Edge, where string) {
		v := byID[vid]
		done[vid] = true
		st := Step{N: len(steps) + 1, Vertex: vid, Kind: v.Kind, Title: v.Title, Where: where}
		if e != nil {
			st.Via, st.Needs, st.Side = e.Label, e.Cond, e.Side
		}
		for _, n := range b.rosters[vid] {
			roster = applyRoster(roster, n)
		}
		for k, val := range b.setsOf[vid] {
			flags[k] = val
			st.Sets = append(st.Sets, k)
		}
		slices.Sort(st.Sets)
		steps = append(steps, st)
	}
	// endsStory reports whether the story ends for good along the cases from vertex v (a game over, not a battle).
	endsStory := func(v string) bool {
		seen := map[string]bool{v: true}
		for q := []string{v}; len(q) > 0; q = q[1:] {
			if byID[q[0]].Kind == KindEnd {
				return true
			}
			for _, e := range out[q[0]] {
				if (e.Kind == EdgeNext || e.Kind == EdgeCase) && !seen[e.To] {
					seen[e.To] = true
					q = append(q, e.To)
				}
			}
		}
		return false
	}
	// leaves plays on from an event: a branch or a choice goes on to its first open case, an event the chain comes to
	// is a step too; it reports where the story goes on when the chain leaves the walk (a battle, another walk, the
	// end of the walk), and false when it comes back to an area.
	leaves := func(from string) (string, bool) {
		cur := from
		for guard := 0; guard < 50; guard++ {
			var next *Edge
			for _, e := range out[cur] {
				e := e
				if e.Kind == EdgeBack && e.Label == "탐험 종료" {
					return e.To, true
				}
				if (e.Kind == EdgeNext || e.Kind == EdgeCase) && holds(e.Cond) && (next == nil || endsStory(next.To) && !endsStory(e.To)) {
					next = &e
				}
			}
			if next == nil {
				return "", false
			}
			if byID[next.To].Kind != KindEvent {
				return next.To, true
			}
			if !done[next.To] {
				step(next.To, next, "")
			}
			cur = next.To
		}
		return "", false
	}
	// flood is the areas the hero can reach from area a through open exits
	flood := func(a string) []string {
		seen := []string{a}
		for i := 0; i < len(seen); i++ {
			for _, e := range out[seen[i]] {
				if e.Kind == EdgeExit && holds(e.Cond) && !slices.Contains(seen, e.To) {
					seen = append(seen, e.To)
				}
			}
		}
		return seen
	}
	cur := g.Vertices[0].ID
	for guard := 0; guard < 400 && cur != ""; guard++ {
		v := byID[cur]
		switch v.Kind {
		case KindExplore:
			if done[cur] {
				cur = ""
				break
			}
			done[cur] = true
			if x := b.d.Explores[b.d.Node(strings.TrimPrefix(cur, "explore:")).Explore]; x != nil {
				b.g.Issues = append(b.g.Issues, WhereaboutsIssues(b.d, cur, x, roster)...)
			}
			var start string
			for _, e := range out[cur] {
				if e.Kind == EdgeOpen && !done[e.To] {
					e := e
					step(e.To, &e, "")
					if to, ok := leaves(e.To); ok {
						cur = to
						goto next
					}
				}
				if e.Kind == EdgeStart {
					start = e.To
				}
			}
			cur = b.playWalk(start, byID, out, holds, done, step, leaves, flood)
		case KindEvent, KindBattle, KindEnd:
			if !done[cur] {
				step(cur, nil, "")
			}
			if v.Kind != KindEvent {
				cur = ""
				for _, e := range out[v.ID] {
					if e.Kind == EdgeNext && !done[e.To] {
						cur = e.To
						break
					}
				}
				break
			}
			if to, ok := leaves(cur); ok && !done[to] {
				cur = to
			} else {
				cur = ""
			}
		default:
			cur = ""
		}
	next:
	}
	for i := range steps {
		steps[i].N = i + 1
	}
	g.Steps = append([]Step{}, steps...)
	for _, v := range g.Vertices {
		if v.Kind != KindEvent || done[v.ID] {
			continue
		}
		branch := false
		for _, e := range g.Edges {
			branch = branch || e.To == v.ID && e.Kind == EdgeCase
		}
		if !branch {
			b.issue("warn", v.ID, fmt.Sprintf("따라가기에서 도달하지 못한 이벤트입니다: %s", v.Title))
		}
	}
}

func (b *builder) condHolds(c content.Cond, flags map[string]int) bool {
	if c.Op != "flag" {
		return true // not decided by flags: taken as true
	}
	return (flags[c.Arg] >= c.N) != c.Not
}

// playWalk plays one walk: main events first, then side events, then the sortie. It returns the vertex the story
// goes on with, "" when it ends or is stuck.
func (b *builder) playWalk(start string, byID map[string]Vertex, out map[string][]Edge, holds func([]string) bool,
	done map[string]bool, step func(string, *Edge, string), leaves func(string) (string, bool), flood func(string) []string) string {
	for guard := 0; guard < 200; guard++ {
		reach := flood(start)
		var main, side, sortie *Edge
		for _, a := range reach {
			for _, e := range out[a] {
				e := e
				if !holds(e.Cond) {
					continue
				}
				switch {
				case (e.Kind == EdgeTalk || e.Kind == EdgeGate) && !done[e.To]:
					if e.Side && side == nil {
						side = &e
					} else if !e.Side && main == nil {
						main = &e
					}
				case e.Kind == EdgeSort && sortie == nil:
					sortie = &e
				}
			}
		}
		pick := main
		if pick == nil {
			pick = side
		}
		if pick != nil {
			step(pick.To, pick, byID[pick.From].Title)
			if to, ok := leaves(pick.To); ok {
				return to
			}
			continue
		}
		if sortie != nil {
			if !done[sortie.To] {
				step(sortie.To, sortie, byID[sortie.From].Title)
			}
			return sortie.To
		}
		b.issue("error", start, "더 할 수 있는 일이 없는데 출진도 열리지 않습니다 (진행 불가)")
		return ""
	}
	return ""
}
