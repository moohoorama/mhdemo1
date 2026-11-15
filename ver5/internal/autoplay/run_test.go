package autoplay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srpg/internal/content"
)

func TestPlayersWinTheFirstBattle(t *testing.T) {
	if testing.Short() {
		t.Skip("plays a battle")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	players := []Player{
		&RulePlayer{Label: "rule", D: d, G: DefaultGenome()},
		&Searcher{Label: "search", D: d, G: DefaultGenome(), K: 2, R: 2, P: 2, B: 1, Par: 4},
	}
	for _, p := range players {
		t.Run(p.Name(), func(t *testing.T) {
			r, err := Run(d, p, Options{Difficulty: "hard", Seed: 1, Stages: 1})
			require.NoError(t, err)
			require.Len(t, r.Stages, 1)
			assert.True(t, r.Stages[0].Won)
			assert.Positive(t, r.LevelScore)
		})
	}
}

func TestLevelScoreWeighsTheTopFiveHalf(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	ref := RefLevel(d)
	levels := map[string]int{}
	for i := range 20 {
		levels[string(rune('a'+i))] = ref
	}
	assert.InDelta(t, ScoreN*7/8, LevelScore(d, levels), 1e-9)
	top := map[string]int{"a": ref, "b": ref, "c": ref, "d": ref, "e": ref}
	assert.InDelta(t, ScoreN/2, LevelScore(d, top), 1e-9)
}
