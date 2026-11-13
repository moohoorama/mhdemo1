package core

import (
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWardsMeditationAndForesight(t *testing.T) {
	traits := map[string]map[string]float64{
		"ranged_ward_t":  {content.FxRangedWard: .3},
		"counter_ward_t": {content.FxCounterWard: .3},
		"foresight_t":    {content.FxStartMana: .25},
		"heavy_t":        {content.FxHeavy: .3},
	}
	setup := func(t *testing.T) (*Engine, *Unit, *Unit) {
		e, a, b := arena(t)
		for id, fx := range traits {
			e.data.Traits[id] = content.Trait{Effects: fx}
		}
		return e, a, b
	}

	t.Run("ranged ward cuts only ranged blows", func(t *testing.T) {
		e, _, b := setup(t)
		arch := e.unit("archer1")
		arch.X, arch.Y = 4, 6 // two cells from the boss: a shot
		shot := e.damage(arch, b, e.basic())
		hero := e.unit("hero")
		melee := e.damage(hero, b, e.basic())
		with(e, "boss", "ranged_ward_t")
		assert.InDelta(t, shot*.7, e.damage(arch, b, e.basic()), 1e-6)
		assert.InDelta(t, melee, e.damage(hero, b, e.basic()), 1e-6)
	})

	t.Run("counter ward takes 30 points off the counter's share", func(t *testing.T) {
		e, a, b := setup(t)
		c := Command{Kind: "attack", Actor: a.ID, Target: b.ID}
		full := e.CounterDamage(c)
		require.Positive(t, full)
		with(e, "hero", "counter_ward_t")
		assert.InDelta(t, float64(full)*.2/.5, float64(e.CounterDamage(c)), 1)
	})

	t.Run("a 30-point heavy counter cancels the ward", func(t *testing.T) {
		e, a, b := setup(t)
		c := Command{Kind: "attack", Actor: a.ID, Target: b.ID}
		full := e.CounterDamage(c)
		with(e, "hero", "counter_ward_t")
		with(e, "boss", "heavy_t")
		assert.InDelta(t, float64(full), float64(e.CounterDamage(c)), 1)
	})

	t.Run("iron armour halves the part above a fifth, and with iron wall the lower share counts", func(t *testing.T) {
		e, _, b := setup(t)
		e.data.Traits["iron_armor_t"] = content.Trait{Effects: map[string]float64{content.FxBulwark: .2}}
		e.data.Traits["ironwall_t"] = content.Trait{Effects: map[string]float64{content.FxBulwark: .1}}
		hp := float64(e.stats(b).MaxHP)
		with(e, "boss", "iron_armor_t")
		assert.InDelta(t, hp*.1, e.bulwark(b, hp*.1), 1e-6)
		assert.InDelta(t, hp*.2+hp*.2/2, e.bulwark(b, hp*.4), 1e-6)
		with(e, "boss", "iron_armor_t", "ironwall_t")
		assert.InDelta(t, hp*.1+hp*.3/2, e.bulwark(b, hp*.4), 1e-6)
	})

	t.Run("foresight starts a battle with a quarter of the spell points", func(t *testing.T) {
		e, _, _ := setup(t)
		def := e.char("troop#1").Def
		own := []string{"foresight_t"}
		u := e.spawn(def, def, "enemy", 0, 0, content.Spawn{Traits: &own})
		assert.Equal(t, int(float64(e.stats(u).MaxMP)*.25), u.MP)
		assert.Positive(t, u.MP)
		u = e.spawn(def, def, "enemy", 1, 0, content.Spawn{Traits: &[]string{}})
		assert.Zero(t, u.MP, "without it a battle starts at 0")
	})

	t.Run("meditation pays a share of HP for a share of spell points", func(t *testing.T) {
		e, a, _ := setup(t)
		e.data.Traits["meditate_art_t"] = content.Trait{Kind: "strategy"}
		e.data.Skills["meditate_t"] = content.Skill{Kind: content.KindHeal, Mode: content.ModeSelf, Shape: content.ShapeSingle, Hit: 100, Effect: "meditate", Coeff: .2, HPCost: .05, Requires: []string{"meditate_art_t"}}
		with(e, "hero", "meditate_art_t")
		st := e.stats(a)
		a.HP, a.MP = st.MaxHP, 0
		c := Command{Kind: "skill", Actor: a.ID, Skill: "meditate_t", Target: a.ID}
		cost := e.hpCost(a, e.data.Skills["meditate_t"])
		r := applyOK(t, e, c)
		paid, gained := 0, 0
		for _, ev := range r.Events {
			switch {
			case ev.Kind == "damage" && ev.Target == a.ID:
				paid = ev.Amount
			case ev.Kind == "effect" && ev.Target == a.ID:
				gained = ev.Amount
			}
		}
		assert.Equal(t, max(1, int(float64(st.MaxHP)*.05+.999)), cost)
		assert.Equal(t, cost, paid)
		assert.Equal(t, int(float64(st.MaxMP)*.2), gained)

		e, a, _ = setup(t)
		e.data.Traits["meditate_art_t"] = content.Trait{Kind: "strategy"}
		e.data.Skills["meditate_t"] = content.Skill{Kind: content.KindHeal, Mode: content.ModeSelf, Shape: content.ShapeSingle, Hit: 100, Effect: "meditate", Coeff: .2, HPCost: .05, Requires: []string{"meditate_art_t"}}
		with(e, "hero", "meditate_art_t")
		a.HP = e.hpCost(a, e.data.Skills["meditate_t"])
		_, err := e.Apply(c)
		assert.ErrorContains(t, err, "InsufficientHP", "it cannot pay its last troops")
	})
}
