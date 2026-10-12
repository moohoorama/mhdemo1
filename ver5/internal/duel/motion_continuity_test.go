package duel

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrivedFightersStopWalking(t *testing.T) {
	scripts, err := LoadScripts("../../assets/duels.json")
	require.NoError(t, err)
	for _, id := range []string{"hulao-fangyue", "hulao-mushun", "hulao-wuanguo"} {
		t.Run(id, func(t *testing.T) {
			story := scripts.Story(id, "defeat")
			moving := at(story, "approach", .5)
			ready := at(story, "close", 0)
			for side := range 2 {
				assert.Equal(t, "walk", moving.Actors[side].Animation)
				for _, local := range []float64{.1, 1, 2.3} {
					waiting := at(story, "close", local)
					assert.Equal(t, ready.Actors[side].Position, waiting.Actors[side].Position)
					assert.Equal(t, "idle", waiting.Actors[side].Animation)
				}
			}
		})
	}
	t.Run("chaser waits before starting and stops on arrival", func(t *testing.T) {
		story := Story{Beats: []Beat{{Kind: "approach", Duration: 1}, {Kind: "straight-chase", Side: 1, Duration: 2}, {Kind: "talk", Duration: 1}}}
		assert.Equal(t, "idle", at(story, "straight-chase", .1).Actors[0].Animation)
		assert.Equal(t, "walk", at(story, "straight-chase", 1).Actors[0].Animation)
		arrived := at(story, "straight-chase", 1.9)
		for _, actor := range arrived.Actors[:2] {
			assert.Equal(t, "idle", actor.Animation)
		}
	})
}

func TestEveryExchangeConnectsAtWeaponRange(t *testing.T) {
	scripts, err := LoadScripts("../../assets/duels.json")
	require.NoError(t, err)
	stories := map[string]Story{}
	for id, script := range scripts {
		stories[id] = scripts.Story(id, script.Outcome)
	}
	for ending, name := range Endings {
		stories[name] = New(ending)
	}
	for name, story := range stories {
		t.Run(name, func(t *testing.T) {
			for i, beat := range story.Beats {
				start := story.Start(i)
				if isStrike(beat.Kind) {
					before := story.Sample(start + .56)
					a, d := before.Actors[beat.Side], before.Actors[defender(beat.Side)]
					assert.LessOrEqual(t, math.Hypot(d.Position-a.Position, d.Lane-a.Lane), attackReach+1e-6, "beat %d: %s", i, beat.Kind)
					locals := []float64{.1, .57, beat.Duration - .001}
					if strikeKind(beat.Kind) != beat.Kind {
						locals = []float64{.57, beat.Duration - .001}
					}
					for _, local := range locals {
						attacker := story.Sample(start + local).Actors[beat.Side]
						assert.InDelta(t, a.Position, attacker.Position, 1e-6, "attacker slid in beat %d", i)
						assert.InDelta(t, a.Lane, attacker.Lane, 1e-6)
					}
				}
				if beat.Kind == "engage" || beat.Kind == "charge-ready" {
					assert.Equal(t, "walk", story.Sample(start + beat.Duration/2).Actors[beat.Side].Animation)
				}
				if i > 0 && beat.Kind != "reinforcements" {
					left, right := story.Sample(start-.000001), story.Sample(start+.000001)
					for side := range 4 {
						assert.InDelta(t, left.Actors[side].Position, right.Actors[side].Position, .001, "position jump in beat %d", i)
						assert.InDelta(t, left.Actors[side].Lane, right.Actors[side].Lane, .001, "lane jump in beat %d", i)
					}
				}
			}
		})
	}
}

func TestReengagementAfterKnockbackAndEvasion(t *testing.T) {
	for _, kind := range []string{"hit", "critical", "yielding-block", "dodge", "critical-dodge"} {
		for side := range 2 {
			t.Run(kind+[]string{" ally", " enemy"}[side], func(t *testing.T) {
				story := prepareStory([]Beat{{Kind: "approach", Duration: 1}, {Kind: kind, Side: side, Duration: 1.3}, {Kind: "block", Side: 1 - side, Duration: 1.2}})
				require.Len(t, story.Beats, 4)
				require.Equal(t, "engage", story.Beats[2].Kind)
				before := story.Sample(story.Start(2))
				moving := story.Sample(story.Start(2) + story.Beats[2].Duration/2)
				assert.Equal(t, "walk", moving.Actors[1-side].Animation)
				assert.Equal(t, before.Actors[side].Position, moving.Actors[side].Position)
				assert.NotEqual(t, before.Actors[1-side].Position, moving.Actors[1-side].Position)
			})
		}
	}
}

func TestKnockbackRetainsPosition(t *testing.T) {
	for _, kind := range []string{"hit", "critical", "backhit"} {
		for side := range 2 {
			t.Run(kind+[]string{" ally attacks", " enemy attacks"}[side], func(t *testing.T) {
				story := Story{Beats: []Beat{{Kind: "approach", Duration: 1}, {Kind: kind, Side: side, Duration: 2.4}, {Kind: "talk", Duration: 1}, {Kind: "advance", Side: 1 - side, Duration: 1}}}
				def := 1 - side
				direction := 1.0
				if side == 1 {
					direction = -1
				}
				previous := at(story, kind, .55).Actors[def].Position
				for _, local := range []float64{.57, .7, .91, 1.5, 2.39} {
					state := at(story, kind, local)
					assert.GreaterOrEqual(t, (state.Actors[def].Position-previous)*direction, 0.0)
					previous = state.Actors[def].Position
				}
				waiting := at(story, "talk", .5)
				assert.InDelta(t, previous, waiting.Actors[def].Position, 1e-9)
				assert.Equal(t, "idle", waiting.Actors[def].Animation)
				returning := at(story, "advance", .5)
				assert.Equal(t, "walk", returning.Actors[def].Animation)
				assert.Less(t, (returning.Actors[def].Position-previous)*direction, 0.0)
			})
		}
	}
}
