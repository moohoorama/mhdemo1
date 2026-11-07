package core

import "math"

// XPPerLevel is the experience between two levels at every level: Character.XP is the
// progress through the current level's bar in 1/100 percent, so a surplus keeps its meaning
// across level-ups and an XPPerLevel grant is always exactly one bar.
const XPPerLevel = 10000

func ExperienceProgress(xp int) float64 {
	return min(100, 100*float64(xp)/XPPerLevel)
}

// Action XP is awarded per effective target. Neither damage magnitude nor
// repeated attacks reduce it: using weak attacks and healing enemies is valid.
// Counterattacks grant defense XP to the recipient; poison/burn grant only a finishing bonus.
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
	n := R.ExpAction * XPPerLevel / 100 * factor * math.Pow(R.ExpGrowth, float64(b.Level-a.Level))
	n *= e.xpScale()
	if spell {
		n *= R.ExpSpell
	}
	if defeated {
		n *= R.ExpDefeat
	}
	return max(1, int(math.Round(n)))
}

// defenseXP is awarded once per incoming strike, including a successfully avoided one.
func (e *Engine) defenseXP(v, u *Unit, spell bool) {
	if n := e.actionXPAmount(v, u, spell, false); n > 0 && e.data.Rules.ExpDefenseShare > 0 {
		e.xp(v.Character, max(1, int(math.Round(float64(n)*e.data.Rules.ExpDefenseShare))))
	}
}

// supportXP is what an ally earns per target of a landed support spell (a heal, a buff): the support
// share of a level, corrected by the target's level against the caster's as an action's experience is,
// so helping a stronger officer teaches more.
func (e *Engine) supportXP(u, q *Unit) int {
	R := e.data.Rules
	diff := float64(e.char(q.Character).Level - e.char(u.Character).Level)
	factor := clamp(1+R.ExpLevelStep*diff, R.ExpMinFactor, R.ExpMaxFactor)
	n := float64(XPPerLevel*R.ExpSupportPercent/100) * factor * math.Pow(R.ExpGrowth, diff)
	n *= e.xpScale()
	return max(1, int(math.Round(n)))
}
func (e *Engine) xp(id string, n int) {
	R := e.data.Rules
	o := e.char(id)
	if o == nil || o.Dead || n <= 0 || (e.unit(id) != nil && e.unit(id).AI) {
		return
	}
	o.XP += n
	leveled := false
	for o.Level < R.MaxLevel && o.XP >= XPPerLevel {
		o.XP -= XPPerLevel
		o.Level++
		o.Points += R.PointsPerLevel
		leveled = true
		e.emit("level", id, "", o.Level)
	}
	if o.Level == R.MaxLevel {
		o.XP = min(o.XP, XPPerLevel-1)
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

// CanLearn reports whether the officer may open the learning window now: the walk or a preparation is on, and
// the officer is a living player officer with points for a trait.
func (e *Engine) CanLearn(id string) bool {
	o := e.char(id)
	kind := e.data.Node(e.state.Node).Kind
	inPrep := (e.state.Phase == "scenario" || e.Deploying()) && (kind == "battle" || kind == "explore")
	return inPrep && o != nil && !o.Dead && e.def(o).Playable && e.canLearn(o)
}

// Learnable lists the traits the officer may learn now: not held, prerequisites met (cost aside).
func (e *Engine) Learnable(id string) []string {
	if o := e.char(id); o != nil {
		return e.learnable(o)
	}
	return nil
}

// openLearning opens the learning window of an officer outside battle, so the points can be spent whenever the
// party is in camp or on the walk, not only after a level-up.
func (e *Engine) openLearning(c Command) error {
	if e.state.Phase != "scenario" && !e.Deploying() {
		return fail("WrongPhase")
	}
	if !e.CanLearn(c.Actor) {
		return fail("NothingToLearn")
	}
	e.state.Learning = append(e.state.Learning, c.Actor)
	return nil
}

// learn spends a character's points on a trait while its learning window is open or whenever CanLearn allows it
// (the trait list of the party window), or closes the window. The window closes by itself when nothing affordable
// is left.
func (e *Engine) learn(c Command) error {
	i := -1
	for k, id := range e.state.Learning {
		if id == c.Actor {
			i = k
		}
	}
	if i < 0 && (c.Kind != "learn" || !e.CanLearn(c.Actor)) {
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
	if i >= 0 && (c.Kind == "learn_close" || !e.canLearn(o)) {
		e.state.Learning = append(e.state.Learning[:i], e.state.Learning[i+1:]...)
	}
	return nil
}

// joinAt brings an officer who joins to the level the story has reached, in the class that level reaches on
// the officer's own promotion path.
func (e *Engine) joinAt(o *Character, level int) {
	e.raiseTo(o, level)
	o.Class = e.promotedAt(o.Def, o.Class, o.Level)
}

// raiseTo lifts an officer who joins below the level the story has reached, with the trait points the
// missing levels would have brought.
func (e *Engine) raiseTo(o *Character, level int) {
	if n := level - o.Level; n > 0 {
		o.Level = level
		o.Points += n * e.data.Rules.PointsPerLevel
	}
}
