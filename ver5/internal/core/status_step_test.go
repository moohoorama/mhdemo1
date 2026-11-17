package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A buff and a debuff of one stat cancel out, so a stat steps from down to even to up; the same side keeps the
// stronger value and the longer duration.
func TestStatusSteps(t *testing.T) {
	e, hero, _ := arena(t)
	t.Run("a debuff ends a buff, and a buff then comes back on", func(t *testing.T) {
		hero.Status = map[string]Status{"attack": {Turns: 3, Value: 20}}
		assert.True(t, e.applyStatus(hero, "attack", 3, -20))
		assert.Zero(t, hero.Status["attack"].Turns, "even")
		assert.True(t, e.applyStatus(hero, "attack", 3, -20))
		assert.Equal(t, Status{Turns: 3, Value: -20}, hero.Status["attack"], "down")
		assert.True(t, e.applyStatus(hero, "attack", 2, 20))
		assert.Zero(t, hero.Status["attack"].Turns, "a buff cancels the debuff")
	})
	t.Run("the same buff again changes nothing unless it lasts longer", func(t *testing.T) {
		hero.Status = map[string]Status{"agility": {Turns: 2, Value: 20}}
		assert.False(t, e.applyStatus(hero, "agility", 2, 20))
		assert.False(t, e.applyStatus(hero, "agility", 1, 20), "a shorter one leaves the longer")
		assert.Equal(t, 2, hero.Status["agility"].Turns)
		assert.True(t, e.applyStatus(hero, "agility", 5, 20), "장구's longer buff")
		assert.Equal(t, 5, hero.Status["agility"].Turns)
	})
}

// A buff teaches only when it changes something, and the experience of one action comes as one sum at its end.
func TestBuffExperienceCountsAndComesOnce(t *testing.T) {
	e, hero, _ := arena(t)
	sage(e, 4, 6)
	xp := func(r Result, id string) (sum, n int) {
		for _, ev := range r.Events {
			if ev.Kind == "experience" && ev.Actor == id {
				sum += ev.Amount
				n++
			}
		}
		return
	}
	r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "bolster"})
	sum, n := xp(r, "sage")
	assert.Positive(t, sum, "a new buff teaches")
	assert.Equal(t, 1, n)
	assert.Equal(t, "experience", r.Events[len(r.Events)-1].Kind, "after everything else the action showed")

	resetActions(e.unit("sage"))
	r = apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "bolster"})
	sum, _ = xp(r, "sage")
	assert.Zero(t, sum, "the same buff on a unit that has it as long teaches nothing")

	hero = e.unit("hero")
	hero.Status["attack"] = Status{Turns: 2, Value: -20}
	resetActions(e.unit("sage"))
	r = apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "bolster"})
	sum, _ = xp(r, "sage")
	assert.Positive(t, sum, "lifting a debuff teaches")
	assert.Zero(t, e.unit("hero").Status["attack"].Turns)
}
