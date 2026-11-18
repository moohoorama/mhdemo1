package content

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Cond is one parsed condition string (flag:x, !alive:liubei, round>=3, any:[a, b], ...).
type Cond struct {
	Not       bool
	Op        string
	Arg, Arg2 string
	N         int
	Any       []Cond
}

// Story nodes may use storyConds; battle events may use battleConds.
var storyConds = map[string]bool{"flag": true, "alive": true, "dead": true, "party": true, "choice": true, "won": true}
var battleConds = map[string]bool{"flag": true, "choice": true, "won": true, "party": true, "round": true, "turnstart": true, "defeated": true,
	"alldefeated": true, "alive": true, "at": true, "allat": true, "adjacent": true, "hp": true, "duel": true, "fled": true, "since": true, "any": true}

var turnNames = []string{"ally", "support", "enemy"}

// ParseCond reads "!?op:args". The forms are flag:key[>=N], alive:X[>=N] (in battle, at least N of X
// stand), dead|party|defeated|alldefeated:X,
// choice:node=option, won:stage, round>=N, turnstart:faction, at|allat:X:area, adjacent:A:B,
// hp:X<N, duel:id, fled:X (left the field by flight), since:event>=N (N rounds or more since the battle event
// last ran) and any:[cond, cond]. A leading ! negates.
func ParseCond(s string) (Cond, error) {
	bad := func() (Cond, error) { return Cond{}, fmt.Errorf("condition %q", s) }
	c := Cond{N: 1}
	if strings.HasPrefix(s, "!") {
		c.Not, s = true, s[1:]
	}
	if n, ok := strings.CutPrefix(s, "round>="); ok {
		v, err := strconv.Atoi(n)
		if err != nil || v < 1 {
			return bad()
		}
		c.Op, c.N = "round", v
		return c, nil
	}
	op, arg, ok := strings.Cut(s, ":")
	if !ok || arg == "" || !storyConds[op] && !battleConds[op] {
		return bad()
	}
	c.Op, c.Arg = op, arg
	switch op {
	case "flag", "alive", "since":
		if name, n, ok := strings.Cut(arg, ">="); ok {
			v, err := strconv.Atoi(n)
			if err != nil || name == "" {
				return bad()
			}
			c.Arg, c.N = name, v
		}
	case "choice":
		node, option, ok := strings.Cut(arg, "=")
		if !ok || node == "" || option == "" {
			return bad()
		}
		c.Arg, c.Arg2 = node, option
	case "turnstart":
		if !contains(turnNames, arg) {
			return bad()
		}
	case "at", "allat":
		who, area, ok := strings.Cut(arg, ":")
		id, err := strconv.Atoi(area)
		if !ok || who == "" || err != nil || id < 1 {
			return bad()
		}
		c.Arg, c.Arg2, c.N = who, area, id
	case "adjacent":
		a, b, ok := strings.Cut(arg, ":")
		if !ok || a == "" || b == "" {
			return bad()
		}
		c.Arg, c.Arg2 = a, b
	case "hp":
		who, n, ok := strings.Cut(arg, "<")
		v, err := strconv.Atoi(n)
		if !ok || who == "" || err != nil || v < 1 || v > 100 {
			return bad()
		}
		c.Arg, c.N = who, v
	case "any":
		inner, ok := strings.CutSuffix(strings.TrimPrefix(arg, "["), "]")
		if !ok || !strings.HasPrefix(arg, "[") {
			return bad()
		}
		for _, part := range strings.Split(inner, ",") {
			sub, err := ParseCond(strings.TrimSpace(part))
			if err != nil || sub.Op == "any" {
				return bad()
			}
			c.Any = append(c.Any, sub)
		}
	}
	return c, nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// battleRefs is what a stage's battle conditions may point at.
type battleRefs struct {
	targets map[string]bool // unit ids and group names
	areas   map[int]bool
	duels   map[string]bool
	events  map[string]bool // the stage's battle event ids
}

// AreaTarget splits an event target that names the units standing in an area, "area:N" or "<target>@area:N"
// (a unit, group or side; none: any unit). ok is false for a plain target.
func AreaTarget(t string) (base string, area int, ok bool) {
	base, num, found := strings.Cut(t, "@area:")
	if !found {
		if num, found = strings.CutPrefix(t, "area:"); !found {
			return "", 0, false
		}
		base = ""
	}
	n, err := strconv.Atoi(num)
	return base, n, err == nil
}

// validTarget reports whether an event target names units: a known unit, group or side, or the units of one standing
// in a known area.
func (r *battleRefs) validTarget(t string) bool {
	base, area, ok := AreaTarget(t)
	if !ok {
		return r.targets[t]
	}
	return r.areas[area] && (base == "" || r.targets[base])
}

// validateCond checks one condition string: story nodes pass refs nil, battle events the stage's refs.
func (d *Data) validateCond(s string, refs *battleRefs) error {
	c, err := ParseCond(s)
	if err != nil {
		return err
	}
	return d.validateParsed(s, c, refs)
}

func (d *Data) validateParsed(s string, c Cond, refs *battleRefs) error {
	if refs == nil && !storyConds[c.Op] || refs != nil && !battleConds[c.Op] {
		return fmt.Errorf("condition %q is not allowed here", s)
	}
	target := func(id string) error {
		if refs != nil && !refs.targets[id] {
			return fmt.Errorf("condition %q: no unit or group %s", s, id)
		}
		return nil
	}
	switch c.Op {
	case "party": // the roster, in a battle too: whether a recruitable officer has joined
		if c.N > 1 || !d.Has(c.Arg) {
			return fmt.Errorf("condition %q: no character %s", s, c.Arg)
		}
	case "alive", "dead":
		if refs == nil && c.N > 1 {
			return fmt.Errorf("condition %q: a count is for battles", s)
		}
		if refs == nil && !d.Has(c.Arg) {
			return fmt.Errorf("condition %q: no character %s", s, c.Arg)
		}
		return target(c.Arg)
	case "choice":
		n := d.Node(c.Arg)
		if _, ok := n.Choices[c.Arg2]; n.Kind != "choice" || !ok {
			return fmt.Errorf("condition %q: no such choice", s)
		}
	case "won":
		if d.StageIndex(c.Arg) < 0 {
			return fmt.Errorf("condition %q: no stage %s", s, c.Arg)
		}
	case "defeated", "alldefeated", "hp", "fled":
		return target(c.Arg)
	case "adjacent":
		if err := target(c.Arg); err != nil {
			return err
		}
		return target(c.Arg2)
	case "at", "allat":
		if !refs.areas[c.N] {
			return fmt.Errorf("condition %q: no area %d", s, c.N)
		}
		return target(c.Arg)
	case "duel":
		if !refs.duels[c.Arg] {
			return fmt.Errorf("condition %q: no duel %s", s, c.Arg)
		}
	case "since":
		if !refs.events[c.Arg] || c.N < 0 {
			return fmt.Errorf("condition %q: no battle event %s", s, c.Arg)
		}
	case "any":
		for _, sub := range c.Any {
			if err := d.validateParsed(s, sub, refs); err != nil {
				return err
			}
		}
	}
	return nil
}

// Waiting reports whether a node stops the story for the player (say, narrate, choice, battle, explore).
func Waiting(kind string) bool {
	return kind == "say" || kind == "narrate" || kind == "choice" || kind == "battle" || kind == "explore"
}

// ChapterIndex is the position of the chapter holding the node, or -1.
func (d *Data) ChapterIndex(node string) int {
	for i, c := range d.Chapters {
		for _, id := range slices.Concat(c.Nodes, c.Events) {
			if id == node {
				return i
			}
		}
	}
	return -1
}
