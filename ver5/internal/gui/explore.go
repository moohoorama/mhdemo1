package gui

import (
	"cmp"
	"fmt"
	"image"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
	"srpg/internal/session"
)

// exploreWalk is the story's walk between battles (an explore node of core): the hero walks an area of the explore
// file small tile by small tile where the player clicks, talks to whoever is clicked, reads signs and goes through
// exits. The area is a scene (scene.go) with no beats: the people standing there by the story's flags are its
// cast. Talking to someone with an event starts it on this very stage (adoptScene); afterwards the area is laid out
// again from the flags. Only where the hero stands is the story's (core walk commands): the rest is the screen's.
type exploreWalk struct {
	engine     *core.Engine // the game it was laid out for: a loaded game lays it out again
	node, area string
	hero       string            // the hero's cast key: his shown name
	people     map[string]string // cast key -> person key of the area
	grid       walkGrid
	aim        *exploreAim
	going      *content.ExploreExit // the exit the hero is going through, while the screen fades out
	fade       float64
	mark       smallTile // where the last click on the ground sent the hero
	markT      float64
	show       bool // R or a right click: show which small tiles can be walked
	talk       []line
	talkI      int
	talkTo     string            // whom the hero is talking to ("" for a sign or a shut exit)
	homes      map[string]string // the way each person faced before turning to the hero
	rest       map[string]float64
	leg        map[string]int // the patrol point each patroller walks to
	walked     bool           // the hero walked since the story was last told where he stands
	rng        *rand.Rand
	plates     []explorePlate // the name plates as last drawn, top last
}

// explorePlate is a name plate of the walk as last drawn (UI px) and what a click on it picks: a person (cast key)
// or an exit.
type explorePlate struct {
	rect   image.Rectangle
	person string
	exit   *content.ExploreExit
}

// exploreAim is what the hero is walking to, and what happens on arrival.
type exploreAim struct {
	person string // cast key
	exit   *content.ExploreExit
	sign   *content.ExploreSign
}

const walkerStep = 1.0 // the people's walking speed, in scene steps

// exploring reports whether the game stands at a walk. Not while a gift's or a joining officer's window holds the
// story line that came before: that line's scene stays on screen, and laying the walk out under it would take
// the scene away every frame.
func (g *Game) exploring() bool {
	return g.held == nil && g.S.Engine != nil && g.S.Screen == "playing" && g.S.Engine.Walking()
}

// ensureExplore lays the area out when the story came to it (or back to it from an event, or by loading).
func (g *Game) ensureExplore(o core.Observation) {
	w := g.explore
	if w != nil && w.engine == g.S.Engine && w.node == o.Node && w.area == o.Spot.Area && g.scene != nil && g.scene.def.ID == "explore:"+o.Spot.Area {
		return
	}
	g.layOutExplore(o)
}

// layOutExplore opens the hero's area as a scene: the hero where the story says he stands, the people whom the
// flags put there, the area's fires and murmurs.
func (g *Game) layOutExplore(o core.Observation) {
	e := g.S.Engine
	x := e.ExploreFile()
	a := x.Areas[o.Spot.Area]
	var left *unitVis // the hero as the event before left him, on this map
	fade := 1.0       // how dark the scene opens: as dark as the one it replaces on the same map, so that it neither flashes to black nor to light
	if g.field != nil && g.scene != nil && g.field.Map.ID == a.Map {
		left = g.scene.actors[g.S.Data.Characters[x.Hero].Name]
		fade = g.scene.fade
	}
	w := &exploreWalk{engine: e, node: o.Node, area: o.Spot.Area, hero: g.S.Data.Characters[x.Hero].Name, people: map[string]string{},
		homes: map[string]string{}, rest: map[string]float64{}, leg: map[string]int{}, markT: math.Inf(1), rng: rand.New(rand.NewPCG(7, 11))}
	def := g.exploreDef(e, o, w)
	var old map[string]*unitVis // the stage the walk takes over unchanged, when it is the same room as lit
	camX, camY, zoom := g.camX, g.camY, g.zoom
	if g.seamless(def, w.hero) {
		old = g.scene.actors
	}
	g.explore = w
	g.openScene(def)
	g.scene.fade = fade
	w.grid = walkGrid{walk: &scenemap.Walk{}}
	if art := g.mapArt(a.Map); art != nil && art.SceneMap != nil {
		w.grid.walk = scenemap.Walkable(art.SceneMap, g.assets.SceneRules)
		w.grid.walk.Within(a.At[0], a.At[1]) // the hero leaves by an exit, not by walking out of the gate
	} else {
		g.toast(a.Map+"은 형식 2 장면 맵이 아니라 걸을 수 없습니다", bad)
	}
	z := 2.0
	if a.Zoom >= 1 {
		z = math.Round(a.Zoom)
	}
	g.setZoomImmediate(g.savedZoom(true, z))
	s := g.scene
	hero := s.actors[w.hero]
	if left != nil && !left.gone && (left.u != hero.u || left.v != hero.v || left.dir != hero.dir) { // he stays where, and the way, the event left him
		hero.u, hero.v, hero.dir = left.u, left.v, left.dir
		w.walked = true
	}
	for key, v := range old { // everybody stays as they stood: the camera and the zoom glide to the walk's own
		if p := s.actors[key]; p != nil && key != w.hero {
			p.u, p.v, p.dir, p.lift = v.u, v.v, v.dir, v.lift
		}
	}
	if t := g.actorTile(w.hero); !w.grid.walk.Open(t[0], t[1]) { // an event may have left him where nobody walks
		hero.u, hero.v = w.grid.nearestOpen(t).cell()
		w.walked = true
	}
	s.camU, s.camV = hero.u, hero.v
	g.camX, g.camY = g.sceneCam()
	if old != nil {
		g.camX, g.camY = camX, camY
		target := g.zoom
		g.zoom, g.zoomFrom, g.zoomTo = zoom, zoom, zoom
		g.setZoom(target)
	}
}

