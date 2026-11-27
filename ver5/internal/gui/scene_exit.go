package gui

import (
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
