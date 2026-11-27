package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestEquipmentUpgrades(t *testing.T) {
	d := data(t)
	e := New(d, 17)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	require.Equal(t, "battle", e.Observe().Dialogue.Kind)
	e.data.Traits["swift"] = content.Trait{Effects: map[string]float64{content.FxMove: 2, content.FxIgnoreZOC: 1}}
	for id, eq := range map[string]content.Equipment{
		"attack_charm":  {Slot: "accessory", Bonus: [6]float64{3, 1}},
		"defense_charm": {Slot: "accessory", Bonus: [6]float64{1, 3}},
		"strong_charm":  {Slot: "accessory", Bonus: [6]float64{3, 3}},
		"spell_charm":   {Slot: "accessory", Bonus: [6]float64{1}, Traits: []string{"fire"}},
		"rider_charm":   {Slot: "accessory", Traits: []string{"gale"}},
		"swift_charm":   {Slot: "accessory", Traits: []string{"swift"}},
	} {
		e.data.Equipment[id] = eq
		e.state.Warehouse[id] = 1
	}
	find := func(actor, item string) (EquipmentUpgrade, bool) {
		choices := e.EquipmentUpgrades()
		i := slices.IndexFunc(choices, func(up EquipmentUpgrade) bool { return up.Actor == actor && up.Item == item })
		if i >= 0 {
			return choices[i], true
		}
		return EquipmentUpgrade{}, false
	}
	recommended := func(actor, item string) bool {
		_, ok := find(actor, item)
		return ok
	}
	hero := e.char("hero")
	t.Run("opposed stats are a trade-off and strict gains qualify", func(t *testing.T) {
		hero.Equipment["accessory"] = "attack_charm"
		before := e.Snapshot()
		assert.False(t, recommended("hero", "defense_charm"))
		assert.False(t, recommended("hero", "attack_charm"), "already worn")
		assert.True(t, recommended("hero", "strong_charm"))
		assert.Equal(t, before, e.Snapshot(), "preview leaves both inventory and gear intact")
		hero.Equipment["accessory"] = "defense_charm"
		assert.False(t, recommended("hero", "attack_charm"))
	})
	t.Run("a new spell is an improvement only when existing effects are preserved", func(t *testing.T) {
		delete(hero.Equipment, "accessory")
		up, ok := find("hero", "spell_charm")
		require.True(t, ok)
		assert.NotEmpty(t, up.Skills)
		hero.Equipment["accessory"] = "strong_charm"
		assert.False(t, recommended("hero", "spell_charm"))
		hero.Equipment["accessory"] = "spell_charm"
		assert.False(t, recommended("hero", "strong_charm"))
		with(e, "hero", "fire")
		assert.True(t, recommended("hero", "strong_charm"), "the learned trait and its spells survive the swap")
	})
	t.Run("a shared item disappears after equip and returns after unequip", func(t *testing.T) {
		delete(hero.Equipment, "accessory")
		assert.True(t, recommended("archer1", "rider_charm"))
		up, ok := find("hero", "rider_charm")
		require.True(t, ok)
		assert.Positive(t, up.MovementGain)
		apply(t, e, Command{Kind: "equip", Actor: "hero", Item: "rider_charm"})
		assert.False(t, recommended("archer1", "rider_charm"))
		assert.False(t, recommended("hero", "rider_charm"))
		up, ok = find("hero", "swift_charm")
		require.True(t, ok, "the new mount keeps the same movement and adds another effect")
		assert.Zero(t, up.MovementGain)
		assert.Equal(t, []string{"swift"}, up.Traits)
		apply(t, e, Command{Kind: "unequip", Actor: "hero", Item: "rider_charm"})
		assert.True(t, recommended("archer1", "rider_charm"))
		e.char("hero").Deputy = "archer1"
		assert.False(t, recommended("archer1", "rider_charm"))
	})
}

func TestEquipmentUpgradesFor(t *testing.T) {
	e := New(data(t), 17)
	apply(t, e, Command{Kind: "next"})
	apply(t, e, Command{Kind: "choose", Option: "go"})
	require.Equal(t, "battle", e.Observe().Dialogue.Kind)
	e.data.Equipment["strong_charm"] = content.Equipment{Slot: "accessory", Bonus: [6]float64{3, 3}}
	e.data.Equipment["weak_charm"] = content.Equipment{Slot: "accessory"}
	delete(e.char("hero").Equipment, "accessory")
	require.Empty(t, e.state.Warehouse["strong_charm"])
	before := e.Snapshot()
	t.Run("a piece not in the warehouse still helps whoever gains from it", func(t *testing.T) {
		ups := e.EquipmentUpgradesFor("strong_charm")
		i := slices.IndexFunc(ups, func(up EquipmentUpgrade) bool { return up.Actor == "hero" })
		require.GreaterOrEqual(t, i, 0)
		assert.Equal(t, "strong_charm", ups[i].Item)
		assert.Equal(t, before, e.Snapshot(), "the preview leaves inventory and gear intact")
	})
	t.Run("a useless or unknown piece helps nobody", func(t *testing.T) {
		assert.Empty(t, e.EquipmentUpgradesFor("weak_charm"))
		assert.Empty(t, e.EquipmentUpgradesFor("no_such_item"))
	})
	t.Run("only the asked piece is tried", func(t *testing.T) {
		e.state.Warehouse["attack_charm"] = 1
		e.data.Equipment["attack_charm"] = content.Equipment{Slot: "accessory", Bonus: [6]float64{3}}
		for _, up := range e.EquipmentUpgradesFor("strong_charm") {
			assert.Equal(t, "strong_charm", up.Item)
		}
	})
}