// exploreDef is the scene the walk's area is as laid out for observation o: the hero where the story says he
// stands, the people whom the flags put there, the area's fires and murmurs. w takes the cast keys of the people.
func (g *Game) exploreDef(e *core.Engine, o core.Observation, w *exploreWalk) *scenes.Scene {
	a := e.ExploreFile().Areas[o.Spot.Area]
	def := &scenes.Scene{ID: "explore:" + o.Spot.Area, Map: a.Map, Time: e.AreaTime(), Grass: a.Grass, Snow: a.Snow, Cast: map[string]scenes.Actor{}}
	at := func(t [2]int) []float64 { u, v := smallTile(t).cell(); return []float64{u, v} }
	def.Cast[w.hero] = scenes.Actor{At: at(o.Spot.At), Face: o.Spot.Face}
	for _, v := range e.Visitors() {
		p := a.People[v.Key]
		key := content.PersonName(v.Key)
		if strings.Contains(v.Key, "#") {
			key = v.Key
		}
		def.Cast[key] = scenes.Actor{At: at(p.At), Face: p.Face, Faction: p.Faction, Sprite: p.Sprite, Pose: p.Pose}
		w.people[key] = v.Key
		if p.Chatter != "" {
			def.Ambient = append(def.Ambient, scenes.Step{Actor: key, Chatter: p.Chatter})
		}
		w.rest[key] = 1 + float64(len(key)%5)/2
	}
	for _, f := range a.Effects {
		def.Ambient = append(def.Ambient, scenes.Step{Effect: f.Effect, At: at(f.At)})
	}
	return def
}

// adoptSame is how far, in cells, a person may stand from the place a scene or the walk puts them at and still be
// taken for standing there: the walk's small tiles centre a person 0.2 cell off the scene's whole cells.
const adoptSame = .4

// seamless reports whether laying the walk out as def would change nothing on the stage now on screen: the same
// map, the same light, lit, and the same people standing where def puts them, the hero apart (he stays where he
// is). Then neither the screen darkens nor a person moves; anything else is a change of place or arrangement, which
// goes through black as before.
func (g *Game) seamless(def *scenes.Scene, hero string) bool {
	s := g.scene
	if g.field == nil || s == nil || s.leaving || s.nightfall || s.fade > .05 || g.field.Map.ID != def.Map || s.tod != def.Time {
		return false
	}
	base := s.base
	if base == nil || base.Snow != def.Snow || (base.Grass == nil) != (def.Grass == nil) || base.Grass != nil && *base.Grass != *def.Grass {
		return false
	}
	if v := s.actors[hero]; v == nil || v.gone {
		return false
	}
	for key, v := range s.actors {
		if _, in := def.Cast[key]; !in && !v.gone && key != hero {
			return false
		}
	}
	for key, a := range def.Cast {
		if key == hero {
			continue
		}
		v := s.actors[key]
		if v == nil || v.gone || v.leave != nil || len(a.At) != 2 || math.Hypot(a.At[0]-v.u, a.At[1]-v.v) > adoptSame || v.pose != a.Pose {
			return false
		}
	}
	return true
}

