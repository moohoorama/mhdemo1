package ai

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
)

// Eval scores the player side's choices by expected strength: every unit is worth its
// power (attack or mind) × (r^N + M)/(M+1) at HP ratio r, nothing once it falls, and a choice scores the
// change it makes to own strength minus the opponents'. Threat picks whose standing cell is weighed by
// the blows that can reach it: "lord" only the lord, "officers" every named officer, each opponent that
// reaches counting with Top its strongest few (0 all of them), the loss weighed by W. With a "-lethal"
// suffix only a cell where those blows could make the unit fall counts, at its whole worth.
//
// XP and Grow steer the experience: an action's expected experience is worth XP × Grow[officer] × the
// actor's own worth per level it brings, so the blows go to the officers the party means to raise.
type Eval struct {
	N, M   float64
	Threat string
	Top    int
	W      float64
	XP     float64
	Grow   map[string]float64
}

// Player scores the choices of the player's own units (auto battle, the balance tools; not the AI
// allies, who play by their policies like the enemy). nil plays them by the greedy policy too.
var Player = &Eval{N: .5, M: 1, Threat: "officers-lethal", W: 1}

// ParseEval reads "n,m,threat,top,w" (as 0.5,1,officers,2,1) for trying other weights; empty is nil.
func ParseEval(s string) (*Eval, error) {
	if s == "" {
		return nil, nil
	}
	var x Eval
	_, err := fmt.Sscanf(strings.ReplaceAll(s, ",", " "), "%g %g %s %d %g", &x.N, &x.M, &x.Threat, &x.Top, &x.W)
	return &x, err
}

const lordWeight = 5 // the lord's fall loses the battle

type evaluator struct {
	*Eval
	e       *core.Engine
	o       core.Observation
	me      *core.UnitView
	reaches map[string]map[content.Point]bool // opponent → the cells it can strike next turn, as asked
	strikes map[[2]string]float64             // damage for a unit pair during this evaluation only
	victory []string
	vital   []string              // own units whose fall loses the battle besides the lord (a defeat goal's defeated:X)
	goal    map[content.Point]int // walking distance to the area a victory goal sends me to; nil without one
	escort  map[content.Point]int // walking distance to the stretch of way ahead of another unit a victory goal sends somewhere
	clear   []string              // opponents near that stretch
	pending bool                  // that goal's story condition does not hold yet
	stop    content.Point         // the farthest cell this turn's move reaches along my whole walk to that goal
	onWay   bool                  // a move reaches that walk at all
}

func (x *evaluator) strike(actor, target string) float64 {
	key := [2]string{actor, target}
	if damage, ok := x.strikes[key]; ok {
		return damage
	}
	damage := x.e.Strike(actor, target)
	if x.strikes == nil {
		x.strikes = make(map[[2]string]float64)
	}
	x.strikes[key] = damage
	return damage
}

func (x *evaluator) opponents(u *core.UnitView) []core.UnitView {
	var out []core.UnitView
	for _, v := range x.o.UnitViews {
		if v.Faction != u.Faction && v.HP > 0 {
			out = append(out, v)
		}
	}
	return out
}

func (x *evaluator) worth(u *core.UnitView, hp int) float64 {
	if hp <= 0 {
		return 0
	}
	r := min(1, float64(hp)/float64(max(1, u.Stats.MaxHP)))
	alive := x.M // the share a unit is worth just for standing, lost all at once when it falls
	if u.Faction == x.me.Faction && (lord(x.e, *u) || slices.Contains(x.vital, u.ID)) {
		alive *= lordWeight // its fall loses the battle; its wounds weigh like anyone's
	}
	w := float64(max(u.Stats.Attack, u.Stats.Mind)) * (math.Pow(r, x.N) + alive) / (x.M + 1)
	if u.Faction == x.me.Faction {
		return w
	}
	if slices.Contains(x.victory, u.ID) || slices.Contains(x.clear, u.ID) {
		w *= 2
	}
	if x.menaces(u) {
		w *= 2
	}
	return w
}

