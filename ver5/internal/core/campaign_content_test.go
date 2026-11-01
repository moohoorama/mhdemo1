package core

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
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
	case "fled":
		for _, u := range e.matching(c.Arg) {
			u.HP = 0
			e.state.Executed["fled-"+e.data.Stages[e.state.Stage].ID+"-"+u.ID] = true
		}
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

// winStage sorties for the stage the engine stands at and wins it the way its first victory goal now asks;
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
		satisfy(t, e, e.battleGoals().Victory[0].If) // an event may have changed the goals
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
		case e.Walking():
			plan := e.ExploreNext()
			require.NotEmpty(t, plan, "the walk at %s is stuck in %s", o.Node, o.Spot.Area)
			for _, c := range plan {
				applyOK(t, e, c)
			}
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
	assert.Equal(t, []string{"B01", "B02", "B03", "B04", "B05", "B06", "B07", "B08", "B09", "B10", "B11", "B12"}, fought)
	assert.Equal(t, 1, e.state.Flags["flag-pengcheng-carts"])
	assert.Positive(t, e.state.Inventory["barley"], "Tao Qian shares the grain from the carts")
	assert.Equal(t, 1, e.state.Flags["flag-sunjian-alive"])
	assert.Equal(t, 1, e.state.Flags["flag-bianshui-generals"])
	assert.Equal(t, 1, e.state.Flags["flag-julu-clear"])
	for _, flag := range []string{"flag-beihai-taishici", "flag-zhugexuan", "flag-gaoshun"} {
		assert.Equal(t, 1, e.state.Flags[flag], flag)
	}
	for _, item := range []string{"gujeong_blade", "jueying", "baima"} {
		assert.Positive(t, e.state.Warehouse[item], item+" is handed over")
	}
	assert.Equal(t, "fire_manual", e.char("liubei").Equipment["accessory"], "Liu Bei takes up the manual at once")
	assert.Positive(t, e.state.Warehouse["tactics_primer"], "the primer goes to the warehouse")
	for _, id := range []string{"jianyong", "fangong", "zhaoyun"} {
		assert.NotNil(t, e.char(id), id+" joins the party")
	}
	assert.True(t, executed(e, "battle-B10-vanguard-clear"), "a quick win at Julu keeps Yan Liang away from Jiejiao")
	assert.False(t, executed(e, "battle-B10-vanguard-strong"))
}

func TestCampaignPlaysThroughWithNoBonusObjective(t *testing.T) {
	e, fought := playCampaign(t, false)
	assert.Len(t, fought, 12)
	assert.Zero(t, e.state.Flags["flag-julu-clear"])
	assert.True(t, executed(e, "battle-B10-vanguard-strong"), "Yan Liang comes back at Jiejiao")
	assert.False(t, executed(e, "battle-B10-vanguard-clear"))
	assert.Equal(t, "fire_manual", e.char("liubei").Equipment["accessory"], "Huangfu Song hands the manual to whoever broke the siege by fire")
}

// enterAreas stands one of the player's officers in every area of the stage that no victory goal names, the
// lord first, so events that wait for someone to reach a place (an ambush springing) happen.
func enterAreas(e *Engine, stage content.Stage) {
	won := map[int]bool{}
	at := regexp.MustCompile(`at:[^:,\]]+:(\d+)`)
	for _, g := range stage.Victory {
		for _, c := range g.If {
			for _, m := range at.FindAllStringSubmatch(c, -1) {
				n, _ := strconv.Atoi(m[1])
				won[n] = true
			}
		}
	}
	var officers []*Unit
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.Faction == "ally" && !u.AI && u.HP > 0 {
			officers = append(officers, u)
		}
	}
	slices.SortStableFunc(officers, func(a, b *Unit) int { return btoi(e.unitDef(b).Lord) - btoi(e.unitDef(a).Lord) })
	for _, a := range stage.Areas {
		if won[a.ID] || len(officers) == 0 {
			continue
		}
		u := officers[0]
		r := a.Rect
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if u != nil && e.occupied(x, y) == nil && e.terrain(u, x, y).Cost > 0 {
					u.X, u.Y = x, y
					officers, u = officers[1:], nil
				}
			}
		}
	}
}

