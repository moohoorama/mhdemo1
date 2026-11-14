package gui

import (
	"image/color"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// effect is a short animation from the fx sheet on the field. Positions are canvas pixels
// relative to the target's feet (fx, fy for the source) and scale with the camera zoom like
// the units do.
type effect struct {
	kind   string // arrow, slash, fire, water, heal, buff, debuff, block, spell_<skill>
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
	// fireArrowC tints 화시's arrow; fireArrowGlow its trail of embers
	fireArrowC    = color.NRGBA{R: 255, G: 96, B: 48, A: 255}
	fireArrowGlow = color.NRGBA{R: 255, G: 170, B: 70, A: 255}
)

// buffTint is the color of a support effect.
var buffTint = map[string]color.NRGBA{
	"speed": {90, 220, 255, 255}, "counter": {210, 220, 240, 255},
	"range": {150, 235, 120, 255}, "morale": {255, 210, 90, 255}, "attack": {255, 110, 80, 255},
	"defense": {100, 160, 255, 255},
}

// A spell with a baked look (assets/graphics/effects/spells/spell_<skill>, made by tools/spellsim.py) plays it as one
// effect over its whole area: drawn on the target cell, or for 폭염 from the first cell of its line.
// It starts as the cast is released and the damage waits for its Hit (a rock landing, a dragon diving).
const spellPrefix = "spell_"

var (
	// spellShake is how long a spell's blow shakes the screen.
	spellShake = map[string]float64{
		"spell_tsunami": .3, "spell_blaze": .22, "spell_inferno": .3, "spell_fire_dragon": .3,
		"spell_water_dragon": .3, "spell_rockfall": .22, "spell_landslide": .35, "spell_earthquake": .6,
		"spell_boulder": .5, "spell_earth_dragon": .35,
	}
	// spellFromSide are drawn coming in from up-left; mirrored when the caster stands to the right.
	spellFromSide = map[string]bool{"spell_tsunami": true, "spell_landslide": true}
	// spellOnGround lie flat on the ground under the units (팔진도's formation) instead of over them.
	spellOnGround = map[string]bool{"spell_eightfold": true}
)

func isSpell(kind string) bool { return strings.HasPrefix(kind, spellPrefix) }

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
	if _, ok := g.assets.lookupEffect(spellPrefix + c.Skill); ok {
		return spellPrefix + c.Skill
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
	spell := isSpell(kind)
	if spell {
		g.shakeT = math.Max(g.shakeT, spellShake[kind])
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
			if spell { // its look covers the whole area and is already playing
				g.spellPiece(kind, actor, t)
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
	g.groundSpells()
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
		case "arrow", "burning_shot": // a flat shot from the shooter's chest to the target's; 화시's burns red and sheds embers
			fx, fy := g.field.Center(e.fu, e.fv)
			ax, ay := fx, fy-e.flift-e.fh*.55
			bx, by := x, chest
			sh := g.assets.effect("arrow")
			n := float64(len(sh.Frames))
			i := int(math.Round(math.Atan2(by-ay, bx-ax)/(2*math.Pi)*n+n)) % len(sh.Frames)
			if e.kind == "arrow" {
				sh.draw(dst, i, ax+(bx-ax)*k, ay+(by-ay)*k, g.toScreen, z)
				break
			}
			ember := g.assets.effect("ember")
			for j := 1; j <= 5; j++ { // the trail, fading behind the head
				t := k - float64(j)*.045
				if t < 0 {
					break
				}
				ember.drawTint(dst, j%len(ember.Frames), ax+(bx-ax)*t, ay+(by-ay)*t+float64(j)*.6, g.toScreen, z, fireArrowGlow, 1-float64(j)/6)
			}
			sh.drawTint(dst, i, ax+(bx-ax)*k, ay+(by-ay)*k, g.toScreen, z, e.tint, 1)
		case "slash":
			sh := g.assets.effect("slash")
			sh.draw(dst, sh.frameOver(k), x, chest, g.toScreen, z)
		case "fire", "water", "heal", "debuff":
			sh := g.assets.effect(e.kind)
			sh.draw(dst, sh.frameOver(k), x, y-e.lift, g.toScreen, z)
		case "buff":
			sh := g.assets.effect("buff")
			sh.drawTint(dst, sh.frameOver(k), x, y-e.lift, g.toScreen, z, e.tint, 1)
		case "block": // the shield faces the blow
			fx, _ := g.field.Center(e.fu, e.fv)
			sh := g.assets.effect("block")
			sh.blit(dst, sh.frameOver(k), x, chest, g.toScreen, z, white, 1, fx < x)
		default:
			if isSpell(e.kind) {
				g.drawSpell(dst, e, x, y)
			}
		case "critical": // a white burst and rays at the chest, a shockwave on the ground
			sh, wave := g.assets.effect("critical"), g.assets.effect("critical_wave")
			wave.draw(dst, wave.frameOver(k), x, y-e.lift, g.toScreen, z)
			sh.draw(dst, sh.frameOver(k), x, chest, g.toScreen, z)
		}
		for _, q := range e.parts {
			if e.t > q.life {
				continue
			}
			sh := g.assets.effect(q.sprite)
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

// spellPiece puts a spell's per-unit piece on a unit it reaches, when the spell has one: <look>_ally on the
// caster's side, <look>_enemy on the other (팔진도 strengthens one and weakens the other).
func (g *Game) spellPiece(kind string, actor, t *unitVis) {
	if t == nil || actor == nil {
		return
	}
	piece := kind + "_enemy"
	if t.ally == actor.ally {
		piece = kind + "_ally"
	}
	if sh, ok := g.assets.lookupEffect(piece); ok {
		g.spawn(piece, actor, t, sh.length(), white)
	}
}

// groundSpells lays the spell looks that lie on the ground under the units, refreshed every tick; it leaves the
// field's ground marks alone when it has none to lay and laid none before (the town walk's click ring uses them).
func (g *Game) groundSpells() {
	var marks []groundMark
	for _, e := range g.effects {
		if !spellOnGround[e.kind] {
			continue
		}
		sh := g.assets.effect(e.kind)
		i := sh.frameAt(e.t)
		if i < 0 {
			continue
		}
		x, y := g.field.Center(e.tu, e.tv)
		an := sh.anchor(i)
		marks = append(marks, groundMark{Sprite{sh.Frames[i], an[0], an[1]}, x, y - e.lift})
	}
	if len(marks) > 0 || g.groundFX {
		g.field.groundMarks = marks
		g.groundFX = len(marks) > 0
	}
}

// drawSpell draws a spell's baked look at its time. A line (폭염) is drawn from the first cell in front
// of the caster: the sprite runs down-right, its _back twin up-left, and mirroring turns those into the
// other two directions. A look that comes in from the side is mirrored to come from the caster's side.
func (g *Game) drawSpell(dst *ebiten.Image, e *effect, x, y float64) {
	kind, mirror := e.kind, false
	if spellOnGround[kind] {
		return
	}
	if kind == "spell_inferno" {
		du, dv := e.tu-e.fu, e.tv-e.fv
		su, sv := 0.0, 0.0
		if math.Abs(du) >= math.Abs(dv) {
			su = math.Copysign(1, du)
		} else {
			sv = math.Copysign(1, dv)
		}
		x, y = g.field.Center(e.fu+su, e.fv+sv)
		if su < 0 || sv < 0 {
			kind = "spell_inferno_back"
		}
		mirror = sv != 0
	} else if spellFromSide[kind] {
		mirror = e.flip(g)
	}
	sh, ok := g.assets.lookupEffect(kind)
	if !ok {
		return
	}
	i := sh.frameAt(e.t)
	if i < 0 {
		return
	}
	sh.blit(dst, i, x, y-e.lift, g.toScreen, g.zoom, white, 1, mirror)
}

// flip says whether a look drawn coming from up-left is mirrored: its caster stands to the right.
func (e *effect) flip(g *Game) bool {
	fx, _ := g.field.Center(e.fu, e.fv)
	tx, _ := g.field.Center(e.tu, e.tv)
	return fx > tx
}
