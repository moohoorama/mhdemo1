package gui

import (
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["learn"] = demoLearn }

// demoLearn opens the first officer's learning window at the demo stage's deployment with points to
// spend: the table (10), the first trait picked (30), sorted by cost (50), the dearest trait picked
// with what it lacks (70), the cheapest learned (90) and the window back after its replay (150).
func demoLearn(d *Demo) error {
	g := d.g
	if err := demoDeploySetup(d); err != nil {
		return err
	}
	id := g.S.View().Deployment[0]
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			if s.Characters[i].ID == id {
				s.Characters[i].Points = 800
			}
		}
		s.Learning = []string{id}
	})
	pick := func(at int) {
		rows := g.learnRows(g.learn[0])
		sortRows(rows, g.learnColumns(), g.learnWin.list.order, func(r *learnRow) bool { return r.ok })
		if at < 0 {
			at += len(rows)
		}
		g.learnWin.pick = rows[at].id
	}
	d.step = func(f int) {
		g.learn = g.S.View().Learners // the demo's frames skip Update, which keeps it
		if len(g.learn) == 0 {
			return
		}
		switch f {
		case 30:
			pick(0)
		case 50:
			g.learnWin.list.order = tableSort{"cost", 2}
		case 70:
			pick(-1)
		case 85:
			g.learnWin.list.order = tableSort{}
			pick(0)
		case 90:
			g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: id, Trait: g.learnWin.pick}})
			g.learnWin.pick = ""
		}
	}
	return nil
}
