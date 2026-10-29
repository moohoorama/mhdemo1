package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestTheEnemyComingUpToASupportPutsThatSupportInFront(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	stage := d.StageIndex("B06")
	require.GreaterOrEqual(t, stage, 0)
	at := func(id string, x, y int) core.UnitView {
		c := d.Characters[id]
		return core.UnitView{Unit: core.Unit{ID: id, Character: id, X: x, Y: y}, Class: c.Class, Def: id}
	}
	g.player.final.Stage = stage
	g.player.final.UnitViews = []core.UnitView{at("zhangfei", 2, 20), at("guanyu", 10, 10), at("liubei", 12, 12), at("lubu", 10, 11)}

	v := g.duelView(core.Event{Kind: "duel", DuelID: "hulao-brothers", Outcome: "victory", Actor: "zhangfei", Target: "lubu", First: "lubu"})
	assert.Equal(t, d.Characters["guanyu"].Name, v.Fighters[0].Name, "Guan Yu, whom Lu Bu came up to, opens the duel")
	assert.Equal(t, g.assets.Duels.StoryFor("hulao-brothers", "victory", "guanyu"), v.Story)
	assert.Equal(t, d.Characters["zhangfei"].Name, v.Fighters[2].Name, "Zhang Fei joins in Guan Yu's place")
}
