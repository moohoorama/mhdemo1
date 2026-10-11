package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
	"testing"
)

func setup(t *testing.T) *Session {
	t.Helper()
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(d, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func request(t *testing.T, s *Session, q Request) Response {
	t.Helper()
	if s.Engine != nil {
		rev := s.Engine.Observe().Revision
		q.Revision = &rev
	}
	r := s.Handle(q)
	if !r.OK {
		t.Fatalf("%s: %s", q.Op, r.Error)
	}
	return r
}
func snapshot(s *Session) string { b, _ := json.Marshal(s.Engine.Snapshot()); return string(b) }
func TestSaveLoadMenusAndRevision(t *testing.T) {
	s := setup(t)
	request(t, s, Request{Op: "new", Seed: 1})
	before := snapshot(s)
	request(t, s, Request{Op: "menu"})
	request(t, s, Request{Op: "resume"})
	if snapshot(s) != before {
		t.Fatal("menu advances game")
	}
	r := s.Handle(Request{Op: "command", Command: core.Command{Kind: "next"}})
	if r.OK || r.Error != "StaleRevision" || before != snapshot(s) {
		t.Fatal("stale revision")
	}
	request(t, s, Request{Op: "save", Slot: "1"})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "next"}})
	if s.Handle(Request{Op: "load", Slot: "1"}).Error != "DiscardRequired" {
		t.Fatal("discard intent")
	}
	v := storage.DefaultSettings()
	v.AI = "step"
	request(t, s, Request{Op: "settings", Settings: &v})
	request(t, s, Request{Op: "load", Slot: "1", Discard: true})
	if snapshot(s) != before || s.Settings.AI != "step" {
		t.Fatal("load changed settings or state")
	}
	if s.Handle(Request{Op: "save", Slot: "1"}).Error != "OverwriteRequired" {
		t.Fatal("overwrite")
	}
	request(t, s, Request{Op: "save", Slot: "1", Overwrite: true})
	before = snapshot(s)
	p := filepath.Join(s.Store.Dir, "slot-1.json")
	_ = os.WriteFile(p, []byte("bad"), 0600)
	if s.Handle(Request{Op: "load", Slot: "1", Discard: true}).OK || snapshot(s) != before {
		t.Fatal("bad load replaced game")
	}
	bad := s.Engine.Snapshot()
	bad.Core = "future"
	if err := s.Store.Save("2", bad, false); err != nil {
		t.Fatal(err)
	}
	if s.Handle(Request{Op: "load", Slot: "2", Discard: true}).Error != "IncompatibleSave" || snapshot(s) != before {
		t.Fatal("incompatible load replaced game")
	}
	request(t, s, Request{Op: "title", Discard: true})
	if s.Engine != nil || s.Screen != "title" {
		t.Fatal("title")
	}
	if s.Handle(Request{Op: "load", Slot: "2", Discard: true}).OK {
		t.Fatal("title incompatible load")
	}
}
func TestEnemyStepSaveContinuation(t *testing.T) {
	s := setup(t)
	request(t, s, Request{Op: "new", Seed: 4})
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}, {Kind: "place", Actor: "hero", X: 1, Y: 2}, {Kind: "place", Actor: "archer1", X: 1, Y: 3}, {Kind: "deployment_finish"}, {Kind: "move", Actor: "hero", X: 2, Y: 3}, {Kind: "end"}} {
		request(t, s, Request{Op: "command", Command: c})
	}
	request(t, s, Request{Op: "ai"})
	request(t, s, Request{Op: "save", Slot: "1"})
	before := snapshot(s)
	a := request(t, s, Request{Op: "ai"})
	after := snapshot(s)
	request(t, s, Request{Op: "load", Slot: "1", Discard: true})
	if snapshot(s) != before {
		t.Fatal("enemy cursor")
	}
	b := request(t, s, Request{Op: "ai"})
	ab, _ := json.Marshal(a.Events)
	bb, _ := json.Marshal(b.Events)
	if string(ab) != string(bb) || after != snapshot(s) {
		t.Fatal("enemy replay mismatch")
	}
}
func TestResolvePhase(t *testing.T) {
	s := setup(t)
	request(t, s, Request{Op: "new", Seed: 3})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "next"}})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "choose", Option: "go"}})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "start"}})
	request(t, s, Request{Op: "auto"})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
	// The same phase resolved command by command must end in the same state.
	twin := setup(t)
	twin.Engine, _ = core.Restore(s.Data, s.Engine.Snapshot())
	actions, err := s.ResolvePhase()
	if err != nil {
		t.Fatal(err)
	}
	for twin.Engine.Observe().Turn == "enemy" {
		if _, err := ai.Step(twin.Engine); err != nil {
			t.Fatal(err)
		}
	}
	if snapshot(s) != snapshot(twin) {
		t.Fatal("resolved phase differs from step-by-step AI")
	}
	if len(actions) < 2 || !actions[len(actions)-1].Barrier {
		t.Fatalf("actions: %d, last barrier %v", len(actions), actions[len(actions)-1].Barrier)
	}
	for i, a := range actions {
		for _, j := range a.After {
			if j >= i {
				t.Fatalf("action %d waits for later action %d", i, j)
			}
		}
	}
	// Any two actions sharing a claim must be ordered through the precedence graph.
	reach := func(from, to int) bool {
		seen := map[int]bool{}
		var walk func(int) bool
		walk = func(i int) bool {
			if i == to {
				return true
			}
			if seen[i] {
				return false
			}
			seen[i] = true
			for _, j := range actions[i].After {
				if walk(j) {
					return true
				}
			}
			return false
		}
		return walk(from)
	}
	for i := range actions {
		for j := 0; j < i; j++ {
			shared := false
			for _, a := range actions[i].Claims() {
				for _, b := range actions[j].Claims() {
					shared = shared || a == b
				}
			}
			if shared && !reach(i, j) {
				t.Fatalf("actions %d and %d share a claim but are unordered", j, i)
			}
		}
	}
}

