package gui

import (
	"path/filepath"
	"strings"
	"testing"
)

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
