package content

import (
	"encoding/json"
	"fmt"
)

const MapVersion = 1
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
	Decorations     [][4]any
	Props           [][]any
	Version         int
	Stage           Stage
	Play, Visual    MapRect
	Cells, Resolved map[string]string
	Default         string
}

func cloneMap[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}
func ImportMap(st Stage) *MapDocument {
	d := &MapDocument{Version: MapVersion, Stage: cloneMap(st), Play: MapRect{0, 0, st.Width, st.Height}, Visual: MapRect{-4, -4, st.Width + 8, st.Height + 8}, Cells: map[string]string{}, Resolved: map[string]string{}, Default: string(st.Tiles[0][0])}
	for y := d.Visual.Y; y < d.Visual.Y+d.Visual.H; y++ {
		for x := d.Visual.X; x < d.Visual.X+d.Visual.W; x++ {
			t := string(st.Tiles[max(0, min(st.Height-1, y))][max(0, min(st.Width-1, x))])
			d.Resolved[Key(x, y)] = t
			for j := 0; j < 2; j++ {
				for i := 0; i < 2; i++ {
					d.Cells[Key(x*2+i, y*2+j)] = t
				}
			}
		}
	}
	d.Stage.Tiles = nil
	d.Stage.Width, d.Stage.Height = 0, 0
	return d
}
func (d *MapDocument) Small(x, y int) string {
	if t := d.Cells[Key(x, y)]; t != "" {
		return t
	}
	return d.Default
}
func (d *MapDocument) Tile(x, y int) string {
	if t := d.Resolved[Key(x, y)]; t != "" {
		return t
	}
	return d.Default
}
func (d *MapDocument) Paint(x, y int, t string) {
	d.Cells[Key(x, y)] = t
	u, v := FloorHalf(x), FloorHalf(y)
	counts := map[string]int{}
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			counts[d.Small(2*u+i, 2*v+j)]++
		}
	}
	winner, n, tied := "", 0, false
	for kind, count := range counts {
		if count > n {
			winner, n, tied = kind, count, false
		} else if count == n {
			tied = true
		}
	}
	if !tied {
		d.Resolved[Key(u, v)] = winner
	}
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
			if d.Resolved[Key(x, y)] == "" {
				d.Resolved[Key(x, y)] = d.Default
			}
			for j := 0; j < 2; j++ {
				for i := 0; i < 2; i++ {
					k := Key(2*x+i, 2*y+j)
					if d.Cells[k] == "" {
						d.Cells[k] = d.Default
					}
				}
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
			row += d.Tile(x, y)
		}
		st.Tiles = append(st.Tiles, row)
	}
	for i := range st.Allies {
		st.Allies[i].X -= d.Play.X
		st.Allies[i].Y -= d.Play.Y
	}
	for _, list := range [][]Spawn{st.Enemies, st.Party} {
		for i := range list {
			list[i].X -= d.Play.X
			list[i].Y -= d.Play.Y
		}
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
	for y := d.Visual.Y * 2; y < (d.Visual.Y+d.Visual.H)*2; y++ {
		for x := d.Visual.X * 2; x < (d.Visual.X+d.Visual.W)*2; x++ {
			if d.Cells[Key(x, y)] == "" {
				return fmt.Errorf("소타일 누락 (%d,%d)", x, y)
			}
			if _, ok := data.Tiles[d.Small(x, y)]; !ok {
				return fmt.Errorf("알 수 없는 지형 (%d,%d)", x, y)
			}
		}
	}
	for y := d.Play.Y; y < d.Play.Y+d.Play.H; y++ {
		for x := d.Play.X; x < d.Play.X+d.Play.W; x++ {
			counts := map[string]int{}
			for j := 0; j < 2; j++ {
				for i := 0; i < 2; i++ {
					counts[d.Small(2*x+i, 2*y+j)]++
				}
			}
			best, n, tie := "", 0, false
			for t, c := range counts {
				if c > n {
					best, n, tie = t, c, false
				} else if c == n {
					tie = true
				}
			}
			if _, ok := data.Tiles[d.Tile(x, y)]; !ok || !tie && best != d.Tile(x, y) {
				return fmt.Errorf("지형 판정 불일치 (%d,%d)", x, y)
			}
		}
	}
	copy := cloneMap(*data)
	st := d.Compile()
	if len(st.Allies) < st.Limit {
		return fmt.Errorf("아군 배치 칸 %d개: 최소 %d개가 필요합니다", len(st.Allies), st.Limit)
	}
	if len(st.Enemies) == 0 {
		return fmt.Errorf("적 병력을 배치하세요")
	}
	enemy := map[string]bool{}
	for _, sp := range st.Enemies {
		enemy[sp.Character] = true
	}
	if st.Goal != "rout" && !enemy[st.Boss] {
		return fmt.Errorf("보스가 없습니다: 승리 조건에서 보스를 지정하세요")
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
			usable = usable || data.Terrain[class.Terrain][string(st.Tiles[p.Y][p.X])].Cost > 0
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
	if d.Version != MapVersion || d.Cells == nil || d.Resolved == nil || d.Play.W < 4 || d.Play.H < 4 || d.Play.W > MapMaxSize || d.Play.H > MapMaxSize || d.Visual.W < d.Play.W || d.Visual.H < d.Play.H || d.Visual.W > MapMaxSize+8 || d.Visual.H > MapMaxSize+8 || !d.Visual.Contains(d.Play.X, d.Play.Y) || !d.Visual.Contains(d.Play.X+d.Play.W-1, d.Play.Y+d.Play.H-1) {
		return fmt.Errorf("맵 문서 버전/자료 오류")
	}
	return nil
}

func (d *MapDocument) PaintSquare(x, y, size int, tile string) {
	x -= (size - 1) / 2
	y -= (size - 1) / 2
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			if d.Visual.Contains(FloorHalf(x+i), FloorHalf(y+j)) {
				d.Paint(x+i, y+j, tile)
			}
		}
	}
}
