package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestStageChangbanRetreat(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, crossing := range []bool{false, true} {
		name := "enemy-elimination-is-not-victory"
		if crossing {
			name = "crossing-releases-the-bridge-guard"
		}
		t.Run(name, func(t *testing.T) {
			e, err := NewAt(d, 1, "c26-battle", "normal")
			require.NoError(t, err)
			for _, id := range e.Observe().Learning {
				applyOK(t, e, Command{Kind: "learn_close", Actor: id})
			}
			applyOK(t, e, Command{Kind: "start"})
			finishDeployment(t, e)

			assert.Nil(t, e.unit("liubei"))
			zhangfei := e.unit("zhangfei")
			require.NotNil(t, zhangfei)
			assert.Equal(t, content.Point{X: 17, Y: 27}, content.Point{X: zhangfei.X, Y: zhangfei.Y})
			assert.True(t, zhangfei.MoveLocked)
			clear(zhangfei.Status)
			assert.Zero(t, e.movement(zhangfei))
			assert.False(t, slices.ContainsFunc(e.LegalActions("zhangfei").Commands, func(c Command) bool { return c.Kind == "move" }))
			assert.Len(t, e.matching("remnants"), 6)
			for _, u := range e.matching("remnants") {
				assert.Equal(t, e.stats(u).MaxHP*66/100, u.HP)
			}
			for _, u := range e.matching("enemy") {
				u.HP = 0
			}
			e.settle()
			assert.Equal(t, "battle", e.state.Phase)
			if !crossing {
				return
			}

			zhaoyun := e.unit("zhaoyun")
			zhaoyun.X, zhaoyun.Y = 17, 31
			e.settle()
			assert.Equal(t, "victory", e.state.Result)
			assert.True(t, e.inArea(zhangfei, 70))
			assert.False(t, zhangfei.MoveLocked)
			assert.True(t, e.state.Executed["battle-B26-bridge-roar"])
			assert.NotNil(t, e.unit("xiahoudun"))
			assert.True(t, slices.ContainsFunc(e.events, func(ev Event) bool {
				return slices.ContainsFunc(ev.Cutscene, func(steps []CutStep) bool {
					return slices.ContainsFunc(steps, func(st CutStep) bool { return st.Kind == "dust" })
				})
			}))
		})
	}
}

func TestStageChangbanEvacuation(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := NewAt(d, 1, "c25v-battle", "easy")
	require.NoError(t, err)
	for _, id := range e.Observe().Learning {
		applyOK(t, e, Command{Kind: "learn_close", Actor: id})
	}
	applyOK(t, e, Command{Kind: "start"})
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, command := range plan {
		applyOK(t, e, command)
	}

	assert.Len(t, e.matching("enemy"), 20)
	e.turnStarted = "enemy"
	e.state.Round = 2
	e.battleEvents()
	assert.Len(t, e.matching("enemy"), 20)
	e.state.Round = 3
	e.battleEvents()
	assert.Len(t, e.matching("second-line"), 20)
	assert.Empty(t, e.matching("main-army"))
	e.state.Round = 5
	e.battleEvents()
	assert.Len(t, e.matching("main-army"), 40)

	stage := d.Stages[e.state.Stage]
	goal := stage.Victory[0].If
	satisfy(t, e, goal[:len(goal)-1])
	e.battleEvents()
	assert.False(t, e.holdsIn(goal, true))
	assert.Equal(t, byte('-'), e.Cell(17, 26).State)
	satisfy(t, e, goal)
	e.battleEvents()
	assert.True(t, e.holdsIn(goal, true))
	assert.Equal(t, byte('x'), e.Cell(17, 26).State)
}
