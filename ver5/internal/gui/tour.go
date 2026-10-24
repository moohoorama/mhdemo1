package gui

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// tourWalk is the walk round the scene maps of format 2 (assets/towns/tour2.json, cmd/_town2): the hero walks
// small tile by small tile where the player clicks, and clicking an exit (a door's building, a way out) walks there
// and opens the next map. Nobody else stands in it: it is for looking at how the maps are drawn. Which small tiles
// can be walked comes from the map rules (scenemap.Walkable).
type tourWalk struct {
	t     *tour
	area  string
	hero  string
	walk  *scenemap.Walk
	aim   *tourExit // the exit the hero is walking to
	going *tourExit // the exit the hero is going through, while the screen fades out
	fade  float64
	mark  smallTile // where the last click on the ground sent the hero
	markT float64
	grid  bool // R or a right click: show which small tiles can be walked
}

// tour is a tour file: areas on scene maps of format 2 joined by exits.
type tour struct {
	Name  string
	Hero  string // the walker: a character's id or shown name
	Start string
	Areas map[string]*tourArea
}

type tourArea struct {
	Map   string
	Name  string
	Zoom  float64   // the camera's zoom on arrival, whole numbers; 2 when 0
	At    smallTile // where the hero stands on arriving from nowhere (the start, an area without a way back)
	Exits []tourExit
}

// tourExit leads to area To: the hero walks to small tile At (or the open one nearest), then stands at the exit
// of To that leads back. Object is the small tile of the anchor of the building it belongs to, whose picture can be
// clicked too.
type tourExit struct {
	Name   string
	At     smallTile
	Object *smallTile
	To     string
}

// smallTile is a small tile (U, V): half a game cell. Its centre is game cell point ((U - .5) / 2, (V - .5) / 2).
type smallTile [2]int

func (t smallTile) cell() (float64, float64) {
	return (float64(t[0]) - .5) / 2, (float64(t[1]) - .5) / 2
}

// smallTileAt is the small tile a game cell point lies on.
func smallTileAt(u, v float64) smallTile {
	return smallTile{int(math.Floor(2*u + 1)), int(math.Floor(2*v + 1))}
}

const tourStep = 2.0 // the hero's walking speed: a scene walk's step is a game cell, a tour step half one

// OpenTour starts the tour of a tour file at its start area.
func (g *Game) OpenTour(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	t := &tour{}
	if err := json.Unmarshal(raw, t); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for id, a := range t.Areas {
		art := g.assets.Maps[a.Map]
		if art == nil || art.SceneMap == nil {
			return fmt.Errorf("%s: area %s: %s is no scene map of format 2", path, id, a.Map)
		}
		for _, e := range a.Exits {
			if t.Areas[e.To] == nil {
				return fmt.Errorf("%s: area %s: exit %q leads to unknown area %q", path, id, e.Name, e.To)
			}
		}
	}
	if t.Areas[t.Start] == nil {
		return fmt.Errorf("%s: start area %q is missing", path, t.Start)
	}
	hero := t.Hero
	if c, ok := g.S.Data.Characters[hero]; ok {
		hero = c.Name
	}
	g.tour = &tourWalk{t: t, hero: hero}
	g.enterTourArea(t.Start, "")
	return nil
}

// enterTourArea opens area id, the hero standing at its exit back to area from (at its At when there is none).
func (g *Game) enterTourArea(id, from string) {
	w := g.tour
	a := w.t.Areas[id]
	art := g.assets.Maps[a.Map]
	w.area, w.aim, w.going, w.fade, w.markT = id, nil, nil, 0, math.Inf(1)
	w.walk = scenemap.Walkable(art.SceneMap, g.assets.SceneRules)
	at, face := a.At, "SE"
	for _, e := range a.Exits {
		if e.To == from {
			at, face = e.At, "NW"
		}
	}
	at = w.nearestOpen(at)
	u, v := at.cell()
	g.openScene(&scenes.Scene{ID: "tour:" + id, Map: a.Map, Cast: map[string]scenes.Actor{w.hero: {At: []float64{u, v}, Face: face}}})
	g.zoom = 2
	if a.Zoom >= 1 {
		g.zoom = math.Round(a.Zoom)
	}
	g.scene.camU, g.scene.camV = u, v
	g.camX, g.camY = g.sceneCam()
}

// nearestOpen is the open small tile nearest t (t itself when open).
func (w *tourWalk) nearestOpen(t smallTile) smallTile {
	best, bestD := t, math.Inf(1)
	for v := w.walk.Origin[1]; v < w.walk.Origin[1]+w.walk.H; v++ {
		for u := w.walk.Origin[0]; u < w.walk.Origin[0]+w.walk.W; u++ {
			if d := math.Hypot(float64(u-t[0]), float64(v-t[1])); w.walk.Open(u, v) && d < bestD {
				best, bestD = smallTile{u, v}, d
			}
		}
	}
	return best
}

func (g *Game) updateTour() error {
	g.advance(1.0 / 60)
	if g.tourTick() {
		g.tourInput()
	}
	return nil
}

