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
	case "since": // the event has fired; its N rounds pass
		key := "battle-" + e.data.Stages[e.state.Stage].ID + "-" + c.Arg
		if rounds := e.state.Fired[key]; len(rounds) > 0 {
			e.state.Round = max(e.state.Round, rounds[len(rounds)-1]+c.N)
		}
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

// endTurn ends the side's turn, first closing the trait windows the officers who levelled up opened.
func endTurn(t *testing.T, e *Engine) {
	for len(e.state.Learning) > 0 {
		applyOK(t, e, Command{Kind: "learn_close", Actor: e.state.Learning[0]})
	}
	applyOK(t, e, Command{Kind: "end"})
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
		endTurn(t, e)
	}
	for i := 0; i < 14 && e.Observe().Phase == "battle"; i++ {
		satisfy(t, e, e.battleGoals().Victory[0].If) // an event may have changed the goals
		if bonus {
			for _, o := range e.battleGoals().Sub {
				satisfy(t, e, o.If)
			}
		}
		endTurn(t, e)
	}
	require.Equal(t, "result", e.Observe().Phase, "%s was not won (round %d, victory %v)", stage.ID, e.state.Round, e.battleGoals().Victory)
	require.Equal(t, "victory", e.Observe().Result, stage.ID)
}

// playCampaign walks the whole story: pages through the lines, takes the option named prefer where a choice offers it
// and otherwise the first, wins every battle, and reports which stages it fought and what the party ended up with.
func playCampaign(t *testing.T, bonus bool, prefer string) (*Engine, []string) {
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
			pick := options[0]
			for _, k := range options { // the first that does not lead to the hero's fall
				if !falls(d, o.Dialogue.Choices[k]) {
					pick = k
					break
				}
			}
			if _, ok := o.Dialogue.Choices[prefer]; ok {
				pick = prefer
			}
			applyOK(t, e, Command{Kind: "choose", Option: pick})
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

// falls reports whether the story from node id runs straight, without another choice, into an end node with a
// Text: the hero falls there.
func falls(d *content.Data, id string) bool {
	for i := 0; i < 50 && id != ""; i++ {
		n := d.Node(id)
		switch n.Kind {
		case "end":
			return n.Text != ""
		case "choice", "battle", "explore", "back":
			return false
		}
		if n.Next != "" {
			id = n.Next
		} else {
			id = d.After(id)
		}
	}
	return false
}

func executed(e *Engine, key string) bool { return e.state.Executed[key] }

func TestCampaignPlaysThroughWithEveryBonusObjective(t *testing.T) {
	e, fought := playCampaign(t, true, "빠른 길")
	assert.Equal(t, []string{"B01", "B02", "B03", "B04", "B05", "B06", "B07", "B08", "B09", "B10", "B11", "B12", "B13", "B14", "B15", "B16", "B17", "B18", "B19", "B20", "B21", "B22", "B23", "B24", "B25", "B26", "B27"}, fought)
	assert.Equal(t, 1, e.state.Flags["flag-mode-yanyi"], "the fast road at Longzhong is the romance")
	assert.Zero(t, e.state.Flags["flag-mode-virtual"])
	assert.Nil(t, e.char("xushu"), "Xu Shu is taken after Changban")
	assert.Positive(t, e.state.Warehouse["mirror_armor"], "driving Yue Jin off at Runan wins his armour")
	assert.Nil(t, e.char("liupi"), "Liu Pi, kept alive at Runan, parts at the south road instead of joining")
	assert.Positive(t, e.state.Inventory["insu_high"], "Liu Pi leaves his seal behind")
	assert.Equal(t, 1, e.state.Flags["flag-pengcheng-carts"])
	assert.Positive(t, e.state.Inventory["barley"], "Tao Qian shares the grain from the carts")
	assert.Equal(t, 1, e.state.Flags["flag-sunjian-alive"])
	assert.Equal(t, 1, e.state.Flags["flag-bianshui-generals"])
	assert.Equal(t, 1, e.state.Flags["flag-bianshui-lubu"])
	assert.Positive(t, e.state.Warehouse["tangni_armor"], "Lu Bu's armour is the reward for driving him off")
	assert.Equal(t, 1, e.state.Flags["flag-julu-clear"])
	for _, flag := range []string{"flag-zhugexuan", "flag-gaoshun"} {
		assert.Equal(t, 1, e.state.Flags[flag], flag)
	}
	for _, item := range []string{"fire_manual", "gujeong_blade", "jueying", "baima", "sanlue", "medicine_pouch"} {
		assert.Positive(t, e.state.Warehouse[item], item+" is handed over to the warehouse")
	}
	for _, id := range []string{"jianyong", "fangong", "sunqian", "gaoshun"} {
		assert.NotNil(t, e.char(id), id+" joins the party")
	}
	assert.Nil(t, e.char("fanya"), "Fan A finds Zhang Fei in the hills but goes back to Xuzhou at once")
	assert.NotNil(t, e.char("chendao"), "Chen Dao joins at the Mangdang village")
	assert.NotNil(t, e.char("zhaoyun"), "Zhao Yun, wandering since Gongsun Zan fell, joins on the flight from Yuan Shao")
	for _, id := range []string{"guanyu", "guanping", "zhoucang", "chenzhen"} {
		assert.NotNil(t, e.char(id), id+" comes with Guan Yu to Mangdang and joins in the battle against Cai Yang")
	}
	assert.NotNil(t, e.char("mizhu"), "Mi Zhu finds Liu Bei again at Yuan Shao's camp")
	assert.NotNil(t, e.char("mifang"), "Mi Fang comes with his brother")
	assert.True(t, executed(e, "battle-B10-vanguard-clear"), "a quick win at Julu keeps Yan Liang away from Jiejiao")
	assert.False(t, executed(e, "battle-B10-vanguard-strong"))
}

func TestCampaignPlaysThroughWithNoBonusObjective(t *testing.T) {
	e, fought := playCampaign(t, false, "느린 길")
	assert.Equal(t, []string{"B22", "B23", "B24", "B25V", "B26V", "B27V"}, fought[21:])
	assert.Equal(t, 1, e.state.Flags["flag-mode-virtual"], "Gao Shun, the Zhuge Xuan flag and the Guan Yu–Mi Fang peace open the slow road")
	assert.Zero(t, e.state.Flags["flag-mode-yanyi"])
	assert.NotNil(t, e.char("xushu"), "Xu Shu stays in the other telling")
	assert.Zero(t, e.state.Flags["flag-julu-clear"])
	assert.True(t, executed(e, "battle-B10-vanguard-strong"), "Yan Liang comes back at Jiejiao")
	assert.False(t, executed(e, "battle-B10-vanguard-clear"))
	assert.Positive(t, e.state.Warehouse["fire_manual"], "Huangfu Song hands the manual over whoever broke the siege by fire")
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
				// an ally the stage fields itself (위연 at 강릉) is not sent out
				npc := slices.ContainsFunc(stage.Party, func(sp content.Spawn) bool { return sp.Character == duel.Ally })
				if !npc && !slices.Contains(dd.Campaign.Roster, duel.Ally) { // an officer who joins later stands in for the start
					dd.Campaign.Roster = append(slices.Clone(dd.Campaign.Roster), duel.Ally)
				}
				for _, n := range d.Nodes {
					if n.Kind == "battle" && n.Stage == stage.ID {
						dd.Campaign.Start = n.ID
					}
				}
				e := New(&dd, 5)
				e.unchecked = true // the officer of the duel is sent out whether or not he can be
				require.Equal(t, si, e.state.Stage)
				if !npc && !slices.Contains(e.state.Deployment, duel.Ally) { // not a required officer: in for the last of the sortie when it is full
					if len(e.state.Deployment) >= stage.Max {
						e.state.Deployment = e.state.Deployment[:stage.Max-1]
					}
					e.state.Deployment = append(e.state.Deployment, duel.Ally)
				}
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
					endTurn(t, e)
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
				satisfy(t, e, duel.If) // a duel that waits for its moment (관우 and 조인 from round 13)
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
	e.unchecked = true // the officers are set down on cells the rules do not allow
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

func TestSwornBrothersJoinWhenTheOathEnds(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e := New(d, 1)
	assert.Equal(t, []string{"liubei"}, keys(e.charIDs), "only Liu Bei starts the walk")
	assert.Equal(t, []string{"liubei"}, e.state.Deployment)

	dd := *d
	dd.Campaign.Start = "c01-battle"
	e = New(&dd, 1)
	assert.ElementsMatch(t, []string{"liubei", "guanyu", "zhangfei"}, keys(e.charIDs), "after the oath they are in the party")
	assert.Equal(t, d.Rules.StageLevel(1), e.char("guanyu").Level)
}

func TestOfficersJoinInTheClassTheirLevelReaches(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	t.Run("Zhao Yun joins Jiejiao at its level, still light cavalry: short of the first promotion", func(t *testing.T) {
		dd := *d
		dd.Campaign.Start = "c10-battle"
		e := New(&dd, 1)
		zy := e.char("zhaoyun")
		require.NotNil(t, zy)
		assert.Equal(t, d.Rules.StageLevel(d.StageOrder("B10")), zy.Level)
		require.Less(t, zy.Level, d.Rules.PromoteLevels[0])
		assert.Equal(t, "light_cavalry", zy.Class)
	})
	t.Run("the difficulty shifts the join level as it shifts the enemy; the flat debug curve does not", func(t *testing.T) {
		dd := *d
		dd.Campaign.Start = "c10-battle"
		for difficulty, level := range map[string]int{"easy": 13, "normal": 17, "hard": 21, "flat_easy": 12, "flat_normal": 12, "flat_hard": 12} {
			e, err := NewGame(&dd, 1, difficulty)
			require.NoError(t, err)
			assert.Equal(t, level, e.char("zhaoyun").Level, difficulty)
		}
	})
	t.Run("a join between battles is at the last battle's level", func(t *testing.T) {
		assert.Equal(t, d.StageOrder("B13"), d.LastBattleOrder("c14-d16"))
	})
	t.Run("the data names the final class and a level steps back along its line", func(t *testing.T) {
		for _, c := range []struct {
			def   string
			level int
			want  string
		}{
			{"gongsunzan", 19, "light_cavalry"}, {"gongsunzan", 20, "heavy_cavalry"}, {"gongsunzan", 39, "heavy_cavalry"}, {"gongsunzan", 40, "royal_guard"},
			{"xiahouyuan", 20, "horse_archer"}, {"xiahouyuan", 40, "horse_crossbow"}, {"zhoucang", 14, "bandit"},
		} {
			assert.Equal(t, c.want, d.PromotedAt(c.def, d.Characters[c.def].Class, c.level), "%s at %d", c.def, c.level)
		}
	})
	t.Run("Liu Bei starts as light infantry though his line ends in the guards", func(t *testing.T) {
		require.Equal(t, "guards", d.Characters["liubei"].Class)
		assert.Equal(t, "light_infantry", New(d, 1).char("liubei").Class)
	})
	t.Run("Zhou Cang joins at 망탕산 on the bandit line, in the robe his class wears", func(t *testing.T) {
		e := New(d, 1)
		o := e.welcome("zhoucang", d.StageOrder("B20"))
		require.GreaterOrEqual(t, o.Level, d.Rules.PromoteLevels[0])
		require.Less(t, o.Level, d.Rules.PromoteLevels[1])
		assert.Equal(t, "ruffian", o.Class)
		assert.True(t, d.Carries(d.Classes[o.Class], nil, d.Equipment[o.Equipment["armor"]]), o.Equipment["armor"])
	})
}
