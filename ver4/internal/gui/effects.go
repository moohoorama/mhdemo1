package gui

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

// effect is a short procedural animation on the field. Positions are canvas pixels
// relative to the target's feet (fx, fy for the source); sizes are canvas pixels and
// scale with the camera zoom like the units do.
type effect struct {
	kind   string // arrow, slash, fire, water, heal, buff, debuff, block
	tu, tv float64
	fu, fv float64
	lift   float64 // target cell surface height
	height float64 // target sprite height, to aim at the chest
	tint   color.NRGBA
	t, dur float64
	parts  []particle
}

// particle moves ballistically from the effect origin; gravity pulls it down the screen.
type particle struct {
	x, y, vx, vy, g float64
	size, life      float64
	col             color.NRGBA
	shape           int // 0 dot, 1 spark line, 2 plus, 3 up arrow, 4 down arrow
}

var (
	hot     = color.NRGBA{R: 255, G: 248, B: 220, A: 255}
	spark   = color.NRGBA{R: 255, G: 214, B: 110, A: 255}
	flameC  = []color.NRGBA{{255, 236, 140, 255}, {255, 170, 60, 255}, {236, 90, 40, 255}, {150, 40, 30, 255}}
	waterC  = []color.NRGBA{{230, 248, 255, 255}, {140, 210, 255, 255}, {70, 150, 230, 255}}
	healC   = color.NRGBA{R: 150, G: 255, B: 150, A: 255}
	debuffC = color.NRGBA{R: 186, G: 110, B: 255, A: 255}
)

// buffTint is the color of a support effect.
var buffTint = map[string]color.NRGBA{
	"speed": {90, 220, 255, 255}, "charge": {255, 150, 60, 255}, "counter": {210, 220, 240, 255},
	"range": {150, 235, 120, 255}, "morale": {255, 210, 90, 255}, "attack": {255, 110, 80, 255},
	"defense": {100, 160, 255, 255},
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
		e.fu, e.fv = from.u, from.v
	}
	r := rand.New(rand.NewPCG(uint64(g.ticks), uint64(len(g.effects))))
	chest := -e.height * .55
	add := func(p particle) { e.parts = append(e.parts, p) }
	switch kind {
	case "slash":
		for i := 0; i < 14; i++ {
			a := r.Float64() * 2 * math.Pi
			v := 60 + r.Float64()*110
			add(particle{x: 0, y: chest, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 40, g: 260, size: 1 + r.Float64()*1.5,
				life: .25 + r.Float64()*.3, col: spark, shape: 1})
		}
	case "fire":
		for i := 0; i < 26; i++ {
			add(particle{x: (r.Float64() - .5) * 18, y: (r.Float64() - .2) * 6, vx: (r.Float64() - .5) * 14,
				vy: -30 - r.Float64()*55, g: -20, size: 2.5 + r.Float64()*3.5, life: .45 + r.Float64()*.5,
				col: flameC[r.IntN(3)]})
		}
		for i := 0; i < 10; i++ {
			add(particle{x: (r.Float64() - .5) * 14, y: chest, vx: (r.Float64() - .5) * 70, vy: -60 - r.Float64()*60,
				g: 120, size: 1, life: .6 + r.Float64()*.4, col: flameC[0], shape: 1})
		}
	case "water":
		for i := 0; i < 22; i++ {
			a := -math.Pi * (.1 + .8*r.Float64())
			v := 50 + r.Float64()*80
			add(particle{x: (r.Float64() - .5) * 10, y: 0, vx: math.Cos(a) * v, vy: math.Sin(a) * v, g: 260,
				size: 1.2 + r.Float64()*1.8, life: .5 + r.Float64()*.4, col: waterC[r.IntN(3)]})
		}
	case "heal":
		for i := 0; i < 12; i++ {
			add(particle{x: (r.Float64() - .5) * 22, y: (r.Float64() - .5) * 6, vy: -18 - r.Float64()*26,
				size: 2 + r.Float64()*1.5, life: .7 + r.Float64()*.4, col: healC, shape: 2})
		}
	case "buff":
		for i := 0; i < 14; i++ {
			shape := 0
			if i%3 == 0 {
				shape = 3
			}
			add(particle{x: (r.Float64() - .5) * 24, y: (r.Float64() - .5) * 6, vy: -24 - r.Float64()*34,
				size: 1.5 + r.Float64()*1.5, life: .7 + r.Float64()*.4, col: tint, shape: shape})
		}
	case "debuff":
		for i := 0; i < 8; i++ {
			add(particle{x: (r.Float64() - .5) * 20, y: chest - 10 - r.Float64()*8, vy: 14 + r.Float64()*16,
				size: 2, life: .7 + r.Float64()*.3, col: debuffC, shape: 4})
		}
	case "critical":
		for i := 0; i < 34; i++ {
			a := r.Float64() * 2 * math.Pi
			v := 120 + r.Float64()*200
			col := hot
			if i%3 == 0 {
				col = spark
			}
			add(particle{x: 0, y: chest, vx: math.Cos(a) * v, vy: math.Sin(a)*v*.7 - 50, g: 320, size: 1.6 + r.Float64()*1.6,
				life: .3 + r.Float64()*.35, col: col, shape: 1})
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
				size: 1 + r.Float64(), life: .2 + r.Float64()*.25, col: spark, shape: 1})
		}
	}
	g.effects = append(g.effects, e)
}

