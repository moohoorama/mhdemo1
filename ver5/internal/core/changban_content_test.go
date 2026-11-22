package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

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
