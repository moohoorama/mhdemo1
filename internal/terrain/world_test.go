package terrain_test

import (
	"bytes"
	"demo1/internal/graphics"
	. "demo1/internal/terrain"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectionAndPicking(t *testing.T) {
	for y := -10; y <= 10; y++ {
		for x := -10; x <= 10; x++ {
			px, py := Project(float64(x)+.5, float64(y)+.5)
			xx, yy := CellAt(px, py)
			if xx != x || yy != y {
				t.Fatal(x, y, xx, yy)
			}
		}
	}
}
func TestMapRoundTripAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.json")
	w := Demo()
	if err := w.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < w.Width*w.Height; i++ {
		if w.Cell(i%w.Width, i/w.Width) != loaded.Cell(i%w.Width, i/w.Width) {
			t.Fatal("roundtrip mismatch")
		}
	}
	copy := w.Clone()
	copy.SetTerrain(0, 0, (w.Cell(0, 0).Ground+1)%3)
	if copy.Cell(0, 0) == w.Cell(0, 0) {
		t.Fatal("clone aliases original")
	}
	if err = os.WriteFile(path, []byte(`{"version":1,"width":999999,"height":2,"water_vertices":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(path); err == nil {
		t.Fatal("accepted invalid map")
	}
}

func TestNeighborhoods512(t *testing.T) {
	for pattern := 0; pattern < 512; pattern++ {
		w := New(3, 3)
		for i := 0; i < 9; i++ {
			w.Set(i%3, i/3, pattern&(1<<i) != 0)
		}
		// Independently derive each of the central tile's nine lattice vertices.
		for j := 0; j < 3; j++ {
			for i := 0; i < 3; i++ {
				want := false
				for y := 0; y < 3; y++ {
					for x := 0; x < 3; x++ {
						if 2*x <= 2+i && 2+i <= 2*x+2 && 2*y <= 2+j && 2+j <= 2*y+2 && !w.At(x, y) {
							want = true
						}
					}
				}
				if w.Vertex(2+i, 2+j) != want {
					t.Fatalf("pattern %d vertex %d,%d", pattern, i, j)
				}
			}
		}
		masks := w.Masks(1, 1)
		for k, bit := range []int{4, 8, 2, 1} {
			if (masks[k]&bit == 0) != w.At(1, 1) {
				t.Fatal("center terrain lost")
			}
		}
	}
}
func TestRiverShapesAndOutside(t *testing.T) {
	for _, cells := range [][][2]int{
		{{1, 1}}, {{1, 0}, {1, 1}, {1, 2}}, {{0, 0}, {1, 1}, {2, 2}},
		{{0, 0}, {2, 0}, {1, 1}, {1, 2}}, {{0, 0}, {0, 1}, {0, 2}},
	} {
		w := New(3, 3)
		for _, p := range cells {
			w.Set(p[0], p[1], true)
		}
		for _, p := range cells {
			for k, bit := range []int{4, 8, 2, 1} {
				if w.Masks(p[0], p[1])[k]&bit != 0 {
					t.Fatal("river center closed")
				}
			}
		}
	}
	w := New(1, 1)
	w.Set(0, 0, true)
	if w.Masks(0, 0) != [4]int{} {
		t.Fatal("outside closes river")
	}
}
func TestLegacyConversion(t *testing.T) {
	for m := 0; m < 16; m++ {
		path := filepath.Join(t.TempDir(), "legacy.json")
		vertices := []bool{m&1 != 0, m&2 != 0, m&8 != 0, m&4 != 0}
		data, _ := json.Marshal(map[string]any{"version": 1, "width": 1, "height": 1, "water_vertices": vertices})
		os.WriteFile(path, data, 0644)
		w, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, v := range vertices {
			if v {
				n++
			}
		}
		if w.Version != MapVersion || w.Width != 1 || w.Height != 1 || w.At(0, 0) != (n >= 2) {
			t.Fatal("legacy conversion", m)
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(data, after) {
			t.Fatal("load modified source")
		}
	}
}
func TestGrassPersistenceAndV2Migration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.json")
	w := New(3, 1)
	w.SetTerrain(0, 0, River)
	w.SetTerrain(1, 0, Grass)
	if err := w.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < w.Width*w.Height; i++ {
		if got.Cell(i%w.Width, i/w.Width) != w.Cell(i%w.Width, i/w.Width) {
			t.Fatal("lost terrain")
		}
	}
	legacy := []byte(`{"version":2,"width":2,"height":1,"water_cells":[true,false]}`)
	if err := os.WriteFile(path, legacy, 0644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil || got.Terrain(0, 0) != River || got.Terrain(1, 0) != Wasteland {
		t.Fatal("v2 conversion", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(legacy, after) {
		t.Fatal("migration modified source")
	}
	for _, bad := range []string{`{"version":3,"width":1,"height":1,"terrain_cells":[3]}`, `{"version":3,"width":1,"height":1,"terrain_cells":[]}`} {
		os.WriteFile(path, []byte(bad), 0644)
		if _, err := Load(path); err == nil {
			t.Fatal("accepted invalid terrain")
		}
	}
}
func TestGroundCoverage(t *testing.T) {
	for _, kind := range []Kind{River, Grass, Wasteland} {
		w := New(1, 1)
		w.SetTerrain(0, 0, kind)
		catalog, err := graphics.Load(os.DirFS("../../assets"))
		if err != nil {
			t.Fatal(err)
		}
		scene := BuildMapScene(w)
		scene.Decorations = nil // This test measures ground coverage only.
		first, err := scene.BuildDrawList(catalog, 0, 0, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		second, err := scene.BuildDrawList(catalog, 1, 0, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		bounds := image.Rect(-16, 0, 16, 16)
		a, b := graphics.RenderPNG(catalog, first, bounds, 1), graphics.RenderPNG(catalog, second, bounds, 1)
		count := 0
		for y := 0; y < 16; y++ {
			for x := 0; x < 32; x++ {
				c := a.NRGBAAt(x, y)
				if c.A != 0 {
					count++
				}
				if kind != River && c != b.NRGBAAt(x, y) {
					t.Fatal("dry terrain animated")
				}
			}
		}
		if count != 256 {
			t.Fatal("large tile coverage", kind, count)
		}
	}
}
