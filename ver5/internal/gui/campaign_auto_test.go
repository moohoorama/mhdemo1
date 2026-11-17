package gui

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/autoplay"
	"srpg/internal/core"
)

func TestCampaignAutoDiscardsAStalePlan(t *testing.T) {
	if !DebugBuild {
		t.Skip("debug controller")
	}
	for _, changedEngine := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed revision", true: "replaced engine"}[changedEngine], func(t *testing.T) {
			g := battleController(t)
			g.Debug = true
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ch := make(chan campaignPlan, 1)
			plan := campaignPlan{engine: g.S.Engine, revision: g.S.View().Revision, commands: []core.Command{{Kind: "end"}}}
			if changedEngine {
				plan.engine, _ = core.Restore(g.S.Data, g.S.Engine.Snapshot())
			} else {
				plan.revision--
			}
			ch <- plan
			g.campaign = &campaignAuto{cancel: cancel, pending: ch}
			before := g.S.Engine.Snapshot()
			g.updateCampaignAuto()
			assert.Equal(t, before, g.S.Engine.Snapshot())
			assert.Empty(t, g.campaign.queue)
			g.stopCampaignAuto()
			assert.ErrorIs(t, ctx.Err(), context.Canceled)
		})
	}
}

func TestCampaignRetryRestoresOnlyTheActualCheckpoint(t *testing.T) {
	g := battleController(t)
	g.player.g = g
	snap := g.S.Engine.Snapshot()
	require.NotNil(t, snap.Checkpoint)
	checkpoint := *snap.Checkpoint
	for i := range snap.Units {
		if snap.Units[i].ID == "hero" {
			snap.Units[i].HP = 0
		}
	}
	snap.Phase, snap.Result = "result", "defeat"
	var err error
	g.S.Engine, err = core.Restore(g.S.Data, snap)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g.campaign = &campaignAuto{cancel: cancel, checkpoint: &checkpoint, try: 1, node: checkpoint.Node}
	g.retryCampaign()
	require.NotNil(t, g.campaign)
	assert.NoError(t, ctx.Err())
	assert.Equal(t, 2, g.campaign.try)
	got := g.S.Engine.Snapshot()
	assert.Equal(t, checkpoint.RNG+uint64(0x9e3779b97f4a7c15), got.RNG)
	assert.Equal(t, checkpoint.Characters, got.Characters)
	assert.Equal(t, checkpoint.Inventory, got.Inventory)
	assert.Equal(t, checkpoint.Warehouse, got.Warehouse)
	assert.Equal(t, checkpoint.Money, got.Money)
	assert.Equal(t, checkpoint.Node, got.Node)
	g.campaign.try = 5
	g.retryCampaign()
	assert.Nil(t, g.campaign)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	assert.Equal(t, got, g.S.Engine.Snapshot())
}

func TestCampaignButtonIsOnlyForDebug(t *testing.T) {
	g := &Game{}
	assert.Nil(t, g.campaignSlot())
	assert.Nil(t, g.campaignPolicySlot())
	g.Debug = true
	assert.Equal(t, DebugBuild, g.campaignSlot() != nil)
	assert.Equal(t, DebugBuild, g.campaignPolicySlot() != nil)
}

func TestCampaignBattleUsesTheNeuralPolicy(t *testing.T) {
	g := battleController(t)
	p := autoplay.CampaignPlayer(g.S.Data)
	p.Context = context.Background()
	m := autoplay.NewBattleModel(8, 17)
	g.campaign = &campaignAuto{player: p, model: m}
	before := g.S.Engine.Snapshot()
	want := m.Select(g.S.Engine)
	g.planCampaign("act", "hero")
	select {
	case plan := <-g.campaign.pending:
		require.NoError(t, plan.err)
		assert.Equal(t, []core.Command{want}, plan.commands)
		assert.Equal(t, before, g.S.Engine.Snapshot())
	case <-time.After(5 * time.Second):
		t.Fatal("neural battle plan timed out")
	}
}

func TestCampaignPolicyCanOnlyChangeWhileStopped(t *testing.T) {
	if !DebugBuild {
		t.Skip("debug controller")
	}
	g := &Game{Debug: true}
	s := g.campaignPolicySlot()
	assert.Contains(t, s.label, "학습")
	s.click()
	assert.True(t, g.campaignRule)
	assert.Contains(t, g.campaignPolicySlot().label, "탐색")
	g.campaign = &campaignAuto{}
	assert.Nil(t, g.campaignPolicySlot().click)
}

func TestCampaignCommandsDoNotWaitForWallClockTime(t *testing.T) {
	if !DebugBuild {
		t.Skip("debug controller")
	}
	g := battleController(t)
	g.player.g = g
	g.Debug = true
	g.campaign = &campaignAuto{cancel: func() {}}
	g.queueCampaign([]core.Command{{Kind: "wait", Actor: "hero"}, {Kind: "wait", Actor: "archer1"}})
	before := g.S.View().Revision
	g.updateCampaignAuto() // show the command's selection for one frame
	g.updateCampaignAuto()
	require.Greater(t, g.S.View().Revision, before)
	require.True(t, g.player.Busy())
	require.Len(t, g.campaign.queue, 1)
	after := g.S.Engine.Snapshot()
	g.updateCampaignAuto()
	assert.Equal(t, after, g.S.Engine.Snapshot(), "the next command still waits for its animation")
	g.player.Skip()
	g.updateCampaignAuto()
	g.updateCampaignAuto()
	assert.Empty(t, g.campaign.queue, "the next command runs without advancing the wall clock")
	assert.Equal(t, 0.0, g.uiSeconds)
	assert.Greater(t, g.S.View().Revision, after.Revision)
}
