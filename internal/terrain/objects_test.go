package terrain

import (
	"path/filepath"
	"testing"
)

func TestDecorationProbabilitiesAndStability(t *testing.T) {
	for _, kind := range []Kind{Grass, RockGrass, RockWasteland} {
		counts := map[Object]int{}
		w := New(256, 256)
		for i := range w.cells {
			w.SetTerrain(i%w.Width, i/w.Width, kind)
		}
		for y := 0; y < w.Height; y++ {
			for x := 0; x < w.Width; x++ {
				objects := w.Objects(x, y)
				if objects != w.Objects(x, y) {
					t.Fatal("unstable decoration")
				}
				for _, object := range objects {
					counts[object]++
				}
			}
		}
		total := float64(w.Width * w.Height * 4)
		first, last, p := SmallRock, LargeRock, .2
		if kind == Grass {
			first, last, p = GrassTuft1, GrassTuft4, .0625
		}
		for object := first; object <= last; object++ {
			fraction := float64(counts[object]) / total
			if fraction < p-.004 || fraction > p+.004 {
				t.Fatalf("kind %v object %d: %.4f", kind, object, fraction)
			}
		}
		copy := w.Clone()
		path := filepath.Join(t.TempDir(), "map.json")
		if err := w.Save(path); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 256; y += 7 {
			for x := 0; x < 256; x += 7 {
				if copy.Objects(x, y) != w.Objects(x, y) || loaded.Objects(x, y) != w.Objects(x, y) {
					t.Fatal("objects changed after save/clone")
				}
			}
		}
	}
}
func TestDecoratedGroundAndDepth(t *testing.T) {
	for _, kind := range []Kind{RockGrass, RockWasteland, Forest} {
		w := New(3, 3)
		base := New(3, 3)
		w.SetTerrain(1, 1, kind)
		base.SetTerrain(1, 1, kind.Base())
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				for _, ground := range []Kind{Grass, Wasteland} {
					if w.MasksFor(x, y, ground) != base.MasksFor(x, y, ground) {
						t.Fatal("object changed ground mask")
					}
				}
			}
		}
	}
	w := New(32, 32)
	for i := range w.cells {
		w.SetTerrain(i%w.Width, i/w.Width, RockGrass)
	}
	lastY := -1.0
	for _, p := range w.Placements() {
		_, y := p.Position()
		if y < lastY {
			t.Fatal("incorrect depth sorting")
		}
		lastY = y
	}
}

func TestForestMinimumAndMarginalProbabilities(t *testing.T) {
	w := New(256, 256)
	for i := range w.cells {
		w.SetTerrain(i%w.Width, i/w.Width, Forest)
	}
	counts := map[Object]int{}
	three, four := 0, 0
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			n := 0
			for _, object := range w.Objects(x, y) {
				counts[object]++
				if object == LargeTree || object == SmallTree {
					n++
				}
			}
			switch n {
			case 3:
				three++
			case 4:
				four++
			default:
				t.Fatal("forest has fewer than three trees", x, y, n)
			}
		}
	}
	total := float64(4 * w.Width * w.Height)
	for object, want := range map[Object]float64{Empty: .1, LargeTree: .4, SmallTree: .4, GrassTuft1: .025, GrassTuft2: .025, GrassTuft3: .025, GrassTuft4: .025} {
		fraction := float64(counts[object]) / total
		if fraction < want-.004 || fraction > want+.004 {
			t.Fatalf("forest object %d fraction %.4f", object, fraction)
		}
	}
	if float64(three)/float64(three+four) < .79 || float64(three)/float64(three+four) > .81 {
		t.Fatal("forest tile distribution", three, four)
	}
}

func TestTreePhasesStableAndStaggered(t *testing.T) {
	offsets := map[int]bool{}
	transitions := map[int]bool{}
	w := New(32, 32)
	for i := range w.cells {
		w.SetTerrain(i%w.Width, i/w.Width, Forest)
	}
	path := filepath.Join(t.TempDir(), "forest.json")
	if err := w.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	a, b := w.Placements(), loaded.Placements()
	for i, p := range a {
		if p.Object < LargeTree {
			continue
		}
		offsets[roll(p.X, p.Y, p.Slot, 6, TreeCycleTicks)] = true
		frames := map[int]bool{}
		changes := 0
		for tick := 0; tick < TreeCycleTicks; tick++ {
			f := p.TreeFrame(tick)
			frames[f] = true
			if f != p.TreeFrame(tick+TreeCycleTicks) || p.Sprite(tick) != b[i].Sprite(tick) {
				t.Fatal("phase changed after cycle/load")
			}
			if f != p.TreeFrame(tick+1) {
				changes++
				transitions[tick%TreeFrameTicks] = true
			}
		}
		if len(frames) != 4 || changes != 4 {
			t.Fatal("incorrect animation cycle")
		}
	}
	if len(offsets) != TreeCycleTicks || len(transitions) != TreeFrameTicks {
		t.Fatal("tree phases remain synchronized")
	}
}
