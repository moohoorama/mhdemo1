package gui

import "srpg/internal/core"

func init() { demos["waypoints"] = demoWaypoints }

// demoWaypoints opens the waypoint window on the start screen, at frame 30 sets its setup (level 38, two
// battle flags), scrolls both its lists to the end from frame 60, shows the 2막-가상 waypoints at frame 125,
// from frame 150 starts at the last battle on the list as set up with its learning windows open, and from frame 180
// closes them on the sortie screen; from 210 the first officer's equipment, from 240 its weapon picker.
func demoWaypoints(d *Demo) error {
	g := d.g
	g.openOverlay("waypoints")
	d.step = func(frame int) {
		switch {
		case frame == 30:
			flags := g.S.Data.StoryFlags()
			g.jumpLevel, g.jumpFlags = 38, map[string]bool{flags[0].Flag: true, flags[len(flags)-1].Flag: true}
		case frame >= 60 && frame < 120:
			g.waypoints.list.top += 40
			g.waypoints.flags.top += 40
		case frame == 125:
			g.pickWaypointAct(g.waypointActFor(g.S.Data.WaypointActs()), "2막-가상")
		case frame == 150:
			ways := g.S.Data.Waypoints()
			g.closeOverlays()
			g.jump(ways[len(ways)-1].Node, "")
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
