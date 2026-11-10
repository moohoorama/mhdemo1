package gui

import (
	"fmt"
	"maps"
	"slices"
	"os"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// temporary: three blows (one landing, one blocked, one critical) and one skill of each sound in turn, so the
// sound effects can be heard over a capture (SRPG_SFX_TRACE).
var sfxDemoOrder = []string{
	"attack", "block", "crit",
	"scorch", "blaze", "torrent", "rockfall", "boulder", "minor_supply", "rouse", "weaken", "confuse",
}

const sfxDemoGap = 170

func init() {
	demos["zz-sfx"] = func(d *Demo) error {
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
			if u.Faction == "enemy" && u.HP > 0 && enemy == "" {
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
		g.S.Data.Stages = slices.Clone(g.S.Data.Stages)
		for i := range g.S.Data.Stages {
			g.S.Data.Stages[i].Duels = nil // no duel or story event cuts in between the blows
			g.S.Data.Stages[i].Events = nil
		}
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
		for _, id := range sfxDemoOrder {
			sk, ok := g.S.Data.Skills[id]
			if !ok {
				continue
			}
			sk.Cost, sk.Coeff = 0, .05
			g.S.Data.Skills[id] = sk
			for _, t := range sk.Requires {
				if !have[t] {
					have[t] = true
					reqs = append(reqs, t)
				}
			}
		}
		hp := map[string]int{}
		for _, u := range snap.Units {
			hp[u.ID] = u.HP
		}
		first := true
		reset := func() {
			g.modals = nil
			d.edit(func(s *core.State) {
				for i := range s.Characters {
					if s.Characters[i].ID == ally {
						s.Characters[i].XP = 0 // no level-up window between the blows
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
						u.X, u.Y, u.HP = ex, ey, hp[u.ID]
						u.Status = map[string]core.Status{}
					}
				}
			})
			first = false
		}
		reset()
		g.zoom, g.zoomFrom, g.zoomTo = 2, 2, 2
		g.camX, g.camY = g.field.Center(float64(ax)+.5, float64(ay)+.5)
		d.step = func(f int) {
			i := (f - 20) / sfxDemoGap
			if f < 20 || (f-20)%sfxDemoGap != 0 || i >= len(sfxDemoOrder) {
				return
			}
			reset()
			id := sfxDemoOrder[i]
			rules := &g.S.Data.Rules
			rules.HitFloor, rules.HitCeil = 100, 100
			rules.Crit = content.Chance{}
			c := core.Command{Kind: "attack", Actor: ally, Target: enemy}
			switch id {
			case "block":
				rules.HitFloor, rules.HitCeil = 0, 0
			case "crit":
				rules.Crit = content.Chance{Base: 100, Max: 100}
			case "attack":
			default:
				target := enemy
				if s := g.S.Data.Skills[id]; s.Kind == "heal" || s.Kind == "buff" {
					target = mate
				}
				c = core.Command{Kind: "skill", Actor: ally, Target: target, Skill: id}
			}
			r := g.request(session.Request{Op: "command", Command: c})
			fmt.Fprintf(os.Stderr, "frame %d %s ok=%v err=%v\n", f, id, r.OK, r.Error)
		}
		return nil
	}
}