// walkSeamless reports whether the walk the story is about to go back to (after the nodes that set, join, grant or
// leave on the way) would be the stage on screen unchanged.
func (g *Game) walkSeamless() bool {
	e, err := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err != nil {
		return false
	}
	if _, err := e.Apply(core.Command{Kind: "next"}); err != nil || !e.Walking() {
		return false
	}
	w := &exploreWalk{hero: g.S.Data.Characters[e.ExploreFile().Hero].Name, people: map[string]string{}, rest: map[string]float64{}}
	return g.seamless(g.exploreDef(e, e.Observe(), w), w.hero)
}

// adoptScene stages the node at index at of an event's scene on the walk's own stage when the scene is set on the
// area's map: whoever the cast marks Stay (or leaves out) stays where the walk left them, the others go where the
// cast puts them as it stands at that node (an earlier node's beats having moved them). Somebody who is far from
// there walks over rather than jumping; someone already there stays as they stand.
func (g *Game) adoptScene(def *scenes.Scene, at int) bool {
	s := g.scene
	if def.Cut || g.explore == nil || s == nil || !strings.HasPrefix(s.def.ID, "explore:") || g.field.Map.ID != def.Map {
		return false
	}
	s.def, s.held = def, map[string]bool{}
	for key := range s.actors {
		if a, cast := def.Cast[key]; !cast || a.Stay {
			delete(s.patrol, key)
			s.paths[key] = nil
		}
	}
	placed := settledCast(def, at)
	for key, a := range def.Cast {
		v := s.actors[key]
		if a.Stay {
			continue
		}
		a = placed[key]
		walking := false
		if v == nil {
			if v = g.castActor(key, a); v == nil {
				continue
			}
			s.actors[key] = v
		} else if len(a.At) == 2 && !v.gone && !a.Off {
			walking = g.walkToPlace(key, v, a)
		} else if len(a.At) == 2 {
			v.u, v.v, v.walk, s.paths[key] = a.At[0], a.At[1], nil, nil
			v.lift = g.field.Lift(int(v.u), int(v.v))
		}
		if a.Face != "" && !walking {
			v.dir = a.Face
		}
		if a.Pose != "" {
			v.pose = a.Pose
			v.play(v.rest())
		}
		v.gone = a.Off
		s.extra[key], s.hold[key] = scenes.Extra(key), a.Hold
	}
	if def.Time != "" {
		g.setSceneTime(def.Time)
	}
	for _, st := range def.Ambient {
		g.startStep(st)
	}
	return true
}

// walkToPlace sends somebody standing on the stage to the place a cast puts them at: nowhere when they stand there
// already, else along a walk that turns them to the cast's facing on arrival. It reports whether they walk.
func (g *Game) walkToPlace(key string, v *unitVis, a scenes.Actor) bool {
	s := g.scene
	v.walk, s.paths[key] = nil, nil
	if math.Hypot(a.At[0]-v.u, a.At[1]-v.v) <= adoptSame {
		return false
	}
	s.paths[key] = scenePath(v.u, v.v, a.At[0], a.At[1])
	s.held[key], s.settle = true, true
	if a.Face != "" {
		if s.arrive == nil {
			s.arrive = map[string]string{}
		}
		s.arrive[key] = a.Face
	}
	return true
}

// settledCast is the scene's cast as it stands when its node at index at opens: each actor's place and facing from
// the cast, moved on by the beats of the nodes before it, as a scene entered in the middle settles them.
func settledCast(def *scenes.Scene, at int) map[string]scenes.Actor {
	cast := maps.Clone(def.Cast)
	for _, n := range def.Nodes[:min(at, len(def.Nodes))] {
		for _, line := range def.Beats[n] {
			for _, group := range line {
				for _, st := range group {
					a, ok := cast[st.Actor]
					if !ok {
						continue
					}
					if len(st.Enter) == 2 {
						a.At, a.Off = st.Enter, false
					}
					if len(st.Place) == 2 {
						a.At, a.Off = st.Place, false
					}
					if len(st.Move) == 2 {
						a.At = st.Move
					}
					if st.Exit {
						a.Off = true
					}
					if st.Face != "" {
						a.Face = st.Face
					}
					cast[st.Actor] = a
				}
			}
		}
	}
	return cast
}

