package sprite

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Art is one sprite drawn with one palette: a row per facing, animations side by side.
type Art struct {
	Name        string
	Pivot       [2]float64
	ShadowPivot [2]float64
	Animations  map[string]Animation
	cell        [2]int
	shadowCell  [2]int
	rows        map[string]int
	sheet       *ebiten.Image
	shadow      *ebiten.Image
	top         float64
}

func newArt(s *Sheet, p Palette, shadow *ebiten.Image) *Art {
	sd := s.Shadow()
	a := &Art{Name: s.Name, Pivot: [2]float64{float64(s.Pivot[0]), float64(s.Pivot[1])},
		ShadowPivot: [2]float64{float64(sd.Pivot[0]), float64(sd.Pivot[1])}, Animations: s.Animations,
		cell: s.Cell, shadowCell: sd.Cell, rows: map[string]int{}, shadow: shadow}
	for i, d := range s.Rows {
		a.rows[d] = i
	}
	img := s.Render(p)
	a.sheet = ebiten.NewImageFromImage(img)
	a.top = a.Pivot[1]
	w, h := s.Cell[0], s.Cell[1]
	col, row := s.Animations["idle"].FirstColumn, a.row("S")
scan:
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if img.RGBAAt(col*w+x, row*h+y).A > 0 {
				a.top = a.Pivot[1] - float64(y)
				break scan
			}
		}
	}
	return a
}

// FrameIndex is the frame shown ms after an animation started; one-shot animations hold the last frame.
func (a *Art) FrameIndex(anim string, ms float64) int {
	an := a.Animations[anim]
	total := 0
	for _, d := range an.MS {
		total += d
	}
	if an.Loop {
		ms = float64(int(ms) % max(1, total))
	} else if ms >= float64(total) {
		return len(an.MS) - 1
	}
	for i, d := range an.MS {
		if ms < float64(d) {
			return i
		}
		ms -= float64(d)
	}
	return len(an.MS) - 1
}

// Length is an animation's duration in seconds.
func (a *Art) Length(anim string) float64 {
	total := 0
	for _, d := range a.Animations[anim].MS {
		total += d
	}
	return float64(total) / 1000
}

// row is the sheet row of a facing. Scene sprites have only the diagonals; a straight
// facing falls back to one next to it, toward the viewer when it can.
func (a *Art) row(dir string) int {
	if r, ok := a.rows[dir]; ok {
		return r
	}
	return a.rows[map[string]string{"N": "NW", "E": "SE", "S": "SW", "W": "SW"}[dir]]
}

func (a *Art) Frame(dir, anim string, i int) *ebiten.Image {
	w, h := a.cell[0], a.cell[1]
	col, row := a.Animations[anim].FirstColumn+i, a.row(dir)
	return a.sheet.SubImage(image.Rect(col*w, row*h, (col+1)*w, (row+1)*h)).(*ebiten.Image)
}

func (a *Art) Shadow(dir, anim string, i int) *ebiten.Image {
	w, h := a.shadowCell[0], a.shadowCell[1]
	col, row := a.Animations[anim].FirstColumn+i, a.row(dir)
	return a.shadow.SubImage(image.Rect(col*w, row*h, (col+1)*w, (row+1)*h)).(*ebiten.Image)
}

// TopOffset is how far above the pivot the idle south frame's first opaque row is (for HP bars).
func (a *Art) TopOffset() float64 { return a.top }

// Library caches Arts by sprite id and palette, and each sprite's tinted shadow.
type Library struct {
	sheets    map[string]*Sheet
	shadowRGB color.RGBA
	mu        sync.Mutex
	arts      map[string]*Art
	shadows   map[string]*ebiten.Image
}

func NewLibrary(sheets map[string]*Sheet, shadowRGB color.RGBA) *Library {
	return &Library{sheets: sheets, shadowRGB: shadowRGB, arts: map[string]*Art{}, shadows: map[string]*ebiten.Image{}}
}

// Art is the sprite drawn with palette p over its defaults; nil for an unknown sprite id.
func (l *Library) Art(id string, p Palette) *Art {
	s := l.sheets[id]
	if s == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	key := id + "|" + p.Key()
	if a, ok := l.arts[key]; ok {
		return a
	}
	sh, ok := l.shadows[id]
	if !ok {
		src := s.Shadow().Image
		im := image.NewRGBA(src.Rect)
		for i := 0; i < len(im.Pix); i += 4 {
			a := uint32(src.Pix[i+3])
			im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = uint8(uint32(l.shadowRGB.R)*a/255), uint8(uint32(l.shadowRGB.G)*a/255), uint8(uint32(l.shadowRGB.B)*a/255), uint8(a)
		}
		sh = ebiten.NewImageFromImage(im)
		l.shadows[id] = sh
	}
	a := newArt(s, p, sh)
	l.arts[key] = a
	return a
}
