package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// splitSortie is the mini deployment with two start groups: 본군 on (1,2) and (2,2), 별동대 on (1,3) led by archer1.
func splitSortie(t *testing.T) *Engine {
	t.Helper()
	e := deployment(t)
	st := &e.data.Stages[0]
	st.Allies = []content.Point{{X: 1, Y: 3}, {X: 1, Y: 2}, {X: 2, Y: 2}}
	st.Areas = []content.Area{
		{ID: 1, Rect: content.MapRect{X: 1, Y: 2, W: 2, H: 1}, Name: "north"},
		{ID: 2, Rect: content.MapRect{X: 1, Y: 3, W: 1, H: 1}, Name: "south"},
	}
	st.Required = []string{"archer1"}
	st.Groups = []content.AllyGroup{{Name: "본군", Area: 1}, {Name: "별동대", Area: 2, Required: []string{"archer1"}}}
	return e
}

func at(e *Engine, id string) content.Point {
	u := e.unit(id)
	return content.Point{X: u.X, Y: u.Y}
}

func TestSplitSortie(t *testing.T) {
	t.Run("a group's required officer stands only on its cells", func(t *testing.T) {
		e := splitSortie(t)
		assertRejected(t, e, Command{Kind: "place", Actor: "archer1", X: 1, Y: 2})
		apply(t, e, Command{Kind: "place", Actor: "archer1", X: 1, Y: 3})
		apply(t, e, Command{Kind: "place", Actor: "hero", X: 2, Y: 2})
		assertRejected(t, e, Command{Kind: "reposition", Actor: "archer1", X: 1, Y: 2})
		assertRejected(t, e, Command{Kind: "reposition", Actor: "hero", X: 1, Y: 3}) // the trade would take archer1 off its group
		for _, c := range e.LegalActions("archer1").Commands {
			assert.NotEqual(t, "reposition", c.Kind, "no cell to go to: the deploy screen lights none")
		}
	})
	t.Run("the others may stand in any group", func(t *testing.T) {
		e := splitSortie(t)
		e.data.Stages[0].Required = nil
		e.data.Stages[0].Groups[1].Required = nil
		apply(t, e, Command{Kind: "place", Actor: "hero", X: 1, Y: 3})
		apply(t, e, Command{Kind: "place", Actor: "archer1", X: 1, Y: 2})
	})
	t.Run("the default placement keeps the leader on its group past the cells' order", func(t *testing.T) {
		e := splitSortie(t)
		require.Equal(t, []string{"hero", "archer1"}, e.state.Deployment)
		finishDeployment(t, e)
		assert.Equal(t, content.Point{X: 1, Y: 3}, at(e, "archer1"))
		assert.Equal(t, content.Point{X: 1, Y: 2}, at(e, "hero"))
	})
}
