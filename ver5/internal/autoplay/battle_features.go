package autoplay

import (
	"math"
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

const BattleStateLen = 64
const BattleActionLen = 64
const BattleFeatureVersion = 1

// BattleChoices contains every legal command of every ready player unit, without heuristic ranking.
type BattleChoices struct {
	State    []float64
	Commands []core.Command
	Features [][]float64
}

func battleState(d *content.Data, e *core.Engine, grow map[string]float64) []float64 {
	o := e.Observe()
	x := make([]float64, BattleStateLen)
	p := Measure(d, e, grow)
	limit := max(1, o.Map.TurnLimit)
	if o.Map.TurnLimit == 0 {
		limit = 50
	}
	x[0], x[1] = float64(o.Round)/float64(limit), float64(max(0, limit-o.Round))/float64(limit)
	x[2], x[3], x[4] = p.Mine.Ratio(), p.Enemy.Ratio(), p.Allies.Ratio()
	x[5], x[6] = float64(p.Units)/20, float64(p.Others)/50
	x[7] = math.Log((p.Mine.Left+.5*p.Allies.Left+1)/(p.Enemy.Left+1)) / 4
	x[8], x[9], x[10] = p.Vital, p.XP/20, float64(p.Goal)/float64(max(1, o.Map.Width+o.Map.Height))
	x[11], x[12] = float64(p.Subs)/float64(max(1, p.SubsTotal)), b2f(o.Map.TurnLimit > 0)
	var hp, mp, leftMP float64
	for _, u := range o.UnitViews {
		if u.HP <= 0 {
			continue
		}
		if mine(u) {
			hp += float64(u.Stats.MaxHP)
			mp += float64(u.Stats.MaxMP)
			leftMP += float64(u.MP)
		}
		channel := 2
		if mine(u) {
			channel = 0
		} else if u.Faction == "enemy" {
			channel = 1
		}
		col, row := min(3, u.X*4/max(1, o.Map.Width)), min(3, u.Y*4/max(1, o.Map.Height))
		x[16+(row*4+col)*3+channel] += float64(u.HP) / 10000
	}
	x[13] = leftMP / max(1, mp)
	for id, n := range o.Inventory {
		it := d.Items[id]
		if it.Effect == content.ItemHP {
			x[14] += float64(n*it.Amount) / max(1, hp)
		} else if it.Effect == content.ItemMP {
			x[15] += float64(n*it.Amount) / max(1, mp)
		}
	}
	return x
}

// BattleInputs reads positions, effects and threats; it never modifies e or consults the old action scores.
func BattleInputs(d *content.Data, e *core.Engine) BattleChoices {
	return battleInputs(d, e, false)
}

func battleInputs(d *content.Data, e *core.Engine, rich bool) BattleChoices {
	o := e.Observe()
	out := BattleChoices{}
	if o.Phase != "battle" || o.Turn != "ally" || len(o.Learning) > 0 {
		return out
	}
	grow := Grow(d, e)
	out.State = battleState(d, e, grow)
	units := map[string]core.UnitView{}
	reaches := map[string]map[content.Point]bool{}
	victory := o.Map.VictoryTargets()
	for _, u := range o.UnitViews {
		units[u.ID] = u
		if u.Faction == "enemy" && u.HP > 0 {
			r := map[content.Point]bool{}
			for _, p := range e.Threat(u.ID) {
				r[p] = true
			}
			reaches[u.ID] = r
		}
	}
	w, h := float64(max(1, o.Map.Width)), float64(max(1, o.Map.Height))
	diag := w + h
	ready := 0
	for _, u := range o.UnitViews {
		if !mine(u) || u.HP <= 0 || u.Done {
			continue
		}
		incoming := map[string]float64{}
		for id := range reaches {
			incoming[id] = e.Strike(id, u.ID)
		}
		move := float64(max(1, d.Classes[u.Class].Move))
		_, attackRange := e.AttackRange(u.ID)
		for _, c := range e.LegalActions(u.ID).Commands {
			a := make([]float64, BattleActionLen)
			kind := slices.Index([]string{"move", "attack", "skill", "item", "wait"}, c.Kind)
			if kind < 0 {
				continue
			}
			a[kind] = 1
			x, y := u.X, u.Y
			if c.Kind == "move" {
				x, y = c.X, c.Y
			}
			a[6], a[7] = float64(x)/w, float64(y)/h
			a[8], a[9] = float64(x-u.X)/move, float64(y-u.Y)/move
			a[10], a[11], a[12] = float64(u.HP)/float64(max(1, u.Stats.MaxHP)), float64(u.MP)/float64(max(1, u.Stats.MaxMP)), float64(u.Level)/50
			a[13], a[14], a[15] = float64(u.Stats.Attack)/1000, float64(u.Stats.Defense)/1000, float64(u.Stats.Mind)/1000
			a[16], a[17] = float64(u.X)/w, float64(u.Y)/h
			a[18], a[19] = b2f(u.Moved), b2f(u.Acted)
			a[20], a[21], a[22], a[23] = b2f(d.Characters[u.Def].Lord), grow[u.Character], float64(attackRange)/10, move/10
			near, goal, wasGoal := diag, diag, diag
			for _, v := range o.UnitViews {
				if v.HP <= 0 {
					continue
				}
				if v.Faction == "enemy" {
					near = min(near, float64(abs(x-v.X)+abs(y-v.Y)))
					if slices.Contains(victory, v.ID) {
						goal = min(goal, float64(abs(x-v.X)+abs(y-v.Y)))
						wasGoal = min(wasGoal, float64(abs(u.X-v.X)+abs(u.Y-v.Y)))
					}
					if reaches[v.ID][content.Point{X: x, Y: y}] {
						a[26] += incoming[v.ID] / float64(max(1, u.Stats.MaxHP))
						a[27] += .1
					}
				} else if abs(x-v.X)+abs(y-v.Y) <= 2 {
					a[62] += .1
				}
			}
			a[24], a[25], a[60] = near/diag, goal/diag, (wasGoal-goal)/move
			q := units[c.Target]
			if rich {
				count := func(v core.UnitView) (int, int) {
					n, least := 0, 10000
					for _, id := range v.Skills {
						if cost := d.Skills[id].Cost; cost > 0 {
							n++
							least = min(least, cost)
						}
					}
					if n == 0 {
						least = 0
					}
					return n, least
				}
				n, _ := count(u)
				a[5] = float64(n) / 10
				n, cost := count(q)
				a[28], a[29] = float64(n)/10, float64(cost)/100
				a[50] = b2f(n > 0)
				if c.Kind == "item" && d.Items[c.Item].Effect == content.ItemMP {
					a[57] = b2f(n == 0)
				}
			}
			if c.Target != "" {
				a[30], a[31] = float64(q.HP)/float64(max(1, q.Stats.MaxHP)), float64(q.MP)/float64(max(1, q.Stats.MaxMP))
				a[32], a[33] = float64(q.X)/w, float64(q.Y)/h
				a[34], a[35], a[36] = b2f(q.Faction == u.Faction), b2f(slices.Contains(victory, q.ID)), b2f(d.Characters[q.Def].Lord)
				a[37], a[38], a[39], a[40] = float64(q.Level)/50, float64(max(q.Stats.Attack, q.Stats.Mind))/1000, float64(q.Stats.Defense)/1000, float64(q.Stats.Mind)/1000
				a[63] = float64(q.Level-u.Level) / 20
			}
			if c.Kind != "move" && c.Kind != "wait" {
				if p, err := e.Preview(c); err == nil {
					a[41], a[42] = p.Hit/100, float64(p.XP)/core.XPPerLevel*grow[u.Character]
					for id, damage := range p.Damage {
						v := units[id]
						if v.Faction == u.Faction {
							a[44] += float64(damage) / float64(max(1, v.Stats.MaxHP))
						} else {
							a[43] += float64(damage) * p.Hit / 100 / float64(max(1, v.HP))
						}
					}
					a[45], a[46], a[47] = float64(p.Healing)/float64(max(1, q.Stats.MaxHP)), float64(p.MPRecovery)/float64(max(1, q.Stats.MaxMP)), float64(p.Cost)/float64(max(1, u.Stats.MaxMP))
					a[61] = float64(len(p.Targets)) / 10
				}
			}
			if id := e.DuelAt(u.ID, x, y); id != "" {
				a[48] = 1
				for _, duel := range o.Map.Duels {
					if duel.ID == id {
						a[49] = b2f(duel.Outcome == "victory")
					}
				}
			}
			skill := d.Skills[c.Skill]
			a[51], a[52], a[53], a[54], a[55] = b2f(skill.Kind == content.KindBuff), b2f(skill.Kind == content.KindHeal), b2f(skill.Kind == content.KindPhysical), b2f(skill.Kind == content.KindMagic), b2f(skill.Effect == "cure")
			a[56] = b2f(q.Status[skill.Effect].Turns > 0)
			a[58], a[59] = float64(e.StepCost(u.ID, x, y))/move, float64(ready)/20
			row := append(slices.Clone(out.State), a...)
			out.Commands, out.Features = append(out.Commands, c), append(out.Features, row)
		}
		ready++
	}
	return out
}
