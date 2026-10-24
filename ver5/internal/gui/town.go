package gui

import (
	"container/heap"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/scenes"
	"srpg/internal/town"
)

// townWalk is the town walk (assets/towns/*.json): the hero walks an area's scene map where the player clicks,
// walks up to the person, sign or door clicked and talks to them or goes through. It plays on the story scene
// (scene.go): the area is a scene with no beats, its people the cast, the hero one more actor walking a path.
type townWalk struct {
	t      *town.Town
	area   string
	hero   string // the hero's cast key: their shown name
	aim    *townAim
	talk   []line
	talkI  int
	talkTo string            // whom the hero is talking to ("" for a sign or a shut door)
	homes  map[string]string // the way each person faced before turning to the hero
	going  *town.Door        // the door the hero is going through, while the screen fades out
	fade   float64           // 0 → 1: the screen going black before the next area
	mark   tile              // where the last click on the ground sent the hero
	markT  float64           // seconds since then
	grid   bool              // R or a right click: show which tiles the hero can walk
}

// townAim is what the hero is walking to, and what happens on arrival.
type townAim struct {
	person string
	sign   *town.Sign
	door   *town.Door
}

const (
	townFade = .35
	townMark = .6 // seconds the click ring spreads on the ground
)

// OpenTown starts the town walk of a town file (assets/towns/<id>.json) at its start.
func (g *Game) OpenTown(path string) error {
	t, err := town.Load(path)
	if err != nil {
		return err
	}
	hero := t.Hero
	if c, ok := g.S.Data.Characters[hero]; ok {
		hero = c.Name
	}
	g.town = &townWalk{t: t, hero: hero}
	for id, a := range t.Areas {
		if g.assets.Maps[a.Map] == nil {
			return fmt.Errorf("town area %s: no scene map %s", id, a.Map)
		}
	}
	g.enterArea(t.Start.Area, t.Start.At, t.Start.Face)
	return nil
}

// enterArea opens area id as a scene, the hero standing at (at) facing face.
func (g *Game) enterArea(id string, at []float64, face string) {
	w := g.town
	a := w.t.Areas[id]
	w.area, w.aim, w.going, w.fade, w.markT, w.homes = id, nil, nil, 0, math.Inf(1), map[string]string{}
	def := &scenes.Scene{ID: "town:" + id, Map: a.Map, Time: a.Time, Cast: map[string]scenes.Actor{
		w.hero: {At: at, Face: face}}}
	for key, p := range a.People {
		def.Cast[key] = scenes.Actor{At: p.At, Face: p.Face, Faction: p.Faction, Sprite: p.Sprite, Hold: p.Hold, Pose: p.Pose}
	}
	g.openScene(def)
	s := g.scene
	for key, p := range a.People {
		if len(p.Patrol) == 2 || p.Chatter != "" {
			g.startStep(scenes.Step{Actor: key, Patrol: p.Patrol, Chatter: p.Chatter})
		}
	}
	if a.Zoom >= 1 {
		g.zoom = math.Round(a.Zoom)
	}
	s.camU, s.camV = at[0], at[1]
	g.camX, g.camY = g.sceneCam()
}

// updateTown steps the walk by one frame and takes the player's clicks.
func (g *Game) updateTown() error {
	g.advance(1.0 / 60)
	if g.townTick() {
		g.townInput()
	}
	return nil
}

// townTick follows the hero with the camera, fades through a door and does what the hero arrived for; false
// while the screen fades to the next area.
func (g *Game) townTick() bool {
	w := g.town
	w.markT += 1.0 / 60
	s := g.scene
	hero := s.actors[w.hero]
	s.camU, s.camV = hero.u, hero.v
	if w.going != nil {
		if w.fade = math.Min(1, w.fade+1.0/60/townFade); w.fade >= 1 {
			back := w.t.Areas[w.going.To].DoorTo(w.area)
			g.enterArea(w.going.To, back.At, back.Face)
		}
		return false
	}
	if w.aim != nil && w.talk == nil && hero.walk == nil && len(s.paths[w.hero]) == 0 {
		g.arrive()
	}
	return true
}

