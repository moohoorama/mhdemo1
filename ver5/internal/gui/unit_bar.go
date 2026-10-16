package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The unit bar (docs/ui-mockup, idea 4): about 1.2 sprite widths, HP split into four
// cells by ticks, team-coloured fill, a white trail that lingers after a hit and leads a
// heal, and a thin MP line. Sizes are game pixels at zoom 1.
const (
	barWidth   = 24
	barHeight  = 7
	barDrop    = .15 // seconds for the fill to fall to the new HP
	barHold    = .45 // the trail waits this long after a hit
	barShrink  = .6  // then shrinks over this long
	barLead    = .2  // on a heal the trail jumps ahead this fast
	barRiseLag = .15 // and the fill follows after this delay
	barRise    = .8  // over this long
)

// barState animates the shown fraction of a unit's HP bar.
type barState struct {
	ready               bool
	target, fill, trail float64
	hold, lag           float64
	fillRate, trailRate float64
}

func (b *barState) update(target, dt float64) {
	if !b.ready {
		*b = barState{ready: true, target: target, fill: target, trail: target}
		return
	}
	if target != b.target {
		switch {
		case target < b.fill:
			b.fillRate = (b.fill - target) / barDrop
			b.trailRate = (b.trail - target) / barShrink
			b.hold, b.lag = barHold, 0
		case target > b.fill:
			b.trailRate = (target - b.trail) / barLead
			b.fillRate = (target - b.fill) / barRise
			b.lag = barRiseLag
		}
		b.target = target
	}
	if b.fill > b.target {
		b.fill = max(b.target, b.fill-b.fillRate*dt)
	} else if b.fill < b.target {
		if b.lag > 0 {
			b.lag -= dt
		} else {
			b.fill = min(b.target, b.fill+b.fillRate*dt)
		}
	}
	if b.trail > b.target {
		if b.hold > 0 {
			b.hold -= dt
		} else {
			b.trail = max(b.target, b.trail-b.trailRate*dt)
		}
	} else if b.trail < b.target {
		b.trail = min(b.target, b.trail+b.trailRate*dt)
	}
	b.trail = max(b.trail, b.fill)
}

func dimmed(c color.RGBA, on bool) color.RGBA {
	if on {
		c.R, c.G, c.B = uint8(float64(c.R)*.45), uint8(float64(c.G)*.45), uint8(float64(c.B)*.45)
	}
	return c
}

// drawUnitBar draws u's bar centred on canvas point (x, top).
func (f *Field) drawUnitBar(u *unitVis, x, top, z float32) {
	dim := f.deployment && !u.ally
	fill := func(x0, y0, w, h float32, c color.RGBA) {
		if w > 0 && h > 0 {
			vector.FillRect(f.Canvas, x0, y0, w, h, dimmed(c, dim), false)
		}
	}
	w, h := barWidth*z, barHeight*z
	l := x - w/2
	fill(l-z, top-z, w+2*z, h+2*z, color.RGBA{0, 0, 0, 255})
	fill(l, top, w, h, color.RGBA{13, 9, 7, 255})
	fill(l, top, w, z, color.RGBA{48, 40, 36, 255})
	in := w - 2*z
	ty, th := top+z, 3*z
	fill(l+z, ty, in, th, color.RGBA{42, 23, 18, 255})
	fill(l+z, ty, in*float32(u.bar.trail), th, color.RGBA{246, 236, 220, 255})
	hi, lo := color.RGBA{238, 106, 79, 255}, color.RGBA{166, 44, 27, 255}
	if u.ally {
		hi, lo = color.RGBA{134, 222, 155, 255}, color.RGBA{59, 154, 87, 255}
	}
	fw := in * float32(u.bar.fill)
	if u.shownHP > 0 {
		fw = max(z, fw)
	}
	fill(l+z, ty, fw, th*.5, hi)
	fill(l+z, ty+th*.5, fw, th*.5, lo)
	for i := 1; i < 4; i++ {
		fill(l+z+in*float32(i)/4-z*.5, ty, max(1, z*.5), th, color.RGBA{0, 0, 0, 140})
	}
	my := ty + th + z
	fill(l+z, my, in, z, color.RGBA{20, 32, 46, 255})
	if u.shownMP > 0 {
		fill(l+z, my, max(z, in*float32(u.shownMP)/float32(max(1, u.maxMP))), z, color.RGBA{93, 154, 216, 255})
	}
}