// tourTick follows the hero with the camera and goes through the exit the hero arrived at; false while the
// screen fades to the next area.
func (g *Game) tourTick() bool {
	w := g.tour
	w.markT += 1.0 / 60
	s := g.scene
	hero := s.actors[w.hero]
	s.camU, s.camV = hero.u, hero.v
	if w.going != nil {
		if w.fade = math.Min(1, w.fade+1.0/60/townFade); w.fade >= 1 {
			g.enterTourArea(w.going.To, w.area)
		}
		return false
	}
	if w.aim != nil && hero.walk == nil && len(s.paths[w.hero]) == 0 {
		w.going, w.aim = w.aim, nil
	}
	return true
}

func (g *Game) tourInput() {
	w := g.tour
	if keyJustPressed(ebiten.KeyR) || mouseJustPressed(ebiten.MouseButtonRight) {
		w.grid = !w.grid
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.zoom = math.Max(1, math.Min(4, g.zoom+math.Copysign(1, wheel)))
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		g.tourClick(g.toCanvas(cursorWorld()))
	}
}

// tourClick sends the hero to the exit under canvas point (x, y), or else to the small tile there.
func (g *Game) tourClick(x, y float64) {
	w := g.tour
	w.aim = nil
	if e := g.tourExitAt(x, y); e != nil {
		if g.walkTour(w.nearestOpen(e.At)) {
			w.aim = e
		}
		return
	}
	if to, ok := w.nearestReachable(g.tourHeroTile(), smallTileAt(g.townCell(x, y))); ok && g.walkTour(to) {
		w.mark, w.markT = to, 0
	}
}

// tourExitAt is the exit whose spot (or a small tile next to it) or whose building's picture lies under (x, y).
func (g *Game) tourExitAt(x, y float64) *tourExit {
	w := g.tour
	at := smallTileAt(g.townCell(x, y))
	a := w.t.Areas[w.area]
	for i := range a.Exits {
		e := &a.Exits[i]
		if abs(at[0]-e.At[0]) <= 1 && abs(at[1]-e.At[1]) <= 1 {
			return e
		}
		if e.Object != nil && g.tourObjectHit(*e.Object, x, y) {
			return e
		}
	}
	return nil
}

// tourObjectHit reports whether canvas point (x, y) falls on the picture of the object anchored on small tile at.
func (g *Game) tourObjectHit(at smallTile, x, y float64) bool {
	f := g.field
	ox, oy := f.OriginX*K, f.OriginY*K
	for _, p := range f.scenePieces {
		if p.Rank != scenemap.RankObject || p.At != [2]int(at) || len(p.Draws) == 0 {
			continue
		}
		d := p.Draws[0]
		img := g.assets.Props[d.Name].Image
		px, py := int(x-ox)-d.X, int(y-oy)-d.Y
		if image.Pt(px, py).In(img.Bounds().Sub(img.Bounds().Min)) {
			_, _, _, alpha := img.At(img.Bounds().Min.X+px, img.Bounds().Min.Y+py).RGBA()
			return alpha > 0
		}
	}
	return false
}

// tourHeroTile is the small tile the hero stands on, or the one they are stepping onto.
func (g *Game) tourHeroTile() smallTile {
	hero := g.scene.actors[g.tour.hero]
	if hero.walk != nil {
		return smallTileAt(hero.walk.tu, hero.walk.tv)
	}
	return smallTileAt(hero.u, hero.v)
}

// walkTour sends the hero along the shortest way (A*, four steps along u and v) to small tile to; false when it
// cannot be reached.
func (g *Game) walkTour(to smallTile) bool {
	w := g.tour
	s := g.scene
	route := w.path(g.tourHeroTile(), to)
	if route == nil {
		return false
	}
	steps := [][2]float64{}
	for _, t := range route[1:] {
		u, v := t.cell()
		steps = append(steps, [2]float64{u, v})
	}
	s.paths[w.hero] = steps
	s.speed[w.hero] = tourStep
	return true
}

func (w *tourWalk) open(t smallTile) bool { return w.walk.Open(t[0], t[1]) }

