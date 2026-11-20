package core

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"srpg/internal/content"
	"strings"
)

func (e *Engine) LegalActions(actor string) Options {
	out := Options{Commands: []Command{}}
	if len(e.state.Learning) > 0 {
		for _, id := range e.state.Learning {
			for _, t := range e.learnable(e.char(id)) {
				if e.char(id).Points >= e.data.Traits[t].Cost {
					out.Commands = append(out.Commands, Command{Kind: "learn", Actor: id, Trait: t})
				}
			}
			out.Commands = append(out.Commands, Command{Kind: "learn_close", Actor: id})
		}
		return out
	}
	if e.Deploying() {
		if actor == "" {
			if e.deploymentReady() {
				out.Commands = append(out.Commands, Command{Kind: "deployment_finish"})
			}
		} else if contains(e.state.Deployment, actor) {
			if e.unit(actor) != nil {
				out.Commands = append(out.Commands, Command{Kind: "unplace", Actor: actor})
				for _, p := range e.data.Stages[e.state.Stage].Allies {
					if e.canReposition(actor, p) {
						out.Commands = append(out.Commands, Command{Kind: "reposition", Actor: actor, X: p.X, Y: p.Y})
					}
				}
			} else {
				for _, p := range e.data.Stages[e.state.Stage].Allies {
					if e.deploymentCell(actor, p) {
						out.Commands = append(out.Commands, Command{Kind: "place", Actor: actor, X: p.X, Y: p.Y})
					}
				}
			}
		}
		return out
	}
	u := e.unit(actor)
	if e.state.Phase != "battle" || u == nil || u.HP <= 0 || !u.ActsIn(e.state.Turn, e.state.Army) || u.Done {
		return out
	}
	out.Commands = append(out.Commands, Command{Kind: "wait", Actor: actor})
	for _, p := range e.moves(u) {
		out.Commands = append(out.Commands, Command{Kind: "move", Actor: actor, X: p.X, Y: p.Y})
	}
	kinds := []string{""}
	kinds = append(kinds, e.skills(e.char(u.Character))...)
	for _, sk := range kinds {
		k := "skill"
		if sk == "" {
			k = "attack"
		}
		for _, id := range keys(e.unitIDs) {
			c := Command{Kind: k, Actor: actor, Target: id, Skill: sk}
			s, _ := e.skillDef(c)
			if s.Mode == content.ModeSelf && id != actor {
				continue
			}
			if _, err := e.validateSkill(u, c, s); err == nil {
				out.Commands = append(out.Commands, c)
			}
		}
	}
	if !u.Acted && !confused(u) {
		stock := e.stock(u)
		for _, item := range keys(stock) {
			if stock[item] < 1 || e.data.Items[item].Effect == content.ItemPromote { // seals are used off the field
				continue
			}
			for _, id := range keys(e.unitIDs) {
				v := e.unit(id)
				if v.HP > 0 && v.Faction == u.Faction && dist(u.X, u.Y, v.X, v.Y) <= 1 {
					out.Commands = append(out.Commands, Command{Kind: "item", Actor: actor, Target: id, Item: item})
				}
			}
		}
	}
	return out
}
func (e *Engine) Preview(c Command) (Preview, error) {
	u := e.unit(c.Actor)
	if e.state.Phase != "battle" || u == nil || u.HP <= 0 {
		return Preview{}, fail("InvalidActor")
	}
	if !u.ActsIn(e.state.Turn, e.state.Army) {
		return Preview{}, fail("NotYourTurn")
	}
	if u.Done {
		return Preview{}, fail("ActionSpent")
	}
	if c.Kind == "item" {
		t := fromState(e.data, e.Snapshot())
		v := t.unit(c.Target)
		if v == nil {
			return Preview{}, fail("InvalidTarget")
		}
		hp, mp := v.HP, v.MP
		if err := t.execute(c); err != nil {
			return Preview{}, err
		}
		p := Preview{Healing: v.HP - hp, MPRecovery: v.MP - mp, Hit: 100, Targets: []string{v.ID}}
		for _, ev := range t.events {
			if ev.Kind == "experience" && ev.Actor == u.Character {
				p.XP += ev.Amount
			}
		}
		p.CritXP = p.XP
		return p, nil
	}
	s, err := e.skillDef(c)
	if err != nil {
		return Preview{}, err
	}
	v, err := e.validateSkill(u, c, s)
	if err != nil {
		return Preview{}, err
	}
	p := Preview{Cost: e.cost(u, s), Hit: e.hitChance(u, v, s), Effect: s.Effect}
	phys := s.Kind == content.KindPhysical
	if phys || s.Kind == content.KindMagic {
		p.Crit = e.critChance(u, v, s)
	}
	if phys {
		p.Double = e.doubleChance(u, v, s)
	}
	var hit []*Unit
	share := map[string]float64{}
	if phys {
		hit, share, _ = e.victims(u, v, s)
	} else {
		hit = e.targets(u, v, s)
	}
	for _, q := range hit {
		p.Targets = append(p.Targets, q.ID)
		heal := 0
		if s.Effect == "heal" {
			heal = min(e.stats(q).MaxHP-q.HP, e.healAmount(u, float64(e.stats(u).Mind)*s.Coeff))
			p.Healing += heal
		}
		if phys || s.Kind == content.KindMagic {
			k := 1.0
			if phys {
				k = share[q.ID]
			}
			d := e.damage(u, q, s) * k
			normal, crit := int(math.Floor(e.bulwark(q, d*e.boost(u, q, s, false)))), int(math.Floor(e.bulwark(q, d*e.boost(u, q, s, true))))
			p.MaxDamage += crit
			if p.Damage == nil {
				p.Damage, p.CritDamage = map[string]int{}, map[string]int{}
			}
			p.Damage[q.ID], p.CritDamage[q.ID] = normal, crit
			p.XP += e.actionXPAmount(u, q, c.Kind == "skill", normal >= q.HP)
			p.CritXP += e.actionXPAmount(u, q, c.Kind == "skill", crit >= q.HP)
		} else if u.Faction == "ally" && (s.Effect != "heal" || heal > 0) {
			p.XP += e.supportXP(u, q)
			p.CritXP += e.supportXP(u, q)
		}
	}
	return p, nil
}

