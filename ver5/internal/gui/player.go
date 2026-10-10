package gui

import (
	"fmt"
	"math"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
	"srpg/internal/sprite"
)

const (
	stepTime        = .28 // seconds per cell while walking
	stagger         = .2  // minimum delay between two actions starting
	shakeTime       = .45
	shakePx         = 3.0
	critShake       = 2.8 // a critical blow shakes the target this many times harder
	chargeTime      = .75 // seconds a unit gathers itself, whitening, before a critical blow
	shakeHz         = 11.0
	blinkTime       = 1.2
	arrowSecPerCell = .07 // an arrow's flight time per cell of distance
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
	id           string
	ally         bool
	art          *sprite.Art
	u, v, lift   float64
	dir, anim    string
	animT        float64
	walk         *walkState
	shakeSign    float64
	shakeT       float64
	shakeAmp     float64
	charge       float64 // 0..1: how white the sprite is drawn while gathering a critical blow
	charging     bool    // holds the first attack frame and whitens
	dying        float64 // < 0: alive
	gone, greyed bool
	ranged       bool
	inactive     bool
	actor        bool // a scene actor: no HP bar, always rests standing
	shownHP      int
	maxHP        int
	flash        float64
	top          float64
}
type walkState struct {
	fu, fv, tu, tv, fl, tl, t float64
}

func (u *unitVis) frozenRest() bool {
	return !u.actor && u.inactive && (u.anim == "idle" || u.anim == "exhausted")
}

