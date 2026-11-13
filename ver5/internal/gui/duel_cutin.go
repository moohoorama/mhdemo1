package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/duelview"
)

// The duel cut-in (docs/ui-mockup, idea 7): once the duel screen is up, black bars close
// in from above and below, leaving a band across the middle. The two officers' portraits,
// each cut on the same diagonal (the ally's right side, the enemy's left), charge in from
// either side and meet on it with a flash and a jolt; then each name is stamped in brush
// script, and the bars open again on the duel. The duel waits while it plays.
const (
	cutBars     = .30 // bars close
	cutBand     = .20 // band opens from its centre line
	cutCharge   = .34 // portraits charge in
	cutClash    = .32 // pause after the impact
	cutStamp    = .26 // each name stamp
	cutBetween  = .26 // gap between the names
	cutHold     = 1.30
	cutFlash    = .42
	cutOpen     = .38
	cutBarFrac  = .31 // each bar's height as a fraction of the screen
	cutPaneL    = .56 // the ally pane's width; the cut runs to 80% of it at the bottom
	cutPaneR    = .552
	cutNameSize = 200.0 // at the design height; smaller screens shrink it with the band
)

type duelCutIn struct {
	view *duelview.View
	t    float64
	done bool
}

func cutTimes() (charge, clash, stampL, stampR, hold, flash, open, end float64) {
	charge = cutBars + cutBand
	clash = charge + cutCharge
	stampL = clash + cutClash
	stampR = stampL + cutStamp + cutBetween
	hold = stampR + cutStamp
	flash = hold + cutHold
	open = flash + cutFlash
	end = open + cutOpen
	return
}

// cutInLength is how long the cut-in holds a staged duel.
func cutInLength(v *duelview.View) float64 {
	if v == nil || !v.Staged {
		return 0
	}
	_, _, _, _, _, _, _, end := cutTimes()
	return end
}

// cutInStep runs the cut-in once a duel's screen is fully up and returns the part of dt
// left for the duel's own clock: the duel waits while the cut-in plays.
func (p *Player) cutInStep(dt float64) float64 {
	v := p.Duel
	c := &p.g.cutIn
	if v == nil || !v.Staged {
		return dt
	}
	if c.view != v {
		*c = duelCutIn{view: v}
	}
	if c.done || v.Clock < duelview.Transition.Intro() {
		return dt
	}
	use := min(dt, cutInLength(v)-c.t)
	p.g.cutInSounds(c.t, c.t+use)
	c.t += use
	if c.t >= cutInLength(v) {
		c.done = true
	}
	return dt - use
}

// cutInSounds strikes the cut-in's cues that fall in (from, to]: the gong and the ring of iron as the
// portraits meet, then a thud for each name stamped.
func (g *Game) cutInSounds(from, to float64) {
	_, clash, stampL, stampR, _, _, _, _ := cutTimes()
	cues := []struct {
		at  float64
		cue string
		vol float64
	}{{clash, "duel-start", 1}, {clash, "block", .8}, {stampL, "hit", .7}, {stampR, "hit", .7}}
	for _, c := range cues {
		if from < c.at && c.at <= to {
			g.sfxVol(c.cue, c.vol)
		}
	}
}

// skipCutIn ends the cut-in at once (the duel's skip button and keys); a skip before the portraits
// meet still sounds the gong.
func (g *Game) skipCutIn() {
	c := &g.cutIn
	if c.view == nil || c.done {
		return
	}
	if _, clash, _, _, _, _, _, _ := cutTimes(); c.t < clash {
		g.sfx("duel-start")
	}
	c.done = true
}

