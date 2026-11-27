package gui

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/duelview"
	"srpg/internal/scenes"
	"srpg/internal/sprite"
	"srpg/internal/storage"
)

// scene is the running scene (assets/scenes/*.json): one stretch of story nodes on one map. Its actors keep where
// the beats took them from node to node; a node entered in the middle of a stretch (a loaded game) first settles
// the beats of the nodes before it.
type scene struct {
	def       *scenes.Scene
	node      string
	actors    map[string]*unitVis
	extra     map[string]bool // cast keys with "#": no bubble when a name matches, never turned toward a speaker
	hold      map[string]bool // cast that keeps its facing while others speak
	held      map[string]bool // turned or moved by this node's beats: auto-facing leaves them alone
	groups    [][]scenes.Step
	gi        int
	begun     bool
	t         float64 // seconds since the running group started
	pending   []scenes.Step
	wait      float64
	paths     map[string][][2]float64
	speed     map[string]float64
	exits     map[string]bool
	ground    *exitGround // where an actor leaving the stage can walk to the front edge
	plays     map[string]string
	emotes    map[string]*emote
	looks     map[string]*look
	effects   []sceneEffect
	petals    []petal // drawPetals' layout, kept from frame to frame
	props     []sceneProp
	spoke     map[string]bool
	shake     float64
	camU      float64
	camV      float64
	fade      float64                 // 1 → 0: the scene opening out of black; 0 → 1 while leaving
	leaving   bool                    // darkening to black before the next node opens another scene
	nightfall bool                    // darkening to black before a beat changes the time of day
	patrol    map[string]*patrol      // sentries walking to and fro
	chatter   map[string]*chatterLine // extras murmuring now and then
	shot      *sceneShot              // an arrow chased by the camera (a Shoot step)
	free      map[string]bool         // walkers whose walk does not hold up the beats
	follow    string                  // the actor the camera keeps on
	zoneHeld  bool                    // the camera was moved off the scene's own shot by the still zone, to (heldX, heldY)
	heldX     float64
	heldY     float64
	focus     string                 // the named actor of the latest beat or line since the last camera step: the camera keeps them in view
	stride    map[string]float64     // cells walked since each walker's last footfall sounded
	back      map[string]string      // walkers backing away, with the way they keep facing
	strikes   []*sceneStrike         // Strike steps playing out
	brawls    map[string]*brawl      // actors trading blows in the background of a melee (scene_melee.go)
	stuck     []stuckArrow           // arrows that came down and stand in the ground
	swing     map[string]string      // riders dealing a blow on the move: the motion the walk must not cut short
	dust      []dustPuff             // dust kicked up by targets driven back by a blow
	arrows    []*sceneArrow          // arrows loosed by Play steps with an Arrow
	tied      map[string]*sprite.Art // bound prisoners, with the art they stand in once untied
	arrive    map[string]string      // adopted actors walking to their places, with the facing they take there
	base      *scenes.Scene          // the scene the stage was built from (an adopted event keeps the walk's)
	tod       string                 // the time of day the stage was last given
	settle    bool                   // adopted actors are walking to their places: the node's first line waits for them
}

type patrol struct {
	ends [2][2]float64
	next int
}

type chatterLine struct {
	id    string
	lines []string
	t     float64
	i     int
}

type sceneEffect struct {
	kind string
	u, v float64
	t    float64
	hit  float64 // seconds since an arrow struck it (a halberd); < 0 not struck
}

type sceneProp struct {
	name   string
	u, v   float64
	follow string // an actor the prop moves with, u and v then its offset from that actor
}

// at is where the prop stands now: its own spot, or its offset from the actor it follows; false once that
// actor has left the stage, so the prop leaves with them.
func (p sceneProp) at(s *scene) (float64, float64, bool) {
	if a := s.actors[p.follow]; a != nil {
		return a.u + p.u, a.v + p.v, !a.gone
	}
	return p.u, p.v, true
}

// art is the map prop the scene prop shows: a name like "straw_mat#2" lets one prop stand in several places.
func (p sceneProp) art() string {
	name, _, _ := strings.Cut(p.name, "#")
	return name
}

type emote struct {
	text  string
	t     float64
	life  float64
	block bool // a bubble line: the group waits for it
}

type look struct {
	back string
	t    float64
}

const (
	emoteTime   = 1.4
	lookTime    = .9
	camEase     = 4.0
	fadeTime    = .5
	chatterLife = 2.4 // seconds a murmured line stays up
	chatPeriod  = 6.0 // from the start of one line of an actor to the start of its next: up chatterLife, silent the rest
	chatSlot    = 1.5 // between the starts of two neighbouring actors' turns
	chatSlots   = chatPeriod / chatSlot
)

// sceneZoom and sceneSpeed are fixed in a story scene; only the walk lets the player change them.
const (
	sceneZoom  = 2
	sceneSpeed = 1
)

// enterScene sets up the scene of a dialogue node; false when the node has none.
func (g *Game) enterScene(o core.Observation) bool {
	def, at := g.scenes.For(o.Node)
	if def == nil || g.mapArt(def.Map) == nil {
		return false
	}
	if g.scene != nil && g.scene.node == o.Node {
		return true
	}
	if (g.scene == nil || g.scene.def != def || g.field == nil || g.field.Map.ID != def.Map) && !g.adoptScene(def, at) && !(g.layOutWalkFor(def) && g.adoptScene(def, at)) {
		g.openScene(def)
		for _, n := range def.Nodes[:at] {
			g.settleNode(n)
		}
	}
	s := g.scene
	s.node, s.held, s.focus = o.Node, map[string]bool{}, ""
	g.startLine(0)
	if s.settle {
		s.groups, s.settle = append([][]scenes.Step{{{}}}, s.groups...), false
	}
	return true
}

