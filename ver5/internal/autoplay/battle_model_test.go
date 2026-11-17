package autoplay

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func battleModelFixture(t *testing.T) (*content.Data, *core.Engine) {
	t.Helper()
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Campaign.Start = "prep1"
	d.Stages[0].Enemies[0].X, d.Stages[0].Enemies[0].Y = 2, 2
	e := core.New(d, 1)
	for _, c := range []core.Command{{Kind: "start"}, {Kind: "place", Actor: "hero", X: 1, Y: 2}, {Kind: "place", Actor: "archer1", X: 1, Y: 3}, {Kind: "deployment_finish"}} {
		_, err := e.Apply(c)
		require.NoError(t, err)
	}
	return d, e
}

func TestBattleModelOffersAllLegalCommandsWithoutChangingState(t *testing.T) {
	d, e := battleModelFixture(t)
	before := e.Snapshot()
	in := BattleInputs(d, e)
	require.Len(t, in.State, BattleStateLen)
	require.Greater(t, len(in.Commands), 4)
	for _, u := range e.Observe().UnitViews {
		if !mine(u) || u.HP <= 0 || u.Done {
			continue
		}
		for _, c := range e.LegalActions(u.ID).Commands {
			assert.Contains(t, in.Commands, c)
		}
	}
	for i, row := range in.Features {
		require.Len(t, row, BattleStateLen+BattleActionLen)
		for _, v := range row {
			require.False(t, math.IsNaN(v) || math.IsInf(v, 0))
		}
		copy, err := core.Restore(d, before)
		require.NoError(t, err)
		_, err = copy.Apply(in.Commands[i])
		require.NoError(t, err)
	}
	assert.Equal(t, before, e.Snapshot())
	m := NewBattleModel(8, 1)
	c := m.Select(e)
	assert.Contains(t, in.Commands, c)
	assert.Equal(t, before, e.Snapshot())
}

func TestBattlePolicyGradientMatchesFiniteDifferences(t *testing.T) {
	logits := []float64{.2, -.7, .4}
	p := probabilities(logits)
	old := math.Log(p[1])
	const advantage, entropy = .7, .01
	grad := policyGradient(p, 1, advantage, &old, entropy)
	loss := func(x []float64) float64 {
		q := probabilities(x)
		ratio := math.Exp(math.Log(q[1]) - old)
		v := -min(ratio*advantage, max(.8, min(1.2, ratio))*advantage)
		for _, prob := range q {
			v += entropy * prob * math.Log(prob)
		}
		return v
	}
	for i := range logits {
		copy := append([]float64{}, logits...)
		copy[i] += 1e-6
		hi := loss(copy)
		copy[i] -= 2e-6
		lo := loss(copy)
		assert.InDelta(t, (hi-lo)/2e-6, grad[i], 1e-6)
	}
	t.Run("clipped updates do not push the probability farther", func(t *testing.T) {
		old := math.Log(.1)
		assert.Equal(t, []float64{0, 0}, policyGradient([]float64{.2, .8}, 0, 1, &old, 0))
		old = math.Log(.9)
		assert.Equal(t, []float64{0, 0}, policyGradient([]float64{.2, .8}, 0, -1, &old, 0))
	})
}

func TestBattlePPOIncreasesTheProbabilityOfTheBetterReturn(t *testing.T) {
	m := NewBattleModel(8, 5)
	trainer := NewBattleTrainer(m, 5)
	in := BattleChoices{State: make([]float64, BattleStateLen), Features: [][]float64{make([]float64, BattleStateLen+BattleActionLen), make([]float64, BattleStateLen+BattleActionLen)}}
	in.Features[0][BattleStateLen] = 1
	in.Features[1][BattleStateLen+1] = 1
	before := m.policy(in.Features)[1]
	for range 30 {
		p := m.policy(in.Features)
		rows := []BattleSample{
			{Inputs: in, Chosen: 0, LogProb: math.Log(p[0]), Return: -2},
			{Inputs: in, Chosen: 1, LogProb: math.Log(p[1]), Return: 3},
		}
		trainer.Improve(rows, 2, 2, .01)
	}
	assert.Greater(t, m.policy(in.Features)[1], before+.2)
	assert.Greater(t, m.policy(in.Features)[1], .9)
}

func TestBattleModelRoundTripAndMalformedWeights(t *testing.T) {
	m := NewBattleModel(8, 1)
	path := filepath.Join(t.TempDir(), "model.json")
	require.NoError(t, m.Save(path))
	loaded, err := LoadBattleModel(path)
	require.NoError(t, err)
	_, e := battleModelFixture(t)
	assert.Equal(t, m.Select(e), loaded.Select(e))
	loaded.Actor.W[0] = loaded.Actor.W[0][:2]
	b, err := json.Marshal(loaded)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o600))
	_, err = LoadBattleModel(path)
	require.Error(t, err)
}