// townInput takes the player's clicks and keys: a click pages the talk or sends the hero; R or a right click shows the tiles.
func (g *Game) townInput() {
	w := g.town
	if keyJustPressed(ebiten.KeyR) || mouseJustPressed(ebiten.MouseButtonRight) {
		w.grid = !w.grid
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.zoom = math.Max(1, math.Min(4, g.zoom+math.Copysign(1, wheel)))
	}
	if w.talk != nil {
		if mouseJustPressed(ebiten.MouseButtonLeft) || keyJustPressed(ebiten.KeyEnter) || keyJustPressed(ebiten.KeySpace) {
			g.nextTownLine()
		}
		return
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		g.townClick(g.toCanvas(cursorWorld()))
	}
}

// townClick sends the hero toward what lies under canvas point (x, y): a person, a door or its building, a sign,
// or else the tile there.
func (g *Game) townClick(x, y float64) {
	w := g.town
	a := w.t.Areas[w.area]
	s := g.scene
	w.aim = nil
	if key := g.townPersonAt(x, y); key != "" {
		p := s.actors[key]
		if pt := s.patrol[key]; pt != nil { // a walker stops on the tile it is stepping to, to be spoken to
			delete(s.patrol, key)
			s.paths[key] = nil
		}
		at := tileAt(p.u, p.v)
		if p.walk != nil {
			at = tileAt(p.walk.tu, p.walk.tv)
		}
		g.walkHero(at.around())
		w.aim = &townAim{person: key}
		return
	}
	if d := g.townDoorAt(x, y); d != nil {
		g.walkHero([]tile{tileAt(d.At[0], d.At[1])})
		w.aim = &townAim{door: d}
		return
	}
	u, v := g.townCell(x, y)
	for i := range a.Signs {
		sg := &a.Signs[i]
		if at := tileAt(sg.At[0], sg.At[1]); tileAt(u, v) == at || g.townPropHit(sg.At, x, y) {
			g.walkHero(at.around())
			w.aim = &townAim{sign: sg}
			return
		}
	}
	m := g.townTiles()
	if to, ok := m.nearestReachable(g.heroTile(), tileAt(u, v)); ok {
		g.walkHero([]tile{to})
		w.mark, w.markT = to, 0
	}
}

// heroTile is the tile the hero stands on, or the one they are stepping onto.
func (g *Game) heroTile() tile {
	hero := g.scene.actors[g.town.hero]
	if hero.walk != nil {
		return tileAt(hero.walk.tu, hero.walk.tv)
	}
	return tileAt(hero.u, hero.v)
}

// walkHero sends the hero tile by tile along the shortest way (A*) to the nearest of goals; the hero stays when
// none can be reached.
func (g *Game) walkHero(goals []tile) {
	w := g.town
	s := g.scene
	route := g.townTiles().path(g.heroTile(), goals)
	if route == nil {
		w.aim = nil
		s.paths[w.hero] = nil
		return
	}
	steps := [][2]float64{}
	for _, t := range route[1:] {
		steps = append(steps, [2]float64{float64(t.u), float64(t.v)})
	}
	s.paths[w.hero] = steps
}

// arrive does what the hero walked for: talks to the person, reads the sign, goes through the door.
func (g *Game) arrive() {
	w := g.town
	s := g.scene
	hero := s.actors[w.hero]
	aim := w.aim
	w.aim = nil
	switch {
	case aim.person != "":
		p := s.actors[aim.person]
		hero.face(p.u, p.v)
		if _, ok := w.homes[aim.person]; !ok {
			w.homes[aim.person] = p.dir
		}
		p.face(hero.u, hero.v)
		g.startTalk(aim.person, w.t.Areas[w.area].People[aim.person].Talk)
	case aim.sign != nil:
		hero.face(aim.sign.At[0], aim.sign.At[1])
		g.startTalk("", aim.sign.Talk)
	case aim.door.To != "":
		w.going = aim.door
	default:
		if len(aim.door.Prop) == 2 {
			hero.face(aim.door.Prop[0], aim.door.Prop[1])
		}
		g.startTalk("", aim.door.Talk)
	}
}

