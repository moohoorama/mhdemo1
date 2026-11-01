package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
	"srpg/internal/scenemap"
	"srpg/internal/session"
)

// Every area of the story's walks is a scene map of format 2, and from where the hero arrives he can walk to each
// exit, next to each person (who may stand anywhere, people block nothing here) and next to each sign.
func TestExploreAreasCanBeWalked(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	for name, x := range g.S.Data.Explores {
		for id, a := range x.Areas {
			t.Run(name+"/"+id, func(t *testing.T) {
				art := g.assets.Maps[a.Map]
				require.True(t, art != nil && art.SceneMap != nil, "%s is no scene map of format 2", a.Map)
				w := walkGrid{walk: scenemap.Walkable(art.SceneMap, g.assets.SceneRules)}
				from := w.nearestOpen(a.At)
				reach := w.reachable(from)
				near := func(at [2]int, what string) {
					for _, d := range []int{2, 1} {
						for _, n := range []smallTile{{at[0] + d, at[1]}, {at[0] - d, at[1]}, {at[0], at[1] + d}, {at[0], at[1] - d}} {
							if _, ok := reach[n]; ok {
								return
							}
						}
					}
					t.Errorf("%s at %v cannot be walked up to", what, at)
				}
				for _, e := range a.Exits {
					_, ok := reach[w.nearestOpen(e.At)]
					assert.True(t, ok, "exit %s cannot be reached", e.Name)
				}
				for key, p := range a.People {
					near(p.At, key)
				}
				for _, s := range a.Signs {
					near(s.At, s.Name)
				}
			})
		}
	}
}

// A joining officer's window that comes up as the story goes back to the walk (조운 in chapter 10, when the Julu
// flag sends the branch straight on to the walk) keeps the line before it on screen: the walk is not laid out
// under it, or the two would take the scene from each other every frame and the window would never show.
func TestHeldStoryKeepsTheWalkAway(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	require.True(t, g.send(session.Request{Op: "jump", Seed: 1, Node: "c10-start"}).OK)
	st := g.S.Engine.Snapshot()
	st.Learning, st.Flags["flag-julu-clear"] = nil, 1
	g.S.Engine, err = core.Restore(g.S.Data, st)
	require.NoError(t, err)
	for g.S.View().Node != "c10-19" {
		require.True(t, g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}}).OK, g.S.View().Node)
	}
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
	require.True(t, g.S.Engine.Walking(), "조운 joins and the story goes on to the walk")
	require.NotNil(t, g.held, "his window holds 조운's last line")
	assert.False(t, g.exploring())
	g.held = nil // the screen lets go of the line once the window is closed
	assert.True(t, g.exploring())
}
