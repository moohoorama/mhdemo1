package main

import (
	"bytes"
	"demo1/internal/terrain"

	"image/png"
	"io/fs"
	"path/filepath"
	"testing"
	"time"
)

func TestEmbeddedTiles(t *testing.T) {
	entries, err := assets.ReadDir("assets/tiles")
	if err != nil || len(entries) != terrain.AssetCount {
		t.Fatalf("expected 38 PNGs: %d, %v", len(entries), err)
	}
	for m := 0; m < terrain.AssetCount; m++ {
		b, err := assets.ReadFile("assets/tiles/" + terrain.AssetName(m))
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		want := terrain.Asset(m)
		if im.Bounds() != want.Bounds() {
			t.Fatalf("tile %d size", m)
		}
		for y := 0; y < 8; y++ {
			for x := 0; x < 16; x++ {
				r, g, b, a := im.At(x, y).RGBA()
				rr, gg, bb, aa := want.At(x, y).RGBA()
				if r != rr || g != gg || b != bb || a != aa {
					t.Fatalf("tile %d pixel %d,%d", m, x, y)
				}
			}
		}
	}
}
func TestEditorStrokeUndoRedoSaveLoad(t *testing.T) {
	g := &game{w: terrain.New(8, 8), path: filepath.Join(t.TempDir(), "map.json")}
	g.stroke = g.w.Clone()
	g.paint(3, 4, terrain.River)
	g.paint(4, 4, terrain.River)
	g.finishStroke()
	if len(g.undo) != 1 || !g.w.At(3, 4) {
		t.Fatal("stroke not recorded")
	}
	g.action("undo")
	if g.w.At(3, 4) || g.w.At(4, 4) {
		t.Fatal("undo stroke failed")
	}
	g.action("redo")
	if !g.w.At(3, 4) || !g.w.At(4, 4) {
		t.Fatal("redo failed")
	}
	g.action("save")
	if g.dirty {
		t.Fatal(g.status)
	}
	g.w.Set(3, 4, false)
	g.action("load")
	if !g.w.At(3, 4) {
		t.Fatal("load failed", g.status)
	}
	g.action("undo")
	if g.w.At(3, 4) {
		t.Fatal("load should be undoable")
	}
}

func TestGrassBrushHistory(t *testing.T) {
	g := &game{w: terrain.New(3, 3), path: filepath.Join(t.TempDir(), "map.json")}
	g.action("grass")
	if g.brush != terrain.Grass {
		t.Fatal("grass selection")
	}
	g.stroke = g.w.Clone()
	g.paint(1, 1, g.brush)
	g.finishStroke()
	if g.w.Terrain(1, 1) != terrain.Grass {
		t.Fatal("grass paint")
	}
	g.action("undo")
	if g.w.Terrain(1, 1) != terrain.Wasteland {
		t.Fatal("grass undo")
	}
	g.action("redo")
	g.action("save")
	g.w.SetTerrain(1, 1, terrain.River)
	g.action("load")
	if g.w.Terrain(1, 1) != terrain.Grass {
		t.Fatal("grass save/load", g.status)
	}
}

func TestObjectSpritesAndForestHistory(t *testing.T) {
	files, err := fs.Sub(assets, "assets/objects")
	if err != nil {
		t.Fatal(err)
	}
	sprites, err := terrain.LoadObjects(files)
	if err != nil {
		t.Fatal(err)
	}
	for tree := 0; tree < 2; tree++ {
		base := sprites[8+tree*4]
		seen := map[string]bool{}
		for frame := 0; frame < 4; frame++ {
			im := sprites[8+tree*4+frame]
			pixels := []byte{}
			for y := 0; y < im.Bounds().Dy(); y++ {
				for x := 0; x < im.Bounds().Dx(); x++ {
					r, g, b, a := im.At(x, y).RGBA()
					pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
					if y >= im.Bounds().Dy()-3 && im.At(x, y) != base.At(x, y) {
						t.Fatal("tree trunk moves")
					}
				}
			}
			seen[string(pixels)] = true
		}
		if len(seen) != 4 {
			t.Fatal("tree needs four distinct foliage frames", tree)
		}
	}
	g := &game{w: terrain.New(3, 3), path: filepath.Join(t.TempDir(), "map.json")}
	for _, action := range []string{"rockgrass", "rockland", "forest"} {
		g.action(action)
		g.stroke = g.w.Clone()
		g.paint(1, 1, g.brush)
		g.finishStroke()
		want := g.w.Objects(1, 1)
		g.action("undo")
		g.action("redo")
		if g.w.Objects(1, 1) != want {
			t.Fatal("history changed objects")
		}
		g.action("save")
		g.action("load")
		if g.w.Objects(1, 1) != want {
			t.Fatal("load changed objects")
		}
	}
}

func TestSwaySpeedAndContinuity(t *testing.T) {
	start := time.Unix(100, 0)
	for _, speed := range []int{1, 2, 4, 8} {
		g := &game{started: start, swaySpeed: speed}
		g.advanceSway(start.Add(time.Second))
		if g.treeTick != 8*speed%terrain.TreeCycleTicks {
			t.Fatalf("speed %d: tick %d", speed, g.treeTick)
		}
		before := g.treeTick
		g.swaySpeed = 1
		g.advanceSway(start.Add(time.Second))
		if g.treeTick != before {
			t.Fatal("speed change jumps")
		}
	}
}
func TestGrassAnimationFrames(t *testing.T) {
	files, _ := fs.Sub(assets, "assets/objects")
	sprites, err := terrain.LoadObjects(files)
	if err != nil {
		t.Fatal(err)
	}
	for object := terrain.GrassTuft1; object <= terrain.GrassTuft4; object++ {
		seen := map[string]bool{}
		base := sprites[terrain.ObjectSprite(object, 0)]
		for frame := 0; frame < 4; frame++ {
			im := sprites[terrain.ObjectSprite(object, frame)]
			var buf bytes.Buffer
			if err := png.Encode(&buf, im); err != nil {
				t.Fatal(err)
			}
			seen[buf.String()] = true
			for y := im.Bounds().Dy() - 2; y < im.Bounds().Dy(); y++ {
				for x := 0; x < im.Bounds().Dx(); x++ {
					if im.At(x, y) != base.At(x, y) {
						t.Fatal("grass roots moved")
					}
				}
			}
			grass := terrain.Placement{Object: object}
			tree := terrain.Placement{Object: terrain.LargeTree}
			tick := frame * terrain.TreeFrameTicks
			if grass.Sprite(tick) != terrain.ObjectSprite(object, tree.TreeFrame(tick)) {
				t.Fatal("grass timing differs")
			}
		}
		if len(seen) != 4 {
			t.Fatalf("grass %d has %d poses", object, len(seen))
		}
	}
}
