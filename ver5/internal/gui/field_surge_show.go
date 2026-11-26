package gui

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/content"
	"srpg/internal/core"
)

// suigongLine is a bubble the show raises: who, when (seconds into the show) and what.
type suigongLine struct {
	unit string
	at   float64
	text string
	big  bool // the story's dialogue box with the speaker's portrait, for someone the close-ups do not show well
}

// suigongCue is something the show does once, at its moment.
type suigongCue struct {
	at float64
	fn func()
}

// suigongShow plays the flood of uidemo_suigong.go (see there for the shots and the timeline's constants) over the
// field of a battle: the ford, the cliff and the dam, the burst, the ford again and the white. Everything in it
// is the field's own picture of the baked surge; the show only moves the camera, the units and the screen's dark
// and white. The demo and the battle's cutscene step "surge" both drive it by time: at(t) is the picture at t
// seconds, and may be asked for any t not before the last.
type suigongShow struct {
	g       *Game
	surge   *Surge
	hill    float64
	soldier map[[2]int]*unitVis // the army's soldiers by their start cells
	start   map[string]float64  // and the u each marches from
	swept   map[string]*suigongWash
	cried   map[string]bool
	cues    []suigongCue
	next    int
	tCut    float64
	end     float64 // when the picture is white all over, a battle swaps its units back
	dark    float64
	white   float64
	extras  []string // ids of the soldiers this show made for itself
	stands  map[string]bool
	node    string  // the story node the scene's show plays in: the white holds until the story has left it
	change  func()  // what happens once, in the dark the story's scene fades to (the army appears, the cast steps aside)
	hold    float64 // when the white began to clear, once the show is over
	scene   *scene  // the story scene the show plays in, whose cast are its units; nil in a battle, over the field's units
	saved   struct {
		zoom, camX, camY, speech float64
		unitsHeld                bool
	}
	clock float64
	over  bool
}

// suigongClearSecs is how long the white takes to clear when the show hands the battle back.
const suigongClearSecs = .7

// A story scene darkens for sceneDarken seconds before the flood's first shot and lights up in sceneLighten.
const sceneDarken, sceneLighten = .35, .5

// suigongStands are the cells on the right cliff over the dam where the first two of the units a show names stand,
// looking down the river.
var suigongStands = []content.Point{
	{X: suigongRiver + suigongWidth + 1, Y: suigongDam - 1},
	{X: suigongRiver + suigongWidth + 2, Y: suigongDam - 1},
}

// newSuigongShow readies the show on the g.field: the soldiers of the ford are the field's units of the army
// (cells from suigongArmy) that stand on the ford's rows, stands name the units on the cliff (in the cells of
// suigongStands), order is what one of them shouts, lines are the other bubbles. The units the field shows are put
// where the show wants them now.
func (g *Game) newSuigongShow(surge *Surge, stands []string, order suigongLine, lines []suigongLine, sc *scene) (*suigongShow, error) {
	s := &suigongShow{g: g, surge: surge, hill: float64(g.field.A.CliffRise), swept: map[string]*suigongWash{}, cried: map[string]bool{},
		soldier: map[[2]int]*unitVis{}, start: map[string]float64{}, stands: map[string]bool{}, scene: sc}
	s.saved.zoom, s.saved.camX, s.saved.camY, s.saved.speech, s.saved.unitsHeld = g.zoom, g.camX, g.camY, g.player.SpeechScale, g.unitsHeld
	g.unitsHeld = true // the soldiers walk on their own, the rules state keeps them on their start cells
	if sc != nil {
		for _, id := range stands { // a scene's cast already stands on the cliff, where its map lifts them
			s.stands[id] = true
			if u := sc.actors[id]; u != nil {
				s.hill = max(s.hill, u.lift)
			}
		}
		return s, s.arrange(order, lines)
	}
	for i, id := range stands[:min(len(stands), len(suigongStands))] {
		s.stands[id] = true
		if u := g.vis[id]; u != nil {
			p := suigongStands[i]
			u.u, u.v, u.walk, u.dir = float64(p.X), float64(p.Y), nil, "SW"
		}
	}
	return s, s.arrange(order, lines)
}

// units are what the show moves about: a scene's cast, or the battle field's units.
func (s *suigongShow) units() map[string]*unitVis {
	if s.scene != nil {
		return s.scene.actors
	}
	return s.g.vis
}

