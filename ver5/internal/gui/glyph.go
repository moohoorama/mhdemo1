package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// glyphEntry is one brush-drawn character in the atlas: its pixels, where they sit in the line box and how far the pen moves.
type glyphEntry struct {
	R   [4]int `json:"r"`
	OX  int    `json:"ox"`
	OY  int    `json:"oy"`
	Adv int    `json:"adv"`
}

type glyphAtlas struct {
	Height int                   `json:"height"`
	Space  int                   `json:"space"`
	Glyphs map[string]glyphEntry `json:"glyphs"`
	sheet  *ebiten.Image
	strips map[string]*ebiten.Image
}

// loadGlyphAtlas reads ui/glyphs.png (white ink on alpha) and ui/glyphs.json from the graphics directory.
func loadGlyphAtlas(dir string) (*glyphAtlas, error) {
	b, err := os.ReadFile(filepath.Join(dir, "ui", "glyphs.json"))
	if err != nil {
		return nil, err
	}
	a := &glyphAtlas{strips: map[string]*ebiten.Image{}}
	if err := json.Unmarshal(b, a); err != nil {
		return nil, fmt.Errorf("glyphs.json: %w", err)
	}
	im, err := loadRGBA(filepath.Join(dir, "ui", "glyphs.png"))
	if err != nil {
		return nil, err
	}
	a.sheet = ebiten.NewImageFromImage(im)
	return a, nil
}

type glyphPlace struct {
	e glyphEntry
	x float64
}

// layout places the characters of s on a line box of height h and returns the line's width; ok is false when
// a character has no glyph.
func (a *glyphAtlas) layout(s string, h float64) (places []glyphPlace, width float64, ok bool) {
	k := h / float64(a.Height)
	for _, r := range s {
		if r == ' ' {
			width += float64(a.Space) * k
			continue
		}
		e, found := a.Glyphs[string(r)]
		if !found {
			return nil, 0, false
		}
		places = append(places, glyphPlace{e, width})
		width += float64(e.Adv) * k
	}
	return places, width, true
}

// has reports whether every character of s is drawn by the atlas.
func (a *glyphAtlas) has(s string) bool {
	_, _, ok := a.layout(s, float64(a.Height))
	return ok
}

// text draws s as brush glyphs in fill with a rim in rim, for a line box h device pixels high. It returns nil when
// a character is missing so the caller can fall back to the font.
func (a *glyphAtlas) text(s string, h float64, fill, rim color.NRGBA) *ebiten.Image {
	key := fmt.Sprint(s, "|", h, "|", fill, "|", rim)
	if im, ok := a.strips[key]; ok {
		return im
	}
	places, width, ok := a.layout(s, h)
	if !ok {
		return nil
	}
	k := h / float64(a.Height)
	pad := math.Ceil(h * .08)
	im := ebiten.NewImage(int(math.Ceil(width+pad*2)), int(math.Ceil(h+pad*2)))
	put := func(dx, dy float64, c color.NRGBA) {
		for _, p := range places {
			r := p.e.R
			sub := a.sheet.SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(k, k)
			op.GeoM.Translate(pad+p.x+float64(p.e.OX)*k+dx, pad+float64(p.e.OY)*k+dy)
			op.ColorScale.ScaleWithColor(c)
			im.DrawImage(sub, op)
		}
	}
	rad := math.Max(1, h*.03)
	put(0, rad*1.5, rim)
	for i := range 12 {
		t := float64(i) * math.Pi / 6
		put(math.Cos(t)*rad, math.Sin(t)*rad, rim)
	}
	put(0, 0, fill)
	a.strips[key] = im
	return im
}
