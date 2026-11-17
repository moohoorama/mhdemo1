package gui

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// TestEveryWalkSortieDraws sends the party on from every walk of the campaign through the screen — whatever the
// sortie brings up first (a granted horse's window, a joining officer, the sortie preparation) — and draws the
// frames that follow. A screen that still holds the walk's observation after the story has left the walk must draw
// without the walk (the 망탕산 sortie granted 적토마 and crashed on the walk that was gone).
func TestEveryWalkSortieDraws(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "scenenode", Stage: "c01-t1"})
	require.NoError(t, err)
	g := d.g
	cursorPos = func() (int, int) { return -1 << 20, -1 << 20 } // no hover over a building, which reads its pixels
	data := g.S.Data
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	frames := func(n int) {
		for range n {
			g.advance(1.0 / 60)
			g.Draw(dst)
		}
	}
	for _, w := range data.Waypoints() {
		n := data.Node(w.Node)
		if n.Kind != "explore" {
			continue
		}
		x := data.Explores[n.Explore]
		for _, area := range sorted(x.Areas) {
			for _, key := range sorted(x.Areas[area].People) {
				p := x.Areas[area].People[key]
				if p.Sortie == nil {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%s", w.Node, area, key), func(t *testing.T) {
					g.held, g.modals = nil, nil
					r := g.request(session.Request{Op: "jump", Seed: 1, Node: w.Node, Discard: true})
					require.True(t, r.OK, r.Error)
					for i := 0; i < 300 && !g.S.Engine.Walking(); i++ { // the walk's opening plays first
						g.modals = nil
						for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
							g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
						}
						if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}}); !r.OK {
							t.Logf("%s: %s", g.S.View().Node, r.Error)
						}
					}
					require.True(t, g.S.Engine.Walking(), "the walk opens after its opening, at %s", g.S.View().Node)
					st := g.S.Engine.Snapshot()
					st.Learning = nil
					st.Spot = content.Spot{Area: area, At: x.Areas[area].At, Face: "SE"}
					for _, c := range append(append([]string{}, p.If...), p.Sortie.If...) {
						if cond, err := content.ParseCond(c); err == nil && cond.Op == "flag" && !cond.Not {
							st.Flags[cond.Arg] = max(st.Flags[cond.Arg], cond.N)
						}
					}
					for _, ev := range p.Event { // a pending talk event would keep the sortie away
						for _, c := range ev.If {
							if cond, err := content.ParseCond(c); err == nil && cond.Op == "flag" && cond.Not {
								st.Flags[cond.Arg] = max(st.Flags[cond.Arg], cond.N)
							}
						}
					}
					g.S.Engine, err = core.Restore(data, st)
					require.NoError(t, err)
					frames(3)
					r = g.request(session.Request{Op: "command", Command: core.Command{Kind: "sortie", Target: key}})
					if !r.OK {
						t.Skipf("no sortie from %s here: %s", key, r.Error)
					}
					frames(30)
					for len(g.modals) > 0 {
						g.modals = g.modals[1:]
						frames(5)
					}
					frames(10)
					require.False(t, g.S.Engine.Walking(), "the party left the walk")
				})
			}
		}
	}
}

// TestStoryDrawsFromEveryWaypoint plays the story on from every waypoint through the screen, a frame drawn after
// every step and every window closed as a player would, until the next battle's preparation, a walk or the end:
// every story node, grant and join is drawn at least once on its way.
func TestStoryDrawsFromEveryWaypoint(t *testing.T) {
	if testing.Short() {
		t.Skip("plays the whole story")
	}
	d, err := NewDemo("../..", DemoOptions{Name: "scenenode", Stage: "c01-t1"})
	require.NoError(t, err)
	g := d.g
	cursorPos = func() (int, int) { return -1 << 20, -1 << 20 } // no hover over a building, which reads its pixels
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	for _, w := range g.S.Data.Waypoints() {
		t.Run(w.Node, func(t *testing.T) {
			g.held, g.modals = nil, nil
			r := g.request(session.Request{Op: "jump", Seed: 1, Node: w.Node, Discard: true})
			require.True(t, r.OK, r.Error)
			for i := 0; i < 400; i++ {
				g.advance(1.0 / 60)
				g.Draw(dst)
				if len(g.modals) > 0 {
					g.modals = g.modals[1:]
					continue
				}
				for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
					g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
				}
				o := g.S.View()
				if o.Phase != "scenario" || o.Dialogue.Kind == "battle" || g.S.Engine.Walking() {
					return
				}
				if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}}); !r.OK {
					t.Fatalf("%s: %s", o.Node, r.Error)
				}
			}
			t.Fatalf("the story did not reach a battle or a walk from %s, at %s", w.Node, g.S.View().Node)
		})
	}
}
