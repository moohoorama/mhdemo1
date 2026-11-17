package gui

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"srpg/internal/core"
	"srpg/internal/session"
)

// temporary: casts 혼란 (lands), 대혼란 (lands), then 혼란 that misses, so the question marks, the loop over a
// confused unit and the block on a miss can be captured.
var confuseFXOrder = []string{"confuse", "great_confusion", "confuse"}

const confuseFXGap = 240

func init() {
	demos["zz-confusefx"] = func(d *Demo) error {
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
		for _, id := range confuseFXOrder {
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
				}
			})
			first = false
		}
		reset()
		g.zoom, g.zoomFrom, g.zoomTo = 2, 2, 2
		g.camX, g.camY = g.field.Center(float64(ax)+.5, float64(ay)+.5)
		marks := [][]string{{"disarm"}, {"seal"}, {"root"}, {"poison"}, {"confusion", "poison"}, {"disarm", "seal", "root"},
			{"confusion", "disarm", "seal", "root", "poison"}}
		d.step = func(f int) {
			if f == 20 {
				reset()
				g.S.Data.Rules.SpellFloor, g.S.Data.Rules.SpellCeil = 100, 100
				r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "skill", Actor: ally, Target: enemy, Skill: "confuse"}})
				fmt.Fprintf(os.Stderr, "frame %d confuse ok=%v err=%v\n", f, r.OK, r.Error)
			}
			i := (f - 200) / 120
			if f < 200 || (f-200)%120 != 0 || i >= len(marks) {
				return
			}
			d.edit(func(st *core.State) {
				for k := range st.Units {
					if st.Units[k].ID == enemy {
						st.Units[k].Status = map[string]core.Status{}
						for _, m := range marks[i] {
							st.Units[k].Status[m] = core.Status{Turns: 2}
						}
					}
				}
			})
			fmt.Fprintf(os.Stderr, "frame %d marks %v\n", f, marks[i])
		}
		return nil
	}
}
