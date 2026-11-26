package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srpg/internal/content"
)

func openRedCliffs(t *testing.T) *Engine {
	t.Helper()
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := NewAt(d, 1, "c27-redcliff-battle", "normal")
	require.NoError(t, err)
	for _, id := range e.Observe().Learning {
		applyOK(t, e, Command{Kind: "learn_close", Actor: id})
	}
	applyOK(t, e, Command{Kind: "start"})
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, command := range plan {
		applyOK(t, e, command)
	}

	return e
}

func TestRedCliffsFireAndWithdrawal(t *testing.T) {
	e := openRedCliffs(t)
	require.True(t, e.state.Executed["battle-B27R-fire-ships"])
	for _, id := range []string{"zhouyu", "chengpu", "huanggai", "ganning", "handang", "liuqi", "liubei", "zhugeliang"} {
		assert.NotNil(t, e.unit(id), id)
	}
	for _, id := range []string{"guanyu", "zhangfei", "zhaoyun"} {
		assert.Nil(t, e.unit(id), id)
	}
	cao, zhang := e.unit("caocao"), e.unit("zhangyun")
	assert.Equal(t, 3, cao.Status["confusion"].Turns)
	assert.Equal(t, -20, cao.Status["attack"].Value)
	assert.Equal(t, "flee", cao.Policy)
	assert.Equal(t, 81, cao.GoalArea)
	assert.Equal(t, "assault", e.unit("zhouyu").Policy)
	assert.Equal(t, byte('F'), e.Cell(5, 15).State)
	assert.Equal(t, byte('F'), e.Cell(14, 15).State)
	assert.Zero(t, zhang.Status["confusion"].Turns)
	assert.Zero(t, zhang.Status["attack"].Value)
	assert.Equal(t, e.stats(zhang).MaxHP, zhang.HP)
	assert.Equal(t, byte('-'), e.Cell(zhang.X, zhang.Y).State)

	cao.X, cao.Y = 18, 0
	e.settle()
	assert.True(t, e.holdsIn([]string{"fled:caocao"}, true))
	assert.Positive(t, zhang.HP)
	assert.Equal(t, "victory", e.Observe().Result)
}

func TestRedCliffsLandArmyCanOpenTheWaterway(t *testing.T) {
	e := openRedCliffs(t)
	liuqi := e.unit("liuqi")
	require.Equal(t, "hold", liuqi.Policy)
	liubei := e.unit("liubei")
	liubei.X, liubei.Y = 32, 10
	e.battleEvents()
	assert.Equal(t, "assault", liuqi.Policy)

	e.unit("zhangyun").HP = 0
	e.battleEvents()
	for _, u := range e.matching("zhang-fleet") {
		if u.HP > 0 {
			assert.Equal(t, "flee", u.Policy, u.ID)
			assert.Equal(t, 88, u.GoalArea, u.ID)
		}
	}
	require.Equal(t, "battle", e.Observe().Phase)
	cao := e.unit("caocao")
	cao.X, cao.Y = 18, 0
	e.settle()
	assert.Equal(t, "victory", e.Observe().Result)
	assert.Equal(t, 1, e.state.Flags["flag-b27r-zhangyun"])
}
