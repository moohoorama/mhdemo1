// Package scenemap is the scene map (the story screen's map) format and the rules that turn a map into the ordered
// list of pictures to draw. The scene viewer (cmd/_scene) and the game (internal/gui) both draw from this list, so
// they show the same picture. docs/presentation-scene.md describes the rules.
//
// Coordinates. A small tile (U, V) is half a game cell; u runs down to the right on screen, v down to the left. A
// point on the ground is (cu, cv) in small tiles, tile U spanning [U, U+1), so game cell (u, v) has its centre at
// (2u+1, 2v+1). Canvas pixels (the game's K = 2 canvas) of a ground point: (16(cu-cv), 8(cu+cv)).
package scenemap

import (
	"bytes"
	"encoding/json"
	"fmt"

	"srpg/internal/content"
)

// Map is a scene map: what lies on each small tile and what stands where. Everything else comes from the rules.
type Map struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Format int    `json:"format"`
	// Size is the playing area in game cells, from game cell (0, 0).
	Size [2]int `json:"size"`
	// Origin is the small tile of Ground[0][0]; the grid covers the playing area and the picture round it.
	Origin [2]int `json:"origin"`
	// Ground holds one glyph per small tile: '.' grass, 'd' soil, '~' water, floors (scene.json "floors").
	Ground []string `json:"ground"`
	// Height holds one height step per small tile ("0" when left out); scene.json "heights" draws each step.
	Height  []string `json:"height,omitempty"`
	Objects []Object `json:"objects"`
	Walls   []Wall   `json:"walls,omitempty"`
	// Grass turns the map's grass toward its climate (content.GrassTint); nil keeps the tileset's.
	Grass *content.GrassTint `json:"grass,omitempty"`
}

// Object stands with its anchor DX, DY canvas pixels from the centre of small tile (U, V).
type Object struct {
	Name   string
	U, V   int
	DX, DY int
}

// Wall is a wall line laid one piece per small tile from From to To, along u or v.
type Wall struct {
	Family   string
	From, To [2]int
}

// Format is the format number of the maps this package reads.
const Format = 2

