package autoplay

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestVariety(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	prepare := func(seed uint64) (*Variety, *core.Engine, []core.Command) {
		e, err := core.NewAt(d, 1, "c14-battle", "easy")
		require.NoError(t, err)
		for l := e.Observe().Learning; len(l) > 0; l = e.Observe().Learning {
			_, err := e.Apply(core.Command{Kind: "learn_close", Actor: l[0]})
			require.NoError(t, err)
		}
		v := NewVariety(d, nil, seed, 5)
		return v, e, v.PrepareCommands(e, 1)
	}
	v, e, cmds := prepare(7)
	require.NotEmpty(t, cmds)
	st := d.Stages[e.Observe().Stage]
	dep := e.Observe().Deployment
	t.Run("the lord and the required go, the rest up to the count", func(t *testing.T) {
		assert.LessOrEqual(t, len(dep), st.Max)
		for _, id := range st.Required {
			assert.Contains(t, dep, id)
		}
		assert.Contains(t, dep, "liubei")
	})
	t.Run("부관임명 before anything else, then goals", func(t *testing.T) {
		for _, c := range Officers(d, e) {
			if len(c.Learned) > 0 && slices.Contains(v.pool(c), "deputy_appoint") {
				assert.Equal(t, "deputy_appoint", c.Learned[0], c.ID)
			}
		}
	})
	t.Run("a piece is worn once before it is worn again", func(t *testing.T) {
		for item, n := range v.Equipped {
			assert.Equal(t, 1, n, item)
		}
	})
	t.Run("the same seed draws the same, another seed another", func(t *testing.T) {
		_, _, again := prepare(7)
		assert.Equal(t, cmds, again)
		_, _, other := prepare(8)
		assert.NotEqual(t, cmds, other)
	})
	t.Run("the last try prepares as the fallback", func(t *testing.T) {
		e, err := core.NewAt(d, 1, "c14-battle", "easy")
		require.NoError(t, err)
		calls := 0
		f := CampaignPlayer(d)
		f.P = 1
		v := NewVariety(d, f, 1, 5)
		v.PrepareCommands(e, 5)
		calls = len(v.Log)
		assert.Equal(t, 1, calls)
		assert.Contains(t, v.Log[0], "search")
	})
}
