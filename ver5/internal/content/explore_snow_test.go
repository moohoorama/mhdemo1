package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExploreAreaSnowStaysWithinZeroAndOne(t *testing.T) {
	d, err := Load("../testdata/mini")
	require.NoError(t, err)
	var hero string
	for id := range d.Characters {
		hero = id
		break
	}
	require.NotEmpty(t, hero)
	for snow, ok := range map[float64]bool{0: true, 0.5: true, 1: true, -0.1: false, 1.5: false} {
		d.Explores = map[string]*Explore{"t": {Hero: hero, Start: Spot{Area: "a", Face: "SW"},
			Areas: map[string]*ExploreArea{"a": {Map: "m", Name: "n", Snow: snow}}}}
		err := d.validateExplores(map[string]Node{})
		if ok {
			assert.NoError(t, err, "snow %v", snow)
		} else {
			assert.ErrorContains(t, err, "snow", "snow %v", snow)
		}
	}
}
