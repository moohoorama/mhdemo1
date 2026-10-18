package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestSpriteFollowsTheClassAndShootsInItsOwnLook(t *testing.T) {
	g, _, _, _, _ := stoneGame(t)
	art := func(def, class, weapon string) string {
		a := g.battleClassArt(def, class, weapon, "dong")
		require.NotNil(t, a)
		return a.ID
	}
	assert.Equal(t, "lubu", art("lubu", "light_cavalry", "spear"))
	assert.Equal(t, "lubu_bow", art("lubu", "horse_archer", "spear"))
	assert.Equal(t, "lubu", art("lubu", "heavy_cavalry", "spear"), "a class with no sprite of its own takes the character's default")
	assert.Equal(t, "light_infantry", art("troop_footman", "light_infantry", "sword"))
	assert.Equal(t, "caocao", art("caocao", "light_cavalry", "spear"))
	assert.Equal(t, "caocao_sword", art("caocao", "light_cavalry", "sword"), "the sprite follows the equipped weapon")
	assert.Equal(t, "caocao_sword", art("caocao", "royal_guard", "sword"))
	assert.Equal(t, "zhaoyun", art("zhaoyun", "light_cavalry", ""))
	assert.Equal(t, "zhaoyun_sword", art("zhaoyun", "light_cavalry", "sword"))
	assert.Equal(t, "zhaoyun", art("zhaoyun", "light_cavalry", "bow"), "no entry for that weapon: the default")

	shoot := func(def string) string {
		a := g.shootArt(def, "horse_archer", "dong")
		require.NotNil(t, a)
		return a.ID
	}
	assert.Equal(t, "lubu_bow", shoot("lubu"))
	assert.Equal(t, "horse_archer", shoot("zhaoyun"), "no own look for the class: the class's, never the spear sprite")
}

func TestShootingLookIsUsedOnlyBeyondTheEightCells(t *testing.T) {
	g, _, _, _, _ := stoneGame(t)
	shoot := g.shootArt("lubu", "horse_archer", "dong")
	actor := &unitVis{id: "x-actor", u: 5, v: 5, shoot: shoot}
	attack := core.Command{Kind: "attack"}
	for _, c := range []struct {
		u, v float64
		want bool
	}{{6, 5, false}, {6, 6, false}, {4, 4, false}, {7, 5, true}, {6, 7, true}, {5, 8, true}} {
		target := &unitVis{id: "x-target", u: c.u, v: c.v}
		assert.Equal(t, c.want, g.shootsFromRange(actor, target, attack), "target at (%v,%v)", c.u, c.v)
	}
	far := &unitVis{id: "x-target", u: 8, v: 5}
	assert.False(t, g.shootsFromRange(&unitVis{id: "x-actor", u: 5, v: 5}, far, attack), "no shoot look")
	assert.False(t, g.shootsFromRange(actor, far, core.Command{Kind: "skill"}), "skills keep the normal look")
}
