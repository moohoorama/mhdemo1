package gui

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// effect is a short animation from the fx sheet on the field. Positions are canvas pixels
// relative to the target's feet (fx, fy for the source) and scale with the camera zoom like
// the units do.
type effect struct {
	kind   string // arrow, slash, fire, water, heal, buff, debuff, block, insu, promote_gather, promote
	tu, tv float64
	fu, fv float64
	lift   float64 // target cell surface height
	height float64 // target sprite height, to aim at the chest
	flift  float64 // source cell surface height and sprite height, where an arrow leaves
	fh     float64
	tint   color.NRGBA
	t, dur float64
	parts  []particle
}

// particle moves ballistically from the effect origin; gravity pulls it down the screen.
type particle struct {
	x, y, vx, vy, g float64
	life            float64
	sprite          string // the fx sprite drawn, at frame
	frame           int
	col             color.NRGBA // the tint of a white sprite; zero for one drawn in its colours
	in              bool        // brightens as it goes instead of fading: drawn in toward the origin
}

var (
	hot   = color.NRGBA{R: 255, G: 248, B: 220, A: 255}
	spark = color.NRGBA{R: 255, G: 214, B: 110, A: 255}
	healC = color.NRGBA{R: 150, G: 255, B: 150, A: 255}
)

// buffTint is the color of a support effect.
var buffTint = map[string]color.NRGBA{
	"speed": {90, 220, 255, 255}, "counter": {210, 220, 240, 255},
	"range": {150, 235, 120, 255}, "morale": {255, 210, 90, 255}, "attack": {255, 110, 80, 255},
	"defense": {100, 160, 255, 255},
}

// effectKind is how an attack or skill looks.
func (g *Game) effectKind(actor *unitVis, c core.Command) string {
	if c.Kind == "attack" {
		if actor != nil && actor.siege {
			return "stone"
		}
		if actor != nil && actor.ranged {
			return "arrow"
		}
		return "slash"
	}
	s := g.S.Data.Skills[c.Skill]
	switch {
	case s.Element == "fire":
		return "fire"
	case s.Element == "water":
		return "water"
	case s.Kind == "heal":
		return "heal"
	case s.Kind == "buff":
		return "buff"
	case s.Kind == "status":
		return "debuff"
	}
	return "slash"
}

