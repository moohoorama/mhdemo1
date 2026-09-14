package main

import (
	"demo1/internal/terrain"
	"image"
	"image/draw"
	"math"
	"testing"
)

func TestSharedEdgesAllCompatibleMasks(t *testing.T) {
	for a := 0; a < 16; a++ {
		for b := 0; b < 16; b++ {
			for i := 0; i <= 100; i++ {
				s := float64(i) / 100
				if (a>>1&1) == (b&1) && (a>>2&1) == (b>>3&1) && math.Abs(sample(a, 1, s)-sample(b, 0, s)) > 1e-12 {
					t.Fatalf("u seam %d/%d", a, b)
				}
				if (a>>3&1) == (b&1) && (a>>2&1) == (b>>1&1) && math.Abs(sample(a, s, 1)-sample(b, s, 0)) > 1e-12 {
					t.Fatalf("v seam %d/%d", a, b)
				}
			}
		}
	}
}

func TestPixelCoverageNoGapsOrOverlap(t *testing.T) {
	counts := map[image.Point]int{}
	for y := -4; y <= 4; y++ {
		for x := -4; x <= 4; x++ {
			im := asset(0)
			px, py := terrain.Project(float64(x)/2, float64(y)/2)
			pixels := 0
			for iy := 0; iy < 8; iy++ {
				for ix := 0; ix < 16; ix++ {
					if im.NRGBAAt(ix, iy).A != 0 {
						counts[image.Pt(int(px)-8+ix, int(py)+iy)]++
						pixels++
					}
				}
			}
			if pixels != 64 {
				t.Fatalf("diamond area %d, want 64", pixels)
			}
		}
	}
	for y := -20; y <= 20; y++ {
		for x := -40; x <= 40; x++ {
			u, v := terrain.Unproject(float64(x)+.5, float64(y)+.5)
			if u >= -2 && u < 2.5 && v >= -2 && v < 2.5 && counts[image.Pt(x, y)] != 1 {
				t.Fatalf("pixel %d,%d covered %d times", x, y, counts[image.Pt(x, y)])
			}
		}
	}
}

func TestDiagonalLandConnected(t *testing.T) {
	for _, mask := range []int{5, 10} {
		if sample(mask, .5, .5) <= .5 {
			t.Fatalf("disconnected saddle %d", mask)
		}
	}
}

func TestWaterLoopsSpatiallyAndChangesSmoothly(t *testing.T) {
	for frame := 0; frame < 8; frame++ {
		changed := false
		for y := 0; y < 8; y++ {
			for x := 0; x < 16; x++ {
				c := waterColor(float64(x)+.5, float64(y)+.5, frame)
				for _, d := range [][2]float64{{8, 4}, {-8, 4}} {
					if c != waterColor(float64(x)+.5+d[0], float64(y)+.5+d[1], frame) {
						t.Fatal("water spatial seam")
					}
				}
				next := waterColor(float64(x)+.5, float64(y)+.5, (frame+1)%8)
				if c != next {
					changed = true
				}
				if math.Abs(float64(c.G)-float64(next.G)) > 8 {
					t.Fatal("water animation discontinuity")
				}
			}
		}
		if !changed {
			t.Fatal("static water frame")
		}
	}
}

func TestLandOverlaysContainNoWaterAndDoNotAnimate(t *testing.T) {
	var tiles [assetCount]image.Image
	for i := range tiles {
		tiles[i] = asset(i)
	}
	for mask := 1; mask < 15; mask++ {
		overlay := asset(mask + 8)
		for y := 0; y < 8; y++ {
			for x := 0; x < 16; x++ {
				c := overlay.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				if c.R < c.B {
					t.Fatal("water color in soil overlay")
				}
				for frame := 0; frame < 8; frame++ {
					composed := image.NewNRGBA(image.Rect(0, 0, 16, 8))
					for _, id := range terrain.Layers(mask, frame) {
						draw.Draw(composed, composed.Bounds(), tiles[id], image.Point{}, draw.Over)
					}
					if composed.NRGBAAt(x, y) != c {
						t.Fatal("soil overlay changed with water frame")
					}
				}
			}
		}
	}
}