var smallSteps = [4]smallTile{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

func (w *tourWalk) path(from, to smallTile) []smallTile {
	if !w.open(to) && to != from {
		return nil
	}
	guess := func(t smallTile) int { return abs(t[0]-to[0]) + abs(t[1]-to[1]) }
	cost := map[smallTile]int{from: 0}
	prev := map[smallTile]smallTile{}
	q := &tourQueue{{from, guess(from)}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(tourNode)
		if cur.t == to {
			out := []smallTile{to}
			for t := to; t != from; {
				t = prev[t]
				out = append([]smallTile{t}, out...)
			}
			return out
		}
		for _, d := range smallSteps {
			n := smallTile{cur.t[0] + d[0], cur.t[1] + d[1]}
			if !w.open(n) {
				continue
			}
			if c, seen := cost[n]; !seen || cost[cur.t]+1 < c {
				cost[n], prev[n] = cost[cur.t]+1, cur.t
				heap.Push(q, tourNode{n, cost[n] + guess(n)})
			}
		}
	}
	return nil
}

// nearestReachable is the open small tile reachable from from that lies closest to want.
func (w *tourWalk) nearestReachable(from, want smallTile) (smallTile, bool) {
	seen := map[smallTile]bool{from: true}
	queue := []smallTile{from}
	best, bestD := from, math.Inf(1)
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		if d := math.Hypot(float64(t[0]-want[0]), float64(t[1]-want[1])); d < bestD {
			best, bestD = t, d
		}
		for _, s := range smallSteps {
			n := smallTile{t[0] + s[0], t[1] + s[1]}
			if !seen[n] && w.open(n) {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return best, best != from || w.open(from)
}

type tourNode struct {
	t smallTile
	f int
}

type tourQueue []tourNode

func (q tourQueue) Len() int           { return len(q) }
func (q tourQueue) Less(i, j int) bool { return q[i].f < q[j].f }
func (q tourQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *tourQueue) Push(x any)        { *q = append(*q, x.(tourNode)) }
func (q *tourQueue) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}

// drawTour draws the area, the exits' name plates, the small tile under the cursor and the walkable tiles (R).
func (g *Game) drawTour(dst *ebiten.Image) {
	w := g.tour
	a := w.t.Areas[w.area]
	g.field.groundMarks = g.field.groundMarks[:0]
	if ring := g.assets.FX["click_ring"]; w.markT < townMark && len(ring.Frames) > 0 {
		x, y := g.field.Center(w.mark.cell())
		f := ring.Frames[ring.frameOver(w.markT/townMark)]
		g.field.groundMarks = append(g.field.groundMarks, groundMark{Sprite{f, ring.Anchor[0], ring.Anchor[1]}, x, y})
	}
	g.drawScene(dst, nil)
	if w.grid {
		for v := w.walk.Origin[1]; v < w.walk.Origin[1]+w.walk.H; v++ {
			for u := w.walk.Origin[0]; u < w.walk.Origin[0]+w.walk.W; u++ {
				if w.walk.Open(u, v) {
					g.smallDiamond(dst, smallTile{u, v}, color.NRGBA{}, color.NRGBA{R: 255, G: 255, B: 255, A: 40})
				} else {
					g.smallDiamond(dst, smallTile{u, v}, color.NRGBA{R: 200, G: 40, B: 30, A: 70}, color.NRGBA{})
				}
			}
		}
		for _, e := range a.Exits {
			g.smallDiamond(dst, w.nearestOpen(e.At), color.NRGBA{R: 60, G: 130, B: 255, A: 110}, color.NRGBA{R: 90, G: 160, B: 255, A: 220})
		}
	}
	x, y := g.toCanvas(cursorWorld())
	hover := g.tourExitAt(x, y)
	if w.going == nil && hover == nil {
		if t := smallTileAt(g.townCell(x, y)); w.open(t) {
			g.smallDiamond(dst, t, color.NRGBA{R: 255, G: 240, B: 200, A: 40}, color.NRGBA{R: 255, G: 240, B: 200, A: 140})
		}
	}
	for i := range a.Exits {
		e := &a.Exits[i]
		px, py := g.field.Center(e.At.cell())
		py -= 24
		if e.Object != nil {
			for _, p := range g.field.scenePieces {
				if p.Rank == scenemap.RankObject && p.At == [2]int(*e.Object) {
					px, py = float64(p.Rect.Min.X+p.Rect.Max.X)/2, float64(p.Rect.Min.Y)
				}
			}
		}
		sx, sy := worldToUI(g.toScreen(px, py))
		g.namePlate(dst, e.Name, sx, sy-6, e == hover)
	}
	g.label(dst, w.t.Name+" · "+a.Name+" ("+a.Map+")", 24, 18, 18, gold)
	g.label(dst, "클릭: 이동, 이름판·건물·출구: 다음 장면   휠: 확대   R·오른쪽 클릭: 걸을 수 있는 칸", 24, 46, 13, muted)
	if k := math.Max(w.fade, g.scene.fade); k > 0 {
		inWorld(func() { rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{A: uint8(255 * k)}) })
	}
}

// smallDiamond fills small tile t's ground diamond and draws its rim, over the map at the camera's zoom.
func (g *Game) smallDiamond(dst *ebiten.Image, t smallTile, fill, rim color.NRGBA) {
	x, y := g.field.Center(t.cell())
	var p vector.Path
	pts := [4][2]float32{}
	for i, d := range [][2]float64{{0, -8}, {16, 0}, {0, 8}, {-16, 0}} {
		sx, sy := worldToUI(g.toScreen(x+d[0], y+d[1]))
		pts[i] = [2]float32{float32(sx), float32(sy)}
	}
	p.MoveTo(pts[0][0], pts[0][1])
	for _, q := range pts[1:] {
		p.LineTo(q[0], q[1])
	}
	p.Close()
	if fill.A > 0 {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(fill)
		fillPath(dst, &p, op)
	}
	if rim.A > 0 {
		for i := range pts {
			q := pts[(i+1)%4]
			strokeLine(dst, pts[i][0], pts[i][1], q[0], q[1], 1, rim)
		}
	}
}
