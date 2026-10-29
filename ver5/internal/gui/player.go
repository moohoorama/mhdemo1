package gui

import (
	"fmt"
	"image"
	"math"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/duelview"
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
	leaveTime       = 1.0 // seconds a unit takes to walk off the battle area when it leaves by flight
	leaveCells      = 1.5 // how far past its cell it walks out
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
	facing       string // the facing the rules state last gave; a change turns the unit to it
	animT        float64
	walk         *walkState
	shakeSign    float64
	shakeT       float64
	shakeAmp     float64
	charge       float64    // 0..1: how white the sprite is drawn while gathering a critical blow
	charging     bool       // holds the first attack frame and whitens
	dying        float64    // < 0: alive
	leave        *walkState // walking off the battle area, fading out; t runs to leaveTime
	arriving     float64    // a scene actor walking on: seconds left of fading in, from leaveTime
	gone, greyed bool
	ranged       bool
	shoot        *sprite.Art // the look it shoots from range in (mounted_range), nil when it has none
	shootClass   string
	siege        bool    // attacks are stones lobbed along an arc
	mounted      bool    // rides a horse: takes the long boat on water
	boatAxis     string  // "u" or "v": how its boat lies, kept while it faces straight N, E, S or W
	wade         float64 // 0..1: how far it has sunk into a ford it stands on (field_wade.go)
	inactive     bool
	actor        bool   // a scene actor: no HP bar, rests standing or in its pose
	pose         string // a scene actor's resting motion (sit, lie …); "" stands
	shownHP      int
	shownMP      int
	maxMP        int
	maxHP        int
	flash        float64
	top          float64
	status       map[string]core.Status
	bar          barState
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
	if _, ok := u.art.Animations[u.pose]; ok && u.actor {
		return u.pose
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
	u.bar.update(float64(u.shownHP)/float64(max(1, u.maxHP)), dt)
	if u.dying >= 0 {
		u.dying += dt
		u.gone = u.dying >= blinkTime
	}
	u.arriving = math.Max(0, u.arriving-dt)
	if l := u.leave; l != nil && !u.gone {
		l.t = math.Min(leaveTime, l.t+dt)
		k := l.t / leaveTime
		u.u, u.v = l.fu+(l.tu-l.fu)*k, l.fv+(l.tv-l.fv)*k
		u.gone = l.t >= leaveTime
	}
}

// leaving is how far u has walked off the battle area, 0..1, or how much of its walk on a scene
// actor still has to fade in; 0 when it is neither leaving nor arriving.
func (u *unitVis) leaving() float64 {
	if u.leave == nil {
		return u.arriving / leaveTime
	}
	return u.leave.t / leaveTime
}

// popup is floating text over a unit (damage, healing, names of skills).
type popup struct {
	unit *unitVis
	text string
	t    float64
	row  int        // stacked above the unit's other fresh popups
	kind numberKind // brush-written damage or heal number; zero for plain text
	bits []spatter  // ink drops or light motes of a number
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
	started []float64 // when each action started on the replay's clock; -1 before
	clock   float64
	waiting []int
	running []*running
	gap     float64
	final   core.Observation
	// looks are the cells' states shown while a replay runs: those of the field before it, changed
	// by each terrain event when the replay reaches it, so a fire starts when its cutscene says so
	looks  map[image.Point]string
	Duel   *duelview.View
	Speech []*battleBubble
	Fading []*battleBubble // lines done with, fading out
	// nextLine asks the replay to move on from the line on screen (a click once it is all out)
	nextLine bool
	// ai marks an AI army's turn: its actions start on screen and log as they start (ai_view.go)
	ai bool
	// aimed is the action the camera last glided to; it starts even if not all of it fits
	aimed int
	// from is the state the AI turn started in, whose army the HUD names while it plays
	from core.Observation
}

func (p *Player) Busy() bool { return len(p.waiting)+len(p.running) > 0 }

// Play queues actions; final is the observation after all of them (synced at the end).
func (p *Player) Play(actions []*replay.Action, final core.Observation) {
	p.play(actions, final, false)
}

// PlayAI queues the AI army's turn that started in from: each action waits to be on screen and
// writes the battle log when it starts.
func (p *Player) PlayAI(from core.Observation, actions []*replay.Action, final core.Observation) {
	p.play(actions, final, true)
	p.from = from
}

// shownTurn is o with the turn the screen is playing: that of an AI turn until its replay ends,
// though the rules have moved on.
func (p *Player) shownTurn(o core.Observation) core.Observation {
	if p.ai && p.Busy() {
		o.Turn, o.Army, o.Round = p.from.Turn, p.from.Army, p.from.Round
	}
	return o
}

func (p *Player) play(actions []*replay.Action, final core.Observation, ai bool) {
	p.g.dropSpeech()
	p.actions, p.final, p.ai, p.aimed = actions, final, ai, -1
	p.done = make([]bool, len(actions))
	p.started = make([]float64, len(actions))
	for i := range p.started {
		p.started[i] = -1
	}
	p.clock = 0
	p.waiting = p.waiting[:0]
	p.running = p.running[:0]
	for i := range actions {
		p.waiting = append(p.waiting, i)
	}
	p.gap = stagger
	p.looks = nil
	if len(actions) == 0 {
		p.g.sync(final)
		return
	}
	p.looks = p.g.shownLooks(final)
}

func (p *Player) Update(dt float64) {
	if !p.Busy() {
		return
	}
	if p.Duel != nil {
		p.Duel.Advance(p.cutInStep(dt)) // the duel's clock waits for the cut-in
	}
	p.gap += dt
	p.clock += dt
	if p.gap >= stagger {
		ready := []int{}
		for k, i := range p.waiting {
			ok := true
			for _, j := range p.actions[i].After {
				ok = ok && p.done[j]
			}
			for _, l := range p.actions[i].Trail {
				ok = ok && p.started[l.Index] >= 0 && p.clock >= p.started[l.Index]+l.Steps*stepTime
			}
			if ok {
				ready = append(ready, k)
			}
		}
		if k := p.nextShown(ready); k >= 0 {
			i := p.waiting[k]
			p.waiting = append(p.waiting[:k], p.waiting[k+1:]...)
			p.logAction(i)
			p.started[i] = p.clock
			r := &running{index: i, beats: p.g.perform(p.actions[i]), i: -1}
			p.running = append(p.running, r)
			p.gap = 0
		}
	}
	if p.nextLine {
		p.nextLine = false
		walking := false
		for _, v := range p.g.vis {
			walking = walking || v.walk != nil
		}
		if !walking { // the line on screen is cut short; a walk under way is not
			for _, r := range p.running {
				r.left = min(r.left, 0)
			}
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
				v.inactive = p.final.Phase != "battle" || !u.ActsIn(p.final.Turn, p.final.Army) || u.Done
			}
		} else {
			kept = append(kept, r)
		}
	}
	p.running = kept
	if !p.Busy() {
		p.Duel = nil
		p.g.dropSpeech()
		p.looks = nil
		p.g.sync(p.final)
	}
}

