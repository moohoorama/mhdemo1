package gui

import (
	"strconv"
	"strings"

	"srpg/internal/core"
)

func init() { demos["hills"] = demoHills }

// demoHills opens a battle with rolling hills (u) and mountains (n), stands the first ally the player moves on cell
// -node "U,V" (a free hill cell) and starts its order, so its move range shows: the unit stands on the hill's crest
// while the marks stay on the ground. Without -node it stays where the battle put it.
//
//	go run ./cmd/_uidemo-vm -ui hills -stage B03 -node 7,9 -frames 90 -at 80 -size 2560x1700
func demoHills(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	id := d.firstAlly()
	for _, u := range g.S.View().UnitViews {
		if u.Faction == "ally" && !u.AI {
			id = u.ID
			break
		}
	}
	if su, sv, ok := strings.Cut(d.node, ","); ok {
		u, err1 := strconv.Atoi(su)
		v, err2 := strconv.Atoi(sv)
		if err1 == nil && err2 == nil {
			d.edit(func(s *core.State) {
				for i := range s.Units {
					if s.Units[i].ID == id {
						s.Units[i].X, s.Units[i].Y = u, v
					}
				}
			})
		}
	}
	d.step = func(f int) {
		if f == 30 {
			if u := g.unitView(g.S.View(), id); u != nil {
				g.begin(*u)
			}
		}
		d.look(id)
	}
	g.zoom = 2
	return nil
}
