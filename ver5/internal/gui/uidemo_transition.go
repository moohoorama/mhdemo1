package gui

import (
	"fmt"
	"slices"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["transition"] = demoTransition }

// demoTransition reads from its chapter's start to the -node story node and on from there, a line every 90 frames, into whatever screen comes
// next (the deployment after a battle node), so the change between the two screens can be watched.
func demoTransition(d *Demo) error {
	if d.node == "" {
		return fmt.Errorf("transition needs -node")
	}
	g := d.g
	var start string
	for _, c := range g.S.Data.Chapters {
		if slices.Contains(c.Nodes, d.node) {
			start = c.Nodes[0]
		}
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: start})
	if g.S.Engine == nil {
		return fmt.Errorf("cannot start at %s", start)
	}
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	for i := 0; i < 200 && g.S.View().Node != d.node; i++ {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
	}
	if n := g.S.View().Node; n != d.node {
		return fmt.Errorf("stopped at %s on the way to %s", n, d.node)
	}
	g.S.Settings.InstantText = true
	shown := 0
	d.step = func(f int) {
		defer g.watchScreen()
		if g.holdForTransition() {
			return
		}
		g.turnPage()
		if o := g.S.View(); o.Phase == "scenario" && o.Dialogue.Kind != "battle" {
			if g.sceneBusy() {
				shown = f
			} else if f-shown >= 90 {
				shown = f
				g.advanceDialogue()
			}
		}
		if o := g.S.View(); o.Phase == "scenario" && o.Dialogue.Kind == "battle" {
			g.startSortie() // in the same update as the click that reached it, as Update does
		}
	}
	return nil
}
