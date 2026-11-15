package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpellCosts(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)

	t.Run("spell costs stay in 12..84 and even", func(t *testing.T) {
		for id, s := range d.Skills {
			if s.Kind == KindPhysical || s.Effect == "meditate" {
				continue
			}
			assert.GreaterOrEqual(t, s.Cost, 12, id)
			assert.LessOrEqual(t, s.Cost, 84, id)
			assert.Zero(t, s.Cost%2, id)
		}
	})

	t.Run("대보급 heals twice 소보급, 대원군 as much as 소보급", func(t *testing.T) {
		minor := d.Skills["minor_supply"].Coeff
		assert.Equal(t, "대보급", d.Skills["medium_supply"].Name)
		assert.Equal(t, 2*minor, d.Skills["medium_supply"].Coeff)
		assert.Equal(t, minor, d.Skills["grand_reinforce"].Coeff)
		assert.Equal(t, d.Skills["medium_supply"].Coeff, d.Skills["transport"].Coeff)
		assert.NotContains(t, d.Skills, "grand_supply")
	})

	t.Run("분발 is 강진's attack twin", func(t *testing.T) {
		rally, fortify := d.Skills["rally"], d.Skills["fortify_line"]
		assert.Equal(t, "attack", rally.Effect)
		assert.Equal(t, fortify.Requires, rally.Requires)
		assert.Equal(t, fortify.Shape, rally.Shape)
		assert.Equal(t, fortify.Max, rally.Max)
		assert.Equal(t, fortify.Duration, rally.Duration)
		assert.Equal(t, fortify.Cost, rally.Cost)
	})
}
