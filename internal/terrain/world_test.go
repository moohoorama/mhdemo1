package terrain_test

import (
	"bytes"
	"demo1/internal/assetbuild"
	"demo1/internal/graphics"
	. "demo1/internal/terrain"
	"encoding/json"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedEdgesAllCompatibleMasks(t *testing.T) {
	for a := 0; a < 16; a++ {
		for b := 0; b < 16; b++ {
			for i := 0; i <= 100; i++ {
				s := float64(i) / 100
				if (a>>1&1) == (b&1) && (a>>2&1) == (b>>3&1) {
					if math.Abs(assetbuild.Sample(a, 1, s)-assetbuild.Sample(b, 0, s)) > 1e-12 {
						t.Fatalf("u seam %d/%d", a, b)
					}
				}
				if (a>>3&1) == (b&1) && (a>>2&1) == (b>>1&1) {
					if math.Abs(assetbuild.Sample(a, s, 1)-assetbuild.Sample(b, s, 0)) > 1e-12 {
						t.Fatalf("v seam %d/%d", a, b)
					}
				}
			}
		}
	}
}
func TestPixelCoverageNoGapsOrOverlap(t *testing.T) {
	counts := map[image.Point]int{}
	for y := -4; y <= 4; y++ {
		for x := -4; x <= 4; x++ {
			im := assetbuild.Asset(0)
			px, py := Project(float64(x)/2, float64(y)/2)
			pixels := 0
			for iy := 0; iy < 8; iy++ {
				for ix := 0; ix < 16; ix++ {
					if im.NRGBAAt(ix, iy).A != 0 {
						counts[image.Pt(int(px)-8+ix, int(py)+iy)]++
						pixels++
					}
				}
			}
			if pixels != 64 {
				t.Fatalf("diamond area %d, want 64", pixels)
			}
		}
	}
	for y := -20; y <= 20; y++ {
		for x := -40; x <= 40; x++ {
			u, v := Unproject(float64(x)+.5, float64(y)+.5)
			if u >= -2 && u < 2.5 && v >= -2 && v < 2.5 {
				if n := counts[image.Pt(x, y)]; n != 1 {
					t.Fatalf("pixel %d,%d covered %d times", x, y, n)
				}
			}
		}
	}
}
func TestDiagonalWaterSeparated(t *testing.T) {
	for _, m := range []int{5, 10} {
		if assetbuild.Sample(m, .5, .5) <= .5 {
			t.Fatalf("connected saddle %d", m)
		}
	}
}
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
		for sy := 0; sy < 6; sy++ {
			for sx := 0; sx < 6; sx++ {
				m := w.Masks(sx/2, sy/2)[sy%2*2+sx%2]
				if sx < 5 {
					n := w.Masks((sx+1)/2, sy/2)[sy%2*2+(sx+1)%2]
					for q := 0; q <= 16; q++ {
						v := float64(q) / 16
						if assetbuild.Sample(m, 1, v) != assetbuild.Sample(n, 0, v) {
							t.Fatalf("horizontal seam pattern %d", pattern)
						}
					}
				}
				if sy < 5 {
					n := w.Masks(sx/2, (sy+1)/2)[(sy+1)%2*2+sx%2]
					for q := 0; q <= 16; q++ {
						u := float64(q) / 16
						if assetbuild.Sample(m, u, 1) != assetbuild.Sample(n, u, 0) {
							t.Fatalf("vertical seam pattern %d", pattern)
						}
					}
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
func TestOverlayAnimation(t *testing.T) {
	var tiles [AssetCount]image.Image
	for i := range tiles {
		tiles[i] = assetbuild.Asset(i)
	}
	for m := 1; m < 15; m++ {
		overlay := assetbuild.Asset(m + 8)
		for y := 0; y < 8; y++ {
			for x := 0; x < 16; x++ {
				c := overlay.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				if c.R < c.B {
					t.Fatal("water color in land overlay")
				}
				for f := 0; f < 8; f++ {
					composed := image.NewNRGBA(image.Rect(0, 0, 16, 8))
					for _, i := range Layers(m, f) {
						draw.Draw(composed, composed.Bounds(), tiles[i], image.Point{}, draw.Over)
					}
					if composed.NRGBAAt(x, y) != c {
						t.Fatal("land animated")
					}
				}
			}
		}
	}
	// Adjacent animation steps, including wrap, must have the same bounded change.
	for f := 0; f < 8; f++ {
		a, b := assetbuild.Asset(f), assetbuild.Asset((f+1)%8)
		changed := false
		for y := 0; y < 8; y++ {
			for x := 0; x < 16; x++ {
				c, d := a.NRGBAAt(x, y), b.NRGBAAt(x, y)
				if c != d {
					changed = true
				}
				if math.Abs(float64(c.G)-float64(d.G)) > 8 {
					t.Fatal("animation discontinuity")
				}
			}
		}
		if !changed {
			t.Fatal("static water frame")
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
func TestWaterSpatialPeriodAndGrassCoverage(t *testing.T) {
	for frame := 0; frame < 8; frame++ {
		for y := 0; y < 8; y++ {
			for x := 0; x < 16; x++ {
				c := assetbuild.WaterColor(float64(x)+.5, float64(y)+.5, frame)
				for _, d := range [][2]float64{{8, 4}, {-8, 4}} {
					if c != assetbuild.WaterColor(float64(x)+.5+d[0], float64(y)+.5+d[1], frame) {
						t.Fatal("water spatial seam")
					}
				}
			}
		}
	}
	for _, kind := range []Kind{River, Grass, Wasteland} {
		w := New(1, 1)
		w.SetTerrain(0, 0, kind)
		catalog, err := graphics.Load(os.DirFS("../../assets/generated"))
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
