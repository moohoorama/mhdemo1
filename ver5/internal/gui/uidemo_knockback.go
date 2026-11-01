package gui

import (
	"fmt"

	"srpg/internal/content"
	"srpg/internal/core"
)

func init() { demos["knockback"] = demoKnockback }

// demoKnockback has the first ally strike an enemy set beside it with 돌파 and every blow landing: the enemy
// takes the blow in its hit motion and slides a cell back without a step.
func demoKnockback(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	e := g.S.Engine
	snap := e.Snapshot()
	free := func(x, y int, id string) bool {
		for _, u := range snap.Units {
			if u.HP > 0 && u.X == x && u.Y == y && u.ID != id {
				return false
			}
		}
		return e.StepCost(id, x, y) > 0
	}
	var ally, enemy string
	var ax, ay, dx, dy int
	for _, u := range snap.Units {
		if u.Faction == "ally" && ally == "" {
			ally, ax, ay = u.ID, u.X, u.Y
		}
	}
	for _, u := range snap.Units {
		if u.Faction == "enemy" && u.HP > 0 && enemy == "" {
			enemy = u.ID
		}
	}
	for _, p := range [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		if dx == 0 && dy == 0 && free(ax+p[0], ay+p[1], enemy) && free(ax+2*p[0], ay+2*p[1], enemy) {
			dx, dy = p[0], p[1]
		}
	}
	if ally == "" || enemy == "" || dx == 0 && dy == 0 {
		return fmt.Errorf("no room to push an enemy beside %s", ally)
	}
	knock := ""
	for id, t := range g.S.Data.Traits {
		if t.Effects[content.FxKnockback] > 0 {
			knock = id
		}
	}
	if knock == "" {
		return fmt.Errorf("no trait pushes a target back")
	}
	g.S.Data.Rules.HitFloor, g.S.Data.Rules.HitCeil = 100, 100
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			if s.Characters[i].ID == ally {
				s.Characters[i].Learned = append(s.Characters[i].Learned, knock)
			}
		}
		for i := range s.Units {
			if s.Units[i].ID == enemy {
				s.Units[i].X, s.Units[i].Y = ax+dx, ay+dy
			}
		}
	})
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	g.camX, g.camY = g.field.Center(float64(ax+dx)+.5, float64(ay+dy)+.5)
	d.step = func(f int) {
		if f == 30 {
			g.confirm(core.Command{Kind: "attack", Actor: ally, Target: enemy})
		}
	}
	return nil
}
