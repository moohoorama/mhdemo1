package gui

import (
	"cmp"
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/duelview"
	"srpg/internal/scenes"
	"srpg/internal/sprite"
)

// scene is the running scene (assets/scenes/*.json): one stretch of story nodes on one map. Its actors keep where
// the beats took them from node to node; a node entered in the middle of a stretch (a loaded game) first settles
// the beats of the nodes before it.
type scene struct {
	def     *scenes.Scene
	node    string
	actors  map[string]*unitVis
	extra   map[string]bool // cast keys with "#": no bubble when a name matches, never turned toward a speaker
	hold    map[string]bool // cast that keeps its facing while others speak
	held    map[string]bool // turned or moved by this node's beats: auto-facing leaves them alone
	groups  [][]scenes.Step
	gi      int
	begun   bool
	t       float64 // seconds since the running group started
	pending []scenes.Step
	wait    float64
	paths   map[string][][2]float64
	speed   map[string]float64
	exits   map[string]bool
	plays   map[string]string
	emotes  map[string]*emote
	looks   map[string]*look
	effects []sceneEffect
	petals  []petal // drawPetals' layout, kept from frame to frame
	props   []sceneProp
	spoke   map[string]bool
	shake   float64
	camU    float64
	camV    float64
	fade    float64                 // 1 → 0: the scene opening out of black
	patrol  map[string]*patrol      // sentries walking to and fro
	chatter map[string]*chatterLine // extras murmuring now and then
}

type patrol struct {
	ends [2][2]float64
	next int
}

type chatterLine struct {
	lines []string
	t     float64
	i     int
}

type sceneEffect struct {
	kind string
	u, v float64
	t    float64
}

type sceneProp struct {
	name string
	u, v float64
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
	emoteTime = 1.4
	lookTime  = .9
	camEase   = 4.0
	fadeTime  = .5
	chatEvery = 2.6
)

// enterScene sets up the scene of a dialogue node; false when the node has none.
func (g *Game) enterScene(o core.Observation) bool {
	def, at := g.scenes.For(o.Node)
	if def == nil || g.assets.Maps[def.Map] == nil {
		return false
	}
	if g.scene != nil && g.scene.node == o.Node {
		return true
	}
	if (g.scene == nil || g.scene.def != def || g.field == nil || g.field.Map.ID != def.Map) && !g.adoptScene(def) && !(g.layOutWalkFor(def) && g.adoptScene(def)) {
		g.openScene(def)
		for _, n := range def.Nodes[:at] {
			g.settleNode(n)
		}
	}
	s := g.scene
	s.node, s.held = o.Node, map[string]bool{}
	g.startLine(0)
	return true
}

// openScene builds the scene's map and cast, lights its ambient effects and frames the cast.
func (g *Game) openScene(def *scenes.Scene) {
	g.field = NewField(g.assets, g.assets.Maps[def.Map])
	s := &scene{def: def, actors: map[string]*unitVis{}, extra: map[string]bool{}, hold: map[string]bool{}, held: map[string]bool{},
		paths: map[string][][2]float64{}, speed: map[string]float64{}, exits: map[string]bool{}, plays: map[string]string{},
		emotes: map[string]*emote{}, looks: map[string]*look{}, fade: 1, patrol: map[string]*patrol{}, chatter: map[string]*chatterLine{}}
	g.scene = s
	for key, a := range def.Cast {
		if v := g.castActor(key, a); v != nil {
			s.actors[key] = v
			s.extra[key], s.hold[key] = scenes.Extra(key), a.Hold
		}
	}
	g.setSceneTime(def.Time)
	for _, st := range def.Ambient {
		g.startStep(st)
	}
	g.zoom = 2
	s.camU, s.camV = g.castCentre()
	g.camX, g.camY = g.sceneCam()
}

// castActor is a cast member standing where the cast puts it; nil without a place or a sprite.
func (g *Game) castActor(key string, a scenes.Actor) *unitVis {
	art := g.actorArt(key, a.Sprite, a.Faction)
	if art == nil || len(a.At) != 2 {
		return nil
	}
	v := &unitVis{id: key, ally: true, dir: a.Face, anim: "idle", dying: -1, shownHP: 1, maxHP: 1, actor: true, art: art, top: art.TopOffset(), pose: a.Pose}
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
	return x, y - 20
}