// openScene builds the scene's map and cast, lights its ambient effects and frames the cast.
func (g *Game) openScene(def *scenes.Scene) {
	if g.field != nil {
		g.field.dispose()
	}
	art := g.mapArt(def.Map)
	if def.Grass != nil || def.Snow > 0 {
		c := *art
		if def.Grass != nil {
			c.Grass = def.Grass
		}
		c.Snow = def.Snow
		art = &c
	}
	g.field = NewField(g.assets, art)
	s := &scene{def: def, actors: map[string]*unitVis{}, extra: map[string]bool{}, hold: map[string]bool{}, held: map[string]bool{},
		paths: map[string][][2]float64{}, speed: map[string]float64{}, exits: map[string]bool{}, plays: map[string]string{},
		emotes: map[string]*emote{}, looks: map[string]*look{}, fade: 1, patrol: map[string]*patrol{}, chatter: map[string]*chatterLine{},
		free: map[string]bool{}, stride: map[string]float64{}, back: map[string]string{}, swing: map[string]string{}}
	s.base = def
	s.ground = newExitGround(art, g.assets.SceneRules)
	g.scene = s
	for key, a := range def.Cast {
		if v := g.castActor(key, a); v != nil {
			v.gone = v.gone || g.S != nil && !g.S.Engine.Holds(a.If)
			s.actors[key] = v
			s.extra[key], s.hold[key] = scenes.Extra(key), a.Hold
			g.tie(key, a.Bound)
		}
	}
	g.setSceneTime(def.Time)
	s.camU, s.camV = g.castCentre() // an Ambient CamAt may open the scene elsewhere
	for _, st := range def.Ambient {
		if g.S == nil || g.S.Engine.Holds(st.If) {
			g.startStep(st)
		}
	}
	g.setZoomImmediate(sceneZoom)
	g.camX, g.camY = g.sceneCam()
}

// castActor is a cast member standing where the cast puts it; nil without a place or a sprite.
func (g *Game) castActor(key string, a scenes.Actor) *unitVis {
	art := g.actorArt(key, a.Sprite, a.Faction, a.Mount)
	if art == nil || len(a.At) != 2 {
		return nil
	}
	v := &unitVis{id: key, ally: true, dir: a.Face, anim: "idle", dying: -1, shownHP: 1, maxHP: 1, actor: true, still: true, art: art, top: art.TopOffset(), pose: a.Pose,
		mounted: g.riderArt(art.ID)}
	v.play(v.rest())
	v.animT = float64(len(key)*37%100) / 100 // extras do not all nod at once
	v.u, v.v, v.gone = a.At[0], a.At[1], a.Off
	v.lift = g.field.Lift(int(v.u), int(v.v))
	return v
}

// castCentre is the middle of the named actors on the stage, or of the map when there are none.
func (g *Game) castCentre() (float64, float64) {
	var u, v, n float64
	for key, a := range g.scene.actors {
		if !a.gone && !g.scene.extra[key] {
			u, v, n = u+a.u, v+a.v, n+1
		}
	}
	if n == 0 {
		return float64(g.field.Map.W)/2 - .5, float64(g.field.Map.H)/2 - .5
	}
	return u / n, v / n
}

// sceneCam is the canvas point the camera looks at: the camera cell, lifted a little so the heads stay clear of
// the dialogue scroll.
func (g *Game) sceneCam() (float64, float64) {
	x, y := g.field.Center(g.scene.camU, g.scene.camV)
	if g.reelPlaying() {
		dx, dy := g.reelOffset()
		x, y = x+dx, y+dy
	}
	return x, y - 20
}

// setSceneTime darkens or tints the scene: night is the battle maps' night, dusk and dawn a lighter wash,
// memory an even faded brown for a scene remembered, deepnight a darker night where nobody carries a torch and only
// the fires show the people.
func (g *Game) setSceneTime(t string) {
	if g.scene != nil {
		g.scene.tod = t
	}
	f := g.field
	f.setNight(t != "", nil)
	f.night.color = map[string]color.NRGBA{"night": nightColor, "dusk": {R: 70, G: 28, B: 12, A: 95}, "dawn": {R: 30, G: 40, B: 80, A: 85},
		"memory": {R: 112, G: 100, B: 84, A: 165}, "deepnight": {R: 5, G: 8, B: 24, A: 222}}[t]
	f.night.flat = t == "memory"
	f.night.torchless = t == "deepnight"
}

// actorArt is the sprite a cast member stands in: the cast's own pick (any sprite id: extras, soldiers, or a
// battle sprite), else the character's scene, battle or class sprite, in the character's faction colour unless
// the cast names another, ridden on the cast's mount.
func (g *Game) actorArt(key, pick, faction, mount string) *sprite.Art {
	def, c, _ := g.character(scenes.Base(key))
	if faction == "" {
		faction = c.Look.Faction
	}
	return g.sceneArt(def, pick, faction, mount)
}

// startLine queues the beats that come before speaker line i of the current node.
func (g *Game) startLine(i int) {
	s := g.scene
	if s == nil {
		return
	}
	s.groups, s.gi, s.begun, s.spoke = s.def.Line(s.node, i), 0, false, map[string]bool{}
}

// sceneBusy reports whether beats are still playing (the line waits for them).
func (g *Game) sceneBusy() bool { return g.scene != nil && g.scene.gi < len(g.scene.groups) }

