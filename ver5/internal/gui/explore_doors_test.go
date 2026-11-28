package gui

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// TestExploreDoorsMatchInside checks the exits into buildings that carry a Face (docs/plan-scene-exits.md): the exit
// back out of the room is walked the other way, it lies on the room's front wall on the side the building's door
// faces in town, and it and the two small tiles inside it are open ground. Arriving, the hero faces into the room,
// and back in town out of the door.
func TestExploreDoorsMatchInside(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	opposite := map[string]string{"NE": "SW", "SW": "NE", "NW": "SE", "SE": "NW"}
	doors := 0
	for _, name := range slices.Sorted(maps.Keys(g.S.Data.Explores)) {
		x := g.S.Data.Explores[name]
		for _, aid := range slices.Sorted(maps.Keys(x.Areas)) {
			for _, e := range x.Areas[aid].Exits {
				room := x.Areas[e.To]
				town := g.assets.Maps[x.Areas[aid].Map]
				if e.Face == "" || room == nil || town == nil || town.SceneMap == nil || len(town.SceneMap.Exits) == 0 {
					continue // only a street's door into a room: the town maps are those with named exits
				}
				doors++
				where := name + "/" + aid + " " + e.Name
				var back *[2]int
				var backFace string
				for _, b := range room.Exits {
					if b.To == aid {
						back, backFace = &b.At, b.Face
					}
				}
				if !assert.NotNil(t, back, "%s: the room has no way back", where) {
					continue
				}
				assert.Equal(t, opposite[e.Face], backFace, "%s: the room's door is walked out the other way", where)
				art := g.assets.Maps[room.Map]
				require.NotNil(t, art, room.Map)
				w, h := 2*art.SceneMap.Size[0], 2*art.SceneMap.Size[1]
				switch backFace {
				case "SW":
					assert.Equal(t, h-1, back[1], "%s: the room's door on its front-left wall", where)
				case "SE":
					assert.Equal(t, w-1, back[0], "%s: the room's door on its front-right wall", where)
				}
				walk := scenemap.Walkable(art.SceneMap, g.assets.SceneRules)
				st := scenemap.ExitStep[backFace]
				for k := range 3 {
					u, v := back[0]-st[0]*k, back[1]-st[1]*k
					assert.True(t, walk.Open(u, v), "%s: (%d, %d) before the room's door is open", where, u, v)
				}
				g.openScene(&scenes.Scene{ID: "check", Map: room.Map, Cast: map[string]scenes.Actor{}})
				g.doorLight(room)
				lit := slices.ContainsFunc(g.scene.props, func(p sceneProp) bool {
					return strings.HasPrefix(p.name, "door_light_"+strings.ToLower(backFace)+"_") && [2]float64{p.u, p.v} == func() [2]float64 { u, v := smallTileCenter(*back); return [2]float64{u, v} }()
				})
				assert.True(t, lit, "%s: the sun comes in at the room's door", where)
				assert.Equal(t, opposite[backFace], x.Arrival(e.To, aid).Face, "%s: entering, the hero faces into the room", where)
				assert.Equal(t, opposite[e.Face], x.Arrival(aid, e.To).Face, "%s: back in town, the hero faces out of the door", where)
			}
		}
	}
	assert.NotZero(t, doors)
	t.Logf("%d doors", doors)
}

// TestMapExitsOpen checks every named exit of the scene maps: its small tile and the four before it, the two cells a
// leaving actor walks straight through, are open ground of one level where they lie on the map.
func TestMapExitsOpen(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	n := 0
	for _, id := range slices.Sorted(maps.Keys(g.assets.Maps)) {
		art := g.assets.Maps[id]
		if art.SceneMap == nil || len(art.SceneMap.Exits) == 0 {
			continue
		}
		gr := newExitGround(art, g.assets.SceneRules)
		for _, e := range art.SceneMap.Exits {
			n++
			at, st := exitCell(e), scenemap.ExitStep[e.Dir]
			in := [2]float64{at[0] - float64(st[0]*exitApproach), at[1] - float64(st[1]*exitApproach)}
			assert.True(t, gr.stepOpen(in, at, smallTileAt(in[0], in[1])) && gr.walk.Open(smallTileAt(in[0], in[1])[0], smallTileAt(in[0], in[1])[1]),
				"%s exit %s: the way in from (%g, %g) to (%g, %g) is open", id, e.ID, in[0], in[1], at[0], at[1])
		}
	}
	assert.NotZero(t, n)
	t.Logf("%d exits", n)
}