// arrange files the soldiers and lays out the cues.
func (s *suigongShow) arrange(order suigongLine, lines []suigongLine) error {
	g := s.g
	for _, u := range s.units() {
		if !u.ally && u.v >= suigongFord {
			s.soldier[[2]int{int(u.u), int(u.v)}] = u
			s.start[u.id] = u.u
		}
	}
	if len(s.soldier) < 8 {
		return fmt.Errorf("suigong: %d soldiers stand on the ford", len(s.soldier))
	}
	s.tCut = tBurst + (s.surge.Cut()-surgeBreak)/surgeRate // the shot changes from the dam to the ford
	s.end = s.tCut + fordWater + fordWhite + fordHold
	s.hold = s.end
	cue := func(at float64, fn func()) { s.cues = append(s.cues, suigongCue{at, fn}) }
	sound := func(at float64, name string, vol float64) { cue(at, func() { g.sfxVol(name, vol) }) }
	sound(tShake, "boulder", 1)
	sound(tShake+.9, "boulder", .7) // the rumble goes on for the whole shaking
	sound(tBurst, "water-big", 1)
	sound(tBurst+.25, "flood-surge", 1) // the flood's rush, from the gap to the swell of the white
	sound(s.tCut+.5, "flood-crash", 1)  // the front breaks on the soldiers
	sound(tBurst+1.7, "flood-spray", 1) // as the spray flies
	for _, l := range lines {
		cue(l.at, func() { g.showBattleSpeech([]content.Speech{{Actor: l.unit, Text: l.text, Offstage: l.big}}) })
		cue(l.at+2.2, g.dropSpeech)
	}
	if order.text != "" {
		cue(order.at, func() {
			g.showBattleSpeech([]content.Speech{{Actor: order.unit, Text: order.text}})
			if u := s.units()[order.unit]; u != nil {
				u.play("attack")
			}
		})
		cue(tHush, g.dropSpeech)
	}
	cue(s.tCut-1.0/60, func() { g.player.Fading = nil }) // a bubble must not fade across the cut
	for _, c := range suigongCries {
		cue(s.tCut+c.at, func() {
			if u := s.soldier[[2]int{c.u + marchLead, c.v}]; u != nil {
				s.cried[u.id] = true
				if s.scene != nil {
					s.scene.emotes[u.id] = &emote{text: c.text, life: 1.6}
				} else {
					g.player.Speech = append(g.player.Speech, &battleBubble{actor: u.id, name: "조조군", text: c.text, typing: startTyping(g.textSeconds)})
				}
			}
		})
	}
	sort.SliceStable(s.cues, func(i, j int) bool { return s.cues[i].at < s.cues[j].at })
	g.player.SpeechScale = 2 // the shots are close-ups
	g.field.Surge = s.surge
	return nil
}

func suigongPhase(t, a, b float64) float64 { return math.Max(0, math.Min(1, (t-a)/(b-a))) }
func suigongSmooth(x float64) float64      { return x * x * (3 - 2*x) }

// Length is how long the show takes, the white's clearing included.
func (s *suigongShow) Length() float64 { return s.end + suigongClearSecs }

// suigongWash is a soldier the water has reached: where it stood and when. The flood carries it down the river and
// fades it out as it goes under, each a little differently.
type suigongWash struct{ at, u, v float64 }

const (
	washDrift  = 2.2 // cells the water carries a soldier down the river, and up to washSpread more
	washSpread = 1.2
	washSecs   = 1.2 // seconds it is carried
	washFade   = .7  // seconds it takes to fade out, after washLag
	washLag    = .2  // seconds under before it begins to fade, and up to washSpread * washLag more
)

// carry puts u where the flood has taken it t seconds in: down the river (+v), swaying, and going out of the picture.
func (w *suigongWash) carry(u *unitVis, t float64) {
	var h uint32 = 2166136261
	for _, c := range []byte(u.id) {
		h = (h ^ uint32(c)) * 16777619
	}
	j := float64(h%1000) / 1000 // this soldier's own measure of the water's strength
	e := t - w.at
	k := suigongSmooth(suigongPhase(e, 0, washSecs))
	u.v = math.Min(w.v+k*(washDrift+washSpread*j), suigongH-1)
	u.u = w.u + .3*k*math.Sin(4*e+6*j)
	u.fade = suigongSmooth(suigongPhase(e, washLag*(1+washSpread*j), washLag*(1+washSpread*j)+washFade))
}