func (g *Game) drawCutIn(dst *ebiten.Image) {
	c := &g.cutIn
	v := g.player.Duel
	if v == nil || c.view != v || c.done || c.t <= 0 {
		return
	}
	t := c.t
	charge, clash, stampL, stampR, _, flash, open, _ := cutTimes()
	W, H := float64(uiW), float64(uiH)
	fullW := W + 1 // uiW rounds the device width down: the fills reach its last pixels
	bar := cutBarFrac * H
	top, bandH := bar, H-2*bar

	// bars: close, then open after the flash
	k := bezier(.6, 0, .2, 1, phase(t, 0, cutBars))
	if t >= open {
		k = 1 - bezier(.6, 0, .2, 1, phase(t, open, cutOpen))
	}
	// the band, while it is up
	if t >= cutBars && t < flash+cutFlash*.3 {
		bk := bezier(.2, .8, .2, 1, phase(t, cutBars, cutBand))
		mid := top + bandH/2
		y0, y1 := mid-bandH/2*bk, mid+bandH/2*bk
		for i := 0; i < 16; i++ {
			s := float64(i) / 15
			d := math.Abs(s-.5) * 2
			c := color.NRGBA{R: uint8(26 + 32*(1-d)*.5), G: uint8(8 + 10*(1-d)*.5), B: uint8(5 + 5*(1-d)*.5), A: 255}
			rect(dst, 0, float32(top+bandH*s), float32(fullW), float32(bandH/15+1), c)
		}
		band := dst.SubImage(image.Rect(0, int(y0*drawPx), int(math.Ceil(fullW*drawPx)), int(y1*drawPx))).(*ebiten.Image)
		g.drawCutPanes(band, v, t, top, bandH, charge, clash)
		g.drawCutNames(band, v, t, top, bandH, stampL, stampR)
	}
	rect(dst, 0, float32(-bar*(1-k)), float32(fullW), float32(bar), color.NRGBA{R: 6, G: 4, B: 3, A: 255})
	rect(dst, 0, float32(bar*k-2), float32(fullW), 2, color.NRGBA{R: gold.R, G: gold.G, B: gold.B, A: 204})
	rect(dst, 0, float32(H-bar*k), float32(fullW), float32(bar+1), color.NRGBA{R: 6, G: 4, B: 3, A: 255})
	rect(dst, 0, float32(H-bar*k), float32(fullW), 2, color.NRGBA{R: gold.R, G: gold.G, B: gold.B, A: 204})

	// the white flashes: the impact, and the cut back to the duel
	a := 0.0
	if t >= clash && t < clash+.18 {
		a = .75 * (1 - (t-clash)/.18)
	}
	if t >= flash && t < open {
		f := (t - flash) / cutFlash
		a = .9 * keyframe(f, []float64{0, .3, 1}, []float64{0, 1, 0})
	}
	if a > 0 {
		rect(dst, 0, 0, float32(fullW), float32(H+1), color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * a)})
	}
}

// cutShake is the screen jolt at the impact and at each stamp, in layout pixels.
func cutShake(t, at, dur, amp float64) float64 {
	if t < at || t >= at+dur {
		return 0
	}
	k := (t - at) / dur
	return keyframe(k, []float64{0, .33, .66, 1}, []float64{0, -amp, amp * .8, 0})
}

func (g *Game) drawCutPanes(dst *ebiten.Image, v *duelview.View, t, top, bandH, charge, clash float64) {
	if t < charge {
		return
	}
	W := float64(uiW) + 1
	ck := bezier(.55, 0, .9, .6, phase(t, charge, cutCharge))
	recoil := keyframe(phase(t, clash, .26), []float64{0, .5, 1}, []float64{0, .025, 0})
	drift := phase(t, clash+cutClash+cutStamp*2+cutBetween, 1.4) * .03
	shake := cutShake(t, clash, .26, 9)
	for side := 0; side < 2; side++ {
		frac := cutPaneL
		if side == 1 {
			frac = cutPaneR
		}
		pw := W * frac
		img := g.cutPane(side, v.Fighters[side].Portrait, pw, bandH, drift)
		x := 0.0
		if side == 1 {
			x = W - pw
		}
		off := -(1 - ck) * 1.05 * pw
		if side == 1 {
			off = -off
		}
		r := -recoil * pw
		if side == 1 {
			r = -r
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate((x+off+r+shake)*drawPx, (top+shake*.6)*drawPx)
		dst.DrawImage(img, op)
	}
	// the shared cut, lit at the impact
	if t >= clash {
		sk := 1 - .65*phase(t, clash, .5)
		x0, x1 := W*cutPaneL+shake, W*cutPaneL*.8+shake
		for _, l := range []struct {
			w float32
			c color.NRGBA
		}{{2.4 * 1.9, color.NRGBA{R: 255, G: 207, B: 122, A: uint8(140 * sk)}}, {.7 * 1.9, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * sk)}}} {
			strokeLine(dst, float32(x0), float32(top), float32(x1), float32(top+bandH), l.w, l.c)
		}
	}
}

// cutPane renders one officer's pane: the portrait filling it, shaded toward the cut and
// the bottom, with the cut-away triangle removed.
func (g *Game) cutPane(side int, portrait *ebiten.Image, pw, ph, drift float64) *ebiten.Image {
	w, h := int(math.Ceil(pw*drawPx)), int(math.Ceil(ph*drawPx))
	if g.cutPanes[side] == nil || g.cutPanes[side].Bounds().Dx() != w || g.cutPanes[side].Bounds().Dy() != h {
		if g.cutPanes[side] != nil {
			g.cutPanes[side].Deallocate()
		}
		g.cutPanes[side] = newLayer(w, h)
	}
	im := g.cutPanes[side]
	im.Clear()
	if portrait != nil {
		iw := pw * 1.07 // just wider than the pane so the drift never bares an edge
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(iw*drawPx/float64(portrait.Bounds().Dx()), iw*drawPx/float64(portrait.Bounds().Dy()))
		d := drift * pw
		if side == 1 {
			d = -d
		}
		x := -.035 * pw
		if side == 1 {
			x = pw - iw + .035*pw
		}
		op.GeoM.Translate((x+d)*drawPx, (ph/2-.38*iw)*drawPx)
		im.DrawImage(portrait, op)
	} else {
		im.Fill(color.NRGBA{R: 40, G: 26, B: 20, A: 255})
	}
	s := float32(drawPx)
	// darken toward the cut and toward the bottom with two stretched ramps
	ramp := cutRamp()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(pw*drawPx/64, ph*drawPx)
	if side == 1 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(pw*drawPx, 0)
	}
	im.DrawImage(ramp[0], op)
	op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(pw*drawPx, ph*drawPx/64)
	im.DrawImage(ramp[1], op)
	var cut vector.Path
	if side == 0 {
		cut.MoveTo(float32(pw)*s, 0)
		cut.LineTo(float32(pw+2)*s, 0)
		cut.LineTo(float32(pw+2)*s, float32(ph)*s)
		cut.LineTo(float32(pw*.8)*s, float32(ph)*s)
	} else {
		cut.MoveTo(-2*s, 0)
		cut.LineTo(float32(pw*.2029)*s, 0)
		cut.LineTo(0, float32(ph)*s)
		cut.LineTo(-2*s, float32(ph)*s)
	}
	cut.Close()
	vector.FillPath(im, &cut, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScaleWhite(), Blend: ebiten.BlendDestinationOut})
	return im
}

