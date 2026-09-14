// tilegenerator writes the project's procedural terrain sprite sheet.
package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tilegenerator output.png")
		os.Exit(2)
	}
	if err := writeTerrain(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}

func writeTerrain(path string) error {
	atlas := image.NewNRGBA(image.Rect(0, 0, 16*8, 8*5))
	for i := 0; i < assetCount; i++ {
		p := image.Pt(i%8*16, i/8*8)
		draw.Draw(atlas, image.Rectangle{Min: p, Max: p.Add(image.Pt(16, 8))}, asset(i), image.Point{}, draw.Src)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, atlas); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