func (g *Game) updateScene(dt float64) {
	g.advanceIntro(dt)
	s := g.scene
	if s == nil {
		return
	}
	for key, v := range s.actors {
		k := 1.0
		if v.walk != nil && s.speed[key] > 0 {
			k = s.speed[key]
		}
		swinging := s.swing[key] != "" && v.anim == s.swing[key]
		if swinging { // a blow dealt on the move plays at its own pace; the ride keeps the walk's speed (walkTo's dur)
			if w := v.walk; w != nil && w.dur <= 0 && k > 1 { // the cell it was in the middle of goes on at the ride's pace
				w.dur = stepTime / k
				w.t /= k
			}
			v.update(dt)
		} else {
			v.update(dt * k)
		}
		if v.fallen() { // struck down: the rest of its way is not walked
			s.paths[key] = nil
		}
		if p := s.paths[key]; v.walk == nil && len(p) > 0 {
			g.sceneFootfall(key, v, p[0])
			anim, at, quick := v.anim, v.animT, v.quick
			g.walkTo(v, p[0][0], p[0][1])
			if swinging = s.swing[key] != "" && anim == s.swing[key]; swinging {
				v.anim, v.animT, v.quick = anim, at, quick
			}
			if s.swing[key] != "" {
				v.walk.dur = stepTime / math.Max(1, s.speed[key])
			}
			if dir, ok := s.back[key]; ok {
				v.dir = dir
			}
			if v.walk.dur > 0 { // the time the last cell ran over goes into this one, so a long ride keeps its pace
				v.walk.t = v.over * v.walk.dur
			} else {
				v.walk.t = v.over * stepTime
			}
			s.paths[key] = p[1:]
		} else if v.walk == nil && len(p) == 0 {
			if dir := s.arrive[key]; dir != "" {
				v.dir = dir
				delete(s.arrive, key)
			}
			if pt := s.patrol[key]; pt != nil {
				s.paths[key] = scenePath(v.u, v.v, pt.ends[pt.next][0], pt.ends[pt.next][1])
				pt.next = 1 - pt.next
			} else if v.anim == "walk" && v.leave == nil && !s.exits[key] {
				v.play(v.rest())
			}
			route := [][2]float64(nil)
			if s.exits[key] { // walks to the front edge, on past it and fades out, like a unit leaving a battle
				route, _ = s.ground.route([2]float64{v.u, v.v}, v.dir, s.crowded(key))
			}
			if len(route) > 0 {
				s.paths[key] = route
				delete(s.back, key)
			} else {
				if s.exits[key] {
					s.exits[key] = false
					d := sceneStep[v.dir]
					v.play("walk")
					v.leave = &walkState{fu: v.u, fv: v.v, tu: v.u + d[0]*leaveCells, tv: v.v + d[1]*leaveCells}
				}
				delete(s.speed, key)
				delete(s.stride, key)
				delete(s.back, key)
			}
		}
	}
	for key, l := range s.looks {
		if l.t -= dt; l.t <= 0 {
			s.actors[key].dir = l.back
			delete(s.looks, key)
		}
	}
	for i := range s.effects {
		e := &s.effects[i]
		if e.kind == "halberd" && e.t < halberdDrop && e.t+dt >= halberdDrop {
			s.shake = math.Max(s.shake, .3)
		}
		if e.kind == "halberd_throw" && e.t < halberdThrow() && e.t+dt >= halberdThrow() {
			s.shake = math.Max(s.shake, .3)
			for key := range s.free { // the riders charging at each other pull up where it lands
				s.paths[key] = nil
				delete(s.free, key)
			}
		}
		e.t += dt
		if e.hit >= 0 {
			e.hit += dt
		}
	}
	for k, e := range s.emotes {
		if e.t += dt; e.t > e.life {
			delete(s.emotes, k)
		}
	}
	s.shake = math.Max(0, s.shake-dt)
	if s.leaving {
		if s.fade = math.Min(1, s.fade+dt/fadeTime); s.fade == 1 {
			s.leaving = false
			g.command(core.Command{Kind: "next"})
		}
	} else if s.nightfall {
		s.fade = math.Min(1, s.fade+dt/fadeTime)
	} else {
		s.fade = math.Max(0, s.fade-dt/fadeTime)
	}
	for key, c := range s.chatter {
		if c.t -= dt; c.t <= 0 && s.emotes[key] == nil && !s.actors[key].gone {
			v := s.actors[key]
			clear := func(text string) bool {
				r := g.bubbleRect(v, text)
				return !slices.ContainsFunc(slices.Collect(maps.Keys(s.emotes)), func(o string) bool {
					return s.actors[o] != nil && !s.actors[o].gone && g.bubbleRect(s.actors[o], s.emotes[o].text).Overlaps(r)
				})
			}
			if text, spoke := c.speak(clear); spoke {
				s.emotes[key] = &emote{text: text, life: chatterLife}
			}
		}
	}
	if a := s.actors[s.follow]; a != nil {
		s.camU, s.camV = a.u, a.v
		s.zoneHeld = false
	}
	if g.show == nil { // the flood frames its own shots
		tx, ty := g.sceneCam()
		if g.reelPlaying() { // the title's shots move their camera themselves (a followed stand-in): no lag behind it
			g.camX, g.camY = tx, ty
		} else {
			if len(s.def.Nodes) > 0 { // a story scene, not a walk the hero leads
				tx, ty = g.keepInView(tx, ty)
			}
			g.panCamera(tx, ty, dt, len(s.def.Nodes) > 0)
		}
	}
	g.updateShot(dt)
	g.updateStrikes(dt)
	g.updateBrawls(dt)
	g.updateArrows(dt)
	if !g.sceneBusy() {
		return
	}
	if !s.begun {
		if !s.nightfall && changesTime(s.groups[s.gi]) && s.fade < 1 {
			s.nightfall = true
		}
		if s.nightfall && s.fade < 1 {
			return
		}
		s.nightfall = false
		s.begun, s.t, s.pending = true, 0, nil
		for _, st := range s.groups[s.gi] {
			if st.After > 0 {
				s.pending = append(s.pending, st)
			} else {
				g.startStep(st)
			}
		}
		return
	}
	s.t += dt
	rest := s.pending[:0]
	for _, st := range s.pending {
		if st.After <= s.t {
			g.startStep(st)
		} else {
			rest = append(rest, st)
		}
	}
	s.pending = rest
	s.wait -= dt
	if s.wait > 0 || len(s.pending) > 0 || s.fade > 0 && changesTime(s.groups[s.gi]) {
		return
	}
	for key, v := range s.actors {
		if s.patrol[key] == nil && !s.free[key] && (len(s.paths[key]) > 0 || v.walk != nil) || s.plays[key] != "" && v.anim == s.plays[key] {
			return
		}
	}
	for _, e := range s.emotes {
		if e.block {
			return
		}
	}
	s.plays = map[string]string{}
	s.gi++
	s.begun = false
}

// keepInView moves the camera target (tx, ty) only when the latest beat's or line's actor has left the still zone
// in the middle of the screen (Settings.CamZone), bringing them back to its centre on the axis they left; inside the
// zone the shot stays as framed, so a scene does not scroll with every speaker. The feet stay above the dialogue scroll.
func (g *Game) keepInView(tx, ty float64) (float64, float64) {
	s := g.scene
	v := s.actors[s.focus]
	if s.zoneHeld { // the zone is judged against the shot the camera was last brought to
		tx, ty = s.heldX, s.heldY
	}
	if v == nil || v.gone || g.zoom <= 0 {
		return tx, ty
	}
	x, feet := g.field.Center(v.u, v.v)
	feet -= v.lift + g.field.groundLift(v.u, v.v)
	mid := feet - v.top/2
	vp := g.battleViewport()
	hw, hh := float64(vp.Dx())/2/g.zoom, float64(vp.Dy())/2/g.zoom
	zw, zh := storage.DefaultSettings().CamZone()
	if g.S != nil {
		zw, zh = g.S.Settings.CamZone()
	}
	if math.Abs(x-tx) > hw*float64(zw)/100 {
		tx = x
	}
	if math.Abs(mid-ty) > hh*float64(zh)/100 {
		ty = mid
	}
	box := dialogueBox().top * uiPx / worldPx
	floor := (box-float64(vp.Min.Y+vp.Dy()/2))/g.zoom - math.Max(8, hh*.2) // a little above the scroll's top edge
	// with no room for both, the feet stay above the scroll
	if feet > ty+floor {
		ty = feet - floor
	}
	s.zoneHeld, s.heldX, s.heldY = true, tx, ty
	return tx, ty
}

// camPanSpeed caps a story scene's camera, in screen widths a second, so a shot far from the last one glides
// there rather than whipping across the map; a cut in the dark of a fade lands at once.
const camPanSpeed = 1.2

