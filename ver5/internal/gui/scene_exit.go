package gui

import (
	"container/heap"
	"math"
	"strconv"

	"srpg/internal/scenemap"
)

// stepLift is the most a walk may go up or down from one small tile to the next: a stair's step (the pavilion's
// steps, scene.json heights 5 and 6), not a cliff or a wall.
const stepLift = 8

// exitGround is what a scene's exit walks need of its map: the playing area in cells and, on a scene map of format 2,
// which small tiles can be stood on and how high each step is.
type exitGround struct {
	w, h int
	m    *scenemap.Map
	walk *scenemap.Walk
	lift func(int) int
}

func newExitGround(art *MapArt, rules *scenemap.Catalog) *exitGround {
	gr := &exitGround{w: art.W, h: art.H}
	if art.SceneMap != nil && rules != nil {
		gr.m, gr.walk = art.SceneMap, scenemap.Footing(art.SceneMap, rules)
		gr.lift = func(step int) int { return rules.Rules.Heights[strconv.Itoa(step)].Lift }
	}
	return gr
}

// stepOpen reports whether a straight walk from a to b stays on open small tiles of one height level, leaving out
// the tiles off the map's grid and the one the walker starts on.
func (gr *exitGround) stepOpen(a, b [2]float64, start smallTile) bool {
	if gr.walk == nil {
		return true
	}
	inside := func(t smallTile) bool { return gr.m.Inside(t[0], t[1]) }
	n := int(math.Ceil(math.Max(math.Abs(b[0]-a[0]), math.Abs(b[1]-a[1])) * 4))
	prev := smallTileAt(a[0], a[1])
	for i := 1; i <= n; i++ {
		k := float64(i) / float64(n)
		t := smallTileAt(a[0]+(b[0]-a[0])*k, a[1]+(b[1]-a[1])*k)
		if t == prev || !inside(t) {
			continue
		}
		if d := gr.lift(gr.m.Step(t[0], t[1])) - gr.lift(gr.m.Step(prev[0], prev[1])); inside(prev) && (d > stepLift || d < -stepLift) {
			return false
		}
		prev = t
		if t != start && !gr.walk.Open(t[0], t[1]) {
			return false
		}
	}
	return true
}

// facesOut reports whether an actor at p, walking on toward dir, goes out over the front edge of the map: its
// south-east or south-west side, which the camera looks at.
func (gr *exitGround) facesOut(p [2]float64, dir string) bool {
	return dir == "SE" && p[0] >= float64(gr.w-1)-1e-9 || dir == "SW" && p[1] >= float64(gr.h-1)-1e-9
}

// exitSteps are the map steps of a route: the two facing the camera cost one each, the two turned away four.
var exitSteps = []struct {
	du, dv float64
	dir    string
	cost   int
}{{1, 0, "SE", 1}, {0, 1, "SW", 1}, {-1, 0, "NW", 4}, {0, -1, "NE", 4}}

// route is the cells an actor at from, last facing dir, walks to leave over the front edge of the map: nothing when it
// already stands at the edge facing out, the cheapest way over open ground otherwise (steps toward the camera by
// preference, never through a closed small tile or another actor's cell when busy says so, where it can help it),
// ending on a step toward the camera so that it keeps facing out as it walks off the map. ok is false when no
// way exists.
func (gr *exitGround) route(from [2]float64, dir string, busy func(u, v float64) bool) (path [][2]float64, ok bool) {
	if gr.facesOut(from, dir) {
		return nil, true
	}
	type node struct{ i, j int }
	loU, loV, hiU, hiV := math.Min(0, from[0]), math.Min(0, from[1]), math.Max(float64(gr.w), from[0]+1), math.Max(float64(gr.h), from[1]+1)
	at := func(n node) [2]float64 { return [2]float64{from[0] + float64(n.i), from[1] + float64(n.j)} }
	home := smallTileAt(from[0], from[1])
	dist, prev, done := map[node]int{{}: 0}, map[node]node{}, map[node]bool{}
	goal, goalCost, goalFrom := node{}, math.MaxInt, node{}
	for {
		cur, best := node{}, math.MaxInt
		for n, d := range dist {
			if !done[n] && d < best {
				cur, best = n, d
			}
		}
		if best == math.MaxInt || best >= goalCost {
			break
		}
		done[cur] = true
		for _, s := range exitSteps {
			next := node{cur.i + int(s.du), cur.j + int(s.dv)}
			p, q := at(cur), at(next)
			if q[0] < loU || q[1] < loV || q[0] > hiU || q[1] > hiV || !gr.stepOpen(p, q, home) {
				continue
			}
			cost := best + s.cost
			if busy != nil && busy(q[0], q[1]) {
				cost += 3
			}
			if gr.facesOut(q, s.dir) && cost < goalCost {
				goal, goalCost, goalFrom = next, cost, cur
			}
			if d, seen := dist[next]; !done[next] && (!seen || cost < d) {
				dist[next], prev[next] = cost, cur
			}
		}
	}
	if goalCost == math.MaxInt {
		return nil, false
	}
	for n := goalFrom; n != (node{}); n = prev[n] {
		path = append([][2]float64{at(n)}, path...)
	}
	return append(path, at(goal)), true
}

// exitTurn is what a turn costs a walk to a map exit, in cells: walks with fewer corners win.
const exitTurn = 3

// exitApproach is how many cells before its exit a walk runs straight in the exit's way, so that whoever leaves by it
// goes out along the same line.
const exitApproach = 2

