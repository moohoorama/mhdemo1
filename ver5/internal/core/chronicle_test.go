package core

import (
	"encoding/json"
	"maps"
	"slices"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// chroniclePlay walks the story from where e stands as playCampaign does (every battle won, option prefer taken
// where a choice offers it), but with the battles' losses told: kill maps a stage id to the conditions
// ("defeated:zhugexuan") to force before the stage is won. reached is told every waypoint the story enters, with
// the state it settled in, the way the session keeps a chronicle point.
func chroniclePlay(t *testing.T, d *content.Data, e *Engine, bonus bool, prefer string, kill map[string][]string, reached func(node, route string, s State)) {
	told := func(from string) {
		if node := e.Reached(from); node != "" {
			reached(node, e.Route(node), e.Snapshot())
		}
	}
	told("")
	apply := func(c Command) {
		from := e.state.Node
		applyOK(t, e, c)
		told(from)
	}
	for step := 0; step < 2000; step++ {
		for len(e.state.Learning) > 0 {
			apply(Command{Kind: "learn_close", Actor: e.state.Learning[0]})
		}
		o := e.Observe()
		switch {
		case o.Phase == "complete":
			return
		case o.Dialogue.Kind == "choice":
			pick := prefer
			if _, ok := o.Dialogue.Choices[pick]; !ok {
				options := slices.Sorted(maps.Keys(o.Dialogue.Choices))
				pick = options[0]
				for _, k := range options {
					if !falls(d, o.Dialogue.Choices[k]) {
						pick = k
						break
					}
				}
			}
			apply(Command{Kind: "choose", Option: pick})
		case e.Walking():
			plan := e.ExploreNext()
			require.NotEmpty(t, plan, "the walk at %s is stuck", o.Node)
			for _, c := range plan {
				apply(c)
			}
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			stage := d.Stages[o.Stage].ID
			applyOK(t, e, Command{Kind: "start"})
			plan, err := e.DeploymentPlan()
			require.NoError(t, err)
			for _, c := range plan {
				applyOK(t, e, c)
			}
			for i := 0; i < 10 && e.Observe().Phase == "battle" && e.state.Round < 5; i++ {
				endTurn(t, e)
			}
			for i := 0; i < 14 && e.Observe().Phase == "battle"; i++ {
				satisfy(t, e, kill[stage])
				satisfy(t, e, e.battleGoals().Victory[0].If)
				if bonus {
					for _, s := range e.battleGoals().Sub {
						satisfy(t, e, s.If)
					}
				}
				endTurn(t, e)
			}
			require.Equal(t, "victory", e.Observe().Result, stage)
			apply(Command{Kind: "continue"})
		case o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate":
			apply(Command{Kind: "next"})
		default:
			t.Fatalf("stopped at %s (%s, %s)", o.Node, o.Phase, o.Dialogue.Kind)
		}
	}
	t.Fatal("the story did not end")
}

// TestEveryWaypointIsReachedAndKept plays the story along both modes and checks that every chapter start and every
// battle is entered one by one, in story order, with a state that loads again and stands at the waypoint: what the
// chronicle keeps (session.record) is told by Engine.Entered, so a waypoint it skipped would never be listed.
func TestEveryWaypointIsReachedAndKept(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	order := map[string]int{}
	for i, w := range d.Waypoints() {
		order[w.Node] = i
	}
	seen := map[string]bool{}
	for _, mode := range []struct {
		name, prefer string
		bonus        bool
	}{{"virtual", "느린 길", true}, {"yanyi", "빠른 길", false}} {
		t.Run(mode.name, func(t *testing.T) {
			var nodes []string
			chroniclePlay(t, d, New(d, 3), mode.bonus, mode.prefer, nil, func(node, _ string, s State) {
				nodes = append(nodes, node)
				seen[node] = true
				w := d.Waypoints()[order[node]]
				if w.Stage != "" {
					assert.Equal(t, node, s.Node, "a battle waypoint is kept standing at the battle")
					assert.Equal(t, "scenario", s.Phase, node)
				} else {
					assert.Equal(t, w.Chapter, d.ChapterNumber(s.Node), "a chapter start is kept inside its chapter: %s at %s", node, s.Node)
				}
				_, err := Restore(d, s)
				assert.NoError(t, err, node)
			})
			assert.True(t, sort.SliceIsSorted(nodes, func(i, j int) bool { return order[nodes[i]] < order[nodes[j]] }), "waypoints come in story order: %v", nodes)
			assert.Equal(t, len(nodes), len(slices.Compact(slices.Clone(nodes))), "no waypoint twice")
			assert.Equal(t, d.Waypoints()[0].Node, nodes[0], "the first chapter's start is reached at the new game")
		})
	}
	for _, w := range d.Waypoints() {
		assert.True(t, seen[w.Node], "%s is never reached on either path", w.Node)
	}
}

// TestB12PointReplaysTheModeBranch plays the story again from the state kept on entering 서주 구원전 (B12):
// the same point leads to 가상 when 제갈현 and 고순 live and 22장 reconciles, and to 연의 when either falls; the
// kept state itself stays as it was through every replay.
func TestB12PointReplaysTheModeBranch(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	var before State
	chroniclePlay(t, d, New(d, 3), true, "느린 길", nil, func(node, _ string, s State) {
		if node == "c12-battle" {
			before = s
		}
	})
	require.Equal(t, "c12-battle", before.Node)
	kept, _ := json.Marshal(before)
	require.NotContains(t, before.Flags, "flag-zhugexuan", "the point is from before the battle")

	replay := func(t *testing.T, bonus bool, prefer string, kill []string) State {
		e, err := Restore(d, before)
		require.NoError(t, err)
		var last State
		chroniclePlay(t, d, e, bonus, prefer, map[string][]string{"B12": kill}, func(string, string, State) {})
		last = e.Snapshot()
		return last
	}
	t.Run("both live: 느린 길 is 가상", func(t *testing.T) {
		end := replay(t, true, "느린 길", nil)
		assert.Equal(t, 1, end.Flags["flag-zhugexuan"])
		assert.Equal(t, 1, end.Flags["flag-guanyu-mifang"])
		assert.Equal(t, 1, end.Flags["flag-mode-virtual"])
		assert.Zero(t, end.Flags["flag-mode-yanyi"])
	})
	t.Run("both live: 빠른 길 is 연의", func(t *testing.T) {
		end := replay(t, true, "빠른 길", nil)
		assert.Equal(t, 1, end.Flags["flag-mode-yanyi"])
		assert.Zero(t, end.Flags["flag-mode-virtual"])
	})
	t.Run("제갈현 falls: 연의 without the choice", func(t *testing.T) {
		end := replay(t, false, "느린 길", []string{"defeated:zhugexuan"})
		assert.Zero(t, end.Flags["flag-zhugexuan"])
		assert.Equal(t, 1, end.Flags["flag-mode-yanyi"])
		assert.Zero(t, end.Flags["flag-mode-virtual"])
	})
	t.Run("고순 falls: 연의 without the choice", func(t *testing.T) {
		end := replay(t, false, "느린 길", []string{"defeated:gaoshun"})
		assert.Zero(t, end.Flags["flag-gaoshun"])
		assert.Equal(t, 1, end.Flags["flag-mode-yanyi"])
		assert.Zero(t, end.Flags["flag-mode-virtual"])
	})
	again, _ := json.Marshal(before)
	assert.JSONEq(t, string(kept), string(again), "the kept state is not changed by the replays")
}

// TestForkArmsFollowTheStory plays the story the ways the forks tell apart and checks the arm each waypoint is
// kept on (the waypoints on a fork are kept once per arm): the 거록 일기토 seen or not till 계교 is over, and after
// the 서주 전투 either both 고순 and 제갈현 gone, 고순 alone, or both alive, whose 느린 길 stays on its arm into
// 가상 and whose 빠른 길 joins 고순 alone's.
func TestForkArmsFollowTheStory(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	type play struct {
		bonus  bool
		prefer string
		kill   []string
	}
	routes := func(p play) map[string]string {
		got := map[string]string{}
		chroniclePlay(t, d, New(d, 3), p.bonus, p.prefer, map[string][]string{"B12": p.kill}, func(node, route string, _ State) { got[node] = route })
		return got
	}
	way := func(node string) content.Waypoint {
		for _, w := range d.Waypoints() {
			if w.Node == node {
				return w
			}
		}
		t.Fatalf("no waypoint %s", node)
		return content.Waypoint{}
	}
	onArm := func(t *testing.T, got map[string]string, fork, arm string) {
		n := 0
		for node, route := range got {
			if f := d.ForkAt(node); f != nil && f.ID == fork {
				n++
				assert.Equal(t, arm, route, "%s", node)
			}
		}
		assert.Positive(t, n, fork)
	}
	t.Run("일기토 seen: the first fork's arm until 계교 is over", func(t *testing.T) {
		got := routes(play{true, "느린 길", nil})
		assert.Equal(t, "seen", got["c10-battle"])
		assert.Empty(t, got["c11-start"], "after 계교 the story is shared again")
		assert.Empty(t, got["c09-battle"], "the fork starts after 거록")
	})
	t.Run("일기토 missed", func(t *testing.T) {
		assert.Equal(t, "unseen", routes(play{false, "빠른 길", nil})["c10-battle"])
	})
	t.Run("both alive, 느린 길: stays on its arm into 가상", func(t *testing.T) {
		got := routes(play{true, "느린 길", nil})
		onArm(t, got, "xuzhou", "both")
		assert.Equal(t, "both", got["c25v-battle"])
		assert.Empty(t, got["c12-battle"], "the fork starts after 서주")
		assert.NotContains(t, got, "c25-battle")
	})
	t.Run("both alive, 빠른 길: joins 고순 alone's arm after the choice", func(t *testing.T) {
		got := routes(play{true, "빠른 길", nil})
		assert.Equal(t, "both", got["c23-battle"])
		assert.Equal(t, "gaoshun", got["c24-battle"])
		assert.Equal(t, "gaoshun", got["c25-battle"])
		assert.Equal(t, "gaoshun", got["c27-redcliff-battle"])
		assert.False(t, way("c25-battle").Act == "2막-가상")
	})
	t.Run("고순 alone", func(t *testing.T) {
		got := routes(play{false, "느린 길", []string{"defeated:zhugexuan"}})
		onArm(t, got, "xuzhou", "gaoshun")
	})
	t.Run("고순 gone: 제갈현 alone goes the same way", func(t *testing.T) {
		onArm(t, routes(play{false, "느린 길", []string{"defeated:gaoshun"}}), "xuzhou", "none")
	})
}
