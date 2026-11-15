package autoplay

import (
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

// Value weights: a battle's result outweighs everything a battle can still change; the rest is strength kept
// and taken, the units whose fall loses, the bonus objectives, the way to an area goal and the experience the
// officers who count gathered in it.
const (
	vWin      = 10
	vMaterial = 3
	vVital    = 2
	vSub      = 1
	vXP       = .5
	vGoal     = .02
)

// Side sums the strength (attack or mind) of one side's units, all and at their HP left.
type Side struct{ Full, Left float64 }

// Ratio is the share of the side's strength still standing.
func (s Side) Ratio() float64 {
	if s.Full == 0 {
		return 0
	}
	return s.Left / s.Full
}

// Parts are what Value sums, kept apart for the learned models' features.
type Parts struct {
	Win, Loss            bool
	Mine, Allies, Enemy  Side
	Vital                float64 // lowest HP share of the units whose fall loses
	Subs, SubsTotal      int
	XP                   float64 // levels gained in the battle, weighed by Grow
	Goal                 int     // walk left to an area goal, in cells; 0 without one
	Round, TurnLimit     int
	LevelGap             float64 // the party's mean level less the enemy's
	Lords, Units, Others int     // living own units whose fall loses, own units, opponents
}

// Measure reads the parts of the battle state's value.
func Measure(d *content.Data, e *core.Engine, grow map[string]float64) Parts {
	o := e.Observe()
	p := Parts{Win: o.Result == "victory", Loss: o.Result == "defeat", Vital: 1, Round: o.Round}
	var vital []string
	if o.Map != nil {
		vital = content.DefeatTargets(o.Map.Defeat)
		p.TurnLimit = o.Map.TurnLimit
	}
	var lm, lt, nm, nt float64
	for _, u := range o.UnitViews {
		power := float64(max(u.Stats.Attack, u.Stats.Mind))
		left := 0.0
		if u.HP > 0 {
			left = power * float64(u.HP) / float64(max(1, u.Stats.MaxHP))
		}
		switch {
		case mine(u):
			p.Mine.Full += power
			p.Mine.Left += left
			if u.HP > 0 {
				p.Units++
				lm += float64(u.Level)
				nm++
			}
			if d.Characters[u.Def].Lord || slices.Contains(vital, u.Character) || slices.Contains(vital, u.ID) {
				p.Vital = min(p.Vital, float64(max(0, u.HP))/float64(max(1, u.Stats.MaxHP)))
				if u.HP > 0 {
					p.Lords++
				}
			}
		case u.Faction == "ally":
			p.Allies.Full += power
			p.Allies.Left += left
		default:
			p.Enemy.Full += power
			p.Enemy.Left += left
			if u.HP > 0 {
				p.Others++
				lt += float64(u.Level)
				nt++
			}
		}
	}
	if nm > 0 && nt > 0 {
		p.LevelGap = lm/nm - lt/nt
	}
	for _, ok := range e.SubDone() {
		p.SubsTotal++
		if ok {
			p.Subs++
		}
	}
	if o.Merit != nil {
		for id, xp := range o.Merit.XP {
			p.XP += grow[id] * float64(xp) / core.XPPerLevel
		}
	}
	for _, m := range e.GoalMarks() {
		if !m.Win || m.Unit == "" || len(m.Cells) == 0 || m.Pending {
			continue
		}
		for _, u := range o.UnitViews {
			if u.ID == m.Unit && u.HP > 0 {
				best := 1 << 30
				for _, c := range m.Cells {
					best = min(best, abs(c.X-u.X)+abs(c.Y-u.Y))
				}
				p.Goal += best
			}
		}
	}
	return p
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Score is the hand-made value of the parts.
func (p Parts) Score() float64 {
	v := vMaterial*(p.Mine.Ratio()+.5*p.Allies.Ratio()-p.Enemy.Ratio()) + vVital*p.Vital + vSub*float64(p.Subs) + vXP*p.XP - vGoal*float64(p.Goal)
	switch {
	case p.Win:
		v += vWin
	case p.Loss:
		v -= vWin
	}
	return v
}

// Value is the hand-made value of the battle state for the player.
func Value(d *content.Data, e *core.Engine, grow map[string]float64) float64 {
	return Measure(d, e, grow).Score()
}