// SetDamagePct scales the damage of every blow to pct percent; the setting travels with snapshots,
// so previews and trial copies deal the same damage.
func (e *Engine) SetDamagePct(pct int) { e.state.DamagePct = pct }

// SetImmortal keeps units standing: no blow brings them below 1 HP. who is a comma-separated list of characters,
// "ally" for every unit of the player's side. Only the simulators use it (cmd/_campaign -immortal), to follow a
// campaign past battles its AI player would lose; it is not saved.
func (e *Engine) SetImmortal(who string) { e.immortal = who }

// SimGrant gives the party n of an item, a piece of equipment or money, as a reward would. Only the simulators use it
// (cmd/_campaign -subs, the bonus objectives its AI player misses).
func (e *Engine) SimGrant(item string, n int) { e.gain(item, n) }

func (e *Engine) undying(v *Unit) bool {
	if e.immortal == "" {
		return false
	}
	for _, w := range strings.Split(e.immortal, ",") {
		if w == v.Character || w == "ally" && v.Faction == "ally" {
			return true
		}
	}
	return false
}

// SetXPPct scales the experience every battle action earns to pct percent.
func (e *Engine) SetXPPct(pct int) { e.state.XPPct = pct }

// Restore rebuilds an engine from a state saved with this data; one saved before the data changed fails
// with ChangedContent, which RestoreChanged takes anyway.
func Restore(d *content.Data, s State) (*Engine, error) { return reported(restore(d, copyOf(s), true)) }

// RestoreChanged restores a state saved before the data last changed (while the game is still being
// made, a save should outlive an edited line or map); the engine carries the data's hash from then on.
func RestoreChanged(d *content.Data, s State) (*Engine, error) {
	s = copyOf(s)
	s.ContentHash = d.Hash
	adaptCharacters(d, &s)
	if s.Checkpoint != nil {
		s.Checkpoint.ContentHash = d.Hash
		adaptCharacters(d, s.Checkpoint)
	}
	return reported(restore(d, s, true))
}

// RestoreWarned, when set, hears the rule mismatches a release build lets a restore pass with (lenientRestore),
// with the state they were found in.
var RestoreWarned func(warnings []string, s State)

func reported(e *Engine, warnings []string, err error) (*Engine, error) {
	if err == nil && len(warnings) > 0 && RestoreWarned != nil {
		RestoreWarned(warnings, e.Snapshot())
	}
	return e, err
}

// adaptCharacters fits the party of a save to edited data: a learned trait the data dropped is forgotten, an
// officer whose class the data moved out of his old line takes the class his level reaches on the new one, and
// a piece of equipment the officer can no longer wear goes back to the inventory.
func adaptCharacters(d *content.Data, s *State) {
	for _, o := range append(pointers(s.Characters), pointers(s.Away)...) {
		def, ok := d.Characters[o.Def]
		if !ok || o.Temp || o.Placement {
			continue
		}
		o.Learned = slices.DeleteFunc(o.Learned, func(t string) bool { _, ok := d.Traits[t]; return !ok })
		if _, ok := d.Classes[o.Class]; !ok || !classReachable(d, d.ClassLine(def.Class)[0], o.Class) {
			o.Class = d.PromotedAt(o.Def, def.Class, o.Level)
		}
		held := append(append([]string{}, innate(o)...), o.Learned...)
		for _, slot := range slices.Sorted(maps.Keys(o.Equipment)) {
			eq := o.Equipment[slot]
			if v, ok := d.Equipment[eq]; ok && v.Slot == slot && d.Carries(d.Classes[o.Class], held, v) {
				continue
			}
			delete(o.Equipment, slot)
			if _, ok := d.Equipment[eq]; ok {
				if s.Inventory == nil {
					s.Inventory = map[string]int{}
				}
				s.Inventory[eq]++
			}
		}
	}
}

func pointers[T any](s []T) []*T {
	p := make([]*T, len(s))
	for i := range s {
		p[i] = &s[i]
	}
	return p
}

// knownDifficulty tells whether a save's difficulty is one the rules still have.
func knownDifficulty(d *content.Data, id string) bool {
	_, ok := d.Rules.DifficultyOf(id)
	return ok || id == "normal"
}