// panCamera eases the camera toward (tx, ty).
func (g *Game) panCamera(tx, ty, dt float64, story bool) {
	if story && g.scene.fade > .9 {
		g.camX, g.camY = tx, ty
		return
	}
	ease := math.Min(1, dt*camEase)
	dx, dy := (tx-g.camX)*ease, (ty-g.camY)*ease
	if d, most := math.Hypot(dx, dy), camPanSpeed*float64(g.battleViewport().Dx())/g.zoom*dt; story && d > most && most > 0 {
		dx, dy = dx*most/d, dy*most/d
	}
	g.camX, g.camY = g.camX+dx, g.camY+dy
}

// changesTime reports whether a group of steps turns the time of day: the scene goes dark first and opens on it.
func changesTime(group []scenes.Step) bool {
	for _, st := range group {
		if st.Time != "" {
			return true
		}
	}
	return false
}

// startStep starts one step of a beat.
func (g *Game) startStep(st scenes.Step) {
	s := g.scene
	v := s.actors[st.Actor]
	if st.IfWalking && (v == nil || v.walk == nil && len(s.paths[st.Actor]) == 0) {
		return
	}
	if st.Wait > 0 {
		s.wait = math.Max(s.wait, st.Wait)
	}
	if st.Surge != "" {
		g.beginSceneSuigong(st) // sets the wait for the whole show
	}
	if st.Intro != "" {
		g.startIntro(st.Intro)
		s.wait = math.Max(s.wait, g.intro.length)
	}
	g.stageStep(st)
	if st.Strike != "" {
		g.startStrike(st)
	}
	if st.Effect == "halberd_throw" { // the beats wait until it has landed
		s.wait = math.Max(s.wait, halberdThrow())
	}
	if st.Shoot != nil {
		g.startShot(v, st)
	}
	if st.Shake > 0 {
		s.shake = math.Max(s.shake, st.Shake)
	}
	if v == nil {
		return
	}
	if !s.extra[st.Actor] && slices.Contains(s.def.Focus, s.node) && (len(st.Enter) == 2 || len(st.Move) == 2 || len(st.Place) == 2 || st.Strike != "" || st.Emote != "" ||
		st.Say != "" || st.Play != "") {
		s.focus = st.Actor
	}
	if len(st.Enter) == 2 || len(st.Move) == 2 || st.Exit {
		g.walkStep(v, st)
	}
	g.meleeStep(v, st)
	if st.Stop {
		s.paths[v.id] = nil
		delete(s.free, v.id)
	}
	if len(st.Place) == 2 {
		v.u, v.v, v.walk, v.gone, v.leave, v.arriving = st.Place[0], st.Place[1], nil, false, nil, 0 // an actor that left stands again
		s.paths[v.id] = nil
		v.lift = g.field.Lift(int(v.u), int(v.v))
		v.play(v.rest())
	}
	if st.Face != "" {
		v.dir, s.held[st.Actor] = st.Face, true
	}
	if st.Look != "" {
		back := v.dir
		if l := s.looks[st.Actor]; l != nil {
			back = l.back
		}
		v.dir, s.looks[st.Actor], s.held[st.Actor] = st.Look, &look{back: back, t: lookTime}, true
	}
	if anim := motion(v.art, st.Play); anim != "" && s.tied[st.Actor] == nil && !v.fallen() {
		v.playQuick(anim, max(1, st.Tempo))
		g.sfxAtCell(g.sceneMotionCue(v, anim), v.u, v.v)
		if !v.art.Animations[anim].Loop {
			s.plays[v.id] = anim
		}
		if len(st.Arrow) == 2 {
			g.looseArrow(v, anim, st.Arrow[0], st.Arrow[1], st.FireArrow)
		}
	}
	if st.Emote != "" {
		s.emotes[v.id] = &emote{text: st.Emote, life: emoteTime}
		g.sfx(emoteCues[st.Emote])
	}
	if st.Say != "" {
		s.emotes[v.id] = &emote{text: st.Say, life: 1.2 + .07*float64(len([]rune(st.Say))), block: true}
	}
	if st.Dead {
		v.killed = true
	}
	if st.Pose != "" && s.tied[st.Actor] == nil {
		v.pose = strings.TrimPrefix(st.Pose, "idle")
		if v.walk == nil && len(s.paths[v.id]) == 0 {
			v.play(v.rest())
		}
	}
	if len(st.Patrol) == 2 {
		s.patrol[v.id] = &patrol{ends: [2][2]float64{{st.Patrol[0], st.Patrol[1]}, {v.u, v.v}}}
	}
	switch st.Chatter {
	case "":
	case "-":
		delete(s.chatter, v.id)
	default:
		s.chatter[v.id] = &chatterLine{id: v.id, lines: strings.Split(st.Chatter, "|"), t: chatStart(st, len(s.chatter))}
	}
}

// stageStep does the parts of a step that change the stage rather than an actor: effects, props, time, camera,
// a new sprite.
func (g *Game) stageStep(st scenes.Step) {
	s := g.scene
	v := s.actors[st.Actor]
	switch {
	case st.Effect == "":
	case st.Effect == "none":
		s.effects = nil
	case len(st.At) == 2:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, u: st.At[0], v: st.At[1], hit: -1})
	case v != nil:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, u: v.u, v: v.v, hit: -1})
	default:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, hit: -1})
	}
	if st.Effect != "" && g.field != nil {
		g.field.night.fires = g.field.night.fires[:0]
		for _, e := range s.effects {
			if e.kind == "fire" || e.kind == "smallfire" || e.kind == "bigfire" || e.kind == "blaze" {
				g.field.night.fires = append(g.field.night.fires, nightFire{e.u, e.v, e.kind == "bigfire" || e.kind == "blaze"})
			}
		}
	}
	if len(st.Prop) == 2 { // [name, actor]: the prop already down moves with that actor from here on
		name, _ := st.Prop[0].(string)
		actor, _ := st.Prop[1].(string)
		if a := s.actors[actor]; a != nil {
			for i, p := range s.props {
				if p.name == name && p.follow == "" {
					s.props[i] = sceneProp{name, p.u - a.u, p.v - a.v, actor}
				}
			}
		}
	} else if len(st.Prop) > 0 {
		name, _ := st.Prop[0].(string)
		kept := s.props[:0]
		for _, p := range s.props {
			if p.name != name {
				kept = append(kept, p)
			}
		}
		s.props = kept
		if len(st.Prop) == 3 {
			u, _ := st.Prop[1].(float64)
			w, _ := st.Prop[2].(float64)
			s.props = append(s.props, sceneProp{name: name, u: u, v: w})
		}
	}
	if st.Time != "" {
		t := st.Time
		if t == "day" {
			t = ""
		}
		g.setSceneTime(t)
	}
	switch {
	case st.Follow == "-":
		s.follow = ""
	case st.Follow != "":
		s.follow = st.Follow
	}
	if st.Follow != "" || st.Cam != "" || len(st.CamAt) == 2 { // the scene frames its own shot from here
		s.focus = ""
	}
	if a := s.actors[st.Cam]; a != nil {
		s.camU, s.camV = a.u, a.v
		s.zoneHeld = false
	} else if len(st.CamAt) == 2 {
		s.camU, s.camV = st.CamAt[0], st.CamAt[1]
		s.zoneHeld = false
	}
	if st.Zoom >= 1 && g.exploring() { // a story scene keeps its fixed zoom
		g.setZoomImmediate(math.Round(st.Zoom))
	}
	if st.Bound != nil {
		g.tie(st.Actor, *st.Bound)
	}
	if v != nil && st.Sprite != "" {
		a := s.def.Cast[st.Actor]
		if art := g.actorArt(st.Actor, st.Sprite, a.Faction, a.Mount); art != nil {
			if s.tied[st.Actor] != nil {
				s.tied[st.Actor], art = art, boundArt(art)
			}
			v.art, v.top = art, art.TopOffset()
			v.play(v.rest())
		}
	}
}

