package gui

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
)

// Where snow lies (a scene's or area's Snow, MapArt.Snow) the trees and bushes are the snow pictures that
// tools/snow_props.py makes beside the originals: same size and pivot, under the name <name>_snow. The leaves turn
// white and the bark grey; the swaying strips have a snow version of every frame. Without map/props_snow.json the
// originals stay.

// snowGrass is the grass of a snowy day when the map or the area names none.
var snowGrass = content.GrassTint{Sat: 0.04, Val: 1.8}

const snowSuffix = "_snow"

// snowBambooLeaves is which of the bamboo's snow versions the game draws: tools/snow_props.py makes them with no
// leaves (a), a third of them (b) and all of them in white with a green cast (c), as <name>_snow_<letter>.
var snowBambooLeaves = "c"

// snowBase is the prop a snow picture was made from; ok is false for a bamboo version not drawn.
func snowBase(name string) (string, bool) {
	if base, ok := strings.CutSuffix(name, snowSuffix); ok {
		return base, true
	}
	if i := strings.LastIndex(name, snowSuffix+"_"); i >= 0 {
		return name[:i], name[i+len(snowSuffix)+1:] == snowBambooLeaves
	}
	return "", false
}

type snowArt struct {
	sprites map[string]Sprite     // by the original's name
	strips  map[string]FrameStrip // by the original's name, frames only
}

func loadSnowArt(dir string) (*snowArt, error) {
	b, err := os.ReadFile(filepath.Join(dir, "map", "props_snow.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var meta struct {
		Image   string
		Sprites map[string]struct {
			Rect  [4]int
			Pivot [2]float64
		}
		Anims map[string]struct {
			Rect   [4]int
			Frames int
			MS     int
			Pivot  [2]float64
		}
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	pix, err := loadRGBA(filepath.Join(dir, meta.Image))
	if err != nil {
		return nil, err
	}
	sheet := ebiten.NewImageFromImage(pix)
	s := &snowArt{sprites: map[string]Sprite{}, strips: map[string]FrameStrip{}}
	for name, e := range meta.Sprites {
		base, ok := snowBase(name)
		if !ok {
			continue
		}
		r := e.Rect
		s.sprites[base] = Sprite{cut(sheet, image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])), e.Pivot[0], e.Pivot[1]}
	}
	for name, e := range meta.Anims {
		base, ok := snowBase(name)
		if !ok {
			continue
		}
		strip := FrameStrip{MS: e.MS}
		x, y, w, h := e.Rect[0], e.Rect[1], e.Rect[2], e.Rect[3]
		for i := range e.Frames {
			strip.Frames = append(strip.Frames, Sprite{cut(sheet, image.Rect(x+i*w, y, x+(i+1)*w, y+h)), e.Pivot[0], e.Pivot[1]})
		}
		s.strips[base] = strip
	}
	return s, nil
}

// snowAssets is a with the snow pictures standing in the places of the trees and bushes they were made from: the
// shadows, pivots and frame times stay the originals', so the plan lays out the same.
func (a *Assets) snowAssets() *Assets {
	if a.snowArt == nil {
		return a
	}
	if a.snowCopy != nil {
		return a.snowCopy
	}
	c := *a
	c.grassTints = nil // the grass copies of these assets hold the snow pictures
	c.Props = make(map[string]Prop, len(a.Props))
	for name, p := range a.Props {
		if s, ok := a.snowArt.sprites[name]; ok {
			p.Sprite = s
		}
		c.Props[name] = p
	}
	c.Anims = make(map[string]FrameStrip, len(a.Anims))
	for name, strip := range a.Anims {
		if s, ok := a.snowArt.strips[name]; ok && len(s.Frames) == len(strip.Frames) {
			strip.Frames = s.Frames
		}
		c.Anims[name] = strip
	}
	c.snowCopy = &c
	a.snowCopy = &c
	return &c
}