// corrupt is a restore failure: the code the screens translate, then what in the state broke the check.
func corrupt(code, format string, a ...any) error {
	return fmt.Errorf("%s: %s", code, fmt.Sprintf(format, a...))
}

// restore checks a saved state and builds its engine. A broken shape (an unknown id, a unit off the map, two on
// one cell) always fails. A state that only breaks a rule (gear the class cannot carry, a status the list lacks)
// fails too, unless lenientRestore: then the engine runs it and the mismatch comes back as a warning.
func restore(d *content.Data, s State, checkpoint bool) (*Engine, []string, error) {
	R := d.Rules
	if s.AI != AIPolicyVersion || !knownDifficulty(d, s.Difficulty) || s.Format != 5 || s.Core != CoreVersion || s.Rules != RulesVersion || s.Content != d.Version {
		return nil, nil, fail("IncompatibleSave")
	}
	if s.ContentHash != d.Hash {
		return nil, nil, fail("ChangedContent")
	}
	var warnings []string
	bad := func(format string, a ...any) (*Engine, []string, error) {
		return nil, nil, corrupt("CorruptSave", format, a...)
	}
	// soft is a rule mismatch: nil when lenientRestore takes it as a warning, else the failure.
	soft := func(code, format string, a ...any) error {
		if lenientRestore {
			warnings = append(warnings, corrupt(code, format, a...).Error())
			return nil
		}
		return corrupt(code, format, a...)
	}
	for i, t := range s.TileStates {
		if s.Stage < 0 || s.Stage >= len(d.Stages) {
			break
		}
		st := d.Stages[s.Stage]
		if _, ok := d.States[t.State]; !ok && t.State != string(content.NoLayer) || t.X < 0 || t.Y < 0 || t.X >= st.Width || t.Y >= st.Height || t.Turns < 0 {
			return bad("tile state %q at (%d,%d) for %d turns", t.State, t.X, t.Y, t.Turns)
		}
		for _, o := range s.TileStates[:i] {
			if o.X == t.X && o.Y == t.Y {
				return bad("two tile states at (%d,%d)", t.X, t.Y)
			}
		}
	}
	if s.Stage < 0 || s.Stage >= len(d.Stages) {
		return bad("stage %d of %d", s.Stage, len(d.Stages))
	}
	if s.Inventory == nil || s.Warehouse == nil || s.Executed == nil || s.Flags == nil || s.Choices == nil {
		return bad("a nil map among inventory, warehouse, executed, flags and choices")
	}
	if !contains([]string{"scenario", "deployment", "battle", "result", "complete"}, s.Phase) {
		return bad("phase %q", s.Phase)
	}
	if s.Phase == "scenario" {
		n := d.Node(s.Node)
		if !content.Waiting(n.Kind) || n.Kind == "battle" && d.StageIndex(n.Stage) != s.Stage || n.Kind == "explore" && s.Explore != n.ID {
			return bad("scenario at node %q (%s) with stage %d and explore %q", s.Node, n.Kind, s.Stage, s.Explore)
		}
	}
	if s.Money < 0 {
		return bad("money %d", s.Money)
	}
	if s.Explore != "" && (s.Phase != "scenario" || d.Node(s.Explore).Kind != "explore" || d.Explores[d.Node(s.Explore).Explore].Areas[s.Spot.Area] == nil) {
		return bad("explore %q in phase %s at area %q", s.Explore, s.Phase, s.Spot.Area)
	}
	if s.Phase == "complete" && s.Node != "" || s.Ending != "" && (s.Phase != "complete" || d.Node(s.Ending).Kind != "end") {
		return bad("phase %s at node %q with ending %q", s.Phase, s.Node, s.Ending)
	}
	seen := map[string]bool{}
	for _, o := range s.Characters {
		def, ok := d.Characters[o.Def]
		switch _, cl := d.Classes[o.Class]; {
		case !ok:
			return bad("character %s: unknown def %q", o.ID, o.Def)
		case !cl:
			return bad("character %s: unknown class %q", o.ID, o.Class)
		case o.Temp != (def.Template || o.Placement):
			return bad("character %s: temp %v, template %v, placement %v", o.ID, o.Temp, def.Template, o.Placement)
		case !o.Temp && o.ID != o.Def:
			return bad("character %s: def %s differs", o.ID, o.Def)
		case seen[o.ID]:
			return bad("character %s twice", o.ID)
		case o.Level < 1 || o.Level > R.MaxLevel:
			return bad("character %s: level %d", o.ID, o.Level)
		case o.Equipment == nil:
			return bad("character %s: nil equipment", o.ID)
		}
		for slot, eq := range o.Equipment {
			if _, ok := d.Equipment[eq]; !ok {
				return bad("character %s: unknown equipment %q in %s", o.ID, eq, slot)
			}
		}
		var err error
		switch t := d.Classes[o.Class].Tier; {
		case !o.Placement && !classReachable(d, d.ClassLine(def.Class)[0], o.Class):
			err = soft("CorruptSave", "character %s: class %s is off the line of %s", o.ID, o.Class, def.Class)
		case o.Placement && !slices.ContainsFunc(d.ClassLine(def.Class), func(c string) bool { return d.ClassFamily(c) == d.ClassFamily(o.Class) }):
			err = soft("CorruptSave", "character %s: placed class %s is off the families of %s", o.ID, o.Class, def.Class)
		case o.XP < 0 || o.XP >= XPPerLevel || o.Points < 0 || o.XPShareRemainder < 0 || o.XPShareRemainder >= 100:
			err = soft("CorruptSave", "character %s: xp %d, points %d, share remainder %d", o.ID, o.XP, o.Points, o.XPShareRemainder)
		case !o.Placement && t > 0 && o.Level < R.PromoteLevels[t-1]:
			err = soft("CorruptSave", "character %s: class %s (tier %d) at level %d, below %d", o.ID, o.Class, t, o.Level, R.PromoteLevels[t-1])
		}
		if err != nil {
			return nil, nil, err
		}
		if o.TraitsOverride != nil {
			set := map[string]bool{}
			for _, id := range *o.TraitsOverride {
				if _, ok := d.Traits[id]; !ok || set[id] {
					if err := soft("CorruptSave", "character %s: innate trait %q unknown or repeated", o.ID, id); err != nil {
						return nil, nil, err
					}
				}
				set[id] = true
			}
		}
		seen[o.ID] = true
		learned := map[string]bool{}
		for _, t := range o.Learned {
			if _, ok := d.Traits[t]; !ok || learned[t] {
				if err := soft("CorruptSave", "character %s: learned trait %q unknown or repeated", o.ID, t); err != nil {
					return nil, nil, err
				}
			}
			learned[t] = true
		}
		held := append(append([]string{}, innate(&o)...), o.Learned...)
		for slot, eq := range o.Equipment {
			v := d.Equipment[eq]
			var err error
			if v.Slot != slot {
				err = soft("CorruptSave", "character %s: %s (a %s) in %s", o.ID, eq, v.Slot, slot)
			} else if !d.Carries(d.Classes[o.Class], held, v) {
				err = soft("CorruptSave", "character %s: class %s cannot carry %s in %s", o.ID, o.Class, eq, slot)
			}
			if err != nil {
				return nil, nil, err
			}
		}
	}
	for _, id := range d.Campaign.Roster {
		if !seen[id] {
			return bad("roster officer %s missing", id)
		}
	}
	deploy := map[string]bool{}
	if len(s.Deployment) < 1 {
		return bad("no one deployed")
	}
	reserve := d.Stages[s.Stage].Reserve // they sortie on top of Max and stand on no cell until a join event calls them in
	sortie := len(s.Deployment)
	for _, id := range s.Deployment {
		if contains(reserve, id) {
			sortie--
		}
	}
	if sortie > d.Stages[s.Stage].Max && s.Phase != "scenario" && s.Phase != "complete" { // the story may add officers after the last battle
		if err := soft("CorruptSave", "%d deployed, at most %d", len(s.Deployment), d.Stages[s.Stage].Max); err != nil {
			return nil, nil, err
		}
	}
	for _, id := range s.Deployment {
		if !seen[id] || deploy[id] || !d.Characters[id].Playable {
			return bad("deployed %s unknown, repeated or not playable", id)
		}
		deploy[id] = true
	}
	for id, c := range d.Characters {
		if c.Lord && seen[id] && !deploy[id] && s.Phase != "scenario" {
			if err := soft("CorruptSave", "lord %s not deployed in phase %s", id, s.Phase); err != nil {
				return nil, nil, err
			}
		}
	}
	for id, n := range s.Inventory {
		if _, ok := d.Items[id]; !ok || n < 0 {
			return bad("inventory item %q ×%d", id, n)
		}
	}
	for id, n := range s.Warehouse {
		if _, ok := d.Equipment[id]; !ok || n < 0 {
			return bad("warehouse equipment %q ×%d", id, n)
		}
	}
	for _, o := range s.Away {
		def, ok := d.Characters[o.Def]
		if _, cl := d.Classes[o.Class]; !ok || !cl || def.Template || o.ID != o.Def || seen[o.ID] || o.Left < 1 || o.Level < 1 || o.Level > R.MaxLevel || o.Equipment == nil {
			return bad("away officer %s (def %s, class %s, level %d, left %d)", o.ID, o.Def, o.Class, o.Level, o.Left)
		}
		seen[o.ID] = true
	}
	if s.Checkpoint != nil {
		if !checkpoint || s.Checkpoint.Checkpoint != nil || s.Checkpoint.Phase != "scenario" || s.Checkpoint.Stage != s.Stage || d.Node(s.Checkpoint.Node).Kind != "battle" {
			return nil, nil, corrupt("CorruptCheckpoint", "checkpoint at node %q in phase %s, stage %d (state's %d)", s.Checkpoint.Node, s.Checkpoint.Phase, s.Checkpoint.Stage, s.Stage)
		}
		_, w, err := restore(d, *s.Checkpoint, false)
		if err != nil {
			return nil, nil, fmt.Errorf("%w (in the checkpoint)", err)
		}
		for _, x := range w {
			warnings = append(warnings, x+" (in the checkpoint)")
		}
	}
	e := fromState(d, s)
	deputies := map[string]bool{}
	for _, o := range s.Characters {
		if o.Deputy == "" {
			continue
		}
		q := e.char(o.Deputy)
		if q == nil || q.ID == o.ID || deputies[q.ID] {
			return nil, nil, corrupt("CorruptDeputy", "%s cannot have %s as deputy", o.ID, o.Deputy)
		}
		if o.Temp && q.Temp { // an NPC officer's deputy, set by its spawn
			deputies[q.ID] = true
			continue
		}
		if o.Dead || !e.playable(e.char(o.ID)) || e.fxOf(&o, content.FxDeputyAppoint) == 0 || !e.playable(q) || e.def(q).Lord || q.Dead || q.Deputy != "" || deploy[q.ID] || len(q.Equipment) > 0 {
			if err := soft("CorruptDeputy", "%s cannot have %s as deputy", o.ID, o.Deputy); err != nil {
				return nil, nil, err
			}
		}
		deputies[q.ID] = true
	}
	for _, id := range s.Deployment {
		if e.char(id).Dead && s.Phase != "battle" && s.Phase != "result" {
			if err := soft("CorruptDeployment", "%s deployed dead in phase %s", id, s.Phase); err != nil {
				return nil, nil, err
			}
		}
	}
	for _, id := range s.Learning {
		if !seen[id] || !d.Characters[e.char(id).Def].Playable {
			return bad("learning %s", id)
		}
	}
	ids := map[string]bool{}
	positions := map[content.Point]bool{}
	stage := d.Stages[s.Stage]
	fixed := map[string]string{}
	fixedUnits := map[string]Unit{}
	joined := map[string]bool{} // a join skips an officer already on the map or fallen, so its unit may be missing
	if s.Phase == "deployment" || s.Phase == "battle" || s.Phase == "result" {
		if s.Checkpoint == nil {
			return bad("phase %s without a checkpoint", s.Phase)
		}
		expected := fromState(d, copyOf(*s.Checkpoint))
		for _, side := range []struct {
			faction string
			spawns  []content.Spawn
		}{{"ally", stage.Party}, {"enemy", stage.Enemies}} {
			for _, sp := range side.spawns {
				u := expected.spawn(sp.Character, sp.Character, side.faction, sp.X, sp.Y, sp)
				if deploy[u.ID] {
					return bad("stage unit %s also deployed", u.ID)
				}
				fixed[u.ID] = side.faction
				fixedUnits[u.ID] = copyOf(*u)
			}
		}
		if s.Phase != "deployment" {
			for _, event := range stage.Events {
				key := "battle-" + stage.ID + "-" + event.ID
				times := len(s.Fired[key]) // an event with Every brings its units each time it ran
				if times == 0 && s.Executed[key] {
					times = 1
				}
				for range times {
					for _, a := range event.Do {
						for _, sp := range a.Spawns {
							side := "ally"
							if sp.Side == "enemy" {
								side = "enemy"
							}
							u := expected.spawn(sp.Character, sp.Character, side, sp.X, sp.Y, sp)
							fixed[u.ID] = side
							fixedUnits[u.ID] = copyOf(*u)
							joined[u.ID] = joined[u.ID] || a.Kind == "join"
						}
					}
				}
			}
		}
	}

	// first is the first rule mismatch among checks, each a soft call or nil; a lenient restore makes them all nil.
	first := func(errs ...error) error {
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		return nil
	}
	for _, u := range s.Units {
		if !stage.ValidPolicy(u.Policy, u.Goal, u.GoalArea) {
			return bad("unit %s: policy %q, goal %v, area %d", u.ID, u.Policy, u.Goal, u.GoalArea) // an event may change a unit's policy, so only its shape is checked
		}
		switch {
		case ids[u.ID]:
			return bad("unit %s twice", u.ID)
		case u.ID != u.Character || !seen[u.Character]:
			return bad("unit %s: character %q", u.ID, u.Character)
		case u.Status == nil || u.Items == nil:
			return bad("unit %s: nil status or items", u.ID)
		case !contains([]string{"ally", "enemy"}, u.Faction):
			return bad("unit %s: faction %q", u.ID, u.Faction)
		case u.X < 0 || u.Y < 0 || u.X >= stage.Width || u.Y >= stage.Height:
			return bad("unit %s at (%d,%d) off the %d×%d map", u.ID, u.X, u.Y, stage.Width, stage.Height)
		}
		for id, n := range u.Items {
			if _, ok := d.Items[id]; !ok || n < 0 {
				return bad("unit %s: item %q ×%d", u.ID, id, n)
			}
		}
		ids[u.ID] = true
		p := content.Point{X: u.X, Y: u.Y}
		if u.HP > 0 && positions[p] {
			return bad("unit %s shares (%d,%d)", u.ID, u.X, u.Y)
		}
		if u.HP > 0 {
			positions[p] = true
		}
		var errs []error
		if base, ok := fixedUnits[u.ID]; ok && (u.AI != base.AI || u.DuelFirst != base.DuelFirst) {
			errs = append(errs, soft("CorruptSave", "unit %s: ai %v, duel first %v; the stage gives %v, %v", u.ID, u.AI, u.DuelFirst, base.AI, base.DuelFirst))
		} else if !ok && (u.AI || u.DuelFirst) {
			errs = append(errs, soft("CorruptSave", "unit %s: ai %v, duel first %v off the stage's units", u.ID, u.AI, u.DuelFirst))
		}
		if u.Facing != "" && !contains([]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}, u.Facing) {
			errs = append(errs, soft("CorruptSave", "unit %s: facing %q", u.ID, u.Facing))
		}
		if u.Faction == "ally" && !deploy[u.ID] && fixed[u.ID] != "ally" {
			errs = append(errs, soft("CorruptSave", "unit %s: an ally neither deployed nor the stage's", u.ID))
		}
		if !stage.Boats && e.openWater(u.X, u.Y) {
			errs = append(errs, soft("CorruptSave", "unit %s on open water at (%d,%d)", u.ID, u.X, u.Y))
		}
		st := e.stats(&u)
		if e.char(u.Character).Dead && u.HP != 0 {
			errs = append(errs, soft("CorruptSave", "unit %s: dead with %d HP", u.ID, u.HP))
		}
		if u.HP < 0 || u.HP > st.MaxHP {
			errs = append(errs, soft("CorruptSave", "unit %s: HP %d of %d", u.ID, u.HP, st.MaxHP))
		}
		if u.MP < 0 {
			errs = append(errs, soft("CorruptSave", "unit %s: MP %d", u.ID, u.MP))
		}
		if u.Attacked && !u.Acted || u.Done && (!u.Acted || !u.Moved) {
			errs = append(errs, soft("CorruptSave", "unit %s: moved %v, acted %v, attacked %v, done %v", u.ID, u.Moved, u.Acted, u.Attacked, u.Done))
		}
		for id, v := range u.Status {
			if !content.Statuses[id] || v.Turns < 1 || v.Turns > 99 || abs(v.Value) > 100 || (v.Source != "" && !ids[v.Source] && e.unit(v.Source) == nil) {
				errs = append(errs, soft("CorruptSave", "unit %s: status %q %+v", u.ID, id, v))
			}
		}
		if err := first(errs...); err != nil {
			return nil, nil, err
		}
	}
	if s.Phase == "deployment" || s.Phase == "battle" || s.Phase == "result" {
		if s.Round < 1 || !contains([]string{"ally", "support", "enemy"}, s.Turn) {
			return bad("round %d, turn %q", s.Round, s.Turn)
		}
		if s.Phase == "result" && !contains([]string{"victory", "defeat"}, s.Result) {
			return bad("result %q", s.Result)
		}
		for id := range fixed {
			if !ids[id] && !joined[id] {
				return bad("stage unit %s missing", id)
			}
		}
		var errs []error
		for id := range deploy {
			if !ids[id] && s.Phase != "deployment" && !contains(reserve, id) {
				errs = append(errs, soft("CorruptSave", "deployed %s has no unit", id))
			}
		}
		for id, faction := range fixed {
			if u := e.unit(id); u != nil && u.Faction != faction && !deploy[id] {
				errs = append(errs, soft("CorruptSave", "stage unit %s on side %s, not %s", id, u.Faction, faction))
			}
		}
		for id := range ids {
			if fixed[id] == "" && !deploy[id] {
				errs = append(errs, soft("CorruptSave", "unit %s comes from neither the stage, its events nor the deployment", id))
			}
		}
		if s.Phase == "deployment" {
			if s.Round != 1 || s.Turn != "ally" || s.Result != "" || len(s.Learning) != 0 {
				errs = append(errs, soft("CorruptSave", "deployment at round %d, turn %q, result %q, %d learning", s.Round, s.Turn, s.Result, len(s.Learning)))
			}
			for _, u := range s.Units {
				if base, ok := fixedUnits[u.ID]; ok && (u.X != base.X || u.Y != base.Y || u.HP != base.HP || u.MP != base.MP) {
					errs = append(errs, soft("CorruptSave", "stage unit %s at (%d,%d) %d HP %d MP before the battle; the stage gives (%d,%d) %d %d", u.ID, u.X, u.Y, u.HP, u.MP, base.X, base.Y, base.HP, base.MP))
				}
				if u.Moved || u.Acted || u.Attacked || u.Done || u.HP <= 0 || !maps.Equal(u.Status, e.startStatus(&u)) {
					errs = append(errs, soft("CorruptSave", "unit %s acted, fell or changed status before the battle", u.ID))
				}
				if deploy[u.ID] { // MP may start above 0 (start_mana) or be full after a promotion on the field
					if st := e.stats(&u); !containsPoint(stage.Allies, content.Point{X: u.X, Y: u.Y}) || e.terrain(&u, u.X, u.Y).Cost <= 0 || u.HP != st.MaxHP || u.MP > st.MaxMP {
						errs = append(errs, soft("CorruptSave", "deployed %s at (%d,%d) with %d HP %d MP", u.ID, u.X, u.Y, u.HP, u.MP))
					}
				}
			}
		}
		if err := first(errs...); err != nil {
			return nil, nil, err
		}
	}
	return e, warnings, nil
}

