package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromotionAway(t *testing.T) {
	seal := Command{Kind: "item", Actor: "hero", Target: "hero", Item: "promo_guard"}
	ready := func(t *testing.T) *Engine {
		e := walkEngine(t)
		e.state.Inventory["promo_guard"] = 1
		e.char("hero").Level = 15
		return e
	}

	t.Run("the walk promotes with a seal", func(t *testing.T) {
		e := ready(t)
		classes, err := e.PromotionsAway("promo_guard", "hero")
		require.NoError(t, err)
		assert.Equal(t, []string{"guard"}, classes)
		r, err := e.Apply(seal)
		require.NoError(t, err)
		assert.Equal(t, "guard", e.char("hero").Class)
		assert.Zero(t, e.state.Inventory["promo_guard"])
		require.Len(t, r.Events, 1)
		assert.Equal(t, Event{Kind: "promotion", Actor: "hero", Target: "guard", Text: "hero", Amount: 15}, r.Events[0])
	})

	t.Run("a piece the new class cannot carry goes back to the warehouse", func(t *testing.T) {
		e := ready(t)
		cl := e.data.Classes["guard"]
		cl.Weapon = "lance"
		e.data.Classes["guard"] = cl
		e.char("hero").Equipment = map[string]string{"weapon": "iron_sword", "armor": "jerkin"}
		stock := e.state.Warehouse["iron_sword"]
		r, err := e.Apply(seal)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"armor": "jerkin"}, e.char("hero").Equipment, "the plate still fits")
		assert.Equal(t, stock+1, e.state.Warehouse["iron_sword"])
		require.Len(t, r.Events, 2)
		assert.Equal(t, Event{Kind: "unequip", Actor: "hero", Target: "iron_sword"}, r.Events[1])
	})

	t.Run("a battle's preparation promotes too", func(t *testing.T) {
		e := ready(t)
		e.state.Node = "fight"
		e.enter()
		_, err := e.Apply(seal)
		require.NoError(t, err)
		assert.Equal(t, "guard", e.char("hero").Class)
	})

	t.Run("the seal asks for the level", func(t *testing.T) {
		e := ready(t)
		e.char("hero").Level = 1
		_, err := e.PromotionsAway("promo_guard", "hero")
		assert.EqualError(t, err, "InsufficientLevel")
		assertRejected(t, e, seal)
	})

	t.Run("without a seal nothing happens", func(t *testing.T) {
		e := ready(t)
		e.state.Inventory["promo_guard"] = 0
		assertRejected(t, e, seal)
	})

	t.Run("other items stay for the battle", func(t *testing.T) {
		e := ready(t)
		e.state.Inventory["tonic"] = 1
		assertRejected(t, e, Command{Kind: "item", Actor: "hero", Target: "hero", Item: "tonic"})
	})

	t.Run("not in the middle of an event", func(t *testing.T) {
		e := ready(t)
		_, err := e.Apply(Command{Kind: "talk", Target: "장비"})
		require.NoError(t, err)
		require.False(t, e.Walking())
		_, err = e.PromotionsAway("promo_guard", "hero")
		assert.EqualError(t, err, "WrongPhase")
		assertRejected(t, e, seal)
	})
}

func TestPromotionItemIsRefusedInBattle(t *testing.T) {
	e, _, _ := arena(t)
	e.char("hero").Level = 15
	_, err := e.Apply(Command{Kind: "item", Actor: "hero", Target: "hero", Item: "promo_guard", Option: "guard"})
	require.Error(t, err)
	assert.Equal(t, "PromoteAway", err.Error())
	assert.Equal(t, "footman", e.char("hero").Class)
	assert.Equal(t, 1, e.state.Inventory["promo_guard"])
	for _, c := range e.LegalActions("hero").Commands {
		assert.NotEqual(t, "promo_guard", c.Item, "no seal among the battle's commands")
	}
}
