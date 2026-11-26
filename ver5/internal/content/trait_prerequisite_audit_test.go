package content

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfficerOwnedTraitPrerequisites(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for id, officer := range d.Characters {
		t.Run(id, func(t *testing.T) {
			for i, trait := range officer.Traits {
				level := i*d.Rules.TraitPlanStep + 1
				for _, pre := range d.Traits[trait].Requires {
					found := slices.Index(officer.Traits, pre) >= 0 && slices.Index(officer.Traits, pre) < i
					for _, plan := range d.Classes[officer.Class].Plan {
						found = found || plan.Trait == pre && plan.Level <= level
					}
					assert.True(t, found, "%s (level %d) requires %s learned first", trait, level, pre)
				}
			}
		})
	}
}

func TestOfficerPlannedTraitPrerequisites(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for id, officer := range d.Characters {
		t.Run(id, func(t *testing.T) {
			if officer.Class == "" {
				return
			}
			reported := map[string]bool{}
			for level := 1; level <= 99; level++ {
				class := d.PromotedAt(id, officer.Class, level)
				traits := d.PlannedTraits(id, class, level, nil)
				for _, trait := range traits {
					for _, pre := range d.Traits[trait].Requires {
						key := trait + ">" + pre
						if !reported[key] {
							reported[key] = true
							assert.Contains(t, traits, pre, "level %d class %s trait %s requires %s", level, class, trait, pre)
						}
					}
				}
			}
		})
	}
}

func TestClassPlannedTraitPrerequisites(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for id, class := range d.Classes {
		t.Run(id, func(t *testing.T) {
			for _, item := range class.Plan {
				for _, pre := range d.Traits[item.Trait].Requires {
					found := false
					for _, earlier := range class.Plan {
						found = found || earlier.Trait == pre && earlier.Level <= item.Level
					}
					assert.True(t, found, "level %d trait %s requires %s at the same or earlier level", item.Level, item.Trait, pre)
				}
			}
		})
	}
}

func TestXiaopeiCaoMainArrival(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	st := d.Stages[d.StageIndex("B15")]
	var arrival *BattleEvent
	for i := range st.Events {
		if st.Events[i].ID == "cao-main" {
			arrival = &st.Events[i]
			break
		}
	}
	require.NotNil(t, arrival)
	assert.Contains(t, arrival.If, "round>=4")
	assert.Contains(t, arrival.If, "defeated:xiahoudun")
	assert.Contains(t, arrival.If, "turnstart:ally")
	for _, action := range arrival.Do {
		for _, sp := range action.Spawns {
			assert.NotEqual(t, "troop_archer", sp.Character)
			assert.NotEqual(t, "xiahouyuan", sp.Character)
			assert.LessOrEqual(t, sp.Y, 11, "%s must arrive farther north", sp.Character)
		}
	}
}

func TestPoisonAndStrictOrderPrerequisites(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	assert.Contains(t, d.Traits["deadly_poison"].Requires, "venom")
	assert.Contains(t, d.Characters["chengyu"].Traits, "venom")
	assert.Contains(t, d.Traits["strict_order"].Requires, "discipline")
	assert.Positive(t, d.Traits["discipline"].Effects[FxStatusImmune])
	assert.Less(t, slices.Index(d.Characters["yujin"].Traits, "discipline"), slices.Index(d.Characters["yujin"].Traits, "strict_order"))
}

func TestRepeatingCrossbowPlanOrder(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for _, class := range []string{"archer", "crossbow", "repeating_crossbow"} {
		assert.Contains(t, d.Classes[class].AllPool(), "snipe_prep", class)
	}
	for _, tc := range []struct {
		level                int
		snipe, pierce, sight bool
	}{
		{49, false, false, false},
		{50, true, false, false},
		{59, true, false, false},
		{60, true, true, false},
		{69, true, true, false},
		{70, true, true, true},
	} {
		t.Run(fmt.Sprintf("level_%d", tc.level), func(t *testing.T) {
			traits := d.PlannedTraits("troop_archer", "repeating_crossbow", tc.level, nil)
			assert.Equal(t, tc.snipe, slices.Contains(traits, "snipe_prep"))
			assert.Equal(t, tc.pierce, slices.Contains(traits, "penetrate_art"))
			assert.Equal(t, tc.sight, slices.Contains(traits, "far_sight"))
		})
	}
}
