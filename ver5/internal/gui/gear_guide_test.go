package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// smithyGame stands by chapter 2's smith in the street (the smithy is an open shed) with the smith's weapons still
// to come, at the c02-walk waypoint.
func smithyGame(t *testing.T) *Game {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	require.True(t, g.send(session.Request{Op: "jump", Seed: 1, Node: "c02-walk"}).OK)
	st := g.S.Engine.Snapshot()
	st.Learning = nil
	st.Flags["c02-gift"] = 1
	for area, a := range g.S.Engine.ExploreFile().Areas {
		if p := a.People["대장장이"]; p != nil {
			st.Spot = content.Spot{Area: area, At: [2]int{p.At[0], p.At[1] + 2}, Face: "NE"}
		}
	}
	st.Warehouse["qinglong_blade"], st.Warehouse["serpent_spear"] = 0, 0
	g.S.Engine, err = core.Restore(g.S.Data, st)
	require.NoError(t, err)
	require.True(t, g.request(session.Request{Op: "command", Command: core.Command{Kind: "talk", Target: "대장장이"}}).OK)
	for i := 0; !g.S.Engine.Walking() && i < 50; i++ {
		require.True(t, g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}}).OK, g.S.View().Node)
	}
	require.True(t, g.S.Engine.Walking())
	return g
}

func TestGearGuide(t *testing.T) {
	equip := func(g *Game, actor, item string) {
		require.True(t, g.request(session.Request{Op: "command", Command: core.Command{Kind: "equip", Actor: actor, Item: item}}).OK)
	}

	t.Run("the smith's weapons wait in the warehouse and the guide leads to them in turn", func(t *testing.T) {
		g := smithyGame(t)
		o := g.S.View()
		assert.Equal(t, 1, o.Warehouse["qinglong_blade"])
		assert.NotEqual(t, "qinglong_blade", g.officer(o, "guanyu").Equipment["weapon"], "nothing is put on by itself")
		piece, owner, slot, ok := g.gearStep()
		require.True(t, ok)
		assert.Equal(t, []string{"qinglong_blade", "guanyu", "weapon"}, []string{piece, owner, slot})
		equip(g, "guanyu", "qinglong_blade")
		piece, owner, _, ok = g.gearStep()
		require.True(t, ok)
		assert.Equal(t, []string{"serpent_spear", "zhangfei"}, []string{piece, owner})
		equip(g, "zhangfei", "serpent_spear")
		_, _, _, ok = g.gearStep()
		assert.False(t, ok, "both are worn")
		assert.Empty(t, g.gear.targets)
	})

	t.Run("길 안내 off hides it", func(t *testing.T) {
		g := smithyGame(t)
		g.S.Settings.NoGuide = true
		_, _, _, ok := g.gearStep()
		assert.False(t, ok)
		assert.NotEmpty(t, g.gear.targets, "it comes back with the setting")
	})

	t.Run("leaving the smithy ends it", func(t *testing.T) {
		g := smithyGame(t)
		st := g.S.Engine.Snapshot()
		st.Spot = g.S.Engine.ExploreFile().Arrival(g.S.Engine.ExploreFile().Start.Area, "")
		e, err := core.Restore(g.S.Data, st)
		require.NoError(t, err)
		g.S.Engine = e
		_, _, _, ok := g.gearStep()
		assert.False(t, ok)
		assert.Empty(t, g.gear.targets)
	})
}
