package gui

import "fmt"

func init() { demos["wei"] = demoWei }

// demoWei shows Cao Ren and the troops around him in the Xinye battle.
func demoWei(d *Demo) error {
	stage, officer := "B23", "caoren"
	if d.stage == "B24" {
		stage, officer = "B24", "xiahoudun"
	} else if d.stage == "B27V" {
		stage = "B27V"
	}
	if err := d.battle(stage); err != nil {
		return err
	}
	for _, u := range d.g.S.View().UnitViews {
		if u.Def == officer {
			d.look(u.ID)
			d.g.zoom = 3
			return nil
		}
	}
	return fmt.Errorf("%s is missing from %s", officer, stage)
}
