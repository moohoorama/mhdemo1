// Package ai selects only commands exposed and evaluated by the core.
package ai

import (
	"fmt"
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
	for _, c := range options {
		s := -1.0
		switch c.Kind {
		case "duel":
			if !duels {
				continue
			}
			s = 2000
		case "attack", "skill":
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
			nearest, old := 999, 999
			for _, v := range o.UnitViews {
				if v.Faction == me.Faction || v.HP <= 0 {
					continue
				}
				nearest = min(nearest, distance(c.X, c.Y, v.X, v.Y))
				old = min(old, distance(me.X, me.Y, v.X, v.Y))
			}
			if me.Faction == "enemy" && old > 5 {
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