func TestBattleGrowthReturnRewardsTheRequestedPlayStyle(t *testing.T) {
	base := BattleOutcome{Won: true, XP: 4, OthersLeft: 1}
	value := base.GrowthReturn()
	t.Run("actual experience", func(t *testing.T) {
		r := base
		r.XP++
		assert.Greater(t, r.GrowthReturn(), value)
	})
	t.Run("supplies have a cost", func(t *testing.T) {
		for used := 1; used <= 6; used++ {
			r := base
			r.ItemsUsed = used
			assert.Less(t, r.GrowthReturn(), value)
		}
	})
	t.Run("ordinary enemies should be cleared", func(t *testing.T) {
		r := base
		r.OthersLeft = 2
		assert.Less(t, r.GrowthReturn(), value)
		r.OthersLeft = 0
		assert.Equal(t, value, r.GrowthReturn())
	})
	t.Run("a loss cannot keep the experience bonus", func(t *testing.T) {
		r := base
		r.Won, r.XP = false, 100
		assert.Equal(t, -10.0, r.GrowthReturn())
		r.Won, r.Error = true, "truncated"
		assert.Equal(t, -10.0, r.GrowthReturn())
	})
	t.Run("supply costs cannot make a win worse than a loss", func(t *testing.T) {
		r := BattleOutcome{Won: true, ItemsUsed: 1000, OthersLeft: 1000}
		assert.Greater(t, r.GrowthReturn(), -10.0)
	})
	t.Run("finishing earlier has no bonus", func(t *testing.T) {
		r := base
		r.Round = 12
		assert.Equal(t, value, r.GrowthReturn())
	})
}

func TestBattleStageModelsOnlyReplaceTheirOwnPolicy(t *testing.T) {
	d, e := battleModelFixture(t)
	root, first := NewBattleModel(8, 1), NewBattleModel(8, 2)
	root.ContentHash, first.ContentHash = d.Hash, d.Hash
	prefer := func(m *BattleModel, kind int) {
		for l := range m.Actor.W {
			clear(m.Actor.W[l])
			clear(m.Actor.B[l])
		}
		m.Actor.W[0][BattleStateLen+kind], m.Actor.W[1][0], m.Actor.W[2][0] = 1, 1, 1
	}
	prefer(root, 4)
	prefer(first, 0)
	assert.Equal(t, "wait", root.Select(e).Kind)
	root.StageModels = map[string]*BattleModel{"another-stage": first}
	assert.Equal(t, "wait", root.Select(e).Kind)
	id := d.Stages[e.Observe().Stage].ID
	root.StageModels = map[string]*BattleModel{id: first}
	assert.Equal(t, "move", root.Select(e).Kind)
	path := filepath.Join(t.TempDir(), "bundle.json")
	require.NoError(t, root.Save(path))
	loaded, err := LoadBattleModel(path)
	require.NoError(t, err)
	assert.Equal(t, first.Select(e), loaded.Select(e))
	first.ContentHash = "different"
	require.Error(t, root.Validate())
	first.ContentHash = d.Hash
	first.StageModels = map[string]*BattleModel{id: root}
	require.Error(t, root.Validate(), "nested stage models are rejected before recursion")
}

func TestBattleModelDoesNotOverrideItsPolicyForAnImmediateWin(t *testing.T) {
	d, e := battleModelFixture(t)
	d.Stages[0].Victory = []content.Goal{{If: []string{"defeated:boss"}}}
	d.Rules.HitFloor, d.Rules.HitCeil = 100, 100
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == "boss" {
			snap.Units[i].HP = 1
		}
	}
	e, err := core.Restore(d, snap)
	require.NoError(t, err)
	m := NewBattleModel(8, 1)
	for l := range m.Actor.W {
		clear(m.Actor.W[l])
		clear(m.Actor.B[l])
	}
	m.Actor.W[0][BattleStateLen+4], m.Actor.W[1][0], m.Actor.W[2][0] = 1, 1, 1 // prefer waiting
	winning := false
	for _, c := range BattleInputs(d, e).Commands {
		copy, err := core.Restore(d, snap)
		require.NoError(t, err)
		_, err = copy.Apply(c)
		require.NoError(t, err)
		winning = winning || copy.Observe().Result == "victory"
	}
	require.True(t, winning)
	assert.Equal(t, "wait", m.Select(e).Kind)
	assert.True(t, reflect.DeepEqual(snap, e.Snapshot()))
}

func TestBattleExamplesRequireVerifiedCompleteBattles(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	var trace bytes.Buffer
	p := &RulePlayer{D: d, G: DefaultGenome()}
	_, err = Run(d, p, Options{Difficulty: "hard", Seed: 1, Stages: 1, Trace: &trace})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	require.NoError(t, os.WriteFile(path, trace.Bytes(), 0o600))
	starts, rows, err := ReadBattleExamples(d, []string{path}, []string{"B01"})
	require.NoError(t, err)
	require.Len(t, starts, 1)
	require.NotEmpty(t, rows)
	for _, row := range rows {
		assert.Greater(t, row.Return, 2.0)
		assert.Less(t, row.Chosen, len(row.Inputs.Commands))
	}
	header, _, ok := bytes.Cut(trace.Bytes(), []byte("\n"))
	require.True(t, ok)
	t.Run("incomplete recording", func(t *testing.T) {
		require.NoError(t, os.WriteFile(path, header, 0o600))
		starts, rows, err := ReadBattleExamples(d, []string{path}, []string{"B01"})
		require.NoError(t, err)
		assert.Empty(t, starts)
		assert.Empty(t, rows)
	})
	t.Run("altered command", func(t *testing.T) {
		altered := append(append([]byte{}, header...), []byte("\n{\"Command\":{\"Kind\":\"nonsense\"}}\n")...)
		require.NoError(t, os.WriteFile(path, altered, 0o600))
		_, _, err := ReadBattleExamples(d, []string{path}, []string{"B01"})
		require.Error(t, err)
	})
	t.Run("altered result", func(t *testing.T) {
		altered := bytes.Replace(trace.Bytes(), []byte(`"Result":"victory"`), []byte(`"Result":"defeat"`), 1)
		require.NotEqual(t, trace.Bytes(), altered)
		require.NoError(t, os.WriteFile(path, altered, 0o600))
		_, _, err := ReadBattleExamples(d, []string{path}, []string{"B01"})
		require.Error(t, err)
	})
}
