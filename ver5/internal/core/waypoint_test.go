package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestNewAt(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	ways := d.Waypoints()
	require.NotEmpty(t, ways)

	t.Run("every waypoint starts where it names", func(t *testing.T) {
		for _, w := range ways {
			e, err := NewAt(d, 1, w.Node, "")
			require.NoError(t, err, w.Node)
			o := e.Observe()
			level := d.Rules.StageLevel(d.NextBattleOrder(w.Node))
			for _, c := range o.Characters {
				assert.GreaterOrEqual(t, c.Level, level, "%s %s", w.Node, c.ID)
			}
			for id := range d.Items {
				assert.GreaterOrEqual(t, o.Inventory[id], waypointStock, "%s %s", w.Node, id)
			}
			if w.Stage != "" {
				assert.Equal(t, "scenario", o.Phase, w.Node)
				assert.Equal(t, w.Stage, d.Stages[o.Stage].ID, w.Node)
			}
			_, err = Restore(d, e.Snapshot())
			assert.NoError(t, err, w.Node)
		}
	})

	t.Run("the story before counts as played", func(t *testing.T) {
		last := ways[len(ways)-1]
		var all []string
		for _, f := range d.StoryFlags() {
			all = append(all, f.Flag)
		}
		e, err := NewAtWith(d, 1, last.Node, "", Start{Flags: all}) // 고순 stays only if he lived through his battle
		require.NoError(t, err)
		s := e.Snapshot()
		leads := d.Leads(last.Node) // the last waypoint is of the other telling (가상 27v장): 연의 25~27장 are not played
		for _, w := range ways {
			if w.Stage != "" && w.Node != last.Node && leads[w.Node] {
				assert.True(t, s.Executed["won-"+w.Stage], w.Stage)
			}
		}
		// in story order, the last join or leave of each officer before the waypoint says whether they are with the party
		with := map[string]bool{}
		for _, n := range d.Nodes {
			if n.ID == last.Node {
				break
			}
			if ci := d.ChapterIndex(n.ID); ci >= 0 {
				nodes := d.Chapters[ci].Nodes
				if slices.Contains(nodes, n.ID) && !leads[n.ID] || !slices.ContainsFunc(nodes, func(id string) bool { return leads[id] }) {
					continue // a node off the route, or an explore event of a chapter off it
				}
			}
			switch {
			case n.Kind == "leave":
				with[n.Character] = false
			case n.Kind == "join":
				with[n.Character] = true
			case n.Kind == "battle": // officers who join on the field
				for _, ev := range d.Stages[d.StageIndex(n.Stage)].Events {
					for _, a := range ev.Do {
						for _, sp := range a.Spawns {
							if a.Kind == "join" {
								with[sp.Character] = true
							}
						}
					}
				}
			}
		}
		for id, in := range with {
			if in {
				assert.NotNil(t, e.char(id), id)
			} else {
				assert.Nil(t, e.char(id), id+" has left")
			}
		}
	})

	t.Run("officers who join on the field are with the party after the battle", func(t *testing.T) {
		first := func(chapter int) string {
			for _, w := range ways {
				if w.Chapter == chapter && w.Stage == "" {
					return w.Node
				}
			}
			t.Fatalf("no waypoint opens chapter %d", chapter)
			return ""
		}
		for chapter, ids := range map[int][]string{
			20: {"zhaoyun"},                                   // B19
			21: {"zhaoyun", "guanyu", "guanping", "zhoucang"}, // B20 brings 관우's party back
		} {
			e, err := NewAt(d, 1, first(chapter), "")
			require.NoError(t, err)
			for _, id := range ids {
				assert.NotNil(t, e.char(id), "%s in chapter %d", id, chapter)
			}
		}
	})

	t.Run("officers wear the pieces the story has handed them", func(t *testing.T) {
		at := func(chapter int) *Engine {
			for _, w := range ways {
				if w.Chapter == chapter && w.Stage == "" {
					e, err := NewAt(d, 1, w.Node, "")
					require.NoError(t, err)
					return e
				}
			}
			t.Fatalf("no waypoint opens chapter %d", chapter)
			return nil
		}
		e := at(3)
		assert.Equal(t, "fire_manual", e.char("liubei").Equipment["accessory"])
		assert.Equal(t, "qinglong_blade", e.char("guanyu").Equipment["weapon"])

		e = at(1)
		assert.NotEqual(t, "fire_manual", e.char("liubei").Equipment["accessory"], "화계서 comes after the second battle")
		assert.Positive(t, e.state.Warehouse["fire_manual"], "the pieces to come wait in the warehouse")
	})

	t.Run("officers are promoted as far as the level allows", func(t *testing.T) {
		last := ways[len(ways)-1]
		e, err := NewAt(d, 1, last.Node, "")
		require.NoError(t, err)
		promoted := 0
		for _, id := range keys(e.charIDs) {
			o := e.char(id)
			assert.Equal(t, e.promotedAt(o.Def, o.Class, o.Level), o.Class, id)
			if d.Classes[o.Class].Tier > 0 {
				promoted++
			}
		}
		assert.Positive(t, promoted)
	})

	t.Run("the difficulty is kept", func(t *testing.T) {
		require.NotEmpty(t, d.Rules.Difficulties)
		id := d.Rules.Difficulties[len(d.Rules.Difficulties)-1].ID
		e, err := NewAt(d, 1, ways[0].Node, id)
		require.NoError(t, err)
		assert.Equal(t, id, e.Snapshot().Difficulty)
		_, err = NewAt(d, 1, ways[0].Node, "nope")
		assert.EqualError(t, err, "UnknownDifficulty")
	})

	t.Run("the battle flags told are kept and the branches follow them", func(t *testing.T) {
		at := func(node string, start Start) *Engine {
			e, err := NewAtWith(d, 1, node, "", start)
			require.NoError(t, err)
			return e
		}
		e := at("c16-battle", Start{})
		assert.Zero(t, e.state.Flags["flag-gaoshun"])
		assert.Nil(t, e.char("gaoshun"), "no flag: the branch that brings him is not taken")

		e = at("c16-battle", Start{Flags: []string{"flag-gaoshun"}})
		assert.Equal(t, 1, e.state.Flags["flag-gaoshun"])
		assert.NotNil(t, e.char("gaoshun"))

		e = at("c12-battle", Start{Flags: []string{"flag-gaoshun"}})
		assert.Zero(t, e.state.Flags["flag-gaoshun"], "the battle that leaves it is not fought yet")
	})

	t.Run("the level told is the party's", func(t *testing.T) {
		e, err := NewAtWith(d, 1, "c16-battle", "", Start{Level: 38})
		require.NoError(t, err)
		for _, c := range e.Observe().Characters {
			assert.GreaterOrEqual(t, c.Level, 38, c.ID)
		}
		_, err = NewAtWith(d, 1, "c16-battle", "", Start{Level: d.Rules.MaxLevel + 1})
		assert.EqualError(t, err, "BadLevel")
	})

	t.Run("an unknown node is refused", func(t *testing.T) {
		_, err := NewAt(d, 1, "nowhere", "")
		assert.EqualError(t, err, "UnknownWaypoint")
	})
}

