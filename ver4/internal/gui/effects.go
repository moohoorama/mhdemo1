package gui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

// effect is a short procedural animation on the field, in cell coordinates.
type effect struct {
	kind           string // arrow, slash, fire, water, heal, buff, status
	fu, fv, tu, tv float64
	t, dur         float64
}

// effectKind is how an attack or skill looks.
func (g *Game) effectKind(actor *unitVis, c core.Command) string {
	if c.Kind == "attack" {
		if actor != nil && actor.ranged {
			return "arrow"
		}
		return "slash"
	}
	s := g.S.Data.Skills[c.Skill]
	switch {
	case s.Mode == "화계":
		return "fire"
	case s.Mode == "수계":
		return "water"
	case s.Kind == "heal":
		return "heal"
	case s.Kind == "buff":
		return "buff"
	case s.Kind == "status":
		return "status"
	}
	return "slash"
}

func (g *Game) spawn(kind string, from, to *unitVis, dur float64) {
	if to == nil {
		return
	}
	e := &effect{kind: kind, tu: to.u, tv: to.v, dur: dur}
	e.fu, e.fv = e.tu, e.tv
	if from != nil {
		e.fu, e.fv = from.u, from.v
	}
	g.effects = append(g.effects, e)
}

// impact starts the effects at every unit an action's events reach.
func (g *Game) impact(kind string, actor *unitVis, events []core.Event) {
	if kind == "arrow" {
		kind = "slash"
	}
	seen := map[string]bool{}
	for _, e := range events {
		switch e.Kind {
		case "damage", "heal", "effect", "miss":
			if seen[e.Target] || e.Kind == "damage" && actor != nil && e.Target == actor.id {
				continue
			}
			seen[e.Target] = true
			g.spawn(kind, actor, g.unitAt(e.Target), .6)
		}
	}
}

func (g *Game) updateEffects(dt float64) {
	kept := g.effects[:0]
	for _, e := range g.effects {
		e.t += dt
		if e.t < e.dur {
			kept = append(kept, e)
		}
	}
	g.effects = kept
}

func (g *Game) drawEffects(dst *ebiten.Image) {
	z := float32(g.zoom)
	for _, e := range g.effects {
		k := e.t / e.dur
		fade := uint8(255 * (1 - k))
		x, y := g.field.Center(e.tu, e.tv)
		sx, sy := g.toScreen(x, y-14)
		px, py := float32(sx), float32(sy)
		switch e.kind {
		case "arrow":
			fx, fy := g.field.Center(e.fu, e.fv)
			ax, ay := g.toScreen(fx, fy-14)
			hx, hy := ax+(sx-ax)*k, ay+(sy-ay)*k-math.Sin(k*math.Pi)*24*float64(z)
			dx, dy := sx-ax, sy-ay
			n := math.Hypot(dx, dy)
			if n == 0 {
				continue
			}
			tx, ty := float32(hx-dx/n*10*float64(z)), float32(hy-dy/n*10*float64(z))
			vector.StrokeLine(dst, tx, ty, float32(hx), float32(hy), z, color.NRGBA{R: 240, G: 226, B: 190, A: 255}, true)
		case "slash":
			r := 12 * z * float32(.5+k)
			c := color.NRGBA{R: 255, G: 250, B: 230, A: fade}
			vector.StrokeLine(dst, px-r, py-r, px+r, py+r, 2*z/2+1, c, true)
			vector.StrokeLine(dst, px+r*.8, py-r, px-r*.6, py+r, 2*z/2+1, c, true)
		case "fire":
			for i := 0; i < 7; i++ {
				a := float64(i) * 2 * math.Pi / 7
				rise := float32(k * 26)
				ox := float32(math.Cos(a+k*3)) * 9 * z * float32(.4+k)
				r := (6 - float32(i%3)) * z * float32(1-k*.6)
				c := color.NRGBA{R: 255, G: uint8(120 + 20*i), B: 40, A: fade}
				vector.FillCircle(dst, px+ox, py-rise*z/2-float32(i%3)*4*z, r, c, true)
			}
		case "water":
			vector.StrokeCircle(dst, px, py+6*z, 18*z*float32(k), 2*z, color.NRGBA{R: 120, G: 190, B: 255, A: fade}, true)
			for i := 0; i < 6; i++ {
				a := float64(i) * math.Pi / 3
				vector.FillCircle(dst, px+float32(math.Cos(a))*14*z*float32(k), py-float32(math.Sin(k*math.Pi))*16*z+float32(i%2)*4*z, 2*z, color.NRGBA{R: 170, G: 220, B: 255, A: fade}, true)
			}
		case "heal":
			for i := 0; i < 8; i++ {
				ox := float32(math.Sin(float64(i)*1.7)) * 10 * z
				rise := float32(math.Mod(k+float64(i)*.13, 1)) * 26 * z
				vector.FillRect(dst, px+ox, py+6*z-rise, 2*z, 2*z, color.NRGBA{R: 150, G: 255, B: 150, A: fade}, false)
			}
		case "buff":
			for i := 0; i < 2; i++ {
				kk := math.Mod(k+float64(i)*.5, 1)
				vector.StrokeCircle(dst, px, py+8*z-float32(kk)*20*z, 12*z, 1.5*z, color.NRGBA{R: 255, G: 214, B: 110, A: fade}, true)
			}
		case "status":
			for i := 0; i < 5; i++ {
				a := k*6 + float64(i)*2*math.Pi/5
				vector.FillCircle(dst, px+float32(math.Cos(a))*11*z, py-6*z+float32(math.Sin(a))*5*z, 2*z, color.NRGBA{R: 196, G: 120, B: 255, A: fade}, true)
			}
		}
	}
}