// layOutWalkFor lays out the walk's area first when an event set on it is entered without it (a loaded game);
// false when the story is in no walk on that map.
func (g *Game) layOutWalkFor(def *scenes.Scene) bool {
	x := g.S.Engine.ExploreFile()
	if def.Cut || x == nil {
		return false
	}
	o := g.S.View()
	if x.Areas[o.Spot.Area].Map != def.Map {
		return false
	}
	g.layOutExplore(o)
	return true
}

// updateExplore steps the walk by one frame: the camera on the hero, the people walking about, the fade through an
// exit, what the hero arrives at.
func (g *Game) updateExplore() {
	g.ensureExplore(g.S.View())
	w := g.explore
	s := g.scene
	const dt = 1.0 / 60
	w.markT += dt
	hero := s.actors[w.hero]
	s.camU, s.camV = hero.u, hero.v
	g.blockPeople()
	if w.going != nil {
		if w.fade = math.Min(1, w.fade+dt/walkFade); w.fade >= 1 {
			to := content.Spot{At: w.going.At, Face: hero.dir} // an exit into an event: through it, out of the walk
			if w.going.Node == "" {
				to = g.S.Engine.ExploreFile().Arrival(w.going.To, w.area)
			}
			w.going = nil
			g.request(session.Request{Op: "command", Command: core.Command{Kind: "walk", Option: to.Area, X: to.At[0], Y: to.At[1], Target: to.Face}})
		}
		return
	}
	g.moveWalkers(dt)
	if hero.walk != nil || len(s.paths[w.hero]) > 0 {
		w.walked = true
		return
	}
	if w.aim != nil && w.talk == nil {
		g.exploreArrive()
		return
	}
	g.reportSpot()
}

// reportSpot tells the story where the hero stopped, once after each walk, so a save keeps it.
func (g *Game) reportSpot() {
	w := g.explore
	if !w.walked {
		return
	}
	w.walked = false
	t := g.actorTile(w.hero)
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "walk", Option: w.area, X: t[0], Y: t[1], Target: g.scene.actors[w.hero].dir}})
}

// blockPeople marks the small tiles where someone stands, or is stepping to, as taken.
func (g *Game) blockPeople() {
	w := g.explore
	w.grid.blocked = map[smallTile]bool{}
	for key, v := range g.scene.actors {
		if !v.gone {
			w.grid.blocked[g.actorTile(key)] = true
		}
	}
}

// moveWalkers sends the people who walk about on their next leg once they have stood a while: a patroller to the
// next point of its round, a wanderer to some open small tile within its reach; one who stays plays its motion
// again now and then.
func (g *Game) moveWalkers(dt float64) {
	w := g.explore
	s := g.scene
	a := g.S.Engine.ExploreFile().Areas[w.area]
	for _, key := range slices.Sorted(maps.Keys(w.people)) {
		p := a.People[w.people[key]]
		v := s.actors[key]
		if v == nil || v.walk != nil || len(s.paths[key]) > 0 || key == w.talkTo || (w.aim != nil && w.aim.person == key) {
			continue
		}
		if w.rest[key] -= dt; w.rest[key] > 0 {
			continue
		}
		if p.Move == nil {
			if p.Play != "" {
				g.startStep(scenes.Step{Actor: key, Play: p.Play})
				w.rest[key] = 2 + w.rng.Float64()
			}
			continue
		}
		from := g.actorTile(key)
		var route []smallTile
		if len(p.Move.Patrol) > 0 {
			points := append([][2]int{p.At}, p.Move.Patrol...)
			n := 2*len(points) - 2
			w.leg[key] = (w.leg[key] + 1) % n
			i := w.leg[key]
			if i >= len(points) {
				i = n - i
			}
			route = w.grid.path(from, points[i])
			w.rest[key] = .8
		} else {
			var near []smallTile
			for t := range w.grid.reachable(from) {
				if abs(t[0]-p.At[0])+abs(t[1]-p.At[1]) <= p.Move.Wander && t != from {
					near = append(near, t)
				}
			}
			slices.SortFunc(near, func(x, y smallTile) int { return x[0]*1000 + x[1] - y[0]*1000 - y[1] })
			if len(near) > 0 {
				route = w.grid.path(from, near[w.rng.IntN(len(near))])
			}
			w.rest[key] = 2 + 3*w.rng.Float64()
		}
		if len(route) > 1 {
			g.walkActor(key, route, walkerStep)
		}
	}
}