// CharacterStats is a character's battle stats with their current class, equipment and deputy,
// for screens outside battle.
func (e *Engine) CharacterStats(id string) (Stats, error) {
	if e.char(id) == nil {
		return Stats{}, fail("InvalidCharacter")
	}
	return e.stats(&Unit{ID: id, Character: id, Faction: "ally", Status: map[string]Status{}}), nil
}

// CharacterView describes the officer's equipment, traits and skills outside battle, without temporary statuses.
func (e *Engine) CharacterView(id string) (UnitView, error) {
	c := e.char(id)
	if c == nil {
		return UnitView{}, fail("InvalidCharacter")
	}
	s, err := e.CharacterStats(id)
	if err != nil {
		return UnitView{}, err
	}
	return UnitView{Unit: Unit{ID: id, Character: id, Faction: "ally", HP: s.MaxHP, MP: s.MaxMP}, Name: e.name(c), Class: c.Class, Def: c.Def, Level: c.Level, XP: c.XP, Points: c.Points, Stats: s, Traits: e.traits(c), Skills: e.skills(c)}, nil
}

// CharacterStatsAs is the character's battle stats were it of the given class, for comparing a promotion.
func (e *Engine) CharacterStatsAs(id, class string) (Stats, error) {
	c := e.char(id)
	if c == nil {
		return Stats{}, fail("InvalidCharacter")
	}
	if _, ok := e.data.Classes[class]; !ok {
		return Stats{}, fail("InvalidClass")
	}
	old := c.Class
	c.Class = class
	defer func() { c.Class = old }()
	return e.CharacterStats(id)
}

