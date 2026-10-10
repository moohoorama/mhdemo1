package content

type MapArt struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Size        [2]int     `json:"size"`
	Tiles       []string   `json:"tiles"`
	Bounds      [4]int     `json:"bounds"`
	Ground      []Fragment `json:"ground"`
	Decorations [][4]any   `json:"decorations"`
	Props       [][]any    `json:"props"`
	Deformed    bool       `json:"deformed"`
	Native      bool       `json:"native"`
	Visual      [4]int     `json:"visual"`
	SmallTiles  []string   `json:"small_tiles"`
}

// Graphics preserves the existing tile atlas contract on the finer half-cell lattice.
func (d *MapDocument) Graphics(data *Data) MapArt {
	st := d.Compile()
	v := d.Visual
	vx, vy := (v.X-d.Play.X)*2, (v.Y-d.Play.Y)*2
	a := MapArt{Ground: make([]Fragment, 0, v.W*v.H*16), ID: st.ID, Name: st.Name, Size: [2]int{st.Width, st.Height}, Tiles: st.Tiles, Native: true, Deformed: true, Visual: [4]int{vx, vy, v.W * 2, v.H * 2}, Props: [][]any{}, Decorations: [][4]any{}}
	for _, deco := range d.Decorations {
		deco[0] = numeric(deco[0]) - float64((d.Play.X-d.Play.Y)*16)
		deco[1] = numeric(deco[1]) - float64((d.Play.X+d.Play.Y)*8)
		a.Decorations = append(a.Decorations, deco)
	}
	for _, prop := range d.Props {
		p := append([]any{}, prop...)
		if len(p) >= 3 {
			p[1] = numeric(p[1]) - float64(d.Play.X)
			p[2] = numeric(p[2]) - float64(d.Play.Y)
		}
		if len(p) >= 5 {
			p[3] = numeric(p[3]) - float64(d.Play.X)
			p[4] = numeric(p[4]) - float64(d.Play.Y)
		}
		a.Props = append(a.Props, p)
	}
	a.Bounds = [4]int{(vx-(vy+v.H*2))*8 - 24, (vx+vy)*4 - 48, ((vx+v.W*2)-vy)*8 + 24, (vx+vy+v.W*2+v.H*2)*4 + 24}
	for y := 0; y < v.H*2; y++ {
		row := ""
		for x := 0; x < v.W*2; x++ {
			row += d.Small(v.X*2+x, v.Y*2+y)
		}
		a.SmallTiles = append(a.SmallTiles, row)
	}
	vertex := func(sx, sy, kind int) bool {
		for y := FloorHalf(sy - 1); y <= FloorHalf(sy); y++ {
			for x := FloorHalf(sx - 1); x <= FloorHalf(sx); x++ {
				if x >= vx && y >= vy && x < vx+v.W*2 && y < vy+v.H*2 {
					t := a.SmallTiles[y-vy][x-vx]
					if data.Tiles[string(t)].Surface >= kind {
						return true
					}
				}
			}
		}
		return false
	}
	mask := func(x, y, kind int) int {
		n := 0
		for bit, p := range [][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
			if vertex(x+p[0], y+p[1], kind) {
				n |= 1 << bit
			}
		}
		return n
	}
	for y := vy; y < vy+v.H*2; y++ {
		for x := vx; x < vx+v.W*2; x++ {
			for j := 0; j < 2; j++ {
				for i := 0; i < 2; i++ {
					dry, soil := mask(x*2+i, y*2+j, 1), mask(x*2+i, y*2+j, 2)
					a.Ground = append(a.Ground, Fragment{X: (x-y)*8 + (i-j)*4, Y: (x+y)*4 + (i+j)*2, Layers: groundLayers[dry][soil]})
				}
			}
		}
	}
	return a
}

func numeric(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

type Fragment struct {
	X, Y   int
	Layers []int
}

var groundLayers = func() [16][16][]int {
	var all [16][16][]int
	for dry := 0; dry < 16; dry++ {
		for soil := 0; soil < 16; soil++ {
			layers := []int{8}
			if soil != 15 {
				layers = []int{23}
				if dry != 15 {
					layers = []int{0}
					if dry != 0 {
						layers = append(layers, 23+dry)
					}
				}
				if soil != 0 {
					layers = append(layers, 8+soil)
				}
			}
			all[dry][soil] = layers
		}
	}
	return all
}()