// tie binds an actor with the prisoner's rope or unties him: bound, he walks and turns but never gestures, bows
// or plays a motion, and stands rather than resting in a pose.
func (g *Game) tie(key string, bound bool) {
	s := g.scene
	v := s.actors[key]
	free, tied := s.tied[key]
	switch {
	case v == nil:
	case bound && !tied:
		if s.tied == nil {
			s.tied = map[string]*sprite.Art{}
		}
		s.tied[key], v.art, v.pose = v.art, boundArt(v.art), ""
		v.play(v.rest())
	case !bound && tied:
		v.art = free
		delete(s.tied, key)
	}
}

// crowded tells whether another actor stands on a cell, for a route that keeps clear of the others.
func (s *scene) crowded(key string) func(u, v float64) bool {
	return func(u, v float64) bool {
		for k, o := range s.actors {
			if k != key && !o.gone && o.leave == nil && math.Hypot(o.u-u, o.v-v) < .6 {
				return true
			}
		}
		return false
	}
}

// sceneStep is the map step of a facing, for an actor walking on past the end of its path.
var sceneStep = map[string][2]float64{"SE": {1, 0}, "SW": {0, 1}, "NW": {-1, 0}, "NE": {0, -1},
	"E": {.7, -.7}, "S": {.7, .7}, "W": {-.7, .7}, "N": {-.7, -.7}}

// walkStep puts an entering actor on the stage and sends it along its walk.
func (g *Game) walkStep(v *unitVis, st scenes.Step) {
	s := g.scene
	if len(st.Enter) == 2 {
		v.u, v.v, v.gone, v.walk, v.leave, v.arriving = st.Enter[0], st.Enter[1], false, nil, nil, leaveTime
		v.lift = g.field.Lift(int(v.u), int(v.v))
	}
	// a walk backing away keeps the way it faced (Back)
	facing := v.dir
	if len(st.Move) == 2 && st.Ease != "" { // one stroke straight there, at its own pace
		delete(s.patrol, v.id)
		tu, tv := st.Move[0], st.Move[1]
		speed := st.Speed
		if speed <= 0 {
			speed = 1
		}
		g.sceneFootfall(v.id, v, [2]float64{tu, tv})
		anim, at, quick := v.anim, v.animT, v.quick
		g.walkTo(v, tu, tv)
		if s.swing[v.id] != "" && anim == s.swing[v.id] { // a blow under way plays on
			v.anim, v.animT, v.quick = anim, at, quick
		}
		v.walk.dur = math.Max(.01, math.Hypot(tu-v.u, tv-v.v)*stepTime/speed)
		switch st.Ease {
		case "in":
			v.walk.accel = .3
		case "out":
			v.walk.ease = true
		}
		s.paths[v.id] = nil
		s.held[v.id] = true
	} else if len(st.Move) == 2 {
		delete(s.patrol, v.id)
		u, w, path := v.u, v.v, [][2]float64{}
		for _, c := range append(st.Via, st.Move) {
			path = append(path, scenePath(u, w, c[0], c[1])...)
			u, w = c[0], c[1]
		}
		s.paths[v.id] = path
		s.held[v.id] = true
	}
	if st.Speed > 0 && st.Ease == "" {
		s.speed[v.id] = st.Speed
	}
	s.exits[v.id] = st.Exit
	if st.Back && len(st.Move) == 2 {
		v.dir = facing
		s.back[v.id] = facing
	} else if len(st.Move) == 2 { // a walk of its own faces the way it goes again
		delete(s.back, v.id)
	}
	if s.free == nil {
		s.free = map[string]bool{}
	}
	s.free[v.id] = st.Free
}

// settleNode puts the stage where a node's beats leave it, at once: for a node entered in the middle of a scene.
func (g *Game) settleNode(node string) {
	s := g.scene
	for _, line := range s.def.Beats[node] {
		for _, group := range line {
			for _, st := range group {
				g.stageStep(st)
				v := s.actors[st.Actor]
				if v == nil {
					continue
				}
				if len(st.Enter) == 2 {
					v.u, v.v, v.gone, v.leave, v.arriving = st.Enter[0], st.Enter[1], false, nil, 0
				}
				if len(st.Move) == 2 {
					v.u, v.v = st.Move[0], st.Move[1]
				}
				if st.Exit {
					v.gone = true
				}
				if st.Face != "" {
					v.dir = st.Face
				}
				if st.Dead {
					v.killed, v.grey = true, 1
				}
				if st.Pose != "" && s.tied[st.Actor] == nil {
					v.pose = strings.TrimPrefix(st.Pose, "idle")
					v.play(v.rest())
				}
				v.lift = g.field.Lift(int(v.u), int(v.v))
			}
		}
	}
}

// motion is the animation a scene step asks for: the sprite's own, or its one-shot pose
// for the old non-combat motion names. A sprite without that pose just stays idle.
func motion(art *sprite.Art, play string) string {
	if _, ok := art.Animations[play]; ok && play != "" {
		return play
	}
	switch play {
	case "salute", "toast", "surprise", "nod", "talk", "exhausted":
		if _, ok := art.Animations["action"]; ok {
			return "action"
		}
	}
	return ""
}

