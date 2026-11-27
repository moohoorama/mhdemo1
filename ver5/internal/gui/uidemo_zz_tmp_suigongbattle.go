package gui

import "srpg/internal/core"

func init() { demos["suigong-battle"] = demoSuigongBattle }

// demoSuigongBattle plays B25's real opening: the deployment is placed, then the battle begins through the game's own
// command so the stage's first event (the flood) is replayed as in play. -frames ~1100.
func demoSuigongBattle(d *Demo) error {
	g := d.g
	e, err := core.NewAt(g.S.Data, 1, "c25-battle", "normal")
	if err != nil {
		return err
	}
	for _, id := range e.Observe().Learning {
		if _, err := e.Apply(core.Command{Kind: "learn_close", Actor: id}); err != nil {
			return err
		}
	}
	if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
		return err
	}
	plan, err := e.DeploymentPlan()
	if err != nil {
		return err
	}
	for _, c := range plan[:len(plan)-1] {
		if _, err := e.Apply(c); err != nil {
			return err
		}
	}
	g.S.Engine, g.S.Screen, g.modals = e, "playing", nil
	g.ensureField(e.Observe())
	g.sync(e.Observe())
	d.step = func(f int) {
		if f == 20 {
			g.command(core.Command{Kind: "deployment_finish"})
		}
	}
	return nil
}
