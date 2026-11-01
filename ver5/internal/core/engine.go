package core

import (
	"encoding/json"
	"fmt"
	"github.com/mlange-42/ark/ecs"
	"slices"
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
	turnStarted      string // the faction whose turn this command began, for turnstart conditions
	lastActor        string // who the command in progress is for, so a duel knows who moved to meet whom
	caster           string // who is using a skill now, and the spell points it began with (인내)
	castMP           int
}

func copyOf[T any](v T) T { b, _ := json.Marshal(v); var c T; _ = json.Unmarshal(b, &c); return c }
func New(d *content.Data, seed uint64) *Engine {
	e := begin(d, seed)
	e.enter()
	return e
}

// begin is a new game before it enters its first node.
func begin(d *content.Data, seed uint64) *Engine {
	c := d.Campaign
	s := State{Format: 5, AI: AIPolicyVersion, Difficulty: "normal", Core: CoreVersion, Rules: RulesVersion, Content: d.Version, ContentHash: d.Hash,
		Seed: seed, RNG: seed, Phase: "scenario", Node: c.Start, Inventory: copyOf(c.Inventory), Warehouse: map[string]int{},
		Deployment: append([]string{}, c.Deployment...), Executed: map[string]bool{}, Flags: map[string]int{}, Choices: map[string]string{}, Money: c.Money}
	if s.Inventory == nil {
		s.Inventory = map[string]int{}
	}
	e := fromState(d, s)
	for _, id := range c.Roster {
		e.addCharacter(id, id)
	}
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
func (e *Engine) Snapshot() State { return copyOf(e.current()) }

// current is the state with its characters and units, sharing their maps and slices with the engine.
// A committed state is never changed in place (Apply runs a command on a copy and swaps it in), so
// what it shares stays as it was; readers must not change it.
func (e *Engine) current() State {
	s := e.state
	for _, id := range keys(e.charIDs) {
		s.Characters = append(s.Characters, *e.char(id))
	}
	for _, id := range keys(e.unitIDs) {
		s.Units = append(s.Units, *e.unit(id))
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
	c := Character{ID: id, Def: def, Class: d.Class, Level: d.Level, Points: d.Points, Temp: d.Template, Equipment: map[string]string{}, Learned: []string{}}
	for _, eq := range d.Equipment {
		c.Equipment[e.data.Equipment[eq].Slot] = eq
	}
	e.charIDs[id] = e.chars.NewEntity(&c)
	return e.char(id)
}
// Observe is what a screen shows of the state. It shares the state's and the content's maps and
// slices instead of copying them: callers only read it.
func (e *Engine) Observe() Observation {
	s := e.current()
	o := Observation{State: s, Dialogue: e.data.Node(s.Node)}
	if s.Phase == "deployment" || s.Phase == "battle" || s.Phase == "result" {
		m := e.data.Stages[s.Stage]
		m.Tiles = slices.Clone(m.Tiles)
		for _, t := range s.TileStates {
			at := t.X*content.CellWidth + content.CellWidth - 1
			m.Tiles[t.Y] = m.Tiles[t.Y][:at] + t.State + m.Tiles[t.Y][at+1:]
		}
		if g := s.Goals; g != nil {
			m.Victory, m.Defeat, m.Sub = g.Victory, g.Defeat, g.Sub
		}
		o.Map = &m
	}
	for _, u := range s.Units {
		c := e.char(u.Character)
		o.UnitViews = append(o.UnitViews, UnitView{Unit: u, Name: e.name(c), Class: c.Class, Def: c.Def, Shoot: e.shootClass(&u), Weapon: e.weaponType(&u), Level: c.Level, XP: c.XP, XPRequired: XPPerLevel,
			XPProgress: ExperienceProgress(c.XP), Stats: e.stats(&u), Attributes: attributes(e.attributes(c)), Points: c.Points,
			Traits: e.traits(c), Skills: e.skills(c)})
	}
	for _, id := range s.Learning {
		c := e.char(id)
		o.Learners = append(o.Learners, Learner{ID: id, Name: e.name(c), Points: c.Points, Options: e.learnable(c)})
	}
	return o
}

// ObservationCache keeps one engine's Observation until the engine is replaced or its state changes,
// so a screen that reads it every frame builds it once per change. A command counts a revision; the
// settings (SetDamagePct, SetXPPct) change the state without one, so they are compared
// too. What it returns is shared: callers read it and never change it.
type ObservationCache struct {
	e *Engine
	o Observation
}

func (c *ObservationCache) Of(e *Engine) Observation {
	s := &e.state
	if c.e != e || c.o.Revision != s.Revision || c.o.DamagePct != s.DamagePct || c.o.XPPct != s.XPPct {
		c.e, c.o = e, e.Observe()
	}
	return c.o
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
	t.lastActor = c.Actor
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

// traits is every trait the character has: innate (or set by its stage), learned, given by equipment, and the deputy's own.
func (e *Engine) traits(c *Character) []string {
	set := map[string]bool{}
	add := func(s []string) {
		for _, id := range s {
			set[id] = true
		}
	}
	if c.TraitsOverride != nil {
		add(*c.TraitsOverride)
	} else {
		add(e.def(c).Traits)
	}
	add(c.Learned)
	for _, eq := range c.Equipment {
		add(e.data.Equipment[eq].Traits)
	}
	if d := e.char(c.Deputy); d != nil && !d.Dead {
		add(e.def(d).Traits)
		add(d.Learned)
	}
	return keys(set)
}

// fx totals one trait effect over the unit's traits and the stances its statuses put on it.
func (e *Engine) fx(u *Unit, key string) float64 {
	n := e.fxOf(e.char(u.Character), key)
	for id, st := range u.Status {
		if st.Turns > 0 {
			n += content.StatusEffects[id][key]
		}
	}
	return n
}
func (e *Engine) fxOf(c *Character, key string) float64 {
	n := 0.0
	for _, t := range e.traits(c) {
		n += e.data.Traits[t].Effects[key]
	}
	return n
}
func (e *Engine) has(u *Unit, key string) bool         { return e.fx(u, key) != 0 }

// ownFx totals one trait effect over the traits the character holds itself, leaving out its deputy's and
// its equipment's.
func (e *Engine) ownFx(c *Character, key string) float64 {
	held := e.def(c).Traits
	if c.TraitsOverride != nil {
		held = *c.TraitsOverride
	}
	n := 0.0
	for _, t := range append(slices.Clone(held), c.Learned...) {
		n += e.data.Traits[t].Effects[key]
	}
	return n
}
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
// enter runs the story from the current node up to the next one that waits for the player (or the end).
func (e *Engine) enter() {
	for {
		if e.state.Node == "" {
			e.state.Phase = "complete"
			return
		}
		n := e.data.Node(e.state.Node)
		switch n.Kind {
		case "battle":
			e.state.Stage = e.data.StageIndex(n.Stage)
			e.releaseRequired(e.data.Stages[e.state.Stage])
			e.fitSortie(e.data.Stages[e.state.Stage])
			return
		case "scene":
			e.state.Place = n.Place
			e.state.Node = e.data.After(n.ID)
		case "set":
			for k, v := range n.Flags {
				e.state.Flags[k] = v
			}
			e.state.Node = e.data.After(n.ID)
		case "branch":
			next := n.Else
			for _, c := range n.Cases {
				if e.holdsAll(c.If) {
					next = c.Next
					break
				}
			}
			if next == "" {
				next = e.data.After(n.ID)
			}
			e.state.Node = next
		case "join", "grant":
			e.once(n)
			e.state.Node = e.data.After(n.ID)
		case "explore":
			if e.state.Explore != n.ID {
				x := e.data.Explores[n.Explore]
				e.state.Explore, e.state.Spot = n.ID, x.Start
				if n.Opening != "" {
					e.state.Node = n.Opening
					continue
				}
			}
			return
		case "back":
			if n.Leave {
				next := e.data.After(e.state.Explore)
				e.state.Explore, e.state.Spot = "", content.Spot{}
				e.state.Node = next
				continue
			}
			if n.Area != "" {
				e.state.Spot = e.data.Explores[e.data.Node(e.state.Explore).Explore].Arrival(n.Area, "")
				if n.At != nil {
					e.state.Spot.At = *n.At
				}
				if n.Face != "" {
					e.state.Spot.Face = n.Face
				}
			}
			e.state.Node = e.state.Explore
		case "end":
			e.state.Phase, e.state.Node = "complete", ""
			return
		default:
			return
		}
	}
}

// once runs a join or grant node the first time the story reaches it.
func (e *Engine) once(n content.Node) {
	if e.state.Executed[n.Event] {
		return
	}
	if n.Kind == "join" {
		e.raiseTo(e.addCharacter(n.Character, n.Character), e.data.Rules.StageLevel(e.data.NextBattleOrder(n.ID)))
		e.state.Deployment = append(e.state.Deployment, n.Character)
		e.state.Executed[n.Event] = true
		e.emit(n.Kind, "", n.Character, n.Count)
		return
	}
	e.gain(n.Item, n.Count)
	if o := e.char(n.Character); o != nil && !o.Dead && e.carries(o, e.data.Equipment[n.Item]) {
		e.wear(o, n.Item)
	}
	e.state.Executed[n.Event] = true
	e.emit(n.Kind, "", n.Item, n.Count)
}

// wear puts a piece from the warehouse on o; the piece it replaces goes back to the warehouse.
func (e *Engine) wear(o *Character, id string) {
	slot := e.data.Equipment[id].Slot
	e.state.Warehouse[id]--
	if old := o.Equipment[slot]; old != "" {
		e.state.Warehouse[old]++
	}
	o.Equipment[slot] = id
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
		if c.Kind == "next" && (n.Kind == "say" || n.Kind == "narrate") {
			e.state.Node = e.data.After(n.ID)
		} else if c.Kind == "choose" && n.Kind == "choice" {
			v, ok := n.Choices[c.Option]
			if !ok {
				return fail("InvalidOption")
			}
			e.state.Choices[n.ID] = c.Option
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
		next := e.data.After(e.state.Node)
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
		if e.Deploying() {
			return e.redeploy(c)
		}
		return e.prepare(c)
	case "walk", "talk", "sortie", "buy", "sell":
		return e.explore(c)
	case "learn", "learn_close":
		return e.learn(c)
	case "place", "unplace", "reposition", "deployment_finish":
		return e.deployCommand(c)
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
	if !u.ActsIn(e.state.Turn, e.state.Army) {
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
		path := e.route(u, p)
		u.X = c.X
		u.Y = c.Y
		u.Moved = true
		u.Steps = len(path) - 1
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
	kind := e.data.Node(e.state.Node).Kind
	if e.state.Phase != "scenario" && !e.Deploying() || kind != "battle" && (kind != "explore" || c.Kind == "deploy" || c.Kind == "start") {
		return fail("WrongPhase")
	}
	if c.Kind == "deputy" || c.Kind == "undeputy" {
		return e.assignDeputy(c)
	}
	if c.Kind == "deploy" {
		s := e.data.Stages[e.state.Stage]
		if len(c.Deployment) < 1 || len(c.Deployment) > s.Max || !e.requiredIn(s, c.Deployment) {
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
		e.wear(o, c.Item)
		return nil
	}
	if c.Kind != "start" {
		return fail("UnknownCommand")
	}
	if st := e.data.Stages[e.state.Stage]; len(e.state.Deployment) < 1 || len(e.state.Deployment) > st.Max || !e.requiredIn(st, e.state.Deployment) {
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
	e.state.Phase = "deployment"
	e.state.Turn = "ally"
	e.state.Round = 1
	e.state.Result = ""
	stage := e.data.Stages[e.state.Stage]
	for _, sp := range stage.Party {
		if !e.data.Characters[sp.Character].Template && contains(e.state.Deployment, sp.Character) {
			return fail("InvalidDeployment")
		}
		e.spawn(sp.Character, sp.Character, "ally", sp.X, sp.Y, sp)
	}
	for _, sp := range stage.Enemies {
		e.spawn(sp.Character, sp.Character, "enemy", sp.X, sp.Y, sp)
	}
	for _, id := range keys(e.unitIDs) {
		e.faceObjective(e.unit(id), nil)
	}
	if stage.Forced { // the officers march straight onto their cells and the battle begins
		plan, err := e.DeploymentPlan()
		if err != nil {
			return err
		}
		for _, c := range plan {
			if err := e.deployCommand(c); err != nil {
				return err
			}
		}
	}
	return nil
}

// carries reports whether the character's class and traits allow wielding the equipment.
func (e *Engine) carries(o *Character, eq content.Equipment) bool {
	return e.data.Carries(e.class(o), e.traits(o), eq)
}

// stageOrder is which battle of the story the current stage is, from 1.
func (e *Engine) stageOrder() int {
	if e.state.Stage < 0 || e.state.Stage >= len(e.data.Stages) {
		return 0
	}
	return e.data.Stages[e.state.Stage].Order
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
	if side == "enemy" || d.Template || sp.AI {
		c.Temp = true
		c.Placement = true
	}
	if c.Temp || sp.Level > 0 {
		c.Level = e.data.SpawnLevel(e.stageOrder(), sp)
	}
	if sp.Class != "" {
		c.Class = sp.Class
	}
	if side == "enemy" {
		c.Class = e.promotedAt(def, c.Class, c.Level)
	}
	if sp.Traits != nil {
		c.TraitsOverride = copyOf(sp.Traits)
	}
	if side == "enemy" {
		e.grantLevelTraits(c)
	}
	if sp.Equipment != nil {
		c.Equipment = map[string]string{}
		for _, id := range *sp.Equipment {
			c.Equipment[e.data.Equipment[id].Slot] = id
		}
	}
	color := d.Look.Faction
	if side == "enemy" && e.state.Stage >= 0 && e.state.Stage < len(e.data.Stages) {
		if f := e.data.Stages[e.state.Stage].Faction; f != "" && sp.Faction == "" {
			color = f
		}
	}
	if sp.Faction != "" {
		color = sp.Faction
	}
	u := Unit{AI: sp.AI, Policy: sp.Policy, Goal: copyGoal(sp.Goal), GoalArea: sp.GoalArea, DuelFirst: sp.DuelFirst, Group: sp.Group, Rank: sp.Rank, ID: id, Character: id, Faction: side, Color: color, Facing: sp.Facing, X: x, Y: y, Status: map[string]Status{}, Items: map[string]int{}}
	npcStock := e.data.Rules.EnemyStock
	if side == "ally" {
		npcStock = e.data.Rules.AllyStock
	}
	if u.NPC() {
		for item, n := range npcStock {
			u.Items[item] = n
		}
	}
	st := e.stats(&u)
	u.HP = st.MaxHP
	u.MP = 0
	if sp.HP != nil {
		u.HP = min(st.MaxHP, max(1, *sp.HP))
	}
	if sp.MP != nil {
		u.MP = min(st.MaxMP, max(0, *sp.MP))
	}
	u.Status = e.startStatus(&u)
	e.unitIDs[id] = e.units.NewEntity(&u)
	return e.unit(id)
}

// promotedAt is the class a character reaches from class after every promotion the level allows, each time to its
// first branch.
func (e *Engine) promotedAt(def, class string, level int) string {
	for {
		cl, next := e.data.Classes[class], e.data.Promotes(def, class)
		if cl.Tier >= len(e.data.Rules.PromoteLevels) || level < e.data.Rules.PromoteLevels[cl.Tier] || len(next) == 0 {
			return class
		}
		class = next[0]
	}
}

// grantLevelTraits gives an enemy the traits its class holds at its level (Rules.EnemyTraits) on top of
// those its stage sets.
func (e *Engine) grantLevelTraits(c *Character) {
	held := slices.Clone(e.def(c).Traits)
	if c.TraitsOverride != nil {
		held = slices.Clone(*c.TraitsOverride)
	}
	for _, lt := range e.data.Rules.EnemyTraits[c.Class] {
		if c.Level >= lt.Level && !contains(held, lt.Trait) {
			held = append(held, lt.Trait)
		}
	}
	c.TraitsOverride = &held
}

// startStatus is what a unit is placed with: the buffs its traits grant every turn.
func (e *Engine) startStatus(u *Unit) map[string]Status {
	s := map[string]Status{}
	for id, k := range e.turnBuffs(u) {
		s[id] = Status{Turns: e.data.Rules.BuffTurns, Value: k}
	}
	return s
}

// turnBuffs are the stat buffs, in percent, the unit's traits put on it every own turn.
func (e *Engine) turnBuffs(u *Unit) map[string]int {
	out := map[string]int{}
	for id := range content.StatStatuses {
		if k := int(e.fx(u, content.FxTurnBuff+id)); k > 0 {
			out[id] = k
		}
	}
	return out
}
func (e *Engine) settle() {
	if e.state.Phase != "battle" {
		return
	}
	if limit := e.data.Stages[e.state.Stage].TurnLimit; limit > 0 && e.state.Round > limit {
		e.state.Phase, e.state.Result = "result", "defeat"
		e.emit("turn-limit", "", "", limit)
		return
	}
	e.leaveByFlight()
	e.autoDuels()
	e.battleEvents()
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
	goals := e.battleGoals()
	loss = loss || !alive["ally"] || e.anyGoal(goals.Defeat)
	win := e.anyGoal(goals.Victory)
	if !loss && !win {
		return
	}
	e.state.Phase = "result"
	e.state.Result = "defeat"
	if loss {
		return
	}
	e.state.Result = "victory"
	e.state.Executed["won-"+stage.ID] = true
	key := "reward-" + stage.ID
	if !e.state.Executed[key] {
		for item, n := range stage.Reward {
			e.gain(item, n)
		}
		e.state.Executed[key] = true
	}
	for _, o := range goals.Sub {
		if key := "sub-" + stage.ID + "-" + o.ID; !e.state.Executed[key] && e.holdsIn(o.If, true) {
			e.state.Executed[key] = true
			for item, n := range o.Reward {
				e.gain(item, n)
			}
			if o.Flag != "" {
				if e.state.PendingFlags == nil {
					e.state.PendingFlags = map[string]int{}
				}
				e.state.PendingFlags[o.Flag] = 1
			}
			e.emit("objective", "", o.ID, 0)
		}
	}
	for k, v := range e.state.PendingFlags {
		e.state.Flags[k] = v
	}
	e.state.PendingFlags = nil
	e.emit("victory", "", stage.ID, 0)
}

// anyGoal reports whether one of the goals holds: all conditions of a goal must.
func (e *Engine) anyGoal(goals []content.Goal) bool {
	for _, g := range goals {
		if e.holdsIn(g.If, true) {
			return true
		}
	}
	return false
}

// SubDone tells for each bonus objective of the current stage whether it holds right now.
func (e *Engine) SubDone() []bool {
	if e.state.Stage < 0 || e.state.Stage >= len(e.data.Stages) {
		return nil
	}
	var out []bool
	for _, o := range e.battleGoals().Sub {
		out = append(out, e.state.Executed["sub-"+e.data.Stages[e.state.Stage].ID+"-"+o.ID] || e.state.Phase == "battle" && e.holdsIn(o.If, true))
	}
	return out
}

// Campaign transitions retire transient units and the previous retry checkpoint.
func (e *Engine) clearBattle() {
	s := e.Snapshot()
	s.Units = nil
	s.Checkpoint = nil
	s.Learning = nil
	s.PendingFlags = nil
	s.TileStates = nil
	s.Goals = nil
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

// SpawnPreview uses the battle initialization path for editor stat previews.
func SpawnPreview(d *content.Data, stage string, sp content.Spawn) UnitView {
	e := fromState(d, State{Difficulty: "normal", Stage: -1})
	u := e.spawn(sp.Character, sp.Character, "enemy", sp.X, sp.Y, sp)
	c := e.char(u.Character)
	c.Level = d.SpawnLevel(d.StageOrder(stage), sp)
	return UnitView{Unit: *u, Name: e.name(c), Class: c.Class, Def: c.Def, Shoot: e.shootClass(u), Weapon: e.weaponType(u), Level: c.Level, Stats: e.stats(u), Traits: e.traits(c), Skills: e.skills(c)}
}

// requiredIn reports whether the sortie holds every officer the stage requires who is able to go.
func (e *Engine) requiredIn(s content.Stage, sortie []string) bool {
	for _, id := range s.Required {
		if o := e.char(id); o != nil && !o.Dead && !e.isDeputy(id) && !contains(sortie, id) {
			return false
		}
	}
	return true
}

// fitSortie puts the officers a stage requires into the sortie; the player trims the rest to fit.
func (e *Engine) fitSortie(s content.Stage) {
	ids := append([]string{}, e.state.Deployment...)
	for i := len(s.Required) - 1; i >= 0; i-- {
		id := s.Required[i]
		if o := e.char(id); o != nil && !o.Dead && !e.isDeputy(id) && !contains(ids, id) {
			ids = append([]string{id}, ids...)
		}
	}
	e.state.Deployment = ids
}

func copyGoal(p *content.Point) *content.Point {
	if p == nil {
		return nil
	}
	q := *p
	return &q
}

// fleeing units that have reached their goal leave the field, as if retreated; the event's Text "flight"
// tells the screen they walk off instead of falling, and the fled condition remembers they got away.
func (e *Engine) leaveByFlight() {
	stage := e.data.Stages[e.state.Stage]
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.HP <= 0 || !content.Leaves(u.Policy) || !slices.Contains(stage.GoalCells(u.Goal, u.GoalArea), content.Point{X: u.X, Y: u.Y}) {
			continue
		}
		u.HP = 0
		e.state.Executed["fled-"+stage.ID+"-"+u.ID] = true
		e.events = append(e.events, Event{Kind: "retreat", Actor: u.ID, Text: "flight"})
	}
}
