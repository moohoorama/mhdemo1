package gui

import (
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