// startTalk opens the dialogue scroll on lines; with is the person spoken to.
func (g *Game) startTalk(with string, talk []town.Line) {
	w := g.town
	if len(talk) == 0 {
		return
	}
	w.talk, w.talkI, w.talkTo = nil, 0, with
	for _, l := range talk {
		if l[0] == "" {
			w.talk = append(w.talk, line{text: l[1]})
			continue
		}
		w.talk = append(w.talk, line{speakers: []string{l[0]}, text: l[1], portrait: g.townPortrait(l[0])})
	}
	g.lineTyping = startTyping(g.uiSeconds)
	g.scrollAnim = scrollAnim{}
	g.scene.spoke, g.scene.held = map[string]bool{}, map[string]bool{w.hero: true, with: true}
}

// townPortrait is the portrait a town person who is no character speaks with.
func (g *Game) townPortrait(name string) string {
	if _, _, ok := g.character(name); ok {
		return ""
	}
	for _, a := range g.town.t.Areas {
		if p, ok := a.People[name]; ok && p.Portrait != "" {
			return p.Portrait
		}
	}
	return ""
}

// nextTownLine hurries the typing, shows the next line, or closes the talk: the person turns back and a walker
// walks on.
func (g *Game) nextTownLine() {
	w := g.town
	if _, done := g.shownText(w.talk[w.talkI]); !done {
		g.lineTyping.hurry(g.uiSeconds)
		return
	}
	if w.talkI+1 < len(w.talk) {
		w.talkI++
		g.lineTyping = startTyping(g.uiSeconds)
		g.scene.spoke = map[string]bool{}
		return
	}
	w.talk = nil
	key := w.talkTo
	p := w.t.Areas[w.area].People[key]
	if v := g.scene.actors[key]; v != nil {
		if len(p.Patrol) == 2 {
			g.startStep(scenes.Step{Actor: key, Patrol: p.Patrol})
		} else if home, ok := w.homes[key]; ok {
			g.startStep(scenes.Step{Actor: key, Look: v.dir})
			g.scene.looks[key].back = home
		}
	}
	delete(w.homes, key)
}

// townPersonAt is the person who can be spoken to whose figure covers canvas point (x, y), the nearest in front.
func (g *Game) townPersonAt(x, y float64) string {
	w := g.town
	people := w.t.Areas[w.area].People
	best, bestY := "", math.Inf(-1)
	for key, v := range g.scene.actors {
		if key == w.hero || v.gone || len(people[key].Talk) == 0 {
			continue
		}
		fx, fy := g.field.Center(v.u, v.v)
		if x >= fx-12 && x <= fx+12 && y >= fy-v.lift-v.top && y <= fy+6 && fy > bestY {
			best, bestY = key, fy
		}
	}
	return best
}

// townDoorAt is the door whose building picture covers canvas point (x, y), or whose spot is near it.
func (g *Game) townDoorAt(x, y float64) *town.Door {
	w := g.town
	a := w.t.Areas[w.area]
	at := tileAt(g.townCell(x, y))
	for i := range a.Doors {
		d := &a.Doors[i]
		if at == tileAt(d.At[0], d.At[1]) || g.townPropHit(d.Prop, x, y) {
			return d
		}
	}
	return nil
}

// townProp is the standing map prop nearest anchor (u, v), within half a cell.
func (g *Game) townProp(at []float64) (Sprite, float64, float64, bool) {
	best, bu, bv := math.Inf(1), 0.0, 0.0
	var found Sprite
	if len(at) != 2 {
		return found, 0, 0, false
	}
	for _, p := range g.field.Map.Props {
		pr, ok := g.assets.Props[p.Name]
		if !ok || pr.Ground || g.town.t.Flat(p.Name) || p.Fill {
			continue
		}
		if d := math.Hypot(p.U-at[0], p.V-at[1]); d < .5 && d < best {
			best, bu, bv, found = d, p.U, p.V, pr.Sprite
		}
	}
	return found, bu, bv, best < .5
}

// townPropHit reports whether canvas point (x, y) falls on the picture of the prop at anchor at.
func (g *Game) townPropHit(at []float64, x, y float64) bool {
	s, u, v, ok := g.townProp(at)
	if !ok {
		return false
	}
	px, py := g.field.Center(u, v)
	b := s.Image.Bounds()
	x0, y0 := px-s.PivotX, py-s.PivotY
	return x >= x0 && x < x0+float64(b.Dx()) && y >= y0 && y < y0+float64(b.Dy())
}

