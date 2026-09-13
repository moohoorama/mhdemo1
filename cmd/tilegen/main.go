// tilegen deliberately uses exact pixels, no resampling or image generation API.
package main

import (
	"demo1/internal/terrain"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func save(path string, im image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, im); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
}
func main() {
	if err := os.MkdirAll("assets/tiles", 0755); err != nil {
		panic(err)
	}
	old, _ := filepath.Glob("assets/tiles/tile_*.png")
	for _, path := range old {
		if err := os.Remove(path); err != nil {
			panic(err)
		}
	}
	var tiles [terrain.AssetCount]image.Image
	atlas := image.NewNRGBA(image.Rect(0, 0, 128, 40))
	for m := 0; m < terrain.AssetCount; m++ {
		tiles[m] = terrain.Asset(m)
		save(filepath.Join("assets/tiles", terrain.AssetName(m)), tiles[m])
		p := image.Pt(m%8*16, m/8*8)
		draw.Draw(atlas, image.Rect(p.X, p.Y, p.X+16, p.Y+8), tiles[m], image.Point{}, draw.Src)
	}
	save("assets/atlas.png", atlas)
	preview := terrain.Enlarge(atlas, 12)
	bg := image.NewNRGBA(preview.Bounds())
	draw.Draw(bg, bg.Bounds(), image.NewUniform(color.NRGBA{24, 33, 37, 255}), image.Point{}, draw.Src)
	draw.Draw(bg, bg.Bounds(), preview, image.Point{}, draw.Over)
	save("assets/atlas-preview.png", bg)
	w := terrain.Demo()
	objects, err := terrain.LoadObjects(os.DirFS("assets/objects"))
	if err != nil {
		panic(err)
	}
	save("assets/objects-preview.png", terrain.SpritePreview(objects))
	save("assets/demo-preview.png", terrain.RenderScene(w, tiles, objects, 2, 0, 0))
	if err := w.Save("example-map.json"); err != nil {
		panic(err)
	}
	fmt.Println("Generated 38 exact 16x8 PNGs, atlas, previews and example-map.json")
}
