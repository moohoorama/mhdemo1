package ai

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

// RouteEnemyTurns is what one enemy by the way costs a walk to a goal, in rounds: about what the player's
// units lose fighting it off on the way (measured by a since-removed simulation: half a round to most of one each).
const RouteEnemyTurns = .5

// routeThreat is an opponent and how near a way has to pass to meet it: four cells, or one for a unit
// that holds its place, which a walk past it does not wake.
type routeThreat struct {
	id    string
	p     content.Point
	reach int
}

func routeThreats(o core.Observation, me *core.UnitView) []routeThreat {
	var out []routeThreat
	for _, v := range o.UnitViews {
		if v.Faction != me.Faction && v.HP > 0 {
			reach := 4
			if v.Policy == "hold" {
				reach = 1
			}
			out = append(out, routeThreat{v.ID, content.Point{X: v.X, Y: v.Y}, reach})
		}
	}
	return out
}

func (t routeThreat) near(p content.Point) bool { return abs(p.X-t.p.X)+abs(p.Y-t.p.Y) <= t.reach }

// goalRoute is how far each cell is from the goal cells along the way that costs me the fewest rounds,
// and that way from my cell: the steps over my movement, and RouteEnemyTurns for every opponent near the
// way. The ways tried are the shortest walk and walks that keep further and further from the opponents.
func goalRoute(e *core.Engine, o core.Observation, me *core.UnitView, goal []content.Point) (map[content.Point]int, []content.Point) {
	threats := routeThreats(o, me)
	danger := func(p content.Point) int {
		n := 0
		for _, t := range threats {
			if t.near(p) {
				n++
			}
		}
		return n
	}
	step := float64(max(1, e.Data().Classes[me.Class].Move))
	var best map[content.Point]int
	var bestWay []content.Point
	bestTurns := 0.0
	for _, k := range []int{0, 1, 2, 4, 8} {
		var extra func(content.Point) int
		if k > 0 {
			extra = func(p content.Point) int { return k * danger(p) }
		}
		dist := walkingFromWith(e, o, me, goal, extra)
		way := downhill(dist, content.Point{X: me.X, Y: me.Y})
		if way == nil {
			return dist, nil
		}
		met := map[string]bool{}
		walked := 0
		for _, p := range way {
			walked += max(1, e.StepCost(me.ID, p.X, p.Y))
			for _, t := range threats {
				if t.near(p) {
					met[t.id] = true
				}
			}
		}
		turns := float64(walked)/step + RouteEnemyTurns*float64(len(met))
		if best == nil || turns < bestTurns {
			best, bestWay, bestTurns = dist, way, turns
		}
	}
	return best, bestWay
}

// downhill is the way down the field from p to a cell at 0, without p; nil when p is off the field.
func downhill(dist map[content.Point]int, p content.Point) []content.Point {
	if _, ok := dist[p]; !ok {
		return nil
	}
	var way []content.Point
	for dist[p] > 0 && len(way) < 999 {
		next := p
		for _, d := range []content.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			q := content.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if v, ok := dist[q]; ok && v < dist[next] {
				next = q
			}
		}
		if next == p {
			break
		}
		p = next
		way = append(way, p)
	}
	return way
}

// escort is what the others do while a unit of theirs walks to a goal: the cells of its way within two
// of its moves, to keep up with and get ahead to, and the opponents near that stretch, to clear first.
func escort(e *core.Engine, o core.Observation, leader *core.UnitView, way []content.Point) (ahead []content.Point, blockers []string) {
	n := min(len(way), 2*max(1, e.Data().Classes[leader.Class].Move))
	ahead = way[:n]
	for _, t := range routeThreats(o, leader) {
		for _, p := range ahead {
			if t.near(p) {
				blockers = append(blockers, t.id)
				break
			}
		}
	}
	return ahead, blockers
}
