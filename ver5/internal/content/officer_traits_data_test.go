package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The traits, costs and stage settings the user decided on 2026-11-17, so that a later data edit does not quietly
// undo them.
func TestOfficerTraitDecisions(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)

	t.Run("each officer can learn their own traits", func(t *testing.T) {
		for id, traits := range map[string][]string{
			"liubei":   {"ambition", "benevolence"},
			"guanyu":   {"awe"},
			"zhangfei": {"arrogance"},
			"zhaoyun":  {"daring"},
			"mizhu":    {"fire_star"},
			"jianyong": {"cheer"},
			"sunqian":  {"swiftness", "even_hand", "lecture"},
			"mifang":   {"hold_ground", "first_aid"},
			"zhoucang": {"swim", "swiftness"},
			"liupi":    {"mountain_footing", "earth_boost"},
		} {
			for _, tr := range traits {
				assert.Contains(t, d.Characters[id].Learn, tr, "%s learns %s", id, tr)
			}
		}
	})

	t.Run("names, costs, prerequisites and effects", func(t *testing.T) {
		for _, c := range []struct {
			id, name string
			cost     int
			requires []string
			fx       map[string]float64
		}{
			{"ambition", "패기", 1000, nil, nil},
			{"hegemony", "패도", 1500, []string{"ambition"}, map[string]float64{"turn_buff:attack": 20, "turn_buff:mind": 20, "turn_buff:defense": 20, "turn_buff:agility": 20, "turn_buff:morale": 20}},
			{"awe", "위진화하", 1500, []string{"intimidate"}, map[string]float64{"crush": 1}},
			{"arrogance", "안하무인", 1500, []string{"roar"}, map[string]float64{"scorn": .5}},
			{"daring", "일신시담", 1500, []string{"lone_warrior"}, map[string]float64{"lone": .1}},
			{"fire_star", "화덕성군", 1500, nil, map[string]float64{"immune:fire": 1}},
			{"cheer", "호인", 1500, nil, map[string]float64{"aura_buff:morale": 20}},
			{"benevolence", "용인술", 700, []string{"deputy_appoint"}, map[string]float64{"deputy_plus": 10}},
			{"even_hand", "지균", 1500, nil, map[string]float64{"aura_buff:agility": 20}},
			{"lecture", "강론", 1000, nil, map[string]float64{"aura_mana": .1}},
			{"hold_ground", "진지고수", 1200, nil, map[string]float64{"hold_ground": .3}},
			{"swiftness", "신속", 1000, nil, map[string]float64{"haste": 2}},
			{"swim", "수영", 1000, nil, map[string]float64{"wet_factor": .2, "ward:water": .5}},
			{"mountain_footing", "산악지형보조", 1000, nil, map[string]float64{"ambush_factor": .05}},
			{"terrain_mend", "지형정비", 1000, nil, nil},
		} {
			tr, ok := d.Traits[c.id]
			require.True(t, ok, c.id)
			assert.Equal(t, c.name, tr.Name, c.id)
			assert.Equal(t, c.cost, tr.Cost, c.id)
			assert.ElementsMatch(t, c.requires, tr.Requires, c.id)
			if c.fx != nil {
				assert.Equal(t, c.fx, tr.Effects, c.id)
			}
		}
	})

	t.Run("조조 holds 패기 from 15 and 패도 from 35, not from the first battle", func(t *testing.T) {
		cao := d.Characters["caocao"]
		assert.NotContains(t, cao.Traits, "ambition")
		assert.Equal(t, map[string]int{"ambition": 15, "hegemony": 35}, cao.TraitAt)
	})

	t.Run("spells: 응급처치 10, 지형정비 3x3 for 20, 교란 two cells, no 도발", func(t *testing.T) {
		assert.Equal(t, 10, d.Skills["first_aid"].Cost)
		assert.Equal(t, ModeSelf, d.Skills["first_aid"].Mode)
		m := d.Skills["terrain_mend"]
		assert.Equal(t, "clear_marks", m.Effect)
		assert.Equal(t, ShapeSquare, m.Shape)
		assert.Equal(t, 20, m.Cost)
		assert.Contains(t, d.Classes["mystic"].Pool, "terrain_mend", "선술사 learns 지형정비")
		dis := d.Skills["disrupt"]
		assert.Equal(t, ModeAny, dis.Mode)
		assert.Equal(t, 2, dis.Max)
		assert.NotContains(t, d.Skills, "provoke")
		assert.NotContains(t, d.Traits, "provoke_prep")
	})

	t.Run("쉬움: weaker foes, more experience, the player's side dealing 120% and taking 80%", func(t *testing.T) {
		v, ok := d.Rules.DifficultyOf("casual")
		require.True(t, ok)
		assert.Equal(t, "쉬움", v.Name)
		assert.Equal(t, "casual", d.Rules.Difficulties[0].ID, "first in the list")
		assert.Equal(t, -.3, v.EnemyShift)
		assert.Equal(t, 150, v.XPPct)
		assert.Equal(t, 120, v.DealtPct)
		assert.Equal(t, 80, v.TakenPct)
	})

	t.Run("water, fords and wet patches are wet", func(t *testing.T) {
		wet := 0
		for _, g := range d.Grounds {
			if g.Wet {
				wet++
			}
		}
		assert.Equal(t, 2, wet, "water and the ford")
		assert.True(t, d.States[d.Rules.CritMarks["water"]].Wet, "the wet patch a water spell leaves")
	})

	t.Run("연주 (B19): every enemy attacks from the start", func(t *testing.T) {
		st := d.Stages[d.StageIndex("B19")]
		for _, sp := range st.Enemies {
			assert.Equal(t, "assault", sp.Policy, sp.Character)
		}
		for _, ev := range st.Events {
			for _, a := range ev.Do {
				assert.NotEqual(t, "ai", a.Kind, "no event turns them to the attack later (%s)", ev.ID)
			}
		}
	})
}
