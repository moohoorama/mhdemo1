package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func storyEngine(t *testing.T, nodes []content.Node) *Engine {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	ids := []string{}
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	d.Nodes = nodes
	d.Chapters = []content.Chapter{{ID: "1", File: "x.json", Nodes: ids}}
	d.Campaign.Start = ids[0]
	return New(d, 1)
}

func next(t *testing.T, e *Engine) {
	_, err := e.Apply(Command{Kind: "next"})
	require.NoError(t, err)
}

func TestSceneSetAndBranchPickTheMatchingCase(t *testing.T) {
	nodes := []content.Node{
		{ID: "a", Kind: "scene", Place: "도원"},
		{ID: "b", Kind: "say", Speaker: "hero", Text: "hi"},
		{ID: "c", Kind: "set", Flags: map[string]int{"oath": 2}},
		{ID: "d", Kind: "branch", Cases: []content.NodeCase{
			{If: []string{"flag:oath>=3"}, Next: "high"},
			{If: []string{"flag:oath>=2", "!flag:other"}, Next: "mid"},
		}, Else: "low"},
		{ID: "high", Kind: "narrate", Text: "high", Next: "z"},
		{ID: "mid", Kind: "narrate", Text: "mid", Next: "z"},
		{ID: "low", Kind: "narrate", Text: "low", Next: "z"},
		{ID: "z", Kind: "end"},
	}
	e := storyEngine(t, nodes)
	o := e.Observe()
	assert.Equal(t, "도원", o.Place)
	assert.Equal(t, "b", o.Node)
	assert.Equal(t, "say", o.Dialogue.Kind)
	next(t, e)
	o = e.Observe()
	assert.Equal(t, "mid", o.Node)
	assert.Equal(t, 2, o.Flags["oath"])
	next(t, e)
	o = e.Observe()
	assert.Equal(t, "complete", o.Phase)
	assert.Empty(t, o.Node)
}

func TestBranchFallsBackToElseThenNextNode(t *testing.T) {
	nodes := []content.Node{
		{ID: "d", Kind: "branch", Cases: []content.NodeCase{{If: []string{"flag:never"}, Next: "x"}}},
		{ID: "after", Kind: "narrate", Text: "after"},
		{ID: "x", Kind: "narrate", Text: "x"},
	}
	e := storyEngine(t, nodes)
	assert.Equal(t, "after", e.Observe().Node, "no Else: the next node of the chapter")
	nodes[0].Else = "x"
	e = storyEngine(t, nodes)
	assert.Equal(t, "x", e.Observe().Node)
}

func TestConditionsReadChoicesCharactersAndVictories(t *testing.T) {
	nodes := []content.Node{
		{ID: "pick", Kind: "choice", Text: "pick", Choices: map[string]string{"go": "d"}},
		{ID: "d", Kind: "branch", Cases: []content.NodeCase{
			{If: []string{"choice:pick=go", "alive:hero", "party:hero", "!dead:hero", "!won:S1"}, Next: "yes"},
		}, Else: "no"},
		{ID: "yes", Kind: "narrate", Text: "yes"},
		{ID: "no", Kind: "narrate", Text: "no"},
	}
	e := storyEngine(t, nodes)
	_, err := e.Apply(Command{Kind: "choose", Option: "go"})
	require.NoError(t, err)
	assert.Equal(t, "yes", e.Observe().Node)
	assert.Equal(t, "go", e.Observe().Choices["pick"])

	s := e.Snapshot()
	s.Executed["won-S1"] = true
	r, err := Restore(e.Data(), s)
	require.NoError(t, err)
	holds, err := content.ParseCond("won:S1")
	require.NoError(t, err)
	assert.True(t, r.holds(holds))
}

func TestStoryStateSurvivesSaveAndRestore(t *testing.T) {
	nodes := []content.Node{
		{ID: "a", Kind: "scene", Place: "막사"},
		{ID: "b", Kind: "set", Flags: map[string]int{"seen": 1}},
		{ID: "c", Kind: "say", Speaker: "hero", Text: "hi"},
	}
	e := storyEngine(t, nodes)
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	o := r.Observe()
	assert.Equal(t, "c", o.Node)
	assert.Equal(t, "막사", o.Place)
	assert.Equal(t, 1, o.Flags["seen"])
}

func TestNextOnTheLastNodeEndsTheCampaign(t *testing.T) {
	e := storyEngine(t, []content.Node{{ID: "only", Kind: "narrate", Text: "the end"}})
	next(t, e)
	assert.Equal(t, "complete", e.Observe().Phase)
}

func fightNodes() []content.Node {
	return []content.Node{
		{ID: "pre", Kind: "say", Speaker: "hero", Text: "before"},
		{ID: "fight", Kind: "battle", Stage: "S1", Text: "go", Next: "post"},
		{ID: "post", Kind: "branch", Cases: []content.NodeCase{{If: []string{"won:S1"}, Next: "win"}}, Else: "lose"},
		{ID: "win", Kind: "say", Speaker: "hero", Text: "won"},
		{ID: "lose", Kind: "narrate", Text: "lost"},
	}
}

func startFight(t *testing.T, e *Engine) {
	next(t, e)
	require.Equal(t, "fight", e.Observe().Node)
	_, err := e.Apply(Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	require.Equal(t, "battle", e.Observe().Phase)
}

func TestStoryFlowsThroughABattleAndBranchesOnTheResult(t *testing.T) {
	e := storyEngine(t, fightNodes())
	startFight(t, e)
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].Faction == "enemy" {
			snap.Units[i].HP = 0
		}
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	_, err = e.Apply(Command{Kind: "wait", Actor: "hero"})
	require.NoError(t, err)
	require.Equal(t, "victory", e.Observe().Result)
	_, err = e.Apply(Command{Kind: "continue"})
	require.NoError(t, err)
	o := e.Observe()
	assert.Equal(t, "scenario", o.Phase)
	assert.Equal(t, "win", o.Node, "the story goes on after the battle node, by the result")
}

func TestRetryAfterDefeatReturnsToTheSortieNotTheScenesBefore(t *testing.T) {
	e := storyEngine(t, fightNodes())
	startFight(t, e)
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].Faction == "ally" {
			snap.Units[i].HP = 0
		}
	}
	e, err := Restore(e.Data(), snap)
	require.NoError(t, err)
	_, err = e.Apply(Command{Kind: "end"})
	require.NoError(t, err)
	require.Equal(t, "defeat", e.Observe().Result)
	require.NoError(t, e.Retry())
	o := e.Observe()
	assert.Equal(t, "scenario", o.Phase)
	assert.Equal(t, "fight", o.Node)
}
