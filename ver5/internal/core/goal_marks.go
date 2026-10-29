package core

import (
	"slices"

	"srpg/internal/content"
)

// GoalMark is a unit or a stage area the battle's goals turn on, for the objective guide. Win marks
// what a victory goal asks for, otherwise a way to lose; Pending marks a goal whose story condition
// (a flag, a round) does not hold yet.
type GoalMark struct {
	Unit         string          `json:",omitempty"`
	Cells        []content.Point `json:",omitempty"`
	Win, Pending bool
}

// GoalMarks are the units and areas the current stage's goals name: whom to defeat, who has to reach
// which area, whom to keep alive (the lord always) and which area the enemy must not reach.
func (e *Engine) GoalMarks() []GoalMark {
	if e.state.Stage < 0 || e.state.Stage >= len(e.data.Stages) {
		return nil
	}
	stage := e.data.Stages[e.state.Stage]
	var out []GoalMark
	add := func(goals []content.Goal, win bool) {
		for _, g := range goals {
			marks, pending := e.condMarks(stage, g.If, false)
			for _, m := range marks {
				m.Win, m.Pending = win, pending
				out = append(out, m)
			}
		}
	}
	goals := e.battleGoals()
	add(goals.Victory, true)
	add(goals.Defeat, false)
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.HP > 0 && u.Faction == "ally" && e.unitDef(u).Lord {
			out = append(out, GoalMark{Unit: u.ID})
		}
	}
	return out
}

// condMarks are the units and areas a goal's conditions name, and whether one of its story conditions
// does not hold yet. A negated battle condition marks too when negated is set: a bonus goal's "!fled"
// shows where the fleeing units head. A win or loss leaves it out, its colour would call that place
// a goal (a loss "defeated, !fled" is one escaping avoids).
func (e *Engine) condMarks(stage content.Stage, conds []string, negated bool) (marks []GoalMark, pending bool) {
	defer func() { marks = uniqueMarks(marks) }()
	for _, s := range conds {
		c, err := content.ParseCond(s)
		if err != nil {
			continue
		}
		switch {
		case c.Op == "flag" || c.Op == "round":
			pending = pending || e.holds(c, true) == c.Not
		case c.Not && !negated:
		case c.Op == "defeated" || c.Op == "alldefeated" || c.Op == "alive":
			for _, u := range e.matching(c.Arg) {
				if u.HP > 0 {
					marks = append(marks, GoalMark{Unit: u.ID})
				}
			}
		case c.Op == "fled":
			for _, u := range e.matching(c.Arg) {
				if u.HP > 0 {
					marks = append(marks, GoalMark{Unit: u.ID})
					if content.Leaves(u.Policy) {
						marks = append(marks, GoalMark{Cells: stage.GoalCells(u.Goal, u.GoalArea)})
					}
				}
			}
		case c.Op == "at" || c.Op == "allat":
			if e.sideOf(c.Arg) == "ally" {
				for _, u := range e.matching(c.Arg) {
					if u.HP > 0 {
						marks = append(marks, GoalMark{Unit: u.ID})
					}
				}
			}
			marks = append(marks, GoalMark{Cells: stage.GoalCells(nil, c.N)})
		}
	}
	return marks, pending
}

// uniqueMarks drops a unit or an area marked twice, as two fleeing carts bound for one corner are.
func uniqueMarks(marks []GoalMark) []GoalMark {
	var out []GoalMark
	for _, m := range marks {
		if !slices.ContainsFunc(out, func(o GoalMark) bool { return o.Unit == m.Unit && slices.Equal(o.Cells, m.Cells) }) {
			out = append(out, m)
		}
	}
	return out
}

// SubMarks are the units and areas the stage's bonus objectives name, for the objective guide; a duel's
// officers have their own marks.
func (e *Engine) SubMarks() []GoalMark {
	if e.state.Stage < 0 || e.state.Stage >= len(e.data.Stages) {
		return nil
	}
	stage := e.data.Stages[e.state.Stage]
	var out []GoalMark
	for _, o := range stage.Sub {
		marks, pending := e.condMarks(stage, o.If, true)
		for _, m := range marks {
			m.Pending = pending
			out = append(out, m)
		}
	}
	return out
}

// DuelMark is an officer on the field who has a stage duel still to fight; officers whose duels join
// them (DuelGroups) share a Group.
type DuelMark struct {
	Unit  string
	Group int
}

// DuelMarks are the officers of the stage's pending duels standing on the field, the ally's supports
// with him.
func (e *Engine) DuelMarks() []DuelMark {
	if e.state.Stage < 0 || e.state.Stage >= len(e.data.Stages) {
		return nil
	}
	stage := e.data.Stages[e.state.Stage]
	groups := stage.DuelGroups()
	var out []DuelMark
	seen := map[string]bool{}
	for i, d := range stage.Duels {
		if e.state.Executed["duel-"+d.ID] {
			continue
		}
		for _, id := range append([]string{d.Ally, d.Enemy}, d.Supports...) {
			if u := e.unit(id); u != nil && u.HP > 0 && !seen[u.ID] {
				seen[u.ID] = true
				out = append(out, DuelMark{Unit: u.ID, Group: groups[i]})
			}
		}
	}
	return out
}