// at draws the show's moment t: the surge, the screen's dark and white, the camera and every soldier's doing.
func (s *suigongShow) at(t float64) {
	g := s.g
	if s.over {
		if s.scene != nil && g.S.View().Node == s.node {
			s.hold = t // the story goes on to the next node under the white
		}
		s.dark, s.white = 0, 1-suigongPhase(t, s.hold, s.hold+suigongClearSecs)
		return
	}
	for s.next < len(s.cues) && s.cues[s.next].at <= t {
		s.cues[s.next].fn()
		s.next++
	}
	st := 0.0
	if t >= tBurst {
		st = surgeBreak + (t-tBurst)*surgeRate
	}
	g.field.Surge, g.field.SurgeT = s.surge, st
	s.dark = math.Min(suigongPhase(t, tFadeOut, tBlack), 1-suigongPhase(t, tBlack, tFadeIn))
	if s.scene != nil { // the story's scene darkens to the ford and the army, which stand there once it is black
		s.dark = math.Max(s.dark, math.Min(suigongPhase(t, 0, sceneDarken), 1-suigongPhase(t, sceneDarken, sceneDarken+sceneLighten)))
		if t >= sceneDarken && s.change != nil {
			s.change()
			s.change = nil
		}
	}
	s.white = suigongPhase(t, tBurst-.06, tBurst) * (1 - suigongPhase(t, tBurst, tBurst+flashSecs))
	s.white = math.Max(s.white, suigongSmooth(suigongPhase(t, s.tCut+fordWater, s.tCut+fordWater+fordWhite)))
	if t >= s.end {
		s.finish()
		s.white = 1
		return
	}
	if s.scene != nil && t < sceneDarken {
		return // the scene's own shot, darkening
	}
	g.setZoomImmediate(4)
	switch {
	case t < tBlack || t >= s.tCut:
		g.camX, g.camY = g.field.Center(suigongFordCamU, suigongFordCamV)
	case t < tScroll:
		g.camX, g.camY = s.guanyuCam()
	default:
		x0, y0 := s.guanyuCam()
		x1, y1 := s.damCam()
		k := suigongSmooth(suigongPhase(t, tScroll, tScroll+scrollSecs))
		g.camX, g.camY = x0+(x1-x0)*k, y0+(y1-y0)*k
	}
	amp := 0.0
	switch {
	case t >= tBurst:
		amp = 2.6 * (1 - suigongPhase(t, tBurst, tBurst+aftershock))
	case t >= tShake:
		amp = .8 + 1.8*suigongPhase(t, tShake, tBurst)
	}
	if amp > 0 {
		f := float64(int(t * 60))
		g.camX += amp * (math.Sin(f*2.3) + .5*math.Sin(f*5.1))
		g.camY += amp * .7 * math.Cos(f*1.9+1)
	}
	marching := t < tBlack || t >= s.tCut
	for id, u := range s.units() {
		u.actor = true
		if s.stands[id] {
			u.lift, u.front, u.dir = s.hill, 40, "SW"
		}
		if u.ally || u.v < suigongFord-1 {
			continue
		}
		switch {
		case u.flood > 4 && s.swept[u.id] == nil: // the water reaches it
			s.swept[u.id] = &suigongWash{at: t, u: u.u, v: u.v}
			u.walk = nil
			u.play("hit")
			u.shakeT, u.shakeAmp, u.shakeSign = .35, 2, 1
		case s.swept[u.id] != nil:
			s.swept[u.id].carry(u, t)
		case !marching: // out of sight while the view is up on the cliff
			u.walk = nil
			u.play("idle")
		case u.walk != nil: // in the middle of a cell: it finishes it
		case s.cried[u.id]: // it has seen the water: it stands looking up the river
			u.dir = "NE"
			if u.anim != "idle" {
				u.play("idle")
			}
		default:
			g.marchCell(u, s.start[u.id], marchPhase(int(u.v)))
		}
	}
}

func (s *suigongShow) guanyuCam() (float64, float64) {
	x, y := s.g.field.Center(suigongRiver+suigongWidth+.5, suigongDam-1)
	return x, y - s.hill
}

func (s *suigongShow) damCam() (float64, float64) {
	x, y := s.g.field.Center(suigongRiver+suigongWidth/2, suigongDam+.8)
	return x, y - s.hill/2
}

// finish hands the battle back at the moment the screen is white: the soldiers made for the show go, the field's
// own units stand where the rules put them and the camera and the zoom are the battle's again.
func (s *suigongShow) finish() {
	g := s.g
	if s.over {
		return
	}
	s.over = true
	for _, id := range s.extras {
		delete(s.units(), id)
		if s.scene != nil {
			delete(s.scene.extra, id)
			delete(s.scene.free, id)
			delete(s.scene.emotes, id)
		}
	}
	s.extras = nil
	g.unitsHeld = s.saved.unitsHeld
	g.player.SpeechScale = s.saved.speech
	g.player.Speech, g.player.Fading = nil, nil
	g.field.Surge, g.field.SurgeT = nil, 0
	for _, u := range s.units() {
		u.actor, u.front, u.shakeT, u.fade = false, 0, 0, 0
	}
	if o := g.S.View(); o.Map != nil && s.scene == nil {
		g.sync(o)
	}
	for _, u := range s.units() {
		u.play(u.rest())
	}
	g.setZoomImmediate(s.saved.zoom)
	g.camX, g.camY = s.saved.camX, s.saved.camY
}

