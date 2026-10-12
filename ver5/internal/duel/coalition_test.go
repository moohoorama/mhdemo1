package duel

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCoalitionDuelStory(t *testing.T) {
	scripts, err := LoadScripts("../../assets/duels.json")
	require.NoError(t, err)
	for id, count := range map[string]int{"hulao-fangyue": 2, "hulao-mushun": 1, "hulao-wuanguo": 3} {
		t.Run(id, func(t *testing.T) {
			story := scripts.Story(id, "defeat")
			exchanges := 0
			for _, b := range story.Beats {
				switch b.Kind {
				case "block", "critical", "yielding-block":
					exchanges++
				}
			}
			assert.Equal(t, count, exchanges)
			for time := 0.0; time < story.Duration(); time += .03 {
				state := story.Sample(time)
				assert.NotEqual(t, "hit", state.Actors[1].Animation)
				assert.NotEqual(t, "exhausted", state.Actors[1].Animation)
			}
			if id == "hulao-wuanguo" {
				start := at(story, "retreat", 0)
				for time := 0.0; time < 3; time += .05 {
					state := at(story, "retreat", time)
					assert.Equal(t, start.Actors[1].Position, state.Actors[1].Position)
					assert.Zero(t, state.Actors[0].Lane)
				}
				final := story.Sample(story.Duration())
				assert.Equal(t, -280.0, final.Actors[0].Position)
				x, y := final.Actors[0].ScreenPosition()
				assert.Greater(t, x, 375.0)
				assert.Greater(t, .5*x+y, 285.0+160)
				assert.NotEqual(t, "exhausted", final.Actors[0].Animation)
			}
		})
	}
}
