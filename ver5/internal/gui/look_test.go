package gui

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestSpriteFollowsTheClassAndShootsInItsOwnLook(t *testing.T) {
	g, _, _, _, _ := stoneGame(t)
	art := func(def, class, weapon string) string {
		a := g.battleClassArt(def, class, weapon, "", "dong")
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
		a := g.shootArt(def, "horse_archer", "", "dong")
		require.NotNil(t, a)
		return a.ID
	}
	assert.Equal(t, "lubu_bow", shoot("lubu"))
	assert.Equal(t, "horse_archer", shoot("zhaoyun"), "no own look for the class: the class's, never the spear sprite")
}

func TestShootingLookIsUsedBeyondTheMeleeReach(t *testing.T) {
	g, _, _, _, _ := stoneGame(t)
	attack := core.Command{Kind: "attack"}
	actor := &unitVis{id: "guanyu", u: 5, v: 5, shoot: g.shootArt("lubu", "horse_archer", "", "dong")}
	far := &unitVis{id: "x-target", u: 7, v: 5}
	assert.False(t, g.shootsFromRange(actor, far, attack, nil), "a rider without 기사 never shoots")
	st := g.S.Engine.Snapshot()
	for i := range st.Characters {
		if st.Characters[i].ID == "guanyu" {
			st.Characters[i].Learned = append(st.Characters[i].Learned, "horse_archery")
		}
	}
	e, err := core.Restore(g.S.Data, st)
	require.NoError(t, err)
	g.S.Engine = e
	for _, c := range []struct {
		u, v float64
		want bool
	}{{6, 5, false}, {5, 4, false}, {6, 6, true}, {4, 4, true}, {7, 5, true}, {5, 7, true}} {
		target := &unitVis{id: "x-target", u: c.u, v: c.v}
		assert.Equal(t, c.want, g.shootsFromRange(actor, target, attack, nil), "target at (%v,%v)", c.u, c.v)
	}
	knocked := []core.Event{{Kind: "knockback", Target: "x-target", Path: []content.Point{{X: 6, Y: 5}, {X: 7, Y: 5}}}}
	assert.False(t, g.shootsFromRange(actor, far, attack, knocked), "judged where the target stood before it was pushed")
	assert.False(t, g.shootsFromRange(actor, far, core.Command{Kind: "skill"}, nil), "skills keep the normal look")
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
	assert.Equal(t, "zhaoyun_light", first.art.ID)
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
	// a light cavalry officer without looks of his own for the classes (관평) takes the place of the first
	// one that has a line of them (an officer with named looks promotes only into those classes)
	snap := g.S.Engine.Snapshot()
	officer, rider := "", ""
	for id, c := range g.S.Data.Characters {
		if c.Playable && c.Class == "light_cavalry" && len(c.Look.Sprites) == 0 && (officer == "" || id < officer) {
			officer = id
		}
	}
	require.NotEmpty(t, officer, "no light cavalry officer without looks of his own")
	for j, c := range snap.Characters {
		if c.Class == "light_cavalry" && slices.Contains(snap.Deployment, c.ID) && rider == "" {
			rider = c.ID
			o := c
			o.ID, o.Def, o.Level = officer, officer, 15
			snap.Characters = append(snap.Characters, o)
			snap.Characters[j].Equipment = map[string]string{}
		}
	}
	require.NotEmpty(t, rider)
	for i, id := range snap.Deployment {
		if id == rider {
			snap.Deployment[i] = officer
		}
	}
	for i := range snap.Units {
		if snap.Units[i].ID == rider {
			snap.Units[i].ID, snap.Units[i].Character = officer, officer
		}
	}
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

func TestAnEquippedHorseOfItsOwnColourRecoloursTheRider(t *testing.T) {
	g, _, _, _, _ := stoneGame(t)
	plain := g.battleClassArt("guanyu", "light_cavalry", "", "", "shu")
	white := g.battleClassArt("guanyu", "light_cavalry", "", "baima", "shu")
	require.NotNil(t, plain)
	require.NotNil(t, white)
	assert.Equal(t, plain.ID, white.ID, "the same sheet")
	assert.NotSame(t, plain, white, "drawn in the horse's colours")
	assert.Same(t, plain, g.battleClassArt("guanyu", "light_cavalry", "", "war_horse", "shu"), "a horse without colours of its own changes nothing")

	st := g.S.Engine.Snapshot()
	for i := range st.Characters {
		if st.Characters[i].ID == "guanyu" {
			st.Characters[i].Equipment["accessory"] = "baima"
		}
	}
	e, err := core.Restore(g.S.Data, st)
	require.NoError(t, err)
	found := false
	for _, u := range e.Observe().UnitViews {
		if u.Character == "guanyu" {
			found = true
			assert.Equal(t, "baima", u.Mount, "a rider shows the horse it rides")
		}
	}
	assert.True(t, found, "Guan Yu is on the field")
}

// 하후돈 wears the eyepatch only once 소패 has been won.
func TestLaterLookFollowsTheStory(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s := core.New(d, 1).Snapshot()
	g := &Game{S: &session.Session{Data: d}}
	g.S.Engine, err = core.Restore(d, s)
	require.NoError(t, err)
	look := d.Characters["xiahoudun"].Look
	assert.Equal(t, "xiahoudun", g.laterLook(look).Portrait)

	s.Executed = map[string]bool{"won-B15": true}
	g.S.Engine, err = core.Restore(d, s)
	require.NoError(t, err)
	after := g.laterLook(look)
	assert.Equal(t, "xiahoudun_oneeye", after.Portrait)
	assert.Equal(t, "xiahoudun_oneeye_noncombat", after.Scene)
}
