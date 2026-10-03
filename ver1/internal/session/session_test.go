package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
	"testing"
)

func setup(t *testing.T) *Session {
	t.Helper()
	d, err := content.Load("../../assets/content/campaign.json")
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
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "결의"}, {Kind: "start"}, {Kind: "move", Actor: "유비", X: 3, Y: 5}, {Kind: "end"}} {
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
