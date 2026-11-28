package gui

import (
	"fmt"
	"time"

	"srpg/internal/core"
	"srpg/internal/session"
	"srpg/internal/storage"
)

func init() { demos["chronicle"] = demoChronicle }

// demoChronicle fills the chronicle with the points of a game that has reached the 서주 전투 and gone on a few
// waypoints on the 고순만 생존 arm, and of a second that took the 둘 다 생존 arm, and shows the window: it opens on the
// title at frame 10, the other arm's chip is picked at 90, the list is scrolled to the 서주 갈림길 header at 150, the question for a
// point is asked at 210, the menu of a running game with its 지난 이야기 row at 270, and from 330 the point loaded.
// The states are waypoint starts (core.NewAt), not real play, so only the window can be judged from it.
func demoChronicle(d *Demo) error {
	g := d.g
	st, data := g.S.Store, g.S.Data
	ways := data.Waypoints()
	b12 := -1
	for i, w := range ways {
		if w.Node == "c12-battle" {
			b12 = i
		}
	}
	if b12 < 0 {
		return fmt.Errorf("chronicle demo needs the 서주 전투")
	}
	base := time.Now().Add(-9 * 24 * time.Hour)
	put := func(i int, route string, at time.Time) error {
		e, err := core.NewAt(data, 1, ways[i].Node, "")
		if err != nil {
			return err
		}
		return st.ChroniclePut(storage.ChronicleMeta{Node: ways[i].Node, Route: route, At: at, Level: 3 + i*2/3}, e.Snapshot())
	}
	routeOf := func(i int) string {
		switch f := data.ForkAt(ways[i].Node); {
		case f == nil:
			return ""
		case f.ID == "julu":
			return "seen"
		}
		return "gaoshun"
	}
	for i := 0; i <= b12+4 && i < len(ways); i++ {
		if err := put(i, routeOf(i), base.Add(time.Duration(i)*97*time.Minute)); err != nil {
			return err
		}
	}
	for k := 1; k <= 3 && b12+k < len(ways); k++ {
		if err := put(b12+k, "both", base.Add(30*time.Hour+time.Duration(k)*80*time.Minute)); err != nil {
			return err
		}
	}
	g.openOverlay("chronicle")
	d.step = func(f int) {
		switch f {
		case 90:
			g.chron.arm["xuzhou"] = "gaoshun"
			g.buildChronicle()
		case 150:
			for _, r := range g.chron.rows {
				if r.fork != nil && r.fork.ID == "xuzhou" {
					g.chron.list.top = r.top
				}
			}
		case 210:
			for _, r := range g.chron.rows {
				if r.fork == nil && r.w.Node == "c12-battle" {
					g.askChronicle(r)
				}
			}
		case 270:
			g.closeOverlays()
			g.S.Engine, g.S.Screen = core.New(data, 1), "playing"
			g.openOverlay("menu")
		case 330:
			g.closeOverlays()
			g.request(session.Request{Op: "chronicle_load", Node: "c12-battle", Discard: true})
		}
	}
	return nil
}
