// Package maplight draws a scene map's whole canvas to an image.RGBA from the rules of internal/scenemap, with no
// ebiten, window or GPU, so a server can render maps from any goroutine. It lays the plan's pictures the way
// internal/gui does and leaves out what only the running game has: animation (every strip shows its first frame),
// the day's light and the people.
package maplight

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"srpg/internal/content"
	"srpg/internal/scenemap"
)

// Sprite is a picture cut from a sheet with the point that lands on the drawing position.
type Sprite struct {
	Image          *image.RGBA
	PivotX, PivotY float64
}

// Atlas holds the pictures of assets/graphics a scene map draws.
type Atlas struct {
	Catalog    *scenemap.Catalog
	Tiles      map[int]Sprite
	Water      []int // tile ids of the water animation
	Structures map[string]Sprite
	Props      map[string]Sprite
	Scenery    map[string]Sprite
	Anims      map[string][]Sprite // first frame first
	Shade      color.RGBA          // the colour of ground shadows
	shadowSrc  map[string]shadowSource
	dir        string
	cache      *cache
}

// cache is what Render makes once and shares between the atlas and its grass-turned copies.
type cache struct {
	mu      sync.Mutex
	tinted  map[content.GrassTint]*Atlas
	shadows map[string]*Sprite
}

type propEntry struct {
	Rect       [4]int
	Pivot      [2]float64
	Shadow     *bool `json:"shadow"`
	ShadowMask *struct {
		Rect  [4]int
		Pivot [2]int
	} `json:"shadow_mask"`
	Layer string
}

// shadowSource is what a prop's ground shadow is cast from: its picture, or the mask its model gave.
type shadowSource struct {
	sheet *image.RGBA
	rect  image.Rectangle
	pivot [2]float64
	mask  *image.Rectangle
	maskP [2]int
}

func loadRGBA(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := image.NewRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out, nil
}

// cut copies the part of a sheet; a copy keeps a sprite free of its sheet and lets a tint turn it alone.
func cut(sheet *image.RGBA, r image.Rectangle) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), sheet, r.Min, draw.Src)
	return out
}

func rect(r [4]int) image.Rectangle { return image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3]) }

// LoadAtlas reads the graphics directory (assets/graphics).
func LoadAtlas(dir string) (*Atlas, error) {
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, err
	}
	var index struct {
		Tileset struct {
			Image      string
			Sprites    map[string][6]int
			Animations map[string]struct{ Frames []int }
		}
		Structures struct {
			Image   string
			Cell    [2]int
			Pivot   [2]float64
			Sprites map[string]int
		}
		Props struct {
			Image   string
			Sprites map[string]propEntry
		}
		Extra struct {
			Image   string
			Sprites map[string]propEntry
		}
		Terrain   string
		ShadowRGB [3]uint8 `json:"shadow_rgb"`
		Anims     map[string]struct {
			Image  string
			Rect   [4]int
			Frames int
			Pivot  [2]float64
		}
	}
	if err := json.Unmarshal(b, &index); err != nil {
		return nil, err
	}
	a := &Atlas{Tiles: map[int]Sprite{}, Structures: map[string]Sprite{}, Props: map[string]Sprite{}, Scenery: map[string]Sprite{},
		Anims: map[string][]Sprite{}, dir: dir, cache: &cache{shadows: map[string]*Sprite{}, tinted: map[content.GrassTint]*Atlas{}},
		shadowSrc: map[string]shadowSource{}}
	a.Shade = color.RGBA{index.ShadowRGB[0], index.ShadowRGB[1], index.ShadowRGB[2], 255}
	if a.Catalog, err = scenemap.LoadCatalog(dir); err != nil {
		return nil, err
	}
	tiles, err := loadRGBA(filepath.Join(dir, index.Tileset.Image))
	if err != nil {
		return nil, err
	}
	for id, r := range index.Tileset.Sprites {
		n, _ := strconv.Atoi(id)
		a.Tiles[n] = Sprite{cut(tiles, rect([4]int{r[0], r[1], r[2], r[3]})), float64(r[4]), float64(r[5])}
	}
	a.Water = index.Tileset.Animations["water"].Frames
	st, err := loadRGBA(filepath.Join(dir, index.Structures.Image))
	if err != nil {
		return nil, err
	}
	w, h := index.Structures.Cell[0], index.Structures.Cell[1]
	for name, i := range index.Structures.Sprites {
		a.Structures[name] = Sprite{cut(st, image.Rect(i*w, 0, (i+1)*w, h)), index.Structures.Pivot[0], index.Structures.Pivot[1]}
	}
	type propSheet struct {
		Image   string
		Sprites map[string]propEntry
	}
	sheets := []propSheet{{index.Props.Image, index.Props.Sprites}, {index.Extra.Image, index.Extra.Sprites}}
	if index.Terrain != "" {
		raw, err := os.ReadFile(filepath.Join(dir, index.Terrain))
		if err != nil {
			return nil, err
		}
		var terrain propSheet
		if err := json.Unmarshal(raw, &terrain); err != nil {
			return nil, fmt.Errorf("%s: %w", index.Terrain, err)
		}
		sheets = append(sheets, terrain)
	}
	for _, s := range sheets {
		if s.Image == "" {
			continue
		}
		pix, err := loadRGBA(filepath.Join(dir, s.Image))
		if err != nil {
			return nil, err
		}
		for name, p := range s.Sprites {
			a.Props[name] = Sprite{cut(pix, rect(p.Rect)), p.Pivot[0], p.Pivot[1]}
			if p.Layer == "ground" || p.Shadow != nil && !*p.Shadow {
				continue
			}
			src := shadowSource{sheet: pix, rect: rect(p.Rect), pivot: p.Pivot}
			if m := p.ShadowMask; m != nil {
				r := rect(m.Rect)
				src.mask, src.maskP = &r, m.Pivot
			}
			a.shadowSrc[name] = src
		}
	}
	sheetOf := map[string]*image.RGBA{}
	for name, e := range index.Anims {
		pix, ok := sheetOf[e.Image]
		if !ok {
			if pix, err = loadRGBA(filepath.Join(dir, e.Image)); err != nil {
				return nil, err
			}
			sheetOf[e.Image] = pix
		}
		x, y, w, h := e.Rect[0], e.Rect[1], e.Rect[2], e.Rect[3]
		for i := range e.Frames {
			a.Anims[name] = append(a.Anims[name], Sprite{cut(pix, image.Rect(x+i*w, y, x+(i+1)*w, y+h)), e.Pivot[0], e.Pivot[1]})
		}
	}
	if err := a.loadScenery(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Atlas) loadScenery() error {
	b, err := os.ReadFile(filepath.Join(a.dir, "map", "scenery.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var meta struct {
		Image   string
		Sprites map[string]propEntry
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return fmt.Errorf("scenery.json: %w", err)
	}
	pix, err := loadRGBA(filepath.Join(a.dir, meta.Image))
	if err != nil {
		return err
	}
	for name, v := range meta.Sprites {
		a.Scenery[name] = Sprite{cut(pix, rect(v.Rect)), v.Pivot[0], v.Pivot[1]}
	}
	return nil
}
