package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestOpeningFacing(t *testing.T) {
	t.Run("allies look at the officer the victory names", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) {})
		assert.Equal(t, "SE", e.unit("hero").Facing, "the boss stands at (9,2), east of the cells")
		assert.Equal(t, "SE", e.unit("archer1").Facing, "the officers face one way")
	})
	t.Run("allies look at the area they must reach", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) {
			s.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: 0, Y: 7, W: 2, H: 1}}}
			s.Victory = []content.Goal{{If: []string{"at:hero:1"}, Text: "flee south"}}
		})
		assert.Equal(t, "SW", e.unit("hero").Facing)
	})
	t.Run("enemies look at the officers' cells", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) {})
		for _, u := range e.matching("enemy") {
			assert.Equal(t, "NW", u.Facing, u.ID)
		}
	})
	t.Run("a written facing stays", func(t *testing.T) {
		e := battleWith(t, func(s *content.Stage) { s.Enemies[0].Facing = "SW" })
		assert.Equal(t, "SW", e.unit("boss").Facing)
	})
}

func TestDeploymentKeepsTheCellOrder(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {})
	hero, archer := e.unit("hero"), e.unit("archer1")
	assert.Equal(t, [2]int{1, 3}, [2]int{hero.X, hero.Y}, "the first officer takes the first cell")
	assert.Equal(t, [2]int{1, 2}, [2]int{archer.X, archer.Y})
}

func TestCutsceneMovesEndFacing(t *testing.T) {
	e := battleWith(t, func(s *content.Stage) {
		s.Events = []content.BattleEvent{{ID: "step", Do: []content.Action{{Kind: "cutscene", Steps: [][]content.Step{{
			{Kind: "move", Unit: "hero", To: &content.Point{X: 1, Y: 2}, Facing: "SE"},
			{Kind: "move", Unit: "archer1", To: &content.Point{X: 0, Y: 2}},
		}}}}}}
	})
	require.True(t, fired(e, "step"))
	hero, archer := e.unit("hero"), e.unit("archer1")
	assert.Equal(t, [2]int{1, 2}, [2]int{hero.X, hero.Y}, "the archer leaves the cell in the same group, so it is free")
	assert.Equal(t, "SE", hero.Facing, "the facing the step names")
	assert.Equal(t, [2]int{0, 2}, [2]int{archer.X, archer.Y})
	assert.Equal(t, "NW", archer.Facing, "no facing named: the way the last step went")
}
