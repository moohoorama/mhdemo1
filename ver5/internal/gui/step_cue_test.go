package gui

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// A boat glides, so the boat sounds instead of a footstep between water cells; the bank's steps and a half step onto or off the water
// still do, and a rider's are hoof beats.
func TestStepCueIsSilentAfloat(t *testing.T) {
	f := &Field{Map: &MapArt{W: 3, H: 1, Tiles: []string{"g~~"}, SunkWater: true}}
	foot, rider := &unitVis{u: 0, v: 0}, &unitVis{u: 0, v: 0, mounted: true}
	assert.Equal(t, "step", f.stepCue(foot, content.Point{X: 1}), "bank to water: the land half is walked")
	assert.Equal(t, "hoof", f.stepCue(rider, content.Point{X: 1}))
	foot.u, rider.u = 1, 1
	assert.Equal(t, "boat", f.stepCue(foot, content.Point{X: 2}), "water to water: the boat glides")
	assert.Equal(t, "boat", f.stepCue(rider, content.Point{X: 2}))
	assert.Equal(t, "step", f.stepCue(foot, content.Point{X: 0}), "water to bank")
	foot.u = 0
	f.Map.SunkWater = false
	foot.u = 1
	assert.Equal(t, "step", f.stepCue(foot, content.Point{X: 2}), "without sunk water there are no boats")
}

// The boat's cue decodes and runs a little over a step (stepTime) so that each one's fade-out meets the next's fade-in.
func TestBoatCueLoads(t *testing.T) {
	f, err := os.Open("../../assets/sounds/boat.ogg")
	require.NoError(t, err)
	defer f.Close()
	pcm, err := decodeAll(vorbis.DecodeWithSampleRate(sampleRate, f))
	require.NoError(t, err)
	seconds := float64(len(pcm)/4) / sampleRate
	assert.Greater(t, seconds, stepTime)
	assert.Less(t, seconds, 2*stepTime)
}
