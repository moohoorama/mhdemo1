package session

import (
	"log"
	"srpg/internal/core"
	"srpg/internal/storage"
	"time"
)

// The chronicle (지난 이야기, docs/system-chronicle.md): each time the story really reaches a chapter's start or a
// battle, the state the player had there is kept as the point of that place (and of the arm of its fork the
// game is on), replacing the point the place had. Points are kept apart from the manual and automatic slots.
// A game started at a waypoint (debug) is not chronicled.

// Chronicle lists the points kept, in no order.
func (s *Session) Chronicle() []storage.ChronicleEntry { return s.Store.ChronicleEntries() }

func (s *Session) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

// lordLevel is the level of the lord (유비) in a state, 0 when he is not in it.
func (s *Session) lordLevel(state core.State) int {
	for _, c := range state.Characters {
		if s.Data.Characters[c.Def].Lord {
			return c.Level
		}
	}
	return 0
}

// chronicleReached keeps the state of the waypoint the story just reached from the node it stood at, if it reached
// one, and says in r when it cannot.
func (s *Session) chronicleReached(from string, r *core.Result) {
	if s.Engine == nil || s.unchronicled || s.Store.Dir == "" {
		return
	}
	node := s.Engine.Reached(from)
	if node == "" {
		return
	}
	state := s.Engine.Snapshot()
	meta := storage.ChronicleMeta{Node: node, Route: s.Engine.Route(node), At: s.now(), Level: s.lordLevel(state)}
	if err := s.Store.ChroniclePut(meta, state); err != nil {
		log.Printf("chronicle: %v", err)
		if r != nil {
			r.Events = append(r.Events, core.Event{Kind: "chronicle-failed", Text: "지난 이야기 기록 실패: " + err.Error()})
		}
	}
}

// loadChronicle starts playing from a point, as load does from a slot.
func (s *Session) loadChronicle(node, route string, changed bool) error {
	snap, err := s.Store.ChronicleLoad(node, route)
	if err != nil {
		return err
	}
	restore := core.Restore
	if changed {
		restore = core.RestoreChanged
	}
	candidate, err := restore(s.Data, snap)
	if err != nil {
		return err
	}
	s.Engine = candidate
	s.unchronicled = false
	return nil
}
