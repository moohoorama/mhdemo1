package content

import (
	"fmt"
	"strconv"
	"strings"
)

// Cond is one parsed condition string of a node (flag:x, !alive:liubei, choice:oath=결의, ...).
type Cond struct {
	Not       bool
	Op        string
	Arg, Arg2 string
	N         int
}

// NodeConds are the condition operators story nodes may use.
var nodeConds = map[string]bool{"flag": true, "alive": true, "dead": true, "party": true, "choice": true, "won": true}

// ParseCond reads "!?op:arg", where flag takes an optional ">=N" (default 1) and choice takes "node=option".
func ParseCond(s string) (Cond, error) {
	c := Cond{N: 1}
	if strings.HasPrefix(s, "!") {
		c.Not, s = true, s[1:]
	}
	op, arg, ok := strings.Cut(s, ":")
	if !ok || arg == "" || !nodeConds[op] {
		return Cond{}, fmt.Errorf("condition %q", s)
	}
	c.Op, c.Arg = op, arg
	switch op {
	case "flag":
		if name, n, ok := strings.Cut(arg, ">="); ok {
			v, err := strconv.Atoi(n)
			if err != nil || name == "" {
				return Cond{}, fmt.Errorf("condition %q", s)
			}
			c.Arg, c.N = name, v
		}
	case "choice":
		node, option, ok := strings.Cut(arg, "=")
		if !ok || node == "" || option == "" {
			return Cond{}, fmt.Errorf("condition %q", s)
		}
		c.Arg, c.Arg2 = node, option
	}
	return c, nil
}

// validateCond checks the references of a condition against the data.
func (d *Data) validateCond(s string) error {
	c, err := ParseCond(s)
	if err != nil {
		return err
	}
	switch c.Op {
	case "alive", "dead", "party":
		if !d.Has(c.Arg) {
			return fmt.Errorf("condition %q: no character %s", s, c.Arg)
		}
	case "choice":
		n := d.Node(c.Arg)
		if _, ok := n.Choices[c.Arg2]; n.Kind != "choice" || !ok {
			return fmt.Errorf("condition %q: no such choice", s)
		}
	case "won":
		if d.StageIndex(c.Arg) < 0 {
			return fmt.Errorf("condition %q: no stage %s", s, c.Arg)
		}
	}
	return nil
}

// Waiting reports whether a node stops the story for the player (say, narrate, choice, battle).
func Waiting(kind string) bool {
	return kind == "say" || kind == "narrate" || kind == "choice" || kind == "battle"
}

// ChapterIndex is the position of the chapter holding the node, or -1.
func (d *Data) ChapterIndex(node string) int {
	for i, c := range d.Chapters {
		for _, id := range c.Nodes {
			if id == node {
				return i
			}
		}
	}
	return -1
}
