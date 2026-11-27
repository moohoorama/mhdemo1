package gui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

// The title's shots are shown like an old film: colours drained toward a warm brown, the light flickering a little,
// grain crawling over the picture, now and then a scratch running down it and a fleck of dust.

const (
	filmSaturation = .42 // share of the colour kept
	filmGrainSize  = 128
	filmGrainAlpha = .07
)

// filmRand is a fixed pseudo-random number in [0, 1) for frame n and a salt: the same film every run.
func filmRand(n, salt int) float64 {
	h := uint32(n)*2654435761 ^ uint32(salt)*2246822519
	h ^= h >> 13
	h *= 3266489917
	h ^= h >> 16
	return float64(h%10000) / 10000
}

// drawFilm puts the drawn shot on dst as old film.
func (g *Game) drawFilm(dst, shot *ebiten.Image) {
	n := g.ticks / 3 // the film's own frames: a little slower than the game's
	var cm colorm.ColorM
	cm.ChangeHSV(0, filmSaturation, 1)
	flicker := 1 + .035*(filmRand(n, 1)-.5)
	cm.Scale(1.04*flicker, .97*flicker, .84*flicker, 1) // a sepia warmth
	cm.Translate(.025, .02, .01, 0)                     // the blacks lifted, as on a worn print
	op := &colorm.DrawImageOptions{}
	if filmRand(n, 2) < .06 { // the gate slips a pixel now and then
		op.GeoM.Translate(0, worldPx)
	}
	colorm.DrawImage(dst, shot, cm, op)

	r := g.reel
	if r.grain == nil {
		r.grain = filmGrain()
	}
	b := dst.Bounds()
	gop := &ebiten.DrawImageOptions{}
	gop.ColorScale.ScaleAlpha(filmGrainAlpha)
	ox, oy := filmRand(n, 3)*filmGrainSize, filmRand(n, 4)*filmGrainSize
	for y := -oy; y < float64(b.Dy()); y += filmGrainSize * 2 {
		for x := -ox; x < float64(b.Dx()); x += filmGrainSize * 2 {
			gop.GeoM.Reset()
			gop.GeoM.Scale(2, 2)
			gop.GeoM.Translate(x, y)
			dst.DrawImage(r.grain, gop)
		}
	}
	inWorld(func() {
		w, h := float32(worldW), float32(worldH)
		if filmRand(n/4, 5) < .35 { // a scratch down the print, held for a few frames
			x := float32(filmRand(n/4, 6)) * w
			rect(dst, x, 0, 1, h, color.NRGBA{R: 235, G: 225, B: 200, A: 70})
		}
		if filmRand(n, 7) < .25 { // a fleck of dust for a frame
			x, y := float32(filmRand(n, 8))*w, float32(filmRand(n, 9))*h
			s := 1 + float32(filmRand(n, 10))*2.5
			rect(dst, x, y, s, s*1.4, color.NRGBA{R: 20, G: 15, B: 10, A: 150})
		}
	})
}

// filmGrain is a tile of fine noise, light and dark specks round a middle grey.
func filmGrain() *ebiten.Image {
	const n = filmGrainSize
	pix := make([]byte, n*n*4)
	for i := range n * n {
		v := filmRand(i, 11)
		c := byte(math.Round(255 * v))
		pix[i*4], pix[i*4+1], pix[i*4+2], pix[i*4+3] = c, c, c, 255
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}