// townCell is the (fractional) cell under canvas point (x, y), on the ground.
func (g *Game) townCell(x, y float64) (float64, float64) {
	f := g.field
	a, b := (x/K-f.OriginX)/16, (y/K-f.OriginY)/8-1
	return (a + b) / 2, (b - a) / 2
}

// tile is one cell of the area's isometric tile map. The hero walks from tile to tile along u or v: down right
// (+u, SE), up left (−u, NW), down left (+v, SW), up right (−v, NE).
type tile struct{ u, v int }

// tileAt is the tile a (fractional) cell position lies on.
func tileAt(u, v float64) tile { return tile{int(math.Floor(u + .5)), int(math.Floor(v + .5))} }

// steps4 are the four ways the hero steps.
var steps4 = []tile{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// around is the four tiles next to t, from which the hero speaks to whoever stands on it.
func (t tile) around() []tile {
	out := []tile{}
	for _, d := range steps4 {
		out = append(out, tile{t.u + d.u, t.v + d.v})
	}
	return out
}

func manhattan(a, b tile) int { return abs(a.u-b.u) + abs(a.v-b.v) }

// tileMap is which tiles of the area can be walked.
type tileMap struct {
	w, h int
	open []bool
}

func (m *tileMap) free(t tile) bool {
	return t.u >= 0 && t.v >= 0 && t.u < m.w && t.v < m.h && m.open[t.v*m.w+t.u]
}

// townTiles marks the area's walkable tiles: open ground (at least three of the tile's four small tiles are
// ".", "d" or "i"), not under a standing prop, not where another person stands. A prop with a footprint covers
// every tile it overlaps by 0.4 of a cell, or half its own size when smaller, on both axes; any other standing
// prop covers the tile its anchor lies on.
func (g *Game) townTiles() *tileMap {
	w := g.town
	f := g.field
	m := &tileMap{w: f.Map.W, h: f.Map.H, open: make([]bool, f.Map.W*f.Map.H)}
	for v := 0; v < m.h; v++ {
		for u := 0; u < m.w; u++ {
			n := 0
			for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				if strings.IndexByte(".di", f.small(2*u+d[0], 2*v+d[1])) >= 0 {
					n++
				}
			}
			m.open[v*m.w+u] = n >= 3
		}
	}
	block := func(t tile) {
		if t.u >= 0 && t.v >= 0 && t.u < m.w && t.v < m.h {
			m.open[t.v*m.w+t.u] = false
		}
	}
	for _, p := range f.Map.Props {
		pr, ok := g.assets.Props[p.Name]
		if !ok || pr.Ground || w.t.Flat(p.Name) || p.Fill {
			continue
		}
		fp, ok := w.t.Footprints[p.Name]
		if !ok {
			block(tileAt(p.U, p.V))
			continue
		}
		u0, v0, u1, v1 := p.U+fp[0], p.V+fp[1], p.U+fp[2], p.V+fp[3]
		needU, needV := math.Min(.4, (u1-u0)/2), math.Min(.4, (v1-v0)/2)
		for v := int(math.Floor(v0 + .5)); v <= int(math.Floor(v1+.5)); v++ {
			for u := int(math.Floor(u0 + .5)); u <= int(math.Floor(u1+.5)); u++ {
				ou := math.Min(u1, float64(u)+.5) - math.Max(u0, float64(u)-.5)
				ov := math.Min(v1, float64(v)+.5) - math.Max(v0, float64(v)-.5)
				if ou >= needU-1e-9 && ov >= needV-1e-9 {
					block(tile{u, v})
				}
			}
		}
	}
	for key, v := range g.scene.actors {
		if key != w.hero && !v.gone {
			block(tileAt(v.u, v.v))
		}
	}
	return m
}