// scenePath walks straight legs, along u then along v, one cell per step; the last step of
// a leg covers what is left when the target lies between cells.
func scenePath(u, v, tu, tv float64) [][2]float64 {
	path := [][2]float64{}
	for u != tu {
		u += math.Copysign(math.Min(1, math.Abs(tu-u)), tu-u)
		path = append(path, [2]float64{u, v})
	}
	for v != tv {
		v += math.Copysign(math.Min(1, math.Abs(tv-v)), tv-v)
		path = append(path, [2]float64{u, v})
	}
	return path
}

// skipBeats finishes the running beats at once: walkers arrive, motions end.
func (g *Game) skipBeats() {
	s := g.scene
	if g.intro != nil && s != nil { // a click ends an officer's introduction only
		g.intro, s.wait = nil, 0
		return
	}
	for s != nil && g.sceneBusy() {
		if !s.begun {
			for _, st := range s.groups[s.gi] {
				g.startStep(st)
			}
			if g.intro != nil { // skipping stops at an officer's first appearance
				s.begun = true
				return
			}
		} else {
			for _, st := range s.pending {
				g.startStep(st)
			}
		}
		for _, k := range s.strikes { // a blow being dealt lands at once
			g.updateStrike(k, strikeLength(k))
			if v := s.actors[k.by]; v != nil {
				v.charging = false
			}
		}
		s.strikes = nil
		for key, v := range s.actors {
			if p := s.paths[key]; len(p) > 0 {
				last := p[len(p)-1]
				if _, backing := s.back[key]; !backing {
					v.face(last[0], last[1])
				}
				v.u, v.v = last[0], last[1]
			} else if v.walk != nil {
				v.u, v.v = v.walk.tu, v.walk.tv
			}
			if s.exits[key] || v.leave != nil {
				v.gone = true
			}
			v.walk, v.arriving, v.lift = nil, 0, g.field.Lift(int(v.u), int(v.v))
			v.play(v.rest())
		}
		for k, e := range s.emotes {
			if e.block {
				delete(s.emotes, k)
			}
		}
		s.paths, s.plays, s.exits, s.speed, s.pending, s.wait = map[string][][2]float64{}, map[string]string{}, map[string]bool{}, map[string]float64{}, nil, 0
		g.skipShot()
		s.gi++
		s.begun = false
	}
}

// faceSpeaker turns the speaker toward the one they speak to, and the named actors of the cast near the speaker, or
// spoken to however far, toward the speaker, unless the cast holds them or this node's beats already turned them. A
// speaker who speaks to no one turns to the nearest of them. People of a walk who stand by an event without being in
// its cast take no part.
func (g *Game) faceSpeaker(name string) {
	s := g.scene
	sp := s.actors[name]
	if sp == nil {
		return
	}
	free := func(key string, v *unitVis) bool {
		return !v.gone && !s.extra[key] && !s.hold[key] && !s.held[key] && v.walk == nil && s.looks[key] == nil
	}
	to := s.actors[g.addressee(name)]
	var near *unitVis
	best := 5.0
	for key, v := range s.actors {
		if _, cast := s.def.Cast[key]; v == sp || v.gone || s.extra[key] || !cast {
			continue
		}
		d := math.Hypot(v.u-sp.u, v.v-sp.v)
		if d >= 5 && v != to {
			continue
		}
		if d < best {
			near, best = v, d
		}
		if free(key, v) {
			v.face(sp.u, sp.v)
		}
	}
	if to == nil {
		to = near
	}
	if to != nil && free(name, sp) {
		sp.face(to.u, to.v)
	}
}

// addressee is the cast key of the one a speaker's line in the scene is said to: the last other speaker on the stage
// to speak before it, or, on the scene's first lines, the first to answer; "" when no one else on the stage speaks.
func (g *Game) addressee(name string) string {
	s := g.scene
	at := slices.Index(s.def.Nodes, s.node)
	if at < 0 || g.S == nil {
		return ""
	}
	other := func(id string) string {
		n := g.S.Data.Node(id)
		if n.Kind != "say" {
			return ""
		}
		for _, sid := range strings.Split(n.Speaker, "·") {
			key := g.S.Data.Characters[sid].Name
			_, cast := s.def.Cast[key]
			if v := s.actors[key]; key != name && cast && v != nil && !v.gone {
				return key
			}
		}
		return ""
	}
	for i := at - 1; i >= 0; i-- {
		if key := other(s.def.Nodes[i]); key != "" {
			return key
		}
	}
	for _, id := range s.def.Nodes[at+1:] {
		if key := other(id); key != "" {
			return key
		}
	}
	return ""
}

// drawScene draws the scene map and its actors, the speakers' bubbles and emotes; the caller darkens it by the fade.
func (g *Game) drawScene(dst *ebiten.Image, speakers []string) {
	g.drawSceneMap(dst, speakers)
	g.drawBalloons(dst, speakers)
}

