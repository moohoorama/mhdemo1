package core

import (
	"math"
	"srpg/internal/content"
)

// ExperienceRequired is the raw cost from level L to L+1. The UI always
// displays progress on a 0..100 scale; surplus raw experience is retained.
func ExperienceRequired(d *content.Data, level int) int {
	return int(math.Round(d.Rules.ExpBase*math.Pow(d.Rules.ExpGrowth, float64(max(1, level)-1)) + 1e-9))
}
func ExperienceProgress(d *content.Data, level, xp int) float64 {
	return min(100, 100*float64(xp)/float64(ExperienceRequired(d, level)))
}

// Action XP is awarded per effective target. Neither damage magnitude nor
// repeated attacks reduce it: using weak attacks and healing enemies is valid.
// Counters earn no XP; poison/burn earn only the finishing bonus, not tick XP.
func (e *Engine) actionXP(u, v *Unit, spell, defeated bool) {
	if n := e.actionXPAmount(u, v, spell, defeated); n > 0 {
		e.xp(u.Character, n)
	}
}
func (e *Engine) actionXPAmount(u, v *Unit, spell, defeated bool) int {
	R := e.data.Rules
	if u.Faction != "ally" || v.Faction != "enemy" {
		return 0
	}
	a, b := e.char(u.Character), e.char(v.Character)
	factor := clamp(1+R.ExpLevelStep*float64(b.Level-a.Level), R.ExpMinFactor, R.ExpMaxFactor)
	n := R.ExpAction * float64(ExperienceRequired(e.data, b.Level)) / 100 * factor
	if spell {
		n *= R.ExpSpell
	}
	if defeated {
		n *= R.ExpDefeat
	}
	return max(1, int(math.Round(n)))
}

// supportXP is what an ally earns per target of a landed support spell.
func (e *Engine) supportXP(u *Unit) int {
	return max(1, ExperienceRequired(e.data, e.char(u.Character).Level)*e.data.Rules.ExpSupportPercent/100)
}
func (e *Engine) xp(id string, n int) {
	R := e.data.Rules
	o := e.char(id)
	if o == nil || o.Dead || n <= 0 {
		return
	}
	o.XP += n
	leveled := false
	for o.Level < R.MaxLevel && o.XP >= ExperienceRequired(e.data, o.Level) {
		o.XP -= ExperienceRequired(e.data, o.Level)
		o.Level++
		o.Points += R.PointsPerLevel
		leveled = true
		e.emit("level", id, "", o.Level)
	}
	if o.Level == R.MaxLevel {
		o.XP = min(o.XP, ExperienceRequired(e.data, R.MaxLevel)-1)
	}
	e.emit("experience", id, "", n)
	if leveled && e.def(o).Playable && e.canLearn(o) && !contains(e.state.Learning, id) {
		e.state.Learning = append(e.state.Learning, id)
	}
	if d := e.char(o.Deputy); d != nil && !d.Dead {
		shared := n*R.DeputyShare + d.XPShareRemainder
		d.XPShareRemainder = shared % 100
		e.xp(d.ID, shared/100)
	}
}

// canLearn reports whether the character can afford any trait it may learn.
func (e *Engine) canLearn(c *Character) bool {
	for _, t := range e.learnable(c) {
		if c.Points >= e.data.Traits[t].Cost {
			return true
		}
	}
	return false
}

// learn spends a character's points on a trait while its learning window is open, or closes the window.
// The window closes by itself when nothing affordable is left.
func (e *Engine) learn(c Command) error {
	i := -1
	for k, id := range e.state.Learning {
		if id == c.Actor {
			i = k
		}
	}
	if i < 0 {
		return fail("WrongPhase")
	}
	o := e.char(c.Actor)
	if c.Kind == "learn" {
		t, ok := e.data.Traits[c.Trait]
		if !ok || !contains(e.learnable(o), c.Trait) {
			return fail("InvalidTrait")
		}
		if o.Points < t.Cost {
			return fail("InsufficientPoints")
		}
		o.Points -= t.Cost
		o.Learned = append(o.Learned, c.Trait)
		e.emit("learn", o.ID, c.Trait, 0)
	}
	if c.Kind == "learn_close" || !e.canLearn(o) {
		e.state.Learning = append(e.state.Learning[:i], e.state.Learning[i+1:]...)
	}
	return nil
}
