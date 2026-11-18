package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/content"
)

// A boat glides, so no footstep sounds between water cells; the bank's steps and a half step onto or off the water
// still do, and a rider's are hoof beats.
func TestStepCueIsSilentAfloat(t *testing.T) {
	f := &Field{Map: &MapArt{W: 3, H: 1, Tiles: []string{"g~~"}, SunkWater: true}}
	foot, rider := &unitVis{u: 0, v: 0}, &unitVis{u: 0, v: 0, mounted: true}
	assert.Equal(t, "step", f.stepCue(foot, content.Point{X: 1}), "bank to water: the land half is walked")
	assert.Equal(t, "hoof", f.stepCue(rider, content.Point{X: 1}))
	foot.u, rider.u = 1, 1
	assert.Empty(t, f.stepCue(foot, content.Point{X: 2}), "water to water: the boat glides")
	assert.Empty(t, f.stepCue(rider, content.Point{X: 2}))
	assert.Equal(t, "step", f.stepCue(foot, content.Point{X: 0}), "water to bank")
	foot.u = 0
	f.Map.SunkWater = false
	foot.u = 1
	assert.Equal(t, "step", f.stepCue(foot, content.Point{X: 2}), "without sunk water there are no boats")
}
