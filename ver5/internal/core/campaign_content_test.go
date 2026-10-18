package core

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// force makes one battle condition true by moving or striking the units it names; the campaign test
// uses it to win each stage in the way its victory goal asks, without playing the fight.
func force(t *testing.T, e *Engine, c content.Cond) {
	switch c.Op {
	case "defeated", "alldefeated":
		for _, u := range e.matching(c.Arg) {
			u.HP = 0
		}
	case "hp":
		for _, u := range e.matching(c.Arg) {
			u.HP = 1
		}
	case "round":
		e.state.Round = max(e.state.Round, c.N)
	case "duel":
		e.state.Executed["duel-"+c.Arg] = true
	case "flag":
		if e.state.PendingFlags == nil {
			e.state.PendingFlags = map[string]int{}
		}
		e.state.PendingFlags[c.Arg] = c.N
	case "any":
		force(t, e, c.Any[0])
	case "at", "allat":
		area := e.data.Stages[e.state.Stage].Areas
		for _, u := range e.matching(c.Arg) {
			if u.HP <= 0 || e.inArea(u, c.N) {
				continue
			}
			placed := false
			for _, a := range area {
				for y := a.Rect.Y; y < a.Rect.Y+a.Rect.H && a.ID == c.N && !placed; y++ {
					for x := a.Rect.X; x < a.Rect.X+a.Rect.W && !placed; x++ {
						if o := e.occupied(x, y); o == nil && e.terrain(u, x, y).Cost > 0 {
							u.X, u.Y, placed = x, y, true
						}
					}
				}
			}
			require.True(t, placed, "no room in area %d for %s", c.N, u.ID)
		}
	}
}

func satisfy(t *testing.T, e *Engine, conds []string) {
	for _, s := range conds {
		c, err := content.ParseCond(s)
		require.NoError(t, err, s)
		if !c.Not {
			force(t, e, c)
		}
	}
}

// winStage sorties for the stage the engine stands at and wins it the way its first victory goal asks;
// with bonus it also does what every bonus objective asks.
func winStage(t *testing.T, e *Engine, bonus bool) {
	stage := e.data.Stages[e.state.Stage]
	applyOK(t, e, Command{Kind: "start"})
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		applyOK(t, e, c)
	}
	// let a few rounds pass first, so that the events that wait for time or for a place have happened
	for i := 0; i < 10 && e.Observe().Phase == "battle" && e.state.Round < 5; i++ {
		applyOK(t, e, Command{Kind: "end"})
	}
	for i := 0; i < 14 && e.Observe().Phase == "battle"; i++ {
		satisfy(t, e, stage.Victory[0].If)
		if bonus {
			for _, o := range stage.Sub {
				satisfy(t, e, o.If)
			}
		}
		applyOK(t, e, Command{Kind: "end"})
	}
	require.Equal(t, "result", e.Observe().Phase, "%s was not won", stage.ID)
	require.Equal(t, "victory", e.Observe().Result, stage.ID)
}

// playCampaign walks the whole story: pages through the lines, takes the first choice, wins every
// battle, and reports which stages it fought and what the party ended up with.
func playCampaign(t *testing.T, bonus bool) (*Engine, []string) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e := New(d, 3)
	var fought []string
	for step := 0; step < 2000; step++ {
		o := e.Observe()
		switch {
		case o.Phase == "complete":
			return e, fought
		case o.Phase == "scenario" && (o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate"):
			applyOK(t, e, Command{Kind: "next"})
		case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
			options := []string{}
			for k := range o.Dialogue.Choices {
				options = append(options, k)
			}
			sort.Strings(options)
			applyOK(t, e, Command{Kind: "choose", Option: options[0]})
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			fought = append(fought, d.Stages[o.Stage].ID)
			winStage(t, e, bonus)
			applyOK(t, e, Command{Kind: "continue"})
		default:
			t.Fatalf("stuck in phase %s at node %s (%s)", o.Phase, o.Node, o.Dialogue.Kind)
		}
	}
	t.Fatal("the campaign did not end")
	return nil, nil
}

func executed(e *Engine, key string) bool { return e.state.Executed[key] }