// drawCutNames stamps the two names in brush script either side of the cut.
func (g *Game) drawCutNames(dst *ebiten.Image, v *duelview.View, t, top, bandH, stampL, stampR float64) {
	W := float64(uiW)
	size := min(cutNameSize, bandH*.62)
	f := g.brush(size)
	shake := cutShake(t, stampL+cutStamp, .18, 6) + cutShake(t, stampR+cutStamp, .18, 6)
	for side, at := range []float64{stampL, stampR} {
		if t < at {
			continue
		}
		k := bezier(.42, 0, 1, 1, phase(t, at, cutStamp))
		sc := keyframe(k, []float64{0, .7, 1}, []float64{3, .92, 1})
		name := v.Fighters[side].Name
		glow := color.NRGBA{R: 80, G: 220, B: 120, A: 255}
		rim := color.NRGBA{R: 11, G: 36, B: 20, A: 255}
		if side == 1 {
			glow, rim = color.NRGBA{R: 255, G: 90, B: 50, A: 255}, color.NRGBA{R: 59, G: 14, B: 7, A: 255}
		}
		img := g.glowText(name, f, glow, rim)
		w, h := float64(img.Bounds().Dx())/drawPx, float64(img.Bounds().Dy())/drawPx
		// each name is centred on the visible part of its own portrait
		anchorX := W * .25
		if side == 1 {
			anchorX = W * .76
		}
		cy := top + bandH/2 + shake
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-w/2*drawPx, -h/2*drawPx)
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate((anchorX+shake)*drawPx, cy*drawPx)
		op.ColorScale.ScaleAlpha(float32(min(1, k/.7)))
		dst.DrawImage(img, op)
	}
}

// glowText renders s in white with a soft coloured glow and a dark drop below it, at
// device resolution.
func (g *Game) glowText(s string, f *text.GoTextFace, glow, rim color.NRGBA) *ebiten.Image {
	key := fmt.Sprint("glow", s, f.Size, glow, drawPx)
	if im, ok := g.textCache[key]; ok {
		return im
	}
	if g.textCache == nil {
		g.textCache = map[string]*ebiten.Image{}
	}
	pad := 24.0
	w := int(math.Ceil((measure(s, f) + 2*pad) * drawPx))
	h := int(math.Ceil((f.Size*1.15 + 2*pad) * drawPx))
	im := ebiten.NewImage(w, h)
	draw := func(dx, dy float64, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate((pad+dx)*drawPx, (pad-f.Size*.1+dy)*drawPx)
		op.ColorScale.ScaleWithColor(c)
		sf := *f
		sf.Size *= drawPx
		text.Draw(im, s, &sf, op)
	}
	for r := 18.0; r > 2; r -= 4 { // the glow: rings of faint copies
		for i := range 12 {
			a := float64(i) * math.Pi / 6
			c := glow
			c.A = uint8(20 + 30*(1-r/18))
			draw(math.Cos(a)*r, math.Sin(a)*r, c)
		}
	}
	draw(0, 4, rim)
	draw(0, 0, color.White)
	g.textCache[key] = im
	return im
}

var cutRamps [2]*ebiten.Image

// cutRamp is the pane shading: [0] a 64×1 ramp, clear to 45% then to 42% black;
// [1] a 1×64 ramp, clear to 50% then to 40% black.
func cutRamp() [2]*ebiten.Image {
	if cutRamps[0] == nil {
		h := image.NewNRGBA(image.Rect(0, 0, 64, 1))
		v := image.NewNRGBA(image.Rect(0, 0, 1, 64))
		for i := range 64 {
			f := (float64(i) + .5) / 64
			h.SetNRGBA(i, 0, color.NRGBA{A: uint8(255 * .42 * max(0, (f-.45)/.55))})
			v.SetNRGBA(0, i, color.NRGBA{A: uint8(255 * .4 * max(0, (f-.5)/.5))})
		}
		cutRamps = [2]*ebiten.Image{ebiten.NewImageFromImage(h), ebiten.NewImageFromImage(v)}
	}
	return cutRamps
}
