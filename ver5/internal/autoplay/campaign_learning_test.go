package autoplay

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestFirstLossProgressDoesNotCountWinsAfterARetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stages []StageReport
		wins   int
		lost   string
	}{
		{"first battle lost", []StageReport{{Stage: "a", Tries: 1}}, 0, "a"},
		{"retry eventually won", []StageReport{{Stage: "a", Tries: 1, Won: true}, {Stage: "b", Tries: 2, Won: true}, {Stage: "c", Tries: 1, Won: true}}, 1, "b"},
		{"later defeat", []StageReport{{Stage: "a", Tries: 1, Won: true}, {Stage: "b", Tries: 1}}, 1, "b"},
		{"all won", []StageReport{{Stage: "a", Tries: 1, Won: true}, {Stage: "b", Tries: 1, Won: true}}, 2, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, lost := (Report{Stages: tc.stages}).FirstLossProgress()
			assert.Equal(t, tc.wins, n)
			assert.Equal(t, tc.lost, lost)
		})
	}
}

type waitingCampaignPlayer struct{ Player }

func (p waitingCampaignPlayer) Act(_ *core.Engine, id string) core.Command {
	return core.Command{Kind: "wait", Actor: id}
}

func TestCampaignStopsAtTheFirstDefeatWhenRetriesAreDisabled(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Campaign.Start = "prep1"
	d.Stages[0].TurnLimit = 1
	for _, tries := range []int{1, 3} {
		r, err := Run(d, waitingCampaignPlayer{Baseline(d)}, Options{Seed: 7, Tries: tries})
		require.NoError(t, err)
		require.Len(t, r.Stages, 1)
		assert.False(t, r.Stages[0].Won)
		assert.Equal(t, tries, r.Stages[0].Tries)
		assert.Equal(t, 0, r.UnbeatenStages)
		assert.Equal(t, d.Stages[0].ID, r.FirstDefeat)
	}
}

func TestPreparationCandidatesPreserveStateAndUseRealResources(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Campaign.Start = "prep1"
	e := core.New(d, 1)
	snap := e.Snapshot()
	snap.Inventory = map[string]int{}
	snap.Warehouse = map[string]int{}
	for id := range d.Equipment {
		snap.Warehouse[id] = 2
	}
	for i := range snap.Characters {
		snap.Characters[i].Points = 3000
	}
	e, err = core.Restore(d, snap)
	require.NoError(t, err)
	before := e.Snapshot()
	for _, slot := range []string{"", "weapon", "armor", "accessory"} {
		in := PreparationInputs(d, e, "hero", slot, false)
		require.NotEmpty(t, in.Commands)
		for i, c := range in.Commands {
			require.Len(t, in.Features[i], 128)
			if c.Kind == "skip" {
				continue
			}
			copy, err := core.Restore(d, before)
			require.NoError(t, err)
			_, err = copy.Apply(c)
			require.NoError(t, err)
			if c.Kind == "learn" {
				assert.Equal(t, charOf(e, "hero").Points-d.Traits[c.Trait].Cost, charOf(copy, "hero").Points)
			}
		}
	}
	assert.Equal(t, before, e.Snapshot())
	m := NewBattleModel(8, 4)
	m.ContentHash = d.Hash
	m.EnableCampaignLearning(8, 5)
	for l := range m.Preparation.Actor.W {
		clear(m.Preparation.Actor.W[l])
		clear(m.Preparation.Actor.B[l])
	}
	m.Preparation.Actor.W[0][65], m.Preparation.Actor.W[1][0], m.Preparation.Actor.W[2][0] = 1, 1, 1
	p := &NeuralPlayer{Searcher: CampaignPlayer(d), Model: m}
	commands := p.PrepareCommands(e, 1)
	require.NoError(t, p.AutoplayError())
	replay, err := core.Restore(d, before)
	require.NoError(t, err)
	learned := 0
	for _, c := range commands {
		_, err = replay.Apply(c)
		require.NoError(t, err)
		if c.Kind == "learn" {
			learned++
		}
	}
	assert.Positive(t, learned)
	assert.Equal(t, e.Snapshot(), replay.Snapshot())
	assert.Equal(t, before.Inventory, e.Observe().Inventory)
	path := filepath.Join(t.TempDir(), "joint.json")
	require.NoError(t, m.Save(path))
	loaded, err := LoadBattleModel(path)
	require.NoError(t, err)
	require.NotNil(t, loaded.Preparation)
	assert.Equal(t, m.Preparation.Actor.W, loaded.Preparation.Actor.W)
}

func TestRichBattleInputsKeepInitialPredictions(t *testing.T) {
	d, e := battleModelFixture(t)
	m := NewBattleModel(8, 18)
	m.ContentHash = d.Hash
	old := m.policy(BattleInputs(d, e).Features)
	m.EnableCampaignLearning(8, 19)
	next := m.policy(battleInputs(d, e, true).Features)
	require.Len(t, next, len(old))
	for i := range old {
		assert.InDelta(t, old[i], next[i], 1e-12)
	}
}

func TestCampaignAssessmentRanksProgressBeforeOtherScores(t *testing.T) {
	a := CampaignAssessment{}
	b := CampaignAssessment{}
	a.Add(Report{Stages: []StageReport{{Stage: "a", Won: true, Tries: 1}}, LevelScore: 0})
	b.Add(Report{Stages: []StageReport{{Stage: "a", Won: true, Tries: 2}}, LevelScore: 100})
	assert.True(t, a.Better(b))
	assert.False(t, b.Better(a))
}
