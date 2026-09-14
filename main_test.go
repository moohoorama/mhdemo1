package main

import (
	"demo1/internal/graphics"
	"demo1/internal/terrain"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEmbeddedTiles(t *testing.T) {
	files, err := fs.Sub(assets, "assets")
	if err != nil {
		t.Fatal(err)
	}
	c, err := graphics.Load(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := terrain.ValidateAssets(c); err != nil {
		t.Fatal(err)
	}
	if got := c.Images[0].Bounds().Size(); got != image.Pt(128, 40) {
		t.Fatalf("terrain atlas size %v", got)
	}
	if got := c.Images[1].Bounds().Size(); got != image.Pt(192, 128) {
		t.Fatalf("object atlas size %v", got)
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
	c, err := graphics.Load(os.DirFS("assets"))
	if err != nil {
		t.Fatal(err)
	}
	for tree := 0; tree < 2; tree++ {
		seen := map[string]bool{}
		for frame := 0; frame < 4; frame++ {
			id := terrain.AssetCount + terrain.ObjectSprite(terrain.LargeTree+terrain.Object(tree), frame)
			seen[string(spritePixels(c, id))] = true
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
	c, err := graphics.Load(os.DirFS("assets"))
	if err != nil {
		t.Fatal(err)
	}
	for object := terrain.GrassTuft1; object <= terrain.GrassTuft4; object++ {
		seen := map[string]bool{}
		baseID := terrain.AssetCount + terrain.ObjectSprite(object, 0)
		for frame := 0; frame < 4; frame++ {
			id := terrain.AssetCount + terrain.ObjectSprite(object, frame)
			seen[string(spritePixels(c, id))] = true
			baseRect, rect := c.Rect(baseID), c.Rect(id)
			baseImage, im := c.Images[c.Sprites[baseID].Sheet], c.Images[c.Sprites[id].Sheet]
			for y := 28; y < 32; y++ {
				for x := 0; x < 24; x++ {
					if im.At(rect.Min.X+x, rect.Min.Y+y) != baseImage.At(baseRect.Min.X+x, baseRect.Min.Y+y) {
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

func spritePixels(c *graphics.Catalog, id int) []byte {
	rect := c.Rect(id)
	im := c.Images[c.Sprites[id].Sheet]
	pixels := make([]byte, 0, rect.Dx()*rect.Dy()*4)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, a := im.At(x, y).RGBA()
			pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	return pixels
}
