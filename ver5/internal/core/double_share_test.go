package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecondStrikeDealsDoubleShare(t *testing.T) {
	e, hero, boss := arena(t)
	e.data.Stages[e.state.Stage].Duels = nil
	with(e, hero.ID, "twice")
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	e.data.Rules.Crit.Base, e.data.Rules.Crit.Max = 0, 0
	boss.HP = 100000
	full := e.damage(hero, boss, e.basic())
	e.events = nil
	require.NoError(t, e.useSkill(hero, Command{Kind: "attack", Actor: hero.ID, Target: boss.ID}))
	hits := []int{}
	for _, ev := range e.events {
		if ev.Kind == "damage" && ev.Actor == hero.ID && ev.Target == boss.ID {
			hits = append(hits, ev.Amount)
		}
	}
	require.Len(t, hits, 2)
	assert.Equal(t, int(math.Floor(full)), hits[0])
	assert.Equal(t, int(math.Floor(full*e.data.Rules.DoubleShare)), hits[1])
}
