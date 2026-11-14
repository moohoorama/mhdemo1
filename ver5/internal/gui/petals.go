package gui

import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// Peach petals falling onto the scene map. Each petal falls from its own height onto its own spot
// of ground, lies there a moment and fades. A petal z canvas px up is that much nearer the camera,
// so it is drawn f(z) times as far from the screen's centre as its spot of ground, and larger and
// lighter: it sweeps past faster than the ground when the camera pans and settles onto its spot
// as it lands.
const (
	petalTop     = 150.0               // highest start above the ground, canvas px
	petalEye     = 210.0               // the camera's height over the ground, canvas px
	petalDensity = 200 / (770.0 * 465) // petals per square canvas px of the window they fall in
	petalWind    = 16.0                // canvas px a second
	petalFadeIn  = .6                  // seconds
	petalFadeOut = .8                  // seconds
)

var petalColors = []color.NRGBA{{R: 242, G: 167, B: 184, A: 255}, {R: 255, G: 214, B: 222, A: 255}, {R: 215, G: 119, B: 144, A: 255}}

type petal struct {
	x, y  float64 // its spot of ground, canvas px
	z     float64 // height above it
	a     float64 // opacity
	frame int     // tumbling: the petal frames are 2, 3 and 4 wide
	c     color.NRGBA
}

// petals lays out the petals t seconds into the effect, lowest first, into buf. They are pure
// functions of t and the camera: petal i's r-th fall picks its spot by hashing (i, r), and the
// spots repeat every window's width and height so that the window around the camera always holds
// them. The window reaches past the screen far enough that no petal appears or vanishes on it.
func (g *Game) petals(t float64, buf []petal) []petal {
	vp := g.battleViewport()
	vw, vh := float64(vp.Dx())/g.zoom, float64(vp.Dy())/g.zoom
	x0, y0 := g.camX-vw/2-100, g.camY-vh/2-60 // wind blows to the right; high petals land below where they show
	w, h := vw+160, vh+225
	wrap := func(v, lo, span float64) float64 {
		m := math.Mod(v-lo, span)
		if m < 0 {
			m += span
		}
		return lo + m
	}
	for i := range int(petalDensity * w * h) {
		k := 60 * hash3(i, -1, 0)
		speed := 18 + 14*hash3(i, -1, 1)
		drift := petalWind * (.6 + .5*hash3(i, -1, 2))
		top := petalTop * (.55 + .45*hash3(i, -1, 3))
		rest := .35 + .4*hash3(i, -1, 4)
		fall := top / speed
		life := fall + rest + petalFadeOut
		age := t + life*hash3(i, -1, 5)
		r := math.Floor(age / life)
		since := age - r*life
		air := math.Min(since, fall)
		start := t - since
		p := petal{frame: 1, c: petalColors[i%len(petalColors)], a: math.Min(1, since/petalFadeIn)}
		p.x = wrap(w*hash3(i, int(r), 6)+drift*air+7/1.3*(math.Sin(1.3*(start+air)+k)-math.Sin(1.3*start+k)), x0, w)
		p.y = wrap(h*hash3(i, int(r), 7), y0, h)
		if since < fall {
			p.z = top*(1-since/fall) + 3*math.Sin(2.1*since+k)*math.Sin(math.Pi*since/fall)
			p.frame = min(2, int(math.Abs(math.Sin(t*3+k))*3))
		} else if since > fall+rest {
			p.a *= 1 - (since-fall-rest)/petalFadeOut
		}
		buf = append(buf, p)
	}
	slices.SortFunc(buf, func(a, b petal) int { return cmp.Compare(a.z, b.z) })
	return buf
}

// drawPetals draws the petals t seconds into the effect: the shadows of those in the air, then the
// petals from the ground up.
func (g *Game) drawPetals(dst *ebiten.Image, t float64) {
	s := g.scene
	s.petals = g.petals(t, s.petals[:0])
	z := g.zoom
	cx, cy := g.toScreen(g.camX, g.camY)
	for _, p := range s.petals {
		if p.z > 0 {
			x, y := g.toScreen(p.x, p.y)
			a := .3 * (1 - p.z/(petalTop*1.2)) * p.a
			rect(dst, float32(math.Round(x)), float32(math.Round(y+2*z)), float32(3*z), float32(z), color.NRGBA{R: 20, G: 32, B: 14, A: uint8(255 * a)})
		}
	}
	fx := g.assets.effect("petal")
	for _, p := range s.petals {
		f := 1 / (1 - p.z/petalEye)
		x, y := g.toScreen(p.x, p.y)
		x, y = cx+(x-cx)*f, cy+(y-cy)*f-p.z*z
		lift := -.1
		if p.z > 0 {
			lift = .45*(f-1) - .08*(1-p.z/petalTop)
		}
		k := z * f
		fx.drawTint(dst, p.frame, 0, 0, func(px, py float64) (float64, float64) { return x + px*k, y + py*k }, k, lighten(p.c, lift), p.a)
	}
}

// lighten moves c toward white by k (0..1), or toward black by -k.
func lighten(c color.NRGBA, k float64) color.NRGBA {
	ch := func(v uint8) uint8 {
		x := float64(v)
		if k > 0 {
			return uint8(x + (255-x)*math.Min(1, k))
		}
		return uint8(x * (1 + k))
	}
	return color.NRGBA{R: ch(c.R), G: ch(c.G), B: ch(c.B), A: c.A}
}