// path is the tiles from from to the nearest open tile of goals, stepping along u or v only, by A* (the
// distance to the nearest goal guides it); nil when none can be reached.
func (m *tileMap) path(from tile, goals []tile) []tile {
	ends := []tile{}
	for _, t := range goals {
		if m.free(t) || t == from {
			ends = append(ends, t)
		}
	}
	if len(ends) == 0 {
		return nil
	}
	guess := func(t tile) int {
		best := math.MaxInt
		for _, e := range ends {
			best = min(best, manhattan(t, e))
		}
		return best
	}
	cost := map[tile]int{from: 0}
	prev := map[tile]tile{}
	q := &nodeQueue{{from, guess(from)}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(node)
		if guess(cur.t) == 0 {
			out := []tile{cur.t}
			for t := cur.t; t != from; {
				t = prev[t]
				out = append([]tile{t}, out...)
			}
			return out
		}
		for _, d := range steps4 {
			n := tile{cur.t.u + d.u, cur.t.v + d.v}
			if !m.free(n) {
				continue
			}
			if c, seen := cost[n]; !seen || cost[cur.t]+1 < c {
				cost[n], prev[n] = cost[cur.t]+1, cur.t
				heap.Push(q, node{n, cost[n] + guess(n)})
			}
		}
	}
	return nil
}

// nearestReachable is the open tile the hero can walk to from from that lies closest to want (want itself when
// it is open and reachable), nearer by the walk when two are as close.
func (m *tileMap) nearestReachable(from, want tile) (tile, bool) {
	dist := map[tile]int{from: 0}
	queue := []tile{from}
	best, bestD, bestW := from, math.Inf(1), 0
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		if d := math.Hypot(float64(t.u-want.u), float64(t.v-want.v)); d < bestD || d == bestD && dist[t] < bestW {
			best, bestD, bestW = t, d, dist[t]
		}
		for _, s := range steps4 {
			n := tile{t.u + s.u, t.v + s.v}
			if _, seen := dist[n]; !seen && m.free(n) {
				dist[n] = dist[t] + 1
				queue = append(queue, n)
			}
		}
	}
	return best, !math.IsInf(bestD, 1)
}

type node struct {
	t tile
	f int // steps so far plus the guess to the nearest goal
}

type nodeQueue []node

func (q nodeQueue) Len() int           { return len(q) }
func (q nodeQueue) Less(i, j int) bool { return q[i].f < q[j].f }
func (q nodeQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *nodeQueue) Push(x any)        { *q = append(*q, x.(node)) }
func (q *nodeQueue) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}

// drawTown draws the area, the name plates over its doors, what the cursor points at and the talk.
func (g *Game) drawTown(dst *ebiten.Image) {
	w := g.town
	a := w.t.Areas[w.area]
	speakers := []string{}
	if w.talk != nil {
		speakers = w.talk[w.talkI].speakers
	}
	g.field.groundMarks = g.field.groundMarks[:0]
	if ring := g.assets.FX["click_ring"]; w.markT < townMark && len(ring.Frames) > 0 {
		x, y := g.field.Center(float64(w.mark.u), float64(w.mark.v))
		f := ring.Frames[ring.frameOver(w.markT/townMark)]
		g.field.groundMarks = append(g.field.groundMarks, groundMark{Sprite{f, ring.Anchor[0], ring.Anchor[1]}, x, y})
	}
	g.drawScene(dst, speakers)
	if w.grid {
		g.drawTownGrid(dst)
	}
	if w.talk == nil && w.going == nil {
		if t := tileAt(g.townCell(g.toCanvas(cursorWorld()))); g.townTiles().free(t) {
			g.tileDiamond(dst, t, color.NRGBA{R: 255, G: 240, B: 200, A: 36}, color.NRGBA{R: 255, G: 240, B: 200, A: 120})
		}
	}
	for _, d := range a.Doors {
		if d.Name == "" {
			continue
		}
		x, y := g.field.Center(d.At[0], d.At[1])
		if s, u, v, ok := g.townProp(d.Prop); ok {
			x, y = g.field.Center(u, v)
			y -= s.PivotY
		} else {
			y -= 18
		}
		sx, sy := worldToUI(g.toScreen(x, y))
		g.namePlate(dst, d.Name, sx, sy-6, true)
	}
	if w.talk == nil && w.going == nil {
		if name := g.townHover(); name != "" {
			x, y := cursorPos()
			g.namePlate(dst, name, float64(x), float64(y)-16, false)
		}
	}
	g.label(dst, w.t.Name+" · "+a.Name, 24, 18, 18, gold)
	g.label(dst, "클릭: 이동·대화·출입   휠: 확대   R·오른쪽 클릭: 타일 보기", 24, 46, 13, muted)
	if w.talk != nil {
		l := w.talk[w.talkI]
		text, done := g.shownText(l)
		g.drawDialogue(dst, l, text, done, fmt.Sprintf("%s|town|%s|%d", strings.Join(l.speakers, "·"), w.talkTo, w.talkI))
	}
	// the area opening out of black and the screen going black at a door cover the plates and labels too
	if k := math.Max(w.fade, g.scene.fade); k > 0 {
		inWorld(func() { rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{A: uint8(255 * k)}) })
	}
}

