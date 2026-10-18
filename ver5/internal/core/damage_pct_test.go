package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDamagePctScalesBlows(t *testing.T) {
	e, hero, _ := arena(t)
	atk, def := e.unit(hero.ID), e.unit("boss")
	full := e.damage(atk, def, e.basic())
	for _, pct := range []int{30, 50, 70} {
		e.SetDamagePct(pct)
		got := e.damage(atk, def, e.basic())
		if want := full * float64(pct) / 100; got < want-1e-9 || got > want+1e-9 && want >= 1 {
			t.Fatalf("%d%%: damage %v, want %v of %v", pct, got, want, full)
		}
	}
	e.SetDamagePct(100)
	if got := e.damage(atk, def, e.basic()); got != full {
		t.Fatalf("100%%: damage %v, want %v", got, full)
	}
}

func TestXPPctScalesActionXP(t *testing.T) {
	e, hero, boss := arena(t)
	atk, def := e.unit(hero.ID), e.unit(boss.ID)
	full := e.actionXPAmount(atk, def, false, false)
	require.Greater(t, full, 10)
	for _, pct := range []int{60, 80, 100} {
		e.SetXPPct(pct)
		got := e.actionXPAmount(atk, def, false, false)
		want := int(math.Round(float64(full) * float64(pct) / 100))
		assert.InDelta(t, want, got, 1, "%d%%", pct)
	}
}
