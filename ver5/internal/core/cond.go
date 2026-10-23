package core

import "srpg/internal/content"

// holdsAll reports whether every story condition holds (nodes pick branches with it).
func (e *Engine) holdsAll(conds []string) bool { return e.holdsIn(conds, false) }

// holdsIn reports whether every condition string holds; battle conditions read the units on the
// map, story ones the roster. Strings were validated at load.
func (e *Engine) holdsIn(conds []string, battle bool) bool {
	for _, s := range conds {
		c, err := content.ParseCond(s)
		if err != nil || e.holds(c, battle) == c.Not {
			return false
		}
	}
	return true
}

// flag is a story flag's value, a battle's own unconfirmed setting first.
func (e *Engine) flag(name string) int {
	if v, ok := e.state.PendingFlags[name]; ok {
		return v
	}
	return e.state.Flags[name]
}

// matching are the units a condition or action names: the unit with that id, every unit of that
// group, or every unit of a side ("ally", "enemy").
func (e *Engine) matching(target string) []*Unit {
	var out []*Unit
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.ID == target || u.Group == target || u.Faction == target {
			out = append(out, u)
		}
	}
	return out
}

func (e *Engine) inArea(u *Unit, id int) bool {
	for _, a := range e.data.Stages[e.state.Stage].Areas {
		if a.ID == id {
			return u.X >= a.Rect.X && u.Y >= a.Rect.Y && u.X < a.Rect.X+a.Rect.W && u.Y < a.Rect.Y+a.Rect.H
		}
	}
	return false
}

func (e *Engine) holds(c content.Cond, battle bool) bool {
	switch c.Op {
	case "flag":
		return e.flag(c.Arg) >= c.N
	case "choice":
		return e.state.Choices[c.Arg] == c.Arg2
	case "won":
		return e.state.Executed["won-"+c.Arg]
	case "alive":
		if battle {
			for _, u := range e.matching(c.Arg) {
				if u.HP > 0 {
					return true
				}
			}
			return false
		}
		o := e.char(c.Arg)
		return o != nil && !o.Dead
	case "dead":
		o := e.char(c.Arg)
		return o != nil && o.Dead
	case "party":
		o := e.char(c.Arg)
		return o != nil && !o.Dead && !o.Temp && e.playable(o)
	case "round":
		return e.state.Round >= c.N
	case "turnstart":
		return e.turnStarted == c.Arg
	case "defeated":
		for _, u := range e.matching(c.Arg) {
			if u.HP <= 0 {
				return true
			}
		}
		return false
	case "alldefeated":
		units := e.matching(c.Arg)
		for _, u := range units {
			if u.HP > 0 {
				return false
			}
		}
		return len(units) > 0
	case "at", "allat":
		any, all := false, true
		for _, u := range e.matching(c.Arg) {
			if u.HP > 0 {
				in := e.inArea(u, c.N)
				any, all = any || in, all && in
				if c.Op == "allat" && !in {
					return false
				}
			}
		}
		return c.Op == "at" && any || c.Op == "allat" && any && all
	case "adjacent":
		for _, a := range e.matching(c.Arg) {
			for _, b := range e.matching(c.Arg2) {
				if a != b && a.HP > 0 && b.HP > 0 && dist(a.X, a.Y, b.X, b.Y) <= 1 {
					return true
				}
			}
		}
		return false
	case "hp":
		for _, u := range e.matching(c.Arg) {
			if u.HP > 0 && u.HP*100 < e.stats(u).MaxHP*c.N {
				return true
			}
		}
		return false
	case "duel":
		return e.state.Executed["duel-"+c.Arg]
	case "fled":
		for _, u := range e.matching(c.Arg) {
			if e.state.Executed["fled-"+e.data.Stages[e.state.Stage].ID+"-"+u.ID] {
				return true
			}
		}
		return false
	case "any":
		for _, sub := range c.Any {
			if e.holds(sub, battle) != sub.Not {
				return true
			}
		}
		return false
	}
	return false
}
