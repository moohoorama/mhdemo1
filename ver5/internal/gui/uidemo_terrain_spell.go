package gui

import (
	"fmt"

	"srpg/internal/core"
)

func init() { demos["terrainspell"] = demoTerrainSpell }

func demoTerrainSpell(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	snapshot := g.S.Engine.Snapshot()
	var skillID string
	for id, skill := range g.S.Data.Skills {
		if skill.TerrainTurns > 0 && skill.Element == "fire" {
			skillID = id
			break
		}
	}
	if skillID == "" {
		return fmt.Errorf("no fire terrain spell")
	}
	var actor string
	for i := range snapshot.Units {
		u := &snapshot.Units[i]
		if u.Faction == "ally" && !u.AI {
			actor, u.MP = u.ID, 100
			for j := range snapshot.Characters {
				c := &snapshot.Characters[j]
				if c.ID == u.Character {
					c.Learned = append(c.Learned, g.S.Data.Skills[skillID].Requires...)
				}
			}
			break
		}
	}
	engine, err := core.Restore(g.S.Data, snapshot)
	if err != nil {
		return err
	}
	g.S.Engine = engine
	g.sync(g.S.View())
	for _, action := range engine.LegalActions(actor).Commands {
		if action.Skill != skillID || !g.field.Inside(action.X-1, action.Y-1) || !g.field.Inside(action.X+1, action.Y+1) {
			continue
		}
		fits := true
		for y := action.Y - 1; y <= action.Y+1; y++ {
			for x := action.X - 1; x <= action.X+1; x++ {
				cell := engine.Cell(x, y)
				cell.State = g.S.Data.Rules.CritMarks["fire"][0]
				fits = fits && g.S.Data.ValidCell(cell) == ""
			}
		}
		if !fits {
			continue
		}
		g.camX, g.camY = g.field.Center(float64(action.X), float64(action.Y))
		d.step = func(frame int) {
			switch frame {
			case 1:
				g.ord = order{actor: actor, stage: "target", pick: action}
				g.hover = hoverCell{action.X, action.Y, true}
			case 30:
				g.hover = hoverCell{}
				g.confirm(action)
			}
		}
		return nil
	}
	return fmt.Errorf("no legal terrain target")
}
