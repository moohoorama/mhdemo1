package assetbuild

import (
	"bytes"
	"demo1/internal/graphics"
	"demo1/internal/terrain"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedAssetsAndLegacyPixels(t *testing.T) {
	out := t.TempDir()
	if err := Generate("../../assets/objects", out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"terrain.png", "objects.png", "catalog.json", "preview.png"} {
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("../../assets/generated", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("stale generated asset %s; run go run ./cmd/assetgen", name)
		}
	}
	c, err := graphics.Load(os.DirFS(out))
	if err != nil {
		t.Fatal(err)
	}
	if err := terrain.ValidateAssets(c); err != nil {
		t.Fatal(err)
	}
	objects, err := LoadObjects(os.DirFS("../../assets/objects"))
	if err != nil {
		t.Fatal(err)
	}
	// Preserve the old renderer's independent placement formulas as a migration oracle.
	worlds := []*terrain.World{terrain.Demo(), terrain.New(3, 2)}
	for i, k := range []terrain.Kind{terrain.River, terrain.Grass, terrain.Wasteland, terrain.RockGrass, terrain.RockWasteland, terrain.Forest} {
		worlds[1].SetTerrain(i%3, i/3, k)
	}
	for wi, w := range worlds {
		scene := terrain.BuildMapScene(w)
		for frame := 0; frame < 8; frame++ {
			tick := frame * 9 // Includes sub-frame plant phase offsets and cycle rollover.
			items, err := scene.BuildDrawList(c, frame, tick, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			got := graphics.RenderPNG(c, items, scene.Bounds, 1)
			want := legacyScene(w, objects, frame, tick)
			if got.Bounds() != want.Bounds() {
				t.Fatal("export bounds changed")
			}
			if !bytes.Equal(got.Pix, want.Pix) {
				for y := 0; y < got.Bounds().Dy(); y++ {
					for x := 0; x < got.Bounds().Dx(); x++ {
						if got.NRGBAAt(x, y) != want.NRGBAAt(x, y) {
							t.Fatalf("world %d frame %d pixel %d,%d changed: %v vs %v", wi, frame, x, y, got.NRGBAAt(x, y), want.NRGBAAt(x, y))
						}
					}
				}
			}
		}
	}
}

func legacyScene(w *terrain.World, objects [terrain.ObjectSpriteCount]image.Image, frame, tick int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, (w.Width+w.Height)*16+32, (w.Width+w.Height)*8+32))
	var tiles [terrain.AssetCount]image.Image
	for i := range tiles {
		tiles[i] = Asset(i)
	}
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			ground := w.MasksFor(x, y, terrain.Grass)
			for k, m := range w.Masks(x, y) {
				px, py := terrain.Project(float64(x)+float64(k%2)/2, float64(y)+float64(k/2)/2)
				p := image.Pt(int(px)+w.Height*16-8+16, int(py)+16)
				for _, i := range terrain.SurfaceLayers(ground[k], m, frame) {
					draw.Draw(out, image.Rectangle{Min: p, Max: p.Add(image.Pt(16, 8))}, tiles[i], image.Point{}, draw.Over)
				}
			}
		}
	}
	for _, p := range w.Placements() {
		sprite := objects[p.Sprite(tick)]
		x, y := p.Position()
		b := sprite.Bounds()
		pt := image.Pt(int(x)+w.Height*16-b.Dx()/2+16, int(y)-b.Dy()+2+16)
		draw.Draw(out, image.Rectangle{Min: pt, Max: pt.Add(b.Size())}, sprite, b.Min, draw.Over)
	}
	return out
}
