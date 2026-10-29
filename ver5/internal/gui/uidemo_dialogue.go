package gui

import (
	"fmt"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["dialogue"] = demoDialogue }

// demoDialogue starts a new game and steps through the opening story lines, so the
// scroll opens, types, and springs to each next line's width.
func demoDialogue(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	d.step = func(f int) {
		if f > 0 && f%150 == 0 {
			g.advanceDialogue()
			g.advanceDialogue()
		}
	}
	return nil
}

func init() { demos["autotext"] = demoAutoText }

// demoAutoText starts a new game with the auto setting on and leaves the story to page itself.
func demoAutoText(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	g.S.Settings.AutoText = true
	d.step = func(int) { g.turnPage() }
	return nil
}

func init() { demos["sceneskip"] = demoSceneSkip }

// demoSceneSkip jumps to the -node story node (chapter 3's start, whose two first scenes share a road) and presses the skip button a little into
// the first scene and again right after it stops.
func demoSceneSkip(d *Demo) error {
	if d.node == "" {
		return fmt.Errorf("sceneskip needs -node")
	}
	g := d.g
	g.request(session.Request{Op: "jump", Seed: 1, Node: d.node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	d.step = func(f int) {
		if (f == 240 || f == 250) && g.scene != nil && g.skipping == nil && g.uiSeconds >= g.skipUntil {
			g.skipping = g.scene.def
		}
		g.turnPage()
	}
	return nil
}
