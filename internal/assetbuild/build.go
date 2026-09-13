package assetbuild

import (
	"demo1/internal/graphics"
	"demo1/internal/terrain"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

// Generate only writes its output directory: source art and maps are never changed.
func Generate(source, out string) error {
	objects, err := LoadObjects(os.DirFS(source))
	if err != nil {
		return err
	}
	c := &graphics.Catalog{Manifest: graphics.Manifest{Version: 1, Sheets: []graphics.Sheet{
		{File: "terrain.png", CellWidth: 16, CellHeight: 8, Columns: 8, Rows: 5},
		{File: "objects.png", CellWidth: 24, CellHeight: 32, Columns: 8, Rows: 4},
	}, Animations: map[string]graphics.Animation{}}}
	for _, s := range c.Sheets {
		c.Images = append(c.Images, image.NewNRGBA(image.Rect(0, 0, s.CellWidth*s.Columns, s.CellHeight*s.Rows)))
	}
	pack := func(sheet, cell int, im image.Image, pivot, oldPivot image.Point) error {
		spec := c.Sheets[sheet]
		offset := pivot.Sub(oldPivot)
		if offset.X < 0 || offset.Y < 0 || offset.X+im.Bounds().Dx() > spec.CellWidth || offset.Y+im.Bounds().Dy() > spec.CellHeight {
			return fmt.Errorf("sprite %d does not fit sheet %d", cell, sheet)
		}
		p := image.Pt(cell%spec.Columns*spec.CellWidth, cell/spec.Columns*spec.CellHeight).Add(offset)
		draw.Draw(c.Images[sheet].(*image.NRGBA), image.Rectangle{Min: p, Max: p.Add(im.Bounds().Size())}, im, im.Bounds().Min, draw.Src)
		c.Sprites = append(c.Sprites, graphics.Sprite{Sheet: sheet, Cell: cell, Pivot: pivot})
		return nil
	}
	for i := 0; i < terrain.AssetCount; i++ {
		if err := pack(0, i, Asset(i), image.Pt(8, 0), image.Pt(8, 0)); err != nil {
			return err
		}
	}
	for i, im := range objects {
		if err := pack(1, i, im, image.Pt(12, 28), image.Pt(im.Bounds().Dx()/2, im.Bounds().Dy()-2)); err != nil {
			return err
		}
	}
	c.Animations["water"] = graphics.Animation{Frames: []int{0, 1, 2, 3, 4, 5, 6, 7}, FrameTicks: 1, Loop: true}
	for object := terrain.SmallRock; object <= terrain.SmallTree; object++ {
		a := graphics.Animation{FrameTicks: terrain.TreeFrameTicks, Loop: true}
		count := 1
		if object >= terrain.GrassTuft1 {
			count = 4
		}
		for f := 0; f < count; f++ {
			a.Frames = append(a.Frames, terrain.AssetCount+terrain.ObjectSprite(object, f))
		}
		c.Animations[terrain.ObjectAnimation(object)] = a
	}
	if err := c.Manifest.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for i, s := range c.Sheets {
		if err := save(filepath.Join(out, s.File), c.Images[i]); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(c.Manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "catalog.json"), append(b, '\n'), 0644); err != nil {
		return err
	}
	scene := terrain.BuildMapScene(terrain.Demo())
	items, err := scene.BuildDrawList(c, 0, 0, nil, nil)
	if err != nil {
		return err
	}
	return save(filepath.Join(out, "preview.png"), graphics.RenderPNG(c, items, graphics.SceneBounds(c, items, scene.Bounds), 2))
}

func save(path string, im image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, im); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
