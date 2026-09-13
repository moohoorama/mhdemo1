package terrain

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
)

const TileWidth, TileHeight = 16, 8
const AssetCount = 38
const GrassAsset = 23

// Sample is bilinear in map coordinates. Adjacent tiles have exactly the same
// edge function because an edge depends only on its two shared vertex values.
func Sample(mask int, u, v float64) float64 {
	b := func(bit int) float64 {
		if mask&bit != 0 {
			return 1
		}
		return 0
	}
	s := b(1)*(1-u)*(1-v) + b(2)*u*(1-v) + b(4)*u*v + b(8)*(1-u)*v
	// Resolve diagonal saddles with connected land at the center. This term is zero
	// on every edge, so the shared-edge contract is preserved.
	if mask == 5 || mask == 10 {
		s += .18 * 16 * u * (1 - u) * v * (1 - v)
	}
	return s
}

// Asset order: eight water frames, soil + fourteen edges, grass + fourteen edges.
func AssetName(i int) string {
	if i < 8 {
		return fmt.Sprintf("water_%d.png", i)
	}
	if i == 8 {
		return "land.png"
	}
	if i == GrassAsset {
		return "grass.png"
	}
	if i > GrassAsset {
		return fmt.Sprintf("grass_edge_%02d.png", i-GrassAsset)
	}
	return fmt.Sprintf("edge_%02d.png", i-8)
}
func Asset(i int) *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 16; x++ {
			u, v := Unproject(float64(x)+.5-8, float64(y)+.5)
			u *= 2
			v *= 2
			if u < 0 || v < 0 || u >= 1 || v >= 1 {
				continue
			}
			var c color.NRGBA
			if i < 8 {
				c = WaterColor(float64(x)+.5, float64(y)+.5, i)

			} else {
				mask := 15
				if i > 8 && i < GrassAsset {
					mask = i - 8
				}
				if i > GrassAsset {
					mask = i - GrassAsset
				}
				s := Sample(mask, u, v)
				if s < .5 {
					continue
				}
				c = color.NRGBA{187, 158, 106, 255}
				if s < .64 {
					c = color.NRGBA{104, 91, 65, 255}
				} else if s < .78 {
					c = color.NRGBA{166, 137, 88, 255}
				} else if ((x%8)*13+(y%4)*23)%11 == 0 {
					c = color.NRGBA{177, 145, 95, 255}
				}
				if i >= GrassAsset {
					c = color.NRGBA{112, 143, 77, 255}
					if s < .63 {
						c = color.NRGBA{74, 103, 65, 255}
					} else if s < .77 {
						c = color.NRGBA{97, 124, 69, 255}
					} else {
						h := ((x%8)*19 + (y%4)*31) % 17
						if h < 3 {
							c = color.NRGBA{123, 151, 82, 255}
						} else if h == 7 {
							c = color.NRGBA{101, 132, 68, 255}
						}
					}
				}
			}
			im.SetNRGBA(x, y, c)
		}
	}
	return im
}
func Layers(mask, frame int) []int {
	if mask == 15 {
		return []int{8}
	}
	if mask == 0 {
		return []int{frame % 8}
	}
	return []int{frame % 8, 8 + mask}
}

// WaterColor combines gentle cross-ripples with short, horizontal reflections.
// Every spatial term repeats under (8,4) and (-8,4); time loops in eight frames.
func WaterColor(x, y float64, frame int) color.NRGBA {
	t := 2 * math.Pi * float64(frame) / 8
	a := 2 * math.Pi * x / 8
	b := 2 * math.Pi * y / 4
	swell := math.Sin(b+.35*math.Sin(a)+t)*.65 + math.Sin(a-b-t)*.35
	ripple := math.Pow(math.Max(0, math.Cos(b+.32*math.Sin(t+a))), 8)
	broken := .5 + .5*math.Sin(a+.5*math.Sin(t))
	glint := ripple * broken
	return color.NRGBA{uint8(44 + 2*swell + 5*glint), uint8(108 + 3*swell + 8*glint), uint8(119 + 3*swell + 8*glint), 255}
}

// The grass mask includes all dry terrain; wasteland is the top overlay.
func SurfaceLayers(ground, land, frame int) []int {
	if land == 15 {
		return []int{8}
	}
	layers := []int{}
	if ground == 15 {
		layers = append(layers, GrassAsset)
	} else {
		layers = append(layers, frame%8)
		if ground != 0 {
			layers = append(layers, GrassAsset+ground)
		}
	}
	if land != 0 {
		layers = append(layers, 8+land)
	}
	return layers
}
func Render(w *World, tiles [AssetCount]image.Image, scale, frame int) *image.NRGBA {
	raw := image.NewNRGBA(image.Rect(0, 0, (w.Width+w.Height)*16, (w.Width+w.Height)*8))
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			ground := w.MasksFor(x, y, Grass)
			for k, m := range w.Masks(x, y) {
				px, py := Project(float64(x)+float64(k%2)/2, float64(y)+float64(k/2)/2)
				p := image.Pt(int(px)+w.Height*16-8, int(py))
				for _, i := range SurfaceLayers(ground[k], m, frame) {
					draw.Draw(raw, image.Rectangle{Min: p, Max: p.Add(image.Pt(16, 8))}, tiles[i], image.Point{}, draw.Over)
				}
			}
		}
	}
	return Enlarge(raw, scale)
}
func Enlarge(src image.Image, scale int) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, src.At(b.Min.X+x/scale, b.Min.Y+y/scale))
		}
	}
	return out
}
