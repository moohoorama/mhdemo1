package content

import (
	"encoding/json"
	"fmt"
	"strings"
)

const MapVersion = 2
const MapMaxSize = 64

type MapRect struct{ X, Y, W, H int }

func (r MapRect) Contains(x, y int) bool { return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H }
func FloorHalf(x int) int {
	if x < 0 {
		return (x - 1) / 2
	}
	return x / 2
}
func Key(x, y int) string { return fmt.Sprintf("%d,%d", x, y) }

type MapDocument struct {
	Decorations  [][4]any
	Props        [][]any
	Version      int
	Stage        Stage
	Play, Visual MapRect
	Cells        map[string]string // "x,y" of a cell: its three layers
	Default      string
	StageFile    string `json:",omitempty"` // file names the document was loaded from; empty for a new one
	MapFile      string `json:",omitempty"`
}

func cloneMap[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}
func ImportMap(st Stage) *MapDocument {
	d := &MapDocument{Version: MapVersion, Stage: cloneMap(st), Play: MapRect{0, 0, st.Width, st.Height}, Visual: MapRect{-4, -4, st.Width + 8, st.Height + 8}, Cells: map[string]string{}, Default: st.Cell(0, 0).String()}
	for y := d.Visual.Y; y < d.Visual.Y+d.Visual.H; y++ {
		for x := d.Visual.X; x < d.Visual.X+d.Visual.W; x++ {
			d.Cells[Key(x, y)] = st.Cell(max(0, min(st.Width-1, x)), max(0, min(st.Height-1, y))).String()
		}
	}
	d.Stage.Tiles = nil
	d.Stage.Width, d.Stage.Height = 0, 0
	return d
}

// Cell is the cell at (x, y); the default where none is stored.
func (d *MapDocument) Cell(x, y int) Cell {
	if c, ok := ParseCell(d.Cells[Key(x, y)]); ok {
		return c
	}
	c, _ := ParseCell(d.Default)
	return c
}

// The layers a brush paints.
const (
	LayerGround    = 'g'
	LayerStructure = 's'
	LayerState     = 'e'
)

