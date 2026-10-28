package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/scenemap"
)

// A night stage is drawn under a blue dark, cut open round the fires burning on its cells, its
// torches (Stage.Lights), the lights its objects carry (a campfire, a lantern) and every unit (the torches the soldiers carry, so they can be told apart),
// with a warm glow over the fires. Radii are world pixels at zoom 1.
const (
	flameLight   = 150.0
	bigFireLight = 440.0
	emberLight   = 95.0
	torchLight   = 80.0
	unitLight    = 62.0
)

var (
	nightColor = color.NRGBA{R: 8, G: 14, B: 38, A: 165}
	fireGlow   = color.NRGBA{R: 255, G: 150, B: 60, A: 255}
)

// nightLayers holds what a night needs between frames.
type nightLayers struct {
	on     bool
	lights []image.Point
	fires  []nightFire      // a scene's fires
	props  []scenemap.Light // the lights the map's objects carry (scenemap.Attachment), canvas points
	color  color.NRGBA      // the dark's colour; nightColor when unset (a scene's dusk or dawn is lighter)
	flat   bool             // an even wash with no lights cut out (a scene remembered)
	dark   *ebiten.Image    // canvas-size scratch
	light  *ebiten.Image    // a soft disc: opaque at the centre, clear at the rim
}

// nightFire is a scene fire at fractional cells; a big fire is a house or a quarter burning, seen from afar.
type nightFire struct {
	u, v float64
	big  bool
}

// lightCells are a stage's torch cells as points.
func lightCells(ps []content.Point) []image.Point {
	out := make([]image.Point, len(ps))
	for i, p := range ps {
		out[i] = image.Pt(p.X, p.Y)
	}
	return out
}

func (f *Field) setNight(on bool, lights []image.Point) {
	f.night.on, f.night.lights = on, lights
}

// drawNight lays the dark over the canvas and lights it round fires, torches and units.
func (f *Field) drawNight(clockMS float64, units []*unitVis) {
	n := &f.night
	if !n.on {
		return
	}
	c := f.Canvas
	b := c.Bounds()
	if n.dark == nil || n.dark.Bounds() != b {
		if n.dark != nil {
			n.dark.Deallocate()
		}
		n.dark = ebiten.NewImage(b.Dx(), b.Dy())
	}
	if n.light == nil {
		n.light = softDisc()
	}
	type lamp struct {
		x, y   float64 // canvas point
		radius float64
		fire   bool
		big    bool
	}
	var lamps []lamp
	at := func(u, v, lift, radius float64, fire, big bool) lamp {
		x, y := f.Center(u, v)
		return lamp{x, y - lift, radius, fire, big}
	}
	cell := func(p image.Point, radius float64, fire bool) lamp {
		return at(float64(p.X), float64(p.Y), f.Lift(p.X, p.Y), radius, fire, false)
	}
	for p, look := range f.looks {
		switch look {
		case "flame":
			lamps = append(lamps, cell(p, flameLight, true))
		case "ember":
			lamps = append(lamps, cell(p, emberLight, true))
		}
	}
	for _, p := range n.lights {
		lamps = append(lamps, cell(p, torchLight, false))
	}
	for _, l := range n.props {
		lamps = append(lamps, lamp{l.X, l.Y, l.Radius, l.Glow, false})
	}
	for _, p := range n.fires {
		r := flameLight
		if p.big {
			r = bigFireLight
		}
		lamps = append(lamps, at(p.u, p.v, f.Lift(int(p.u), int(p.v)), r, true, p.big))
	}
	for _, u := range units {
		if !u.gone && u.visible() {
			lamps = append(lamps, at(u.u, u.v, u.lift+f.groundLift(u.u, u.v), unitLight, false, false))
		}
	}
	z := f.renderZoom
	if z <= 0 {
		z = 1
	}
	t := clockMS / 1000
	place := func(l lamp, scale float64) *ebiten.DrawImageOptions {
		x, y := f.renderPoint(l.x, l.y)
		su, sv := l.x/32, l.y/16 // a phase of its own for each lamp
		flicker := 1 + .05*math.Sin(t*7.3+su*13+sv*7) + .03*math.Sin(t*13.1+su)
		if l.big { // a big fire heaves slowly and flares
			flicker = 1 + .12*math.Sin(t*2.3+su*5) + .07*math.Sin(t*5.9+sv*3) + .04*math.Sin(t*13.7+su)
		}
		r := l.radius * z * flicker * scale
		w := float64(n.light.Bounds().Dx())
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-w/2, -w/2)
		op.GeoM.Scale(2*r/w, 2*r/w*.62) // flattened like the ground it lights
		op.GeoM.Translate(x, y)
		return op
	}
	dark := n.color
	if dark.A == 0 {
		dark = nightColor
	}
	n.dark.Fill(dark)
	if n.flat {
		lamps = nil
	}
	for _, l := range lamps {
		op := place(l, 1)
		op.Blend = ebiten.BlendDestinationOut
		n.dark.DrawImage(n.light, op)
	}
	c.DrawImage(n.dark, nil)
	for _, l := range lamps {
		if !l.fire {
			continue
		}
		op := place(l, .8)
		op.Blend = ebiten.BlendLighter
		op.ColorScale.ScaleWithColor(fireGlow)
		if l.big {
			op.ColorScale.ScaleAlpha(.34)
		} else {
			op.ColorScale.ScaleAlpha(.22)
		}
		c.DrawImage(n.light, op)
	}
}

// softDisc is a white disc whose alpha falls from 1 at the centre to 0 at the rim.
func softDisc() *ebiten.Image {
	const s = 128
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	for y := range s {
		for x := range s {
			d := math.Hypot(float64(x)+.5-s/2, float64(y)+.5-s/2) / (s / 2)
			a := max(0, 1-d)
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * a * a * (3 - 2*a))})
		}
	}
	return ebiten.NewImageFromImage(img)
}
