package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestMountedShotDirectRangedAccuracy(t *testing.T) {
	e, _, target := arena(t)
	u := e.unit("archer1")
	u.X, u.Y = 4, 6
	e.data.Traits["mounted_test"] = content.Trait{Effects: map[string]float64{content.FxMountedShot: 15}}
	with(e, u.Character, "mounted_test")
	shot := e.basic()
	shot.Hit = 50
	u.Steps = 2
	base := e.hitChanceFor(u, target, shot, true)
	u.Steps = 3
	assert.InDelta(t, base+15, e.hitChanceFor(u, target, shot, true), .001)
	assert.InDelta(t, base, e.hitChanceFor(u, target, shot, false), .001, "counterattacks do not benefit")
	melee := e.unit("hero")
	melee.Steps = 3
	with(e, melee.Character, "mounted_test")
	assert.Equal(t, e.hitChance(melee, target, shot), e.hitChanceFor(melee, target, shot, true), "melee attacks do not benefit")
	e.state.Turn = "enemy"
	assert.Equal(t, e.hitChance(u, target, shot), e.hitChanceFor(u, target, shot, true), "the bonus requires its own turn")
	e.state.Turn = "ally"
	e.data.Traits["immune_test"] = content.Trait{Effects: map[string]float64{content.FxRangedImmune: 1}}
	with(e, target.Character, "immune_test")
	assert.Zero(t, e.hitChanceFor(u, target, shot, true), "immunity takes precedence")
}

func TestMountedShotPreviewMatchesDirectHitChance(t *testing.T) {
	e, _, target := arena(t)
	u := e.unit("archer1")
	u.X, u.Y, u.Steps = 4, 6, 3
	e.data.Rules.BasicAttack.Hit = 50
	e.data.Traits["mounted_test"] = content.Trait{Effects: map[string]float64{content.FxMountedShot: 15}}
	with(e, u.Character, "mounted_test")
	p, err := e.Preview(Command{Kind: "attack", Actor: u.ID, Target: target.ID})
	require.NoError(t, err)
	assert.Equal(t, e.hitChanceFor(u, target, e.basic(), true), p.Hit)
	assert.InDelta(t, e.hitChance(u, target, e.basic())+15, p.Hit, .001)
}
