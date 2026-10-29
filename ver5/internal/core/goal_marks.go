package core

import "srpg/internal/content"

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
			var marks []GoalMark
			pending := false
			for _, s := range g.If {
				c, err := content.ParseCond(s)
				if err != nil {
					continue
				}
				switch {
				case c.Not:
				case c.Op == "defeated" || c.Op == "alldefeated":
					for _, u := range e.matching(c.Arg) {
						if u.HP > 0 {
							marks = append(marks, GoalMark{Unit: u.ID})
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
				case c.Op == "flag" || c.Op == "round":
					pending = pending || !e.holds(c, true)
				}
			}
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
