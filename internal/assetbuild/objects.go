package assetbuild

import (
	"demo1/internal/terrain"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
)

// Source PNGs keep the original generated alpha. Sprites are sampled at their
// intended game resolution once, then rendered using nearest-neighbor scaling.
var objectSources = []struct {
	Name                  string
	Width, Height, Frames int
}{
	{"rock_small.png", 8, 6, 1}, {"rock_medium.png", 11, 8, 1},
	{"rock_medium2.png", 10, 9, 1}, {"rock_large.png", 14, 12, 1},
	{"grass_0.png", 6, 5, 1}, {"grass_1.png", 6, 4, 1},
	{"grass_2.png", 5, 5, 1}, {"grass_3.png", 6, 5, 1},
	{"tree_0-redrawn.png", 22, 20, 4}, {"tree_1-redrawn.png", 20, 18, 4},
}

// blackOutline is a one-game-pixel, four-neighbor silhouette effect.
// Padding keeps the artwork intact; the renderer compensates the foot anchor.
func blackOutline(source image.Image) *image.NRGBA {
	b := source.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()+2, b.Dy()+2))
	solid := func(x, y int) bool { _, _, _, a := source.At(b.Min.X+x, b.Min.Y+y).RGBA(); return a >= 64*257 }
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if !solid(x, y) {
				continue
			}
			for _, d := range []image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				if !solid(x+d.X, y+d.Y) {
					out.SetNRGBA(x+1+d.X, y+1+d.Y, color.NRGBA{0, 0, 0, 255})
				}
			}
		}
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if solid(x, y) {
				out.Set(x+1, y+1, source.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	}
	return out
}
func LoadObjects(files fs.FS) ([terrain.ObjectSpriteCount]image.Image, error) {
	var sprites [terrain.ObjectSpriteCount]image.Image
	index := 0
	for _, spec := range objectSources {
		f, err := files.Open(spec.Name)
		if err != nil {
			return sprites, err
		}
		source, err := png.Decode(f)
		f.Close()
		if err != nil {
			return sprites, err
		}
		width, height := source.Bounds().Dx()/spec.Frames, source.Bounds().Dy()
		// One shared crop across animation cells preserves the foot anchor.
		crop := image.Rectangle{}
		for frame := 0; frame < spec.Frames; frame++ {
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					_, _, _, a := source.At(x+frame*width, y).RGBA()
					if a > 4096 {
						p := image.Rect(x, y, x+1, y+1)
						if crop.Empty() {
							crop = p
						} else {
							crop = crop.Union(p)
						}
					}
				}
			}
		}
		if crop.Empty() {
			return sprites, fmt.Errorf("empty sprite %s", spec.Name)
		}
		for frame := 0; frame < spec.Frames; frame++ {
			sprite := image.NewNRGBA(image.Rect(0, 0, spec.Width, spec.Height))
			for y := 0; y < spec.Height; y++ {
				for x := 0; x < spec.Width; x++ {
					sx := crop.Min.X + (2*x+1)*crop.Dx()/(2*spec.Width) + frame*width
					sy := crop.Min.Y + (2*y+1)*crop.Dy()/(2*spec.Height)
					sprite.Set(x, y, source.At(sx, sy))
				}
			}
			// Keep the trunk/root pixels identical; only the crown uses animation frames.
			if spec.Frames > 1 && frame > 0 {
				base := sprites[index-frame]
				for y := spec.Height - max(3, spec.Height/3); y < spec.Height; y++ {
					for x := 0; x < spec.Width; x++ {
						sprite.Set(x, y, base.At(x, y))
					}
				}
			}
			sprites[index] = sprite
			index++
		}
	}
	// Bend the blades in four poses while keeping their roots anchored.
	for tuft := 0; tuft < 4; tuft++ {
		base := sprites[4+tuft]
		for frame := 0; frame < 4; frame++ {
			b := base.Bounds()
			pose := image.NewNRGBA(image.Rect(0, 0, b.Dx()+4, b.Dy()))
			for y := 0; y < b.Dy(); y++ {
				shift := 0
				switch frame {
				case 1:
					if y < b.Dy()-2 {
						shift = 1
					}
				case 2:
					if y < b.Dy()-2 {
						shift = 2
					}
				case 3:
					if y < b.Dy()-2 {
						shift = -1
					}
				}
				for x := 0; x < b.Dx(); x++ {
					pose.Set(x+2+shift, y, base.At(x, y))
				}
			}
			sprites[terrain.ObjectSprite(terrain.GrassTuft1+terrain.Object(tuft), frame)] = pose
		}
	}
	for i, sprite := range sprites {
		sprites[i] = blackOutline(sprite)
	}
	return sprites, nil
}
