package gui

import (
	"encoding/json"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// Shots is a hand-drawn pixel sheet, assets/graphics/<name>.{png,json} plus <name>_light.png for
// additive layers (built by tools/build_effects.py): "effects/projectiles" (the flying stone, its
// ground shadow and its landing dust) and "effects/fx" (effects and their particles) at the
// characters' pixel scale, "map/outside" (scenery outside the battle area) and "ui/ui" (UI
// pictures, drawn at four art pixels per UI pixel); also "effects/spells/<spell>", one sheet per simulated spell
// effect baked by tools/spellsim.py, each frame cropped tight with an anchor of its own.
type Shots map[string]Shot

// Shot is one sprite: its frames, the additive frames drawn over them (nil if it has none), the
// anchor (sprite pixels) that lands on the drawing position, or one per frame (Anchors) for frames
// cropped tight, per-frame milliseconds for a one-shot animation, and for a spell the seconds into
// it when its blow lands (Hit).
type Shot struct {
	Anchor  [2]float64
	Anchors [][2]float64
	Frames  []*ebiten.Image
	Light   []*ebiten.Image
	MS      []int
	Hit     float64
}

// anchor is frame i's anchor.
func (s Shot) anchor(i int) [2]float64 {
	if i < len(s.Anchors) {
		return s.Anchors[i]
	}
	return s.Anchor
}

func LoadShots(dir, name string) (Shots, error) {
	b, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return nil, err
	}
	var index map[string]struct {
		Anchor  [2]float64
		Anchors [][2]float64
		Frames  [][4]int
		Light   bool
		MS      []int
		Hit     float64
	}
	if err := json.Unmarshal(b, &index); err != nil {
		return nil, err
	}
	load := func(file string) (*ebiten.Image, error) {
		pix, err := loadRGBA(filepath.Join(dir, file))
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(pix), nil
	}
	sheet, err := load(name + ".png")
	if err != nil {
		return nil, err
	}
	var light *ebiten.Image
	p := Shots{}
	for id, s := range index {
		shot := Shot{Anchor: s.Anchor, Anchors: s.Anchors, MS: s.MS, Hit: s.Hit}
		if s.Light && light == nil {
			if light, err = load(name + "_light.png"); err != nil {
				return nil, err
			}
		}
		for _, r := range s.Frames {
			rect := image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])
			shot.Frames = append(shot.Frames, cut(sheet, rect))
			if s.Light {
				shot.Light = append(shot.Light, cut(light, rect))
			}
		}
		p[id] = shot
	}
	return p, nil
}

// frameAt is the frame of a one-shot animation t seconds in, or -1 once it has finished.
func (s Shot) frameAt(t float64) int {
	ms := t * 1000
	for i, d := range s.MS {
		if ms < float64(d) {
			return i
		}
		ms -= float64(d)
	}
	return -1
}

// length is a one-shot animation's duration in seconds.
func (s Shot) length() float64 {
	total := 0
	for _, d := range s.MS {
		total += d
	}
	return float64(total) / 1000
}

// frameOver is the frame of an animation stretched over an effect, k (0..1) of the way in.
func (s Shot) frameOver(k float64) int {
	return max(0, min(len(s.Frames)-1, int(k*float64(len(s.Frames)))))
}

var white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

// screenFX draws frame i of fx sprite name with its anchor (dx, dy) game pixels from screen
// point (x, y), at the camera's zoom, tinted c and at opacity a.
func (g *Game) screenFX(dst *ebiten.Image, name string, i int, x, y, dx, dy float64, c color.NRGBA, a float64) {
	z := g.zoom
	at := func(cx, cy float64) (float64, float64) { return x + cx*z, y + cy*z }
	g.assets.effect(name).drawTint(dst, i, dx, dy, at, z, c, a)
}

// drawUI draws frame i of UI picture name (four art pixels per UI pixel) with its anchor at UI
// point (x, y), scaled by k, tinted c and smoothly filtered so it stays clean at any UI scale.
func (g *Game) drawUI(dst *ebiten.Image, name string, i int, x, y, k float64, c color.NRGBA) {
	s := g.assets.UI[name]
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-s.Anchor[0], -s.Anchor[1])
	op.GeoM.Scale(k/4, k/4)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	drawScaled(dst, s.Frames[i], op)
}

// draw blits frame i with its anchor at canvas point (x, y), snapped to the canvas pixel grid
// so the dots stay crisp at any zoom.
func (s Shot) draw(dst *ebiten.Image, i int, x, y float64, toScreen func(x, y float64) (float64, float64), zoom float64) {
	s.blit(dst, i, x, y, toScreen, zoom, white, 1, false)
}

// drawAlpha is draw at opacity a.
func (s Shot) drawAlpha(dst *ebiten.Image, i int, x, y float64, toScreen func(x, y float64) (float64, float64), zoom, a float64) {
	s.blit(dst, i, x, y, toScreen, zoom, white, a, false)
}

// drawTint is drawAlpha with the colours multiplied by c, for sprites drawn in white.
func (s Shot) drawTint(dst *ebiten.Image, i int, x, y float64, toScreen func(x, y float64) (float64, float64), zoom float64, c color.NRGBA, a float64) {
	s.blit(dst, i, x, y, toScreen, zoom, c, a, false)
}

// blit is drawTint, mirrored left to right about the anchor when mirror is set.
func (s Shot) blit(dst *ebiten.Image, i int, x, y float64, toScreen func(x, y float64) (float64, float64), zoom float64, c color.NRGBA, a float64, mirror bool) {
	an := s.anchor(i)
	ax := an[0]
	w := float64(s.Frames[i].Bounds().Dx())
	if mirror {
		ax = w - ax
	}
	px, py := toScreen(math.Round(x-ax), math.Round(y-an[1]))
	at := func(blend ebiten.Blend) *ebiten.DrawImageOptions {
		op := &ebiten.DrawImageOptions{Blend: blend}
		op.ColorScale.ScaleWithColor(c)
		op.ColorScale.ScaleAlpha(float32(a))
		if mirror {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate(math.Round(px), math.Round(py))
		return op
	}
	drawScaled(dst, s.Frames[i], at(ebiten.BlendSourceOver))
	if s.Light != nil {
		drawScaled(dst, s.Light[i], at(ebiten.BlendLighter))
	}
}