// exploreInput takes the keys of the walk: R or a right click shows the small tiles, the wheel zooms, Enter or
// Enter pages a talk; Space does so in debug mode.
func (g *Game) exploreInput() {
	w := g.explore
	if keyJustPressed(ebiten.KeyR) || mouseJustPressed(ebiten.MouseButtonRight) {
		w.show = !w.show
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.setZoom(math.Max(1, math.Min(4, g.zoomTo+math.Copysign(1, wheel))))
	}
	if w.talk != nil && (keyJustPressed(ebiten.KeyEnter) || g.Debug && keyJustPressed(ebiten.KeySpace)) {
		g.nextExploreLine()
	}
}

// exploreClick pages the talk, or sends the hero toward what lies under canvas point (x, y): a person, an exit or
// its building, a sign, or else the small tile there.
func (g *Game) exploreClick(x, y float64) {
	w := g.explore
	s := g.scene
	if w.talk != nil {
		g.nextExploreLine()
		return
	}
	if w.going != nil {
		return
	}
	w.aim = nil
	from := g.actorTile(w.hero)
	key, ex := g.exploreTargetAt(x, y)
	if key != "" {
		s.paths[key] = nil // a walker stops on the small tile it is stepping to
		at := g.actorTile(key)
		var route []smallTile
		for _, d := range []int{2, 1} {
			if route = w.grid.path(from, smallTile{at[0] + d, at[1]}, smallTile{at[0] - d, at[1]}, smallTile{at[0], at[1] + d}, smallTile{at[0], at[1] - d}); route != nil {
				break
			}
		}
		if route != nil {
			g.walkActor(w.hero, route, heroStep)
			w.aim = &exploreAim{person: key}
			g.sfx("ok")
		}
		return
	}
	a := g.S.Engine.ExploreFile().Areas[w.area]
	if ex != nil {
		if route := w.grid.path(from, w.grid.nearestOpen(ex.At)); route != nil {
			g.walkActor(w.hero, route, heroStep)
			w.aim = &exploreAim{exit: ex}
			g.sfx("ok")
		}
		return
	}
	at := smallTileAt(g.cellAt(x, y))
	for i := range a.Signs {
		sg := &a.Signs[i]
		if abs(at[0]-sg.At[0]) <= 1 && abs(at[1]-sg.At[1]) <= 1 || g.objectHit(sg.At, x, y) {
			t := smallTile(sg.At)
			if route := w.grid.path(from, smallTile{t[0] + 2, t[1]}, smallTile{t[0], t[1] + 2}, smallTile{t[0] - 2, t[1]}, smallTile{t[0], t[1] - 2}); route != nil {
				g.walkActor(w.hero, route, heroStep)
				w.aim = &exploreAim{sign: sg}
				g.sfx("ok")
			}
			return
		}
	}
	if to, ok := w.grid.nearestReachable(from, at); ok {
		if route := w.grid.path(from, to); route != nil {
			g.walkActor(w.hero, route, heroStep)
			w.mark, w.markT = to, 0
		}
	}
}

// exploreArrive does what the hero walked for: talks to the person (their event, the sortie question, their shop
// or their lines), reads the sign, goes through the exit or hears why it is shut.
func (g *Game) exploreArrive() {
	w := g.explore
	s := g.scene
	hero := s.actors[w.hero]
	aim := w.aim
	w.aim = nil
	g.reportSpot()
	a := g.S.Engine.ExploreFile().Areas[w.area]
	switch {
	case aim.person != "":
		p := s.actors[aim.person]
		hero.face(p.u, p.v)
		if _, ok := w.homes[aim.person]; !ok {
			w.homes[aim.person] = p.dir
		}
		p.face(hero.u, hero.v)
		key := w.people[aim.person]
		person := a.People[key]
		for _, v := range g.S.Engine.Visitors() {
			if v.Key != key {
				continue
			}
			switch v.Do {
			case "event":
				g.request(session.Request{Op: "command", Command: core.Command{Kind: "talk", Target: key}})
			case "sortie":
				if g.askWith(person.Sortie.Ask, "예", "아니오", func() {
					g.request(session.Request{Op: "command", Command: core.Command{Kind: "sortie", Target: key}})
				}) {
					g.confirmTitle = "출진 확인"
				}
				g.turnBack(aim.person)
			case "shop":
				g.shopFrom = key
				g.openOverlay("shop")
				g.turnBack(aim.person)
			default:
				g.startExploreTalk(aim.person, person.Talk)
			}
		}
	case aim.sign != nil:
		hero.face(smallTile(aim.sign.At).cell())
		g.startExploreTalk("", aim.sign.Talk)
	case g.S.Engine.ExitOpen(*aim.exit):
		w.going, w.fade = aim.exit, 0
	default:
		g.startExploreTalk("", aim.exit.Shut)
	}
}

