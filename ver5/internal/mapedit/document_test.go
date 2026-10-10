package mapedit

import (
	"encoding/json"
	"fmt"
	"testing"

	"srpg/internal/content"
)

func TestTerrainHistory(t *testing.T) {
	st := content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{"....", "....", "....", "...."}}
	d := Import(st)
	d.Paint(0, 0, "~")
	d.Paint(1, 0, "~")
	if d.Tile(0, 0) != "." {
		t.Fatal("tie changed existing terrain")
	}
	d.Paint(0, 1, "~")
	if d.Tile(0, 0) != "~" {
		t.Fatal("majority did not change")
	}
	d.Paint(0, 1, ".")
	if d.Tile(0, 0) != "~" {
		t.Fatal("tie lost water history")
	}
	b, _ := json.Marshal(d)
	var loaded Document
	json.Unmarshal(b, &loaded)
	if loaded.Tile(0, 0) != "~" {
		t.Fatal("load lost history")
	}
}
func TestResizeKeepsWorldPositions(t *testing.T) {
	d := Import(content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{"....", "....", "....", "...."}, Enemies: []content.Spawn{{Character: "x", X: 1, Y: 1}}})
	if err := d.Resize(Rect{X: -2, Y: -1, W: 6, H: 5}); err != nil {
		t.Fatal(err)
	}
	st := d.Compile()
	if st.Enemies[0].X != 3 || st.Enemies[0].Y != 2 {
		t.Fatalf("normalized spawn %+v", st.Enemies[0])
	}
	d.Resize(Rect{X: 2, Y: 2, W: 4, H: 4})
	if len(d.Stage.Enemies) != 0 {
		t.Fatal("outside spawn retained")
	}
}
func TestNegativeCell(t *testing.T) {
	if FloorHalf(-1) != -1 || FloorHalf(-3) != -2 {
		t.Fatal("negative division")
	}
}

func TestHistoryRestoresTerrainAndSpawns(t *testing.T) {
	d := Import(content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{"....", "....", "....", "...."}, Enemies: []content.Spawn{{Character: "x", X: 0, Y: 0}}})
	var h History
	h.Push(d)
	d.Paint(0, 0, "~")
	d.Paint(1, 0, "~")
	d.Paint(0, 1, "~")
	d.Resize(Rect{X: 1, Y: 1, W: 4, H: 4})
	if !h.Undo(&d) || d.Tile(0, 0) != "." || len(d.Stage.Enemies) != 1 {
		t.Fatal("undo lost state")
	}
	if !h.Redo(&d) || d.Tile(0, 0) != "~" || len(d.Stage.Enemies) != 0 {
		t.Fatal("redo lost state")
	}
}
func TestFourCorners(t *testing.T) {
	for _, r := range []Rect{{X: -1, Y: -1, W: 5, H: 5}, {X: 0, Y: -1, W: 5, H: 5}, {X: 0, Y: 0, W: 5, H: 5}, {X: -1, Y: 0, W: 5, H: 5}} {
		t.Run(Key(r.X, r.Y), func(t *testing.T) {
			d := Import(content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{"....", "....", "....", "...."}, Allies: []content.Point{{X: 1, Y: 1}}})
			if err := d.Resize(r); err != nil {
				t.Fatal(err)
			}
			st := d.Compile()
			p := st.Allies[0]
			if p.X+r.X != 1 || p.Y+r.Y != 1 {
				t.Fatalf("world anchor moved %+v", p)
			}
		})
	}
}

func TestPluralityAndFourWayTie(t *testing.T) {
	d := Import(content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{"....", "....", "....", "...."}})
	d.Paint(0, 0, "~")
	d.Paint(1, 0, "~")
	d.Paint(0, 1, ":")
	if d.Tile(0, 0) != "~" {
		t.Fatal("2:1:1 failed to resolve to plurality")
	}
	d.Paint(1, 0, "f")
	if d.Tile(0, 0) != "~" {
		t.Fatal("four-way tie lost previous terrain")
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Document
	if err := json.Unmarshal(b, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Tile(0, 0) != "~" {
		t.Fatal("four-way tie not persisted")
	}
}

func TestBrushSizesAndUndo(t *testing.T) {
	for _, n := range []int{1, 2, 4, 8} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			d := Import(content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{"....", "....", "....", "...."}})
			var h History
			h.Push(d)
			d.PaintSquare(0, 0, n, "~")
			count := 0
			for _, tile := range d.Cells {
				if tile == "~" {
					count++
				}
			}
			if count != n*n {
				t.Fatalf("painted %d, expected %d", count, n*n)
			}
			if !h.Undo(&d) {
				t.Fatal("undo failed")
			}
			for _, tile := range d.Cells {
				if tile != "." {
					t.Fatal("undo left brush marks")
				}
			}
			d.PaintSquare(d.Visual.X*2, d.Visual.Y*2, n, "~")
			for key, tile := range d.Cells {
				if tile == "~" {
					var x, y int
					if _, err := fmt.Sscanf(key, "%d,%d", &x, &y); err != nil {
						t.Fatal(err)
					}
					if !d.Visual.Contains(FloorHalf(x), FloorHalf(y)) {
						t.Fatal("brush escaped visual bounds")
					}
				}
			}
		})
	}
}
