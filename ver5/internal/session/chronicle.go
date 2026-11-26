package session

import (
	"fmt"
	"log"
	"slices"
	"srpg/internal/core"
	"srpg/internal/storage"
	"time"
)

// The chronicle (지난 이야기, docs/system-chronicle.md): each time the story really reaches a chapter's start or a
// battle, the state the player had there is kept as a point of the line the game plays. Points are written
// once and kept apart from the manual and automatic slots. A game started at a waypoint (debug) is not
// chronicled, and a save from before the chronicle has no line, so its next point starts a line with nothing
// before it.

const noChronicle = "-" // Run of a game whose points are not kept

// ChronicleLine is a line as the chronicle window lists it.
type ChronicleLine struct {
	storage.ChronicleLine
	Number  int // 1.. in the order the lines were started
	Current bool
	From    *ChronicleFork // where the line left another; nil for a line that started the story
	Entries []storage.ChronicleEntry
	Err     string // set when the list could not be read at all
}

// ChronicleFork names the line a line was started from and the point it left it at.
type ChronicleFork struct {
	Number int
	Entry  storage.ChronicleEntry
}

// Chronicle lists the lines with their points, oldest line first.
func (s *Session) Chronicle() []ChronicleLine {
	lines := s.Store.ChronicleLines()
	number := map[string]int{}
	for i, l := range lines {
		number[l.ID] = i + 1
	}
	out := make([]ChronicleLine, 0, len(lines))
	for _, l := range lines {
		v := ChronicleLine{ChronicleLine: l, Number: number[l.ID], Current: l.ID == s.run}
		es, err := s.Store.ChronicleEntries(l.ID)
		if err != nil {
			v.Err = err.Error()
		}
		v.Entries = es
		if l.Parent != "" {
			if i := slices.IndexFunc(es, func(e storage.ChronicleEntry) bool { return e.Seq == l.ParentSeq }); i >= 0 {
				n := number[es[i].Line]
				if n == 0 { // the parent has no point of its own listed: the line it took over from does
					n = number[l.Parent]
				}
				v.From = &ChronicleFork{Number: n, Entry: es[i]}
			}
		}
		out = append(out, v)
	}
	return out
}

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

// stamp puts the chronicle line the game plays on a state about to be saved.
func (s *Session) stamp(state core.State) core.State {
	state.Run, state.Steps = s.run, s.steps
	return state
}

// adopt takes the line a loaded state belongs to; a state from before the chronicle has none.
func (s *Session) adopt(state core.State) {
	s.run, s.steps = state.Run, state.Steps
	s.Engine.ClearLine()
}

// chronicleReached keeps the state of the waypoint the story just reached from the node it stood at, if it reached
// one, and says in r when it cannot.
func (s *Session) chronicleReached(from string, r *core.Result) {
	if s.Engine == nil || s.run == noChronicle || s.Store.Dir == "" {
		return
	}
	node := s.Engine.Reached(from)
	if node == "" {
		return
	}
	if err := s.record(node); err != nil {
		log.Printf("chronicle: %v", err)
		if r != nil {
			r.Events = append(r.Events, core.Event{Kind: "chronicle-failed", Text: "지난 이야기 기록 실패: " + err.Error()})
		}
	}
}

// record keeps the engine's state as the point of node. The line goes on when the state is the latest of its
// line; otherwise the story is being played again from an earlier point, and a new line is started from it, so
// the points the old line has are never replaced. A node the line already reached is not kept again.
func (s *Session) record(node string) error {
	run, steps := s.run, s.steps
	var es []storage.ChronicleEntry
	if run != "" {
		var err error
		if es, err = s.Store.ChronicleEntries(run); err != nil || (len(es) > 0 && es[len(es)-1].Seq < steps) || len(es) == 0 && steps > 0 {
			run, steps, es = "", 0, nil // its history is gone: start over, inventing nothing
		}
	}
	if i := slices.IndexFunc(es, func(e storage.ChronicleEntry) bool { return e.Seq <= steps && e.Node == node }); i >= 0 {
		return nil
	}
	at := s.now()
	switch {
	case run == "":
		id, err := s.Store.ChronicleBegin("", 0, at)
		if err != nil {
			return err
		}
		run = id
	case len(es) > 0 && es[len(es)-1].Seq > steps:
		id, err := s.Store.ChronicleBegin(run, steps, at)
		if err != nil {
			return err
		}
		run = id
	}
	state := s.Engine.Snapshot()
	state.Run, state.Steps = run, steps+1
	meta := storage.ChronicleMeta{Line: run, Seq: steps + 1, Node: node, At: at, Level: s.lordLevel(state)}
	if err := s.Store.ChronicleAdd(meta, state); err != nil {
		return err
	}
	s.run, s.steps = run, steps+1
	return nil
}

// loadChronicle starts playing from a point of a line's list, as load does from a slot; the line the game
// plays is the point's own, so going on from it keeps what the line already has.
func (s *Session) loadChronicle(line string, seq int, changed bool) error {
	snap, err := s.Store.ChronicleLoad(line, seq)
	if err != nil {
		return err
	}
	if snap.Run == "" || snap.Steps != seq {
		return fmt.Errorf("CorruptSave: chronicle point without a line")
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
	s.adopt(snap)
	return nil
}