// menaces reports whether opponent v can strike, next turn, an own unit whose fall loses the battle.
func (x *evaluator) menaces(v *core.UnitView) bool {
	reach := x.reachOf(v.ID)
	for _, u := range x.o.UnitViews {
		if u.Faction == x.me.Faction && u.HP > 0 && (lord(x.e, u) || slices.Contains(x.vital, u.ID)) && reach[content.Point{X: u.X, Y: u.Y}] {
			return true
		}
	}
	return false
}

// reachOf is the cells opponent id can strike next turn.
func (x *evaluator) reachOf(id string) map[content.Point]bool {
	if x.reaches == nil {
		x.reaches = map[string]map[content.Point]bool{}
	}
	set, ok := x.reaches[id]
	if !ok {
		set = map[content.Point]bool{}
		for _, c := range x.e.Threat(id) {
			set[c] = true
		}
		x.reaches[id] = set
	}
	return set
}

// incoming is the damage u may take standing at p before its side moves again.
func (x *evaluator) incoming(u *core.UnitView, p content.Point) int {
	var blows []float64
	for _, v := range x.opponents(x.me) {
		if x.reachOf(v.ID)[p] {
			blows = append(blows, x.strike(v.ID, u.ID))
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(blows)))
	if x.Top > 0 && len(blows) > x.Top {
		blows = blows[:x.Top]
	}
	sum := 0.0
	for _, b := range blows {
		sum += b
	}
	return int(sum)
}

func (x *evaluator) wary(u *core.UnitView) bool {
	return lord(x.e, *u) || strings.HasPrefix(x.Threat, "officers") && !x.e.Data().Characters[u.Def].Template
}

// risk is the strength u stands to lose at p.
func (x *evaluator) risk(u *core.UnitView, p content.Point, hp int) float64 {
	if !x.wary(u) {
		return 0
	}
	left := hp - x.incoming(u, p)
	if strings.HasSuffix(x.Threat, "-lethal") {
		if left > 0 {
			return 0
		}
		return x.W * x.worth(u, hp)
	}
	return x.W * (x.worth(u, hp) - x.worth(u, left))
}

// standing is the HP an ally can count on at its own cell, after what may strike it there.
func (x *evaluator) standing(u *core.UnitView) int {
	if !x.wary(u) {
		return u.HP
	}
	return u.HP - x.incoming(u, content.Point{X: u.X, Y: u.Y})
}

func (x *evaluator) unit(id string) *core.UnitView {
	for i := range x.o.UnitViews {
		if x.o.UnitViews[i].ID == id {
			return &x.o.UnitViews[i]
		}
	}
	return nil
}

// reach is the nearest and farthest a unit strikes from, with its basic attack or a damaging skill.
func (x *evaluator) reach(u *core.UnitView) (lo, hi int) {
	lo, hi = x.e.AttackRange(u.ID)
	for _, name := range u.Skills {
		s := x.e.Data().Skills[name]
		if (s.Kind == content.KindPhysical || s.Kind == content.KindMagic) && s.Max > 0 {
			lo, hi = min(lo, max(1, s.Min)), max(hi, s.Max)
		}
	}
	return lo, hi
}

