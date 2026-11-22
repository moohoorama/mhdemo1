package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
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

	v := g.duelView(core.Event{Kind: "duel", DuelID: "hulao-brothers", Outcome: "victory", Actor: "zhangfei", Target: "lubu", First: "lubu", Fighter: "guanyu"})
	assert.Equal(t, d.Characters["guanyu"].Name, v.Fighters[0].Name, "Guan Yu, whom Lu Bu came up to, opens the duel")
	assert.Equal(t, g.assets.Duels.StoryFor("hulao-brothers", "victory", "guanyu"), v.Story)
	assert.Equal(t, d.Characters["zhangfei"].Name, v.Fighters[2].Name, "Zhang Fei joins in Guan Yu's place")
}

func TestTurnEndWaitsForCaiyangDuelBeforeRetreat(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()

	g.player.final.Stage = d.StageIndex("B20")
	g.vis = map[string]*unitVis{"caiyang": {id: "caiyang", shownHP: 20, dying: -1}}
	action := &replay.Action{}
	action.Add(core.Command{Kind: "end"}, []core.Event{
		{Kind: "duel", DuelID: "mangdang-guanyu-caiyang", Outcome: "victory", Actor: "guanyu", Target: "caiyang"},
		{Kind: "death", Actor: "caiyang"},
	})
	seenDuel := false
	for _, beat := range g.perform(action) {
		beat.fn()
		if g.player.Duel != nil {
			seenDuel = true
		}
		if !seenDuel || g.player.Duel != nil {
			require.Equal(t, 20, g.vis["caiyang"].shownHP)
		}
	}
	assert.True(t, seenDuel)
	assert.Zero(t, g.vis["caiyang"].shownHP)
	assert.Equal(t, 2, g.sfxCalls, "one duel cue and one fall cue")
}
