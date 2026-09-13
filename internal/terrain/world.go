package terrain

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// Kind is the editable terrain of one large diamond. Base controls ground blending.
type Kind uint8

const (
	River Kind = iota
	Grass
	Wasteland
	RockGrass
	RockWasteland
	Forest
)

func (k Kind) String() string {
	switch k {
	case River:
		return "RIVER"
	case Grass:
		return "GRASSLAND"
	case RockGrass:
		return "ROCK / GRASS"
	case RockWasteland:
		return "ROCK / SOIL"
	case Forest:
		return "TREES / GRASS"
	default:
		return "WASTELAND"
	}
}

func (k Kind) Base() Kind {
	switch k {
	case RockGrass, Forest:
		return Grass
	case RockWasteland:
		return Wasteland
	default:
		return k
	}
}

type World struct {
	Version int    `json:"version"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Cells   []Kind `json:"terrain_cells"`
}

func New(w, h int) *World {
	world := &World{4, w, h, make([]Kind, w*h)}
	for i := range world.Cells {
		world.Cells[i] = Wasteland
	}
	return world
}
func (w *World) Clone() *World        { c := *w; c.Cells = append([]Kind(nil), w.Cells...); return &c }
func (w *World) Inside(x, y int) bool { return x >= 0 && y >= 0 && x < w.Width && y < w.Height }
func (w *World) Terrain(x, y int) Kind {
	if !w.Inside(x, y) {
		return River
	}
	return w.Cells[y*w.Width+x]
}
func (w *World) SetTerrain(x, y int, k Kind) bool {
	if !w.Inside(x, y) || k > Forest {
		return false
	}
	i := y*w.Width + x
	if w.Cells[i] == k {
		return false
	}
	w.Cells[i] = k
	return true
}

// Binary helpers for legacy river/wasteland maps.
func (w *World) At(x, y int) bool { return w.Inside(x, y) && w.Terrain(x, y) == River }
func (w *World) Set(x, y int, water bool) bool {
	k := Wasteland
	if water {
		k = River
	}
	return w.SetTerrain(x, y, k)
}

// Vertex evaluates a point on the half-cell lattice, ignoring outside cells.
func (w *World) Vertex(sx, sy int) bool { return w.vertexAtLeast(sx, sy, Wasteland) }
func (w *World) vertexAtLeast(sx, sy int, kind Kind) bool {
	for y := (sy - 1) / 2; y <= sy/2; y++ {
		for x := (sx - 1) / 2; x <= sx/2; x++ {
			// At the origin, truncation skips only the outside neighbor.
			if w.Inside(x, y) && w.Terrain(x, y).Base() >= kind {
				return true
			}
		}
	}
	return false
}

// Masks are ordered top, right, left, bottom (row-major in map coordinates).
func (w *World) Masks(x, y int) [4]int { return w.MasksFor(x, y, Wasteland) }
func (w *World) MasksFor(x, y int, kind Kind) [4]int {
	var masks [4]int
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			for bit, p := range [][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
				if w.vertexAtLeast(2*x+i+p[0], 2*y+j+p[1], kind) {
					masks[j*2+i] |= 1 << bit
				}
			}
		}
	}
	return masks
}
func Project(u, v float64) (float64, float64)   { return (u - v) * 16, (u + v) * 8 }
func Unproject(x, y float64) (float64, float64) { return x/32 + y/16, y/16 - x/32 }
func CellAt(x, y float64) (int, int) {
	u, v := Unproject(x, y)
	return int(math.Floor(u)), int(math.Floor(v))
}
func Demo() *World {
	w := New(28, 22)
	for y := 0; y <= w.Height; y++ {
		for x := 0; x <= w.Width; x++ {
			center := 17 + 4*math.Sin(float64(y)*.23)
			river := math.Abs(float64(x)-center) < 2.0+0.6*math.Sin(float64(y)*.4)
			lake := math.Pow((float64(x)-10)/4.8, 2)+math.Pow((float64(y)-8)/3.2, 2) < 1
			branch := y > 7 && y < 17 && math.Abs(float64(x)-(11+float64(y-8)*.85)) < 1.3
			w.Set(x, y, river || lake || branch)
			if !w.At(x, y) && (math.Pow((float64(x)-9)/9, 2)+math.Pow((float64(y)-9)/8, 2) < 1 || x < 5 && y > 11) {
				w.SetTerrain(x, y, Grass)
			}
		}
	}
	// Showcase compact woodland and both rocky ground types.
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			if w.Terrain(x, y) == Grass && x > 3 && x < 9 && y > 10 && y < 16 {
				w.SetTerrain(x, y, Forest)
			}
			if w.Terrain(x, y) == Grass && x > 11 && x < 15 && y > 3 && y < 7 {
				w.SetTerrain(x, y, RockGrass)
			}
			if w.Terrain(x, y) == Wasteland && x > 20 && x < 25 && y > 13 && y < 18 {
				w.SetTerrain(x, y, RockWasteland)
			}
		}
	}
	return w
}
func (w *World) Save(path string) error {
	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	// Atomic replacement avoids leaving half a map on an interrupted save.
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, append(b, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func Load(path string) (*World, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data struct {
		Version  int    `json:"version"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Cells    []bool `json:"water_cells"`
		Vertices []bool `json:"water_vertices"`
		Terrain  []Kind `json:"terrain_cells"`
	}
	if err = json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	if data.Width < 1 || data.Height < 1 || data.Width > 256 || data.Height > 256 {
		return nil, fmt.Errorf("invalid map dimensions")
	}
	w := New(data.Width, data.Height)
	switch data.Version {
	case 3, 4:
		if len(data.Terrain) != w.Width*w.Height {
			return nil, fmt.Errorf("invalid terrain count")
		}
		for _, k := range data.Terrain {
			if k > Forest || (data.Version == 3 && k > Wasteland) {
				return nil, fmt.Errorf("invalid terrain kind %d", k)
			}
		}
		w.Cells = data.Terrain
	case 2:
		if len(data.Cells) != w.Width*w.Height {
			return nil, fmt.Errorf("invalid cell count")
		}
		for i, v := range data.Cells {
			w.Set(i%w.Width, i/w.Width, v)
		}
	case 1:
		if len(data.Vertices) != (w.Width+1)*(w.Height+1) {
			return nil, fmt.Errorf("invalid vertex count")
		}
		for y := 0; y < w.Height; y++ {
			for x := 0; x < w.Width; x++ {
				n := 0
				for _, p := range [][2]int{{x, y}, {x + 1, y}, {x + 1, y + 1}, {x, y + 1}} {
					if data.Vertices[p[1]*(w.Width+1)+p[0]] {
						n++
					}
				}
				w.Set(x, y, n >= 2)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported map version %d", data.Version)
	}
	return w, nil
}
