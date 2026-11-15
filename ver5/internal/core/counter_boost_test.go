package core

import (
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
)

func TestCounterBoostAddsHalfABlowToEveryCounter(t *testing.T) {
	e, a, _ := arena(t)
	R := e.data.Rules
	e.data.Traits["counter_boost"] = content.Trait{Effects: map[string]float64{content.FxCounterBoost: .5}}
	with(e, "hero", "counter_boost")
	assert.InDelta(t, R.PlainCounterShare+.5, e.counterShare(a), 1e-9, "a plain counter")
	a.Status["counter"] = Status{Turns: 1}
	assert.InDelta(t, R.CounterShare+.5, e.counterShare(a), 1e-9, "in the 반격술 stance")
}

func TestCounterTellsTheCellsItsPierceReaches(t *testing.T) {
	e, a, b := arena(t)
	e.char("hero").Class = "rider"
	e.char("hero").Equipment = map[string]string{"weapon": "spear"}
	e.events = nil
	e.counterBack(b, a)
	var area []content.Point
	for _, ev := range e.events {
		if ev.Kind == "area" && ev.Actor == a.ID {
			area = ev.Path
		}
	}
	assert.Equal(t, []content.Point{{X: b.X + 1, Y: b.Y}}, area, "the empty cell behind the attacker")
}