// startExploreTalk opens the dialogue scroll on lines; with is the cast key of the person spoken to.
func (g *Game) startExploreTalk(with string, talk []content.Line) {
	w := g.explore
	if len(talk) == 0 {
		g.turnBack(with)
		return
	}
	w.talk, w.talkI, w.talkTo = nil, 0, with
	for _, l := range talk {
		if l[0] == "" {
			w.talk = append(w.talk, line{text: l[1]})
			continue
		}
		name := l[0]
		if c, ok := g.S.Data.Characters[name]; ok {
			name = c.Name
		}
		w.talk = append(w.talk, line{speakers: []string{name}, text: l[1], portrait: g.explorePortrait(name)})
	}
	g.lineTyping = startTyping(g.textSeconds)
	g.dialogueAnim = dialogueAnim{}
	g.scene.spoke, g.scene.held = map[string]bool{}, map[string]bool{w.hero: true, with: true}
}

// explorePortrait is the portrait a person of the area who is no character speaks with.
func (g *Game) explorePortrait(name string) string {
	if _, _, ok := g.character(name); ok {
		return ""
	}
	a := g.S.Engine.ExploreFile().Areas[g.explore.area]
	for key, p := range a.People {
		if content.PersonName(key) == name && p.Portrait != "" {
			return p.Portrait
		}
	}
	return ""
}

// nextExploreLine hurries the typing, shows the next line, or closes the talk.
func (g *Game) nextExploreLine() {
	w := g.explore
	if _, done := g.shownText(w.talk[w.talkI]); !done {
		g.lineTyping.hurry(g.textSeconds)
		return
	}
	if w.talkI+1 < len(w.talk) {
		w.talkI++
		g.lineTyping = startTyping(g.textSeconds)
		g.scene.spoke = map[string]bool{}
		return
	}
	w.talk = nil
	g.turnBack(w.talkTo)
	w.talkTo = ""
}

// turnBack turns a person who turned to the hero back the way they faced, after a moment.
func (g *Game) turnBack(key string) {
	w := g.explore
	v := g.scene.actors[key]
	home, ok := w.homes[key]
	if v == nil || !ok {
		return
	}
	delete(w.homes, key)
	w.rest[key] = 1.5
	g.startStep(scenes.Step{Actor: key, Look: v.dir})
	g.scene.looks[key].back = home
}

// exploreTargetAt is what a click on canvas point (x, y) picks: the person (cast key) or the exit whose name plate
// lies there, else the person whose figure covers it, else the exit whose spot or building does.
func (g *Game) exploreTargetAt(x, y float64) (string, *content.ExploreExit) {
	w := g.explore
	sx, sy := worldToUI(g.toScreen(x, y))
	for i := len(w.plates) - 1; i >= 0; i-- {
		if p := w.plates[i]; image.Pt(int(sx), int(sy)).In(p.rect) {
			if p.exit != nil {
				return "", g.exitOpenAt(p.exit)
			}
			if v := g.scene.actors[p.person]; v != nil && !v.gone {
				return p.person, nil
			}
		}
	}
	if key := g.explorePersonAt(x, y); key != "" {
		return key, nil
	}
	return "", g.exploreExitAt(x, y)
}

// explorePersonAt is the person whose figure covers canvas point (x, y), the nearest in front.
func (g *Game) explorePersonAt(x, y float64) string {
	w := g.explore
	best, bestY := "", math.Inf(-1)
	for key, v := range g.scene.actors {
		if _, ok := w.people[key]; !ok || v.gone {
			continue
		}
		fx, fy := g.field.Center(v.u, v.v)
		top := fy - v.lift - g.field.groundLift(v.u, v.v) - v.top
		if x >= fx-14 && x <= fx+14 && y >= top-2 && y <= fy+6 && fy > bestY {
			best, bestY = key, fy
		}
	}
	return best
}

