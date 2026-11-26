package gui

import (
	"fmt"
	"maps"
	"os"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["counterspell"] = demoCounterspell }

func demoCounterspell(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	snap := g.S.Engine.Snapshot()
	var ally, allyChar, enemy, enemyChar string
	var ax, ay, ex, ey int
	for _, u := range snap.Units {
		if u.Faction == "ally" && ally == "" {
			ally, allyChar, ax, ay = u.ID, u.Character, u.X, u.Y
		}
		if u.Faction == "enemy" && u.HP > 0 && enemy == "" {
			enemy, enemyChar = u.ID, u.Character
		}
	}
	if ally == "" || enemy == "" {
		return fmt.Errorf("missing demo units")
	}
	found := false
	for _, p := range [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		x, y := ax+p[0], ay+p[1]
		free := g.S.Engine.StepCost(enemy, x, y) > 0
		for _, u := range snap.Units {
			if u.HP > 0 && u.ID != enemy && u.X == x && u.Y == y {
				free = false
			}
		}
		if free {
			ex, ey = x, y
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no adjacent demo cell")
	}
	g.S.Data.Skills = maps.Clone(g.S.Data.Skills)
	skillID := ""
	for id, skill := range g.S.Data.Skills {
		if skill.Kind == content.KindMagic && skill.Element == "fire" && skill.Shape == content.ShapeSingle && len(skill.Requires) == 1 && skill.Min <= 1 {
			skillID = id
			break
		}
	}
	if skillID == "" {
		return fmt.Errorf("no single-target fire spell")
	}
	sk := g.S.Data.Skills[skillID]
	sk.Cost, sk.Coeff = 0, .05
	g.S.Data.Skills[skillID] = sk
	g.S.Data.Rules.SpellFloor, g.S.Data.Rules.SpellCeil = 100, 100
	reflectTrait := ""
	for id, trait := range g.S.Data.Traits {
		if trait.Effects[content.FxReflect] > 0 {
			reflectTrait = id
			break
		}
	}
	if reflectTrait == "" {
		return fmt.Errorf("no reflecting trait")
	}
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			c := &s.Characters[i]
			if c.ID == allyChar {
				c.Learned = append(c.Learned, sk.Requires...)
			}
			if c.ID == enemyChar {
				c.Learned = append(c.Learned, reflectTrait)
			}
		}
		for i := range s.Units {
			if s.Units[i].ID == enemy {
				s.Units[i].X, s.Units[i].Y = ex, ey
			}
		}
	})
	g.zoom, g.zoomFrom, g.zoomTo = 2, 2, 2
	g.camX, g.camY = g.field.Center(float64(ax)+.5, float64(ay)+.5)
	d.step = func(f int) {
		if f == 20 {
			r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "skill", Actor: ally, Target: enemy, Skill: skillID}})
			fmt.Fprintf(os.Stderr, "counterspell demo actor=%s target=%s ok=%v error=%v events=%+v\n", ally, enemy, r.OK, r.Error, r.Events)
		}
	}
	return nil
}