func (g *Game) spawn(kind string, from, to *unitVis, dur float64, tint color.NRGBA) {
	if to == nil {
		return
	}
	e := &effect{kind: kind, tu: to.u, tv: to.v, fu: to.u, fv: to.v, lift: to.lift, height: to.top, dur: dur, tint: tint}
	if from != nil {
		e.fu, e.fv, e.flift, e.fh = from.u, from.v, from.lift, from.top
	}
	r := rand.New(rand.NewPCG(uint64(g.ticks), uint64(len(g.effects))))
	chest := -e.height * .55
	add := func(p particle) { e.parts = append(e.parts, p) }
	switch kind {
	case "slash":
		for i := 0; i < 14; i++ {
			a := r.Float64() * 2 * math.Pi
			v := 60 + r.Float64()*110
			add(particle{x: 0, y: chest, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 40, g: 260,
				life: .25 + r.Float64()*.3, sprite: "spark", col: spark})
		}
	case "fire":
		for i := 0; i < 40; i++ {
			add(particle{x: (r.Float64() - .5) * 18, y: (r.Float64() - .2) * 6, vx: (r.Float64() - .5) * 14,
				vy: -30 - r.Float64()*55, g: -20, life: .45 + r.Float64()*.5, sprite: "ember", frame: r.IntN(3)})
		}
		for i := 0; i < 10; i++ {
			add(particle{x: (r.Float64() - .5) * 14, y: chest, vx: (r.Float64() - .5) * 70, vy: -60 - r.Float64()*60,
				g: 120, life: .6 + r.Float64()*.4, sprite: "ember"})
		}
	case "water":
		for i := 0; i < 22; i++ {
			a := -math.Pi * (.1 + .8*r.Float64())
			v := 50 + r.Float64()*80
			add(particle{x: (r.Float64() - .5) * 10, y: 0, vx: math.Cos(a) * v, vy: math.Sin(a) * v, g: 260,
				life: .5 + r.Float64()*.4, sprite: "water_drop", frame: r.IntN(3)})
		}
	case "heal":
		for i := 0; i < 12; i++ {
			add(particle{x: (r.Float64() - .5) * 22, y: (r.Float64() - .5) * 6, vy: -18 - r.Float64()*26,
				life: .7 + r.Float64()*.4, sprite: "heal_mote"})
		}
	case "buff":
		for i := 0; i < 14; i++ {
			sprite := "buff_dot"
			if i%3 == 0 {
				sprite = "buff_arrow"
			}
			add(particle{x: (r.Float64() - .5) * 24, y: (r.Float64() - .5) * 6, vy: -24 - r.Float64()*34,
				life: .7 + r.Float64()*.4, col: tint, sprite: sprite})
		}
	case "debuff":
		for i := 0; i < 8; i++ {
			add(particle{x: (r.Float64() - .5) * 20, y: chest - 10 - r.Float64()*8, vy: 14 + r.Float64()*16,
				life: .7 + r.Float64()*.3, sprite: "debuff_arrow"})
		}
	case "critical":
		for i := 0; i < 34; i++ {
			a := r.Float64() * 2 * math.Pi
			v := 120 + r.Float64()*200
			col := hot
			if i%3 == 0 {
				col = spark
			}
			add(particle{x: 0, y: chest, vx: math.Cos(a) * v, vy: math.Sin(a)*v*.7 - 50, g: 320,
				life: .3 + r.Float64()*.35, sprite: "spark", col: col})
		}
	case "promote_gather":
		// motes come in from all round to the chest, ending where they meet
		for i := 0; i < 24; i++ {
			a := r.Float64() * 2 * math.Pi
			d := 34 + r.Float64()*26
			life := .7 + r.Float64()*.3
			x, y := math.Cos(a)*d, math.Sin(a)*d*.5+chest*.6
			add(particle{x: x, y: y, vx: -x / life, vy: (chest*.6 - y) / life, life: life, sprite: "promote_mote", in: true})
		}
	case "promote":
		for i := 0; i < 26; i++ {
			a := r.Float64() * 2 * math.Pi
			v := 50 + r.Float64()*70
			add(particle{x: 0, y: chest * .6, vx: math.Cos(a) * v, vy: math.Sin(a)*v*.5 - 30, g: -40,
				life: .5 + r.Float64()*.4, sprite: "promote_mote"})
		}
	case "block":
		// sparks fly back toward the attacker's side
		fx, _ := g.field.Center(e.fu, e.fv)
		tx, _ := g.field.Center(e.tu, e.tv)
		side := -1.0
		if fx > tx {
			side = 1
		}
		for i := 0; i < 12; i++ {
			a := -math.Pi/2 + side*(.3+r.Float64()*1.1)
			v := 70 + r.Float64()*90
			add(particle{x: side * 8, y: chest, vx: math.Cos(a) * v, vy: math.Sin(a) * v, g: 300,
				life: .2 + r.Float64()*.25, sprite: "spark", col: spark})
		}
	}
	g.effects = append(g.effects, e)
}

// impact starts the effects at every unit an action's events reach, and shakes the
// screen a little for heavy blows. A stone shakes it always and raises dust at every other
// unit it reaches (its landing cell has the stone's own dust).
func (g *Game) impact(kind string, actor, target *unitVis, events []core.Event, tint color.NRGBA) {
	if kind == "arrow" {
		kind = "slash"
	}
	if kind == "stone" {
		g.shakeT = math.Max(g.shakeT, .22)
	}
	seen := map[string]bool{}
	for _, e := range events {
		switch e.Kind {
		case "critical":
			if t := g.unitAt(e.Target); t != nil {
				g.spawn("critical", actor, t, .7, hot)
				g.shakeT = .4
			}
		case "damage", "heal", "effect":
			if seen[e.Target] || e.Kind == "damage" && actor != nil && e.Target == actor.id {
				continue
			}
			seen[e.Target] = true
			t := g.unitAt(e.Target)
			if kind == "stone" {
				if t != nil && t != target {
					g.dust(t)
				}
				continue
			}
			dur := map[string]float64{"slash": .55, "fire": 1.0, "water": .9, "heal": 1.1, "buff": 1.2, "debuff": 1.1}[kind]
			g.spawn(kind, actor, t, dur, tint)
			if e.Kind == "damage" && t != nil && e.Amount*5 >= t.maxHP {
				g.shakeT = .22
			}
		case "miss":
			if t := g.unitAt(e.Target); t != nil && !seen[e.Target] {
				seen[e.Target] = true
				if kind == "stone" && t != target {
					g.dust(t)
				}
				g.spawn("block", actor, t, .6, spark)
			}
		}
	}
}

