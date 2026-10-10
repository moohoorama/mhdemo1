package core

import (
	"encoding/json"
	"fmt"
	"github.com/mlange-42/ark/ecs"
	"sort"
	"srpg/internal/content"
	"strconv"
	"strings"
)

type Engine struct {
	data             *content.Data
	state            State
	world            *ecs.World
	chars            *ecs.Map1[Character]
	units            *ecs.Map1[Unit]
	charIDs, unitIDs map[string]ecs.Entity
	events           []Event
}

func copyOf[T any](v T) T { b, _ := json.Marshal(v); var c T; _ = json.Unmarshal(b, &c); return c }
func New(d *content.Data, seed uint64) *Engine {
	c := d.Campaign
	s := State{Format: 2, AI: AIPolicyVersion, Difficulty: "normal", Core: CoreVersion, Rules: RulesVersion, Content: d.Version, ContentHash: d.Hash,
		Seed: seed, RNG: seed, Phase: "scenario", Node: c.Start, Inventory: copyOf(c.Inventory), Warehouse: map[string]int{},
		Deployment: append([]string{}, c.Deployment...), Executed: map[string]bool{}}
	if s.Inventory == nil {
		s.Inventory = map[string]int{}
	}
	e := fromState(d, s)
	for _, id := range c.Roster {
		e.addCharacter(id, id)
	}
	e.enter()
	return e
}
func fromState(d *content.Data, s State) *Engine {
	e := &Engine{data: d, state: s, world: ecs.NewWorld(), charIDs: map[string]ecs.Entity{}, unitIDs: map[string]ecs.Entity{}}
	e.chars = ecs.NewMap1[Character](e.world)
	e.units = ecs.NewMap1[Unit](e.world)
	for i := range s.Characters {
		c := s.Characters[i]
		e.charIDs[c.ID] = e.chars.NewEntity(&c)
	}
	for i := range s.Units {
		u := s.Units[i]
		e.unitIDs[u.ID] = e.units.NewEntity(&u)
	}
	e.state.Characters = nil
	e.state.Units = nil
	return e
}
func (e *Engine) Snapshot() State {
	s := copyOf(e.state)
	for _, id := range keys(e.charIDs) {
		s.Characters = append(s.Characters, copyOf(*e.char(id)))
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
func (e *Engine) char(id string) *Character {
	h, ok := e.charIDs[id]
	if !ok {
		return nil
	}
	return e.chars.Get(h)
}
func (e *Engine) unit(id string) *Unit {
	h, ok := e.unitIDs[id]
	if !ok {
		return nil
	}
	return e.units.Get(h)
}

// def is the data definition behind a character.
func (e *Engine) def(c *Character) content.Character { return e.data.Characters[c.Def] }
func (e *Engine) class(c *Character) content.Class   { return e.data.Classes[c.Class] }
func (e *Engine) unitClass(u *Unit) content.Class    { return e.class(e.char(u.Character)) }
func (e *Engine) unitDef(u *Unit) content.Character  { return e.def(e.char(u.Character)) }
func (e *Engine) name(c *Character) string {
	n := e.def(c).Name
	if i := strings.IndexByte(c.ID, '#'); i >= 0 {
		n += c.ID[i+1:]
	}
	return n
}

// addCharacter records a character made from definition def under id.
func (e *Engine) addCharacter(id, def string) *Character {
	if c := e.char(id); c != nil {
		return c
	}
	d := e.data.Characters[def]
	c := Character{ID: id, Def: def, Class: d.Class, Level: d.Level, Temp: d.Template, Equipment: map[string]string{}, Learned: []string{}}
	for _, eq := range d.Equipment {
		c.Equipment[e.data.Equipment[eq].Slot] = eq
	}
	e.charIDs[id] = e.chars.NewEntity(&c)
	return e.char(id)
}
func (e *Engine) Observe() Observation {
	s := e.Snapshot()
	o := Observation{State: s, Dialogue: copyOf(e.data.Node(s.Node))}
	if s.Phase == "battle" || s.Phase == "result" {
		m := copyOf(e.data.Stages[s.Stage])
		o.Map = &m
	}
	for _, u := range s.Units {
		c := e.char(u.Character)
		o.UnitViews = append(o.UnitViews, UnitView{Unit: u, Name: e.name(c), Class: c.Class, Def: c.Def, Level: c.Level, XP: c.XP, XPRequired: ExperienceRequired(e.data, c.Level),
			XPProgress: ExperienceProgress(e.data, c.Level, c.XP), Stats: e.stats(&u), Attributes: attributes(e.attributes(c)), Points: c.Points,
			Traits: e.traits(c), Skills: e.skills(c)})
	}
	for _, id := range s.Learning {
		c := e.char(id)
		o.Learners = append(o.Learners, Learner{ID: id, Name: e.name(c), Points: c.Points, Options: e.learnable(c)})
	}
	return o
}

// Data is the content bundle the engine runs.
func (e *Engine) Data() *content.Data { return e.data }
func fail(code string) error          { return fmt.Errorf("%s", code) }
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

// traits is every trait the character has: innate, learned, given by class and equipment, and the deputy's own.
func (e *Engine) traits(c *Character) []string {
	set := map[string]bool{}
	add := func(s []string) {
		for _, id := range s {
			set[id] = true
		}
	}
	add(e.def(c).Traits)
	add(c.Learned)
	add(e.class(c).Grants)
	for _, eq := range c.Equipment {
		add(e.data.Equipment[eq].Traits)
	}
	if d := e.char(c.Deputy); d != nil && !d.Dead {
		add(e.def(d).Traits)
		add(d.Learned)
	}
	return keys(set)
}

// fx totals one trait effect over the unit's traits.
func (e *Engine) fx(u *Unit, key string) float64 { return e.fxOf(e.char(u.Character), key) }
func (e *Engine) fxOf(c *Character, key string) float64 {
	n := 0.0
	for _, t := range e.traits(c) {
		n += e.data.Traits[t].Effects[key]
	}
	return n
}
func (e *Engine) has(u *Unit, key string) bool         { return e.fx(u, key) != 0 }
func (e *Engine) hasTrait(c *Character, t string) bool { return contains(e.traits(c), t) }

// learnable lists the traits the character may learn now: in the learn pool, prerequisites met, not yet owned.
func (e *Engine) learnable(c *Character) []string {
	owned := e.traits(c)
	pool := map[string]bool{}
	for _, t := range e.def(c).Learn {
		pool[t] = true
	}
	for _, t := range e.class(c).AllPool() {
		pool[t] = true
	}
	for _, eq := range c.Equipment {
		for _, t := range e.data.Equipment[eq].Learn {
			pool[t] = true
		}
	}
	out := []string{}
	for _, t := range keys(pool) {
		if contains(owned, t) {
			continue
		}
		ok := true
		for _, q := range e.data.Traits[t].Requires {
			ok = ok && contains(owned, q)
		}
		if ok {
			out = append(out, t)
		}
	}
	return out
}
func attributes(a [6]float64) (out [6]int) {
	for i, v := range a {
		out[i] = int(v)
	}
	return out
}

// skills is every skill the character may use: given by class and equipment, or opened by its traits.
func (e *Engine) skills(c *Character) []string {
	set := map[string]bool{}
	for _, s := range e.class(c).Skills {
		set[s] = true
	}
	for _, eq := range c.Equipment {
		for _, s := range e.data.Equipment[eq].Skills {
			set[s] = true
		}
	}
	owned := e.traits(c)
	for id, s := range e.data.Skills {
		if len(s.Requires) == 0 {
			continue
		}
		ok := true
		for _, t := range s.Requires {
			ok = ok && contains(owned, t)
		}
		if ok {
			set[id] = true
		}
	}
	return keys(set)
}

// enter runs what a scenario node does on arrival: a preparation picks its stage, join and grant act once.
func (e *Engine) enter() {
	for {
		n := e.data.Node(e.state.Node)
		switch n.Kind {
		case "preparation":
			e.state.Stage = e.data.StageIndex(n.Stage)
			return
		case "join", "grant":
			if !e.state.Executed[n.Event] {
				if n.Kind == "join" {
					e.addCharacter(n.Character, n.Character)
					e.state.Deployment = append(e.state.Deployment, n.Character)
				} else if _, ok := e.data.Equipment[n.Item]; ok {
					e.state.Warehouse[n.Item] += n.Count
				} else {
					e.state.Inventory[n.Item] += n.Count
				}
				e.state.Executed[n.Event] = true
				e.emit(n.Kind, "", n.Character+n.Item, n.Count)
			}
			e.state.Node = n.Next
		default:
			return
		}
	}
}
func (e *Engine) execute(c Command) error {
	if len(e.state.Learning) > 0 && c.Kind != "learn" && c.Kind != "learn_close" {
		return fail("LearningPending")
	}
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
		e.enter()
		return nil
	case "continue":
		if e.state.Phase != "result" || e.state.Result != "victory" {
			return fail("WrongPhase")
		}
		next := e.data.Stages[e.state.Stage].Next
		e.clearBattle()
		if next == "" {
			e.state.Phase = "complete"
			e.state.Node = ""
			return nil
		}
		e.state.Phase = "scenario"
		e.state.Node = next
		active := []string{}
		for _, id := range e.state.Deployment {
			if !e.char(id).Dead {
				active = append(active, id)
			}
		}
		e.state.Deployment = active
		e.enter()
		return nil
	case "equip", "unequip", "deploy", "start", "deputy", "undeputy":
		return e.prepare(c)
	case "learn", "learn_close":
		return e.learn(c)
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
	case "place":
		return e.place(u, c)
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
		path := e.route(u, p)
		u.X = c.X
		u.Y = c.Y
		u.Moved = true
		e.events = append(e.events, Event{Kind: "move", Actor: u.ID, Path: path})
		return nil
	case "item", "attack", "skill":
		var err error
		if c.Kind == "item" {
			err = e.useItem(u, c)
		} else {
			err = e.useSkill(u, c)
		}
		if err == nil {
			u.Moved, u.Acted, u.Done = true, true, true
		}
		return err
	}
	return fail("UnknownCommand")
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
		if len(c.Deployment) < 1 || len(c.Deployment) > s.Limit {
			return fail("InvalidDeployment")
		}
		seen := map[string]bool{}
		for _, id := range c.Deployment {
			if ch := e.char(id); ch == nil || ch.Dead || e.isDeputy(id) || seen[id] || !e.def(ch).Playable {
				return fail("InvalidDeployment")
			}
			seen[id] = true
		}
		for _, id := range keys(e.charIDs) {
			if e.def(e.char(id)).Lord && !seen[id] {
				return fail("InvalidDeployment")
			}
		}
		e.state.Deployment = append([]string{}, c.Deployment...)
		return nil
	}
	if c.Kind == "equip" || c.Kind == "unequip" {
		o := e.char(c.Actor)
		if o == nil || !e.def(o).Playable || o.Dead || e.isDeputy(o.ID) {
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
		if !e.carries(o, eq) {
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
	if len(e.state.Deployment) > e.data.Stages[e.state.Stage].Limit {
		return fail("InvalidDeployment")
	}
	// Remove prior transient units and enemies from the preparation checkpoint.
	s := e.Snapshot()
	s.Units = nil
	s.Characters = nil
	s.Checkpoint = nil
	for _, id := range keys(e.charIDs) {
		if o := e.char(id); !o.Temp {
			s.Characters = append(s.Characters, copyOf(*o))
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
		e.spawn(id, id, "ally", p.X, p.Y, content.Spawn{})
	}
	for _, sp := range stage.Enemies {
		e.spawn(sp.Character, sp.Character, "enemy", sp.X, sp.Y, sp)
	}
	return nil
}

// carries reports whether the character's class can wield the equipment.
func (e *Engine) carries(o *Character, eq content.Equipment) bool {
	cl := e.class(o)
	return !(eq.Slot == "weapon" && eq.Type != cl.Weapon || eq.Slot == "armor" && eq.Type != cl.Armor)
}

// spawn puts a unit on the map. A template definition gets a numbered character of its own;
// a named one is the party member or boss recorded under its own id.
func (e *Engine) spawn(id, def, side string, x, y int, sp content.Spawn) *Unit {
	d := e.data.Characters[def]
	if d.Template {
		n := 1
		for e.char(def+"#"+strconv.Itoa(n)) != nil {
			n++
		}
		id = def + "#" + strconv.Itoa(n)
	}
	c := e.addCharacter(id, def)
	if sp.Level > 0 {
		c.Level = sp.Level
	}
	color := d.Look.Faction
	if side == "enemy" && e.state.Stage < len(e.data.Stages) {
		if f := e.data.Stages[e.state.Stage].Faction; f != "" && sp.Faction == "" {
			color = f
		}
	}
	if sp.Faction != "" {
		color = sp.Faction
	}
	u := Unit{ID: id, Character: id, Faction: side, Color: color, X: x, Y: y, Status: map[string]Status{}, Items: map[string]int{}}
	if side == "enemy" {
		for item, n := range e.data.Rules.EnemyStock {
			u.Items[item] = n
		}
	}
	st := e.stats(&u)
	u.HP = st.MaxHP
	u.MP = 0
	e.unitIDs[id] = e.units.NewEntity(&u)
	return e.unit(id)
}
func (e *Engine) settle() {
	if e.state.Phase != "battle" {
		return
	}
	stage := e.data.Stages[e.state.Stage]
	alive := map[string]bool{}
	loss := false
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.HP > 0 {
			alive[u.Faction] = true
		} else if u.Faction == "ally" && e.unitDef(u).Lord {
			loss = true // the campaign is lost with its lord
		}
	}
	loss, win := loss || !alive["ally"], !alive["enemy"]
	if stage.Goal != "rout" {
		boss := e.unit(stage.Boss)
		win = boss.HP*100 <= e.stats(boss).MaxHP*stage.Threshold
	}
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
	s.Learning = nil
	s.Round = 0
	s.Turn = ""
	s.Result = ""
	kept := []Character{}
	for _, o := range s.Characters {
		if !o.Temp {
			kept = append(kept, o)
		}
	}
	s.Characters = kept
	*e = *fromState(e.data, s)
}
