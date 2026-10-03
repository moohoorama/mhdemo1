package core

import (
	"encoding/json"
	"fmt"
	"github.com/mlange-42/ark/ecs"
	"sort"
	"srpg/internal/content"
)

type Engine struct {
	data                *content.Data
	state               State
	world               *ecs.World
	officers            *ecs.Map1[Officer]
	units               *ecs.Map1[Unit]
	officerIDs, unitIDs map[string]ecs.Entity
	events              []Event
}

func copyOf[T any](v T) T { b, _ := json.Marshal(v); var c T; _ = json.Unmarshal(b, &c); return c }
func New(d *content.Data, seed uint64) *Engine {
	s := State{Format: 1, AI: AIPolicyVersion, Difficulty: "normal", Core: CoreVersion, Rules: RulesVersion, Content: d.Version, ContentHash: d.Hash, Seed: seed, RNG: seed, Phase: "scenario", Node: "oath", Inventory: copyOf(d.InitialInventory), Warehouse: map[string]int{}, Deployment: []string{"유비", "관우", "장비"}, Executed: map[string]bool{}}
	e := fromState(d, s)
	for _, id := range e.state.Deployment {
		e.addOfficer(id)
	}
	return e
}
func fromState(d *content.Data, s State) *Engine {
	e := &Engine{data: d, state: s, world: ecs.NewWorld(), officerIDs: map[string]ecs.Entity{}, unitIDs: map[string]ecs.Entity{}}
	e.officers = ecs.NewMap1[Officer](e.world)
	e.units = ecs.NewMap1[Unit](e.world)
	for i := range s.Officers {
		o := s.Officers[i]
		e.officerIDs[o.ID] = e.officers.NewEntity(&o)
	}
	for i := range s.Units {
		u := s.Units[i]
		e.unitIDs[u.ID] = e.units.NewEntity(&u)
	}
	e.state.Officers = nil
	e.state.Units = nil
	return e
}
func (e *Engine) Snapshot() State {
	s := copyOf(e.state)
	for _, id := range keys(e.officerIDs) {
		s.Officers = append(s.Officers, copyOf(*e.officer(id)))
	}
	for _, id := range keys(e.unitIDs) {
		s.Units = append(s.Units, copyOf(*e.unit(id)))
	}
	return s
}
func keys[V any](m map[string]V) []string {
	k := make([]string, 0, len(m))
	for id := range m {
		k = append(k, id)
	}
	sort.Strings(k)
	return k
}
func (e *Engine) officer(id string) *Officer {
	h, ok := e.officerIDs[id]
	if !ok {
		return nil
	}
	return e.officers.Get(h)
}
func (e *Engine) unit(id string) *Unit {
	h, ok := e.unitIDs[id]
	if !ok {
		return nil
	}
	return e.units.Get(h)
}
func (e *Engine) addOfficer(id string) {
	if e.officer(id) != nil {
		return
	}
	d := e.data.Officers[id]
	o := Officer{ID: id, Class: d.Class, Level: d.Level, Equipment: map[string]string{}, Learned: []string{}}
	for _, eq := range d.Equipment {
		o.Equipment[e.data.Equipment[eq].Slot] = eq
	}
	e.officerIDs[id] = e.officers.NewEntity(&o)
}
func (e *Engine) Observe() Observation {
	s := e.Snapshot()
	o := Observation{State: s, Dialogue: copyOf(e.data.Node(s.Node))}
	if s.Phase == "battle" || s.Phase == "result" {
		m := copyOf(e.data.Stages[s.Stage])
		o.Map = &m
	}
	for _, u := range s.Units {
		of := e.officer(u.Officer)
		o.UnitViews = append(o.UnitViews, UnitView{Unit: u, Name: e.data.Officers[u.Officer].Name, Class: of.Class, Level: of.Level, XP: of.XP, XPRequired: ExperienceRequired(of.Level), XPProgress: ExperienceProgress(of.Level, of.XP), Stats: e.stats(&u), Traits: e.traits(&u, false), Skills: e.skills(&u)})
	}
	return o
}
func fail(code string) error { return fmt.Errorf("%s", code) }
func (e *Engine) Apply(c Command) (r Result, err error) {
	// A private ECS world is the transaction. Only a fully successful command is published.
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("InternalError: %v", p)
		}
	}()
	t := fromState(e.data, e.Snapshot())
	if err = t.execute(c); err != nil {
		return Result{}, err
	}
	t.settle()
	t.state.Revision++
	r = Result{t.state.Revision, t.events}
	*e = *t
	return r, nil
}
func (e *Engine) Retry() error {
	if e.state.Phase != "result" || e.state.Result != "defeat" || e.state.Checkpoint == nil {
		return fail("WrongPhase")
	}
	rev := e.state.Revision + 1
	t, err := Restore(e.data, *e.state.Checkpoint)
	if err != nil {
		return err
	}
	t.state.Revision = rev
	*e = *t
	return nil
}
func (e *Engine) rand(n int) int {
	e.state.RNG += 0x9e3779b97f4a7c15
	z := e.state.RNG
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return int(z % uint64(n))
}
func (e *Engine) emit(k, a, t string, n int) {
	e.events = append(e.events, Event{Kind: k, Actor: a, Target: t, Amount: n})
}
func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
func (e *Engine) traits(u *Unit, learning bool) []string {
	o := e.officer(u.Officer)
	d := e.data.Officers[o.ID]
	set := map[string]bool{}
	add := func(s []string) {
		for _, id := range s {
			set[id] = true
		}
	}
	if learning {
		add(e.data.CommonLearn)
		add(d.Learn)
		add(e.data.Classes[o.Class].Learn)
	} else {
		add(d.Traits)
		add(o.Learned)
		if e.data.Classes[o.Class].Tier > 0 {
			set["부관"] = true
		}
		if deputy := e.officer(o.Deputy); deputy != nil && !deputy.Dead {
			add(e.data.Officers[deputy.ID].Traits)
			add(deputy.Learned)
		}
	}
	for _, eq := range o.Equipment {
		v := e.data.Equipment[eq]
		if learning {
			add(v.Learn)
		} else {
			add(v.Traits)
		}
	}
	return keys(set)
}
func (e *Engine) has(u *Unit, t string) bool { return contains(e.traits(u, false), t) }
func (e *Engine) skills(u *Unit) []string {
	set := map[string]bool{}
	for _, s := range e.data.Classes[e.officer(u.Officer).Class].Skills {
		set[s] = true
	}
	for _, eq := range e.officer(u.Officer).Equipment {
		for _, s := range e.data.Equipment[eq].Skills {
			set[s] = true
		}
	}
	for _, t := range e.traits(u, false) {
		for _, s := range e.data.Traits[t].Skills {
			set[s] = true
		}
	}
	return keys(set)
}
func (e *Engine) execute(c Command) error {
	switch c.Kind {
	case "next", "choose":
		if e.state.Phase != "scenario" {
			return fail("WrongPhase")
		}
		n := e.data.Node(e.state.Node)
		if c.Kind == "next" && n.Kind == "dialogue" {
			e.state.Node = n.Next
		} else if c.Kind == "choose" && n.Kind == "choice" {
			v, ok := n.Choices[c.Option]
			if !ok {
				return fail("InvalidOption")
			}
			e.state.Node = v
		} else {
			return fail("WrongPhase")
		}
		e.automatic()
		return nil
	case "continue":
		if e.state.Phase != "result" || e.state.Result != "victory" {
			return fail("WrongPhase")
		}
		if e.state.Stage == 2 {
			e.state.Phase = "complete"
			e.state.Node = ""
			e.clearBattle()
			return nil
		}
		e.state.Stage++
		e.state.Phase = "scenario"
		e.state.Node = []string{"", "join", "sword"}[e.state.Stage]
		e.clearBattle()
		active := []string{}
		for _, id := range e.state.Deployment {
			if !e.officer(id).Dead {
				active = append(active, id)
			}
		}
		e.state.Deployment = active
		e.automatic()
		return nil
	case "equip", "unequip", "deploy", "start", "deputy", "undeputy":
		return e.prepare(c)
	case "end":
		if e.state.Phase != "battle" {
			return fail("WrongPhase")
		}
		e.endFaction()
		return nil
	}
	if e.state.Phase != "battle" {
		return fail("WrongPhase")
	}
	u := e.unit(c.Actor)
	if u == nil || u.HP <= 0 {
		return fail("InvalidActor")
	}
	if u.Faction != e.state.Turn {
		return fail("NotYourTurn")
	}
	if u.Done {
		return fail("ActionSpent")
	}
	switch c.Kind {
	case "wait":
		u.Moved = true
		u.Acted = true
		u.Done = true
		return nil
	case "move":
		if u.Moved {
			return fail("MoveSpent")
		}
		if u.Status["confusion"].Turns > 0 || u.Status["root"].Turns > 0 {
			return fail("Immobilized")
		}
		p := content.Point{X: c.X, Y: c.Y}
		if !containsPoint(e.moves(u), p) {
			return fail("OutOfRange")
		}
		u.X = c.X
		u.Y = c.Y
		u.Moved = true
		e.emit("move", u.ID, "", 0)
		return nil
	case "learn":
		if u.Acted {
			return fail("ActionSpent")
		}
		o := e.officer(u.Officer)
		t, ok := e.data.Traits[c.Trait]
		if !ok || !contains(e.traits(u, true), c.Trait) || contains(o.Learned, c.Trait) {
			return fail("InvalidTrait")
		}
		if o.Points < t.Cost {
			return fail("InsufficientPoints")
		}
		o.Points -= t.Cost
		o.Learned = append(o.Learned, c.Trait)
		e.emit("learn", u.ID, c.Trait, 0)
		return nil
	case "duel":
		return e.duel(u, c)
	case "item":
		return e.useItem(u, c)
	case "attack", "skill":
		return e.useSkill(u, c)
	}
	return fail("UnknownCommand")
}
func (e *Engine) automatic() {
	for {
		n := e.data.Node(e.state.Node)
		if n.Kind != "join" && n.Kind != "grant" {
			return
		}
		if !e.state.Executed[n.Event] {
			if n.Kind == "join" {
				e.addOfficer(n.Officer)
				e.state.Deployment = append(e.state.Deployment, n.Officer)
			} else {
				e.state.Warehouse[n.Item] += n.Count
			}
			e.state.Executed[n.Event] = true
			e.emit(n.Kind, "", n.Officer+n.Item, n.Count)
		}
		e.state.Node = n.Next
	}
}
func (e *Engine) prepare(c Command) error {
	if e.state.Phase != "scenario" || e.data.Node(e.state.Node).Kind != "preparation" {
		return fail("WrongPhase")
	}
	if c.Kind == "deputy" || c.Kind == "undeputy" {
		return e.assignDeputy(c)
	}
	if c.Kind == "deploy" {
		s := e.data.Stages[e.state.Stage]
		if len(c.Deployment) < 1 || len(c.Deployment) > s.Limit || !contains(c.Deployment, "유비") {
			return fail("InvalidDeployment")
		}
		seen := map[string]bool{}
		for _, id := range c.Deployment {
			if e.officer(id) == nil || e.officer(id).Dead || e.isDeputy(id) || seen[id] || !contains([]string{"유비", "관우", "장비", "간옹"}, id) {
				return fail("InvalidDeployment")
			}
			seen[id] = true
		}
		e.state.Deployment = append([]string{}, c.Deployment...)
		return nil
	}
	if c.Kind == "equip" || c.Kind == "unequip" {
		o := e.officer(c.Actor)
		if o == nil || !contains([]string{"유비", "관우", "장비", "간옹"}, c.Actor) {
			return fail("InvalidActor")
		}
		if o.Dead || e.isDeputy(o.ID) {
			return fail("InvalidActor")
		}
		eq, ok := e.data.Equipment[c.Item]
		if !ok {
			return fail("InvalidEquipment")
		}
		old := o.Equipment[eq.Slot]
		if c.Kind == "unequip" {
			if old != c.Item {
				return fail("InvalidEquipment")
			}
			delete(o.Equipment, eq.Slot)
			e.state.Warehouse[old]++
			return nil
		}
		cl := e.data.Classes[o.Class]
		if eq.Slot == "weapon" && eq.Type != cl.Weapon || eq.Slot == "armor" && eq.Type != cl.Armor {
			return fail("IncompatibleEquipment")
		}
		if e.state.Warehouse[c.Item] < 1 {
			return fail("MissingEquipment")
		}
		e.state.Warehouse[c.Item]--
		if old != "" {
			e.state.Warehouse[old]++
		}
		o.Equipment[eq.Slot] = c.Item
		return nil
	}
	if c.Kind != "start" {
		return fail("UnknownCommand")
	}
	// Remove prior transient units and enemies from the preparation checkpoint.
	s := e.Snapshot()
	s.Units = nil
	s.Officers = nil
	s.Checkpoint = nil
	for _, id := range []string{"유비", "관우", "장비", "간옹"} {
		if o := e.officer(id); o != nil {
			s.Officers = append(s.Officers, copyOf(*o))
		}
	}
	t := fromState(e.data, s)
	*e = *t
	cp := e.Snapshot()
	e.state.Checkpoint = &cp
	e.state.Phase = "battle"
	e.state.Turn = "ally"
	e.state.Round = 1
	e.state.Result = ""
	stage := e.data.Stages[e.state.Stage]
	for i, id := range e.state.Deployment {
		p := stage.Allies[i]
		e.spawn(id, "ally", p.X, p.Y)
	}
	for _, sp := range stage.Enemies {
		e.addOfficer(sp.Officer)
		e.spawn(sp.Officer, "enemy", sp.X, sp.Y)
	}
	return nil
}
func (e *Engine) spawn(id, f string, x, y int) {
	u := Unit{ID: id, Officer: id, Faction: f, X: x, Y: y, Status: map[string]Status{}, Items: map[string]int{}}
	if f == "enemy" {
		u.Items["소군량"] = 1
	}
	st := e.stats(&u)
	u.HP = st.MaxHP
	u.MP = st.MaxMP / 2
	e.unitIDs[id] = e.units.NewEntity(&u)
}
func (e *Engine) settle() {
	if e.state.Phase != "battle" {
		return
	}
	loss := e.unit("유비") == nil || e.unit("유비").HP <= 0
	stage := e.data.Stages[e.state.Stage]
	boss := e.unit(stage.Boss)
	win := boss.HP*100 <= e.stats(boss).MaxHP*stage.Threshold
	if !loss && !win {
		return
	}
	e.state.Phase = "result"
	e.state.Result = "defeat"
	if loss {
		return
	}
	e.state.Result = "victory"
	key := "reward-" + stage.ID
	if !e.state.Executed[key] {
		for item, n := range stage.Reward {
			e.state.Inventory[item] += n
		}
		e.state.Executed[key] = true
	}
	e.emit("victory", "", stage.ID, 0)
}

// Campaign transitions retire transient units and the previous retry checkpoint.
func (e *Engine) clearBattle() {
	s := e.Snapshot()
	s.Units = nil
	s.Checkpoint = nil
	s.Round = 0
	s.Turn = ""
	s.Result = ""
	officers := []Officer{}
	for _, o := range s.Officers {
		if contains([]string{"유비", "관우", "장비", "간옹"}, o.ID) {
			officers = append(officers, o)
		}
	}
	s.Officers = officers
	*e = *fromState(e.data, s)
}