func TestPartialDeploymentSaveAndAuto(t *testing.T) {
	s := setup(t)
	request(t, s, Request{Op: "new", Seed: 7})
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}, {Kind: "place", Actor: "hero", X: 1, Y: 2}} {
		request(t, s, Request{Op: "command", Command: c})
	}
	request(t, s, Request{Op: "save", Slot: "1"})
	before := snapshot(s)
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "unplace", Actor: "hero"}})
	request(t, s, Request{Op: "load", Slot: "1", Discard: true})
	if snapshot(s) != before {
		t.Fatal("partial deployment not restored")
	}
	request(t, s, Request{Op: "auto"})
	o := s.Engine.Observe()
	if o.Phase != "battle" || o.Round != 1 || o.Turn != "ally" {
		t.Fatal("auto did not complete deployment")
	}
	for _, u := range o.Units {
		if u.Faction == "ally" && (u.Moved || u.Acted || u.Done) {
			t.Fatal("auto continued into battle")
		}
	}
}

func TestEnemyTurnLearning(t *testing.T) {
	for _, mode := range []string{"resolve", "advance", "auto", "ai"} {
		for _, learn := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/learn=%t", mode, learn), func(t *testing.T) {
				s := setup(t)
				s.Data.Rules.ExpDefenseShare = .5
				request(t, s, Request{Op: "new", Seed: 3})
				for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}} {
					request(t, s, Request{Op: "command", Command: c})
				}
				for _, c := range []core.Command{{Kind: "place", Actor: "hero", X: 1, Y: 2}, {Kind: "place", Actor: "archer1", X: 1, Y: 3}, {Kind: "deployment_finish"}} {
					request(t, s, Request{Op: "command", Command: c})
				}
				request(t, s, Request{Op: "command", Command: core.Command{Kind: "end"}})
				state := s.Engine.Snapshot()
				for i := range state.Characters {
					if state.Characters[i].ID == "hero" {
						state.Characters[i].XP = core.ExperienceRequired(s.Data, state.Characters[i].Level) - 1
						state.Characters[i].Points = 900
					}
				}
				for i := range state.Units {
					if state.Units[i].ID == "boss" {
						state.Units[i].X, state.Units[i].Y = 2, 2
					}
				}
				var err error
				s.Engine, err = core.Restore(s.Data, state)
				if err != nil {
					t.Fatal(err)
				}
				advance := func() []core.Event {
					t.Helper()
					switch mode {
					case "resolve":
						actions, err := s.ResolvePhase()
						if err != nil {
							t.Fatal(err)
						}
						var events []core.Event
						for _, a := range actions {
							for _, step := range a.Steps {
								events = append(events, step.Events...)
							}
						}
						return events
					case "advance":
						events, err := s.AdvanceEnemies()
						if err != nil {
							t.Fatal(err)
						}
						return events
					default:
						return request(t, s, Request{Op: mode}).Events
					}
				}
				var events []core.Event
				for i := 0; i < 100 && len(s.Engine.Observe().Learning) == 0; i++ {
					events = append(events, advance()...)
				}
				o := s.Engine.Observe()
				if o.Turn != "enemy" || len(o.Learning) != 1 || o.Learning[0] != "hero" {
					t.Fatalf("expected learning during enemy turn, got %s / %v", o.Turn, o.Learning)
				}
				leveled := false
				for _, e := range events {
					leveled = leveled || e.Kind == "level" && e.Actor == "hero"
					if e.Kind == "learn" {
						t.Fatal("enemy AI learned a player trait")
					}
				}
				if !leveled {
					t.Fatal("enemy attack did not level the hero")
				}
				before := snapshot(s)
				if mode == "ai" {
					rev := o.Revision
					r := s.Handle(Request{Op: "ai", Revision: &rev})
					if r.OK || r.Error != "LearningPending" {
						t.Fatalf("AI during learning: %+v", r)
					}
				} else {
					advance()
				}
				if snapshot(s) != before {
					t.Fatal("pending learning did not pause enemy AI")
				}
				request(t, s, Request{Op: "save", Slot: "1"})
				request(t, s, Request{Op: "load", Slot: "1", Discard: true})
				if snapshot(s) != before {
					t.Fatal("save/load did not preserve pending learning")
				}
				if learn {
					r := request(t, s, Request{Op: "command", Command: core.Command{Kind: "learn", Actor: "hero", Trait: "poison_t"}})
					if len(r.Events) != 1 || r.Events[0].Kind != "learn" || r.Events[0].Target != "poison_t" {
						t.Fatalf("player learning events: %v", r.Events)
					}
				}
				request(t, s, Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: "hero"}})
				rev := s.Engine.Observe().Revision
				r := s.Handle(Request{Op: "command", Revision: &rev, Command: core.Command{Kind: "end"}})
				if r.OK || r.Error != "NotYourTurn" {
					t.Fatalf("player ended enemy turn: %+v", r)
				}
				for i := 0; i < 100 && s.Engine.Observe().Phase == "battle" && s.Engine.Observe().Turn == "enemy"; i++ {
					advance()
				}
				if o := s.Engine.Observe(); o.Phase == "battle" && o.Turn == "enemy" {
					t.Fatal("enemy turn did not resume after learning")
				}
			})
		}
	}
}
