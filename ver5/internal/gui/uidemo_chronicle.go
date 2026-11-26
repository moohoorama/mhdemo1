package gui

import (
	"fmt"
	"time"

	"srpg/internal/core"
	"srpg/internal/session"
	"srpg/internal/storage"
)

func init() { demos["chronicle"] = demoChronicle }

// demoChronicle fills the chronicle with two lines and shows the window: the first line has reached the story up to
// the 12th battle and a few points after it; the second left it at that battle's point and has played on again. Its states
// are waypoint starts (core.NewAt), not real play, so only the window can be judged from it. The window opens on the
// title at frame 10, the first line's tab at 90, scrolled up at 150, the question for a point at 210, the menu
// of a running game with its 지난 이야기 row at 300, and from 330 the first line's 12th-battle point loaded.
func demoChronicle(d *Demo) error {
	g := d.g
	st, data := g.S.Store, g.S.Data
	ways := data.Waypoints()
	b12, battles := -1, 0
	for i, w := range ways {
		if w.Stage != "" {
			if battles++; battles == 12 {
				b12 = i
			}
		}
	}
	if b12 < 0 {
		return fmt.Errorf("chronicle demo needs 12 battles")
	}
	base := time.Now().Add(-9 * 24 * time.Hour)
	add := func(line string, seq, i int, at time.Time, level int) error {
		e, err := core.NewAt(data, 1, ways[i].Node, "")
		if err != nil {
			return err
		}
		s := e.Snapshot()
		s.Run, s.Steps = line, seq
		return st.ChronicleAdd(storage.ChronicleMeta{Line: line, Seq: seq, Node: ways[i].Node, At: at, Level: level}, s)
	}
	first, err := st.ChronicleBegin("", 0, base)
	if err != nil {
		return err
	}
	seq := 0
	var shared time.Time
	for i := 0; i <= b12+4 && i < len(ways); i++ {
		seq++
		at := base.Add(time.Duration(i) * 97 * time.Minute)
		if err := add(first, seq, i, at, 3+i*2/3); err != nil {
			return err
		}
		if i == b12 {
			shared = at
		}
	}
	second, err := st.ChronicleBegin(first, b12+1, shared.Add(26*time.Hour))
	if err != nil {
		return err
	}
	for k := 1; k <= 3 && b12+k < len(ways); k++ {
		if err := add(second, b12+1+k, b12+k, shared.Add(26*time.Hour+time.Duration(k)*80*time.Minute), 3+(b12+k)*2/3); err != nil {
			return err
		}
	}
	g.openOverlay("chronicle")
	d.step = func(f int) {
		switch f {
		case 90:
			g.pickChronicleLine(0)
		case 150:
			g.chron.list.top = 0
		case 210:
			l := g.chron.lines[0]
			g.askChronicle(l, l.Entries[b12])
		case 270:
			g.closeOverlays()
			g.S.Engine, g.S.Screen = core.New(data, 1), "playing"
			g.openOverlay("menu")
		case 330:
			g.closeOverlays()
			g.request(session.Request{Op: "chronicle_load", Line: g.chron.lines[0].ID, Seq: b12 + 1, Discard: true})
		}
	}
	return nil
}