// impact starts the effects at every unit an action's events reach, and shakes the
// screen a little for heavy blows.
func (g *Game) impact(kind string, actor *unitVis, events []core.Event, tint color.NRGBA) {
	if kind == "arrow" {
		kind = "slash"
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
			dur := map[string]float64{"slash": .55, "fire": 1.0, "water": .9, "heal": 1.1, "buff": 1.2, "debuff": 1.1}[kind]
			g.spawn(kind, actor, t, dur, tint)
			if e.Kind == "damage" && t != nil && e.Amount*5 >= t.maxHP {
				g.shakeT = .22
			}
		case "miss":
			if t := g.unitAt(e.Target); t != nil && !seen[e.Target] {
				seen[e.Target] = true
				g.spawn("block", actor, t, .6, spark)
			}
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

func with8(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(max(0, min(255, a*255)))
	return c
}

var lighter = &vector.DrawPathOptions{AntiAlias: true, Blend: ebiten.BlendLighter}

// glow is an additive soft disc (three nested circles).
func glow(dst *ebiten.Image, x, y, r float32, c color.NRGBA, a float64) {
	for i, k := range []float32{1, .66, .36} {
		var p vector.Path
		p.Arc(x, y, r*k, 0, 2*math.Pi, vector.Clockwise)
		op := *lighter
		op.ColorScale.ScaleWithColor(with8(c, a*[]float64{.18, .28, .45}[i]))
		fillPath(dst, &p, &op)
	}
}

// ellipse strokes a flat ground ellipse (iso: half as tall as wide).
func ellipse(dst *ebiten.Image, x, y, r, w float32, c color.NRGBA, a float64) {
	var p vector.Path
	for i := 0; i <= 32; i++ {
		t := float64(i) / 32 * 2 * math.Pi
		px, py := x+r*float32(math.Cos(t)), y+r*.5*float32(math.Sin(t))
		if i == 0 {
			p.MoveTo(px, py)
		} else {
			p.LineTo(px, py)
		}
	}
	op := *lighter
	op.ColorScale.ScaleWithColor(with8(c, a))
	strokePath(dst, &p, vector.StrokeOptions{Width: w}, &op)
}

func stroke(dst *ebiten.Image, x0, y0, x1, y1, w float32, c color.NRGBA, a float64) {
	strokeLine(dst, x0, y0, x1, y1, w, with8(c, a))
}

func (g *Game) drawEffects(dst *ebiten.Image) {
	z := g.zoom
	for _, e := range g.effects {
		k := e.t / e.dur
		x, y := g.field.Center(e.tu, e.tv)
		sx, sy := g.toScreen(x, y-e.lift)
		px, py := float32(sx), float32(sy)
		zf := float32(z)
		chest := py - float32(e.height*.55*z)
		switch e.kind {
		case "arrow":
			fx, fy := g.field.Center(e.fu, e.fv)
			ax, ay := g.toScreen(fx, fy-e.height*.55)
			bx, by := float64(px), float64(chest)
			hx, hy := ax+(bx-ax)*k, ay+(by-ay)*k-math.Sin(k*math.Pi)*30*z
			// direction along the arc
			k2 := math.Min(1, k+.05)
			nx, ny := ax+(bx-ax)*k2-hx, ay+(by-ay)*k2-math.Sin(k2*math.Pi)*30*z-hy
			n := math.Hypot(nx, ny)
			if n == 0 {
				continue
			}
			nx, ny = nx/n, ny/n
			L := 9 * z
			tx, ty := hx-nx*L, hy-ny*L
			for i := 1; i <= 4; i++ { // faint trail
				d := float64(i) * 5 * z
				stroke(dst, float32(tx-nx*d), float32(ty-ny*d), float32(tx-nx*(d-4*z)), float32(ty-ny*(d-4*z)), zf, hot, .35-.07*float64(i))
			}
			stroke(dst, float32(tx), float32(ty), float32(hx), float32(hy), 1.4*zf, color.NRGBA{200, 160, 100, 255}, 1)
			stroke(dst, float32(hx), float32(hy), float32(hx+nx*3*z), float32(hy+ny*3*z), 2*zf, hot, 1)
			stroke(dst, float32(tx), float32(ty), float32(tx-nx*3*z-ny*2*z), float32(ty-ny*3*z+nx*2*z), zf, color.NRGBA{240, 240, 240, 255}, 1)
		case "slash":
			if k < .45 { // two crossing crescents drawn on
				d := k / .45
				for i, s := range []float64{1, -1} {
					var p vector.Path
					n := int(2 + d*14)
					for j := 0; j <= n; j++ {
						t := float64(j)/16*2.2 - 1.1
						r := 15 * z
						qx := float64(px) + s*math.Sin(t)*r*.9 + s*float64(i)*2
						qy := float64(chest) + math.Cos(t)*r*.35*s - t*r*.55
						if j == 0 {
							p.MoveTo(float32(qx), float32(qy))
						} else {
							p.LineTo(float32(qx), float32(qy))
						}
					}
					op := *lighter
					op.ColorScale.ScaleWithColor(with8(hot, 1-k))
					strokePath(dst, &p, vector.StrokeOptions{Width: 2.4 * zf, LineCap: vector.LineCapRound}, &op)
				}
			}
			glow(dst, px, chest, float32(6+k*18)*zf, hot, 1-k)
		case "fire":
			glow(dst, px, py-6*zf, float32(10+k*8)*zf, flameC[1], (1-k)*1.2)
			ellipse(dst, px, py, float32(8+k*12)*zf, 2*zf, flameC[2], (1-k)*.9)
		case "water":
			ellipse(dst, px, py, float32(4+k*22)*zf, 2.5*zf, waterC[1], 1-k)
			ellipse(dst, px, py, float32(2+k*14)*zf, 1.5*zf, waterC[0], (1-k)*.8)
			if k < .5 {
				var p vector.Path // a rising water column
				h := float32(math.Sin(k/.5*math.Pi)) * 26 * zf
				p.MoveTo(px-5*zf, py)
				p.LineTo(px-2*zf, py-h)
				p.LineTo(px+2*zf, py-h)
				p.LineTo(px+5*zf, py)
				op := *lighter
				op.ColorScale.ScaleWithColor(with8(waterC[1], .6))
				fillPath(dst, &p, &op)
			}
		case "heal", "buff":
			tint := e.tint
			if e.kind == "heal" {
				tint = healC
			}
			ring := float32(10+6*math.Sin(k*math.Pi)) * zf
			ellipse(dst, px, py, ring, 2.2*zf, tint, math.Min(1, 1.6*math.Sin(k*math.Pi)))
			ellipse(dst, px, py, ring*.65, zf, hot, math.Sin(k*math.Pi)*.6)
			// light pillar
			h := float32(e.height*1.1*z) * float32(math.Min(1, k*3))
			for i, w := range []float32{9, 5, 2} {
				op := *lighter
				var p vector.Path
				p.MoveTo(px-w*zf, py)
				p.LineTo(px-w*zf*.6, py-h)
				p.LineTo(px+w*zf*.6, py-h)
				p.LineTo(px+w*zf, py)
				op.ColorScale.ScaleWithColor(with8(tint, (1-k*k)*[]float64{.3, .4, .6}[i]))
				fillPath(dst, &p, &op)
			}
		case "debuff":
			for i := 0; i < 6; i++ {
				a := k*9 + float64(i)*math.Pi/3
				r := (12 - 4*k) * z
				qx := float64(px) + math.Cos(a)*r
				qy := float64(chest) - 6*z + math.Sin(a)*r*.4
				glow(dst, float32(qx), float32(qy), 3*zf, debuffC, 1-k)
			}
			ellipse(dst, px, py, float32(14-6*k)*zf, 1.5*zf, debuffC, (1-k)*.8)
		case "block":
			g.barrier(dst, e, px, chest, k)
		case "critical": // a white burst, rays and a shockwave on the ground
			if k < .35 {
				glow(dst, px, chest, float32(12+k*90)*zf, hot, 1.4*(1-k/.35))
			}
			for i := 0; i < 8; i++ {
				a := float64(i)*math.Pi/4 + .3
				r0, r1 := (6+k*40)*z, (16+k*70)*z
				stroke(dst, px+float32(math.Cos(a)*r0), chest+float32(math.Sin(a)*r0*.7), px+float32(math.Cos(a)*r1),
					chest+float32(math.Sin(a)*r1*.7), 2.2*zf, hot, 1-k)
			}
			ellipse(dst, px, py, float32(6+k*44)*zf, 3*zf, spark, 1-k)
			ellipse(dst, px, py, float32(3+k*28)*zf, 1.5*zf, hot, (1-k)*.8)
		}
		for _, q := range e.parts {
			if e.t > q.life {
				continue
			}
			a := 1 - e.t/q.life
			qx := float32(float64(px) + (q.x+q.vx*e.t)*z)
			qy := float32(float64(py) + (q.y+q.vy*e.t+.5*q.g*e.t*e.t)*z)
			s := float32(q.size) * zf
			switch q.shape {
			case 0:
				glow(dst, qx, qy, s*1.6, q.col, a)
			case 1:
				vx, vy := q.vx, q.vy+q.g*e.t
				n := math.Hypot(vx, vy)
				if n > 0 {
					l := 4 * z
					stroke(dst, qx, qy, qx-float32(vx/n*l), qy-float32(vy/n*l), s, q.col, a)
				}
			case 2:
				stroke(dst, qx-s, qy, qx+s, qy, s*.7, q.col, a)
				stroke(dst, qx, qy-s, qx, qy+s, s*.7, q.col, a)
			case 3, 4:
				d := float32(1)
				if q.shape == 4 {
					d = -1
				}
				stroke(dst, qx, qy+d*s*1.5, qx, qy-d*s*1.5, s*.6, q.col, a)
				stroke(dst, qx-s, qy-d*s*.4, qx, qy-d*s*1.5, s*.6, q.col, a)
				stroke(dst, qx+s, qy-d*s*.4, qx, qy-d*s*1.5, s*.6, q.col, a)
			}
		}
	}
}

var barrierC = color.NRGBA{R: 130, G: 210, B: 255, A: 255}

// barrier is the guard that stops a blow: a hex-patterned shield pops up on the side
// facing the attacker ("팡"), flashes at the contact point, ripples and flickers out.
func (g *Game) barrier(dst *ebiten.Image, e *effect, px, chest float32, k float64) {
	fx, _ := g.field.Center(e.fu, e.fv)
	tx, _ := g.field.Center(e.tu, e.tv)
	side := float32(-1)
	if fx > tx {
		side = 1
	}
	zf := float32(g.zoom)
	pop := 1.0 // scale: overshoots, then settles
	switch {
	case k < .12:
		pop = .6 + k/.12*.5
	case k < .22:
		pop = 1.1 - (k-.12)/.1*.1
	}
	fade := 1.0
	if k > .45 {
		fade = 1 - (k-.45)/.55
		if int(k*40)%2 == 0 { // flicker while fading
			fade *= .55
		}
	}
	cx := px + side*10*zf
	rx, ry := float32(7*pop)*zf, float32(15*pop)*zf
	// shield body: a curved pane, brighter at its rim
	var body vector.Path
	for i := 0; i <= 24; i++ {
		t := float64(i) / 24 * 2 * math.Pi
		x, y := cx+rx*float32(math.Cos(t)), chest+ry*float32(math.Sin(t))
		if i == 0 {
			body.MoveTo(x, y)
		} else {
			body.LineTo(x, y)
		}
	}
	op := *lighter
	op.ColorScale.ScaleWithColor(with8(barrierC, .28*fade))
	fillPath(dst, &body, &op)
	op = *lighter
	op.ColorScale.ScaleWithColor(with8(color.NRGBA{220, 245, 255, 255}, .9*fade))
	strokePath(dst, &body, vector.StrokeOptions{Width: 1.5 * zf}, &op)
	// hex cells inside the pane
	hr := 3.2 * zf * float32(pop)
	for _, c := range [][2]float32{{0, 0}, {0, -1.75}, {0, 1.75}, {-1.1, -.87}, {-1.1, .87}, {1.1, -.87}, {1.1, .87}} {
		hx, hy := cx+c[0]*hr*.62, chest+c[1]*hr
		var h vector.Path
		for i := 0; i <= 6; i++ {
			a := float64(i) * math.Pi / 3
			x, y := hx+hr*.55*float32(math.Cos(a)), hy+hr*.9*float32(math.Sin(a))
			if i == 0 {
				h.MoveTo(x, y)
			} else {
				h.LineTo(x, y)
			}
		}
		op := *lighter
		op.ColorScale.ScaleWithColor(with8(barrierC, .5*fade))
		strokePath(dst, &h, vector.StrokeOptions{Width: .8 * zf}, &op)
	}
	// ripple from the contact point and the flash itself
	contact := cx + side*rx*.4
	for i, d := range []float64{0, .12} {
		if r := k - d; r > 0 && r < .5 {
			ellipse(dst, contact, chest, float32(r/.5*20)*zf, 1.4*zf, color.NRGBA{200, 240, 255, 255}, (1-r/.5)*[]float64{.9, .6}[i])
		}
	}
	if k < .3 {
		glow(dst, contact, chest, float32(4+k*30)*zf, hot, 1-k/.3)
	}
}