// CounterDamage is the most damage the target can return to the actor after c lands,
// or 0 when c is not a physical action or the actor is out of the target's reach.
func (e *Engine) CounterDamage(c Command) int {
	u, v := e.counterUnits(c)
	if u == nil {
		return 0
	}
	return int(math.Floor(e.damage(v, u, e.basic()) * (e.data.Rules.CritMultiplier + e.fx(v, content.FxCritBoost)) * e.counterTaken(u, e.counterShare(v))))
}

// CounterOdds returns the target's chances when it can counter the action. The counter
// still requires the target to survive the attack.
func (e *Engine) CounterOdds(c Command) (hit, crit, double float64, ok bool) {
	u, v := e.counterUnits(c)
	if u == nil {
		return 0, 0, 0, false
	}
	s := e.basic()
	return e.hitChance(v, u, s), e.critChance(v, u, s), e.doubleChance(v, u, s), true
}

func (e *Engine) counterUnits(c Command) (*Unit, *Unit) {
	u, v := e.unit(c.Actor), e.unit(c.Target)
	if u == nil || v == nil || v.HP <= 0 || v.Faction == u.Faction || confused(v) {
		return nil, nil
	}
	s, err := e.skillDef(c)
	if err != nil || s.Kind != content.KindPhysical || !e.has(v, content.FxRetaliate) && !e.inRange(v, u, e.basic()) {
		return nil, nil
	}
	return u, v
}

