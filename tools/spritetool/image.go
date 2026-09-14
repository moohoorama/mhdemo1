package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

func processFile(input, output string, o options) error {
	source, err := decodePNG(input)
	if err != nil {
		return err
	}
	frames, err := sourceFrames(source, o)
	if err != nil {
		return err
	}
	processed := make([]image.Image, 0, len(frames))
	for _, frame := range frames {
		im := frame
		if o.size.set {
			im = resizeNearest(im, o.size.size.X, o.size.size.Y)
		}
		if o.outline > 0 {
			im = addOutline(im, o.outline)
		}
		processed = append(processed, im)
	}
	return encodePNG(output, joinHorizontal(processed))
}

func sourceFrames(source image.Image, o options) ([]image.Image, error) {
	b := source.Bounds()
	if o.crop.set {
		if !o.crop.rect.In(b) {
			return nil, fmt.Errorf("crop %v is outside source bounds %v", o.crop.rect, b)
		}
		b = o.crop.rect
	}
	if b.Dx()%o.frames != 0 {
		return nil, fmt.Errorf("source width %d is not divisible by %d frames", b.Dx(), o.frames)
	}
	frameWidth := b.Dx() / o.frames
	if frameWidth < 1 {
		return nil, fmt.Errorf("source width %d is too small for %d frames", b.Dx(), o.frames)
	}
	localCrop := image.Rect(0, 0, frameWidth, b.Dy())
	if o.trim {
		localCrop = transparentBounds(source, b, o.frames)
		if localCrop.Empty() {
			return nil, fmt.Errorf("source has no visible pixels")
		}
	}
	first, last := 0, o.frames
	if o.frame >= 0 {
		first, last = o.frame, o.frame+1
	}
	result := make([]image.Image, 0, last-first)
	for i := first; i < last; i++ {
		r := localCrop.Add(image.Pt(b.Min.X+i*frameWidth, b.Min.Y))
		result = append(result, copyImage(source, r))
	}
	return result, nil
}

func transparentBounds(source image.Image, bounds image.Rectangle, frames int) image.Rectangle {
	frameWidth := bounds.Dx() / frames
	result := image.Rectangle{}
	for frame := 0; frame < frames; frame++ {
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < frameWidth; x++ {
				_, _, _, a := source.At(bounds.Min.X+frame*frameWidth+x, bounds.Min.Y+y).RGBA()
				if a <= 4096 {
					continue
				}
				p := image.Rect(x, y, x+1, y+1)
				if result.Empty() {
					result = p
				} else {
					result = result.Union(p)
				}
			}
		}
	}
	return result
}

func copyImage(source image.Image, bounds image.Rectangle) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), source, bounds.Min, draw.Src)
	return out
}

func resizeNearest(source image.Image, width, height int) *image.NRGBA {
	b := source.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := b.Min.X + (2*x+1)*b.Dx()/(2*width)
			sy := b.Min.Y + (2*y+1)*b.Dy()/(2*height)
			out.Set(x, y, source.At(sx, sy))
		}
	}
	return out
}

func addOutline(source image.Image, width int) *image.NRGBA {
	b := source.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()+2*width, b.Dy()+2*width))
	solid := func(x, y int) bool {
		if x < 0 || y < 0 || x >= b.Dx() || y >= b.Dy() {
			return false
		}
		_, _, _, a := source.At(b.Min.X+x, b.Min.Y+y).RGBA()
		return a >= 64*257
	}
	for y := -width; y < b.Dy()+width; y++ {
		for x := -width; x < b.Dx()+width; x++ {
			if solid(x, y) {
				continue
			}
			outlined := false
			for d := 1; d <= width && !outlined; d++ {
				outlined = solid(x-d, y) || solid(x+d, y) || solid(x, y-d) || solid(x, y+d)
			}
			if outlined {
				out.SetNRGBA(x+width, y+width, color.NRGBA{0, 0, 0, 255})
			}
		}
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if solid(x, y) {
				out.Set(x+width, y+width, source.At(b.Min.X+x, b.Min.Y+y))
			}
		}
	}
	return out
}

func joinHorizontal(images []image.Image) *image.NRGBA {
	width, height := 0, 0
	for _, im := range images {
		width += im.Bounds().Dx()
		height = max(height, im.Bounds().Dy())
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	x := 0
	for _, im := range images {
		b := im.Bounds()
		draw.Draw(out, image.Rect(x, 0, x+b.Dx(), b.Dy()), im, b.Min, draw.Src)
		x += b.Dx()
	}
	return out
}

func packAtlas(inputs []string, output string, spec atlasValue, pivot image.Point, sourcePivot anchorValue) error {
	if len(inputs) > spec.grid.X*spec.grid.Y {
		return fmt.Errorf("%d sprites do not fit %dx%d atlas", len(inputs), spec.grid.X, spec.grid.Y)
	}
	out := image.NewNRGBA(image.Rect(0, 0, spec.cell.X*spec.grid.X, spec.cell.Y*spec.grid.Y))
	for i, path := range inputs {
		im, err := decodePNG(path)
		if err != nil {
			return err
		}
		b := im.Bounds()
		source := image.Pt(sourcePivot.x.resolve(b.Dx()), sourcePivot.y.resolve(b.Dy()))
		offset := pivot.Sub(source)
		if offset.X < 0 || offset.Y < 0 || offset.X+b.Dx() > spec.cell.X || offset.Y+b.Dy() > spec.cell.Y {
			return fmt.Errorf("sprite %s (%dx%d) does not fit %dx%d cell at pivot %v", path, b.Dx(), b.Dy(), spec.cell.X, spec.cell.Y, pivot)
		}
		cell := image.Pt(i%spec.grid.X*spec.cell.X, i/spec.grid.X*spec.cell.Y)
		target := image.Rectangle{Min: cell.Add(offset), Max: cell.Add(offset).Add(b.Size())}
		draw.Draw(out, target, im, b.Min, draw.Src)
	}
	return encodePNG(output, out)
}
