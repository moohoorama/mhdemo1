package gui

import (
	"fmt"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["scrollbar"] = demoScrollbar }

// demoScrollbar opens the sortie preparation of the -node battle node (default c25-battle: ten officers, so the detail
// table scrolls) with the table scrolled three rows down, and shows the six scrollbar styles of scrollbar_style.go one after another: style k is on
// from frame 30k to 30k+29, so the frames 25, 55, 85, 115, 145 and 175 show 현행 and A to E on the same screen. Each
// style is for comparing the designs only, not a chosen one.
func demoScrollbar(d *Demo) error {
	g := d.g
	node := d.node
	if node == "" {
		node = "c25-battle"
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	cursorPos = func() (int, int) { return -1000, -1000 }
	d.step = func(f int) {
		if f%30 == 0 {
			scrollBarStyle = (f / 30) % len(scrollBarStyles)
		}
		if f == 20 {
			g.party.officerList.top = 3
		}
	}
	if len(g.S.View().Deployment) == 0 {
		return fmt.Errorf("scrollbar: %s has no officers to sortie", node)
	}
	return nil
}
