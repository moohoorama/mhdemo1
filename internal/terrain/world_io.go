package terrain

import (
	"encoding/json"
	"fmt"
	"os"
)

func (w *World) Save(path string) error {
	b, err := json.MarshalIndent(struct {
		Version int    `json:"version"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
		Cells   []Cell `json:"cells"`
	}{MapVersion, w.Width, w.Height, w.cells}, "", "  ")
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
		Split    []Cell `json:"cells"`
	}
	if err = json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	if data.Width < 1 || data.Height < 1 || data.Width > 256 || data.Height > 256 {
		return nil, fmt.Errorf("invalid map dimensions")
	}
	w := New(data.Width, data.Height)
	switch data.Version {
	case MapVersion:
		if len(data.Split) != w.Width*w.Height {
			return nil, fmt.Errorf("invalid cell count")
		}
		for i, c := range data.Split {
			if c.Ground > Wasteland || c.Decoration > Trees || (c.Ground == River && c.Decoration != NoDecoration) {
				return nil, fmt.Errorf("invalid cell %d", i)
			}
			w.cells[i] = c
		}
	case 3, 4:
		if len(data.Terrain) != w.Width*w.Height {
			return nil, fmt.Errorf("invalid terrain count")
		}
		for _, k := range data.Terrain {
			if k > Forest || (data.Version == 3 && k > Wasteland) {
				return nil, fmt.Errorf("invalid terrain kind %d", k)
			}
		}
		for i, k := range data.Terrain {
			w.cells[i] = k.Cell()
		}
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
