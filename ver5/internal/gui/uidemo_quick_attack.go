package gui

import (
	"fmt"
	"srpg/internal/core"
)

func init() { demos["quick-attack"] = demoQuickAttack }
func demoQuickAttack(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	var actor core.UnitView
	var targetX, targetY int
	found := false
	for _, u := range g.S.View().UnitViews {
		if u.Faction != "ally" || u.AI || u.Done {
			continue
		}
		for _, c := range g.S.Engine.LegalActions(u.ID).Commands {
			if c.Kind == "move" && abs(c.X-u.X)+abs(c.Y-u.Y) >= 3 {
				actor = u
				targetX, targetY = c.X, c.Y
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("no movable ally")
	}
	var enemy string
	d.edit(func(s *core.State) {
		for i := range s.Units {
			if s.Units[i].Faction != actor.Faction && s.Units[i].HP > 0 {
				s.Units[i].X, s.Units[i].Y = targetX, targetY
				enemy = s.Units[i].ID
				break
			}
		}
	})
	if enemy == "" {
		return fmt.Errorf("no enemy")
	}
	selected, clicked := false, false
	d.step = func(f int) {
		if f >= 40 && !g.busy() && !selected {
			g.mapClick(actor.X, actor.Y)
			selected = true
			fmt.Printf("selected: %s stage=%s targets=%v\n", actor.ID, g.ord.stage, g.ord.attackFrom)
		}
		if f >= 160 && !g.busy() && selected && !clicked {
			g.mapClick(targetX, targetY)
			clicked = true
			fmt.Printf("clicked: stage=%s pick=%s\n", g.ord.stage, g.ord.pick.Kind)
		}
		d.look(actor.ID)
	}
	g.zoom = 2
	return nil
}
