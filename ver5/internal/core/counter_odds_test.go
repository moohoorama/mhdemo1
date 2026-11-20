package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterOdds(t *testing.T) {
	e, u, v := arena(t)
	c := Command{Kind: "attack", Actor: "hero", Target: "boss"}
	hit, crit, double, ok := e.CounterOdds(c)
	assert.True(t, ok)
	assert.Equal(t, e.hitChance(v, u, e.basic()), hit)
	assert.Equal(t, e.critChance(v, u, e.basic()), crit)
	assert.Equal(t, e.doubleChance(v, u, e.basic()), double)
	v.Status["confusion"] = Status{Turns: 1}
	_, _, _, ok = e.CounterOdds(c)
	assert.False(t, ok)
}
