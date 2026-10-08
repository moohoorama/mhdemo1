// Package ai selects only commands exposed and evaluated by the core.
package ai

import (
	"fmt"
	"srpg/internal/content"
	"srpg/internal/core"
)

const Version = core.AIPolicyVersion

func Choose(e *core.Engine, actor string) (core.Command, error) {
	return ChooseWithDuels(e, actor, true)
}
func ChooseWithDuels(e *core.Engine, actor string, duels bool) (core.Command, error) {
	o := e.Observe()
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
	best := options[0]
	score := -1e9
	var walk map[content.Point]int
	var threat map[content.Point]int
	for _, c := range options {
		s := -1.0
		switch c.Kind {
		case "attack", "skill":
			if e.DuelFor(c) {
				if !duels {
					continue
				}
				s = 2000
				break
			}
			p, err := e.Preview(c)
			if err != nil {
				continue
			}
			s = float64(p.MaxDamage)*p.Hit/100 - float64(p.Cost)*.3
			if me.Faction == "ally" && c.Target == o.Map.Boss {
				s *= 2
			}
			if s <= 0 {
				s = -5
			}
			if c.Skill == "소회복" {
				for _, v := range o.UnitViews {
					if v.ID == c.Target {
						if v.Faction != me.Faction {
							s = -5
							continue
						}
						s = float64(p.Healing) * p.Hit / 100 * .9
						if v.ID == "유비" && v.HP*2 < v.Stats.MaxHP {
							s *= 3
						}
						if me.Faction == "enemy" {
							s *= .25
						}
					}
				}
			}
			if c.Skill == "소회복" && p.Healing <= 0 {
				s = -5
			}
			if c.Skill == "반격" && me.Status["counter"].Turns == 0 {
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
				if v.ID == c.Target && v.ID == "유비" && v.HP*2 < v.Stats.MaxHP {
					s *= 3
				}
			}
			if p.Healing < 100 {
				s = -5
			}
		case "learn":
			if me.Faction == "ally" {
				priorities := map[string]float64{"철벽": 1200, "기병숙련": 1150, "보병숙련": 1150, "궁병숙련": 1150, "속공": 0, "방패": 900, "독공": 1100}
				s = priorities[c.Trait]
				if s == 0 {
					s = -5
				}
			}
		case "move":
			if walk == nil {
				walk = walking(e, o, me)
			}
			nearest, old := far(walk, c.X, c.Y), far(walk, me.X, me.Y)
			if me.Faction == "enemy" && old > 5 && o.Map.Goal != "rout" && !o.Map.Charge {
				s = -5
				break
			}

			s = float64(old-nearest)*3 - .1
			if me.Acted && (me.Faction == "enemy" || old <= 1) {
				s = -2
			} // Hold the selected firing position after acting.
			if me.ID == "유비" {
				s -= 1
			}
			if me.Faction == "ally" && (o.Map.Charge || me.ID == "유비") {
				if threat == nil {
					threat = threatened(e, o, me.Faction)
				}
				if o.Map.Charge {
					s += hold(e, o, me, c.X, c.Y, threat)
				} else if n := threat[content.Point{X: c.X, Y: c.Y}]; n >= 2 || n == 1 && me.HP*100 < me.Stats.MaxHP*70 {
					s -= 10 * float64(n) // 유비's retreat loses the battle
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

// hold scores a move when the enemy is coming: stepping into its reach without striking
// first is a loss, so is a cell many enemies reach (most of all for 유비, whose retreat
// loses the battle), and a hurt unit wants a cell that heals (village or castle floor).
func hold(e *core.Engine, o core.Observation, me *core.UnitView, x, y int, threat map[content.Point]int) float64 {
	s := 0.0
	lo, hi := e.AttackRange(me.ID)
	strike := false
	for _, v := range o.UnitViews {
		if d := distance(x, y, v.X, v.Y); v.Faction != me.Faction && v.HP > 0 && d >= lo && d <= hi {
			strike = true
		}
	}
	n := threat[content.Point{X: x, Y: y}]
	switch {
	case n > 0 && !strike:
		s -= 40 // outweighs any approach (3 per cell)
	case me.ID == "유비" && n >= 2, n >= 3:
		s -= 8 * float64(n-1)
	}
	if me.HP*100 < me.Stats.MaxHP*60 && (o.Map.Tiles[y][x] == 'k' || o.Map.Tiles[y][x] == 'v') {
		s += 6
	}
	return s
}

// walking is how far each cell is from the nearest opposing unit, walking around
// what me cannot enter (water, trees, walls) at its terrain costs.
func walking(e *core.Engine, o core.Observation, me *core.UnitView) map[content.Point]int {
	dist := map[content.Point]int{}
	queue := []content.Point{}
	for _, v := range o.UnitViews {
		if v.Faction != me.Faction && v.HP > 0 {
			p := content.Point{X: v.X, Y: v.Y}
			dist[p] = 0
			queue = append(queue, p)
		}
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
func Next(e *core.Engine) (core.Command, error) {
	o := e.Observe()
	if o.Phase != "battle" {
		return core.Command{}, fmt.Errorf("WrongPhase")
	}
	for _, u := range o.UnitViews {
		if u.Faction == o.Turn && u.HP > 0 && !u.Done {
			return Choose(e, u.ID)
		}
	}
	return core.Command{Kind: "end"}, nil
}
