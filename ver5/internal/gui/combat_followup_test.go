package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestFollowupReplayKeepsExchangeOrder(t *testing.T) {
	for _, kind := range []string{"damage", "duel"} {
		t.Run(kind, func(t *testing.T) {
			c := core.Command{Kind: "attack", Actor: "a", Target: "b"}
			events := []core.Event{
				{Kind: kind, Actor: "a", Target: "b"},
				{Kind: "retreat", Actor: "b"},
				{Kind: "followup-start", Actor: "a", Target: "c"},
				{Kind: "damage", Actor: "a", Target: "c"},
				{Kind: "double", Actor: "a", Target: "c"},
				{Kind: "damage", Actor: "a", Target: "c"},
				{Kind: "damage", Actor: "c", Target: "a"},
				{Kind: "double", Actor: "c", Target: "a"},
				{Kind: "damage", Actor: "c", Target: "a"},
				{Kind: "followup-end", Actor: "a", Target: "c"},
				{Kind: "followup-start", Actor: "a", Target: "d"},
				{Kind: "damage", Actor: "a", Target: "d"},
				{Kind: "followup-end", Actor: "a", Target: "d"},
			}
			steps := splitFollowups(c, events)
			require.Len(t, steps, 3)
			assert.Equal(t, events[:2], steps[0].Events)
			assert.Equal(t, "c", steps[1].Command.Target)
			blow, counters := splitCounters("a", steps[1].Events)
			assert.Len(t, splitDouble(blow), 2)
			require.Len(t, counters, 1)
			assert.Len(t, splitDouble(counters[0].events), 2)
			assert.Equal(t, "d", steps[2].Command.Target)
		})
	}
}

func TestResultExcludesSupportAndKeepsRetreatedParty(t *testing.T) {
	o := core.Observation{UnitViews: []core.UnitView{
		{Unit: core.Unit{ID: "party", Faction: "ally", HP: 10}},
		{Unit: core.Unit{ID: "retreated", Faction: "ally", HP: 0}},
		{Unit: core.Unit{ID: "support", Faction: "ally", AI: true, HP: 10}},
		{Unit: core.Unit{ID: "enemy", Faction: "enemy", HP: 10}},
	}}
	units := resultUnits(o)
	require.Len(t, units, 2)
	assert.Equal(t, "party", units[0].ID)
	assert.Equal(t, "retreated", units[1].ID)
}

func TestImportantBattleEventsDoNotToast(t *testing.T) {
	g := battleController(t)
	g.record(nil, []core.Event{{Kind: "retreat", Actor: "hero"}, {Kind: "victory"}})
	assert.Empty(t, g.toasts)
}
