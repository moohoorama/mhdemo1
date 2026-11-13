package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestAnEventChangesTheBattleGoals(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B07" {
			d.Campaign.Start = n.ID
		}
	}
	e := New(d, 1)
	apply(t, e, Command{Kind: "start"})
	finishDeployment(t, e)
	assert.Equal(t, []string{"alldefeated:enemy"}, e.Observe().Map.Victory[0].If, "the rear guard first")

	e.state.Round = 3
	e.battleEvents()
	assert.Equal(t, []string{"at:liubei:1"}, e.Observe().Map.Victory[0].If, "away to the east once Lu Bu comes")
	assert.Equal(t, d.Stages[e.state.Stage].Defeat, e.Observe().Map.Defeat, "a goals action without Defeat keeps them")
	marked := false
	for _, m := range e.GoalMarks() {
		marked = marked || m.Unit == "liubei" && m.Win
	}
	assert.True(t, marked, "the guide sends Liu Bei to the east end")

	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, e.Observe().Map.Victory, r.Observe().Map.Victory, "a save keeps the changed goals")

	u := e.unit("liubei")
	u.X, u.Y = 26, 5
	e.settle()
	assert.Equal(t, "victory", e.Observe().Result)
	e.clearBattle()
	assert.Nil(t, e.state.Goals, "the next battle starts from its own goals")
}

func TestBonusGoalMarksWhereUnitsFlee(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B04" {
			d.Campaign.Start = n.ID
		}
	}
	e := New(d, 1)
	apply(t, e, Command{Kind: "start"})
	corner := e.data.Stages[e.state.Stage].GoalCells(nil, 2)
	require.NotEmpty(t, corner)
	carts, areas := 0, 0
	for _, m := range e.SubMarks() {
		if m.Unit != "" && e.unit(m.Unit).Group == "carts" {
			carts++
		}
		if assert.ObjectsAreEqual(corner, m.Cells) {
			areas++
		}
	}
	assert.Equal(t, 2, carts, "each loot cart once")
	assert.Equal(t, 1, areas, "the corner the carts flee to, once")
}

func TestEveryStageRestoresOnceDeployed(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind != "battle" {
			continue
		}
		t.Run(n.Stage, func(t *testing.T) {
			d.Campaign.Start = n.ID
			e := New(d, 1)
			apply(t, e, Command{Kind: "start"})
			_, err := Restore(d, e.Snapshot())
			assert.NoError(t, err, "an NPC stepped back to a lower class carries only what that class can")
		})
	}
}

func TestASteppedBackNPCTakesTheShopWeaponOfItsClass(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B03" {
			d.Campaign.Start = n.ID
		}
	}
	e := New(d, 1)
	apply(t, e, Command{Kind: "start"})
	g := e.char("gaoshun")
	require.NotNil(t, g)
	assert.Equal(t, "light_infantry", g.Class)
	assert.Equal(t, "iron_sword", g.Equipment["weapon"], "철창 becomes 철검")
	assert.Equal(t, "leather_armor", g.Equipment["armor"])
}