func (x *evaluator) action(c core.Command, duels bool) (float64, bool) {
	me := x.me
	here := content.Point{X: me.X, Y: me.Y}
	p, err := x.e.Preview(c)
	if err != nil {
		return 0, false
	}
	skill := x.e.Data().Skills[c.Skill]
	gain := 0.0
	switch {
	case len(p.Damage) > 0:
		if duels && me.DuelFirst && x.e.DuelPending(me.ID, c.Target) {
			return 0, false
		}
		for id, d := range p.Damage {
			q := x.unit(id)
			if q == nil {
				continue
			}
			lost := (x.worth(q, q.HP) - x.worth(q, q.HP-d)) * p.Hit / 100
			if q.Faction == me.Faction {
				lost = -lost
			}
			gain += lost
		}
		if back := int(float64(x.e.CounterDamage(c)) / x.e.Data().Rules.CritMultiplier); back > 0 {
			gain -= x.worth(me, me.HP) - x.worth(me, me.HP-back)
		}
	case p.Healing > 0 || p.MPRecovery > 0:
		var mine []*core.UnitView
		for _, id := range p.Targets {
			if q := x.unit(id); q != nil && q.Faction == me.Faction {
				mine = append(mine, q)
			}
		}
		used, urgent := 0, false
		for _, q := range mine {
			h := x.standing(q)
			heal := min(p.Healing/max(1, len(mine)), q.Stats.MaxHP-q.HP)
			gain += x.worth(q, h+heal) - x.worth(q, h)
			// liquor fills past the maximum, but what lies above it only waits for the next spell
			mp := min(p.MPRecovery, q.Stats.MaxMP-q.MP)
			if !x.nearFight(q) {
				mp = 0 // far from any opponent the points come back by themselves before they are needed
			}
			if mp > 0 {
				gain += .02 * float64(max(q.Stats.Attack, q.Stats.Mind)) * float64(mp) / float64(q.Stats.MaxMP)
			}
			used += max(0, heal) + max(0, mp)
			urgent = urgent || h*10 < q.Stats.MaxHP*3
		}
		if gain <= 0 {
			return 0, false // a draught or a meal that changes nothing is not worth an action and a piece
		}
		if it, ok := x.e.Data().Items[c.Item]; c.Kind == "item" && ok && used*2 < it.Amount && !urgent {
			return 0, false // a piece is spent when at least half of it counts, or to save someone in danger
		}
	case skill.Effect == "cure":
		for _, id := range p.Targets {
			if q := x.unit(id); q != nil && q.Faction == me.Faction {
				gain += x.relief(q)
			}
		}
		if gain <= 0 {
			return 0, false
		}
		gain *= p.Hit / 100
	case skill.Kind == content.KindBuff:
		k := float64(x.e.Data().Rules.BuffPercent) / 100
		for _, id := range p.Targets {
			q := x.unit(id)
			if q == nil || q.Faction != me.Faction || q.Status[skill.Effect].Turns > 0 || q.Status["confusion"].Turns > 0 || !x.engaged(q) {
				continue
			}
			gain += x.worth(q, q.HP) * k
		}
		gain *= p.Hit / 100
	default:
		return 0, false
	}
	gain -= .1 * float64(p.Cost)
	if x.XP > 0 {
		gain += x.XP * x.Grow[me.Character] * float64(p.XP) / 10000 * p.Hit / 100 * x.worth(me, me.HP)
	}
	return gain - x.risk(me, here, me.HP), true
}

