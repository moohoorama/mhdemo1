package core

import (
	"math"
	"slices"
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCounterDealsItsRuleCoefficientNotAFullBlow(t *testing.T) {
	e, a, b := arena(t)
	want := int(math.Floor(e.damageAt(b, a, e.basic(), e.counterBlow(a, b))))
	full := int(math.Floor(e.damageAt(b, a, e.basic(), ext{})))
	require.Greater(t, full, want, "the counter must be weaker than a plain blow for this to tell them apart")
	for range 200 {
		e.events = nil
		a.HP, b.HP = e.stats(a).MaxHP, e.stats(b).MaxHP
		got := e.attackOn(b, a, []*Unit{a}, e.basic(), map[string]float64{a.ID: 1}, e.counterBlow(a, b), true, false, false)[a.ID]
		if got == 0 || slices.ContainsFunc(e.events, func(ev Event) bool { return ev.Kind == "critical" || ev.Kind == "double" }) {
			continue
		}
		assert.Equal(t, want, got)
		return
	}
	t.Fatal("no plain landed counter in 200 tries")
}
