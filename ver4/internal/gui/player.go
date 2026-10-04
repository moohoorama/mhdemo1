package gui

import (
	"fmt"
	"image/color"
	"math"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
)

const (
	stepTime  = .28 // seconds per cell while walking
	stagger   = .2  // minimum delay between two actions starting
	shakeTime = .45
	shakePx   = 3.0
	shakeHz   = 11.0
	blinkTime = 1.2
)

var directions = []string{"E", "SE", "S", "SW", "W", "NW", "N", "NE"} // by screen angle from +x

// facing is the screen direction of a map step: the iso projection turns (du, dv) into screen x/y.
func facing(du, dv float64) string {
	sx, sy := (du-dv)*16, (du+dv)*8
	a := math.Mod(math.Atan2(sy, sx)*180/math.Pi+360, 360)
	return directions[int((a+22.5)/45)%8]
}

// unitVis is what the screen shows of a unit. The rules state is already final when a
// replay starts; these fields catch up action by action.
type unitVis struct {
	id, faction  string
	ally         bool
	art          *UnitArt
	u, v, lift   float64
	dir, anim    string
	animT        float64
	walk         *walkState
	shakeSign    float64
	shakeT       float64
	dying        float64 // < 0: alive
	gone, greyed bool
	ranged       bool
	shownHP      int
	maxHP        int
	flash        float64
	top          float64
}
type walkState struct {
	fu, fv, tu, tv, fl, tl, t float64
}

func (u *unitVis) frame() int { return u.art.FrameIndex(u.anim, u.animT*1000) }
func (u *unitVis) play(anim string) {
	u.anim, u.animT = anim, 0
}
func (u *unitVis) rest() string {
	if u.shownHP*2 <= u.maxHP {
		return "exhausted"
	}
	return "idle"
}
func (u *unitVis) face(tu, tv float64) {
	if tu != u.u || tv != u.v {
		u.dir = facing(tu-u.u, tv-u.v)
	}
}
func (u *unitVis) visible() bool {
	if u.dying >= 0 {
		return int(u.dying*12)%2 == 0
	}
	return true
}
func (u *unitVis) shakeOffset() float64 {
	if u.shakeT <= 0 {
		return 0
	}
	t := shakeTime - u.shakeT
	return u.shakeSign * shakePx * math.Sin(2*math.Pi*shakeHz*t) * (u.shakeT / shakeTime)
}
func (u *unitVis) update(dt float64) {
	u.animT += dt
	if w := u.walk; w != nil {
		w.t = math.Min(stepTime, w.t+dt)
		k := w.t / stepTime
		u.u, u.v = w.fu+(w.tu-w.fu)*k, w.fv+(w.tv-w.fv)*k
		u.lift = w.fl + (w.tl-w.fl)*k
		if w.t >= stepTime {
			u.walk = nil
		}
	}
	a := u.art.Animations[u.anim]
	if !a.Loop && u.animT >= u.art.Length(u.anim) && u.dying < 0 {
		u.play(u.rest())
	}
	u.shakeT = math.Max(0, u.shakeT-dt)
	u.flash = math.Max(0, u.flash-dt*4)
	if u.dying >= 0 {
		u.dying += dt
		u.gone = u.dying >= blinkTime
	}
}

// popup is floating text over a unit (damage, healing, names of skills).
type popup struct {
	unit *unitVis
	text string
	col  color.RGBA
	t    float64
}

type beat struct {
	dur float64
	fn  func()
}
type running struct {
	index int
	beats []beat
	i     int
	left  float64
}

// Player shows resolved actions with claim replay: an action starts as soon as every
// action it waits for is done (and at least `stagger` after the previous start).
type Player struct {
	g       *Game
	actions []*replay.Action
	done    []bool
	waiting []int
	running []*running
	gap     float64
	final   core.Observation
	Duel    *core.Event
}

func (p *Player) Busy() bool { return len(p.waiting)+len(p.running) > 0 }

// Play queues actions; final is the observation after all of them (synced at the end).
func (p *Player) Play(actions []*replay.Action, final core.Observation) {
	p.actions, p.final = actions, final
	p.done = make([]bool, len(actions))
	p.waiting = p.waiting[:0]
	p.running = p.running[:0]
	for i := range actions {
		p.waiting = append(p.waiting, i)
	}
	p.gap = stagger
	if len(actions) == 0 {
		p.g.sync(final)
	}
}

func (p *Player) Update(dt float64) {
	if !p.Busy() {
		return
	}
	p.gap += dt
	for k, i := range p.waiting {
		ready := true
		for _, j := range p.actions[i].After {
			ready = ready && p.done[j]
		}
		if ready && p.gap >= stagger {
			p.waiting = append(p.waiting[:k], p.waiting[k+1:]...)
			r := &running{index: i, beats: p.g.perform(p.actions[i]), i: -1}
			p.running = append(p.running, r)
			p.gap = 0
			break
		}
	}
	for _, r := range p.running {
		r.left -= dt
		for r.left <= 0 && r.i < len(r.beats) {
			r.i++
			if r.i < len(r.beats) {
				r.beats[r.i].fn()
				r.left += r.beats[r.i].dur
			}
		}
	}
	kept := p.running[:0]
	for _, r := range p.running {
		if r.i >= len(r.beats) {
			p.done[r.index] = true
		} else {
			kept = append(kept, r)
		}
	}
	p.running = kept
	if !p.Busy() {
		p.Duel = nil
		p.g.sync(p.final)
	}
}

