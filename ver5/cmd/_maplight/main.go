// Temporary tool: renders a scene map with internal/maplight and compares it with the game's own canvas.
//
//	go run ./cmd/_maplight S_zhuo out.png [reference.png]
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"srpg/internal/maplight"
	"srpg/internal/scenemap"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: _maplight MAP out.png [reference.png]")
		os.Exit(2)
	}
	a, err := maplight.LoadAtlas("assets/graphics")
	must(err)
	b, err := os.ReadFile("assets/graphics/maps/" + os.Args[1] + ".json")
	must(err)
	m, ok, err := scenemap.Parse(b)
	must(err)
	if !ok {
		must(fmt.Errorf("not a format 2 map"))
	}
	img, _, err := a.Render(m)
	must(err)
	save(os.Args[2], img)
	if len(os.Args) > 3 {
		f, err := os.Open(os.Args[3])
		must(err)
		ref, err := png.Decode(f)
		must(err)
		compare(img, ref, os.Args[2])
	}
}

func compare(img *image.RGBA, ref image.Image, out string) {
	if ref.Bounds() != img.Bounds() {
		fmt.Println("size differs:", img.Bounds(), ref.Bounds())
		return
	}
	diff := image.NewRGBA(img.Bounds())
	n, bad, sum := 0, 0, 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r1, g1, b1, a1 := img.At(x, y).RGBA()
			r2, g2, b2, a2 := ref.At(x, y).RGBA()
			d := abs(int(r1>>8)-int(r2>>8)) + abs(int(g1>>8)-int(g2>>8)) + abs(int(b1>>8)-int(b2>>8)) + abs(int(a1>>8)-int(a2>>8))
			if a1 == 0 && a2 == 0 {
				continue
			}
			n++
			sum += d
			if d > 40 {
				bad++
				diff.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else if d > 0 {
				diff.Set(x, y, color.RGBA{255, 200, 0, 255})
			}
		}
	}
	fmt.Printf("drawn pixels %d, mean diff %.2f, differing >40: %d (%.2f%%)\n", n, float64(sum)/float64(max(n, 1)), bad, 100*float64(bad)/float64(max(n, 1)))
	save(out+".diff.png", diff)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	must(err)
	must(png.Encode(f, img))
	must(f.Close())
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