// mapExits are the map's named exits in cells: the exit's cell and the cell step of its way out.
func (gr *exitGround) mapExits() []scenemap.Exit {
	if gr.m == nil {
		return nil
	}
	return gr.m.Exits
}

// exitCell is an exit's small tile as a scene position (an odd tile is a whole cell).
func exitCell(e scenemap.Exit) [2]float64 {
	return [2]float64{float64(e.At[0]-1) / 2, float64(e.At[1]-1) / 2}
}

// throughExit is the cells an actor at from walks to leave by the map exit named name (by the nearest one when
// name is ""): over open ground to the cell exitApproach cells inside the exit, turning as little as it can, then
// straight on in the exit's way through the exit's cell and on to the edge of the board, where it fades out. dir is the way out, for the walk on past it. ok is false when the
// map has no such exit or no way leads to it.
func (gr *exitGround) throughExit(from [2]float64, name string, busy func(u, v float64) bool) (path [][2]float64, dir string, ok bool) {
	var exits []scenemap.Exit
	for _, e := range gr.mapExits() {
		if name == "" || e.ID == name {
			exits = append(exits, e)
		}
	}
	if len(exits) == 0 {
		return nil, "", false
	}
	start := [2]float64{math.Round(from[0]), math.Round(from[1])}
	home := smallTileAt(from[0], from[1])
	type cell struct{ i, j int }
	at := func(c cell) [2]float64 { return [2]float64{start[0] + float64(c.i), start[1] + float64(c.j)} }
	steps := [4]cell{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}
	type state struct {
		c   cell
		dir int // the step it came by, 4 at the start
	}
	goals := map[cell]int{} // approach cell → exit index
	for i, e := range exits {
		ec, st := exitCell(e), scenemap.ExitStep[e.Dir]
		a := [2]float64{ec[0] - float64(st[0]*exitApproach), ec[1] - float64(st[1]*exitApproach)}
		if a[0]-start[0] != math.Trunc(a[0]-start[0]) || a[1]-start[1] != math.Trunc(a[1]-start[1]) {
			continue // not on this walker's grid of whole cells
		}
		goals[cell{int(a[0] - start[0]), int(a[1] - start[1])}] = i
	}
	if len(goals) == 0 {
		return nil, "", false
	}
	pad := 2.0
	lo := [2]float64{-pad, -pad}
	hi := [2]float64{float64(gr.w) + pad, float64(gr.h) + pad}
	if gr.m != nil {
		lo = [2]float64{float64(gr.m.Origin[0])/2 - pad, float64(gr.m.Origin[1])/2 - pad}
		hi = [2]float64{float64(gr.m.Origin[0]+len(gr.m.Ground[0]))/2 + pad, float64(gr.m.Origin[1]+len(gr.m.Ground))/2 + pad}
	}
	dist := map[state]int{{cell{}, 4}: 0}
	prev := map[state]state{}
	h := &exitHeap{{state: [3]int{0, 0, 4}}}
	var goal state
	found := false
	for h.Len() > 0 {
		top := heap.Pop(h).(exitItem)
		cur := state{cell{top.state[0], top.state[1]}, top.state[2]}
		if top.cost > dist[cur] {
			continue
		}
		if _, ok := goals[cur.c]; ok {
			goal, found = cur, true
			break
		}
		for d, s := range steps {
			next := cell{cur.c.i + s.i, cur.c.j + s.j}
			p, q := at(cur.c), at(next)
			if q[0] < lo[0] || q[1] < lo[1] || q[0] > hi[0] || q[1] > hi[1] || !gr.stepOpen(p, q, home) {
				continue
			}
			cost := top.cost + 1
			if cur.dir != 4 && cur.dir != d {
				cost += exitTurn
			}
			if busy != nil && busy(q[0], q[1]) {
				cost += 3
			}
			ns := state{next, d}
			if old, seen := dist[ns]; !seen || cost < old {
				dist[ns], prev[ns] = cost, cur
				heap.Push(h, exitItem{state: [3]int{next.i, next.j, d}, cost: cost})
			}
		}
	}
	if !found {
		return nil, "", false
	}
	e := exits[goals[goal.c]]
	st := scenemap.ExitStep[e.Dir]
	var cells []cell
	for s := goal; s.dir != 4; s = prev[s] {
		cells = append([]cell{s.c}, cells...)
	}
	if start != from {
		path = append(path, start)
	}
	for _, c := range cells {
		path = append(path, at(c))
	}
	last := at(goal.c)
	for k := 1; k <= exitApproach; k++ {
		q := [2]float64{last[0] + float64(st[0]), last[1] + float64(st[1])}
		if !gr.stepOpen(last, q, home) {
			return nil, "", false
		}
		last = q
		path = append(path, q)
	}
	// on past the exit along its way to the board's edge, fading only once off it; a closed tile stops the walk there
	for gr.m != nil {
		q := [2]float64{last[0] + float64(st[0]), last[1] + float64(st[1])}
		t := smallTileAt(q[0], q[1])
		if !gr.m.Inside(t[0], t[1]) || !gr.stepOpen(last, q, home) {
			break
		}
		path = append(path, q)
		last = q
	}
	return path, e.Dir, true
}

type exitItem struct {
	state [3]int
	cost  int
}

type exitHeap []exitItem

func (h exitHeap) Len() int           { return len(h) }
func (h exitHeap) Less(i, j int) bool { return h[i].cost < h[j].cost }
func (h exitHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *exitHeap) Push(x any)        { *h = append(*h, x.(exitItem)) }
func (h *exitHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}
