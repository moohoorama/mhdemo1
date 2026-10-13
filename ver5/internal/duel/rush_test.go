package duel

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRushAttackAdvancesAndHoldsPosition(t *testing.T) {
	for side := range 2 {
		kinds := []string{"hit", "block", "critical-block", "yielding-block", "dodge", "critical-dodge", "critical", "backhit"}
		for _, kind := range kinds {
			t.Run(kind+[]string{" ally", " enemy"}[side], func(t *testing.T) {
				story := prepareStory([]Beat{{Kind: "reinforcements", Duration: 1}, {Kind: "rush-" + kind, Side: side, Duration: 1.3}, {Kind: "talk", Duration: 1}})
				ready := at(story, "charge-ready", 0)
				preparing := at(story, "charge-ready", .1)
				assert.Equal(t, "walk", preparing.Actors[side].Animation)
				assert.Equal(t, ready.Actors[side].Facing, preparing.Actors[side].Facing)
				start := at(story, "rush-"+kind, 0)
				d := start.Actors[defender(side)]
				distance := func(a Actor) float64 { return math.Hypot(a.Position-d.Position, a.Lane-d.Lane) }
				assert.InDelta(t, chargeReach, distance(start.Actors[side]), 1e-6)
				previous := chargeReach
				for _, local := range []float64{.15, .3, .45, .56} {
					state := at(story, "rush-"+kind, local)
					a := state.Actors[side]
					assert.Equal(t, "attack", a.Animation)
					assert.Less(t, distance(a), previous)
					assert.Equal(t, start.Actors[side].Facing, a.Facing)
					previous = distance(a)
				}
				contact := at(story, "rush-"+kind, .56)
				assert.InDelta(t, attackReach, distance(contact.Actors[side]), 1e-6)
				end := at(story, "rush-"+kind, 1.29)
				assert.InDelta(t, contact.Actors[side].Position, end.Actors[side].Position, 1e-6)
				assert.InDelta(t, contact.Actors[side].Lane, end.Actors[side].Lane, 1e-6)
				// The charge drives the target back, restoring the charge distance.
				driven := end.Actors[defender(side)]
				assert.InDelta(t, chargeReach, math.Hypot(driven.Position-end.Actors[side].Position, driven.Lane-end.Actors[side].Lane), 1e-6, "defender must be driven back")
				// The following talk must keep the advanced position too.
				following := story.Sample(story.Start(len(story.Beats)-1) + .2)
				assert.InDelta(t, end.Actors[side].Position, following.Actors[side].Position, 1e-6)
				assert.InDelta(t, end.Actors[side].Lane, following.Actors[side].Lane, 1e-6)
			})
		}
	}
}

func TestRushNeverBacksAway(t *testing.T) {
	story := prepareStory([]Beat{{Kind: "approach", Duration: 1}, {Kind: "rush-block", Side: 1, Duration: 1.3}, {Kind: "rush-block", Side: 1, Duration: 1.3}})
	for _, b := range story.Beats {
		assert.NotEqual(t, "charge-ready", b.Kind, "closer than charge distance must not step back")
	}
	for time := 0.; time < story.Duration(); time += .02 {
		s := story.Sample(time)
		if time > 1.2 && time < 1.5 {
			assert.NotEqual(t, "walk", s.Actors[1].Animation, "no walking retreat before a charge")
		}
	}
}

func TestCampaignMixesStandingAndRushAttacks(t *testing.T) {
	scripts, err := LoadScripts("../../assets/duels.json")
	require.NoError(t, err)
	for _, id := range []string{"sishui-guan-yu", "hulao-zhang-fei", "hulao-guan-yu", "hulao-liu-bei"} {
		t.Run(id, func(t *testing.T) {
			standing, rushing := 0, 0
			for _, beat := range scripts.Story(id, "victory").Beats {
				if !isStrike(beat.Kind) {
					continue
				}
				if strikeKind(beat.Kind) == beat.Kind {
					standing++
				} else {
					rushing++
				}
			}
			assert.Positive(t, standing)
			assert.Positive(t, rushing)
		})
	}
}