// Parse reads a scene map; ok is false when data is a map of another format.
func Parse(data []byte) (m *Map, ok bool, err error) {
	var head struct {
		Format int `json:"format"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, false, err
	}
	if head.Format != Format {
		return nil, false, nil
	}
	m = &Map{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, true, err
	}
	if len(m.Ground) == 0 {
		return nil, true, fmt.Errorf("%s: no ground", m.ID)
	}
	for _, row := range m.Ground {
		if len(row) != len(m.Ground[0]) {
			return nil, true, fmt.Errorf("%s: ground rows differ in length", m.ID)
		}
	}
	if len(m.Height) > 0 && (len(m.Height) != len(m.Ground) || len(m.Height[0]) != len(m.Ground[0])) {
		return nil, true, fmt.Errorf("%s: height grid does not match the ground", m.ID)
	}
	return m, true, nil
}

// Encode writes the map as the map files are laid out: one key per line, a grid row, an object or a wall line per
// line, numbers packed.
func (m *Map) Encode() ([]byte, error) {
	var b bytes.Buffer
	value := func(v any) error {
		var e bytes.Buffer
		enc := json.NewEncoder(&e)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return err
		}
		b.Write(bytes.TrimSuffix(e.Bytes(), []byte("\n")))
		return nil
	}
	list := func(key string, n int, item func(int) any, last bool) error {
		fmt.Fprintf(&b, " %q: [", key)
		for i := range n {
			b.WriteString("\n  ")
			if err := value(item(i)); err != nil {
				return err
			}
			if i < n-1 {
				b.WriteByte(',')
			}
		}
		if n > 0 {
			b.WriteString("\n ")
		}
		b.WriteString("]")
		if !last {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return nil
	}
	b.WriteString("{\n")
	for _, f := range []struct {
		key string
		v   any
	}{{"id", m.ID}, {"name", m.Name}, {"format", m.Format}, {"size", m.Size}, {"origin", m.Origin}} {
		fmt.Fprintf(&b, " %q: ", f.key)
		if err := value(f.v); err != nil {
			return nil, err
		}
		b.WriteString(",\n")
	}
	if m.Grass != nil {
		b.WriteString(` "grass": `)
		if err := value(m.Grass); err != nil {
			return nil, err
		}
		b.WriteString(",\n")
	}
	rows := func(r []string) func(int) any { return func(i int) any { return r[i] } }
	if err := list("ground", len(m.Ground), rows(m.Ground), false); err != nil {
		return nil, err
	}
	if len(m.Height) > 0 {
		if err := list("height", len(m.Height), rows(m.Height), false); err != nil {
			return nil, err
		}
	}
	if err := list("objects", len(m.Objects), func(i int) any { return m.Objects[i] }, len(m.Walls) == 0); err != nil {
		return nil, err
	}
	if len(m.Walls) > 0 {
		if err := list("walls", len(m.Walls), func(i int) any { return m.Walls[i] }, true); err != nil {
			return nil, err
		}
	}
	b.WriteString("}\n")
	return b.Bytes(), nil
}

// W and H are the grid's size in small tiles.
func (m *Map) W() int { return len(m.Ground[0]) }
func (m *Map) H() int { return len(m.Ground) }

// Inside reports whether small tile (u, v) is on the grid.
func (m *Map) Inside(u, v int) bool {
	x, y := u-m.Origin[0], v-m.Origin[1]
	return x >= 0 && y >= 0 && x < m.W() && y < m.H()
}

// Glyph is the ground glyph of small tile (u, v); off the grid the nearest edge tile's.
func (m *Map) Glyph(u, v int) byte {
	x, y := clamp(u-m.Origin[0], m.W()), clamp(v-m.Origin[1], m.H())
	return m.Ground[y][x]
}

// Step is the height step of small tile (u, v); 0 off the grid, so the picture's edges show their faces.
func (m *Map) Step(u, v int) int {
	if len(m.Height) == 0 || !m.Inside(u, v) {
		return 0
	}
	return int(m.Height[v-m.Origin[1]][u-m.Origin[0]] - '0')
}

// Bounds is the canvas area round the grid in map pixels (half the canvas), as the old maps stored it.
func (m *Map) Bounds() [4]int {
	vx, vy, vw, vh := m.Origin[0], m.Origin[1], m.W(), m.H()
	return [4]int{(vx-(vy+vh))*8 - 24, (vx+vy)*4 - 48, ((vx+vw)-vy)*8 + 24, (vx+vy+vw+vh)*4 + 24}
}

// Tall reports whether any tile is raised.
func (m *Map) Tall() bool {
	for _, row := range m.Height {
		for _, c := range row {
			if c != '0' {
				return true
			}
		}
	}
	return false
}

func clamp(x, n int) int { return max(0, min(n-1, x)) }

func (o *Object) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 3 && len(raw) != 5 {
		return fmt.Errorf("object %s: want [name, U, V] or [name, U, V, dx, dy]", b)
	}
	targets := []any{&o.Name, &o.U, &o.V, &o.DX, &o.DY}
	for i := range raw {
		if err := json.Unmarshal(raw[i], targets[i]); err != nil {
			return fmt.Errorf("object %s: %w", b, err)
		}
	}
	return nil
}

func (o Object) MarshalJSON() ([]byte, error) {
	if o.DX == 0 && o.DY == 0 {
		return json.Marshal([]any{o.Name, o.U, o.V})
	}
	return json.Marshal([]any{o.Name, o.U, o.V, o.DX, o.DY})
}

func (w *Wall) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 3 {
		return fmt.Errorf("wall %s: want [family, [U, V], [U, V]]", b)
	}
	for i, t := range []any{&w.Family, &w.From, &w.To} {
		if err := json.Unmarshal(raw[i], t); err != nil {
			return fmt.Errorf("wall %s: %w", b, err)
		}
	}
	if w.From[0] != w.To[0] && w.From[1] != w.To[1] {
		return fmt.Errorf("wall %s: runs neither along u nor along v", b)
	}
	return nil
}

func (w Wall) MarshalJSON() ([]byte, error) { return json.Marshal([]any{w.Family, w.From, w.To}) }
