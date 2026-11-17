package gui

import (
	"image"
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// This audit records current behavior, including findings that are not acceptance criteria.
func TestAuditDeniedSound(t *testing.T) {
	if os.Getenv("SRPG_SFX_AUDIT") != "1" {
		t.Skip("set SRPG_SFX_AUDIT=1 to audit feedback sounds")
	}
	trace, err := os.CreateTemp(t.TempDir(), "sfx")
	require.NoError(t, err)
	oldTrace, oldCursor := sfxTrace, cursorPos
	sfxTrace = trace
	t.Cleanup(func() { sfxTrace, cursorPos = oldTrace, oldCursor; trace.Close() })
	read := func(run func()) string {
		require.NoError(t, trace.Truncate(0))
		_, err := trace.Seek(0, 0)
		require.NoError(t, err)
		run()
		b, err := os.ReadFile(trace.Name())
		require.NoError(t, err)
		return strings.TrimSpace(string(b))
	}

	t.Run("disabled menu click", func(t *testing.T) {
		g := &Game{buttons: []button{{rect: image.Rect(0, 0, 20, 20), click: func() {}, denied: true}}}
		cursorPos = func() (int, int) { return 10, 10 }
		got := read(g.leftClick)
		assert.Contains(t, got, "ui-denied")
		t.Logf("observed cues: %q", got)
	})
	t.Run("confirmation and modal close", func(t *testing.T) {
		g := &Game{S: &session.Session{}}
		require.True(t, g.askWith("출진하시겠습니까?", "예", "아니오", func() {}))
		t.Logf("confirmation yes: %q", read(g.acceptConfirmation))
		for _, kind := range []string{"objective", "levelup", "promotion", "trait", "acquire"} {
			g.modals = []*modal{{kind: kind}}
			t.Logf("%s acknowledgement: %q", kind, read(g.closeModal))
		}
	})

	d, err := NewDemo("../..", DemoOptions{Name: "explore", Stage: "start:c01-walk"})
	require.NoError(t, err)
	g := d.g
	t.Cleanup(g.Close)
	base := g.S.Engine.Snapshot()
	base.Learning = nil

	t.Run("every explore area and direction", func(t *testing.T) {
		areas, records, exits := 0, 0, 0
		for _, node := range g.S.Data.Nodes {
			if node.Kind != "explore" {
				continue
			}
			x := g.S.Data.Explores[node.Explore]
			for _, area := range sorted(x.Areas) {
				a := x.Areas[area]
				t.Run(node.Explore+"/"+area, func(t *testing.T) {
					s := base
					s.Node, s.Explore = node.ID, node.ID
					s.Spot = content.Spot{Area: area, At: a.At, Face: "NE"}
					g.S.Engine, err = core.Restore(g.S.Data, s)
					require.NoError(t, err)
					g.scene, g.explore, g.toasts = nil, nil, nil
					g.ensureExplore(g.S.View())
					require.Empty(t, g.toasts)
					areas++
					for _, face := range directions {
						g.scene.actors[g.explore.hero].dir = face
						g.explore.walked = true
						assert.Empty(t, read(g.reportSpot), face)
						assert.Empty(t, g.toasts, face)
						assert.Equal(t, face, g.S.View().Spot.Face)
						records++
					}
					for _, exit := range a.Exits {
						for _, face := range directions {
							s := base
							s.Flags = maps.Clone(base.Flags)
							s.Node, s.Explore = node.ID, node.ID
							s.Spot = content.Spot{Area: area, At: exit.At, Face: face}
							for _, cond := range exit.If {
								c, err := content.ParseCond(cond)
								require.NoError(t, err)
								require.Equal(t, "flag", c.Op, "extend the audit for new condition kinds")
								s.Flags[c.Arg] = c.N
								if c.Not {
									s.Flags[c.Arg]--
								}
							}
							g.S.Engine, err = core.Restore(g.S.Data, s)
							require.NoError(t, err)
							require.True(t, g.S.Engine.ExitOpen(exit))
							to := content.Spot{At: exit.At, Face: face}
							if exit.To != "" {
								to = x.Arrival(exit.To, area)
							}
							var response session.Response
							got := read(func() {
								response = g.send(session.Request{Op: "command", Command: core.Command{Kind: "walk", Option: to.Area, X: to.At[0], Y: to.At[1], Target: to.Face}})
							})
							require.True(t, response.OK, "%s/%s %s: %s", area, exit.Name, face, response.Error)
							assert.NotContains(t, got, "ui-denied")
							exits++
						}
					}
				})
			}
		}
		t.Logf("areas=%d direction records=%d exit commands=%d", areas, records, exits)
	})
	t.Run("all sortie questions on reference screens", func(t *testing.T) {
		old := g.S.Settings.UIScale
		t.Cleanup(func() { g.S.Settings.UIScale = old; g.fit(Width, Height) })
		count := 0
		for _, screen := range []struct{ w, h, scale int }{{1280, 850, 100}, {2400, 1080, 130}, {2400, 1080, 160}, {2048, 1536, 130}} {
			g.S.Settings.UIScale = screen.scale
			g.fit(float64(screen.w), float64(screen.h))
			for file, x := range g.S.Data.Explores {
				for area, a := range x.Areas {
					for person, p := range a.People {
						if p.Sortie == nil {
							continue
						}
						_, _, fits := g.confirmLayout(p.Sortie.Ask, uiW, uiH)
						assert.True(t, fits, "%s/%s/%s %dx%d %d%%", file, area, person, screen.w, screen.h, screen.scale)
						count++
					}
				}
			}
		}
		t.Logf("sortie question layouts=%d", count)
	})
	t.Run("every duel viewer entry", func(t *testing.T) {
		count := 0
		for _, stage := range g.S.Data.Stages {
			for _, duel := range stage.Duels {
				require.NoError(t, g.OpenDuel(duel.ID, "", false))
				count++
			}
		}
		g.gallery = nil
		t.Logf("duel viewer entries=%d", count)
	})
}
