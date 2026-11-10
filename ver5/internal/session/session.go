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
	view     core.ObservationCache
}
type Request struct {
	ID          any          `json:"id,omitempty"`
	Op          string       `json:"op"`
	Seed        uint64       `json:"seed"`
	Revision    *uint64      `json:"revision,omitempty"`
	Command     core.Command `json:"command"`
	Actor, Slot string
	Node        string            `json:"node,omitempty"`       // the waypoint a jump starts at
	Difficulty  string            `json:"difficulty,omitempty"` // a new game's difficulty; none is normal
	Overwrite   bool              `json:"overwrite"`
	Discard     bool              `json:"discard_current"`
	Changed     bool              `json:"load_changed"` // load a save made before the data changed
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

// View is what the engine shows now, built once per change of its state; callers only read it.
// With no game it is the zero Observation, whose empty Phase matches no screen.
func (s *Session) View() core.Observation {
	if s.Engine == nil {
		return core.Observation{}
	}
	return s.view.Of(s.Engine)
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
		s.Engine.SetDamagePct(s.Settings.DamageScale())
		s.Engine.SetXPPct(s.Settings.XPScale())
		o := s.View()
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
		if q.Revision == nil || *q.Revision != s.View().Revision {
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
		e, err := core.NewGame(s.Data, q.Seed, q.Difficulty)
		if err != nil {
			return nil, err
		}
		s.Engine = e
		s.Screen = "playing"
		s.Dirty = true
		return nil, nil
	case "jump":
		if err := discard(); err != nil {
			return nil, err
		}
		e, err := core.NewAt(s.Data, q.Seed, q.Node, q.Difficulty)
		if err != nil {
			return nil, err
		}
		s.Engine = e
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
		if !storage.IsManualSlot(q.Slot) {
			return nil, fmt.Errorf("InvalidSlot")
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
		restore := core.Restore
		if q.Changed {
			restore = core.RestoreChanged
		}
		candidate, err := restore(s.Data, snap)
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
			o := s.View()
			if o.Phase == "battle" && o.Turn != "ally" && q.Command.Kind != "learn" && q.Command.Kind != "learn_close" {
				return nil, fmt.Errorf("NotYourTurn")
			}
			v, err = s.apply(q.Command)
		} else if q.Op == "ai" {
			if s.View().Turn == "ally" && !s.Engine.Deploying() {
				return nil, fmt.Errorf("NotYourTurn")
			}
			if len(s.View().Learning) > 0 {
				return nil, fmt.Errorf("LearningPending")
			}
			v, err = s.stepAI()
		} else {
			turn := s.View().Turn
			phase := s.View().Phase
			deadline := time.Now().Add(10 * time.Second)
			for i := 0; i < 1000; i++ {
				if time.Now().After(deadline) {
					return v.Events, fmt.Errorf("AITimeLimit")
				}
				o := s.View()
				if (o.Phase != "battle" && o.Phase != "deployment") || o.Turn != turn || o.Phase != phase {
					break
				}
				if turn != "ally" && len(o.Learning) > 0 {
					break
				}
				step, x := s.stepAI()
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

// AdvanceEnemies plays every AI army's turn up to the allies' and saves at allied turn starts.
func (s *Session) AdvanceEnemies() ([]core.Event, error) {
	out := []core.Event{}
	if s.Engine == nil {
		return out, nil
	}
	deadline := time.Now().Add(10 * time.Second)
	for i := 0; i < 1000; i++ {
		if time.Now().After(deadline) {
			return out, fmt.Errorf("AITimeLimit")
		}
		o := s.View()
		if o.Phase != "battle" || o.Turn == "ally" || len(o.Learning) > 0 {
			return out, nil
		}
		v, err := s.stepAI()
		if err != nil {
			return out, err
		}
		out = append(out, v.Events...)
		s.Dirty = true
	}
	return out, fmt.Errorf("AILimit")
}

// ResolvePhase runs the current AI army's whole turn through the core, one command at
// a time exactly as AdvanceEnemies does, and returns the actions for claim replay.
// It pauses for player learning; the replay only shows the returned state.
// Commands use the same turn-start autosave path as manual play.
func (s *Session) ResolvePhase() ([]*replay.Action, error) {
	if s.Engine == nil {
		return nil, fmt.Errorf("NoGame")
	}
	turn, army := s.View().Turn, s.View().Army
	actions := []*replay.Action{}
	deadline := time.Now().Add(10 * time.Second)
	for i := 0; ; i++ {
		if i == 1000 {
			return actions, fmt.Errorf("AILimit")
		}
		if time.Now().After(deadline) {
			return actions, fmt.Errorf("AITimeLimit")
		}
		o := s.View()
		if o.Phase != "battle" || o.Turn != turn || o.Army != army || len(o.Learning) > 0 {
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
		r, err := s.apply(c)
		if err != nil {
			return actions, err
		}
		a.Add(c, r.Events)
		s.Dirty = true
	}
	replay.Link(actions)
	return actions, nil
}

func (s *Session) stepAI() (core.Result, error) {
	c, err := ai.Next(s.Engine)
	if err != nil {
		return core.Result{}, err
	}
	return s.apply(c)
}

// apply runs a command and autosaves: the walk as it stood when the confidant asked, once the party
// sorties, so loading it walks again before saying yes; and every allied turn start of a battle.
func (s *Session) apply(c core.Command) (core.Result, error) {
	var walk *core.State
	if c.Kind == "sortie" {
		state := s.Engine.Snapshot()
		walk = &state
	}
	r, err := s.Engine.Apply(c)
	if err != nil {
		return r, err
	}
	if walk != nil {
		s.autosave(*walk, &r)
	}
	for _, event := range r.Events {
		if event.Kind == "deployment_finish" || event.Kind == "turn" && event.Target == "ally" {
			if state := s.Engine.Snapshot(); state.Phase == "battle" {
				s.autosave(state, &r)
			}
			break
		}
	}
	return r, nil
}

// autosave writes the state to the next automatic slot; a failure is told as an event, not an error.
func (s *Session) autosave(state core.State, r *core.Result) {
	if err := s.Store.Autosave(state); err != nil {
		r.Events = append(r.Events, core.Event{Kind: "autosave-failed", Text: "자동 저장 실패: " + err.Error()})
	}
}