// townHover is the name of what the cursor points at: a person, a door's building, a sign.
func (g *Game) townHover() string {
	x, y := g.toCanvas(cursorWorld())
	if key := g.townPersonAt(x, y); key != "" {
		return key
	}
	if d := g.townDoorAt(x, y); d != nil {
		if d.Name == "" {
			return "민가"
		}
		return d.Name
	}
	at := tileAt(g.townCell(x, y))
	for _, sg := range g.town.t.Areas[g.town.area].Signs {
		if at == tileAt(sg.At[0], sg.At[1]) || g.townPropHit(sg.At, x, y) {
			return sg.Name
		}
	}
	return ""
}

// namePlate is a small dark plate with a name, centred on UI point (x, y)'s bottom edge.
func (g *Game) namePlate(dst *ebiten.Image, name string, x, y float64, sign bool) {
	f := g.serif(15, 800)
	tw := measure(name, f)
	px, py := x-tw/2-10, y-26
	body, rim := color.NRGBA{R: 30, G: 20, B: 14, A: 210}, goldDim
	if sign {
		body, rim = color.NRGBA{R: 92, G: 30, B: 20, A: 235}, gold
	}
	rect(dst, float32(px), float32(py), float32(tw+20), 24, body)
	strokeRect(dst, float32(px)+.5, float32(py)+.5, float32(tw+19), 23, 1, rim)
	write(dst, name, f, px+10, py+1, ink)
}

// tileDiamond fills tile t's ground diamond and draws its rim, over the map at the camera's zoom.
func (g *Game) tileDiamond(dst *ebiten.Image, t tile, fill, rim color.Color) {
	x, y := g.field.Center(float64(t.u), float64(t.v))
	pts := [4][2]float32{}
	for i, d := range [][2]float64{{0, -16}, {32, 0}, {0, 16}, {-32, 0}} {
		sx, sy := worldToUI(g.toScreen(x+d[0], y+d[1]))
		pts[i] = [2]float32{float32(sx), float32(sy)}
	}
	var p vector.Path
	p.MoveTo(pts[0][0], pts[0][1])
	for _, q := range pts[1:] {
		p.LineTo(q[0], q[1])
	}
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(fill)
	fillPath(dst, &p, op)
	for i := range pts {
		q := pts[(i+1)%4]
		strokeLine(dst, pts[i][0], pts[i][1], q[0], q[1], 1.5, rim)
	}
}

// drawTownGrid shades the tiles the hero cannot walk red and the door tiles blue (R or a right click), to check the tile map
// against the pictures.
func (g *Game) drawTownGrid(dst *ebiten.Image) {
	m := g.townTiles()
	for v := 0; v < m.h; v++ {
		for u := 0; u < m.w; u++ {
			if !m.free(tile{u, v}) {
				g.tileDiamond(dst, tile{u, v}, color.NRGBA{R: 200, G: 40, B: 30, A: 90}, color.NRGBA{R: 230, G: 70, B: 50, A: 160})
			} else {
				g.tileDiamond(dst, tile{u, v}, color.NRGBA{}, color.NRGBA{R: 255, G: 255, B: 255, A: 50})
			}
		}
	}
	for _, d := range g.town.t.Areas[g.town.area].Doors {
		g.tileDiamond(dst, tileAt(d.At[0], d.At[1]), color.NRGBA{R: 60, G: 130, B: 255, A: 90}, color.NRGBA{R: 90, G: 160, B: 255, A: 200})
	}
}
