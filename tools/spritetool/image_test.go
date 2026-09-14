package main

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

func TestSharedTrimAndFrameSlicing(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 12, 5))
	ink := color.NRGBA{90, 140, 70, 255}
	source.SetNRGBA(1, 1, ink)
	source.SetNRGBA(10, 3, ink)
	o := options{frames: 2, frame: -1, trim: true}
	frames, err := sourceFrames(source, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || frames[0].Bounds().Size() != image.Pt(4, 3) || frames[1].Bounds().Size() != image.Pt(4, 3) {
		t.Fatalf("unexpected shared crop: %v %v", frames[0].Bounds(), frames[1].Bounds())
	}
	if _, _, _, a := frames[0].At(0, 0).RGBA(); a == 0 {
		t.Fatal("first frame shifted")
	}
	if _, _, _, a := frames[1].At(3, 2).RGBA(); a == 0 {
		t.Fatal("second frame shifted")
	}
}

func TestNearestResizeUsesPixelCenters(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	for x := 0; x < 4; x++ {
		source.SetNRGBA(x, 0, color.NRGBA{uint8(x), 0, 0, 255})
	}
	got := resizeNearest(source, 2, 1)
	if got.NRGBAAt(0, 0).R != 1 || got.NRGBAAt(1, 0).R != 3 {
		t.Fatal("unexpected nearest-neighbor samples")
	}
}

func TestOutlineKeepsArtworkAndAddsOnePixel(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	ink := color.NRGBA{120, 160, 80, 255}
	source.SetNRGBA(0, 0, ink)
	source.SetNRGBA(1, 0, ink)
	source.SetNRGBA(1, 1, ink)
	got := addOutline(source, 1)
	if got.Bounds() != image.Rect(0, 0, 5, 5) {
		t.Fatal("outline padding")
	}
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			c := got.NRGBAAt(x, y)
			if source.NRGBAAt(x-1, y-1).A != 0 {
				if c != ink {
					t.Fatal("artwork changed")
				}
				continue
			}
			border := false
			for _, d := range []image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				if source.NRGBAAt(x-1+d.X, y-1+d.Y).A != 0 {
					border = true
				}
			}
			if border && c != (color.NRGBA{0, 0, 0, 255}) {
				t.Fatal("outline not opaque black")
			}
			if !border && c.A != 0 {
				t.Fatal("outline thicker than one pixel")
			}
		}
	}
}

func TestAtlasPackingUsesExplicitPivots(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "sprite.png")
	output := filepath.Join(dir, "atlas.png")
	sprite := image.NewNRGBA(image.Rect(0, 0, 4, 6))
	sprite.SetNRGBA(2, 4, color.NRGBA{255, 0, 0, 255})
	if err := encodePNG(input, sprite); err != nil {
		t.Fatal(err)
	}
	spec := atlasValue{set: true, cell: image.Pt(8, 8), grid: image.Pt(2, 1)}
	anchor := anchorValue{set: true, x: axisAnchor{center: true}, y: axisAnchor{end: true, offset: -2}}
	if err := packAtlas([]string{input}, output, spec, image.Pt(4, 6), anchor); err != nil {
		t.Fatal(err)
	}
	atlas, err := decodePNG(output)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, _, a := atlas.At(4, 6).RGBA(); r == 0 || a == 0 {
		t.Fatal("source pivot was not mapped to atlas pivot")
	}
	if _, _, _, a := atlas.At(12, 6).RGBA(); a != 0 {
		t.Fatal("unused atlas cell is not transparent")
	}
}
