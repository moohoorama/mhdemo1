package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/content"
)

func TestLevelFactor(t *testing.T) {
	ratio := content.Rules{StatLevelBase: 10, ExpRatioPower: 4, ExpMinFactor: 0.25, ExpMaxFactor: 3}
	t.Run("the strength ratio means the same at any level", func(t *testing.T) {
		assert.InDelta(t, 1.853, levelFactor(ratio, 50, 60), 0.001, "70/60, 17% stronger")
		assert.InDelta(t, 1.464, levelFactor(ratio, 10, 12), 0.001, "22/20, 10% stronger")
		assert.InDelta(t, 1, levelFactor(ratio, 30, 30), 0.001)
	})
	t.Run("clamped at both ends", func(t *testing.T) {
		assert.Equal(t, 3.0, levelFactor(ratio, 10, 20), "30/20 to the fourth is 5.06")
		assert.Equal(t, 0.25, levelFactor(ratio, 30, 10))
	})
	t.Run("without a power, the old curve of the level difference", func(t *testing.T) {
		old := content.Rules{StatLevelBase: 10, ExpLevelStep: 0.12, ExpGrowth: 1.035, ExpMinFactor: 0.25, ExpMaxFactor: 2}
		assert.InDelta(t, 2.821, levelFactor(old, 50, 60), 0.001)
		assert.InDelta(t, 2.821, levelFactor(old, 10, 20), 0.001, "ten levels apart is the same wherever")
	})
}
