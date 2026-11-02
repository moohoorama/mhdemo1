package core

import (
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
)

func TestGrassFootingRaisesTheFactorOnlyOnBareGrass(t *testing.T) {
	e, a, _ := arena(t)
	factors := func() map[string]float64 {
		out := map[string]float64{}
		for _, cell := range []string{"g--", "gt-", "gv-", "l--"} {
			setCell(e, a.X, a.Y, cell)
			out[cell] = e.terrain(a, a.X, a.Y).Factor
		}
		return out
	}
	before := factors()
	e.data.Traits["grass_footing"] = content.Trait{Effects: map[string]float64{content.FxGrassFactor: .05}}
	with(e, "hero", "grass_footing")
	after := factors()
	assert.InDelta(t, before["g--"]+.05, after["g--"], 1e-9, "bare grass")
	for _, cell := range []string{"gt-", "gv-", "l--"} {
		assert.Equal(t, before[cell], after[cell], cell)
	}
}