// Skip shows the final state at once; the outcome is already decided.
func (p *Player) Skip() {
	if !p.Busy() {
		return
	}
	for _, i := range p.waiting {
		p.logAction(i)
	}
	p.waiting, p.running, p.Duel = nil, nil, nil
	p.g.dropSpeech()
	p.looks = nil
	p.g.popups = nil
	p.g.sync(p.final)
}

// logAction writes an AI action's commands to the battle log as it is shown.
func (p *Player) logAction(i int) {
	if p.ai {
		for _, s := range p.actions[i].Steps {
			p.g.logCommand(s.Command, s.Events)
		}
	}
}

func (g *Game) unitAt(id string) *unitVis { return g.vis[id] }

// perform turns one action into timed beats; each beat's function runs when it starts.
func (g *Game) perform(a *replay.Action) []beat {
	beats := []beat{}
	add := func(d float64, f func()) { beats = append(beats, beat{d, f}) }
	steps := []replay.Step{}
	for _, s := range a.Steps {
		steps = append(steps, splitFollowups(s.Command, s.Events)...)
	}
	for _, s := range steps {
		c, events := s.Command, s.Events
		actor := g.unitAt(c.Actor)
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
			combatEvents := events
			for i, e := range events {
				if e.Kind == "duel" {
					combatEvents = events[:i]
					break
				}
			}
			blow, counters := splitCounters(c.Actor, combatEvents)
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
				aim := actor
				for _, e := range k.events {
					if e.Kind == "damage" || e.Kind == "miss" {
						aim = g.unitAt(e.Target)
						break
					}
				}
				add(.15, func() {})
				for _, part := range splitDouble(k.events) {
					crit := false
					for _, e := range part {
						crit = crit || e.Kind == "critical"
					}
					g.swing(add, back, aim, core.Command{Kind: "attack"}, part, crit)
				}
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
						t.shownMP = min(t.maxMP, t.shownMP+e.Amount)
						g.say(t, fmt.Sprintf("MP +%d", e.Amount))
					} else {
						t.shownHP = min(t.maxHP, t.shownHP+e.Amount)
						g.say(t, fmt.Sprintf("HP +%d", e.Amount))
					}
				}
			})
		case "learn":
			add(.6, func() { g.say(actor, "학습: "+g.traitName(c.Trait)) })
		case "end":
			ev := events
			add(0, func() { g.strike(nil, ev) })
			for _, e := range events {
				if e.Kind == "turn" {
					turn := e
					add(1.0, func() { g.showTurn(turn.Amount, turn.Target, turn.Text) })
				}
			}
		default:
			add(.1, func() {})
		}
		// forced moves (an event action) walk together, and a cutscene plays its groups of steps
		var walks []timedFn
		for _, e := range events {
			if e.Kind == "move" && (c.Kind != "move" || e.Actor != c.Actor) {
				walks = append(walks, g.walkTimeline(e.Actor, e.Path, e.Facing, 0)...)
			}
		}
		if len(walks) > 0 {
			addTimeline(add, walks, timelineEnd(walks))
		}
		for _, e := range events {
			if u := g.unitAt(e.Actor); e.Kind == "retreat" && e.Text == "flight" && u != nil {
				dir := heading(e.Actor, events)
				add(leaveTime, func() { g.leaveField(u, dir) })
			}
		}
		// the camera goes first to where a run of arrivals or of terrain changes happens, so the
		// player sees an army come in or a fire start instead of a screen that seems to wait
		looking := ""
		look := func(kind string, x, y int) {
			if looking != kind {
				looking = kind
				add(panTime, func() { g.panToCell(x, y) })
			}
		}
		for i, e := range events {
			if e.Kind != "terrain" && e.Kind != "unit-arrival" {
				looking = ""
			}
			switch e.Kind {
			case "duel":
				view := g.duelView(e)
				ev := events[i+1:]
				add(view.Length()+cutInLength(view), func() { g.player.Duel = view })
				add(blinkTime, func() { g.player.Duel = nil; g.strike(nil, ev) })
			case "status":
				// an event's stat change on many units at once shows on all of them together
				if i > 0 && events[i-1].Kind == "status" && events[i-1].Text == e.Text {
					continue
				}
				var run []core.Event
				for _, f := range events[i:] {
					if f.Kind != "status" || f.Text != e.Text {
						break
					}
					run = append(run, f)
				}
				add(1.2, func() {
					for _, f := range run {
						t := g.unitAt(f.Target)
						kind, tint := "debuff", spark
						if f.Amount > 0 {
							kind, tint = "buff", healC
						}
						g.spawn(kind, nil, t, 1.1, tint)
						g.say(t, fmt.Sprintf("%s %+d%%", statusLabels[f.Text], f.Amount))
					}
				})
			case "cutscene":
				entries, total := g.cutsceneTimeline(e.Cutscene)
				addTimeline(add, entries, total)
			case "goals":
				add(.3, func() { g.modals = append(g.modals, &modal{kind: "objective"}) })
			case "terrain":
				event := e
				if x, y, ok := g.areaCenter(e.Amount); ok {
					look("terrain", x, y)
				}
				add(.4, func() { g.changeLooks(event) })
			case "mark":
				// the marks one spell leaves come up together
				event, wait := e, .3
				if i > 0 && events[i-1].Kind == "mark" {
					wait = 0
				}
				add(wait, func() { g.markLook(event) })
			case "unit-arrival":
				event := e
				if e.Unit != nil {
					look("arrival", e.Unit.X, e.Unit.Y)
				}
				add(.15, func() {
					if event.Unit == nil {
						return
					}
					if line, _ := g.eventText(event); line != "" {
						g.logPending = append(g.logPending, line)
					}
					if final := g.unitView(g.player.final, event.Actor); final != nil {
						view := *final
						view.Unit = *event.Unit
						g.syncUnit(view, g.player.final)
					}
				})
			case "speech":
				// the lines are said one after another, each in its own bubble
				for _, line := range e.Speech {
					lines := []content.Speech{line}
					add(max(2.4, 1.5+float64(len([]rune(line.Text)))*.09), func() {
						g.showBattleSpeech(lines)
						g.logPending = append(g.logPending, g.name(line.Actor)+": "+line.Text)
					})
				}
				add(0, func() { g.dropSpeech() })
			}
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

// splitFollowups preserves the completed original exchange before each extra attack and each next
// target of a skill that strikes them one by one.
func splitFollowups(c core.Command, events []core.Event) []replay.Step {
	steps := []replay.Step{{Command: c}}
	for _, e := range events {
		switch e.Kind {
		case "followup-start", "next-strike":
			steps = append(steps, replay.Step{Command: core.Command{Kind: "attack", Actor: e.Actor, Target: e.Target}})
		case "followup-end":
		default:
			i := len(steps) - 1
			steps[i].Events = append(steps[i].Events, e)
		}
	}
	return steps
}

// counter is a counter-attack inside an action's events: who strikes back and what it does.
type counter struct {
	by     string
	events []core.Event
}

// splitCounters separates the counter-attacks (and assists) from an attack's events. A counter is a
// blow (critical, damage, miss) by someone other than the actor, with the retreats its blows
// cause; each one is shown as its own swing after the attack lands.
func splitCounters(actor string, events []core.Event) ([]core.Event, []counter) {
	blow, counters := []core.Event{}, []counter{}
	struck := map[string]string{}
	for _, e := range events {
		if e.Kind == "damage" {
			struck[e.Target] = e.Actor
		}
		back := e.Actor != actor && e.Actor != "" && (e.Kind == "critical" || e.Kind == "damage" || e.Kind == "miss" || e.Kind == "double")
		switch {
		case back && (len(counters) == 0 || counters[len(counters)-1].by != e.Actor):
			counters = append(counters, counter{by: e.Actor, events: []core.Event{e}})
		case back || e.Kind == "retreat" && len(counters) > 0 && struck[e.Actor] == counters[len(counters)-1].by:
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
	shooting := g.shootsFromRange(actor, target, c)
	if shooting && kind == "slash" {
		kind = "arrow"
	}
	look := actor.art
	tint := buffTint[g.S.Data.Skills[c.Skill].Effect]
	start := func() {
		if target != nil && target != actor {
			actor.face(target.u, target.v)
		}
		if shooting {
			actor.art = actor.shoot
		}
		actor.play("attack")
		if c.Kind == "skill" {
			g.say(actor, g.skillName(c.Skill))
		}
		for _, e := range events {
			if e.Kind == "cost" {
				actor.shownMP = max(0, actor.shownMP-e.Amount)
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
	if kind == "stone" && target != nil && target != actor {
		// leaves the bucket as attack frame 2 starts, and the blow lands with it
		flight := g.lobFlight(actor, target)
		add(flight, func() { g.throw(actor, target, flight) })
	}
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
	add(wait, func() {
		actor.art = look
		g.impact(kind, actor, target, events, tint)
		g.strike(actor, events)
	})
}

// shootsFromRange tells whether the basic attack is a shot in the actor's shoot look: it has one and the
// target stands beyond the eight cells around it, judged where both stand when the action ends.
func (g *Game) shootsFromRange(actor, target *unitVis, c core.Command) bool {
	if actor == nil || target == nil || actor == target || actor.shoot == nil || c.Kind != "attack" {
		return false
	}
	cell := func(v *unitVis) (int, int) {
		for _, w := range g.player.final.UnitViews {
			if w.ID == v.id {
				return w.X, w.Y
			}
		}
		return int(math.Round(v.u)), int(math.Round(v.v))
	}
	ax, ay := cell(actor)
	tx, ty := cell(target)
	return math.Abs(float64(ax-tx)) > 1 || math.Abs(float64(ay-ty)) > 1
}

// heading is the direction of id's last step among events, zero when it did not walk.
func heading(id string, events []core.Event) image.Point {
	for i := len(events) - 1; i >= 0; i-- {
		if e := events[i]; e.Kind == "move" && e.Actor == id && len(e.Path) > 1 {
			a, b := e.Path[len(e.Path)-2], e.Path[len(e.Path)-1]
			return image.Pt(b.X-a.X, b.Y-a.Y)
		}
	}
	return image.Point{}
}

// leaveField walks u off the battle area across the nearest edge (the way it was heading when it
// stands on a corner), darkening like the ground there and fading out; off the edge it fades in place.
func (g *Game) leaveField(u *unitVis, dir image.Point) {
	cu, cv := int(math.Round(u.u)), int(math.Round(u.v))
	out := func(d image.Point) bool { return d != (image.Point{}) && !g.field.Inside(cu+d.X, cv+d.Y) }
	var d image.Point
	for _, c := range []image.Point{dir, {1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		if out(c) {
			d = c
			break
		}
	}
	tu, tv := u.u+float64(d.X)*leaveCells, u.v+float64(d.Y)*leaveCells
	u.shownHP = 0
	u.walk = nil
	u.face(tu, tv)
	u.play("walk")
	u.leave = &walkState{fu: u.u, fv: u.v, tu: tu, tv: tv}
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
	if u := g.unitView(g.S.View(), id); u != nil {
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
				g.sayNumber(t, numberCrit, e.Amount)
				continue
			}
			g.sayNumber(t, numberDamage, e.Amount)
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
			g.sayNumber(t, numberHeal, e.Amount)
		case "recover-hp", "pouch-heal":
			if u := g.unitAt(e.Actor); u != nil {
				u.shownHP = min(u.maxHP, u.shownHP+e.Amount)
				g.sayNumber(u, numberHeal, e.Amount)
			}
		case "recover-mp":
			if u := g.unitAt(e.Actor); u != nil {
				u.shownMP = min(u.maxMP, u.shownMP+e.Amount)
				g.say(u, fmt.Sprintf("MP +%d", e.Amount))
			}
		case "experience":
			g.say(g.unitAt(e.Actor), fmt.Sprintf("EXP +%d", g.expShown(e.Actor, e.Amount)))
		case "effect":
			if t != nil {
				t.flash = .3
			}
		case "retreat", "death":
			if u := g.unitAt(e.Actor); u != nil && e.Text != "flight" {
				u.shownHP = 0
				u.dying = 0
			}
		}
	}
}
