package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipStopsAtIntroduction(t *testing.T) {
	g := battleController(t)
	g.scene = &scene{actors: map[string]*unitVis{}, paths: map[string][][2]float64{}, plays: map[string]string{}, emotes: map[string]*emote{},
		groups: [][]sceneStep{{{Wait: 1}}, {{Intro: "관우"}}, {{Wait: 1}}}}
	g.skipBeats()
	assert.NotNil(t, g.intro, "a skip plays up to the introduction")
	assert.Equal(t, 1, g.scene.gi)
	g.skipBeats()
	assert.Nil(t, g.intro, "the next click ends the introduction")
	assert.True(t, g.sceneBusy(), "the beats after it still wait")
}