// Paint puts a glyph on one layer of a cell and reports whether the cell changed. A structure only
// goes on a ground it stands on; a new ground sweeps away a structure that cannot stand on it.
func (d *MapDocument) Paint(data *Data, x, y int, layer, glyph byte) bool {
	c := d.Cell(x, y)
	n := c
	switch layer {
	case LayerGround:
		n.Ground = glyph
		if st, ok := data.Structures[string(n.Structure)]; ok && !strings.Contains(st.Grounds, string(glyph)) {
			n.Structure = NoLayer
		}
	case LayerStructure:
		n.Structure = glyph
	case LayerState:
		n.State = glyph
	}
	if n == c || data.ValidCell(n) != "" {
		return false
	}
	d.Cells[Key(x, y)] = n.String()
	return true
}
func (d *MapDocument) Resize(r MapRect) error {
	if r.W < 4 || r.H < 4 || r.W > MapMaxSize || r.H > MapMaxSize {
		return fmt.Errorf("맵 크기는 4~%d칸이어야 합니다", MapMaxSize)
	}
	d.Play = r
	d.Visual = MapRect{min(d.Visual.X, r.X-4), min(d.Visual.Y, r.Y-4), max(d.Visual.X+d.Visual.W, r.X+r.W+4) - min(d.Visual.X, r.X-4), max(d.Visual.Y+d.Visual.H, r.Y+r.H+4) - min(d.Visual.Y, r.Y-4)}
	if d.Visual.W > MapMaxSize+8 || d.Visual.H > MapMaxSize+8 {
		d.Visual = MapRect{r.X - 4, r.Y - 4, r.W + 8, r.H + 8}
	}
	for y := r.Y - 4; y < r.Y+r.H+4; y++ {
		for x := r.X - 4; x < r.X+r.W+4; x++ {
			if d.Cells[Key(x, y)] == "" {
				d.Cells[Key(x, y)] = d.Default
			}
		}
	}
	allies := []Point{}
	for _, p := range d.Stage.Allies {
		if r.Contains(p.X, p.Y) {
			allies = append(allies, p)
		}
	}
	d.Stage.Allies = allies
	filter := func(src []Spawn) []Spawn {
		out := []Spawn{}
		for _, s := range src {
			if r.Contains(s.X, s.Y) {
				out = append(out, s)
			}
		}
		return out
	}
	d.Stage.Enemies = filter(d.Stage.Enemies)
	d.Stage.Party = filter(d.Stage.Party)
	return nil
}
func (d *MapDocument) Compile() Stage {
	st := cloneMap(d.Stage)
	st.Width, st.Height = d.Play.W, d.Play.H
	st.Tiles = nil
	for y := d.Play.Y; y < d.Play.Y+d.Play.H; y++ {
		row := ""
		for x := d.Play.X; x < d.Play.X+d.Play.W; x++ {
			row += d.Cell(x, y).String()
		}
		st.Tiles = append(st.Tiles, row)
	}
	for i := range st.Allies {
		st.Allies[i].X -= d.Play.X
		st.Allies[i].Y -= d.Play.Y
	}
	shift := func(p *Point) {
		if p != nil {
			p.X -= d.Play.X
			p.Y -= d.Play.Y
		}
	}
	for _, list := range [][]Spawn{st.Enemies, st.Party} {
		for i := range list {
			list[i].X -= d.Play.X
			list[i].Y -= d.Play.Y
			shift(list[i].Goal)
		}
	}
	for i := range st.Events {
		for j := range st.Events[i].Do {
			for k := range st.Events[i].Do[j].Spawns {
				st.Events[i].Do[j].Spawns[k].X -= d.Play.X
				st.Events[i].Do[j].Spawns[k].Y -= d.Play.Y
				shift(st.Events[i].Do[j].Spawns[k].Goal)
			}
			shift(st.Events[i].Do[j].Goal)
			for k := range st.Events[i].Do[j].Moves {
				st.Events[i].Do[j].Moves[k].To.X -= d.Play.X
				st.Events[i].Do[j].Moves[k].To.Y -= d.Play.Y
			}
			for _, group := range st.Events[i].Do[j].Steps {
				for k := range group {
					shift(group[k].To)
				}
			}
		}
	}
	for i := range st.Areas {
		st.Areas[i].Rect.X -= d.Play.X
		st.Areas[i].Rect.Y -= d.Play.Y
	}
	return st
}
func (d *MapDocument) Validate(data *Data) error {
	if d.Visual.W < d.Play.W || d.Visual.H < d.Play.H || d.Visual.W > MapMaxSize+8 || d.Visual.H > MapMaxSize+8 || !d.Visual.Contains(d.Play.X, d.Play.Y) || !d.Visual.Contains(d.Play.X+d.Play.W-1, d.Play.Y+d.Play.H-1) {
		return fmt.Errorf("그림 영역 범위 오류")
	}
	if d.Version != MapVersion || d.Play.W < 4 || d.Play.H < 4 || d.Play.W > MapMaxSize || d.Play.H > MapMaxSize {
		return fmt.Errorf("맵 크기 또는 버전 오류")
	}
	for y := d.Visual.Y; y < d.Visual.Y+d.Visual.H; y++ {
		for x := d.Visual.X; x < d.Visual.X+d.Visual.W; x++ {
			c, ok := ParseCell(d.Cells[Key(x, y)])
			if !ok {
				return fmt.Errorf("타일 누락 (%d,%d)", x, y)
			}
			if why := data.ValidCell(c); why != "" {
				return fmt.Errorf("지형 오류 (%d,%d): %s", x, y, why)
			}
		}
	}
	copy := cloneMap(*data)
	copy.Explores = data.Explores // not in the bundle's JSON
	st := d.Compile()
	if len(st.Allies) < st.Max {
		return fmt.Errorf("아군 배치 칸 %d개: 최소 %d개가 필요합니다", len(st.Allies), st.Max)
	}
	if len(st.Enemies) == 0 {
		return fmt.Errorf("적 병력을 배치하세요")
	}
	enemy := map[string]bool{}
	for _, sp := range st.Enemies {
		enemy[sp.Character] = true
	}
	for _, event := range st.Events {
		for _, a := range event.Do {
			for _, sp := range a.Spawns {
				if a.Kind == "spawn" && sp.Side == "enemy" {
					enemy[sp.Character] = true
				}
			}
		}
	}
	for _, event := range st.Duels {
		if !enemy[event.Enemy] {
			return fmt.Errorf("일기토 %s의 적 장수가 없습니다: 일기토를 수정하거나 삭제하세요", event.ID)
		}
	}
	for _, p := range st.Allies {
		if p.X < 0 || p.Y < 0 || p.X >= st.Width || p.Y >= st.Height {
			return fmt.Errorf("아군 배치 위치가 영역 밖입니다 (%d,%d)", p.X, p.Y)
		}
		usable := false
		for _, id := range data.Campaign.Roster {
			class := data.Classes[data.Characters[id].Class]
			usable = usable || data.Move(class.Terrain, st.Cell(p.X, p.Y), st.Boats).Cost > 0
		}
		if !usable {
			return fmt.Errorf("아군이 배치할 수 없는 지형 (%d,%d)", p.X, p.Y)
		}
	}
	i := copy.StageIndex(st.ID)
	if i < 0 {
		copy.Stages = append(copy.Stages, st)
	} else {
		copy.Stages[i] = st
	}
	return copy.Validate()
}

func (d *MapDocument) ValidateShape() error {
	if d.Version != MapVersion || d.Cells == nil || d.Play.W < 4 || d.Play.H < 4 || d.Play.W > MapMaxSize || d.Play.H > MapMaxSize || d.Visual.W < d.Play.W || d.Visual.H < d.Play.H || d.Visual.W > MapMaxSize+8 || d.Visual.H > MapMaxSize+8 || !d.Visual.Contains(d.Play.X, d.Play.Y) || !d.Visual.Contains(d.Play.X+d.Play.W-1, d.Play.Y+d.Play.H-1) {
		return fmt.Errorf("맵 문서 버전/자료 오류")
	}
	return nil
}

// PaintSquare paints a square of size cells around (x, y), inside the picture area.
func (d *MapDocument) PaintSquare(data *Data, x, y, size int, layer, glyph byte) {
	x -= (size - 1) / 2
	y -= (size - 1) / 2
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			if d.Visual.Contains(x+i, y+j) {
				d.Paint(data, x+i, y+j, layer, glyph)
			}
		}
	}
}
