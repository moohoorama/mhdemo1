package ai

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

// RushNext plays the player's side the way a careless player would: every officer strikes the
// nearest enemy in reach, or else walks toward the nearest one, and never uses a skill or an item.
// The AI allies and the enemy play as Next does. Early stages should be won this way.
func RushNext(e *core.Engine) (core.Command, error) {
	o := e.Observe()
	if o.Phase != "battle" || len(o.Learning) > 0 {
		return Next(e)
	}
	for _, u := range o.UnitViews {
		if u.ActsIn(o.Turn, o.Army) && u.HP > 0 && !u.Done && u.Faction == "ally" && !u.AI {
			return rush(e, o, u), nil
		}
	}
	return Next(e)
}

func rush(e *core.Engine, o core.Observation, me core.UnitView) core.Command {
	at := map[string]content.Point{}
	for _, v := range o.UnitViews {
		at[v.ID] = content.Point{X: v.X, Y: v.Y}
	}
	var walk map[content.Point]int
	best, score := core.Command{Kind: "wait", Actor: me.ID}, 0
	for _, c := range e.LegalActions(me.ID).Commands {
		s := 0
		switch c.Kind {
		case "attack":
			p := at[c.Target]
			s = 1000 - distance(me.X, me.Y, p.X, p.Y)
		case "move":
			if walk == nil {
				walk = walking(e, o, &me)
			}
			s = far(walk, me.X, me.Y) - far(walk, c.X, c.Y)
		}
		if s > score {
			best, score = c, s
		}
	}
	return best
}
