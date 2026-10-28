package gui

import "srpg/internal/core"

func init() {
	demos["aiturn"] = func(d *Demo) error { return demoAITurn(d, "together") }
	demos["aiturn-one"] = func(d *Demo) error { return demoAITurn(d, "one") }
}

// demoAITurn ends the allied phase with the camera on the first officer and plays the AI armies' turns
// (allied, then enemy): each action waits until it is on screen, the camera glides to the next
// one and the log follows the replay.
func demoAITurn(d *Demo, replay string) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.S.Settings.Replay = replay
	d.step = func(f int) {
		switch f {
		case 1:
			for _, u := range g.S.View().UnitViews {
				if u.Faction == "ally" && !u.AI {
					d.look(u.ID)
					break
				}
			}
		case 20:
			g.command(core.Command{Kind: "end"})
		}
		// the editor's trial battle is not the "playing" screen whose loop starts the AI turns
		if o := g.S.View(); f > 20 && o.Phase == "battle" && o.Turn != "ally" && !g.busy() {
			g.resolvePhase()
		}
	}
	return nil
}
