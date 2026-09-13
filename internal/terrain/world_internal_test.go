package terrain

import "testing"

func TestThreeTerrainNeighborhoods(t *testing.T) {
	for pattern := 0; pattern < 19683; pattern++ {
		w := New(3, 3)
		n := pattern
		for i := range w.cells {
			w.SetTerrain(i%w.Width, i/w.Width, Kind(n%3))
			n /= 3
		}
		for sy := 0; sy <= 6; sy++ {
			for sx := 0; sx <= 6; sx++ {
				want := River
				for y := 0; y < 3; y++ {
					for x := 0; x < 3; x++ {
						if 2*x <= sx && sx <= 2*x+2 && 2*y <= sy && sy <= 2*y+2 {
							want = max(want, w.Terrain(x, y))
						}
					}
				}
				for _, kind := range []Kind{Grass, Wasteland} {
					if w.vertexAtLeast(sx, sy, kind) != (want >= kind) {
						t.Fatalf("pattern %d vertex %d,%d", pattern, sx, sy)
					}
				}
			}
		}
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				dry, soil := w.MasksFor(x, y, Grass), w.Masks(x, y)
				for k := 0; k < 4; k++ {
					if soil[k]&dry[k] != soil[k] {
						t.Fatal("soil outside dry mask")
					}
				}
			}
		}
	}
}