// drawSceneMap draws the scene map and its actors, props and effects, without the balloons: the walk draws its
// name plates before them, so that no plate covers a balloon.
func (g *Game) drawSceneMap(dst *ebiten.Image, speakers []string) {
	s := g.scene
	actors := make([]*unitVis, 0, len(s.actors))
	for _, key := range slices.Sorted(maps.Keys(s.actors)) {
		actors = append(actors, s.actors[key])
	}
	// overlapping people keep one order from frame to frame: the lower on screen later, then the one further right
	slices.SortStableFunc(actors, func(a, b *unitVis) int {
		ax, ay := g.field.Center(a.u, a.v)
		bx, by := g.field.Center(b.u, b.v)
		return cmp.Or(cmp.Compare(ay, by), cmp.Compare(ax, bx))
	})
	talking := map[string]bool{}
	if !g.sceneBusy() {
		for _, name := range speakers {
			talking[name] = true
		}
	}
	for name, v := range s.actors { // a speaker turns and, standing, gestures once when their line comes up
		if talking[name] && !s.spoke[name] && v.walk == nil && v.anim == v.rest() && !v.gone {
			s.spoke[name] = true
			if slices.Contains(s.def.Focus, s.node) {
				s.focus = name
			}
			g.faceSpeaker(name)
			if v.anim == "idle" && s.tied[name] == nil {
				v.play("action")
			}
		}
	}
	g.field.beatShadows = g.field.beatShadows[:0]
	for _, p := range s.props {
		if u, v, ok := p.at(s); ok && g.assets.Props[p.art()].Shadow.Image != nil {
			sh := g.assets.Props[p.art()].Shadow
			x, y := g.field.Center(u, v)
			g.field.beatShadows = append(g.field.beatShadows, beatShadow{x, y, sh})
		}
	}
	g.field.setGoals(nil)
	// the field draws only what is in view, at its own scale, on a canvas just the size of the view laid on whole map
	// pixels; enlarged afterwards as the whole map's canvas was, the picture is the same, and a big map costs no more
	vp := g.battleViewport()
	hw, hh := float64(vp.Dx())/2/g.zoom, float64(vp.Dy())/2/g.zoom
	rx, ry := math.Floor(g.camX-hw)-2, math.Floor(g.camY-hh)-2
	g.field.prepareRender(1, rx, ry, roundUp(int(2*hw)+6, 64), roundUp(int(2*hh)+6, 64))
	g.field.DrawBounds = image.Rect(int(g.camX-hw)-64, int(g.camY-hh)-96, int(g.camX+hw)+64, int(g.camY+hh)+96)
	g.field.night.extra = g.arrowLights(g.field.night.extra[:0])
	g.field.Draw(g.clockMS, actors, nil)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(rx-g.camX, ry-g.camY)
	op.GeoM.Scale(g.zoom, g.zoom)
	op.GeoM.Translate(float64(vp.Min.X+vp.Dx()/2), float64(vp.Min.Y+vp.Dy()/2))
	if s.shake > 0 {
		op.GeoM.Translate(math.Round(math.Sin(g.clockMS/18)*6*math.Min(1, s.shake*4)), 0)
	}
	op.Filter = ebiten.FilterPixelated
	inWorld(func() {
		rect(dst, 0, 0, float32(worldW), float32(worldH), color.RGBA{68, 27, 13, 255})
		drawScaled(dst, g.field.Canvas, op)
		for _, p := range s.props {
			if pr, ok := g.assets.Props[p.art()]; ok {
				u, v, shown := p.at(s)
				if !shown {
					continue
				}
				x, y := g.field.Center(u, v)
				g.drawCanvasSprite(dst, pr.Sprite, x, y, 1)
				if a := s.actors[p.follow]; a != nil && !a.gone { // whoever rides in it stands in it, not under it
					anim, frame := g.field.pose(a)
					ax, ay := g.field.Center(a.u, a.v)
					ay -= a.lift + g.field.groundLift(a.u, a.v)
					g.drawCanvasSprite(dst, Sprite{a.art.Frame(a.dir, anim, frame), a.art.Pivot[0], a.art.Pivot[1]}, ax, ay, 1)
				}
			}
		}
		g.drawSceneEffects(dst)
		g.drawDust(dst)
		g.drawArrows(dst)
		g.drawEffects(dst)
		g.drawShot(dst)
		g.drawGlow(dst)
		if a := g.sceneFlash(); a > 0 {
			rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{255, 255, 255, uint8(255 * a)})
		}
		g.drawShotScreen(dst)
	})
}

// drawBalloons draws the speakers' dots, emotes and lines over their heads.
func (g *Game) drawBalloons(dst *ebiten.Image, speakers []string) {
	s := g.scene
	talking := map[string]bool{}
	if !g.sceneBusy() {
		for _, name := range speakers {
			talking[name] = true
		}
	}
	type balloon struct {
		name, mark string
		v          *unitVis
		talking    bool
	}
	var balloons []balloon
	for name := range talking {
		if v := s.actors[name]; v != nil && !v.gone && s.emotes[name] == nil && !slices.Contains(s.def.Mute, s.node) {
			balloons = append(balloons, balloon{name, "", v, true})
		}
	}
	for name, e := range s.emotes {
		if v := s.actors[name]; v != nil && !v.gone {
			balloons = append(balloons, balloon{name, e.text, v, false})
		}
	}
	// the lower a balloon's tail points, the later it is drawn, so overlapping ones keep one order
	slices.SortFunc(balloons, func(a, b balloon) int {
		ax, ay := g.field.Center(a.v.u, a.v.v)
		bx, by := g.field.Center(b.v.u, b.v.v)
		return cmp.Or(cmp.Compare(ay-a.v.lift-a.v.top, by-b.v.lift-b.v.top), cmp.Compare(ax, bx), strings.Compare(a.name, b.name))
	})
	for _, b := range balloons {
		g.bubble(dst, b.v, b.mark, b.talking)
	}
}

// glowImage is a corner light fading out along the diagonal, white at the top left corner.
var glowImage = func() *ebiten.Image {
	const n = 128
	im := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			d := math.Min(1, math.Hypot(float64(x), float64(y))/n)
			a := uint8(255 * math.Pow(1-d, 2.5))
			im.SetRGBA(x, y, color.RGBA{a, a, a, a})
		}
	}
	return ebiten.NewImageFromImage(im)
}()

// drawGlow washes the screen with the scene's far light: for fire_nw, a red glow from the top left corner (the
// north-west) that swells and flickers like a city burning beyond the horizon.
func (g *Game) drawGlow(dst *ebiten.Image) {
	if g.scene.def.Glow != "fire_nw" {
		return
	}
	t := g.clockMS / 1000
	k := .7 + .18*math.Sin(t*1.3) + .08*math.Sin(t*3.7+1) + .04*math.Sin(t*9.1+2)
	reach := float64(worldW) * (.6 + .05*math.Sin(t*.8))
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	op.GeoM.Scale(reach/128, reach/128)
	op.ColorScale.Scale(float32(.6*k), float32(.13*k), float32(.03*k), 1)
	dst.DrawImage(glowImage, op)
}

// bubbleRect is where the balloon of text over actor v lies on the UI, before the screen's edges push it in.
func (g *Game) bubbleRect(v *unitVis, text string) image.Rectangle {
	x, y := g.field.Center(v.u, v.v)
	sx, sy := worldToUI(g.toScreen(x, y-v.lift-g.field.groundLift(v.u, v.v)-v.top))
	k := g.bubbleScale()
	w, h := g.assets.Bubble.Size("", text)
	w, h = w*k, h*k
	return image.Rect(int(sx-28*k), int(sy-h), int(sx-28*k+w), int(sy))
}

// bubbleScale is the size of the balloons over people: drawn for a story scene's zoom 2, half that for the title's
// shots seen at zoom 1.
func (g *Game) bubbleScale() float64 {
	if g.reelPlaying() {
		return .5
	}
	return 1
}

