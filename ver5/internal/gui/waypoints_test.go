package gui

import (
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// waypointScreen opens the waypoint window of a game at a w×h screen and gives a draw that shows it again.
func waypointScreen(t *testing.T, sc image.Point) (*Game, func()) {
	t.Helper()
	setPointer(t)
	d, err := NewDemo("../..", DemoOptions{Name: "waypoints"})
	require.NoError(t, err)
	g := d.g
	g.fit(float64(sc.X), float64(sc.Y))
	cursorPos = func() (int, int) { return -1 << 20, -1 << 20 }
	dst := ebiten.NewImage(sc.X, sc.Y)
	t.Cleanup(dst.Deallocate)
	draw := func() {
		g.advance(1.0 / 60)
		g.Draw(dst)
	}
	draw()
	require.Equal(t, "waypoints", g.overlay())
	return g, draw
}

// tap lets a press go on the middle of r, as a player's tap does.
func tap(g *Game, r image.Rectangle) {
	at := r.Min.Add(r.Size().Div(2))
	pointer = touchPointer{}
	press(at.X, at.Y, true)
	g.updateWaypoints()
	release(at.X, at.Y)
	g.updateWaypoints()
}

// pressButtonAt presses the button at the middle of r as the game's click does.
func pressButtonAt(g *Game, r image.Rectangle) {
	at := r.Min.Add(r.Size().Div(2))
	pointer = touchPointer{}
	press(at.X, at.Y, true)
	g.leftClick()
	release(at.X, at.Y)
}

func buttonRect(t *testing.T, g *Game, label string) image.Rectangle {
	t.Helper()
	for _, b := range g.buttons {
		if b.label == label {
			return b.rect
		}
	}
	t.Fatalf("no button %q", label)
	return image.Rectangle{}
}

var waypointScreens = []struct {
	name string
	sc   image.Point
}{{"desktop", image.Pt(1280, 850)}, {"phone", image.Pt(2400, 1080)}}

func TestWaypointActTabs(t *testing.T) {
	for _, c := range waypointScreens {
		t.Run(c.name, func(t *testing.T) {
			g, draw := waypointScreen(t, c.sc)
			acts := g.S.Data.WaypointActs()
			require.Len(t, acts, 4)
			for _, act := range acts {
				var want []string
				for _, w := range g.S.Data.Waypoints() {
					if w.Act == act {
						want = append(want, w.Node)
					}
				}
				if g.waypointActFor(acts) != act {
					pressButtonAt(g, buttonRect(t, g, act))
				}
				g.waypoints.list.top = 0
				draw()
				assert.Equal(t, act, g.waypointActFor(acts))

				// scroll to the end of the group and tap the last row: it is the group's last waypoint
				g.waypoints.list.top = g.waypoints.list.end
				draw()
				require.NotEmpty(t, g.waypoints.list.rows, act)
				last := g.waypoints.list.rows[len(g.waypoints.list.rows)-1]
				assert.LessOrEqual(t, last.r.Max.Y, g.waypoints.list.view.Max.Y, "the last row is not cut by the list's edge")
				g.jumpNode = ""
				tap(g, last.r)
				assert.Equal(t, want[len(want)-1], g.jumpNode, act)
				assert.Equal(t, "difficulty", g.overlay(), "a pick asks the difficulty")
				g.closeOverlays()
				g.openOverlay("waypoints")
				draw()
			}
		})
	}
	t.Run("each act keeps its own scroll", func(t *testing.T) {
		g, draw := waypointScreen(t, image.Pt(1280, 850))
		g.waypoints.list.top = 200
		pressButtonAt(g, buttonRect(t, g, "2막"))
		draw()
		assert.Zero(t, g.waypoints.list.top)
		g.waypoints.list.top = 90
		pressButtonAt(g, buttonRect(t, g, "1막"))
		draw()
		assert.Equal(t, 200, g.waypoints.list.top)
		pressButtonAt(g, buttonRect(t, g, "2막"))
		draw()
		assert.Equal(t, 90, g.waypoints.list.top)
	})
}

func TestWaypointFlagsReachable(t *testing.T) {
	for _, c := range waypointScreens {
		t.Run(c.name, func(t *testing.T) {
			g, draw := waypointScreen(t, c.sc)
			flags := g.S.Data.StoryFlags()
			require.Greater(t, len(flags), 10)
			l := &g.waypoints.flags
			require.Positive(t, l.end, "the flags do not fit the panel, so it scrolls")
			require.NotEmpty(t, l.bar, "and shows its scrollbar")

			t.Run("scrolling reaches every row, and each row toggles its own flag", func(t *testing.T) {
				g.jumpFlags = nil
				seen := map[int]bool{}
				for top := 0; ; top += l.view.Dy() / 2 {
					l.top = min(top, l.end)
					draw()
					for _, r := range l.rows {
						assert.GreaterOrEqual(t, r.r.Min.Y, l.view.Min.Y)
						assert.LessOrEqual(t, r.r.Max.Y, l.view.Max.Y)
						if r.r.Min.Y == l.view.Min.Y && l.top > 0 || r.r.Max.Y == l.view.Max.Y && l.top < l.end {
							continue // cut by an edge the list goes on past: a later scroll shows it whole
						}
						if !seen[r.index] {
							seen[r.index] = true
							tap(g, r.r)
						}
					}
					if l.top == l.end {
						break
					}
				}
				for _, f := range flags {
					assert.True(t, g.jumpFlags[f.Flag], f.Flag)
				}
				assert.Len(t, seen, len(flags))
			})
			t.Run("모두 and 없음", func(t *testing.T) {
				draw()
				g.jumpFlags = nil
				pressButtonAt(g, buttonRect(t, g, "모두"))
				assert.Len(t, g.jumpFlags, len(flags))
				draw()
				pressButtonAt(g, buttonRect(t, g, "없음"))
				assert.Empty(t, g.jumpFlags)
			})
			t.Run("a drag on the flags scrolls and toggles nothing", func(t *testing.T) {
				g.jumpFlags, l.top = nil, 0
				draw()
				at := l.rows[1].r.Min.Add(l.rows[1].r.Size().Div(2))
				pointer = touchPointer{}
				press(at.X, at.Y, true)
				g.updateWaypoints()
				press(at.X, at.Y-120, false)
				g.updateWaypoints()
				release(at.X, at.Y-120)
				g.updateWaypoints()
				assert.Equal(t, 120, l.top)
				assert.Empty(t, g.jumpFlags)
			})
			t.Run("the scrollbar thumb drags the flags to the end without moving the list", func(t *testing.T) {
				l.top, g.waypoints.list.top = 0, 0
				draw()
				at := image.Pt(l.bar.Min.X+1, l.bar.Min.Y+4)
				pointer = touchPointer{}
				press(at.X, at.Y, true)
				g.updateWaypoints()
				press(at.X, l.bar.Max.Y+40, false)
				g.updateWaypoints()
				release(at.X, l.bar.Max.Y+40)
				g.updateWaypoints()
				assert.Equal(t, l.end, l.top)
				assert.Zero(t, g.waypoints.list.top)
			})
		})
	}
}

// TestWaypointJumpUsesSetFlags tells a flag by tapping its row, picks a waypoint and starts: the game there has
// what the flag brings (고순 joins at 16장 only when his battle left the flag), and none without the tap.
func TestWaypointJumpUsesSetFlags(t *testing.T) {
	for _, tapped := range []bool{true, false} {
		g, draw := waypointScreen(t, image.Pt(1280, 850))
		flags := g.S.Data.StoryFlags()
		gao := slices.IndexFunc(flags, func(f content.StoryFlag) bool { return f.Flag == "flag-gaoshun" })
		require.GreaterOrEqual(t, gao, 0)
		l := &g.waypoints.flags
		if tapped {
			for top := 0; len(g.jumpFlags) == 0 && top <= l.end; top += 60 {
				l.top = top
				draw()
				for _, r := range l.rows {
					if r.index == gao && r.r.Min.Y > l.view.Min.Y && r.r.Max.Y < l.view.Max.Y {
						tap(g, r.r)
					}
				}
			}
			require.Equal(t, map[string]bool{"flag-gaoshun": true}, g.jumpFlags)
		}
		g.jump("c16-battle", "")
		has := slices.ContainsFunc(g.S.View().Characters, func(c core.Character) bool { return c.ID == "gaoshun" })
		assert.Equal(t, tapped, has, "tapped %v", tapped)
	}
}

func TestWaypointWrapLines(t *testing.T) {
	g, _ := waypointScreen(t, image.Pt(1280, 850))
	for _, f := range g.S.Data.StoryFlags() {
		for _, size := range []float64{12, 14} {
			text := f.Text + " " + f.Later
			lines := g.wrapLines(text, size, 316)
			joined := ""
			for _, line := range lines {
				assert.LessOrEqual(t, g.width(line, size), 316.0, line)
				joined += line
			}
			assert.Equal(t, len([]rune(text))-countSpaces(text), len([]rune(joined))-countSpaces(joined), "no text is lost: "+f.Flag)
		}
	}
}

func countSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		}
	}
	return n
}
