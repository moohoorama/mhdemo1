package gui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
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
			b.trailRate = math.Abs(target-b.trail) / barLead
			b.hold = 0
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

// drawUnitBar draws u's bar centred on canvas point (x, top) from the UI sheet's bar parts, each
// cut to the share it shows; z is the canvas pixels per game pixel.
func (f *Field) drawUnitBar(u *unitVis, x, top, z float32) {
	l := float64(x - barWidth*z/2)
	part := func(name string, frame int, share float64, at, y float64, bright ...float32) {
		s := f.A.UI[name]
		im := s.Frames[frame]
		b := im.Bounds()
		w := int(math.Round(float64(b.Dx()) * share))
		if w <= 0 {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.Anchor[0], -s.Anchor[1])
		op.GeoM.Scale(float64(z), float64(z))
		op.GeoM.Translate(at, y)
		if f.deployment && !u.ally {
			op.ColorScale.Scale(.45, .45, .45, 1)
		}
		for _, k := range bright {
			op.ColorScale.Scale(k, k, k, 1)
		}
		f.Canvas.DrawImage(im.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Min.X+w, b.Max.Y)).(*ebiten.Image), op)
	}
	y, in := float64(top), l+float64(z) // the case's top left inside its edge, and the HP trough's left
	part("bar_frame", 0, 1, l, y)
	part("bar_trail", 0, u.bar.trail, in, y+float64(z))
	fill := u.bar.fill
	if u.shownHP > 0 {
		fill = max(fill, 1/float64(barWidth-2)) // never less than a pixel while the unit lives
	}
	side := 0
	if u.ally {
		side = 1
	}
	part("bar_fill", side, fill, in, y+float64(z))
	part("bar_ticks", 0, 1, in, y+float64(z))
	if u.shownMP > u.maxMP { // a drink poured past the max: the full bar brightens and dims
		part("bar_mp", 0, 1, in, y+5*float64(z), float32(.55+1.2*overMPPulse(f.clockMS/1000)))
	} else if u.shownMP > 0 {
		part("bar_mp", 0, max(1/float64(barWidth-2), float64(u.shownMP)/float64(max(1, u.maxMP))), in, y+5*float64(z))
	}
}