// setSceneTime darkens or tints the scene: night is the battle maps' night, dusk and dawn a lighter wash,
// memory an even faded brown for a scene remembered.
func (g *Game) setSceneTime(t string) {
	f := g.field
	f.setNight(t != "", nil)
	f.night.color = map[string]color.NRGBA{"night": nightColor, "dusk": {R: 70, G: 28, B: 12, A: 95}, "dawn": {R: 30, G: 40, B: 80, A: 85},
		"memory": {R: 112, G: 100, B: 84, A: 165}}[t]
	f.night.flat = t == "memory"
}

// actorArt is the sprite a cast member stands in: the cast's own pick (any sprite id: extras, soldiers, or a
// battle sprite), else the character's scene, battle or class sprite, in the character's faction colour unless
// the cast names another.
func (g *Game) actorArt(key, pick, faction string) *sprite.Art {
	def, c, _ := g.character(scenes.Base(key))
	if faction == "" {
		faction = c.Look.Faction
	}
	return g.sceneArt(def, pick, faction)
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
		v.update(dt * k)
		if p := s.paths[key]; v.walk == nil && len(p) > 0 {
			g.walkTo(v, p[0][0], p[0][1])
			s.paths[key] = p[1:]
		} else if v.walk == nil && len(p) == 0 {
			if pt := s.patrol[key]; pt != nil {
				s.paths[key] = scenePath(v.u, v.v, pt.ends[pt.next][0], pt.ends[pt.next][1])
				pt.next = 1 - pt.next
			} else if v.anim == "walk" && v.leave == nil {
				v.play(v.rest())
			}
			if s.exits[key] { // walks on past the end of its path and fades out, like a unit leaving a battle
				s.exits[key] = false
				d := sceneStep[v.dir]
				v.play("walk")
				v.leave = &walkState{fu: v.u, fv: v.v, tu: v.u + d[0]*leaveCells, tv: v.v + d[1]*leaveCells}
			}
			delete(s.speed, key)
		}
	}
	for key, l := range s.looks {
		if l.t -= dt; l.t <= 0 {
			s.actors[key].dir = l.back
			delete(s.looks, key)
		}
	}
	for i := range s.effects {
		s.effects[i].t += dt
	}
	for k, e := range s.emotes {
		if e.t += dt; e.t > e.life {
			delete(s.emotes, k)
		}
	}
	s.shake = math.Max(0, s.shake-dt)
	s.fade = math.Max(0, s.fade-dt/fadeTime)
	for key, c := range s.chatter {
		if c.t -= dt; c.t <= 0 && s.emotes[key] == nil && !s.actors[key].gone {
			s.emotes[key] = &emote{text: c.lines[c.i%len(c.lines)], life: emoteTime + .4}
			c.i++
			c.t = chatEvery + float64((len(key)*13+c.i*7)%10)/10
		}
	}
	tx, ty := g.sceneCam()
	ease := math.Min(1, dt*camEase)
	g.camX, g.camY = g.camX+(tx-g.camX)*ease, g.camY+(ty-g.camY)*ease
	if !g.sceneBusy() {
		return
	}
	if !s.begun {
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
	if s.wait > 0 || len(s.pending) > 0 {
		return
	}
	for key, v := range s.actors {
		if s.patrol[key] == nil && (len(s.paths[key]) > 0 || v.walk != nil) || s.plays[key] != "" && v.anim == s.plays[key] {
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

// startStep starts one step of a beat.
func (g *Game) startStep(st scenes.Step) {
	s := g.scene
	v := s.actors[st.Actor]
	if st.Wait > 0 {
		s.wait = math.Max(s.wait, st.Wait)
	}
	if st.Intro != "" {
		g.startIntro(st.Intro)
		s.wait = math.Max(s.wait, g.intro.length)
	}
	g.stageStep(st)
	if st.Shake > 0 {
		s.shake = math.Max(s.shake, st.Shake)
	}
	if v == nil {
		return
	}
	if len(st.Enter) == 2 || len(st.Move) == 2 || st.Exit {
		g.walkStep(v, st)
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
	if anim := motion(v.art, st.Play); anim != "" {
		v.play(anim)
		if !v.art.Animations[anim].Loop {
			s.plays[v.id] = anim
		}
	}
	if st.Emote != "" {
		s.emotes[v.id] = &emote{text: st.Emote, life: emoteTime}
	}
	if st.Say != "" {
		s.emotes[v.id] = &emote{text: st.Say, life: 1.2 + .07*float64(len([]rune(st.Say))), block: true}
	}
	if st.Pose != "" {
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
		s.chatter[v.id] = &chatterLine{lines: strings.Split(st.Chatter, "|"), t: float64(len(v.id)%7) / 4}
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
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, u: st.At[0], v: st.At[1]})
	case v != nil:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, u: v.u, v: v.v})
	default:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect})
	}
	if st.Effect != "" && g.field != nil {
		g.field.night.fires = g.field.night.fires[:0]
		for _, e := range s.effects {
			if e.kind == "fire" || e.kind == "smallfire" || e.kind == "bigfire" {
				g.field.night.fires = append(g.field.night.fires, nightFire{e.u, e.v, e.kind == "bigfire"})
			}
		}
	}
	if len(st.Prop) > 0 {
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
			s.props = append(s.props, sceneProp{name, u, w})
		}
	}
	if st.Time != "" {
		t := st.Time
		if t == "day" {
			t = ""
		}
		g.setSceneTime(t)
	}
	if a := s.actors[st.Cam]; a != nil {
		s.camU, s.camV = a.u, a.v
	} else if len(st.CamAt) == 2 {
		s.camU, s.camV = st.CamAt[0], st.CamAt[1]
	}
	if st.Zoom >= 1 {
		g.zoom = math.Round(st.Zoom)
	}
	if v != nil && st.Sprite != "" {
		a := s.def.Cast[st.Actor]
		if art := g.actorArt(st.Actor, st.Sprite, a.Faction); art != nil {
			v.art, v.top = art, art.TopOffset()
			v.play(v.rest())
		}
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
	if len(st.Move) == 2 {
		delete(s.patrol, v.id)
		s.paths[v.id] = scenePath(v.u, v.v, st.Move[0], st.Move[1])
		s.held[v.id] = true
	}
	if st.Speed > 0 {
		s.speed[v.id] = st.Speed
	}
	s.exits[v.id] = st.Exit
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
				if st.Pose != "" {
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
		for key, v := range s.actors {
			if p := s.paths[key]; len(p) > 0 {
				last := p[len(p)-1]
				v.face(last[0], last[1])
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
		s.gi++
		s.begun = false
	}
}

// faceSpeaker turns the named actors of the cast near a speaker toward them, and the speaker toward the nearest of
// them, unless the cast holds them or this node's beats already turned them. People of a walk who stand by an event
// without being in its cast take no part.
func (g *Game) faceSpeaker(name string) {
	s := g.scene
	sp := s.actors[name]
	if sp == nil {
		return
	}
	free := func(key string, v *unitVis) bool {
		return !v.gone && !s.extra[key] && !s.hold[key] && !s.held[key] && v.walk == nil && s.looks[key] == nil
	}
	var near *unitVis
	best := 5.0
	for key, v := range s.actors {
		if _, cast := s.def.Cast[key]; v == sp || v.gone || s.extra[key] || !cast {
			continue
		}
		d := math.Hypot(v.u-sp.u, v.v-sp.v)
		if d >= 5 {
			continue
		}
		if d < best {
			near, best = v, d
		}
		if free(key, v) {
			v.face(sp.u, sp.v)
		}
	}
	if near != nil && free(name, sp) {
		sp.face(near.u, near.v)
	}
}

// drawScene draws the scene map and its actors, the speakers' bubbles and emotes.
func (g *Game) drawScene(dst *ebiten.Image, speakers []string) {
	s := g.scene
	actors := make([]*unitVis, 0, len(s.actors))
	for _, v := range s.actors {
		actors = append(actors, v)
	}
	talking := map[string]bool{}
	if !g.sceneBusy() {
		for _, name := range speakers {
			talking[name] = true
		}
	}
	for name, v := range s.actors { // a speaker gestures once when their line comes up
		if talking[name] && !s.spoke[name] && v.walk == nil && v.anim == "idle" && !v.gone {
			s.spoke[name] = true
			g.faceSpeaker(name)
			v.play("action")
		}
	}
	g.field.beatShadows = g.field.beatShadows[:0]
	for _, p := range s.props {
		if sh := g.assets.Props[p.name].Shadow; sh.Image != nil {
			x, y := g.field.Center(p.u, p.v)
			g.field.beatShadows = append(g.field.beatShadows, beatShadow{x, y, sh})
		}
	}
	g.field.Draw(g.clockMS, actors, nil)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-g.camX, -g.camY)
	op.GeoM.Scale(g.zoom, g.zoom)
	vp := g.battleViewport()
	op.GeoM.Translate(float64(vp.Min.X+vp.Dx()/2), float64(vp.Min.Y+vp.Dy()/2))
	if s.shake > 0 {
		op.GeoM.Translate(math.Round(math.Sin(g.clockMS/18)*6*math.Min(1, s.shake*4)), 0)
	}
	op.Filter = ebiten.FilterPixelated
	inWorld(func() {
		rect(dst, 0, 0, float32(worldW), float32(worldH), fieldGround) // a wide screen sees past the canvas
		g.drawBackdrop(dst)
		drawScaled(dst, g.field.Canvas, op)
		for _, p := range s.props {
			if pr, ok := g.assets.Props[p.name]; ok {
				x, y := g.field.Center(p.u, p.v)
				g.drawCanvasSprite(dst, pr.Sprite, x, y, 1)
			}
		}
		g.drawSceneEffects(dst)
	})
	type balloon struct {
		name, mark string
		v          *unitVis
		talking    bool
	}
	var balloons []balloon
	for name := range talking {
		if v := s.actors[name]; v != nil && !v.gone && s.emotes[name] == nil {
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
	if s.fade > 0 {
		inWorld(func() { rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{A: uint8(255 * s.fade)}) })
	}
}

// drawBackdrop lays the scene's far picture (mountains, a pass, a burning city) across the top of the screen,
// behind the map, darkened with the scene's time of day.
func (g *Game) drawBackdrop(dst *ebiten.Image) {
	im := g.assets.Backdrops[g.scene.def.Backdrop]
	if im == nil {
		return
	}
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	k := float64(worldW) / float64(w)
	bottom := float64(worldH) * .42
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterPixelated}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(0, bottom-float64(h)*k)
	shade := float32(1)
	if n := g.field.night; n.on && !n.flat {
		dark := n.color
		if dark.A == 0 {
			dark = nightColor
		}
		shade = 1 - float32(dark.A)/255*.9
		op.ColorScale.Scale(shade, shade, shade*1.05, 1)
	}
	drawScaled(dst, im, op)
	// the near ground of the picture runs on down behind the map, so the backdrop has no bottom edge
	r, gr, b, _ := im.At(w/2, h-1).RGBA()
	ground := color.NRGBA{R: uint8(float32(r>>8) * shade), G: uint8(float32(gr>>8) * shade), B: uint8(math.Min(255, float64(float32(b>>8)*shade*1.05))), A: 255}
	rect(dst, 0, float32(bottom), float32(worldW), float32(worldH)-float32(bottom), ground)
}

// bubble is a speech balloon above the actor's head: animated dots while talking, or an
// emote mark or a short line.
func (g *Game) bubble(dst *ebiten.Image, v *unitVis, mark string, talking bool) {
	if talking {
		mark = strings.Repeat("•", 1+int(g.clockMS/300)%3)
	}
	x, y := g.field.Center(v.u, v.v)
	sx, sy := worldToUI(g.toScreen(x, y-v.lift-v.top))
	w, h := g.assets.Bubble.Size("", mark)
	// a speaker off the screen talks from its edge, the tail turned toward where they stand
	const margin = 12.0
	bx := max(margin, min(float64(uiW)-margin-w, sx-28))
	by := max(margin, min(dialogueBox().top-margin-h, sy-h))
	tail := duelview.Tail{Up: by > sy}
	if bx != sx-28 {
		tail.X = sx - bx
	}
	g.assets.Bubble.DrawText(dst, "", mark, tail, bx, by, drawPx)
}

// drawSceneEffects draws the stage effects: peach petals falling all around the camera,
// fires, small fires, smoke and steam at map positions. Particles are pure functions of the
// effect's age (and the camera, for the petals), so skipping beats or replaying a line shows the same thing.
func (g *Game) drawSceneEffects(dst *ebiten.Image) {
	for _, e := range g.scene.effects {
		x, y := g.field.Center(e.u, e.v)
		switch e.kind {
		case "petals":
			g.drawPetals(dst, e.t)
		case "fire":
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
	glow := g.assets.FX["campfire_glow"]
	glow.drawAlpha(dst, 0, x, y-8, g.toScreen, z, .7+.2*math.Sin(t*9))
	tall, small := g.assets.Flames["flame"], g.assets.Flames["flame_small"]
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
	puff := g.assets.Flames["smoke"]
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
