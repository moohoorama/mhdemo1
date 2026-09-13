// Package graphics loads prepared sprite sheets and renders a shared draw list.
// It knows nothing about terrain, editing, or source image preprocessing.
package graphics

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/fs"
)

type Sheet struct {
	File       string `json:"file"`
	CellWidth  int    `json:"cell_width"`
	CellHeight int    `json:"cell_height"`
	Columns    int    `json:"columns"`
	Rows       int    `json:"rows"`
}

type Sprite struct {
	Sheet int         `json:"sheet"`
	Cell  int         `json:"cell"`
	Pivot image.Point `json:"pivot"`
}

type Animation struct {
	Frames     []int `json:"frames"`
	FrameTicks int   `json:"frame_ticks"` // One tick is 125ms.
	Loop       bool  `json:"loop"`
}

func (a Animation) Frame(tick int) int {
	n := max(0, tick) / a.FrameTicks
	if a.Loop {
		n %= len(a.Frames)
	} else {
		n = min(n, len(a.Frames)-1)
	}
	return a.Frames[n]
}

type Manifest struct {
	Version    int                  `json:"version"`
	Sheets     []Sheet              `json:"sheets"`
	Sprites    []Sprite             `json:"sprites"`
	Animations map[string]Animation `json:"animations"`
}

type Catalog struct {
	Manifest
	Images []image.Image
}

func (m Manifest) Validate() error {
	if m.Version != 1 || len(m.Sheets) == 0 || len(m.Sprites) == 0 {
		return fmt.Errorf("invalid asset catalog version or empty catalog")
	}
	for i, s := range m.Sheets {
		if !fs.ValidPath(s.File) || s.CellWidth < 1 || s.CellHeight < 1 || s.Columns < 1 || s.Rows < 1 || s.CellWidth > 4096 || s.CellHeight > 4096 || s.Columns > 256 || s.Rows > 256 {
			return fmt.Errorf("invalid sheet %d", i)
		}
	}
	for i, s := range m.Sprites {
		if s.Sheet < 0 || s.Sheet >= len(m.Sheets) {
			return fmt.Errorf("invalid sprite %d sheet", i)
		}
		sheet := m.Sheets[s.Sheet]
		if s.Cell < 0 || s.Cell >= sheet.Columns*sheet.Rows || s.Pivot.X < 0 || s.Pivot.Y < 0 || s.Pivot.X > sheet.CellWidth || s.Pivot.Y > sheet.CellHeight {
			return fmt.Errorf("invalid sprite %d cell or pivot", i)
		}
	}
	for name, a := range m.Animations {
		if len(a.Frames) == 0 || a.FrameTicks < 1 {
			return fmt.Errorf("invalid animation %q", name)
		}
		for _, i := range a.Frames {
			if i < 0 || i >= len(m.Sprites) {
				return fmt.Errorf("invalid animation %q sprite", name)
			}
		}
	}
	return nil
}

func Load(files fs.FS) (*Catalog, error) {
	b, err := fs.ReadFile(files, "catalog.json")
	if err != nil {
		return nil, err
	}
	c := &Catalog{}
	if err := json.Unmarshal(b, &c.Manifest); err != nil {
		return nil, err
	}
	if err := c.Manifest.Validate(); err != nil {
		return nil, err
	}
	for _, s := range c.Sheets {
		f, err := files.Open(s.File)
		if err != nil {
			return nil, err
		}
		im, err := png.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.File, err)
		}
		if im.Bounds().Size() != image.Pt(s.CellWidth*s.Columns, s.CellHeight*s.Rows) {
			return nil, fmt.Errorf("%s: sheet dimensions do not match catalog", s.File)
		}
		c.Images = append(c.Images, im)
	}
	return c, nil
}

func (c *Catalog) Rect(id int) image.Rectangle {
	sprite := c.Sprites[id]
	sheet := c.Sheets[sprite.Sheet]
	p := image.Pt(sprite.Cell%sheet.Columns*sheet.CellWidth, sprite.Cell/sheet.Columns*sheet.CellHeight)
	return image.Rectangle{Min: p, Max: p.Add(image.Pt(sheet.CellWidth, sheet.CellHeight))}
}
