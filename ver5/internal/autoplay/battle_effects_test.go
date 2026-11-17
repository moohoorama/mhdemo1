package autoplay

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestEffectInputsSeparateBuffsAndConsumableAmounts(t *testing.T) {
	d, e := battleModelFixture(t)
	d.Skills["test_attack"] = content.Skill{Effect: "attack", Duration: 3}
	d.Skills["test_defense"] = content.Skill{Effect: "defense", Duration: 3}
	a := effectInputs(d, nil, core.UnitView{}, core.Command{Kind: "skill", Skill: "test_attack"})
	b := effectInputs(d, nil, core.UnitView{}, core.Command{Kind: "skill", Skill: "test_defense"})
	assert.Equal(t, 1.0, a[0])
	assert.Zero(t, a[1])
	assert.Equal(t, 1.0, b[1])
	assert.Zero(t, b[0])
	d.Items["test_small"] = content.Item{Amount: 100}
	d.Items["test_large"] = content.Item{Amount: 200}
	a = effectInputs(d, nil, core.UnitView{}, core.Command{Kind: "item", Item: "test_small"})
	b = effectInputs(d, nil, core.UnitView{}, core.Command{Kind: "item", Item: "test_large"})
	assert.Less(t, a[11], b[11])
	before := e.Snapshot()
	m := NewBattleModel(8, 1)
	m.ContentHash = d.Hash
	old := m.policy(m.inputs(d, e).Features)
	m.EnableEffectInputs()
	m.EnableEffectInputs()
	require.NoError(t, m.Validate())
	in := m.inputs(d, e)
	next := m.policy(in.Features)
	for i := range old {
		assert.InDelta(t, old[i], next[i], 1e-12)
		require.Len(t, in.Features[i], 144)
	}
	assert.Equal(t, before, e.Snapshot())
	path := filepath.Join(t.TempDir(), "effects.json")
	require.NoError(t, m.Save(path))
	loaded, err := LoadBattleModel(path)
	require.NoError(t, err)
	assert.Equal(t, next, loaded.policy(loaded.inputs(d, e).Features))
}

func TestGoalInputInsertionPreservesLearnedEffectColumns(t *testing.T) {
	m := NewBattleModel(8, 1)
	m.EnableEffectInputs()
	m.Actor.W[0][128] = 2
	old := make([]float64, 144)
	old[128] = 1
	want := m.Actor.Predict(old)
	m.EnableGoalInputs()
	require.NoError(t, m.Validate())
	next := make([]float64, 152)
	next[136] = 1
	assert.Equal(t, want, m.Actor.Predict(next))
	assert.Equal(t, 2.0, m.Actor.W[0][136])
}
