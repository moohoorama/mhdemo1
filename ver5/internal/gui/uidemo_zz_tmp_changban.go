package gui

import "srpg/internal/core"

func init() {
	demos["changban-finale"] = demoChangbanFinale
	demos["redcliff-opening"] = demoRedcliffOpening
}

// demoChangbanFinale plays the last legal move and the full bridge replay on the real campaign stage.
func demoChangbanFinale(d *Demo) error {
	g := d.g
	e, err := core.NewAt(g.S.Data, 1, "c26-battle", "normal")
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
	for _, c := range plan {
		if _, err := e.Apply(c); err != nil {
			return err
		}
	}
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == "zhaoyun" {
			snap.Units[i].X, snap.Units[i].Y, snap.Units[i].Facing = 17, 30, "SW"
		}
	}
	e, err = core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	g.S.Engine, g.S.Screen, g.modals = e, "playing", nil
	g.ensureField(e.Observe())
	g.sync(e.Observe())
	d.look("zhaoyun")
	moved, waited := false, false
	d.step = func(frame int) {
		if frame < 20 {
			return
		}
		if len(g.modals) > 0 && (!g.busy() || g.briefed()) {
			g.closeModal()
			return
		}
		if g.busy() {
			return
		}
		switch {
		case !moved:
			g.command(core.Command{Kind: "move", Actor: "zhaoyun", X: 17, Y: 31})
			moved = true
		case !waited:
			g.command(core.Command{Kind: "wait", Actor: "zhaoyun"})
			waited = true
		}
	}
	return nil
}

func demoRedcliffOpening(d *Demo) error {
	g := d.g
	e, err := core.NewAt(g.S.Data, 1, "c27-redcliff-battle", "normal")
	if err != nil {
		return err
	}
	for _, id := range e.Observe().Learning {
		if _, err := e.Apply(core.Command{Kind: "learn_close", Actor: id}); err != nil {
			return err
		}
	}
	g.S.Engine, g.S.Screen, g.modals = e, "playing", nil
	d.step = func(frame int) {
		switch {
		case frame == 5:
			g.command(core.Command{Kind: "start"})
		case frame == 60 && e.Deploying():
			g.command(core.Command{Kind: "deployment_finish"})
		case frame >= 100 && g.briefed():
			g.closeModal()
		}
	}
	return nil
}