// exploreExitAt is the exit whose spot (or a small tile next to it) or whose building's picture lies under (x, y):
// the open one of those sharing a spot.
func (g *Game) exploreExitAt(x, y float64) *content.ExploreExit {
	a := g.S.Engine.ExploreFile().Areas[g.explore.area]
	at := smallTileAt(g.cellAt(x, y))
	var found *content.ExploreExit
	for i := range a.Exits {
		e := &a.Exits[i]
		if abs(at[0]-e.At[0]) <= 1 && abs(at[1]-e.At[1]) <= 1 || e.Object != nil && g.objectHit(*e.Object, x, y) {
			if g.S.Engine.ExitOpen(*e) {
				return e
			}
			if found == nil {
				found = e
			}
		}
	}
	return found
}

// drawExplore draws the area, the plates over its exits, the name under the cursor, the talk and the walk's
// buttons.
func (g *Game) drawExplore(dst *ebiten.Image, o core.Observation) {
	g.ensureExplore(o)
	w := g.explore
	a := g.S.Engine.ExploreFile().Areas[w.area]
	var speakers []string
	if w.talk != nil {
		speakers = w.talk[w.talkI].speakers
	}
	g.clickRing(w.mark, w.markT)
	g.drawScene(dst, speakers)
	if w.show {
		var exits []smallTile
		for _, e := range a.Exits {
			exits = append(exits, e.At)
		}
		g.drawWalkGrid(dst, &w.grid, exits)
	}
	x, y := g.toCanvas(cursorWorld())
	idle := w.talk == nil && w.going == nil && !g.popupPending()
	var hover *content.ExploreExit
	var person string
	if !pointer.used { // a finger leaves no pointer resting over the map: its last touch would ride along with the scroll
		person, hover = g.exploreTargetAt(x, y)
	}
	if idle && !pointer.used && hover == nil && person == "" {
		if t := smallTileAt(g.cellAt(x, y)); w.grid.open(t) {
			g.smallDiamond(dst, t, cursorFill, cursorRim)
		}
	}
	guideExit, guidePerson := g.exploreGuide()
	var plates []namedPlate
	seen := map[[2]int]bool{}
	for i := range a.Exits {
		e := &a.Exits[i]
		if seen[e.At] || !g.S.Engine.ExitOpen(*e) && e.Object == nil {
			continue
		}
		seen[e.At] = true
		px, py := g.exitPlatePoint(*e)
		style := plateDim
		for j, o := range a.Exits {
			if o.At == e.At && guideExit[j] != plateDim {
				style = guideExit[j]
			}
		}
		sx, sy := worldToUI(g.toScreen(px, py))
		plates = append(plates, namedPlate{g.exitName(a, e), sx, sy - 6, style, idle && hover != nil && hover.At == e.At, "", e, px, py})
	}
	for key, pk := range w.people {
		v := g.scene.actors[key]
		if v == nil || v.gone || w.talk != nil || scenes.Extra(key) && !talkable(a.People[pk]) {
			continue
		}
		style := guidePerson[pk]
		x, y := g.field.Center(v.u, v.v)
		sx, sy := worldToUI(g.toScreen(x, y-v.lift-g.field.groundLift(v.u, v.v)-v.top-4))
		plates = append(plates, namedPlate{content.PersonName(pk), sx, sy, style, idle && person == key, key, nil, x, y})
	}
	// the one pointed at over all, then the guide's plates; among the rest the plate of whoever stands lower on screen,
	// then further right, on top, so that overlapping plates keep one order from frame to frame
	slices.SortStableFunc(plates, func(p, q namedPlate) int {
		if p.hover != q.hover {
			return map[bool]int{false: -1, true: 1}[p.hover]
		}
		return cmp.Or(int(p.style)-int(q.style), cmp.Compare(p.footY, q.footY), cmp.Compare(p.footX, q.footX), strings.Compare(p.name, q.name))
	})
	w.plates = w.plates[:0]
	for _, p := range plates {
		w.plates = append(w.plates, explorePlate{g.namePlate(dst, p.name, p.x, p.y, p.style, p.hover), p.person, p.exit})
	}
	if w.talk != nil {
		l := w.talk[w.talkI]
		text, done := g.shownText(l)
		g.drawDialogue(dst, l, text, done, fmt.Sprintf("%s|explore|%s|%d", strings.Join(l.speakers, "·"), w.talkTo, w.talkI))
	}
	g.blackOut(dst, math.Max(w.fade, g.scene.fade))
}