// bubble is a speech balloon above the actor's head: animated dots while talking, or an
// emote mark or a short line.
func (g *Game) bubble(dst *ebiten.Image, v *unitVis, mark string, talking bool) {
	if talking {
		mark = strings.Repeat("•", 1+int(g.clockMS/300)%3)
	}
	x, y := g.field.Center(v.u, v.v)
	sx, sy := worldToUI(g.toScreen(x, y-v.lift-g.field.groundLift(v.u, v.v)-v.top))
	k := g.bubbleScale()
	w, h := g.assets.Bubble.Size("", mark)
	w, h = w*k, h*k
	// a speaker off the screen talks from its edge, the tail turned toward where they stand; above the dialogue box,
	// or on the walk, where no box is open, from the screen's bottom
	const margin = 12.0
	floor := dialogueBox().top
	if w := g.explore; w != nil && w.talk == nil {
		floor = float64(uiH)
	}
	bx := max(margin, min(float64(uiW)-margin-w, sx-28*k))
	by := max(margin, min(floor-margin-h, sy-h))
	tail := duelview.Tail{Up: by > sy}
	if bx != sx-28*k {
		tail.X = (sx - bx) / k
	}
	g.assets.Bubble.DrawText(dst, "", mark, tail, bx/k, by/k, drawPx*k) // DrawText scales its corner along with the balloon
}

// drawSceneEffects draws the stage effects: peach petals falling all around the camera,
// fires, small fires, smoke and steam at map positions. Particles are pure functions of the
// effect's age (and the camera, for the petals), so skipping beats or replaying a line shows the same thing.
// Effects on raised ground rise with it, like the people standing there.
func (g *Game) drawSceneEffects(dst *ebiten.Image) {
	for _, e := range g.scene.effects {
		x, y := g.field.Center(e.u, e.v)
		y -= g.field.Lift(int(e.u), int(e.v)) + g.field.groundLift(e.u, e.v) // a fire or smoke on raised ground burns on its top
		switch e.kind {
		case "petals":
			g.drawPetals(dst, e.t)
		case "fire", "blaze": // a blaze is a fire built up high in a brazier: no bigger, but it lights as far as a burning house
			g.flames(dst, x, y, e.t, true)
		case "smallfire":
			g.flames(dst, x, y, e.t, false)
		case "bigfire":
			g.flames(dst, x-12, y-4, e.t, true)
			g.flames(dst, x+12, y+2, e.t+.4, true)
			g.flames(dst, x, y+8, e.t+.8, true)
			g.smoke(dst, x, y-10, e.t, 3, 1)
		case "smoke":
			g.smoke(dst, x, y, e.t, 3, 1)
		case "steam":
			g.smoke(dst, x, y+20, e.t, 1, .45)
		case "birds":
			g.birds(dst, x, y, e.t)
		case "halberd":
			g.drawPlantedHalberd(dst, x, y, e.t)
		case "halberd_throw":
			g.drawThrownHalberd(dst, x, y, e.t, e.hit)
		case "glint":
			for i := 0; i < 3; i++ {
				if k := e.t - float64(i)*.45; k >= 0 && k < .3 {
					if p, ok := g.assets.Props[fmt.Sprintf("glint_%d", min(2, int(k/.1)))]; ok {
						g.drawCanvasSprite(dst, p.Sprite, x+float64(i*9-9), y-float64(i%2*6), 1)
					}
				}
			}
		}
	}
}

// flames draws a campfire standing on canvas point (x, y): a flickering glow, the scene's flame
// tongues in a small ring (or one small tongue) and smoke puffs rising above.
func (g *Game) flames(dst *ebiten.Image, x, y, t float64, big bool) {
	z := g.zoom
	glow := g.assets.effect("campfire_glow")
	glow.drawAlpha(dst, 0, x, y-8, g.toScreen, z, .7+.2*math.Sin(t*9))
	tall, small := g.assets.Anims["flame"], g.assets.Anims["flame_small"]
	spots := [][3]float64{{-6, -2, 0}, {6, -2, 0}, {0, -3, 1}, {-3, 2, 1}, {4, 2, 1}} // back ones first
	if !big {
		spots = [][3]float64{{-2, 0, 0}, {2, 1, 0}}
	}
	for i, spot := range spots {
		strip := tall
		if spot[2] == 0 {
			strip = small
		}
		if len(strip.Frames) > 0 {
			f := strip.Frames[(int(t*1000/float64(strip.MS))+i*2)%len(strip.Frames)]
			g.drawCanvasSprite(dst, f, x+spot[0], y+spot[1], 1)
		}
	}
	if big {
		g.smoke(dst, x, y, t, 2, 1)
	}
}

// birds draws a flock flushed from cover at canvas point (x, y): seven birds flapping up and away for three seconds.
func (g *Game) birds(dst *ebiten.Image, x, y, t float64) {
	if t > 3 {
		return
	}
	for i := 0; i < 7; i++ {
		k := float64(i)
		d := math.Max(0, t-k*.08)
		bx := x + math.Sin(k*2.3)*10 - d*(28+k*4)
		by := y - 10 - d*(34+math.Mod(k*5.7, 12)) - math.Sin(d*9+k)*2
		if p, ok := g.assets.Props[fmt.Sprintf("bird_%d", (int(d*12)+i)%3)]; ok {
			g.drawCanvasSprite(dst, p.Sprite, bx, by, math.Min(1, (3-t)*2))
		}
	}
}

// smoke draws n puffs rising from canvas point (x, y) at opacity a.
func (g *Game) smoke(dst *ebiten.Image, x, y, t float64, n int, a float64) {
	puff := g.assets.Anims["smoke"]
	for i := 0; i < n && len(puff.Frames) > 0; i++ {
		life := math.Mod(t*.5+float64(i)/float64(n), 1)
		f := puff.Frames[min(len(puff.Frames)-1, int(life*float64(len(puff.Frames))))]
		g.drawCanvasSprite(dst, f, x+math.Sin(float64(i)*1.7+t)*6, y-26-life*30, a*(1-life*.5))
	}
}

// drawCanvasSprite draws s with its pivot at canvas point (x, y) at the camera's zoom and opacity a,
// snapped to the canvas pixel grid.
func (g *Game) drawCanvasSprite(dst *ebiten.Image, s Sprite, x, y, a float64) {
	px, py := g.toScreen(math.Round(x-s.PivotX), math.Round(y-s.PivotY))
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(a))
	op.GeoM.Scale(g.zoom, g.zoom)
	op.GeoM.Translate(math.Round(px), math.Round(py))
	drawScaled(dst, s.Image, op)
}

// AddScenes reads one more scene file, for a trial chapter staged outside assets/scenes (cmd/_scene15).
func (g *Game) AddScenes(path string) error {
	if g.scenes == nil {
		return fmt.Errorf("no scene book")
	}
	return g.scenes.Add(path)
}

// RestartScene drops the running scene, so the next node opens its scene afresh (cmd/_scene15 jumping within a scene).
func (g *Game) RestartScene() {
	g.scene, g.shownNode, g.intro = nil, "", nil
}

// roundUp is n rounded up to a multiple of step: a canvas sized so does not change with every small zoom or resize.
func roundUp(n, step int) int { return (n + step - 1) / step * step }