// Skip shows the final state at once; the outcome is already decided.
func (p *Player) Skip() {
	if !p.Busy() {
		return
	}
	p.waiting, p.running, p.Duel = nil, nil, nil
	p.g.popups = nil
	p.g.sync(p.final)
}

func (g *Game) unitAt(id string) *unitVis { return g.vis[id] }

// perform turns one action into timed beats; each beat's function runs when it starts.
func (g *Game) perform(a *replay.Action) []beat {
	beats := []beat{}
	add := func(d float64, f func()) { beats = append(beats, beat{d, f}) }
	actor := g.unitAt(a.Actor)
	for _, s := range a.Steps {
		c, events := s.Command, s.Events
		switch c.Kind {
		case "move":
			for _, e := range events {
				if e.Kind != "move" || actor == nil {
					continue
				}
				for _, p := range e.Path[1:] {
					cell := p
					add(stepTime, func() { g.stepTo(actor, cell) })
				}
				add(0, func() { actor.play(actor.rest()) })
			}
		case "attack", "skill":
			target := g.unitAt(c.Target)
			if actor == nil {
				continue
			}
			ms := actor.art.Animations["attack"].MS
			gather := float64(ms[0]+ms[1]) / 1000
			follow := float64(ms[2]+ms[3])/1000 + .25
			kind := g.effectKind(actor, c)
			add(gather, func() {
				if target != nil {
					actor.face(target.u, target.v)
				}
				actor.play("attack")
				if c.Kind == "skill" {
					g.say(actor, c.Skill, color.RGBA{180, 220, 255, 255})
				}
				if kind == "arrow" && target != nil && target != actor {
					g.spawn("arrow", actor, target, gather)
				}
			})
			wait := follow
			for _, e := range events {
				if e.Kind == "retreat" {
					wait = math.Max(wait, blinkTime)
				} else if e.Kind == "damage" {
					wait = math.Max(wait, shakeTime)
				}
			}
			ev := events
			add(wait, func() { g.impact(kind, actor, ev); g.strike(actor, ev) })
		case "item":
			ev := events
			add(.6, func() {
				for _, e := range ev {
					if e.Kind == "item" {
						if t := g.unitAt(e.Target); t != nil {
							g.spawn("heal", nil, t, .6)
							g.say(t, fmt.Sprintf("+%d", e.Amount), color.RGBA{135, 245, 144, 255})
						}
					}
				}
			})
		case "learn":
			add(.6, func() { g.say(actor, "학습: "+c.Trait, color.RGBA{255, 224, 140, 255}) })
		case "duel":
			for _, e := range events {
				if e.Kind == "duel" {
					duel := e
					add(2.6, func() { g.player.Duel = &duel })
				}
			}
			ev := events
			add(blinkTime, func() { g.player.Duel = nil; g.strike(nil, ev) })
		case "end":
			ev := events
			add(0, func() { g.strike(nil, ev) })
			for _, e := range events {
				if e.Kind == "turn" {
					turn := e
					add(1.0, func() {
						g.banner = fmt.Sprintf("%d턴 · %s 진영", turn.Amount, factionName(turn.Target))
						g.bannerT = 1.4
					})
				}
			}
		default:
			add(.1, func() {})
		}
		for _, e := range events {
			if e.Kind == "victory" {
				add(1.2, func() { g.banner, g.bannerT = "승리", 2 })
			}
		}
	}
	if len(beats) == 0 {
		add(.1, func() {})
	}
	return beats
}

func (g *Game) stepTo(u *unitVis, p content.Point) {
	tu, tv := float64(p.X), float64(p.Y)
	u.face(tu, tv)
	u.walk = &walkState{fu: u.u, fv: u.v, tu: tu, tv: tv, fl: u.lift, tl: g.field.Lift(p.X, p.Y)}
	if u.anim != "walk" {
		u.play("walk")
	}
}

func (g *Game) say(u *unitVis, text string, c color.RGBA) {
	if u != nil {
		g.popups = append(g.popups, &popup{unit: u, text: text, col: c})
	}
}

// strike shows the outcome events of a blow: damage with a hit reaction that shakes away
// from the attacker, misses, healing and effects, retreats blinking out.
func (g *Game) strike(attacker *unitVis, events []core.Event) {
	for _, e := range events {
		t := g.unitAt(e.Target)
		switch e.Kind {
		case "damage":
			if t == nil {
				continue
			}
			t.shownHP = max(0, t.shownHP-e.Amount)
			t.play("hit")
			t.flash = .25
			src := g.unitAt(e.Actor)
			if src == nil {
				src = attacker
			}
			t.shakeSign = 1
			if src != nil {
				sx, _ := g.field.Center(src.u, src.v)
				tx, _ := g.field.Center(t.u, t.v)
				if sx > tx {
					t.shakeSign = -1
				}
			}
			t.shakeT = shakeTime
			g.say(t, fmt.Sprintf("-%d", e.Amount), color.RGBA{255, 236, 160, 255})
		case "miss":
			g.say(t, "빗나감", color.RGBA{200, 210, 220, 255})
		case "heal":
			if t != nil {
				t.shownHP = min(t.maxHP, t.shownHP+e.Amount)
			}
			g.say(t, fmt.Sprintf("+%d", e.Amount), color.RGBA{135, 245, 144, 255})
		case "effect":
			if t != nil {
				t.flash = .3
			}
		case "retreat":
			if u := g.unitAt(e.Actor); u != nil {
				u.shownHP = 0
				u.dying = 0
			}
		}
	}
}