// TestJumpIntoTheOtherTelling starts at the 가상 chapters, which the story reaches only by 「느린 길」 at 융중: the walk
// there takes 25장's branch into 25v장 and never plays 연의 25~27장 (서서 stays, 곽준 does not join, the mode is 가상).
func TestJumpIntoTheOtherTelling(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, node := range []string{"c25v-start", "c26v-start", "c27v-start"} {
		t.Run(node, func(t *testing.T) {
			e, err := NewAt(d, 1, node, "casual")
			require.NoError(t, err)
			assert.NotNil(t, e.char("xushu"), "Xu Shu leaves only after the romance's Changban")
			if node != "c27v-start" { // he joins after 강릉 (B26V) in this telling
				assert.Nil(t, e.char("huojun"), "Huo Jun joins only in the romance's chapter 27")
			}
			assert.False(t, e.state.Executed["won-B25"], "the romance's battles are not played")
			assert.False(t, e.state.Executed["won-B27"])
			if node != "c25v-start" {
				assert.Equal(t, 1, e.state.Flags["flag-mode-virtual"])
				assert.Zero(t, e.state.Flags["flag-mode-yanyi"])
				assert.True(t, e.state.Executed["won-B25V"])
			}
		})
	}
	e, err := NewAt(d, 1, "c27-start", "casual")
	require.NoError(t, err)
	assert.True(t, e.state.Executed["won-B26"], "the romance's own chapters are played on the way to its chapter 27")
	assert.False(t, e.state.Executed["won-B25V"])
}