type namedPlate struct {
	name   string
	x, y   float64
	style  plate
	hover  bool
	person string               // the cast key of the person it names
	exit   *content.ExploreExit // or the exit
	footX  float64              // canvas point of the person's feet, or of the exit's plate: what plates are layered by
	footY  float64
}

// talkable reports whether talking to a person does anything: an extra who does shows a name plate like the named.
func talkable(p *content.ExplorePerson) bool {
	return p != nil && (len(p.Talk) > 0 || len(p.Event) > 0 || p.Sortie != nil || p.Shop != "")
}

// exploreGuide is how the guide marks the exits of the hero's area (by index) and its people (by key): the main
// story's way green, a side story's and a shop's (the shopkeeper and the door into their shop) yellow, the main one
// winning where both lead. Nothing when the player turned the guide off.
func (g *Game) exploreGuide() (map[int]plate, map[string]plate) {
	exits, people := map[int]plate{}, map[string]plate{}
	if g.S.Settings.NoGuide {
		return exits, people
	}
	mark := func(m map[int]plate, k int, side bool) {
		if side && m[k] == plateMain {
			return
		}
		m[k] = map[bool]plate{false: plateMain, true: plateSide}[side]
	}
	for _, goal := range g.S.Engine.Goals() {
		if goal.First >= 0 {
			mark(exits, goal.First, goal.Side)
		} else if !goal.Side || people[goal.Person] != plateMain {
			people[goal.Person] = map[bool]plate{false: plateMain, true: plateSide}[goal.Side]
		}
	}
	x := g.S.Engine.ExploreFile()
	a := x.Areas[g.explore.area]
	shopPlate := func(shop string) plate {
		if len(g.S.Data.NewStock(g.S.Engine.ExploreName(), shop)) > 0 {
			return plateHot
		}
		return plateSide
	}
	// shopIn is the plate of the shop in an area: the hottest of its shopkeepers', plateDim when it has none.
	shopIn := func(area string) plate {
		found := plateDim
		if to := x.Areas[area]; to != nil {
			for _, p := range to.People {
				if p.Shop != "" {
					found = max(found, shopPlate(p.Shop))
				}
			}
		}
		return found
	}
	for i, ex := range a.Exits {
		if !g.S.Engine.ExitOpen(ex) {
			continue
		}
		if shop := shopIn(ex.To); shop != plateDim {
			mark(exits, i, true)
			if shop == plateHot && exits[i] != plateMain {
				exits[i] = plateHot
			}
		}
	}
	for key, p := range a.People {
		if p.Shop != "" && people[key] != plateMain {
			people[key] = shopPlate(p.Shop)
		}
	}
	return exits, people
}

// exitPlatePoint is the canvas point an exit's plate stands over: its doorway, a little above the ground, so the
// plate marks where the hero goes in rather than floating over the roof; an exit into a building keeps it on the
// building's picture, over the side of it nearest the door, where a door spot out in a yard or before a gate
// would leave it on bare ground beside it.
func (g *Game) exitPlatePoint(e content.ExploreExit) (float64, float64) {
	u, v := smallTile(e.At).cell()
	x, y := g.field.Center(u, v)
	y -= g.field.groundLift(u, v) + 24
	if e.Object == nil {
		return x, y
	}
	r, ok := g.objectRect(*e.Object)
	if !ok {
		return x, y
	}
	inX, inY := float64(r.Dx())/8, math.Min(24, float64(r.Dy())/3)
	return math.Max(float64(r.Min.X)+inX, math.Min(float64(r.Max.X)-inX, x)), math.Max(float64(r.Min.Y)+inY, math.Min(float64(r.Max.Y)-inY, y))
}

// exitOpenAt is the exit open now among those sharing e's spot, else e.
func (g *Game) exitOpenAt(e *content.ExploreExit) *content.ExploreExit {
	a := g.S.Engine.ExploreFile().Areas[g.explore.area]
	for i := range a.Exits {
		if o := &a.Exits[i]; o.At == e.At && g.S.Engine.ExitOpen(*o) {
			return o
		}
	}
	return e
}

// exitName is the name an exit's plate shows: of the one open now among those sharing its spot.
func (g *Game) exitName(a *content.ExploreArea, e *content.ExploreExit) string {
	for _, other := range a.Exits {
		if other.At == e.At && g.S.Engine.ExitOpen(other) {
			return other.Name
		}
	}
	return e.Name
}
