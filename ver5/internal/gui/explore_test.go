package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenemap"
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