// Strike is the expected damage of a's best blow on b that a can pay for, its basic attack or a
// damaging skill with normal hits weighed by their chance, wherever the two stand. The AI weighs
// threats and the blows a move would open with it.
func (e *Engine) Strike(a, b string) float64 {
	u, v := e.unit(a), e.unit(b)
	if u == nil || v == nil || u.HP <= 0 || v.HP <= 0 {
		return 0
	}
	best := e.damage(u, v, e.basic()) * e.hitChance(u, v, e.basic()) / 100
	for _, name := range e.skills(e.char(u.Character)) {
		s := e.data.Skills[name]
		if (s.Kind == content.KindPhysical || s.Kind == content.KindMagic) && e.cost(u, s) <= u.MP {
			best = max(best, e.damage(u, v, s)*e.boost(u, v, s, false)*e.hitChance(u, v, s)/100)
		}
	}
	return best
}

// MoveCosts gives the movement cost to each reachable cell this turn.
func (e *Engine) MoveCosts(id string) map[content.Point]int {
	u := e.unit(id)
	if u == nil {
		return nil
	}
	costs, _ := e.reach(u)
	return costs
}

// AttackRange is the distance band the unit's basic attack falls in; CanStrikeFrom tells the exact cells.
func (e *Engine) AttackRange(id string) (lo, hi int) {
	u := e.unit(id)
	if u == nil {
		return 0, 0
	}
	return e.rangeFor(u, e.basic())
}

