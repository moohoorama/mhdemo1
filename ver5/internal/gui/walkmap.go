package gui

import (
	"container/heap"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/scenemap"
)

// Walking on a scene map of format 2 (the explore screen, the tour of cmd/_town2): the hero steps small tile by
// small tile along u and v where the map rules (scenemap.Walkable) let him, round whoever stands in the way.

const (
	walkFade = .35 // seconds the screen goes black through an exit
	markTime = .6  // seconds the click ring spreads on the ground
	heroStep = 2.0 // the hero's walking speed: a scene walk's step is a game cell, a small step half one
)

// smallTile is a small tile (U, V): half a game cell. Its centre is game cell point ((U - .5) / 2, (V - .5) / 2).
type smallTile [2]int

func (t smallTile) cell() (float64, float64) {
	return (float64(t[0]) - .5) / 2, (float64(t[1]) - .5) / 2
}

// smallTileAt is the small tile a game cell point lies on.
func smallTileAt(u, v float64) smallTile {
	return smallTile{int(math.Floor(2*u + 1)), int(math.Floor(2*v + 1))}
}

var smallSteps = [4]smallTile{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// the small tile under the cursor where the hero can walk
var cursorFill, cursorRim = color.NRGBA{R: 255, G: 240, B: 200, A: 40}, color.NRGBA{R: 255, G: 240, B: 200, A: 140}

// walkGrid is which small tiles of a map can be walked now: the map's own (scenemap.Walkable) less those someone
// stands on.
type walkGrid struct {
	walk    *scenemap.Walk
	blocked map[smallTile]bool
}

func (w *walkGrid) open(t smallTile) bool { return w.walk.Open(t[0], t[1]) && !w.blocked[t] }

// nearestOpen is the open small tile nearest t (t itself when open).
func (w *walkGrid) nearestOpen(t smallTile) smallTile {
	best, bestD := t, math.Inf(1)
	for v := w.walk.Origin[1]; v < w.walk.Origin[1]+w.walk.H; v++ {
		for u := w.walk.Origin[0]; u < w.walk.Origin[0]+w.walk.W; u++ {
			if d := math.Hypot(float64(u-t[0]), float64(v-t[1])); w.open(smallTile{u, v}) && d < bestD {
				best, bestD = smallTile{u, v}, d
			}
		}
	}
	return best
}

// path is the small tiles from from to the nearest open one of goals, stepping along u or v only, by A* (the
// distance to the nearest goal guides it); nil when none can be reached.
func (w *walkGrid) path(from smallTile, goals ...smallTile) []smallTile {
	var ends []smallTile
	for _, t := range goals {
		if w.open(t) || t == from {
			ends = append(ends, t)
		}
	}
	if len(ends) == 0 {
		return nil
	}
	guess := func(t smallTile) int {
		best := math.MaxInt
		for _, e := range ends {
			best = min(best, abs(t[0]-e[0])+abs(t[1]-e[1]))
		}
		return best
	}
	cost := map[smallTile]int{from: 0}
	prev := map[smallTile]smallTile{}
	q := &walkQueue{{from, guess(from)}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(walkNode)
		if guess(cur.t) == 0 {
			out := []smallTile{cur.t}
			for t := cur.t; t != from; {
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
				heap.Push(q, walkNode{n, cost[n] + guess(n)})
			}
		}
	}
	return nil
}

// reachable is every open small tile that can be walked to from from, with its distance in steps.
func (w *walkGrid) reachable(from smallTile) map[smallTile]int {
	dist := map[smallTile]int{from: 0}
	queue := []smallTile{from}
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		for _, s := range smallSteps {
			n := smallTile{t[0] + s[0], t[1] + s[1]}
			if _, seen := dist[n]; !seen && w.open(n) {
				dist[n] = dist[t] + 1
				queue = append(queue, n)
			}
		}
	}
	return dist
}

// nearestReachable is the small tile reachable from from that lies closest to want, nearer by the walk when two
// are as close.
func (w *walkGrid) nearestReachable(from, want smallTile) (smallTile, bool) {
	best, bestD, bestW := from, math.Inf(1), 0
	for t, steps := range w.reachable(from) {
		d := math.Hypot(float64(t[0]-want[0]), float64(t[1]-want[1]))
		if d < bestD || d == bestD && (steps < bestW || steps == bestW && (t[0] < best[0] || t[0] == best[0] && t[1] < best[1])) {
			best, bestD, bestW = t, d, steps
		}
	}
	return best, best != from || w.open(from)
}

