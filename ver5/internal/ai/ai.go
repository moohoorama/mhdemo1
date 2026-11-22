// Package ai selects only commands exposed and evaluated by the core.
package ai

import (
	"fmt"
	"hash/fnv"
	"slices"
	"srpg/internal/content"
	"srpg/internal/core"
	"sync"
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
	if Player != nil && me.Faction == "ally" && !me.AI {
		return chooseEval(e, o, me, options, duels), nil
	}
	if me.Policy == "react" && !inReach(e, o, me) {
		return core.Command{Kind: "wait", Actor: actor}, nil
	}
	best := options[0]
	score := -1e9
	ties := 0
	dry := spentCaster(e, me)
	prefer := e.Data().Classes[me.Class].Prefer
	spells := offense(e, me)
	scheme := prefer == content.KindStatus && schemes(e, o, me, options)
	reaches := sync.OnceValue(func() bool { return casts(e, o, me, options, spells) })
	var walk, goal map[content.Point]int
	var threat map[content.Point]int
	var stop content.Point
	onWay := false
	for _, c := range options {
		s := -1.0
		switch c.Kind {
		case "attack", "skill":
			if content.Leaves(me.Policy) {
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
			switch {
			case skill.Kind == content.KindMagic:
				if prefer == content.KindMagic && s > 0 {
					s *= 1.5 // a staff officer burns, floods and buries before it hinders
				}
				if scheme {
					s = -5 // a schemer keeps its points for the scheme that still lands on someone
				}
			case skill.Kind == content.KindStatus && me.Stats.Mind > me.Stats.Attack:
				s = hinder(e, o, me, skill, p)
				switch {
				case s <= 0:
				case prefer == content.KindStatus:
					s *= 3 // a schemer's own way of fighting comes before a plain blow
				case prefer != "":
					s *= .5 // the other casters hinder only where it is worth twice their own spell
				}
			case skill.Effect == "heal":
				s = mend(e, o, me, p, prefer)
				if s > 0 && (prefer == content.KindMagic || prefer == content.KindStatus) && reaches() {
					s = -5 // a caster that fights with spells heals only while none of them reaches an opponent this turn
				}
			case skill.Effect == "cure":
				s = relieve(e, o, me, p, prefer)
			}
			if prefer == content.KindHeal && skill.Kind == content.KindHeal && s > 0 {
				s += 50 // a healer tends the wounded before it takes a step toward the fight (approach gives 40 and a bit)
			}
			if skill.Effect == "meditate" {
				s = -5
				if dry && (me.HP-e.HPCost(me.ID, c.Skill))*5 > me.Stats.MaxHP {
					s = 30 // out of spell points, it gathers them again where it stands, while that leaves it a fifth of its strength
				}
			}
			if skill.Effect == "counter" && me.Status["counter"].Turns == 0 {
				for _, v := range o.UnitViews {
					if v.Faction != me.Faction && v.HP > 0 && distance(me.X, me.Y, v.X, v.Y) <= 2 {
						s = 35
					}
				}
			}
		case "item":
			if !me.UseItems || me.Policy == "run" {
				continue // only a unit told to may eat what it carries, and a running one never stops for it
			}
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
			if me.Policy == "march" || content.Leaves(me.Policy) {
				if goal == nil {
					var way []content.Point
					goal, way = plan(e, o, me, o.Map.GoalCells(me.Goal, me.GoalArea))
					stop, onWay = farthest(way, options)
				}
				s = float64(far(goal, me.X, me.Y)-far(goal, c.X, c.Y))*3 - .1 - .01*float64(distance(me.X, me.Y, c.X, c.Y)) // of equal cells, the one fewest steps away
				if onWay && stop == (content.Point{X: c.X, Y: c.Y}) {
					s += 3 // as far along the whole planned walk as this turn goes
				}
				break
			}
			if dry {
				s = -1 // a caster with no spell left does not walk into a melee: it waits
				break
			}
			if walk == nil {
				walk = walking(e, o, me)
			}
			s = approach(e, o, me, c.X, c.Y, walk, duels, spells)
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
			score, best, ties = s, c, 1
		} else if s == score {
			ties++
			if tieBreak(o, me.ID, ties)%uint64(ties) == 0 {
				best = c // of equally good actions, a different one each round and battle: a player sees every spell
			}
		}
	}
	return best, nil
}

// tieBreak is the n-th draw among an actor's equally scored actions, fixed by the battle's seed and the round so a
// reloaded turn plays the same, without touching the engine's own random state.
func tieBreak(o core.Observation, actor string, n int) uint64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d/%d/%s/%d", o.Seed, o.Round, actor, n)
	x := h.Sum64()
	x ^= x >> 31 // spread the low bits the modulo reads
	x *= 0x9e3779b97f4a7c15
	return x ^ x>>29
}

