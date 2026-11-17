package autoplay

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srpg/internal/content"
	"srpg/internal/core"
)

func TestPlayersWinTheFirstBattle(t *testing.T) {
	if testing.Short() {
		t.Skip("plays a battle")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	players := []Player{
		&RulePlayer{Label: "rule", D: d, G: DefaultGenome()},
		&Searcher{Label: "search", D: d, G: DefaultGenome(), K: 2, R: 2, P: 2, B: 1, Par: 4},
		&Searcher{Label: "search-depth2", D: d, G: DefaultGenome(), K: 2, R: 2, Depth: 2, P: 2, B: 1, Par: 4},
	}
	for _, p := range players {
		t.Run(p.Name(), func(t *testing.T) {
			r, err := Run(d, p, Options{Difficulty: "hard", Seed: 1, Stages: 1})
			require.NoError(t, err)
			require.Len(t, r.Stages, 1)
			assert.True(t, r.Stages[0].Won)
			assert.Positive(t, r.LevelScore)
		})
	}
}

func TestBattleTraceReplaysCampaignState(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	p := &RulePlayer{Label: "rule", D: d, G: DefaultGenome()}
	var trace bytes.Buffer
	r, err := Run(d, p, Options{Difficulty: "hard", Seed: 1, Stages: 1, Trace: &trace, TraceStage: "B01"})
	require.NoError(t, err)
	dec := json.NewDecoder(&trace)
	var start struct {
		Stage    string
		Try      int
		Snapshot core.State
	}
	require.NoError(t, dec.Decode(&start))
	assert.Equal(t, "B01", start.Stage)
	assert.Equal(t, 1, start.Try)
	e, err := core.Restore(d, start.Snapshot)
	require.NoError(t, err)
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	for {
		var step struct {
			Command core.Command
			Units   []core.UnitView
			Events  []core.Event
		}
		err := dec.Decode(&step)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		result, err := e.Apply(step.Command)
		require.NoError(t, err)
		require.Equal(t, step.Units, e.Observe().UnitViews)
		require.Equal(t, step.Events, result.Events)
	}
	assert.Equal(t, "victory", e.Observe().Result)
	assert.Equal(t, r.Stages[0].Levels, Levels(d, e))
}

func TestLevelScoreWeighsTheTopFiveHalf(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	ref := RefLevel(d)
	levels := map[string]int{}
	for i := range 20 {
		levels[string(rune('a'+i))] = ref
	}
	assert.InDelta(t, ScoreN*7/8, LevelScore(d, levels), 1e-9)
	top := map[string]int{"a": ref, "b": ref, "c": ref, "d": ref, "e": ref}
	assert.InDelta(t, ScoreN/2, LevelScore(d, top), 1e-9)
}

func TestAnEarlyEndingIsNotACampaignClear(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Nodes = append(d.Nodes, content.Node{ID: "early-end", Kind: "end", Text: "early ending"})
	d.Chapters[0].Nodes = append(d.Chapters[0].Nodes, "early-end")
	d.Campaign.Start = "early-end"
	r, err := Run(d, Baseline(d), Options{Seed: 1})
	require.NoError(t, err)
	assert.False(t, r.Cleared)
	assert.Empty(t, r.Stages)
	assert.Equal(t, 2, r.ExpectedBattles)
	assert.Equal(t, "early-end", r.EndNode)
	assert.Equal(t, d.Hash, r.ContentHash)
}

func TestStoryChoicesAvoidPrematureEndings(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "immediate ending", true: "nested ending"}[nested], func(t *testing.T) {
			d, err := content.Load("../testdata/mini")
			require.NoError(t, err)
			bad := "early-end"
			if nested {
				bad = "inner-fork"
			}
			d.Nodes = append(d.Nodes,
				content.Node{ID: "fork", Kind: "choice", Choices: map[string]string{"a": bad, "b": "prep1"}},
				content.Node{ID: "inner-fork", Kind: "choice", Choices: map[string]string{"a": "early-end", "b": "early-end"}},
				content.Node{ID: "early-end", Kind: "end", Text: "early ending"})
			d.Chapters[0].Nodes = append(d.Chapters[0].Nodes, "fork", "inner-fork", "early-end")
			d.Campaign.Start = "fork"
			e := core.New(d, 1)
			before := e.Snapshot()
			c := ChooseStory(e, Baseline(d))
			assert.Equal(t, "b", c.Option)
			assert.Equal(t, before, e.Snapshot(), "looking ahead does not change the real game")
			_, err = e.Apply(c)
			require.NoError(t, err)
			assert.Equal(t, "battle", e.Observe().Dialogue.Kind)
		})
	}
}

func TestStoryReachesBattlesAfterTheCourtChoices(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, tc := range []struct{ node, stage string }{{"c17-walk", "B17"}, {"c19-ye", "B19"}} {
		t.Run(tc.stage, func(t *testing.T) {
			e, err := core.NewAt(d, 1, tc.node, "hard")
			require.NoError(t, err)
			p := &RulePlayer{D: d, G: DefaultGenome()}
			for range 500 {
				o := e.Observe()
				require.NotEqual(t, "complete", o.Phase, "premature ending: %s", o.Ending)
				var commands []core.Command
				switch {
				case len(o.Learning) > 0:
					commands = []core.Command{p.Learn(e)}
				case e.Walking():
					commands = e.ExploreNext()
				case o.Dialogue.Kind == "choice":
					commands = []core.Command{ChooseStory(e, p)}
				case o.Dialogue.Kind == "battle":
					assert.Equal(t, tc.stage, d.Stages[o.Stage].ID)
					return
				default:
					commands = []core.Command{{Kind: "next"}}
				}
				require.NotEmpty(t, commands)
				for _, c := range commands {
					_, err := e.Apply(c)
					require.NoError(t, err)
				}
			}
			t.Fatal("story did not reach the battle")
		})
	}
}

func TestStoryChoiceSeesAnEndingAfterReturningToTheWalk(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Nodes = append(d.Nodes,
		content.Node{ID: "fork", Kind: "choice", Choices: map[string]string{"a": "bad-flag", "b": "good-flag"}},
		content.Node{ID: "bad-flag", Kind: "set", Flags: map[string]int{"bad": 1}, Next: "walk"},
		content.Node{ID: "good-flag", Kind: "set", Flags: map[string]int{"bad": 0}, Next: "walk"},
		content.Node{ID: "walk", Kind: "explore", Explore: "test-walk"},
		content.Node{ID: "judgment", Kind: "branch", Cases: []content.NodeCase{{If: []string{"flag:bad"}, Next: "early-end"}}, Else: "continue"},
		content.Node{ID: "continue", Kind: "back", Leave: true},
		content.Node{ID: "early-end", Kind: "end", Text: "early ending"})
	d.Chapters[0].Nodes = append([]string{"fork", "bad-flag", "good-flag", "walk"}, d.Chapters[0].Nodes...)
	d.Chapters[0].Events = append(d.Chapters[0].Events, "judgment", "continue", "early-end")
	d.Campaign.Start = "fork"
	d.Explores = map[string]*content.Explore{"test-walk": {
		Start: content.Spot{Area: "street", Face: "SE"},
		Areas: map[string]*content.ExploreArea{"street": {
			People: map[string]*content.ExplorePerson{"guide": {Event: []content.ExploreEvent{{Node: "judgment"}}}},
		}},
	}}
	e := core.New(d, 1)
	before := e.Snapshot()
	c := ChooseStory(e, Baseline(d))
	assert.Equal(t, "b", c.Option)
	assert.Equal(t, before, e.Snapshot())
}
