package gui

import "srpg/internal/core"

func init() { demos["forecast"] = demoForecast }

func demoForecast(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	snap := g.S.Engine.Snapshot()
	actor, enemy := "", ""
	ax, ay, ei := 0, 0, -1
	for i, u := range snap.Units {
		if u.Faction == "ally" && actor == "" {
			actor, ax, ay = u.ID, u.X, u.Y
		}
		if u.Faction == "enemy" && enemy == "" {
			enemy, ei = u.ID, i
		}
	}
	for i, u := range snap.Units {
		if u.X == ax+1 && u.Y == ay {
			snap.Units[i].X, snap.Units[i].Y = snap.Units[ei].X, snap.Units[ei].Y
		}
	}
	snap.Units[ei].X, snap.Units[ei].Y = ax+1, ay
	var err error
	g.S.Engine, err = core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	g.sync(g.S.View())
	g.camX, g.camY = g.field.Center(float64(ax)+.5, float64(ay)+.5)
	d.step = func(f int) {
		if f >= 1 {
			t := g.unitView(g.S.View(), enemy)
			g.ord = order{actor: actor, stage: "target", pick: core.Command{Kind: "attack", Actor: actor}}
			g.hover = hoverCell{t.X, t.Y, true}
		}
	}
	return nil
}
