// Package ai selects only commands exposed and evaluated by the core.
package ai

import (
	"fmt"
	"slices"
	"srpg/internal/content"
	"srpg/internal/core"
)

const Version = core.AIPolicyVersion

func Choose(e *core.Engine, actor string) (core.Command, error) {
	return ChooseWithDuels(e, actor, true)
}
func ChooseWithDuels(e *core.Engine, actor string, duels bool) (core.Command, error) {
	o := e.Observe()
	if len(o.Learning) > 0 {
		return chooseLearning(e, o), nil
	}
	var me *core.UnitView
	for i := range o.UnitViews {
		if o.UnitViews[i].ID == actor {
			me = &o.UnitViews[i]
		}
	}
	if me == nil {
		return core.Command{}, fmt.Errorf("InvalidActor")
	}
	options := e.LegalActions(actor).Commands
	if len(options) == 0 {
		return core.Command{}, fmt.Errorf("NoLegalActions")
	}
	if me.Policy == "react" && !inReach(e, o, me) {
		return core.Command{Kind: "wait", Actor: actor}, nil
	}
	best := options[0]
	score := -1e9
	var walk, goal map[content.Point]int
	var threat map[content.Point]int
	for _, c := range options {
		s := -1.0
		switch c.Kind {
		case "attack", "skill":
			if me.Policy == "flee" {
				continue // a fleeing unit only runs
			}
			if duels && me.DuelFirst && e.DuelPending(me.ID, c.Target) {
				continue // a duel partner is met in the duel, not shot or struck from afar
			}
			p, err := e.Preview(c)
			if err != nil {
				continue
			}
			s = float64(p.MaxDamage)*p.Hit/100 - float64(p.Cost)*.3
			if me.Faction == "ally" && slices.Contains(o.Map.VictoryTargets(), c.Target) {
				s *= 2
			}
			if s <= 0 {
				s = -5
			}
			skill := e.Data().Skills[c.Skill]
			if skill.Effect == "heal" {
				for _, v := range o.UnitViews {
					if v.ID == c.Target {
						if v.Faction != me.Faction {
							s = -5
							continue
						}
						s = float64(p.Healing) * p.Hit / 100 * .9
						if lord(e, v) && v.HP*2 < v.Stats.MaxHP {
							s *= 3
						}
						if me.Faction == "enemy" {
							s *= .25
						}
					}
				}
			}
			if skill.Effect == "heal" && p.Healing <= 0 {
				s = -5
			}
			if skill.Effect == "counter" && me.Status["counter"].Turns == 0 {
				for _, v := range o.UnitViews {
					if v.Faction != me.Faction && v.HP > 0 && distance(me.X, me.Y, v.X, v.Y) <= 2 {
						s = 35
					}
				}
			}
		case "item":
			p, err := e.Preview(c)
			if err != nil {
				continue
			}
			s = float64(p.Healing) * .85
			for _, v := range o.UnitViews {
				if v.ID == c.Target && v.Faction != me.Faction {
					s = -5
				}
			}
			for _, v := range o.UnitViews {
				if v.ID == c.Target && lord(e, v) && v.HP*2 < v.Stats.MaxHP {
					s *= 3
				}
			}
			if p.Healing < 100 {
				s = -5
			}
		case "move":
			if !duels && e.DuelAt(me.ID, c.X, c.Y) != "" {
				continue // moving next to a duel partner would start the duel
			}
			if duels && me.DuelFirst {
				for _, v := range o.UnitViews {
					if e.DuelPending(me.ID, v.ID) && distance(c.X, c.Y, v.X, v.Y) == 1 {
						s = 1500 // a cell where the duel can start comes before anything else
					}
				}
				if s > 0 {
					break
				}
			}
			if me.Policy == "march" || me.Policy == "flee" {
				if goal == nil {
					goal = walkingFrom(e, o, me, o.Map.GoalCells(me.Goal, me.GoalArea))
				}
				s = float64(far(goal, me.X, me.Y)-far(goal, c.X, c.Y))*3 - .1 - .01*float64(distance(me.X, me.Y, c.X, c.Y)) // of equal cells, the one fewest steps away
				break
			}
			if walk == nil {
				walk = walking(e, o, me)
			}
			s = approach(e, o, me, c.X, c.Y, walk, duels)
			if lord(e, *me) {
				if threat == nil {
					threat = threatened(e, o, me.Faction)
				}
				s -= 1
				if n := threat[content.Point{X: c.X, Y: c.Y}]; n >= 2 || n == 1 && me.HP*100 < me.Stats.MaxHP*70 {
					s -= 50 * float64(n) // the lord's retreat loses the battle, so not even a cell to strike from
				}
			}

		case "wait":
			s = -.5
		}
		if s > score {
			score = s
			best = c
		}
	}
	return best, nil
}

// inReach reports whether an opposing unit stands where me could strike it this turn (its move
// and then its reach), which is when a react unit leaves its post.
func inReach(e *core.Engine, o core.Observation, me *core.UnitView) bool {
	reach := map[content.Point]bool{}
	for _, p := range e.Threat(me.ID) {
		reach[p] = true
	}
	for _, v := range o.UnitViews {
		if v.HP > 0 && v.Faction != me.Faction && reach[content.Point{X: v.X, Y: v.Y}] {
			return true
		}
	}
	return false
}