func TestEveryCampaignDuelStartsWhenItsOfficersMeet(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for si, stage := range d.Stages {
		for _, duel := range stage.Duels {
			t.Run(duel.ID, func(t *testing.T) {
				dd := *d
				if !slices.Contains(dd.Campaign.Roster, duel.Ally) { // an officer who joins later stands in for the start
					dd.Campaign.Roster = append(slices.Clone(dd.Campaign.Roster), duel.Ally)
				}
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
				enterAreas(e, stage)
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
				for _, id := range keys(e.unitIDs) {
					if u := e.unit(id); u.Faction == "enemy" && u.Character != duel.Enemy {
						u.HP = 0
					}
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

// TestChangsheSiegeUnfolds plays the B02 story beat by beat with no move by the player: the
// brothers set the southeast fence alight, the turbans pour out of its gate into their ambush, Cao
// Cao and Sun Jian fire the northeast and southwest fences, the castle gates open, and the battle is
// won once Bo Cai falls.
func TestChangsheSiegeUnfolds(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	dd := *d
	dd.Campaign.Start = "c02-battle"
	e := New(&dd, 3)
	applyOK(t, e, Command{Kind: "start"})
	require.Equal(t, "battle", e.state.Phase, "the brothers sortie with no deployment screen")
	require.True(t, executed(e, "battle-B02-night-raid"), "the whole night attack plays before the first turn")
	assert.Equal(t, byte('F'), e.Cell(16, 20).State, "the southeast fence burns")
	assert.Equal(t, byte('F'), e.Cell(13, 6).State, "Cao Cao fires the northeast fence")
	assert.Equal(t, byte('F'), e.Cell(5, 23).State, "Sun Jian fires the southwest fence")
	assert.Equal(t, byte('O'), e.Cell(6, 11).State, "the castle gate stands open")
	assert.NotNil(t, e.unit("caocao"))
	assert.NotNil(t, e.unit("zumao"))
	assert.GreaterOrEqual(t, e.matching("so1")[0].X, 17, "the soldiers by the gate ran out of it")
	for _, id := range []string{"bocai", "zhangbao"} {
		assert.Equal(t, -20, e.unit(id).Status["attack"].Value, "the turbans lose heart when they see they are surrounded")
		assert.Equal(t, -20, e.unit(id).Status["defense"].Value)
	}
	assert.Equal(t, "assault", e.unit("huangfusong").Policy)
	assert.Equal(t, "hold", e.unit("bocai").Policy)
	for e.state.Round < 3 {
		applyOK(t, e, Command{Kind: "end"})
	}
	assert.Equal(t, "march", e.unit("bocai").Policy, "Bo Cai comes out in the third round")
	e.unit("bocai").HP = 0
	applyOK(t, e, Command{Kind: "end"})
	assert.Equal(t, "victory", e.Observe().Result)
}

func TestYingchuanGuanYuAttackDoesNotSummonLiuBeiDuel(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	dd := *d
	dd.Campaign.Start = "c02-battle"
	e := New(&dd, 3)
	applyOK(t, e, Command{Kind: "start"})

	bocai, guanyu, liubei := e.unit("bocai"), e.unit("guanyu"), e.unit("liubei")
	require.NotNil(t, bocai)
	require.NotNil(t, guanyu)
	require.NotNil(t, liubei)
	bocai.X, bocai.Y = 2, 2
	guanyu.X, guanyu.Y = 2, 3
	liubei.X, liubei.Y = 3, 3
	bocai.HP = e.stats(bocai).MaxHP * 69 / 100

	r := applyOK(t, e, Command{Kind: "attack", Actor: "guanyu", Target: "bocai"})
	assert.Empty(t, duelsIn(r), "Guanyu's attack cannot start Liu Bei's duel from two cells away")
	assert.False(t, e.state.Executed["duel-yingchuan-liubei-bocai"])
	assert.Positive(t, e.unit("bocai").HP, "Bo Cai remains on the field after surviving the attack")

	liubei = e.unit("liubei")
	liubei.X, liubei.Y = 3, 2
	r = applyOK(t, e, Command{Kind: "wait", Actor: "liubei"})
	require.Len(t, duelsIn(r), 1, "the duel starts when Liu Bei meets Bo Cai")
	assert.Equal(t, "yingchuan-liubei-bocai", duelsIn(r)[0].DuelID)
}
