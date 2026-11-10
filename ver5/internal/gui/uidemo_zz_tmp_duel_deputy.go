package gui

import (
	"slices"

	"srpg/internal/core"
)

func init() { demos["dueldeputy"] = demoDuelDeputy }

// demoDuelDeputy plays the 사수관 duel with 관우 riding as 유비's deputy: 관우 leaves the field, 유비 stands next to
// 화웅 and waits (frame 30), and the duel opens with 관우 fighting it for 유비's unit.
func demoDuelDeputy(d *Demo) error {
	if err := d.battle("B05"); err != nil {
		return err
	}
	g := d.g
	snap := g.S.Engine.Snapshot()
	var target core.Unit
	for _, u := range snap.Units {
		if u.Character == "huaxiong" {
			target = u
		}
	}
	snap.Units = slices.DeleteFunc(snap.Units, func(u core.Unit) bool { return u.Character == "guanyu" })
	snap.Deployment = slices.DeleteFunc(snap.Deployment, func(id string) bool { return id == "guanyu" })
	for i := range snap.Units {
		if snap.Units[i].Character == "liubei" {
			snap.Units[i].X, snap.Units[i].Y = target.X+1, target.Y
		}
	}
	for i := range snap.Characters {
		switch c := &snap.Characters[i]; c.ID {
		case "liubei":
			if !slices.Contains(c.Learned, "deputy_appoint") {
				c.Learned = append(c.Learned, "deputy_appoint")
			}
			c.Deputy = "guanyu"
		case "guanyu":
			c.Equipment = map[string]string{}
		}
	}
	e, err := core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.vis = map[string]*unitVis{}
	g.sync(g.S.View())
	d.look("liubei")
	d.step = func(f int) {
		if f == 30 {
			g.confirm(core.Command{Kind: "wait", Actor: "liubei"})
		}
	}
	return nil
}
