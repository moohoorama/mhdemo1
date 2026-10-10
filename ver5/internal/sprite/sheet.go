// Package sprite loads baked .spr sheets (see docs/sprite-binary.md) and renders them with per-group colours.
package sprite

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	magic   = "MHSP"
	version = 1
	maxSize = 256 << 20
)

type Animation struct {
	FirstColumn int   `json:"first_column"`
	Frames      int   `json:"frames"`
	MS          []int `json:"ms"`
	Loop        bool  `json:"loop"`
}

// Tag is one entry of the tag table: pixels tagged Index take the colour of Group's Shade.
type Tag struct {
	Group string
	Shade int
	Color color.RGBA // default, RGB only
}

// Sheet is a baked sprite: Rows x Columns cells of Cell pixels, each a palette-index plane and a tag plane.
type Sheet struct {
	ID, Name   string
	Cell       [2]int
	Pivot      [2]int
	Rows       []string
	Columns    int
	Animations map[string]Animation
	Tags       []Tag        // Tags[i] is tag index i+1
	Colors     []color.RGBA // palette (non-premultiplied); index 0 is transparent
	pix, tag   []byte       // cell-major planes, row-major inside a cell
	shadow     *Shadow
	once       sync.Once
}

type reader struct {
	b   []byte
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil || n < 0 || n > len(r.b) {
		r.err = io.ErrUnexpectedEOF
		return make([]byte, max(n, 0)&0xff)
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}
func (r *reader) u8() int  { return int(r.take(1)[0]) }
func (r *reader) u16() int { return int(binary.LittleEndian.Uint16(pad(r.take(2)))) }
func (r *reader) u32() int { return int(binary.LittleEndian.Uint32(pad(r.take(4)))) }
func (r *reader) str() string {
	return string(r.take(r.u16()))
}
func (r *reader) rgba() color.RGBA {
	c := pad(r.take(4))
	return color.RGBA{c[0], c[1], c[2], c[3]}
}

func pad(b []byte) []byte { return append(b[:len(b):len(b)], 0, 0, 0, 0) }

func Load(path string) (*Sheet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// LoadDir loads every *.spr in dir, keyed by file stem.
func LoadDir(dir string) (map[string]*Sheet, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.spr"))
	if err != nil {
		return nil, err
	}
	out := map[string]*Sheet{}
	for _, p := range paths {
		if out[strings.TrimSuffix(filepath.Base(p), ".spr")], err = Load(p); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func Decode(b []byte) (*Sheet, error) {
	if len(b) < 6 || string(b[:4]) != magic {
		return nil, fmt.Errorf("not a sprite file (bad magic)")
	}
	if v := binary.LittleEndian.Uint16(b[4:]); v != version {
		return nil, fmt.Errorf("unsupported version %d (want %d)", v, version)
	}
	z, err := zlib.NewReader(bytes.NewReader(b[6:]))
	if err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}
	defer z.Close()
	payload, err := io.ReadAll(io.LimitReader(z, maxSize))
	if err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}
	r := &reader{b: payload}
	s := &Sheet{Animations: map[string]Animation{}}
	s.ID, s.Name = r.str(), r.str()
	s.Cell = [2]int{r.u16(), r.u16()}
	s.Pivot = [2]int{r.u16(), r.u16()}
	for i := r.u8(); i > 0 && r.err == nil; i-- {
		s.Rows = append(s.Rows, r.str())
	}
	type anim struct {
		name string
		Animation
	}
	var anims []anim
	for i := r.u8(); i > 0 && r.err == nil; i-- {
		a := anim{name: r.str()}
		a.FirstColumn, a.Frames, a.Loop = r.u16(), r.u16(), r.u8() != 0
		for j := r.u8(); j > 0 && r.err == nil; j-- {
			a.MS = append(a.MS, r.u32())
		}
		anims = append(anims, a)
	}
	s.Tags = make([]Tag, r.u8())
	seen := make([]bool, len(s.Tags))
	for range s.Tags {
		i := r.u8()
		t := Tag{Group: r.str(), Shade: r.u8(), Color: r.rgba()}
		if r.err != nil {
			break
		}
		if i < 1 || i > len(s.Tags) || seen[i-1] {
			return nil, fmt.Errorf("bad tag index %d", i)
		}
		seen[i-1], s.Tags[i-1] = true, t
	}
	s.Colors = make([]color.RGBA, r.u16())
	for i := range s.Colors {
		s.Colors[i] = r.rgba()
	}
	s.Columns = r.u16()
	if r.err != nil {
		return nil, fmt.Errorf("header: %w", r.err)
	}
	w, h := s.Cell[0], s.Cell[1]
	if w == 0 || h == 0 || len(s.Rows) == 0 || s.Columns == 0 || len(s.Colors) == 0 {
		return nil, fmt.Errorf("empty sheet (cell %dx%d, %d rows, %d columns, %d colours)", w, h, len(s.Rows), s.Columns, len(s.Colors))
	}
	if s.Pivot[0] > w || s.Pivot[1] > h {
		return nil, fmt.Errorf("pivot %v outside cell %dx%d", s.Pivot, w, h)
	}
	for _, a := range anims {
		if a.Frames == 0 || len(a.MS) != a.Frames || a.FirstColumn+a.Frames > s.Columns {
			return nil, fmt.Errorf("animation %q: %d frames, %d durations, first column %d of %d", a.name, a.Frames, len(a.MS), a.FirstColumn, s.Columns)
		}
		s.Animations[a.name] = a.Animation
	}
	n := len(s.Rows) * s.Columns * w * h
	if len(r.b) != 2*n {
		return nil, fmt.Errorf("pixel data is %d bytes, want %d (truncated or trailing)", len(r.b), 2*n)
	}
	s.pix, s.tag = make([]byte, n), make([]byte, n)
	for c, cell := 0, w*h; c < len(s.Rows)*s.Columns; c++ {
		copy(s.pix[c*cell:], r.take(cell))
		copy(s.tag[c*cell:], r.take(cell))
	}
	for i := range s.pix {
		if int(s.pix[i]) >= len(s.Colors) || int(s.tag[i]) > len(s.Tags) {
			return nil, fmt.Errorf("pixel %d: palette index %d of %d or tag %d of %d out of range", i, s.pix[i], len(s.Colors), s.tag[i], len(s.Tags))
		}
	}
	return s, nil
}
