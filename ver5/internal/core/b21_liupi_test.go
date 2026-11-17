package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// 유벽 makes for the south castle and leaves the field there, so that the castle cell is free for 유비: his getting
// away is the bonus objective, and he does not die on the way out.
func TestYunanLiupiLeavesByTheCastle(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := NewAt(d, 1, "c21-battle", "")
	require.NoError(t, err)
	e.state.Learning = nil
	_, err = e.Apply(Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	require.Equal(t, "battle", e.state.Phase)
	u := e.unit("liupi")
	require.NotNil(t, u)
	require.True(t, content.Leaves(u.Policy), "유벽 flees, policy %q", u.Policy)
	st := d.Stages[e.state.Stage]
	castle := st.GoalCells(nil, 2)
	require.NotEmpty(t, castle)
	u.X, u.Y = castle[0].X, castle[0].Y
	e.unchecked, e.state.Learning = true, nil
	r, err := e.Apply(Command{Kind: "end"})
	require.NoError(t, err)
	assert.True(t, e.holdsAll([]string{"fled:liupi"}), "he got away")
	assert.Nil(t, e.occupied(castle[0].X, castle[0].Y), "the castle cell is free")
	for _, ev := range r.Events {
		assert.False(t, ev.Kind == "say" && ev.Actor == "liupi", "no dying words: %+v", ev)
	}
	sub := st.Sub[slices.IndexFunc(st.Sub, func(o content.Objective) bool { return o.ID == "liupi" })]
	assert.Equal(t, []string{"fled:liupi"}, sub.If)
}