// CanStrikeFrom tells whether the unit standing at from would reach (x, y) with its basic attack.
func (e *Engine) CanStrikeFrom(id string, from content.Point, x, y int) bool {
	u := e.unit(id)
	return u != nil && e.reachesFrom(u, e.basic(), from, x, y)
}

// CanCastFrom tells whether the unit's skill would reach (x, y) from `from`, as it would from where it stands.
func (e *Engine) CanCastFrom(id, skill string, from content.Point, x, y int) bool {
	u := e.unit(id)
	s, ok := e.data.Skills[skill]
	return u != nil && ok && e.reachesFrom(u, s, from, x, y)
}

// ShootsAt tells whether the unit's basic attack from `from` on (x, y) would be a shot rather than a
// melee blow (a rider with 기사 beyond its spear's reach).
func (e *Engine) ShootsAt(id string, from content.Point, x, y int) bool {
	u := e.unit(id)
	if u == nil {
		return false
	}
	_, ranged := e.weaponReach(u, e.basic(), from, x, y)
	return ranged
}

// ActionCells are the cells the action reaches from where its actor stands: its weapon's own pattern for
// an attack, a skill's band, the cells around for an item.
func (e *Engine) ActionCells(c Command) []content.Point {
	u := e.unit(c.Actor)
	if u == nil {
		return nil
	}
	from := content.Point{X: u.X, Y: u.Y}
	if c.Kind == "item" {
		out := []content.Point{from}
		for _, d := range []content.Point{{Y: -1}, {X: -1}, {X: 1}, {Y: 1}} {
			out = append(out, content.Point{X: u.X + d.X, Y: u.Y + d.Y})
		}
		return out
	}
	s, err := e.skillDef(c)
	if err != nil {
		return nil
	}
	set := map[content.Point]bool{}
	e.cellsFrom(e.offsets(u, s), from, set)
	return sortedCells(set)
}