func (u *unitVis) frame() int {
	if u.frozenRest() {
		return 0
	}
	return u.art.FrameIndex(u.anim, u.animT*1000)
}
func (u *unitVis) play(anim string) {
	if _, ok := u.art.Animations[anim]; !ok {
		anim = "idle"
	}
	u.anim, u.animT = anim, 0
}
func (u *unitVis) rest() string {
	if !u.actor && u.shownHP*2 <= u.maxHP {
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
	return u.shakeSign * shakePx * max(1, u.shakeAmp) * math.Sin(2*math.Pi*shakeHz*t) * min(1, u.shakeT/shakeTime)
}
func (u *unitVis) update(dt float64) {
	if u.charging {
		u.charge = math.Min(1, u.charge+dt/chargeTime)
	} else {
		if u.frozenRest() {
			u.animT = 0
		} else {
			u.animT += dt
		}
		u.charge = math.Max(0, u.charge-dt*5)
	}
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
	t    float64
	row  int // stacked above the unit's other fresh popups
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
			id := p.actions[r.index].Actor
			if v, u := p.g.unitAt(id), p.g.unitView(p.final, id); v != nil && u != nil {
				v.inactive = p.final.Phase != "battle" || u.Faction != p.final.Turn || u.Done
			}
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
		kind := c.Kind
		for _, e := range events {
			if e.Kind == "duel" { // an attack that set off a stage duel shows the duel instead
				kind = "duel"
			}
		}
		switch kind {
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
			blow, counters := splitCounters(c.Actor, events)
			for i, part := range splitDouble(blow) {
				crit := false
				for _, e := range part {
					crit = crit || e.Kind == "critical"
				}
				if i > 0 {
					add(.15, func() {})
				}
				g.swing(add, actor, target, c, part, crit)
			}
			for _, k := range counters {
				back := g.unitAt(k.by)
				if back == nil {
					continue
				}
				add(.15, func() {})
				g.swing(add, back, actor, core.Command{Kind: "attack"}, k.events, false)
			}
		case "item":
			ev := events
			add(.6, func() {
				for _, e := range ev {
					t := g.unitAt(e.Target)
					if e.Kind != "item" || t == nil {
						continue
					}
					g.spawn("heal", nil, t, 1.1, healC)
					if g.S.Data.Items[c.Item].Effect == content.ItemMP {
						g.say(t, fmt.Sprintf("MP +%d", e.Amount))
					} else {
						t.shownHP = min(t.maxHP, t.shownHP+e.Amount)
						g.say(t, fmt.Sprintf("HP +%d", e.Amount))
					}
				}
			})
		case "learn":
			add(.6, func() { g.say(actor, "학습: "+g.traitName(c.Trait)) })
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

// counter is a counter-attack inside an action's events: who strikes back and what it does.
type counter struct {
	by     string
	events []core.Event
}

// splitCounters separates the counter-attacks from an attack's events. A counter is a
// blow (critical, damage, miss) by someone other than the actor, with the actor's retreat
// it may cause; each one is shown as its own swing after the attack lands.
func splitCounters(actor string, events []core.Event) ([]core.Event, []counter) {
	blow, counters := []core.Event{}, []counter{}
	for _, e := range events {
		back := e.Actor != actor && e.Actor != "" && (e.Kind == "critical" || e.Kind == "damage" || e.Kind == "miss")
		switch {
		case back && (len(counters) == 0 || counters[len(counters)-1].by != e.Actor):
			counters = append(counters, counter{by: e.Actor, events: []core.Event{e}})
		case back || e.Kind == "retreat" && e.Actor == actor && len(counters) > 0:
			k := &counters[len(counters)-1]
			k.events = append(k.events, e)
		default:
			blow = append(blow, e)
		}
	}
	return blow, counters
}

// splitDouble separates an attack's second blows, which follow its first "double" event,
// so they are shown as a second swing.
func splitDouble(events []core.Event) [][]core.Event {
	for i, e := range events {
		if e.Kind == "double" {
			return [][]core.Event{events[:i], events[i:]}
		}
	}
	return [][]core.Event{events}
}

// swing adds the beats of one blow: the attack motion (preceded, for a critical blow, by
// holding its first frame while the unit glows white) and then the impact.
func (g *Game) swing(add func(float64, func()), actor, target *unitVis, c core.Command, events []core.Event, crit bool) {
	ms := actor.art.Animations["attack"].MS
	gather := float64(ms[0]+ms[1]) / 1000
	follow := float64(ms[2]+ms[3])/1000 + .25
	kind := g.effectKind(actor, c)
	tint := buffTint[g.S.Data.Skills[c.Skill].Effect]
	start := func() {
		if target != nil && target != actor {
			actor.face(target.u, target.v)
		}
		actor.play("attack")
		if c.Kind == "skill" {
			g.say(actor, g.skillName(c.Skill))
		}
		for _, e := range events {
			if e.Kind == "cost" {
				g.say(actor, fmt.Sprintf("MP -%d", e.Amount))
			}
		}
	}
	if crit {
		add(chargeTime, func() {
			start()
			actor.charging = true
		})
	}
	add(gather, func() {
		if crit {
			actor.charging = false
		} else {
			start()
		}
	})
	if kind == "arrow" && target != nil && target != actor {
		// released on the third frame, it lands after its flight
		flight := math.Max(.12, math.Hypot(target.u-actor.u, target.v-actor.v)*arrowSecPerCell)
		add(flight, func() { g.spawn("arrow", actor, target, flight, spark) })
	}
	wait := follow
	for _, e := range events {
		if e.Kind == "retreat" {
			wait = math.Max(wait, blinkTime)
		} else if e.Kind == "damage" {
			wait = math.Max(wait, shakeTime)
		}
	}
	if crit {
		wait = math.Max(wait, shakeTime*1.6)
	}
	add(wait, func() { g.impact(kind, actor, events, tint); g.strike(actor, events) })
}

func (g *Game) stepTo(u *unitVis, p content.Point) { g.walkTo(u, float64(p.X), float64(p.Y)) }

// walkTo starts one walking step to a (fractional) cell position.
func (g *Game) walkTo(u *unitVis, tu, tv float64) {
	u.face(tu, tv)
	u.walk = &walkState{fu: u.u, fv: u.v, tu: tu, tv: tv, fl: u.lift,
		tl: g.field.Lift(int(math.Round(tu)), int(math.Round(tv)))}
	if u.anim != "walk" {
		u.play("walk")
	}
}

func (g *Game) say(u *unitVis, text string) {
	if u == nil {
		return
	}
	row := 0
	for _, p := range g.popups {
		if p.unit == u && p.t < .45 {
			row++
		}
	}
	g.popups = append(g.popups, &popup{unit: u, text: text, row: row})
}

// expShown converts raw experience to the 0..100 scale the experience bar uses.
func (g *Game) expShown(id string, raw int) int {
	if u := g.unitView(g.S.Engine.Observe(), id); u != nil {
		return max(1, int(math.Round(float64(raw)*100/float64(max(1, u.XPRequired)))))
	}
	return raw
}

// strike shows the outcome events of a blow: damage with a hit reaction that shakes away
// from the attacker, misses, healing and effects, retreats blinking out.
func (g *Game) strike(attacker *unitVis, events []core.Event) {
	crit := map[string]bool{}
	for _, e := range events {
		if e.Kind == "critical" {
			crit[e.Target] = true
		}
	}
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
			t.shakeT, t.shakeAmp = shakeTime, 1
			if crit[e.Target] {
				t.shakeT, t.shakeAmp, t.flash = shakeTime*1.6, critShake, .5
				g.say(t, fmt.Sprintf("치명타! HP -%d", e.Amount))
				continue
			}
			g.say(t, fmt.Sprintf("HP -%d", e.Amount))
		case "miss": // the target blocks: its block motion when the sprite has one, else a flinch
			if t != nil {
				if _, ok := t.art.Animations["block"]; ok {
					t.play("block")
				} else {
					t.shakeSign, t.shakeT = 1, shakeTime*.6
				}
				if src := g.unitAt(e.Actor); src != nil {
					t.face(src.u, src.v)
				}
			}
			g.say(t, "막음!")
		case "heal":
			if t != nil {
				t.shownHP = min(t.maxHP, t.shownHP+e.Amount)
			}
			g.say(t, fmt.Sprintf("HP +%d", e.Amount))
		case "recover-hp":
			if u := g.unitAt(e.Actor); u != nil {
				u.shownHP = min(u.maxHP, u.shownHP+e.Amount)
				g.say(u, fmt.Sprintf("HP +%d", e.Amount))
			}
		case "recover-mp":
			g.say(g.unitAt(e.Actor), fmt.Sprintf("MP +%d", e.Amount))
		case "experience":
			g.say(g.unitAt(e.Actor), fmt.Sprintf("EXP +%d", g.expShown(e.Actor, e.Amount)))
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
