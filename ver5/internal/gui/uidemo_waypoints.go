package gui

import (
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["waypoints"] = demoWaypoints }

// demoWaypoints opens the waypoint window on the start screen, scrolls it to the end from frame 60,
// from frame 150 starts at the last battle on the list with its learning windows open, and from frame 180
// closes them on the sortie screen; from 210 the first officer's equipment, from 240 its weapon picker.
func demoWaypoints(d *Demo) error {
	g := d.g
	g.openOverlay("waypoints")
	d.step = func(frame int) {
		switch {
		case frame >= 60 && frame < 120:
			g.waypoints.top += 20
		case frame == 150:
			ways := g.S.Data.Waypoints()
			g.closeOverlays()
			g.request(session.Request{Op: "jump", Node: ways[len(ways)-1].Node, Seed: g.seed})
			g.learn = g.S.View().Learners
		case frame == 180:
			for _, l := range g.learn {
				g.command(core.Command{Kind: "learn_close", Actor: l.ID})
			}
			g.learn = nil
		case frame == 210:
			g.openOverlay("equipment")
		case frame == 240:
			g.openPicker("weapon")
		}
	}
	return nil
}
