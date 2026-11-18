package core

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// walkStory pages through lines from where e stands, picking option at a choice that offers it (the first option
// elsewhere), walking the explore maps and winning every battle (winStage), until the campaign ends; it returns the
// nodes it stopped at.
func walkStory(t *testing.T, e *Engine, option string) []string {
	var seen []string
	for step := 0; step < 2000; step++ {
		for len(e.state.Learning) > 0 {
			applyOK(t, e, Command{Kind: "learn_close", Actor: e.state.Learning[0]})
		}
		o := e.Observe()
		switch {
		case o.Phase == "complete":
			return seen
		case o.Dialogue.Kind == "choice":
			seen = append(seen, o.Node)
			pick := option
			if _, ok := o.Dialogue.Choices[pick]; !ok {
				options := slices.Sorted(maps.Keys(o.Dialogue.Choices))
				pick = options[0]
			}
			applyOK(t, e, Command{Kind: "choose", Option: pick})
		case e.Walking():
			plan := e.ExploreNext()
			require.NotEmpty(t, plan, "the walk at %s is stuck", o.Node)
			for _, c := range plan {
				applyOK(t, e, c)
			}
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			seen = append(seen, o.Node)
			winStage(t, e, false)
			applyOK(t, e, Command{Kind: "continue"})
		case o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate":
			seen = append(seen, o.Node)
			applyOK(t, e, Command{Kind: "next"})
		default:
			t.Fatalf("stopped at %s (%s, %s)", o.Node, o.Phase, o.Dialogue.Kind)
		}
	}
	t.Fatal("the story did not end")
	return nil
}

func TestLongzhongSettlesTheMode(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	start := func(t *testing.T, all bool) *Engine {
		e, err := NewAtWith(d, 1, "c24-start", "normal", Start{Flags: []string{"flag-gaoshun", "flag-zhugexuan"}})
		require.NoError(t, err)
		require.NotNil(t, e.char("gaoshun"))
		e.state.Flags["flag-guanyu-mifang"] = btoi(all) // the replayed chapter 22 sets it while Gao Shun is in the party
		return e
	}
	t.Run("short of a condition, the romance: no choice, ends at chapter 27", func(t *testing.T) {
		e := start(t, false)
		seen := walkStory(t, e, "")
		assert.Contains(t, seen, "c24-n09")
		assert.NotContains(t, seen, "c24-path")
		assert.Contains(t, seen, "c27-card")
		assert.NotContains(t, seen, "c25v-s3")
		assert.Equal(t, 1, e.state.Flags["flag-mode-yanyi"])
		assert.Zero(t, e.state.Flags["flag-mode-virtual"])
		require.NotNil(t, e.char("zhugeliang"))
		assert.Equal(t, "white_feather_fan", e.char("zhugeliang").Equipment["weapon"])
		assert.Nil(t, e.char("xushu"), "Xu Shu is taken after Changban")
	})
	t.Run("all three and the fast road, the romance", func(t *testing.T) {
		e := start(t, true)
		seen := walkStory(t, e, "빠른 길")
		assert.Contains(t, seen, "c24-path")
		assert.Contains(t, seen, "c27-card")
		assert.NotContains(t, seen, "c25v-s3")
		assert.Equal(t, 1, e.state.Flags["flag-mode-yanyi"])
	})
	t.Run("all three and the slow road, the other telling from chapter 25", func(t *testing.T) {
		e := start(t, true)
		seen := walkStory(t, e, "느린 길")
		assert.Contains(t, seen, "c25-b08", "the two common scenes of chapter 25 come first")
		assert.NotContains(t, seen, "c25-s3")
		assert.NotContains(t, seen, "c27-card")
		assert.Contains(t, seen, "c27v-card")
		assert.Equal(t, 1, e.state.Flags["flag-mode-virtual"])
		assert.NotNil(t, e.char("xushu"))
		assert.NotNil(t, e.char("weiyan"))
	})
}