type walkNode struct {
	t smallTile
	f int // steps so far plus the guess to the nearest goal
}

type walkQueue []walkNode

func (q walkQueue) Len() int           { return len(q) }
func (q walkQueue) Less(i, j int) bool { return q[i].f < q[j].f }
func (q walkQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *walkQueue) Push(x any)        { *q = append(*q, x.(walkNode)) }
func (q *walkQueue) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}

// walkActor sends a scene actor along route (its first tile is where it stands) at speed.
func (g *Game) walkActor(key string, route []smallTile, speed float64) {
	s := g.scene
	steps := [][2]float64{}
	for _, t := range route[1:] {
		u, v := t.cell()
		steps = append(steps, [2]float64{u, v})
	}
	s.paths[key] = steps
	s.speed[key] = speed
}

// actorTile is the small tile a scene actor stands on, or the one it is stepping onto.
func (g *Game) actorTile(key string) smallTile {
	a := g.scene.actors[key]
	if a.walk != nil {
		return smallTileAt(a.walk.tu, a.walk.tv)
	}
	return smallTileAt(a.u, a.v)
}

// cellAt is the (fractional) game cell under canvas point (x, y), on the ground.
func (g *Game) cellAt(x, y float64) (float64, float64) {
	f := g.field
	a, b := (x/K-f.OriginX)/16, (y/K-f.OriginY)/8-1
	return (a + b) / 2, (b - a) / 2
}

// objectTop is the top centre on the canvas of the picture of the map object anchored on small tile at.
func (g *Game) objectTop(at smallTile) (float64, float64, bool) {
	for _, p := range g.field.scenePieces {
		if p.Rank == scenemap.RankObject && p.At == [2]int(at) {
			return float64(p.Rect.Min.X+p.Rect.Max.X) / 2, float64(p.Rect.Min.Y), true
		}
	}
	return 0, 0, false
}

// objectHit reports whether canvas point (x, y) falls on the picture of the object anchored on small tile at.
func (g *Game) objectHit(at smallTile, x, y float64) bool {
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

// clickRing puts the spreading click ring on small tile at, t seconds after the click.
func (g *Game) clickRing(at smallTile, t float64) {
	g.field.groundMarks = g.field.groundMarks[:0]
	if ring := g.assets.FX["click_ring"]; t < markTime && len(ring.Frames) > 0 {
		x, y := g.field.Center(at.cell())
		f := ring.Frames[ring.frameOver(t/markTime)]
		g.field.groundMarks = append(g.field.groundMarks, groundMark{Sprite{f, ring.Anchor[0], ring.Anchor[1]}, x, y})
	}
}

// drawWalkGrid shades the small tiles that cannot be walked red and marks those of exits blue (R or a right click).
func (g *Game) drawWalkGrid(dst *ebiten.Image, w *walkGrid, exits []smallTile) {
	for v := w.walk.Origin[1]; v < w.walk.Origin[1]+w.walk.H; v++ {
		for u := w.walk.Origin[0]; u < w.walk.Origin[0]+w.walk.W; u++ {
			if w.open(smallTile{u, v}) {
				g.smallDiamond(dst, smallTile{u, v}, color.NRGBA{}, color.NRGBA{R: 255, G: 255, B: 255, A: 40})
			} else {
				g.smallDiamond(dst, smallTile{u, v}, color.NRGBA{R: 200, G: 40, B: 30, A: 70}, color.NRGBA{})
			}
		}
	}
	for _, t := range exits {
		g.smallDiamond(dst, w.nearestOpen(t), color.NRGBA{R: 60, G: 130, B: 255, A: 110}, color.NRGBA{R: 90, G: 160, B: 255, A: 220})
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

// namePlate is a small dark plate with a name, centred on UI point (x, y)'s bottom edge; a red one marks a place
// to go into.
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

// blackOut covers the world with black at strength k (a fade).
func blackOut(dst *ebiten.Image, k float64) {
	if k > 0 {
		inWorld(func() { rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{A: uint8(255 * min(1, k))}) })
	}
}
