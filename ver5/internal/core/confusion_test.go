package core

import (
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfusionAnswersNoBlow(t *testing.T) {
	attack := Command{Kind: "attack", Actor: "hero", Target: "boss"}
	t.Run("counter", func(t *testing.T) {
		e, _, b := arena(t)
		b.Status["confusion"] = Status{Turns: 2}
		assert.Zero(t, e.CounterDamage(attack))
		assert.False(t, answered(apply(t, e, attack), "boss", "hero"))
	})
	t.Run("counter stance", func(t *testing.T) {
		e, _, b := arena(t)
		b.Status["confusion"] = Status{Turns: 2}
		b.Status["counter"] = Status{Turns: 2}
		assert.False(t, answered(apply(t, e, attack), "boss", "hero"))
	})
	t.Run("guard stance", func(t *testing.T) {
		e, a, b := arena(t)
		b.Status["guard"] = Status{Turns: 2}
		require.Zero(t, e.hitChance(a, b, e.basic()), "경계 dodges a melee blow")
		b.Status["confusion"] = Status{Turns: 2}
		assert.NotZero(t, e.hitChance(a, b, e.basic()))
	})
	t.Run("guardian", func(t *testing.T) {
		setup := func(confusedGuardian bool) *Engine {
			e, _, b := arena(t)
			e.data.Traits["guardian_t"] = content.Trait{Effects: map[string]float64{content.FxGuardian: .5}}
			w := e.unit("troop#1")
			w.HP, w.X, w.Y = 100, b.X-1, b.Y+1 // next to the boss and the hero
			with(e, w.Character, "guardian_t")
			b.Status["confusion"] = Status{Turns: 2}
			if confusedGuardian {
				w.Status["confusion"] = Status{Turns: 2}
			}
			return e
		}
		require.True(t, answered(apply(t, setup(false), attack), "troop#1", "hero"), "호위 answers for the boss")
		assert.False(t, answered(apply(t, setup(true), attack), "troop#1", "hero"))
	})
}

func TestConfusionBlocksItems(t *testing.T) {
	e, a, _ := arena(t)
	a.Status["confusion"] = Status{Turns: 2}
	for _, c := range e.LegalActions("hero").Commands {
		assert.NotEqual(t, "item", c.Kind)
	}
	_, err := e.Apply(Command{Kind: "item", Actor: "hero", Target: "hero", Item: "ration"})
	assert.Error(t, err)
}

func TestCureAllLiftsHarmfulStatuses(t *testing.T) {
	e, a, _ := arena(t)
	e.data.Items["cure_all"] = content.Item{Name: "만능약", Effect: content.ItemCure}
	e.stock(a)["cure_all"] = 1
	a.Status["root"] = Status{Turns: 2}
	a.Status["seal"] = Status{Turns: 2}
	a.Status["attack"] = Status{Turns: 2, Value: 20}
	apply(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "cure_all"})
	a = e.unit("hero") // Apply works on a copy of the state
	assert.Zero(t, a.Status["root"].Turns)
	assert.Zero(t, a.Status["seal"].Turns)
	assert.Equal(t, 2, a.Status["attack"].Turns, "a buff stays")
}
