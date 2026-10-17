package gui

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// An officer's first appearance (docs/ui-mockup, idea 9): the scene dims and the ink-wash
// portrait seeps in through spreading blots of ink while, beside it, the name is brushed
// in, the courtesy name and a red rule appear and the biography types out one letter a
// frame. It holds long enough to read, then fades. A scene step {"intro": name} plays it;
// the scene waits for it and a click skips it.
const (
	introDelay  = .4
	introSeep   = 2.0
	introName   = .6
	introCourt  = .3 // after the name starts
	introRule   = .5
	introBioAt  = .4  // the biography starts typing this long after the name
	introTyping = 60. // letters a second: one a frame
	introHold   = 2.6 // after the portrait and the biography are both complete
	introFade   = .5
	introBlots  = 70
	introTex    = 512
)

type officerIntro struct {
	ref        string
	t          float64
	blots      []introBlot
	court, bio string
	typedBy    float64 // when the last letter of the biography is out
	length     float64
}

type introBlot struct{ x, y, r, at float64 }

type introImages struct {
	mask, canvas, disc, edge *ebiten.Image
}

func (g *Game) startIntro(ref string) {
	r := rand.New(rand.NewPCG(77, uint64(len(ref))))
	in := &officerIntro{ref: ref}
	if _, c, ok := g.character(ref); ok {
		in.court, in.bio = g.introLines(c.Biography)
	}
	in.typedBy = introDelay + introBioAt + float64(len([]rune(in.bio)))/introTyping
	in.length = max(introDelay+introSeep, in.typedBy) + introHold + introFade
	for i := range introBlots {
		spread := min(1, float64(i)/20+.2)
		in.blots = append(in.blots, introBlot{
			x: 256 + (r.Float64()-.5)*300*spread, y: 270 + (r.Float64()-.5)*300*spread,
			r: 30 + r.Float64()*90, at: r.Float64()*.5 + float64(i)/introBlots*.4,
		})
	}
	g.intro = in
}

func (g *Game) advanceIntro(dt float64) {
	if g.intro != nil {
		if g.intro.t += dt; g.intro.t >= g.intro.length {
			g.intro = nil
		}
	}
}

// introTextures are a soft disc (solid to 55% of its radius) and the portrait's edge
// fade, both as alpha masks.
func (g *Game) introTextures() *introImages {
	im := &g.introImg
	if im.disc != nil {
		return im
	}
	disc := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	edge := image.NewNRGBA(image.Rect(0, 0, introTex, introTex))
	for y := range 128 {
		for x := range 128 {
			d := math.Hypot(float64(x)+.5-64, float64(y)+.5-64) / 64
			disc.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * max(0, min(1, (1-d)/.45)))})
		}
	}
	for y := range introTex {
		for x := range introTex {
			d := math.Hypot(float64(x)+.5-256, float64(y)+.5-270)
			edge.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * max(0, min(1, (256-d)/106)))})
		}
	}
	im.disc, im.edge = ebiten.NewImageFromImage(disc), ebiten.NewImageFromImage(edge)
	im.mask, im.canvas = ebiten.NewImage(introTex, introTex), ebiten.NewImage(introTex, introTex)
	return im
}

func (g *Game) drawIntro(dst *ebiten.Image) {
	in := g.intro
	if in == nil {
		return
	}
	_, c, ok := g.character(in.ref)
	if !ok {
		return
	}
	t := in.t
	out := 1 - phase(t, in.length-introFade, introFade)
	rect(dst, 0, 0, float32(uiW), float32(uiH), color.NRGBA{R: 6, G: 5, B: 4, A: uint8(170 * min(1, t/.3) * out)})

	// the portrait seeps in where the ink blots have spread
	tex := g.introTextures()
	k := phase(t, introDelay, introSeep) * 1.25
	tex.mask.Clear()
	for _, b := range in.blots {
		e := max(0, min(1, (k-b.at)/.45))
		if e <= 0 {
			continue
		}
		rr := b.r * (1 - math.Pow(1-e, 3))
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(rr*2/128, rr*2/128)
		op.GeoM.Translate(b.x-rr, b.y-rr)
		tex.mask.DrawImage(tex.disc, op)
	}
	tex.canvas.Clear()
	if src := g.assets.Portraits[c.Look.Portrait]; src != nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(introTex/float64(src.Bounds().Dx()), introTex/float64(src.Bounds().Dy()))
		tex.canvas.DrawImage(src, op)
	}
	tex.canvas.DrawImage(tex.mask, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	tex.canvas.DrawImage(tex.edge, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	size := .82 * float64(uiH)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(size/introTex, size/introTex)
	op.GeoM.Translate(.06*float64(uiW), .09*float64(uiH))
	op.ColorScale.ScaleAlpha(float32(out))
	drawScaled(dst, tex.canvas, op)

	// name, courtesy name, rule and biography to the right
	x, y := .56*float64(uiW), .22*float64(uiH)
	nameAt := introDelay // the text runs alongside the portrait
	if t >= nameAt {
		face := g.brush(228)
		name := g.name(in.ref)
		img := g.textImage(name, face, color.NRGBA{R: 246, G: 238, B: 219, A: 255}, 6)
		drawReveal(dst, img, x, y-40, bezier(.4, .1, .3, 1, phase(t, nameAt, introName)), out)
	}
	if in.court != "" {
		ck := bezier(.2, .8, .2, 1, phase(t, nameAt+introCourt, introCourt))
		cc := gold
		cc.A = uint8(255 * ck * out)
		writeSpaced(dst, in.court, g.serif(31, 700), x, y+225+12*(1-ck), .2, cc)
	}
	rw := (float64(uiW)*.95 - x) * .7 * bezier(.2, .8, .2, 1, phase(t, nameAt+introCourt, introRule))
	for i := 0; i < 20; i++ {
		f := float64(i) / 20
		c := color.NRGBA{R: 184, G: 50, B: 31, A: uint8(255 * (1 - f) * out)}
		rect(dst, float32(x+rw*f), float32(y+290), float32(rw/20+1), 4, c)
	}
	typed := int((t - nameAt - introBioAt) * introTyping)
	if typed > 0 {
		bc := ink
		bc.A = uint8(255 * out)
		g.label(dst, prefixLines(wrapWidth(in.bio, g.face(26), float64(uiW)*.95-x), typed), x, y+316, 26, bc)
	}
}

// introLines splits a biography into the courtesy-name line and the rest.
func (g *Game) introLines(bio string) (court, rest string) {
	rest = bio
	if m := courtesyName.FindStringSubmatchIndex(bio); m != nil {
		court = "자 " + bio[m[2]:m[3]]
		rest = strings.TrimSpace(bio[:m[0]] + bio[m[1]:])
	}
	return court, rest
}
