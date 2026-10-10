package gui

import (
	"testing"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func battleController(t *testing.T) *Game {
	t.Helper()
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(d, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{S: s}
	if r := g.send(session.Request{Op: "new", Seed: 17}); !r.OK {
		t.Fatal(r.Error)
	}
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}} {
		if r := g.send(session.Request{Op: "command", Command: c}); !r.OK {
			t.Fatal(r.Error)
		}
	}
	return g
}

func TestCompletedActionClearsSelection(t *testing.T) {
	for _, kind := range []string{"item", "wait"} {
		t.Run(kind, func(t *testing.T) {
			g := battleController(t)
			g.selected = "hero"
			r := g.send(session.Request{Op: "command", Command: core.Command{Kind: kind, Actor: "hero", Target: "hero", Item: "tonic"}})
			if !r.OK {
				t.Fatal(r.Error)
			}
			check(t, g.selected, "")
		})
	}
}

func TestRecoveryPreviewUpdatesAfterCommand(t *testing.T) {
	g := battleController(t)
	o := g.S.Engine.Observe()
	u := g.unitView(o, "hero")
	before := g.unitRecovery(u.ID, o.Revision)
	if r := g.send(session.Request{Op: "command", Command: core.Command{Kind: "item", Actor: u.ID, Target: u.ID, Item: "tonic"}}); !r.OK {
		t.Fatal(r.Error)
	}
	o = g.S.Engine.Observe()
	got := g.unitRecovery(u.ID, o.Revision)
	check(t, got, g.S.Engine.NextTurnRecovery(u.ID))
	if g.recoveryRevision == 0 || g.recoveryRevision != o.Revision {
		t.Fatal("cached the recovery from the previous command")
	}
	if before.MP == 0 {
		t.Fatal("fixture should need spell points initially")
	}
}
