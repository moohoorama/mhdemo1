package gui

import (
	"math"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/replay"
)

// followMovers points the camera at the enemy phase as it plays (docs/ui-mockup, idea 3):
// of the enemy actions running together, the one whose cell is nearest the middle of all
// of them. The camera only moves when that group changes, so it does not wander while
// the same units act.
func (p *Player) followMovers() {
	g := p.g
	ids := []string{}
	cells := map[string]content.Point{}
	for _, r := range p.running {
		a := p.actions[r.index]
		u := g.unitView(p.final, a.Actor)
		if u == nil || u.Faction != "enemy" {
			continue
		}
		ids = append(ids, a.Actor)
		cells[a.Actor] = actionCell(a, u.X, u.Y)
	}
	if len(ids) == 0 {
		return
	}
	sort.Strings(ids)
	key := strings.Join(ids, ",")
	if key == p.followed {
		return
	}
	p.followed = key
	var cx, cy float64
	for _, c := range cells {
		cx += float64(c.X) / float64(len(cells))
		cy += float64(c.Y) / float64(len(cells))
	}
	best, bestD := ids[0], math.Inf(1)
	for _, id := range ids {
		c := cells[id]
		if d := math.Hypot(float64(c.X)-cx, float64(c.Y)-cy); d < bestD {
			best, bestD = id, d
		}
	}
	g.panToCell(cells[best].X, cells[best].Y)
}

// actionCell is where an action plays out: its move's destination, else the actor's
// cell before it.
func actionCell(a *replay.Action, x, y int) content.Point {
	if p, ok := a.Before[a.Actor]; ok {
		x, y = p.X, p.Y
	}
	for _, s := range a.Steps {
		if s.Command.Kind == "move" {
			x, y = s.Command.X, s.Command.Y
		}
	}
	return content.Point{X: x, Y: y}
}