func (g *Game) updateEffects(dt float64) {
	lobs := g.lobs[:0]
	for _, l := range g.lobs {
		l.Update(dt)
		if !l.Done(g.assets.Projectiles) {
			lobs = append(lobs, l)
		}
	}
	g.lobs = lobs
	kept := g.effects[:0]
	for _, e := range g.effects {
		e.t += dt
		if e.t < e.dur {
			kept = append(kept, e)
		}
	}
	g.effects = kept
	g.shakeT = math.Max(0, g.shakeT-dt)
}

// shake is the screen offset of the current heavy-blow shake.
func (g *Game) shake() (float64, float64) {
	if g.shakeT <= 0 {
		return 0, 0
	}
	k := g.shakeT / .22 * 4
	return math.Sin(g.shakeT*90) * k, math.Cos(g.shakeT*70) * k * .6
}

func (g *Game) drawEffects(dst *ebiten.Image) {
	z := g.zoom
	defer func() {
		for _, l := range g.lobs {
			l.Draw(dst, g.assets.Projectiles, g.toScreen, z)
		}
	}()
	for _, e := range g.effects {
		k := e.t / e.dur
		x, y := g.field.Center(e.tu, e.tv)
		chest := y - e.lift - e.height*.55
		switch e.kind {
		case "arrow": // a flat shot from the shooter's chest to the target's
			fx, fy := g.field.Center(e.fu, e.fv)
			ax, ay := fx, fy-e.flift-e.fh*.55
			bx, by := x, chest
			sh := g.assets.FX["arrow"]
			n := float64(len(sh.Frames))
			i := int(math.Round(math.Atan2(by-ay, bx-ax)/(2*math.Pi)*n+n)) % len(sh.Frames)
			sh.draw(dst, i, ax+(bx-ax)*k, ay+(by-ay)*k, g.toScreen, z)
		case "slash":
			sh := g.assets.FX["slash"]
			sh.draw(dst, sh.frameOver(k), x, chest, g.toScreen, z)
		case "fire", "water", "heal", "debuff":
			sh := g.assets.FX[e.kind]
			sh.draw(dst, sh.frameOver(k), x, y-e.lift, g.toScreen, z)
		case "buff":
			sh := g.assets.FX["buff"]
			sh.drawTint(dst, sh.frameOver(k), x, y-e.lift, g.toScreen, z, e.tint, 1)
		case "block": // the shield faces the blow
			fx, _ := g.field.Center(e.fu, e.fv)
			sh := g.assets.FX["block"]
			sh.blit(dst, sh.frameOver(k), x, chest, g.toScreen, z, white, 1, fx < x)
		case "insu": // the seal of the new rank rises over the head, dot for dot with the sprites
			k := 1 - math.Pow(1-min(1, e.t/.35), 2)
			a := min(1, e.t/.15, (e.dur-e.t)/.2)
			sh := g.assets.UI["item_insu"]
			sx, sy := g.toScreen(x, y-e.lift-e.height-14-6*k)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-float64(sh.Frames[0].Bounds().Dx())/2, -float64(sh.Frames[0].Bounds().Dy())/2)
			op.GeoM.Scale(z/4, z/4)
			op.GeoM.Translate(math.Round(sx), math.Round(sy))
			op.ColorScale.ScaleAlpha(float32(a))
			drawScaled(dst, sh.Frames[0], op)
		case "promote_gather", "promote":
			sh := g.assets.FX[e.kind]
			sh.draw(dst, sh.frameOver(k), x, y-e.lift, g.toScreen, z)
		case "critical": // a white burst and rays at the chest, a shockwave on the ground
			sh, wave := g.assets.FX["critical"], g.assets.FX["critical_wave"]
			wave.draw(dst, wave.frameOver(k), x, y-e.lift, g.toScreen, z)
			sh.draw(dst, sh.frameOver(k), x, chest, g.toScreen, z)
		}
		for _, q := range e.parts {
			if e.t > q.life {
				continue
			}
			sh := g.assets.FX[q.sprite]
			qx, qy := x+q.x+q.vx*e.t, y-e.lift+q.y+q.vy*e.t+.5*q.g*e.t*e.t
			c := white
			if q.col.A > 0 {
				c = q.col
			}
			a := 1 - e.t/q.life
			if q.in {
				a = e.t / q.life
			}
			sh.drawTint(dst, q.frame, qx, qy, g.toScreen, z, c, a)
		}
	}
}