// hinder is what a status spell is worth to a caster: for every opponent it lands on that is not under it
// yet, what the status takes from that opponent over the turns it lasts (burden).
func hinder(e *core.Engine, o core.Observation, me *core.UnitView, skill content.Skill, p core.Preview) float64 {
	sum := 0.0
	for _, v := range o.UnitViews {
		if !slices.Contains(p.Targets, v.ID) || v.Faction == me.Faction || v.HP <= 0 || v.Status[skill.Effect].Turns > 0 {
			continue
		}
		sum += burden(e, o, v, skill.Effect, skill.Duration)
	}
	if sum <= 0 {
		return -5
	}
	return sum*p.Hit/100 - float64(p.Cost)*.3
}

// burden is what a status takes from v over the turns it lasts (two at most): the blows v would deal that it
// takes away or weakens — all of them for confusion, an officer's own kind of blow for disarm and seal, a share
// for a lowered stat — or the strength poison and burns bleed.
func burden(e *core.Engine, o core.Observation, v core.UnitView, status string, turns int) float64 {
	rules := e.Data().Rules
	n := float64(min(max(turns, 1), 2))
	switch status {
	case "poison":
		return float64(v.Stats.MaxHP*rules.PoisonPercent/100) * n
	case "burn":
		return float64(v.Stats.MaxHP*rules.BurnPercent/100) * n
	}
	harm := 0.0
	strike := e.Strike
	if status == "confusion" {
		strike = e.StrikeAfterConfusion
	}
	for _, u := range o.UnitViews {
		if u.Faction != v.Faction && u.HP > 0 {
			harm = max(harm, strike(v.ID, u.ID))
		}
	}
	w := float64(rules.BuffPercent) / 100
	switch status {
	case "confusion":
		w = 1
	case "disarm":
		w = map[bool]float64{true: 1, false: .1}[v.Stats.Attack >= v.Stats.Mind]
	case "seal":
		w = map[bool]float64{true: 1, false: .1}[v.Stats.Mind > v.Stats.Attack]
	case "root":
		w = .3
	}
	return w * harm * n
}

// schemes tells a schemer that one of its status spells this turn lands on an opponent not under it yet;
// while none does (all already hindered, or none in reach) its attack spells are free to burn its points.
func schemes(e *core.Engine, o core.Observation, me *core.UnitView, options []core.Command) bool {
	if me.Stats.Mind <= me.Stats.Attack {
		return false
	}
	for _, c := range options {
		skill := e.Data().Skills[c.Skill]
		if c.Kind != "skill" || skill.Kind != content.KindStatus {
			continue
		}
		if p, err := e.Preview(c); err == nil && hinder(e, o, me, skill, p) > 0 {
			return true
		}
	}
	return false
}

// mend is what a healing spell is worth: the strength it gives back to allies worn down enough to need it, a
// quarter of their strength gone — half for a caster whose class fights with spells, so a staff officer does
// not stand behind its line topping up scratches instead of casting. Three times for a lord under half.
// An enemy only heals at a quarter of that (half for a healer's class): its own line is not worth the
// player's time to wear down twice.
func mend(e *core.Engine, o core.Observation, me *core.UnitView, p core.Preview, prefer string) float64 {
	sum := 0.0
	for _, v := range o.UnitViews {
		if !slices.Contains(p.Targets, v.ID) || v.Faction != me.Faction || v.HP <= 0 {
			continue
		}
		worn := v.HP*4 < v.Stats.MaxHP*3
		if prefer == content.KindMagic || prefer == content.KindStatus {
			worn = v.HP*2 < v.Stats.MaxHP
		}
		if !worn {
			continue
		}
		h := float64(min(p.Healing/max(1, len(p.Targets)), v.Stats.MaxHP-v.HP))
		if lord(e, v) && v.HP*2 < v.Stats.MaxHP {
			h *= 3
		}
		sum += h
	}
	if sum <= 0 {
		return -5
	}
	s := sum * p.Hit / 100 * .9
	if me.Faction == "enemy" {
		s *= map[bool]float64{true: .5, false: .25}[prefer == content.KindHeal]
	}
	return s
}

