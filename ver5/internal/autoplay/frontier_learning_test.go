package autoplay

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestStagePreparationPreservesOtherStages(t *testing.T) {
	d, e := battleModelFixture(t)
	m := NewBattleModel(8, 1)
	m.ContentHash = d.Hash
	m.EnableCampaignLearning(8, 2)
	stage := d.Stages[0].ID
	want := m.Preparation.Actor.W[0][0]
	require.NoError(t, m.EnableStagePreparation(stage))
	expert := m.StageModels[stage].Preparation
	expert.Actor.W[0][0]++
	assert.Equal(t, want, m.Preparation.Actor.W[0][0])
	p := &NeuralPlayer{Searcher: CampaignPlayer(d), Model: m}
	selected, key := p.preparationModel(e)
	assert.Same(t, expert, selected)
	assert.Equal(t, "prep:"+stage, key)
	assert.Contains(t, NewCampaignTrainer(m, 3).Models, key)
	path := filepath.Join(t.TempDir(), "stage-prep.json")
	require.NoError(t, m.Save(path))
	loaded, err := LoadBattleModel(path)
	require.NoError(t, err)
	assert.Equal(t, expert.Actor.W, loaded.StageModels[stage].Preparation.Actor.W)
	delete(m.StageModels, stage)
	selected, key = p.preparationModel(e)
	assert.Same(t, m.Preparation, selected)
	assert.Equal(t, "prep", key)
}

func TestStageExpertOwnsIndependentWeights(t *testing.T) {
	m := NewBattleModel(8, 1)
	m.EnableCampaignLearning(8, 2)
	want := m.Actor.W[0][0]
	require.NoError(t, m.AddStageExpert("next"))
	require.NoError(t, m.Validate())
	expert := m.StageModels["next"]
	require.NotNil(t, expert)
	assert.Nil(t, expert.Preparation)
	assert.Nil(t, expert.StageModels)
	expert.Actor.W[0][0]++
	assert.Equal(t, want, m.Actor.W[0][0])
	require.NoError(t, m.AddStageExpert("next"))
	assert.Same(t, expert, m.StageModels["next"])
}

func TestCollectedFrontierRetainsRealUnpreparedState(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Campaign.Start = "prep1"
	d.Stages[0].TurnLimit = 10
	d.Rules.Difficulties = append(d.Rules.Difficulties, content.Difficulty{ID: "hard", XPPct: 100})
	m := NewBattleModel(8, 3)
	m.ContentHash = d.Hash
	stage := d.Stages[0].ID
	start, report, err := CollectFrontier(d, m, 7, stage)
	require.NoError(t, err)
	require.NotNil(t, start)
	assert.Equal(t, 0, start.Prefix)
	assert.Equal(t, uint64(7), start.Seed)
	assert.NotEmpty(t, report.Stages)
	e, err := core.Restore(d, start.State)
	require.NoError(t, err)
	assert.Equal(t, "scenario", e.Observe().Phase)
	assert.Equal(t, stage, d.Stages[e.Observe().Stage].ID)
	before := e.Snapshot()
	_, err = PlayFrontier(d, m, *start, 9, FrontierOptions{})
	require.NoError(t, err)
	assert.Equal(t, before, start.State)
	teacher, err := PlayFrontier(d, m, *start, 10, FrontierOptions{Teacher: &FrontierTeacher{K: 1, Risk: 1, Threat: "officers", Explore: .2, Global: true}})
	require.NoError(t, err)
	require.NotNil(t, teacher.FinalState)
	assert.NotEmpty(t, teacher.Samples[""])
	assert.Equal(t, before, start.State)
	wrong := *start
	wrong.Stage = "wrong"
	_, err = PlayFrontier(d, m, wrong, 9, FrontierOptions{})
	require.ErrorContains(t, err, "invalid real campaign frontier")
}
