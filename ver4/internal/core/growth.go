package core

import "math"

// ExperienceRequired is the raw cost from level L to L+1. The UI always
// displays progress on a 0..100 scale; surplus raw experience is retained.
func ExperienceRequired(level int) int {
	return int(math.Round(100*math.Pow(1.035, float64(max(1, level)-1)) + 1e-9))
}
func ExperienceProgress(level, xp int) float64 {
	return min(100, 100*float64(xp)/float64(ExperienceRequired(level)))
}

// Action XP is awarded per effective target. Neither damage magnitude nor
// repeated attacks reduce it: using weak attacks and healing enemies is valid.
// Counters earn no XP; poison/burn earn only the finishing bonus, not tick XP.
func (e *Engine) actionXP(u, v *Unit, base int, spell, defeated bool) {
	if n := e.actionXPAmount(u, v, base, spell, defeated); n > 0 {
		e.xp(u.Officer, n)
	}
}
func (e *Engine) actionXPAmount(u, v *Unit, base int, spell, defeated bool) int {
	if u.Faction != "ally" || v.Faction != "enemy" {
		return 0
	}
	a, b := e.officer(u.Officer), e.officer(v.Officer)
	factor := clamp(1+.12*float64(b.Level-a.Level), .25, 2)
	n := float64(base) * float64(ExperienceRequired(b.Level)) / 100 * factor
	if spell {
		n *= 2
	}
	if defeated {
		n *= 2
	}
	return max(1, int(math.Round(n)))
}

// supportXP is what an ally earns per target of a landed support spell.
func (e *Engine) supportXP(u *Unit) int {
	return max(1, ExperienceRequired(e.officer(u.Officer).Level)*16/100)
}
func (e *Engine) xp(id string, n int) {
	o := e.officer(id)
	if o == nil || o.Dead || n <= 0 {
		return
	}
	o.XP += n
	for o.Level < 99 && o.XP >= ExperienceRequired(o.Level) {
		o.XP -= ExperienceRequired(o.Level)
		o.Level++
		o.Points += 100
		e.emit("level", id, "", o.Level)
	}
	if o.Level == 99 {
		o.XP = min(o.XP, ExperienceRequired(99)-1)
	}
	e.emit("experience", id, "", n)
	if d := e.officer(o.Deputy); d != nil && !d.Dead {
		shared := n*85 + d.XPShareRemainder
		d.XPShareRemainder = shared % 100
		e.xp(d.ID, shared/100)
	}
}
