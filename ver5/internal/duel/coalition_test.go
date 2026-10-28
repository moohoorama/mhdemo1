package duel

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCoalitionDuelStory(t *testing.T) {
	scripts, err := loadScripts()
	require.NoError(t, err)
	for _, id := range []string{"hulao-mushun", "hulao-fangyue"} {
		t.Run(id, func(t *testing.T) {
			story := scripts.Story(id, "defeat")
			for time := 0.; time < story.Duration(); time += .03 {
				state := story.Sample(time)
				assert.NotEqual(t, "hit", state.Actors[1].Animation)
				assert.NotEqual(t, "exhausted", state.Actors[1].Animation)
			}
		})
	}

	t.Run("Mushun is dodged repeatedly then cut down from behind", func(t *testing.T) {
		story := scripts.Story("hulao-mushun", "defeat")
		dodges := 0
		for _, b := range story.Beats {
			if strikeKind(b.Kind) == "dodge" {
				dodges++
				assert.Zero(t, b.Side)
			}
		}
		assert.GreaterOrEqual(t, dodges, 4)

		before := at(story, "slash-through", .3)
		flash := at(story, "slash-through", .5)
		after := at(story, "slash-through", 1.0)
		assert.Zero(t, before.Flash)
		assert.Greater(t, flash.Flash, .9)
		assert.Less(t, after.Actors[1].Position, after.Actors[0].Position, "Lubu ends behind Mushun")
		assert.Equal(t, "hit", after.Actors[0].Animation)
		assert.NotEqual(t, "SE", after.Actors[1].Facing, "Lubu turns away after the cut")

		start := story.Start(len(story.Beats) - 1)
		early := story.Sample(start + .05)
		last := story.Sample(story.Duration())
		assert.Equal(t, "exhausted", last.Actors[0].Animation)
		assert.Less(t, early.Actors[0].Gray, last.Actors[0].Gray)
		assert.Equal(t, 1.0, last.Actors[0].Gray)
		assert.Zero(t, last.Actors[1].Gray)
	})

	t.Run("Fangyue is cut down at the end", func(t *testing.T) {
		story := scripts.Story("hulao-fangyue", "defeat")
		last := story.Sample(story.Duration())
		assert.Equal(t, 1.0, last.Actors[0].Gray)
		assert.Zero(t, last.Actors[1].Gray)
	})
}
