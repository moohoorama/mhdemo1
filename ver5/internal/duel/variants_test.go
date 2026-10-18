package duel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func stagesWith(d content.Duel) *content.Data {
	return &content.Data{Stages: []content.Stage{{Duels: []content.Duel{d}}}}
}

var (
	plain   = []content.Beat{{Kind: "talk", Speech: "먼저 말한다", Side: 0, Duration: 1}, {Kind: "win", Side: 0, Duration: 1}}
	hasty   = []content.Beat{{Kind: "talk", Speech: "내가 먼저다", Side: 1, Duration: 1}, {Kind: "win", Side: 0, Duration: 1}}
	losing  = []content.Beat{{Kind: "talk", Side: 1, Duration: 1}, {Kind: "win", Side: 1, Duration: 1}}
	pairDef = content.Duel{ID: "d", Ally: "a", Enemy: "b", Outcome: "victory", AllyResult: "stay", EnemyResult: "death", Beats: plain}
)

func TestVariantStagingFollowsWhoMovedFirst(t *testing.T) {
	d := pairDef
	d.Variants = []content.DuelVariant{{First: "b", Beats: hasty}}
	scripts, err := FromData(stagesWith(d))
	require.NoError(t, err)
	assert.Equal(t, "내가 먼저다", scripts.StoryFor("d", "victory", "b").Beats[0].Speech)
	assert.Equal(t, "먼저 말한다", scripts.StoryFor("d", "victory", "a").Beats[0].Speech, "no variant for them: the plain staging")
	assert.Equal(t, "먼저 말한다", scripts.Story("d", "victory").Beats[0].Speech)
	assert.Len(t, scripts.StoryFor("d", "defeat", "b").Beats, len(New(1).Beats), "a different outcome falls back to the default timeline")
}

func TestDuelsWithoutBeatsUseTheDefaultTimeline(t *testing.T) {
	d := pairDef
	d.Beats = nil
	scripts, err := FromData(stagesWith(d))
	require.NoError(t, err)
	assert.NotContains(t, scripts, "d")
}

func TestBadVariantsAreRejected(t *testing.T) {
	for name, variants := range map[string][]content.DuelVariant{
		"no first officer": {{Beats: hasty}},
		"duplicate first":  {{First: "b", Beats: hasty}, {First: "b", Beats: hasty}},
		"wrong ending":     {{First: "b", Beats: losing}},
		"empty staging":    {{First: "b"}},
	} {
		t.Run(name, func(t *testing.T) {
			d := pairDef
			d.Variants = variants
			_, err := FromData(stagesWith(d))
			assert.Error(t, err)
		})
	}
	d := pairDef
	d.Beats = losing
	_, err := FromData(stagesWith(d))
	assert.Error(t, err, "the staging must end the way the outcome says")
}

func TestLeavingTheFieldNeedsTheRetreatResult(t *testing.T) {
	fleeing := []content.Beat{{Kind: "talk", Side: 1, Duration: 1}, {Kind: "withdrawn", Side: 1, Duration: 1}}
	d := pairDef
	d.Beats = fleeing
	d.EnemyResult = "retreat"
	_, err := FromData(stagesWith(d))
	assert.NoError(t, err)
	for _, result := range []string{"stay", "death"} {
		d.EnemyResult = result
		_, err = FromData(stagesWith(d))
		assert.Error(t, err, result)
	}
}
