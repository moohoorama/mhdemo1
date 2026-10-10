package session

import (
	"fmt"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
	"srpg/internal/storage"
	"time"
)

type Session struct {
	Data     *content.Data
	Engine   *core.Engine
	Store    storage.Store
	Settings storage.Settings
	Dirty    bool
	Screen   string
}
type Request struct {
	ID          any          `json:"id,omitempty"`
	Op          string       `json:"op"`
	Seed        uint64       `json:"seed"`
	Revision    *uint64      `json:"revision,omitempty"`
	Command     core.Command `json:"command"`
	Actor, Slot string
	Overwrite   bool              `json:"overwrite"`
	Discard     bool              `json:"discard_current"`
	Settings    *storage.Settings `json:"settings,omitempty"`
}
type Response struct {
	ID          any               `json:"id,omitempty"`
	OK          bool              `json:"ok"`
	Error       string            `json:"error,omitempty"`
	Screen      string            `json:"screen"`
	Events      []core.Event      `json:"events,omitempty"`
	Observation *core.Observation `json:"observation,omitempty"`
	Options     *core.Options     `json:"options,omitempty"`
	Preview     *core.Preview     `json:"preview,omitempty"`
	Slots       []string          `json:"slots,omitempty"`
	Settings    storage.Settings  `json:"settings"`
}

func New(d *content.Data, dir string) (*Session, error) {
	s := &Session{Data: d, Store: storage.Store{Dir: dir}, Screen: "title"}
	v, err := s.Store.LoadSettings()
	s.Settings = v
	return s, err
}
func (s *Session) Handle(q Request) Response {
	r := Response{ID: q.ID, OK: true}
	events, err := s.handle(q, &r)
	if err != nil {
		r.OK = false
		r.Error = err.Error()
	}
	r.Events = events
	r.Screen = s.Screen
	r.Settings = s.Settings
	if s.Engine != nil {
		o := s.Engine.Observe()
		r.Observation = &o
	}
	return r
}
func (s *Session) handle(q Request, r *Response) ([]core.Event, error) {
	need := func() error {
		if s.Engine == nil {
			return fmt.Errorf("NoGame")
		}
		return nil
	}
	rev := func() error {
		if err := need(); err != nil {
			return err
		}
		if q.Revision == nil || *q.Revision != s.Engine.Observe().Revision {
			return fmt.Errorf("StaleRevision")
		}
		return nil
	}
	discard := func() error {
		if s.Dirty && !q.Discard {
			return fmt.Errorf("DiscardRequired")
		}
		return nil
	}
	switch q.Op {
	case "new":
		if err := discard(); err != nil {
			return nil, err
		}
		s.Engine = core.New(s.Data, q.Seed)
		s.Screen = "playing"
		s.Dirty = true
		return nil, nil
	case "observe":
		return nil, nil
	case "slots":
		r.Slots = s.Store.Slots()
		return nil, nil
	case "settings":
		if q.Settings != nil {
			if err := s.Store.SaveSettings(*q.Settings); err != nil {
				return nil, err
			}
			s.Settings = *q.Settings
		}
		return nil, nil
	case "menu":
		s.Screen = "menu"
		return nil, nil
	case "resume":
		if err := need(); err != nil {
			return nil, err
		}
		s.Screen = "playing"
		return nil, nil
	case "title", "quit":
		if err := discard(); err != nil {
			return nil, err
		}
		s.Screen = q.Op
		if q.Op == "title" {
			s.Engine = nil
			s.Dirty = false
		}
		return nil, nil
	case "save":
		if err := need(); err != nil {
			return nil, err
		}
		if err := s.Store.Save(q.Slot, s.Engine.Snapshot(), q.Overwrite); err != nil {
			return nil, err
		}
		s.Dirty = false
		return nil, nil
	case "load":
		if err := discard(); err != nil {
			return nil, err
		}
		snap, err := s.Store.Load(q.Slot)
		if err != nil {
			return nil, err
		}
		candidate, err := core.Restore(s.Data, snap)
		if err != nil {
			return nil, err
		}
		s.Engine = candidate
		s.Dirty = false
		s.Screen = "playing"
		return nil, nil
	case "legal":
		if err := need(); err != nil {
			return nil, err
		}
		v := s.Engine.LegalActions(q.Actor)
		r.Options = &v
		return nil, nil
	case "preview":
		if err := need(); err != nil {
			return nil, err
		}
		v, err := s.Engine.Preview(q.Command)
		if err != nil {
			return nil, err
		}
		r.Preview = &v
		return nil, nil
	case "retry":
		if err := rev(); err != nil {
			return nil, err
		}
		if err := s.Engine.Retry(); err != nil {
			return nil, err
		}
		s.Dirty = true
		return nil, nil
	case "command", "ai", "auto":
		if err := rev(); err != nil {
			return nil, err
		}
		var v core.Result
		var err error
		if q.Op == "command" {
			o := s.Engine.Observe()
			if o.Phase == "battle" && o.Turn == "enemy" {
				return nil, fmt.Errorf("NotYourTurn")
			}
			v, err = s.Engine.Apply(q.Command)
		} else if q.Op == "ai" {
			if s.Engine.Observe().Turn != "enemy" {
				return nil, fmt.Errorf("NotYourTurn")
			}
			v, err = ai.Step(s.Engine)
		} else {
			turn := s.Engine.Observe().Turn
			deadline := time.Now().Add(10 * time.Second)
			for i := 0; i < 1000; i++ {
				if time.Now().After(deadline) {
					return v.Events, fmt.Errorf("AITimeLimit")
				}
				o := s.Engine.Observe()
				if o.Phase != "battle" || o.Turn != turn {
					break
				}
				step, x := ai.Step(s.Engine)
				if x != nil {
					return v.Events, x
				}
				s.Dirty = true
				v.Revision = step.Revision
				v.Events = append(v.Events, step.Events...)
				if i == 999 {
					err = fmt.Errorf("AILimit")
				}
			}
		}
		if len(v.Events) > 0 || err == nil {
			s.Dirty = true
		}
		return v.Events, err
	}
	return nil, fmt.Errorf("UnknownOperation")
}