func TestCampaignPlaysThroughWithEveryBonusObjective(t *testing.T) {
	e, fought := playCampaign(t, true)
	assert.Equal(t, []string{"B01", "B03", "B04", "B05", "B07", "B08", "B09"}, fought)
	assert.Equal(t, 1, e.state.Flags["flag-yingchuan-village"])
	assert.Equal(t, 1, e.state.Flags["flag-xufu-alive"])
	assert.Equal(t, 1, e.state.Flags["flag-sunjian-alive"])
	assert.Equal(t, 1, e.state.Flags["flag-bianshui-generals"])
	assert.Equal(t, 1, e.state.Flags["flag-julu-clear"])
	for _, item := range []string{"fire_manual", "gujeong_blade", "yitian_sword"} {
		assert.Positive(t, e.state.Warehouse[item], item+" is handed over")
	}
	for _, id := range []string{"jianyong", "fangong", "zhaoyun"} {
		assert.NotNil(t, e.char(id), id+" joins the party")
	}
	assert.True(t, executed(e, "battle-B09-vanguard-clear"), "a clean win at Julu eases the vanguard at Jiejiao")
	assert.False(t, executed(e, "battle-B09-vanguard-strong"))
}

func TestCampaignPlaysThroughWithNoBonusObjective(t *testing.T) {
	e, fought := playCampaign(t, false)
	assert.Len(t, fought, 7)
	assert.Zero(t, e.state.Flags["flag-yingchuan-village"])
	assert.Zero(t, e.state.Flags["flag-julu-clear"])
	assert.Zero(t, e.state.Warehouse["fire_manual"], "the manual stays with its giver")
	assert.True(t, executed(e, "battle-B09-vanguard-strong"), "the vanguard at Jiejiao is the hard one")
	assert.False(t, executed(e, "battle-B09-vanguard-clear"))
}

func TestEveryCampaignDuelStartsWhenItsOfficersMeet(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for si, stage := range d.Stages {
		for _, duel := range stage.Duels {
			t.Run(duel.ID, func(t *testing.T) {
				dd := *d
				for _, n := range d.Nodes {
					if n.Kind == "battle" && n.Stage == stage.ID {
						dd.Campaign.Start = n.ID
					}
				}
				e := New(&dd, 5)
				require.Equal(t, si, e.state.Stage)
				applyOK(t, e, Command{Kind: "start"})
				plan, err := e.DeploymentPlan()
				require.NoError(t, err)
				for _, c := range plan {
					applyOK(t, e, c)
				}
				// clear the field of everyone else, so events that wait for a clearing (or for time) happen
				for _, id := range keys(e.unitIDs) {
					if u := e.unit(id); u.Faction == "enemy" && u.Character != duel.Enemy && u.Character != duel.Ally {
						u.HP = 0
					}
				}
				for i := 0; i < 6 && e.Observe().Phase == "battle"; i++ {
					applyOK(t, e, Command{Kind: "end"})
				}
				if e.Observe().Phase != "battle" {
					t.Skip("the battle was decided by the clearing alone")
				}
				a, b := e.unit(duel.Ally), e.unit(duel.Enemy)
				require.NotNil(t, a, "the ally of the duel never reaches the field")
				require.NotNil(t, b, "the enemy of the duel never reaches the field")
				if a.HP <= 0 || b.HP <= 0 || e.state.Executed["duel-"+duel.ID] {
					t.Skip("the duel already happened on its own")
				}
				placed := false
				for _, off := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					x, y := b.X+off[0], b.Y+off[1]
					if o := e.occupied(x, y); o == nil && e.terrain(a, x, y).Cost > 0 && x >= 0 && y >= 0 && x < stage.Width && y < stage.Height {
						a.X, a.Y, placed = x, y, true
						break
					}
				}
				require.True(t, placed, "no free cell next to the enemy")
				applyOK(t, e, Command{Kind: "end"})
				assert.True(t, e.state.Executed["duel-"+duel.ID], "the officers met and the duel did not start")
				for who, result := range map[string]string{duel.Ally: duel.AllyResult, duel.Enemy: duel.EnemyResult} {
					u := e.unit(who)
					if result != "stay" {
						assert.Zero(t, u.HP, who+" leaves the field")
					} else {
						assert.Positive(t, u.HP, who+" stays")
					}
				}
			})
		}
	}
}
