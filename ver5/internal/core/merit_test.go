package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeritKeepsTheBattlesExperienceAndSpoils(t *testing.T) {
	e, a, b := arena(t)
	from := e.char("hero").Level
	require.NotNil(t, e.state.Merit)
	assert.Equal(t, from, e.state.Merit.From["hero"])
	b.HP = 1
	apply(t, e, Command{Kind: "attack", Actor: a.ID, Target: "boss"})
	saved, err := Restore(e.data, e.Snapshot()) // a game saved and loaded in the middle keeps it
	require.NoError(t, err)
	e = saved
	assert.Positive(t, e.state.Merit.XP["hero"])
	if e.state.Result != "victory" {
		e.unit("boss").HP = 0
		apply(t, e, Command{Kind: "wait", Actor: "hero"})
	}
	require.Equal(t, "victory", e.state.Result)
	assert.Equal(t, e.data.Stages[e.state.Stage].Reward, e.state.Merit.Spoils)
}
