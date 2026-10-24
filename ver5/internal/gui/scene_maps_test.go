package gui

import (
	"image"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenemap"
)

// A format 2 scene map is drawn from the same plan the scene viewer draws (cmd/_scene): the field keeps the plan's
// standing pieces in the plan's order, with the same pictures, and sorts people in among them by the same rule.
func TestScenePlanInField(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	a := d.g.assets
	for _, id := range []string{"S_checkpoint", "S_tent_mid", "S_castle_town"} {
		t.Run(id, func(t *testing.T) {
			art := a.Maps[id]
			require.NotNil(t, art.SceneMap, "%s is format 2", id)
			plan, err := scenemap.Build(art.SceneMap, a.SceneRules, scenemap.Options{Decor: true})
			require.NoError(t, err)
			f := NewField(a, art)
			defer f.dispose()
			require.Len(t, f.scenePieces, len(plan.Solid))
			for i, p := range plan.Solid {
				assert.Equal(t, p.Name, f.scenePieces[i].Name)
				assert.Equal(t, p.Draws, f.scenePieces[i].Draws)
			}
		})
	}
	t.Run("people behind the checkpoint wall go before it, people at the gate after it", func(t *testing.T) {
		f := NewField(a, a.Maps["S_checkpoint"])
		defer f.dispose()
		var items []drawItem
		wall := -1
		for _, it := range f.sceneItems() {
			if wall < 0 && f.scenePieces[len(items)].Name == "swall2" {
				wall = len(items)
			}
			items = append(items, it)
		}
		require.GreaterOrEqual(t, wall, 0)
		person := func(u, v float64) drawItem {
			x, y := f.Center(u, v)
			r := image.Rect(int(x)-12, int(y)-60, int(x)+12, int(y)+4)
			k := f.sceneKey(x, y, 0, scenemap.RankPerson, r)
			return drawItem{y: y, kind: 1, rect: r, key: &k}
		}
		behind, front := person(7.6, 1.2), person(9.0, 1.4)
		items = append(items, behind, front)
		order := map[*scenemap.Key]int{}
		for i, it := range f.orderScene(items) {
			order[it.key] = i
		}
		assert.Less(t, order[behind.key], order[items[wall].key])
		assert.Greater(t, order[front.key], order[items[wall].key])
	})
}

// BenchmarkCheckpointFieldDraw-8           1000   341573 ns/op   607341 B/op   3112 allocs/op # old format, no ceiling
// BenchmarkCheckpointFieldDraw-8           1000   652868 ns/op   961217 B/op   4724 allocs/op # old format with the ceiling
// BenchmarkCheckpointFieldDraw/whole-8      200  3893275 ns/op  6993350 B/op  36011 allocs/op # format 2, every picture drawn
// BenchmarkCheckpointFieldDraw/whole-8      200  1247460 ns/op  2759721 B/op  13840 allocs/op # format 2, blocks drawn once
// BenchmarkCheckpointFieldDraw/screen-8     300  1014017 ns/op  2150758 B/op  10006 allocs/op # format 2, a 1280x720 view
func BenchmarkCheckpointFieldDraw(b *testing.B) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		b.Fatal(err)
	}
	f := NewField(d.g.assets, d.g.assets.Maps["S_checkpoint"])
	defer f.dispose()
	for _, tc := range []struct {
		name string
		view image.Rectangle
	}{{"whole", image.Rectangle{}}, {"screen", image.Rect(0, 0, 1280, 720)}} {
		b.Run(tc.name, func(b *testing.B) {
			x, y := f.Center(6, 4) // the gate, as the scene camera looks at it
			f.DrawBounds = tc.view.Add(image.Pt(int(x)-tc.view.Dx()/2, int(y)-tc.view.Dy()/2))
			for i := 0; i < b.N; i++ {
				f.Draw(0, nil, nil)
			}
		})
	}
}

// The scenario scene maps made by tools/scene_maps are native maps: every one loads, names only pictures and
// animations the game has, and draws.
func TestSceneMapsLoadAndDraw(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		t.Fatal(err)
	}
	a := d.g.assets
	files, _ := filepath.Glob("../../assets/graphics/maps/S_*.json")
	want := map[string]bool{"S_vil_peace": true, "S_vil_burnt": true, "S_tent": true, "S_road": true, "S_hill": true, "S_army_camp": true,
		"S_castle_out": true, "S_castle_out_open": true, "S_courtyard": true, "S_rampart": true, "S_office": true, "S_palace": true, "S_house": true, "S_garden": true}
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".json")
		delete(want, id)
		art := a.Maps[id]
		if art == nil {
			t.Errorf("%s is not in index.json maps", id)
			continue
		}
		if strings.HasPrefix(id, "S_peach") || id == "S_village" || id == "S_camp" {
			continue // the first three scenes keep their own scale
		}
		if !art.Native || !art.Scene {
			t.Errorf("%s: not a native scene map", id)
		}
		v := art.Visual
		if len(art.SmallTiles) != v[3] || len(art.SmallTiles[0]) != v[2] || len(art.Tiles) != art.H || len(art.Tiles[0]) != art.W {
			t.Errorf("%s: small tiles %dx%d do not fit visual %v and size %dx%d", id, len(art.SmallTiles[0]), len(art.SmallTiles), v, art.W, art.H)
		}
		for _, p := range art.Props {
			if _, ok := a.Props[p.Name]; !ok {
				t.Errorf("%s: no prop %q", id, p.Name)
			}
		}
		for _, face := range art.Faces {
			if _, ok := a.Props[face.Name]; !ok {
				t.Errorf("%s: no face picture %q", id, face.Name)
			}
		}
		if art.SceneMap != nil {
			if _, err := scenemap.Build(art.SceneMap, a.SceneRules, scenemap.Options{Decor: true}); err != nil {
				t.Errorf("%s: %v", id, err)
			}
		}
		for _, dec := range art.Decorations {
			if _, ok := a.Animations[dec.Animation]; !ok {
				t.Errorf("%s: no tile animation %q", id, dec.Animation)
			}
		}
		f := NewField(a, art)
		f.Draw(0, nil, nil)
		f.dispose()
	}
	for id := range want {
		t.Errorf("scene map %s is missing", id)
	}
}

// Every dialogue node of the scenario stands in a scene (assets/scenes/*.json), and the scenes' maps exist.
func TestScenarioNodesAreStaged(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		t.Fatal(err)
	}
	data := d.g.S.Data
	scenes := d.g.scenes
	if scenes == nil {
		t.Fatal("assets/scenes does not load")
	}
	for id, def := range scenes.Scenes {
		if d.g.assets.Maps[def.Map] == nil {
			t.Errorf("%s: no map %q", id, def.Map)
		}
	}
	placed := false
	for _, n := range data.Nodes {
		switch n.Kind {
		case "scene":
			placed = true
		case "battle":
			continue
		case "say", "narrate", "choice":
			if placed {
				if def, _ := scenes.For(n.ID); def == nil {
					t.Errorf("node %s has a place but no scene", n.ID)
				}
			}
		}
	}
}

// The format 2 tour (cmd/_town2) opens on every area and every exit's spot can be reached from where the hero stands
// on arriving.
func TestTourAreas(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	require.NoError(t, g.OpenTour("../../assets/towns/tour2.json"))
	for id, a := range g.tour.t.Areas {
		t.Run(id, func(t *testing.T) {
			g.enterTourArea(id, "")
			for _, e := range a.Exits {
				assert.NotNil(t, g.tour.path(g.tourHeroTile(), g.tour.nearestOpen(e.At)), "exit %s cannot be reached", e.Name)
			}
		})
	}
}
