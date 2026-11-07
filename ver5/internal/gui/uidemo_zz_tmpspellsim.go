package gui

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"srpg/internal/core"
	"srpg/internal/session"
)

// temporary: casts the fifteen simulated spells in turn (troops round the target for the area ones).
var simFXOrder = []string{"blaze", "scorch", "water_dragon", "boulder"}

const simFXGap = 320

func init() {
	demos["zz-spellsim"] = func(d *Demo) error {
		if err := d.battle(d.stage); err != nil {
			return err
		}
		g := d.g
		snap := g.S.Engine.Snapshot()
		var ally, mate, enemy string
		var ax, ay int
		for _, u := range snap.Units {
			if u.Faction == "ally" && ally == "" {
				ally, ax, ay = u.ID, u.X, u.Y
			}
		}
		for _, u := range snap.Units {
			if u.Faction == "ally" && u.ID != ally && mate == "" {
				mate = u.ID
			}
			if u.Faction == "enemy" && u.HP > 0 && enemy == "" && strings.Contains(u.ID, "#") {
				enemy = u.ID
			}
		}
		e := g.S.Engine
		free := func(x, y int, id string) bool {
			for _, u := range snap.Units {
				if u.HP > 0 && u.X == x && u.Y == y && u.ID != id {
					return false
				}
			}
			return e.StepCost(id, x, y) > 0
		}
		dirs := [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}
		ex, ey, mx, my := -1, -1, -1, -1
		for _, p := range dirs {
			if ex < 0 && free(ax+p[0], ay+p[1], enemy) {
				ex, ey = ax+p[0], ay+p[1]
			}
		}
		for _, p := range dirs {
			if mx < 0 && (ax+p[0] != ex || ay+p[1] != ey) && free(ax+p[0], ay+p[1], mate) {
				mx, my = ax+p[0], ay+p[1]
			}
		}
		var extras []string
		for _, u := range snap.Units {
			if u.Faction == "enemy" && u.HP > 0 && u.ID != enemy && strings.Contains(u.ID, "#") && len(extras) < 3 {
				extras = append(extras, u.ID)
			}
		}
		var spots [][2]int
		for _, p := range dirs {
			x, y := ex+p[0], ey+p[1]
			if (x != ax || y != ay) && (x != mx || y != my) && len(spots) < len(extras) {
				spots = append(spots, [2]int{x, y})
			}
		}
		fmt.Fprintln(os.Stderr, "ally", ally, ax, ay, "enemy", enemy, ex, ey, "mate", mate, mx, my)
		g.S.Data.Rules.HitFloor, g.S.Data.Rules.HitCeil = 100, 100
		g.S.Data.Skills = maps.Clone(g.S.Data.Skills)
		reqs := []string{}
		have := map[string]bool{}
		for _, c := range snap.Characters {
			if c.ID == ally {
				for _, t := range c.Learned {
					have[t] = true
				}
				for _, t := range g.S.Data.Characters[c.Def].Traits {
					have[t] = true
				}
			}
		}
		for _, id := range simFXOrder {
			sk := g.S.Data.Skills[id]
			sk.Cost, sk.Coeff = 0, .05
			g.S.Data.Skills[id] = sk
			for _, t := range sk.Requires {
				if !have[t] {
					have[t] = true
					reqs = append(reqs, t)
				}
			}
		}
		first := true
		reset := func() {
			d.edit(func(s *core.State) {
				for i := range s.Characters {
					if s.Characters[i].ID == ally {
						s.Characters[i].XP = 0
						if first {
							s.Characters[i].Learned = append(s.Characters[i].Learned, reqs...)
						}
					}
				}
				for i := range s.Units {
					u := &s.Units[i]
					switch u.ID {
					case ally:
						u.Acted, u.Attacked, u.Moved, u.Done = false, false, false, false
						u.Status = map[string]core.Status{}
					case mate:
						u.X, u.Y = mx, my
						u.Status = map[string]core.Status{}
					case enemy:
						u.X, u.Y = ex, ey
						u.Status = map[string]core.Status{}
					}
					for j, id := range extras {
						if u.ID == id && j < len(spots) {
							u.X, u.Y = spots[j][0], spots[j][1]
							u.Status = map[string]core.Status{}
						}
					}
				}
			})
			first = false
		}
		reset()
		g.zoom, g.zoomFrom, g.zoomTo = 3, 3, 3
		g.camX, g.camY = g.field.Center(float64(ex)+.5, float64(ey)+.5)
		g.camY -= 30
		d.step = func(f int) {
			i := (f - 20) / simFXGap
			if f < 20 || (f-20)%simFXGap != 0 || i >= len(simFXOrder) {
				return
			}
			reset()
			id := simFXOrder[i]
			s := g.S.Data.Skills[id]
			target := enemy
			if s.Kind == "heal" || s.Kind == "buff" {
				target = mate
			}
			r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "skill", Actor: ally, Target: target, Skill: id}})
			fmt.Fprintf(os.Stderr, "frame %d %s ok=%v err=%v\n", f, id, r.OK, r.Error)
		}
		return nil
	}
}