// overlay is the screen's dark and white over the picture.
func (s *suigongShow) overlay(dst *ebiten.Image) {
	if s.dark > 0 {
		rect(dst, 0, 0, float32(uiW), float32(uiH), color.NRGBA{A: uint8(255 * s.dark)})
	}
	if s.white > 0 {
		rect(dst, 0, 0, float32(uiW), float32(uiH), color.NRGBA{255, 255, 255, uint8(255 * s.white)})
	}
}

// Advance runs the show's clock dt seconds on and reports that it is over, the white cleared.
func (s *suigongShow) Advance(dt float64) bool {
	s.clock += dt
	s.at(s.clock)
	return s.over && s.clock >= s.hold+suigongClearSecs
}

// suigongSeconds is how long the show of surge takes.
func suigongSeconds(surge *Surge) float64 {
	return tBurst + (surge.Cut()-surgeBreak)/surgeRate + fordWater + fordWhite + fordHold + suigongClearSecs
}

// beginSuigong starts the flood show of the surge called name over the battle. The group tells the cast: the unit of
// the step "surge" leads the army across the ford; the units of its "face" steps stand on the cliff over the dam
// (suigongStands); its "say" steps are the bubbles, each at its Seconds, and the line of a unit on the cliff is the
// order that the dam be broken, held until the view leaves the cliff. The field's soldiers of the army are copies of
// one of the stage's own troops, which the show lets go as the white comes; every other unit is held out of sight
// until the show hands the battle back.
func (g *Game) beginSuigong(name string, group []core.CutStep) {
	g.endSuigong()
	surge, err := g.assets.Surge(name)
	if err != nil || g.field == nil {
		g.toast(fmt.Sprintf("수공: %v", err), bad)
		return
	}
	var leader string
	var stands []string
	for _, st := range group {
		switch st.Kind {
		case "surge":
			leader = st.Unit
		case "face":
			stands = append(stands, st.Unit)
		}
	}
	o := g.S.View()
	var foot, horse *core.UnitView
	for i := range o.UnitViews {
		u := &o.UnitViews[i]
		if u.Faction == "ally" || u.HP <= 0 || !g.S.Data.Characters[u.Def].Template {
			continue
		}
		if slices.Contains(g.S.Data.Classes[u.Class].Tags, content.TagMounted) {
			horse = cmpOr(horse, u)
		} else {
			foot = cmpOr(foot, u)
		}
	}
	if foot == nil {
		g.toast(fmt.Sprintf("수공: 보병 병사가 없다 (%d units, phase %s)", len(o.UnitViews), o.Phase), bad)
		return
	}
	if horse == nil {
		horse = foot
	}
	for id, v := range g.vis {
		if slices.Contains(stands, id) {
			continue
		}
		v.u, v.v, v.walk = suigongW-1, 0, nil // out of the shots, which look at the river and the ford
		if id == leader {
			v.u, v.v, v.dir = suigongRiver+2+marchLead, suigongFord+1, "NW"
		}
	}
	var extras []string
	for i, r := range suigongArmy() {
		c := *foot
		if r.rides {
			c = *horse
		}
		c.ID, c.X, c.Y, c.Facing = fmt.Sprintf("surge#%d", i), r.u, r.v, "NW"
		g.syncUnit(c, o)
		extras = append(extras, c.ID)
	}
	var order suigongLine
	var rest []suigongLine
	for _, st := range group {
		if st.Kind != "say" {
			continue
		}
		if l := (suigongLine{st.Unit, st.Seconds, st.Text, !slices.Contains(stands, st.Unit)}); slices.Contains(stands, st.Unit) && order.text == "" {
			order = l
		} else {
			rest = append(rest, l)
		}
	}
	show, err := g.newSuigongShow(surge, stands, order, rest, nil)
	if err != nil {
		for _, id := range extras {
			delete(g.vis, id)
		}
		g.toast(err.Error(), bad)
		return
	}
	show.extras = extras
	g.show = show
	show.at(0)
}

func cmpOr(old, u *core.UnitView) *core.UnitView {
	if old != nil {
		return old
	}
	return u
}

// endSuigong ends a show that is still going at once, the battle as the rules state has it.
func (g *Game) endSuigong() {
	if g.show == nil {
		return
	}
	g.show.finish()
	g.show = nil
}