// AdvanceEnemies honors the one-command setting and saves at every command boundary.
func (s *Session) AdvanceEnemies() ([]core.Event, error) {
	out := []core.Event{}
	if s.Settings.AI != "auto" || s.Engine == nil {
		return out, nil
	}
	deadline := time.Now().Add(10 * time.Second)
	for i := 0; i < 1000; i++ {
		if time.Now().After(deadline) {
			return out, fmt.Errorf("AITimeLimit")
		}
		o := s.Engine.Observe()
		if o.Phase != "battle" || o.Turn != "enemy" {
			return out, nil
		}
		v, err := ai.Step(s.Engine)
		if err != nil {
			return out, err
		}
		out = append(out, v.Events...)
		s.Dirty = true
		if err = s.Store.Save("auto", s.Engine.Snapshot(), true); err != nil {
			return out, err
		}
	}
	return out, fmt.Errorf("AILimit")
}

// ResolvePhase runs the current AI faction's whole phase through the core, one command at
// a time exactly as AdvanceEnemies does, and returns the actions for claim replay. The
// returned state is final; the replay only shows it. It autosaves once at the end.
func (s *Session) ResolvePhase() ([]*replay.Action, error) {
	if s.Engine == nil {
		return nil, fmt.Errorf("NoGame")
	}
	turn := s.Engine.Observe().Turn
	actions := []*replay.Action{}
	deadline := time.Now().Add(10 * time.Second)
	for i := 0; ; i++ {
		if i == 1000 {
			return actions, fmt.Errorf("AILimit")
		}
		if time.Now().After(deadline) {
			return actions, fmt.Errorf("AITimeLimit")
		}
		o := s.Engine.Observe()
		if o.Phase != "battle" || o.Turn != turn {
			break
		}
		c, err := ai.Next(s.Engine)
		if err != nil {
			return actions, err
		}
		var a *replay.Action
		if n := len(actions); n > 0 && actions[n-1].Actor == c.Actor && c.Actor != "" && !actions[n-1].Barrier {
			a = actions[n-1]
		} else {
			a = replay.New(o, c.Actor)
			actions = append(actions, a)
		}
		r, err := s.Engine.Apply(c)
		if err != nil {
			return actions, err
		}
		a.Add(c, r.Events)
		s.Dirty = true
	}
	replay.Link(actions)
	return actions, s.Store.Save("auto", s.Engine.Snapshot(), true)
}