// move scores a cell by the best blow it opens, a little below the blow itself, halved for every
// further turn of walking it needs; a cell that starts a stage duel the officer comes out of standing
// is worth half of him (the duel teaches a whole level), and a victory goal's area pulls its unit the
// harder the fewer rounds are left for the walk. The turns of walking count the way around walls and rivers,
// as far as the walk to the nearest opponent tells it. Less what the mover risks standing there.
func (x *evaluator) move(c core.Command, walk map[content.Point]int, duels bool) float64 {
	me := x.me
	if duels && me.DuelFirst {
		for _, v := range x.o.UnitViews {
			if x.e.DuelPending(me.ID, v.ID) && distance(c.X, c.Y, v.X, v.Y) == 1 {
				return 1e6
			}
		}
	}
	lo, hi := x.reach(me)
	step := max(1, x.e.Data().Classes[me.Class].Move)
	best := 0.0
	for _, v := range x.opponents(me) {
		d := distance(c.X, c.Y, v.X, v.Y)
		if d < lo || duels && me.DuelFirst && x.e.DuelPending(me.ID, v.ID) {
			continue
		}
		hit := int(x.strike(me.ID, v.ID))
		turns := 0
		if d > hi {
			// out of reach from c: the walk there is at least the walk to the nearest opponent, so a cell
			// near by line but behind a wall or a river does not pull the unit back and forth along it
			turns = (max(d, far(walk, c.X, c.Y)) - hi + step - 1) / step
		}
		best = max(best, .9*(x.worth(&v, v.HP)-x.worth(&v, v.HP-hit))*math.Pow(.5, float64(turns)))
	}
	best += .01 * float64(far(walk, me.X, me.Y)-far(walk, c.X, c.Y)) // walls: a step that walks nearer breaks ties
	if id := x.e.DuelAt(me.ID, c.X, c.Y); duels && id != "" {
		for _, d := range x.o.Map.Duels {
			if d.ID == id && d.AllyResult == "stay" {
				best += .5 * x.worth(me, me.HP)
			}
		}
	}
	if x.goal != nil && !x.pending {
		left := 8.0 // without a turn limit, as if eight rounds were left
		if limit := x.o.Map.TurnLimit; limit > 0 {
			left = float64(max(1, limit-x.o.Round+1))
		}
		need := float64(far(x.goal, me.X, me.Y)) / float64(step)
		pull := .3 * min(1, max(.2, 2*need/left))
		w := x.worth(me, me.HP)
		best += pull * w * float64(far(x.goal, me.X, me.Y)-far(x.goal, c.X, c.Y))
		if far(x.goal, c.X, c.Y) == 0 {
			best += 10 * w
		}
		if x.onWay && x.stop == (content.Point{X: c.X, Y: c.Y}) {
			best += pull * w // as far along the whole planned walk as this turn goes
		}
	}
	if x.escort != nil {
		best += escortPull * x.worth(me, me.HP) * float64(far(x.escort, me.X, me.Y)-far(x.escort, c.X, c.Y))
	}
	return best - x.risk(me, content.Point{X: c.X, Y: c.Y}, me.HP)
}

// relief is what ending q's harmful statuses is worth, weighed as hinder weighs casting them and on the
// scale a buff is scored on here (a buff of BuffPercent over its two turns is worth that share of q): over
// the turns each status has left (two at most), all of q for confusion, q's own kind of fighting when
// disarmed or sealed, a share of it when rooted or its stats are lowered, or the HP poison and fire take.
// Away from any fight only the bleeding counts.
func (x *evaluator) relief(q *core.UnitView) float64 {
	rules := x.e.Data().Rules
	fighting := x.nearFight(q)
	physical := q.Stats.Attack >= q.Stats.Mind
	whole := x.worth(q, q.HP)
	sum := 0.0
	for k, st := range q.Status {
		if !content.Harmful[k] || st.Turns <= 0 {
			continue
		}
		turns := float64(min(st.Turns, 2))
		share := float64(rules.BuffPercent) / 100
		switch k {
		case "poison", "burn":
			pct := map[string]int{"poison": rules.PoisonPercent, "burn": rules.BurnPercent}[k]
			sum += (whole - x.worth(q, q.HP-q.Stats.MaxHP*pct/100)) * turns
			continue
		case "confusion":
			share = 1
		case "disarm":
			share = map[bool]float64{true: 1, false: .1}[physical]
		case "seal":
			share = map[bool]float64{true: .1, false: 1}[physical]
		case "root":
			share = .3
		}
		if fighting {
			sum += share * whole * turns / 2
		}
	}
	return sum
}

// engaged reports whether u meets an opponent by the next turn: one can strike u where it stands, or u can
// strike one. A buff on a unit with nobody in reach either way (across a river, behind a wall) is wasted.
func (x *evaluator) engaged(u *core.UnitView) bool {
	mine := map[content.Point]bool{}
	for _, p := range x.e.Threat(u.ID) {
		mine[p] = true
	}
	for _, v := range x.opponents(u) {
		if x.reachOf(v.ID)[content.Point{X: u.X, Y: u.Y}] || mine[content.Point{X: v.X, Y: v.Y}] {
			return true
		}
	}
	return false
}

