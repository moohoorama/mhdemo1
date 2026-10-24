package gui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointSmallGridProps(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	art := d.g.assets.Maps["S_checkpoint"]
	require.NotNil(t, art)
	require.NotEmpty(t, art.Props)
	assert.True(t, art.SmallSnap)
	assert.False(t, art.CliffRepeat)
	for _, tc := range []struct {
		name string
		u, v float64
	}{
		{"watchtower", 6.75, 1.75},
		{"notice_post", 9.75, 5.25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, p := range art.Props {
				if p.Name == tc.name {
					assert.InDelta(t, tc.u, p.U, .02)
					assert.InDelta(t, tc.v, p.V, .02)
					return
				}
			}
			t.Fatalf("missing prop %q", tc.name)
		})
	}
	for _, face := range art.Faces {
		if strings.HasPrefix(face.Name, "swall_v") {
			assert.LessOrEqual(t, face.V, 14, "wall reaches beyond the near hillside")
			assert.Equal(t, "SW", face.Side)
		}
		if strings.HasPrefix(face.Name, "tcliff_") && face.U == 17 && face.V == 16 {
			t.Fatal("adjacent highland tiles must not have a wall between them")
		}
	}
	for _, tc := range []struct {
		name   string
		u, v   int
		side   string
		dx, dy int
	}{
		{"tcliff_08_2", 17, 1, "SW", 8, -4},
		{"tcliff_02_2", 17, 15, "NE", -8, 4},
		{"tcliff_01_0", -18, 16, "NW", 8, 4},
		{"tcliff_04_2", 41, 16, "SE", -8, -4},
		{"tcliff_08_0", 0, 33, "SW", 8, -4},
		{"cfoot_se", 41, 1, "SE", -8, -4},
		{"sgate_open_v_5", 17, 9, "SW", 8, -4},
		{"jshade_80_0", 18, 1, "SW", -8, -2},
		{"jshade_80_1", 19, 1, "SW", -8, -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, face := range art.Faces {
				if face.Name == tc.name && face.U == tc.u && face.V == tc.v {
					assert.Equal(t, tc.side, face.Side)
					assert.Equal(t, tc.dx, face.DX)
					assert.Equal(t, tc.dy, face.DY)
					return
				}
			}
			t.Fatalf("missing face %q at (%d,%d)", tc.name, tc.u, tc.v)
		})
	}
	cornerFound := false
	nearCornerFound := false
	for _, p := range art.Props {
		assert.NotContains(t, p.Name, "tcliff_00_", "interior soil needs no cliff-top outline")
		if p.Name == "tcliff_12_2" && p.U == 20.25 && p.V == 16.25 {
			cornerFound = true
		}
		if p.Name == "tcliff_06_2" && p.U == 20.25 && p.V == 7.25 {
			nearCornerFound = true
		}
	}
	assert.True(t, cornerFound, "east and south faces should meet at (41,33)")
	assert.True(t, nearCornerFound, "NE and SE faces should meet at (41,15)")
	assert.True(t, hiddenCliffFace("tcliff_02_2"), "the back face should not draw at 0 degrees")
	assert.False(t, hiddenCliffFace("tcliff_06_2"), "the southeast face should draw at 0 degrees")
	f := NewField(d.g.assets, art)
	assert.True(t, f.highlandAt(17, 16), "an interior highland tile has a ceiling even without an open wall")
	assert.True(t, f.highlandAt(17, 15))
	assert.False(t, f.highlandAt(17, 14))
	assert.NotEmpty(t, f.ceilingItems)
	foundShadow := false
	f.eachProp(false, func(name string, _ Sprite, x, y float64) {
		if name == "jshade_80_0" {
			assert.InDelta(t, f.OriginX*K+256, x, .01)
			assert.InDelta(t, f.OriginY*K+162, y, .01)
			foundShadow = true
		}
	})
	assert.True(t, foundShadow, "missing rendered corner shadow")
	f.dispose()
	closed := d.g.assets.Maps["S_checkpoint_closed"]
	require.NotNil(t, closed)
	assert.True(t, closed.SmallSnap)
	assert.False(t, closed.CliffRepeat)
	assert.Equal(t, art.Highland, closed.Highland)
	assert.Len(t, closed.Props, len(art.Props))
	assert.Len(t, closed.Faces, len(art.Faces))
	for _, face := range closed.Faces {
		if face.Name == "sgate_closed_v_5" {
			assert.Equal(t, 17, face.U)
			assert.Equal(t, 9, face.V)
			assert.Equal(t, "SW", face.Side)
			return
		}
	}
	t.Fatal("missing closed gate")
}

// BenchmarkCheckpointFieldDraw/baseline-8          1000     341573 ns/op   607341 B/op   3112 allocs/op # Apple M2
// BenchmarkCheckpointFieldDraw/raised_ceiling-8    1000     652868 ns/op   961217 B/op   4724 allocs/op # ceiling and extended cliff top
// BenchmarkCheckpointFieldDraw/one_cliff-8         1000     635701 ns/op   877065 B/op   4253 allocs/op # small-grid snap, one cliff draw
func BenchmarkCheckpointFieldDraw(b *testing.B) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		ceiling bool
	}{{"baseline", false}, {"raised_ceiling", true}} {
		b.Run(tc.name, func(b *testing.B) {
			art := *d.g.assets.Maps["S_checkpoint"]
			if !tc.ceiling {
				art.Highland = nil
			}
			f := NewField(d.g.assets, &art)
			defer f.dispose()
			b.ResetTimer()
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
