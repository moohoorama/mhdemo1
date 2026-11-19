package gui

import (
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
)

func TestPortraitSuppressed(t *testing.T) {
	g := &Game{}
	c := content.Character{Look: content.Look{Portrait: "someone", NoPortrait: true}}
	assert.Nil(t, g.portraitArt(c, ""))
	assert.Nil(t, g.storyPortrait(c))
}