// nearFight reports whether an opponent stands within two turns' walk and reach of u.
func (x *evaluator) nearFight(u *core.UnitView) bool {
	_, hi := x.reach(u)
	far := 2*max(1, x.e.Data().Classes[u.Class].Move) + hi
	for _, v := range x.opponents(u) {
		if distance(u.X, u.Y, v.X, v.Y) <= far {
			return true
		}
	}
	return false
}

// escortPull is how much a step nearer the way ahead of a unit walking to a goal is worth, in shares of
// the walker's own worth: as strong as a goal pulls its own unit when time runs short.
const escortPull = .2

func chooseEval(e *core.Engine, o core.Observation, me *core.UnitView, options []core.Command, duels bool) core.Command {
	if r := rankEval(e, o, me, options, duels, Player); len(r) > 0 {
		return r[0].Command
	}
	return options[0]
}

// Scored is a command with the score the evaluator gave it.
type Scored struct {
	Command core.Command
	Score   float64
}

// Rank scores the legal commands of the actor's unit by the weights x, best first (ties keep the engine's
// order). Commands the evaluator rules out are left out; nil when the unit cannot act.
func Rank(e *core.Engine, actor string, x *Eval, duels bool) []Scored {
	o := e.Observe()
	for i := range o.UnitViews {
		if o.UnitViews[i].ID == actor {
			return rankEval(e, o, &o.UnitViews[i], e.LegalActions(actor).Commands, duels, x)
		}
	}
	return nil
}

func rankEval(e *core.Engine, o core.Observation, me *core.UnitView, options []core.Command, duels bool, weights *Eval) []Scored {
	x := &evaluator{Eval: weights, e: e, o: o, me: me, victory: o.Map.VictoryTargets(), vital: content.DefeatTargets(o.Map.Defeat)}
	marks := e.GoalMarks()
	var cells []content.Point
	for _, n := range marks {
		if n.Win {
			cells = append(cells, n.Cells...)
		}
	}
	for _, m := range marks {
		if !m.Win || m.Unit == "" || len(cells) == 0 {
			continue
		}
		if m.Unit == me.ID {
			x.goal, _ = goalRoute(e, o, me, cells)
			x.pending, x.escort, x.clear = m.Pending, nil, nil
			if !x.pending {
				_, way := plan(e, o, me, cells)
				x.stop, x.onWay = farthest(way, options)
			}
			break
		}
		// another unit walks to the goal: keep up with it and clear its way
		for i := range o.UnitViews {
			if v := &o.UnitViews[i]; v.ID == m.Unit && v.Faction == me.Faction && v.HP > 0 && !m.Pending {
				_, way := goalRoute(e, o, v, cells)
				var ahead []content.Point
				ahead, x.clear = escort(e, o, v, way)
				if len(ahead) > 0 {
					x.escort = walkingFrom(e, o, me, ahead)
				}
			}
		}
	}
	var out []Scored
	var walk map[content.Point]int
	for _, c := range options {
		s, ok := 0.0, true
		switch c.Kind {
		case "attack", "skill", "item":
			if x.onWay {
				ok = false // a unit walking to its goal walks first and acts where it stops
				break
			}
			s, ok = x.action(c, duels)
		case "move":
			if !duels && e.DuelAt(me.ID, c.X, c.Y) != "" {
				ok = false
				break
			}
			if walk == nil {
				walk = walking(e, o, me)
			}
			s = x.move(c, walk, duels)
		case "wait":
			s = -.5 - x.risk(me, content.Point{X: me.X, Y: me.Y}, me.HP)
		default:
			ok = false
		}
		if ok {
			out = append(out, Scored{c, s})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}
