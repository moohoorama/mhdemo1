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
	assert.Equal(t, "zhaoyun_light", art("zhaoyun", "light_cavalry", ""))
	assert.Equal(t, "zhaoyun", art("zhaoyun", "heavy_cavalry", ""), "each class of his line its own sheet")
	assert.Equal(t, "zhaoyun_sword", art("zhaoyun", "light_cavalry", "sword"))
	assert.Equal(t, "zhaoyun_light", art("zhaoyun", "light_cavalry", "bow"), "no entry for that weapon: the class's")
	assert.Equal(t, "liubei_guards", art("liubei", "guards", "sword"))
	assert.Equal(t, "sunjian", art("sunjian", "light_cavalry", "spear"))
	assert.Equal(t, "heavy_cavalry", art("sunjian", "heavy_cavalry", "spear"), "drawn once: promoted, the class's own")

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

func TestBattleUnitTakesItsNewLookWhenItsWeaponOrClassChanges(t *testing.T) {
	g, _, _, _, o := stoneGame(t)
	var u core.UnitView
	for _, v := range o.UnitViews {
		if v.Faction == "ally" {
			u = v
			break
		}
	}
	u.ID = "x-look"
	shown := func(def, class, weapon string) *unitVis {
		u.Def, u.Class, u.Weapon = def, class, weapon
		g.syncUnit(u, o)
		return g.vis[u.ID]
	}
	first := shown("zhaoyun", "light_cavalry", "spear")
	assert.Equal(t, "zhaoyun", first.art.ID)
	first.dir = "NE"
	sword := shown("zhaoyun", "light_cavalry", "sword")
	assert.Same(t, first, sword, "the same unit, only its picture changes")
	assert.Equal(t, "zhaoyun_sword", sword.art.ID, "a sword equipped before the battle shows")
	assert.Equal(t, "NE", sword.dir)

	u.ID = "x-promote"
	assert.Equal(t, "archer", shown("jianyong", "archer", "bow").art.ID)
	assert.Equal(t, "crossbow", shown("jianyong", "crossbow", "bow").art.ID, "promoted mid-battle: the new class's sprite")
}

// One 인수 per tier: on a class with two ways up it offers both, the officer's pick decides, and the picker shows each
// class with the look and stats it would bring.
func TestInsuOffersEachClassAndThePickDecides(t *testing.T) {
	g, _, _, _, _ := stoneGame(t)
	snap := g.S.Engine.Snapshot()
	officer := ""
	for _, u := range snap.Units {
		for j, c := range snap.Characters {
			if u.Faction == "ally" && c.ID == u.Character && c.Class == "light_cavalry" && officer == "" {
				officer = c.ID
				snap.Characters[j].Level = 15
			}
		}
	}
	require.NotEmpty(t, officer)
	snap.Inventory["insu_mid"], snap.Inventory["insu_high"] = 1, 1
	e, err := core.Restore(g.S.Data, snap)
	require.NoError(t, err)
	g.S.Engine = e
	var picks []core.Command
	for _, c := range e.LegalActions(officer).Commands {
		if c.Kind == "item" && (c.Item == "insu_mid" || c.Item == "insu_high") {
			picks = append(picks, c)
		}
	}
	require.Len(t, picks, 2, "the 중급 인수 once per class, the 상급 not at all")
	assert.ElementsMatch(t, []string{"heavy_cavalry", "horse_archer"}, []string{picks[0].Option, picks[1].Option})

	g.ord = order{actor: officer}
	branches := g.branches(picks)
	require.Len(t, branches, 2)
	for _, b := range branches {
		assert.Equal(t, "light_cavalry", b.from)
		assert.Equal(t, b.cmd.Option, b.to)
		assert.NotNil(t, b.new)
	}

	_, err = e.Apply(core.Command{Kind: "item", Actor: officer, Target: officer, Item: "insu_mid"})
	assert.Error(t, err, "two ways up: the class must be named")
	_, err = e.Apply(core.Command{Kind: "item", Actor: officer, Target: officer, Item: "insu_mid", Option: "horse_archer"})
	require.NoError(t, err)
	assert.Equal(t, "horse_archer", g.unitView(e.Observe(), officer).Class)
}
