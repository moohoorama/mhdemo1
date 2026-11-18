package core

import (
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChapter22Gear covers the effects the gear of chapters 22–27 brought in: 금흉갑's blocked second strike,
// 유성추's sure 봉격, 비단 도포's 매력, 제갈건's max spell points and 맹덕신서's recovery.
func TestChapter22Gear(t *testing.T) {
	setup := func(t *testing.T) (*Engine, *Unit, *Unit) {
		e, a, b := arena(t)
		e.data.Stages[e.state.Stage].Duels = nil
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		e.data.Rules.Crit.Base, e.data.Rules.Crit.Max = 0, 0
		for id, fx := range map[string]map[string]float64{
			"double_ward_t": {content.FxDoubleWard: 1},
			"crit_immune_t": {content.FxCritImmune: 1},
			"crit_always_t": {content.FxCritAlways: 1},
			"meteor_t":      {content.FxHitStatus + "disarm": 2},
			"immune_t":      {content.FxStatusImmune: 1},
			"charm_t":       {content.FxCharm: 10},
			"mana_max_t":    {content.FxManaMax: 30},
			"mana_plus_t":   {content.FxManaPlus: 10},
		} {
			e.data.Traits[id] = content.Trait{Effects: fx}
		}
		b.HP = 100000
		return e, a, b
	}
	hits := func(e *Engine, from, to string) (dealt, missed int) {
		for _, ev := range e.events {
			if ev.Actor == from && ev.Target == to {
				switch ev.Kind {
				case "damage":
					dealt++
				case "miss":
					missed++
				}
			}
		}
		return
	}
	attack := func(t *testing.T, e *Engine, a, b *Unit) {
		e.events = nil
		require.NoError(t, e.useSkill(a, Command{Kind: "attack", Actor: a.ID, Target: b.ID}))
	}

	t.Run("the second strike on a double-ward wearer is blocked", func(t *testing.T) {
		e, a, b := setup(t)
		with(e, a.ID, "twice")
		attack(t, e, a, b)
		dealt, missed := hits(e, a.ID, b.ID)
		require.Equal(t, 2, dealt, "without the ward both strikes land")
		assert.Zero(t, missed)

		e, a, b = setup(t)
		with(e, a.ID, "twice")
		with(e, b.ID, "double_ward_t")
		attack(t, e, a, b)
		dealt, missed = hits(e, a.ID, b.ID)
		assert.Equal(t, 1, dealt)
		assert.Equal(t, 1, missed)
	})

	t.Run("a crit on a crit-immune wearer is blocked", func(t *testing.T) {
		e, a, b := setup(t)
		with(e, a.ID, "crit_always_t")
		with(e, b.ID, "crit_immune_t", "double_ward_t")
		attack(t, e, a, b)
		dealt, missed := hits(e, a.ID, b.ID)
		assert.Zero(t, dealt)
		assert.Positive(t, missed)
	})

	t.Run("a landed hit always disarms, unless statuses never land", func(t *testing.T) {
		e, a, b := setup(t)
		with(e, a.ID, "meteor_t")
		attack(t, e, a, b)
		assert.Equal(t, 2, b.Status["disarm"].Turns)

		e, a, b = setup(t)
		with(e, a.ID, "meteor_t")
		with(e, b.ID, "immune_t")
		attack(t, e, a, b)
		assert.Zero(t, b.Status["disarm"].Turns)
	})

	t.Run("charm raises 매력, max HP and the deputy blend", func(t *testing.T) {
		e, a, _ := setup(t)
		o := e.char(a.ID)
		before, hp := e.attributes(o)[5], e.stats(a).MaxHP
		with(e, a.ID, "charm_t")
		assert.Equal(t, before+10, e.attributes(o)[5])
		assert.Greater(t, e.stats(a).MaxHP, hp)
	})

	t.Run("max spell points and recovery grow by flat amounts", func(t *testing.T) {
		e, a, _ := setup(t)
		before := e.stats(a)
		with(e, a.ID, "mana_max_t", "mana_plus_t")
		after := e.stats(a)
		assert.Equal(t, before.MaxMP+30, after.MaxMP)
		assert.Equal(t, before.Recovery+10, after.Recovery)
	})
}

// TestChapter22GearData checks the new pieces of equipment carry the effects the design gives them.
func TestChapter22GearData(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	fx := func(item string) map[string]float64 {
		out := map[string]float64{}
		for _, tr := range d.Equipment[item].Traits {
			for k, v := range d.Traits[tr].Effects {
				out[k] += v
			}
		}
		return out
	}
	for _, c := range []struct {
		item, slot, kind string
		effects          map[string]float64
	}{
		{"dilu", "accessory", "any", map[string]float64{content.FxTerrainFloor: 1}},
		{"yang_youji_bow", "weapon", "bow", map[string]float64{content.FxRangedReach: 2}},
		{"formation_manual", "accessory", "any", map[string]float64{content.FxAuraBuff + "defense": 20}},
		{"silk_robe", "armor", "robe", map[string]float64{content.FxCharm: 10}},
		{"golden_breastplate", "armor", "armor", map[string]float64{content.FxCritImmune: 1, content.FxDoubleWard: 1}},
		{"mengde_book", "accessory", "any", map[string]float64{content.FxManaPlus: 10}},
		{"yitian_sword", "weapon", "sword", map[string]float64{content.FxEvade: 15}},
		{"meteor_hammer", "weapon", "fist", map[string]float64{content.FxHitStatus + "disarm": 2}},
		{"white_feather_fan", "weapon", "fan", map[string]float64{content.FxCostHalf: 1}},
		{"crane_robe", "armor", "robe", map[string]float64{content.FxTurnBuff + "mind": 20}},
		{"zhuge_kerchief", "accessory", "any", map[string]float64{content.FxManaMax: 30}},
	} {
		t.Run(c.item, func(t *testing.T) {
			eq, ok := d.Equipment[c.item]
			require.True(t, ok)
			assert.Equal(t, c.slot, eq.Slot)
			assert.Equal(t, c.kind, eq.Type)
			assert.Zero(t, eq.Price, "a unique piece is not sold")
			assert.Equal(t, c.effects, fx(c.item))
		})
	}
	t.Run("의천검 with 보법 takes 30 points off physical hits", func(t *testing.T) {
		assert.Equal(t, 30.0, fx("yitian_sword")[content.FxEvade]+d.Traits["footwork"].Effects[content.FxEvade])
	})
}
