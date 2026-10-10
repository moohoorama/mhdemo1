package mapedit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"srpg/internal/content"
)

func TestSaveRoundTrip(t *testing.T) {
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	d := Import(data.Stages[0])
	d.Paint(16, 8, "~")
	d.Paint(17, 8, "~")
	if err := Save(root, d, data); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(filepath.Join(root, "assets/data/maps/B01.json"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tile(8, 4) != "." || loaded.Small(16, 8) != "~" {
		t.Fatal("history not persisted")
	}
	art := loaded.Graphics(data)
	if art.ID != "B01" || !art.Native || art.Tiles[4][8] != '.' || len(art.Ground) != d.Visual.W*d.Visual.H*16 {
		t.Fatal("graphics mismatched")
	}
	// Failed validation must leave the sole saved map unchanged.
	before, err := os.ReadFile(filepath.Join(root, "assets/data/maps/B01.json"))
	if err != nil {
		t.Fatal(err)
	}
	d.Stage.Enemies = nil
	if Save(root, d, data) == nil {
		t.Fatal("invalid map accepted")
	}
	after, err := os.ReadFile(filepath.Join(root, "assets/data/maps/B01.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("failure changed map")
	}

}
func TestSaveRejectsPathTraversal(t *testing.T) {
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	d := Import(data.Stages[0])
	d.Stage.ID = "../escape"
	if Save(t.TempDir(), d, data) == nil {
		t.Fatal("path traversal accepted")
	}
}

// BenchmarkGraphics-8  1000  25031917 ns/op  28445092 B/op  437495 allocs/op # baseline
// BenchmarkGraphics-8  1000  17654128 ns/op   5303829 B/op   51863 allocs/op # typed fragments, cached layers
// BenchmarkGraphics measures complete source-to-art generation for a 64×64 battlefield.
func BenchmarkGraphics(b *testing.B) {
	data, err := content.Load("../../assets/data")
	if err != nil {
		b.Fatal(err)
	}
	d := Import(data.Stages[0])
	d.Resize(Rect{W: 64, H: 64})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Graphics(data)
	}
}

func TestLoadRejectsMalformedBounds(t *testing.T) {
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	d := Import(data.Stages[0])
	d.Visual.W = MaxSize + 9
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("unbounded document accepted")
	}
}

func TestOneDocumentFeedsCampaignAndGraphics(t *testing.T) {
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source := "../../assets/data"
	err = filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dest := filepath.Join(root, "assets/data", relative)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, b, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := Clone(*data.Maps[data.Stages[0].ID])
	if err := doc.Resize(Rect{X: -2, Y: -1, W: doc.Play.W + 2, H: doc.Play.H + 1}); err != nil {
		t.Fatal(err)
	}
	doc.Paint(16, 8, "~")
	doc.Paint(17, 8, "~")
	hp, mp := 17, 0
	doc.Stage.Enemies[0].HP = &hp
	doc.Stage.Enemies[0].MP = &mp
	doc.Stage.Enemies[0].Facing = "SW"
	if err := Save(root, &doc, data); err != nil {
		t.Fatal(err)
	}
	fresh, err := content.Load(filepath.Join(root, "assets/data"))
	if err != nil {
		t.Fatal(err)
	}
	loaded := fresh.Maps[doc.Stage.ID]
	st := fresh.Stages[fresh.StageIndex(doc.Stage.ID)]
	if loaded.Small(16, 8) != "~" || st.Tiles[5][10] != '.' {
		t.Fatal("document and campaign differ")
	}
	if st.Enemies[0].X != doc.Stage.Enemies[0].X+2 || st.Enemies[0].Y != doc.Stage.Enemies[0].Y+1 || st.Enemies[0].Facing != "SW" {
		t.Fatal("origin or direction lost")
	}
	if st.Enemies[0].HP == nil || *st.Enemies[0].HP != 17 || st.Enemies[0].MP == nil || *st.Enemies[0].MP != 0 {
		t.Fatal("initial values lost")
	}
	if fresh.Hash == data.Hash {
		t.Fatal("content hash did not change")
	}
	art := loaded.Graphics(fresh)
	if art.Tiles[5][10] != '.' || art.SmallTiles[8-doc.Play.Y*2-art.Visual[1]][16-doc.Play.X*2-art.Visual[0]] != '~' {
		t.Fatal("graphics not derived from the same source")
	}
	for _, path := range []string{"assets/data/stages", "assets/graphics/maps", "tools/mapeditor"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatal("generated duplicate", path)
		}
	}
}
