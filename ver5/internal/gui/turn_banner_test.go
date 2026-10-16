package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTurnBannerHoldsThePhase(t *testing.T) {
	g := &Game{speed: 1}
	g.showTurn(7, "enemy")
	assert.True(t, g.busy(), "the enemy waits while its banner plays")
	for i := 0; i < int(turnLength*60)+2; i++ {
		g.advanceTurnBanner(1.0 / 60)
	}
	assert.False(t, g.busy(), "and moves once the banner is done")
}
