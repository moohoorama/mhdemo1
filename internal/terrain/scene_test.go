package terrain

import (
	"demo1/internal/graphics"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestSceneCacheInvalidation(t *testing.T) {
	w := New(3, 3)
	var cache SceneCache
	first := cache.Ensure(w)
	if cache.Ensure(w) != first {
		t.Fatal("unchanged map rebuilt")
	}
	w.SetTerrain(1, 1, Wasteland)
	if cache.Ensure(w) != first {
		t.Fatal("no-op edit rebuilt map")
	}
	w.SetTerrain(1, 1, Forest)
	second := cache.Ensure(w)
	if second == first || len(second.Decorations) != 1 {
		t.Fatal("edit did not rebuild placements")
	}
	if cache.Ensure(w) != second {
		t.Fatal("cache not reused")
	}
	replacement := w.Clone()
	if cache.Ensure(replacement) == second {
		t.Fatal("replacement map reused old cache")
	}
}

func TestDrawListAnimationAndSharedDepth(t *testing.T) {
	c, err := graphics.Load(os.DirFS("../../assets/generated"))
	if err != nil {
		t.Fatal(err)
	}
	w := New(2, 2)
	w.SetTerrain(0, 0, Forest)
	w.SetTerrain(1, 1, River)
	s := BuildMapScene(w)
	a, err := s.BuildDrawList(c, 0, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.BuildDrawList(c, 1, 16, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatal("animation changed placement count")
	}
	changedWater, changedPlant := false, false
	for i := range a {
		if a[i].Position != b[i].Position {
			t.Fatal("animation moved ground anchors")
		}
		if a[i].Sprite != b[i].Sprite {
			if a[i].Layer == 0 {
				changedWater = true
			} else {
				changedPlant = true
			}
		}
	}
	if !changedWater || !changedPlant {
		t.Fatal("missing animation")
	}
	extra := []graphics.DrawItem{{Sprite: 38, Layer: 1, Depth: 5, Position: image.Pt(0, 5)}, {Sprite: 39, Layer: 1, Depth: 100, Position: image.Pt(0, 100)}}
	items, err := s.BuildDrawList(c, 0, 0, extra, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != len(a)+2 {
		t.Fatal("missing extra sprites")
	}
	for i := 1; i < len(items); i++ {
		x, y := items[i-1], items[i]
		if x.Layer > y.Layer || (x.Layer == y.Layer && x.Depth > y.Depth) {
			t.Fatal("incorrect shared depth order")
		}
	}
}

func TestSplitCellsAndLegacyMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "map.json")
	old := `{"version":4,"width":6,"height":1,"terrain_cells":[0,1,2,3,4,5]}`
	if err := os.WriteFile(path, []byte(old), 0644); err != nil {
		t.Fatal(err)
	}
	w, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if w.Cell(i, 0) != Kind(i).Cell() {
			t.Fatal("legacy preset conversion", i)
		}
	}
	// An independent combination must survive saving without a combined Kind enum.
	want := Cell{Ground: Wasteland, Decoration: Trees}
	if !w.SetCell(0, 0, want) {
		t.Fatal("split cell rejected")
	}
	if w.Masks(0, 0) != ([4]int{15, 15, 15, 15}) {
		t.Fatal("ground blending reads decoration instead of ground")
	}
	if err := w.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 5 || loaded.Cell(0, 0) != want {
		t.Fatal("split cell roundtrip")
	}
	for _, bad := range []string{
		`{"version":5,"width":1,"height":1,"cells":[]}`,
		`{"version":5,"width":1,"height":1,"cells":[{"ground":3,"decoration":0}]}`,
		`{"version":5,"width":1,"height":1,"cells":[{"ground":0,"decoration":1}]}`,
		`{"version":5,"width":1,"height":1,"cells":[{"ground":1,"decoration":3}]}`,
	} {
		if err := os.WriteFile(path, []byte(bad), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("invalid split cell accepted", bad)
		}
	}
}
