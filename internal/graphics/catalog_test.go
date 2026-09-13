package graphics

import (
	"image"
	"testing"
)

func TestCatalogValidation(t *testing.T) {
	valid := func() Manifest {
		return Manifest{Version: 1, Sheets: []Sheet{{File: "sheet.png", CellWidth: 24, CellHeight: 24, Columns: 4, Rows: 10}}, Sprites: []Sprite{{Pivot: image.Pt(12, 20)}}, Animations: map[string]Animation{"idle": {Frames: []int{0}, FrameTicks: 1, Loop: true}}}
	}
	for _, mutate := range []func(*Manifest){
		func(m *Manifest) { m.Sheets[0].File = "../outside.png" },
		func(m *Manifest) { m.Sheets[0].CellWidth = 0 },
		func(m *Manifest) { m.Sprites[0].Sheet = 1 },
		func(m *Manifest) { m.Sprites[0].Cell = 40 },
		func(m *Manifest) { m.Sprites[0].Pivot.Y = 25 },
		func(m *Manifest) { m.Animations["idle"] = Animation{Frames: []int{1}, FrameTicks: 1} },
		func(m *Manifest) { m.Animations["idle"] = Animation{Frames: []int{0}, FrameTicks: 0} },
	} {
		m := valid()
		mutate(&m)
		if m.Validate() == nil {
			t.Fatal("invalid catalog accepted")
		}
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestAnimationTiming(t *testing.T) {
	a := Animation{Frames: []int{10, 11, 12, 13}, FrameTicks: 2, Loop: true}
	for tick, want := range []int{10, 10, 11, 11, 12, 12, 13, 13, 10} {
		if a.Frame(tick) != want {
			t.Fatal("loop timing", tick)
		}
	}
	a.Loop = false
	if a.Frame(100) != 13 || a.Frame(-1) != 10 {
		t.Fatal("one-shot clamping")
	}
}
