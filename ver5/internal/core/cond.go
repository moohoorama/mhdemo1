package core

import "srpg/internal/content"

// holdsAll reports whether every condition string holds. Strings were validated at load.
func (e *Engine) holdsAll(conds []string) bool {
	for _, s := range conds {
		c, err := content.ParseCond(s)
		if err != nil || e.holds(c) == c.Not {
			return false
		}
	}
	return true
}

func (e *Engine) holds(c content.Cond) bool {
	switch c.Op {
	case "flag":
		return e.state.Flags[c.Arg] >= c.N
	case "alive":
		o := e.char(c.Arg)
		return o != nil && !o.Dead
	case "dead":
		o := e.char(c.Arg)
		return o != nil && o.Dead
	case "party":
		o := e.char(c.Arg)
		return o != nil && !o.Dead && !o.Temp && e.playable(o)
	case "choice":
		return e.state.Choices[c.Arg] == c.Arg2
	case "won":
		return e.state.Executed["won-"+c.Arg]
	}
	return false
}