func lord(e *core.Engine, v core.UnitView) bool {
	return v.Faction == "ally" && e.Data().Characters[v.Def].Lord
}

// Learning is the command that answers an open learning window; false when none is open.
func Learning(e *core.Engine) (core.Command, bool) {
	o := e.Observe()
	if len(o.Learning) == 0 {
		return core.Command{}, false
	}
	return chooseLearning(e, o), true
}

// chooseLearning spends a learner's points on the dearest trait it can afford, then closes the window.
func chooseLearning(e *core.Engine, o core.Observation) core.Command {
	var best core.Command
	score := -1
	for _, c := range e.LegalActions("").Commands {
		s := 0
		if c.Kind == "learn" {
			s = e.Data().Traits[c.Trait].Cost
		}
		if s > score {
			score, best = s, c
		}
	}
	return best
}
func distance(x, y, a, b int) int { return abs(x-a) + abs(y-b) }

// threatened counts, for every cell, the opposing units that could attack it next turn.
func threatened(e *core.Engine, o core.Observation, faction string) map[content.Point]int {
	cells := map[content.Point]int{}
	for _, v := range o.UnitViews {
		if v.Faction != faction && v.HP > 0 {
			for _, p := range e.Threat(v.ID) {
				cells[p]++
			}
		}
	}
	return cells
}

// approach scores a move toward the other side. A cell to strike from comes first, the farther from
// the nearest opponent the better, so an archer shoots from the full reach of its bow; then a cell
// nearer the closest opponent. A cell inside the unit's least reach of an opponent is never taken:
// from there it could not strike that one at all.
func approach(e *core.Engine, o core.Observation, me *core.UnitView, x, y int, walk map[content.Point]int, duels bool) float64 {
	lo, hi := e.AttackRange(me.ID)
	near, strike := 999, false
	for _, v := range o.UnitViews {
		if v.Faction == me.Faction || v.HP <= 0 {
			continue
		}
		d := distance(x, y, v.X, v.Y)
		if d < lo {
			return -5
		}
		near = min(near, d)
		strike = strike || d <= hi && !(duels && me.DuelFirst && e.DuelPending(me.ID, v.ID))
	}
	if strike {
		return 40 + float64(near) // above any approach (3 a cell), under any blow from where it stands
	}
	return float64(far(walk, me.X, me.Y)-far(walk, x, y))*3 - .1
}

// walking is how far each cell is from the nearest opposing unit, walking around
// what me cannot enter (water, trees, walls) at its terrain costs.
func walking(e *core.Engine, o core.Observation, me *core.UnitView) map[content.Point]int {
	var seeds []content.Point
	for _, v := range o.UnitViews {
		if v.Faction != me.Faction && v.HP > 0 {
			seeds = append(seeds, content.Point{X: v.X, Y: v.Y})
		}
	}
	return walkingFrom(e, o, me, seeds)
}

// walkingFrom is how far each cell is from the nearest seed cell, walking as me would.
func walkingFrom(e *core.Engine, o core.Observation, me *core.UnitView, seeds []content.Point) map[content.Point]int {
	dist := map[content.Point]int{}
	queue := []content.Point{}
	for _, p := range seeds {
		dist[p] = 0
		queue = append(queue, p)
	}
	// costs are small integers, so a relaxing FIFO search settles quickly
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range []content.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			q := content.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if q.X < 0 || q.Y < 0 || q.X >= o.Map.Width || q.Y >= o.Map.Height {
				continue
			}
			// the cost of stepping p → q is paid entering q when walking toward the enemy,
			// i.e. entering p from q; use p's cost so a blocked cell is never passed through
			cost := e.StepCost(me.ID, p.X, p.Y)
			if dist[p] == 0 {
				cost = 1 // the enemy's own cell
			}
			if cost == 0 || e.StepCost(me.ID, q.X, q.Y) == 0 && (q.X != me.X || q.Y != me.Y) {
				continue
			}
			if old, ok := dist[q]; !ok || dist[p]+cost < old {
				dist[q] = dist[p] + cost
				queue = append(queue, q)
			}
		}
	}
	return dist
}

func far(dist map[content.Point]int, x, y int) int {
	if d, ok := dist[content.Point{X: x, Y: y}]; ok {
		return d
	}
	return 999
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func Step(e *core.Engine) (core.Result, error) {
	c, err := Next(e)
	if err != nil {
		return core.Result{}, err
	}
	return e.Apply(c)
}

// Next is the command Step would apply: the first ready unit's choice, or ending the faction.
func Next(e *core.Engine) (core.Command, error) { return NextWithDuels(e, true) }

// NextWithDuels is Next with stage duels allowed or not.
func NextWithDuels(e *core.Engine, duels bool) (core.Command, error) {
	o := e.Observe()
	if o.Phase == "deployment" {
		commands, err := e.DeploymentPlan()
		if err != nil {
			return core.Command{}, err
		}
		return commands[0], nil
	}
	if o.Phase != "battle" {
		return core.Command{}, fmt.Errorf("WrongPhase")
	}
	if len(o.Learning) > 0 {
		return chooseLearning(e, o), nil
	}
	for _, u := range o.UnitViews {
		if u.ActsIn(o.Turn, o.Army) && u.HP > 0 && !u.Done {
			return ChooseWithDuels(e, u.ID, duels)
		}
	}
	return core.Command{Kind: "end"}, nil
}