// relieve is what a cure is worth: for every ally it reaches, what its harmful statuses still take from it over
// the turns they have left (burden). Twice for a healer's class, whose own work it is.
func relieve(e *core.Engine, o core.Observation, me *core.UnitView, p core.Preview, prefer string) float64 {
	sum := 0.0
	for _, v := range o.UnitViews {
		if !slices.Contains(p.Targets, v.ID) || v.Faction != me.Faction || v.HP <= 0 {
			continue
		}
		for k, st := range v.Status {
			if content.Harmful[k] && st.Turns > 0 {
				sum += burden(e, o, v, k, st.Turns)
			}
		}
	}
	if sum <= 0 {
		return -5
	}
	s := sum*p.Hit/100 - float64(p.Cost)*.3
	if prefer == content.KindHeal {
		s *= 2
	}
	return s
}

// casts tells whether one of spells reaches an opponent this turn, from where me stands or from a cell it
// can move to.
func casts(e *core.Engine, o core.Observation, me *core.UnitView, options []core.Command, spells []string) bool {
	for _, c := range options {
		if k := e.Data().Skills[c.Skill].Kind; c.Kind == "skill" && slices.Contains(spells, c.Skill) && (k == content.KindMagic || k == content.KindStatus) {
			return true
		}
	}
	for _, c := range options {
		if c.Kind != "move" {
			continue
		}
		for _, v := range o.UnitViews {
			if v.Faction == me.Faction || v.HP <= 0 {
				continue
			}
			for _, id := range spells {
				if e.CanCastFrom(me.ID, id, content.Point{X: c.X, Y: c.Y}, v.X, v.Y) {
					return true
				}
			}
		}
	}
	return false
}

// offense lists the attack and status spells a caster (a unit whose mind outweighs its attack) can pay for now:
// a cell they reach an opponent from is where it walks to, as a cell to strike from is for a warrior.
func offense(e *core.Engine, me *core.UnitView) []string {
	if me.Stats.Mind <= me.Stats.Attack {
		return nil
	}
	var out []string
	for _, id := range me.Skills {
		s := e.Data().Skills[id]
		if (s.Kind == content.KindMagic || s.Kind == content.KindStatus) && s.Mode != content.ModeSelf && e.SkillCost(me.ID, id) <= me.MP {
			out = append(out, id)
		}
	}
	return out
}

// spentCaster tells a caster (a unit whose mind outweighs its attack, as 서생 and 군사 do) that has attack or
// status spells but too few spell points for any of them now.
func spentCaster(e *core.Engine, me *core.UnitView) bool {
	if me.Stats.Mind <= me.Stats.Attack {
		return false
	}
	has := false
	for _, id := range me.Skills {
		if k := e.Data().Skills[id].Kind; k != content.KindMagic && k != content.KindStatus {
			continue
		}
		has = true
		if e.SkillCost(me.ID, id) <= me.MP {
			return false
		}
	}
	return has
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

// approach scores a move toward the other side. A cell to strike from comes first (or, for a caster, to cast
// one of spells from), the farther from the nearest opponent the better, so an archer shoots from the full
// reach of its bow; then a cell nearer the closest opponent. A cell inside the unit's least reach of an
// opponent is never taken: from there it could not strike that one at all.
func approach(e *core.Engine, o core.Observation, me *core.UnitView, x, y int, walk map[content.Point]int, duels bool, spells []string) float64 {
	lo, _ := e.AttackRange(me.ID)
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
		strike = strike || e.CanStrikeFrom(me.ID, content.Point{X: x, Y: y}, v.X, v.Y) && !(duels && me.DuelFirst && e.DuelPending(me.ID, v.ID))
		for _, id := range spells {
			strike = strike || e.CanCastFrom(me.ID, id, content.Point{X: x, Y: y}, v.X, v.Y)
		}
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
	return walkingFromWith(e, o, me, seeds, nil)
}

// walkingFromWith is walkingFrom with extra paid for entering a cell, when extra is set.
func walkingFromWith(e *core.Engine, o core.Observation, me *core.UnitView, seeds []content.Point, extra func(content.Point) int) map[content.Point]int {
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
			if extra != nil {
				cost += extra(p)
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
