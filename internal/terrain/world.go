package terrain

import (
	"math"
)

// Kind names editor presets. Cell.Ground accepts only River, Grass and Wasteland.
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

// Kind remains the editor's brush preset; persistent cells separate the two axes.
type Decoration uint8

const (
	NoDecoration Decoration = iota
	Rocks
	Trees
)

type Cell struct {
	Ground     Kind       `json:"ground"`
	Decoration Decoration `json:"decoration"`
}

func (c Cell) String() string {
	switch c.Decoration {
	case Rocks:
		return "ROCK / " + c.Ground.String()
	case Trees:
		return "TREES / " + c.Ground.String()
	default:
		return c.Ground.String()
	}
}

func (k Kind) Cell() Cell {
	c := Cell{Ground: k.Base()}
	switch k {
	case RockGrass, RockWasteland:
		c.Decoration = Rocks
	case Forest:
		c.Decoration = Trees
	}
	return c
}

const MapVersion = 5

type World struct {
	Version       int
	Width, Height int
	cells         []Cell
	revision      uint64
}

func New(w, h int) *World {
	world := &World{Version: MapVersion, Width: w, Height: h, cells: make([]Cell, w*h)}
	for i := range world.cells {
		world.cells[i] = Cell{Ground: Wasteland}
	}
	return world
}
func (w *World) Clone() *World        { c := *w; c.cells = append([]Cell(nil), w.cells...); return &c }
func (w *World) Revision() uint64     { return w.revision }
func (w *World) Inside(x, y int) bool { return x >= 0 && y >= 0 && x < w.Width && y < w.Height }
func (w *World) Cell(x, y int) Cell {
	if !w.Inside(x, y) {
		return Cell{Ground: River}
	}
	return w.cells[y*w.Width+x]
}
func (w *World) Terrain(x, y int) Kind {
	c := w.Cell(x, y)
	switch c.Decoration {
	case Rocks:
		if c.Ground == Grass {
			return RockGrass
		}
		return RockWasteland
	case Trees:
		return Forest
	}
	return c.Ground
}
func (w *World) SetCell(x, y int, c Cell) bool {
	if !w.Inside(x, y) || c.Ground > Wasteland || c.Decoration > Trees || (c.Ground == River && c.Decoration != NoDecoration) {
		return false
	}
	i := y*w.Width + x
	if w.cells[i] == c {
		return false
	}
	w.cells[i] = c
	w.revision++
	return true
}
func (w *World) SetTerrain(x, y int, k Kind) bool {
	if k > Forest {
		return false
	}
	return w.SetCell(x, y, k.Cell())
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
			if w.Inside(x, y) && w.Cell(x, y).Ground >= kind {
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
