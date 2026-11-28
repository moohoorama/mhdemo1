package gui

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenes"
)

// Every shot of the title reel stands on a known map, every cast member has a sprite, every step names a cast
// member, and every motion it plays is on that member's sheet: a slip would only show as someone missing or
// standing still on the title.
func TestTitleReelContent(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "titlereel"})
	require.NoError(t, err)
	g := d.g
	require.NotNil(t, g.reel, "assets/scenes/title/reel.json loads")
	ids := map[string]bool{}
	for _, shot := range g.reel.reel.Shots {
		t.Run(shot.ID, func(t *testing.T) {
			assert.False(t, ids[shot.ID], "shot ids are unique")
			ids[shot.ID] = true
			require.NotNil(t, g.mapArt(shot.Map), "map %s", shot.Map)
			arts := map[string]bool{}
			for key, a := range shot.Cast {
				art := g.actorArt(key, a.Sprite, a.Faction, a.Mount)
				if assert.NotNil(t, art, "%s has a sprite", key) {
					arts[key] = true
					if a.Pose != "" && a.Pose != "pant" {
						assert.NotEmpty(t, motion(art, a.Pose), "%s rests in %s", key, a.Pose)
					}
				}
			}
			check := func(st scenes.Step) {
				for _, name := range []string{st.Actor, st.Strike, st.Follow, st.Cam, st.Brawl, st.Drop} {
					if name != "" && name != "-" {
						assert.Contains(t, shot.Cast, name, "a step names %s", name)
					}
				}
				if a, ok := shot.Cast[st.Actor]; ok && st.Play != "" && arts[st.Actor] {
					assert.NotEmpty(t, motion(g.actorArt(st.Actor, a.Sprite, a.Faction, a.Mount), st.Play), "%s plays %s", st.Actor, st.Play)
				}
			}
			for _, st := range shot.Ambient {
				check(st)
			}
			for _, group := range shot.Groups {
				for _, st := range group {
					check(st)
				}
			}
		})
	}
}

// Nobody and nothing comes into sight or goes out of it in the middle of a shot's remembered circle: people and
// boats come in and go out past its rim. The shots are played on the scene player frame by frame, and whatever is
// first seen, or last seen, well inside the circle is a slip (a boat popping up on the water, a man fading out). A
// man who gets down off his horse (Place) or is thrown from it (Drop) appears beside it, and is let be.
func TestTitleReelNoPopIn(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "titlereel"})
	require.NoError(t, err)
	g := d.g
	require.NotNil(t, g.reel)
	const dt, inner = 1.0 / 30, .8
	for i, shot := range g.reel.reel.Shots {
		t.Run(shot.ID, func(t *testing.T) {
			placed := map[string]bool{}
			for _, group := range shot.Groups {
				for _, st := range group {
					if len(st.Place) == 2 {
						placed[st.Actor] = true
					}
					if st.Drop != "" {
						placed[st.Drop] = true
					}
				}
			}
			g.openShot(i)
			r, s := g.reel, g.scene
			vp := g.battleViewport()
			// depth inside the circle as a share of its radius: 0 at the centre, 1 at the rim
			depth := func(u, v, lift float64) float64 {
				x, y := g.field.Center(u, v)
				sx := (x-g.camX)*g.zoom + float64(vp.Dx())/2
				sy := (y-lift-g.camY)*g.zoom + float64(vp.Dy())/2
				return math.Hypot(sx-r.cx, sy-r.cy) / r.r
			}
			seen := func() map[string]float64 {
				out := map[string]float64{}
				for key, v := range s.actors {
					if !v.gone && !strings.HasPrefix(key, "cam#") {
						out[key] = depth(v.u, v.v, v.lift)
					}
				}
				for _, p := range s.props {
					if u, v, ok := p.at(s); ok {
						out["prop "+p.name] = depth(u, v, 0)
					}
				}
				return out
			}
			before := seen()
			var slips []string
			for f := 1; f < 60*30 && g.sceneBusy(); f++ {
				g.updateScene(dt)
				now := seen()
				for k, at := range now {
					if was, ok := before[k]; (!ok || was >= 1) && at < inner && !placed[k] {
						slips = append(slips, fmt.Sprintf("%s comes into sight at %.0f%% of the radius, %.1fs", k, at*100, float64(f)*dt))
					}
				}
				for k, was := range before {
					if _, ok := now[k]; !ok && was < inner && !strings.HasPrefix(k, "prop ") {
						slips = append(slips, fmt.Sprintf("%s goes out of sight at %.0f%% of the radius, %.1fs", k, was*100, float64(f)*dt))
					}
				}
				before = now
			}
			assert.Empty(t, slips)
		})
	}
}