// offsets are the cells, relative to the unit, that it reaches with the skill.
func (e *Engine) offsets(u *Unit, s content.Skill) []content.Point {
	_, hi := e.rangeFor(u, s)
	var out []content.Point
	for y := -hi; y <= hi; y++ {
		for x := -hi; x <= hi; x++ {
			if e.reachesFrom(u, s, content.Point{}, x, y) {
				out = append(out, content.Point{X: x, Y: y})
			}
		}
	}
	return out
}

// cellsFrom adds to set the map cells the offsets reach from a cell.
func (e *Engine) cellsFrom(offsets []content.Point, from content.Point, set map[content.Point]bool) {
	stage := e.data.Stages[e.state.Stage]
	for _, d := range offsets {
		if x, y := from.X+d.X, from.Y+d.Y; x >= 0 && y >= 0 && x < stage.Width && y < stage.Height {
			set[content.Point{X: x, Y: y}] = true
		}
	}
}

func sortedCells(set map[content.Point]bool) []content.Point {
	out := make([]content.Point, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Y == out[j].Y {
			return out[i].X < out[j].X
		}
		return out[i].Y < out[j].Y
	})
	return out
}

// StepCost is what entering (x, y) costs the unit, 0 where it cannot enter.
func (e *Engine) StepCost(id string, x, y int) int {
	u := e.unit(id)
	if u == nil {
		return 0
	}
	return e.terrain(u, x, y).Cost
}

// Terrain is what the cell at (x, y) is to the unit — its class's cost and factor with the unit's own traits.
func (e *Engine) Terrain(id string, x, y int) (content.Terrain, bool) {
	u := e.unit(id)
	if u == nil {
		return content.Terrain{}, false
	}
	return e.terrain(u, x, y), true
}

// Threat is every cell the unit could attack next time it acts: its movement reach from
// a fresh turn, widened by its basic attack and damaging skills.
func (e *Engine) Threat(id string) []content.Point {
	r := e.Ranges(id)
	set := map[content.Point]bool{}
	for _, p := range append(r.Attack, r.Spell...) {
		set[p] = true
	}
	return sortedCells(set)
}

// Ranges are what a unit can do next time it acts, from a fresh turn: the cells it can end its move on, those
// its basic attack and physical skills, or its stratagems, reach from anywhere its move gets to, and those its
// basic attack and physical skills reach from where it stands.
type Ranges struct {
	Move, Attack, Spell, Strike []content.Point
}

func (e *Engine) Ranges(id string) Ranges {
	v := e.unit(id)
	if v == nil || v.HP <= 0 || e.state.Phase != "battle" {
		return Ranges{}
	}
	u := copyOf(*v)
	u.Moved, u.Done = false, false
	cost, _ := e.reach(&u)
	if len(cost) == 0 {
		cost = map[content.Point]int{{X: u.X, Y: u.Y}: 0}
	}
	attack, spell := map[content.Point]bool{}, map[content.Point]bool{}
	for _, d := range e.offsets(&u, e.basic()) {
		attack[d] = true
	}
	for _, name := range e.skills(e.char(u.Character)) {
		s := e.data.Skills[name]
		switch {
		case s.Mode == content.ModeSelf:
		case s.Kind == content.KindPhysical:
			for _, d := range e.offsets(&u, s) {
				attack[d] = true
			}
		case s.Kind == content.KindMagic || s.Kind == content.KindStatus:
			for _, d := range e.offsets(&u, s) {
				spell[d] = true
			}
		}
	}
	move := map[content.Point]bool{}
	for p := range cost {
		if o := e.occupied(p.X, p.Y); o == nil || o.ID == u.ID {
			move[p] = true
		}
	}
	from := func(offsets, starts map[content.Point]bool) []content.Point {
		if len(offsets) == 0 {
			return nil
		}
		ds := sortedCells(offsets)
		set := map[content.Point]bool{}
		for p := range starts {
			e.cellsFrom(ds, p, set)
		}
		return sortedCells(set)
	}
	moved := map[content.Point]bool{}
	for p := range cost {
		moved[p] = true
	}
	here := map[content.Point]bool{{X: u.X, Y: u.Y}: true}
	return Ranges{Move: sortedCells(move), Attack: from(attack, moved), Spell: from(spell, moved), Strike: from(attack, here)}
}
