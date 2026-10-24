package ai

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// rushCampaign walks the story from its start, pages through the lines, takes the first choice and
// fights every battle with RushNext until it has fought stages battles or lost one. It returns each
// battle's result by stage.
func rushCampaign(t *testing.T, seed uint64, stages int) (*core.Engine, map[string]string) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e := core.New(d, seed)
	apply := func(c core.Command) {
		r, err := e.Apply(c)
		require.NoError(t, err, "%+v", c)
		for _, ev := range r.Events {
			switch ev.Kind {
			case "retreat", "duel", "victory", "defeat":
				o := e.Observe()
				t.Logf("  r%d %s %s %s%s", o.Round, ev.Kind, ev.Actor, ev.Target, ev.DuelID)
			}
		}
	}
	results := map[string]string{}
	for step := 0; step < 3000 && len(results) < stages; step++ {
		o := e.Observe()
		switch {
		case o.Phase == "scenario" && (o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate"):
			apply(core.Command{Kind: "next"})
		case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
			options := []string{}
			for k := range o.Dialogue.Choices {
				options = append(options, k)
			}
			sort.Strings(options)
			apply(core.Command{Kind: "choose", Option: options[0]})
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			id := d.Stages[o.Stage].ID
			apply(core.Command{Kind: "start"})
			for i := 0; i < 20000 && e.Observe().Phase != "result"; i++ {
				c, err := RushNext(e)
				require.NoError(t, err)
				apply(c)
			}
			o := e.Observe()
			for _, c := range o.Characters {
				if c.ID == "liubei" || c.ID == "guanyu" || c.ID == "zhangfei" {
					t.Logf("  after %s: %s Lv%d", id, c.ID, c.Level)
				}
			}
			results[id] = o.Result
			t.Logf("%s: %s in round %d", id, o.Result, o.Round)
			if o.Result != "victory" {
				return e, results
			}
			for c, ok := Learning(e); ok; c, ok = Learning(e) {
				apply(c)
			}
			apply(core.Command{Kind: "continue"})
		default:
			t.Fatalf("stuck in phase %s at node %s (%s)", o.Phase, o.Node, o.Dialogue.Kind)
		}
	}
	return e, results
}

// TestRushWinsTheFirstStages: a player who only charges the nearest enemy still gets through the
// first three stages in at least four games of five (docs/stage_lv_balance.md), and B02's story plays
// out by itself in every game.
func TestRushWinsTheFirstStages(t *testing.T) {
	wins := 0
	for seed := uint64(1); seed <= 5; seed++ {
		e, results := rushCampaign(t, seed, 3)
		if results["B02"] == "victory" && results["B05"] == "victory" && results["B06"] == "victory" {
			wins++
		} else {
			t.Logf("seed %d: %v", seed, results)
		}
		executed := e.Snapshot().Executed
		for _, k := range []string{"battle-B02-night-raid", "battle-B02-chengyuanzhi-sorties", "duel-yingchuan-liubei-bocai"} {
			assert.True(t, executed[k], "seed %d: %s", seed, k)
		}
	}
	assert.GreaterOrEqual(t, wins, 4, "careless campaigns won of 5")
}
