package autoplay

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMLPLearnsASmoothFunction(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	var X, Y [][]float64
	for range 2000 {
		a, b := rng.Float64()*4-2, rng.Float64()*4-2
		X = append(X, []float64{a, b})
		Y = append(Y, []float64{math.Sin(a) + b*b})
	}
	m := NewMLP([]int{2, 16, 16, 1}, 3)
	groups := make([]int, len(X))
	for i := range groups {
		groups[i] = i
	}
	m.Fit(X, Y, groups, 60, 32, .01, 4)
	assert.Less(t, m.ValLoss, .05)
	assert.InDelta(t, math.Sin(1)+.25, m.Predict([]float64{1, .5})[0], .25)
}
