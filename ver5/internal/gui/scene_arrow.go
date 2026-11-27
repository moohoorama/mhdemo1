package gui

import (
	"fmt"
	"hash/fnv"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// sceneArrow is an arrow of a Play step with an Arrow: it leaves the archer's chest as the bow is let go (the end of
// the attack motion's drawing frames) and comes down on a cell, flying on a shallow arc.
type sceneArrow struct {
	fu, fv, fh float64 // where it leaves: the archer's cell and the height of its chest above the ground
	tu, tv     float64
	delay, t   float64
	dur        float64
	loosed     bool
	fire       bool    // a fire arrow: it flies burning and burns on where it comes down
	down       float64 // seconds it has stood in the ground
}

const (
	arrowPeak       = 44.0 // canvas px the arc climbs above a straight line
	arrowMinFlight  = .55
	arrowFlightCell = .07  // seconds per cell, the battle map's pace
	arrowFireLight  = 36.0 // world px a fire arrow lights round itself in the air
	arrowBurn       = 6.0  // seconds a fire arrow burns at most where it comes down; each its own share of it
)

// looseArrow queues the arrow of an archer playing anim: it leaves after the motion's first two frames, as in battle.
func (g *Game) looseArrow(v *unitVis, anim string, tu, tv float64, fire bool) {
	ms := v.art.Animations[anim].MS
	release := 0.0
	for i := 0; i < len(ms) && i < 2; i++ {
		release += float64(ms[i]) / 1000
	}
	d := math.Hypot(tu-v.u, tv-v.v)
	s := g.scene
	s.arrows = append(s.arrows, &sceneArrow{fu: v.u, fv: v.v, fh: g.field.Lift(int(math.Round(v.u)), int(math.Round(v.v))) + v.top*.55, tu: tu, tv: tv,
		delay: release, dur: math.Max(arrowMinFlight, d*arrowFlightCell), fire: fire})
}

func (g *Game) updateArrows(dt float64) {
	s := g.scene
	kept := s.arrows[:0]
	for _, a := range s.arrows {
		a.t += dt
		if !a.loosed && a.t >= a.delay {
			a.loosed = true
			g.sfxAtCell("shoot", a.fu, a.fv)
		}
		if a.t < a.delay+a.dur {
			kept = append(kept, a)
		} else if len(s.stuck) < maxStuck {
			s.stuck = append(s.stuck, stuckArrow{a})
		}
	}
	s.arrows = kept
	for _, a := range s.stuck {
		if a.fire {
			a.down += dt
		}
	}
}

// burnTime is how long a fire arrow burns where it came down: from half to all of arrowBurn, each its own.
func (a *sceneArrow) burnTime() float64 {
	return arrowBurn * (.5 + .5*chatHash(fmt.Sprintf("%.2f,%.2f", a.tu, a.tv), int(a.fu*7+a.fv)))
}

// burning is how bright a fire arrow down in the ground still burns, 1 to 0.
func (a *sceneArrow) burning() float64 {
	if !a.fire {
		return 0
	}
	life := a.burnTime()
	if a.down >= life {
		return 0
	}
	return math.Min(1, (life-a.down)/1.5)
}

// arrowLights are the lights the fire arrows give, in the air and burning in the ground, as canvas points.
func (g *Game) arrowLights(out []scenemap.Light) []scenemap.Light {
	s := g.scene
	if s == nil {
		return out
	}
	for _, a := range s.arrows {
		if a.fire && a.loosed {
			x, y := g.arrowAt(a, math.Min(1, (a.t-a.delay)/a.dur))
			out = append(out, scenemap.Light{X: x, Y: y, Radius: arrowFireLight, Glow: false})
		}
	}
	for _, a := range s.stuck {
		if k := a.burning(); k > 0 {
			x, y := g.arrowAt(a.sceneArrow, 1)
			out = append(out, scenemap.Light{X: x, Y: y, Radius: flameLight * (.18 + .12*k), Glow: true})
		}
	}
	return out
}

// stuckArrow is an arrow that came down and stands in the ground where it fell, slanting the way it flew.
type stuckArrow struct{ *sceneArrow }

const maxStuck = 600

// arrowAt is where an arrow is at share k of its flight, on the canvas: the arc climbs higher the further it flies.
func (g *Game) arrowAt(a *sceneArrow, k float64) (float64, float64) {
	fx0, fy0 := g.field.Center(a.fu, a.fv)
	tx, ty := g.field.Center(a.tu, a.tv)
	fy0 -= a.fh
	ty -= g.field.Lift(int(math.Round(a.tu)), int(math.Round(a.tv))) + 4
	peak := math.Max(arrowPeak, math.Hypot(tx-fx0, ty-fy0)*.22)
	return fx0 + (tx-fx0)*k, fy0 + (ty-fy0)*k - 4*peak*k*(1-k)
}

// drawArrows draws the arrows in flight with a thin shadow on the ground beneath.
func (g *Game) drawArrows(dst *ebiten.Image) {
	fx := g.assets.effect("arrow")
	if len(fx.Frames) == 0 {
		return
	}
	n := float64(len(fx.Frames))
	frame := func(a *sceneArrow, k float64) int {
		x, y := g.arrowAt(a, k-.02)
		x2, y2 := g.arrowAt(a, k)
		return int(math.Round(math.Atan2(y2-y, x2-x)/(2*math.Pi)*n+n)) % len(fx.Frames)
	}
	flame := g.assets.Anims["flame_small"]
	tongue := func(x, y, t, a float64) { // a small flame licking up from canvas point (x, y)
		if len(flame.Frames) > 0 {
			g.drawCanvasSprite(dst, flame.Frames[int(t*1000/float64(flame.MS))%len(flame.Frames)], x, y, a)
		}
	}
	for _, a := range g.scene.stuck {
		x, y := g.arrowAt(a.sceneArrow, 1)
		fx.draw(dst, frame(a.sceneArrow, 1), x, y+3, g.toScreen, g.zoom)
		if k := a.burning(); k > 0 {
			tongue(x, y+3, a.down+a.tu, k)
		}
	}
	for _, a := range g.scene.arrows {
		if !a.loosed {
			continue
		}
		k := math.Min(1, (a.t-a.delay)/a.dur)
		x, y := g.arrowAt(a, k)
		fx.draw(dst, frame(a, math.Min(1, k+.02)), x, y, g.toScreen, g.zoom)
		if a.fire {
			tongue(x, y+2, a.t, 1)
		}
	}
}

// chatHash spreads the actors that murmur: the same name always gets the same number, near names far apart.
func chatHash(id string, i int) float64 {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s/%d", id, i)
	return float64(h.Sum32()%1000) / 1000
}

// chatStart is the wait before an actor's first bubble: the step's Lag, else its slot's turn, so that the people
// who murmur take turns instead of all speaking together.
func chatStart(st scenes.Step, slot int) float64 {
	if st.Lag > 0 {
		return st.Lag
	}
	return float64(slot%chatSlots) * chatSlot
}

// chatRetry is how long an actor whose balloon would cover another waits before it tries again.
const chatRetry = .3

// speak takes the next line of the actor's murmur, when its turn has come, and sets the wait for the one after: the
// text and whether there is a bubble to show (a turn of silence is none). A line whose balloon would cover one that
// is up (clear says no) waits chatRetry and stays the next line.
func (c *chatterLine) speak(clear func(text string) bool) (string, bool) {
	text := c.lines[c.i%len(c.lines)]
	if text != "" && !clear(text) {
		c.t = chatRetry
		return "", false
	}
	c.i++
	c.t = chatPeriod
	return text, text != ""
}
