package mapedit

import (
	"encoding/json"
	"fmt"
	"testing"

	"srpg/internal/content"
)

const grass = "g--g--g--g--"

func grassStage(extra func(*content.Stage)) content.Stage {
	st := content.Stage{ID: "test", Width: 4, Height: 4, Tiles: []string{grass, grass, grass, grass}}
	if extra != nil {
		extra(&st)
	}
	return st
}

func testData(t *testing.T) *content.Data {
	t.Helper()
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPaintLayersFollowWhatStandsOnWhat(t *testing.T) {
	data := testData(t)
	d := Import(grassStage(nil))
	if !d.Paint(data, 0, 0, content.LayerGround, 'w') || d.Cell(0, 0).String() != "w--" {
		t.Fatal("ground not painted")
	}
	if d.Paint(data, 0, 0, content.LayerStructure, 't') || d.Cell(0, 0).String() != "w--" {
		t.Fatal("a tree stood on water")
	}
	if !d.Paint(data, 0, 0, content.LayerStructure, 'b') || d.Cell(0, 0).String() != "wb-" {
		t.Fatal("no bridge on water")
	}
	if !d.Paint(data, 1, 0, content.LayerStructure, 'v') || d.Cell(1, 0).String() != "gv-" {
		t.Fatal("no village on grass")
	}
	if !d.Paint(data, 1, 0, content.LayerGround, 'w') || d.Cell(1, 0).String() != "w--" {
		t.Fatal("water kept a village")
	}
	if d.Paint(data, 2, 0, content.LayerStructure, 'p') {
		t.Fatal("a gate stood on grass")
	}
	d.Paint(data, 2, 0, content.LayerState, 'e')
	if d.Cell(2, 0).String() != "g-e" {
		t.Fatal("state not painted")
	}
	if d.Paint(data, 2, 0, content.LayerState, 'x') {
		t.Fatal("unknown state painted")
	}
	b, _ := json.Marshal(d)
	var loaded Document
	if err := json.Unmarshal(b, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Cell(0, 0).String() != "wb-" || loaded.Cell(2, 0).String() != "g-e" {
		t.Fatal("load lost cells")
	}
}
func TestResizeKeepsWorldPositions(t *testing.T) {
	d := Import(grassStage(func(st *content.Stage) { st.Enemies = []content.Spawn{{Character: "x", X: 1, Y: 1}} }))
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
	data := testData(t)
	d := Import(grassStage(func(st *content.Stage) { st.Enemies = []content.Spawn{{Character: "x", X: 0, Y: 0}} }))
	var h History
	h.Push(d)
	d.Paint(data, 0, 0, content.LayerGround, 'w')
	d.Resize(Rect{X: 1, Y: 1, W: 4, H: 4})
	if !h.Undo(&d) || d.Cell(0, 0).String() != "g--" || len(d.Stage.Enemies) != 1 {
		t.Fatal("undo lost state")
	}
	if !h.Redo(&d) || d.Cell(0, 0).String() != "w--" || len(d.Stage.Enemies) != 0 {
		t.Fatal("redo lost state")
	}
}
func TestFourCorners(t *testing.T) {
	for _, r := range []Rect{{X: -1, Y: -1, W: 5, H: 5}, {X: 0, Y: -1, W: 5, H: 5}, {X: 0, Y: 0, W: 5, H: 5}, {X: -1, Y: 0, W: 5, H: 5}} {
		t.Run(Key(r.X, r.Y), func(t *testing.T) {
			d := Import(grassStage(func(st *content.Stage) { st.Allies = []content.Point{{X: 1, Y: 1}} }))
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

func TestBrushSizesAndUndo(t *testing.T) {
	data := testData(t)
	for _, n := range []int{1, 2, 4, 8} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			d := Import(grassStage(nil))
			var h History
			h.Push(d)
			d.PaintSquare(data, 0, 0, n, content.LayerGround, 'w')
			count := 0
			for _, c := range d.Cells {
				if c == "w--" {
					count++
				}
			}
			if count != n*n {
				t.Fatalf("painted %d, expected %d", count, n*n)
			}
			if !h.Undo(&d) {
				t.Fatal("undo failed")
			}
			for _, c := range d.Cells {
				if c != "g--" {
					t.Fatal("undo left brush marks")
				}
			}
			d.PaintSquare(data, d.Visual.X, d.Visual.Y, n, content.LayerGround, 'w')
			for key, c := range d.Cells {
				if c == "w--" {
					var x, y int
					if _, err := fmt.Sscanf(key, "%d,%d", &x, &y); err != nil {
						t.Fatal(err)
					}
					if !d.Visual.Contains(x, y) {
						t.Fatal("brush escaped visual bounds")
					}
				}
			}
		})
	}
}
