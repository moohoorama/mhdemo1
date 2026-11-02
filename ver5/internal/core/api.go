package core

import (
	"fmt"
	"maps"
	"math"
	"sort"
	"srpg/internal/content"
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
	if !u.Acted {
		stock := e.stock(u)
		for _, item := range keys(stock) {
			if stock[item] < 1 {
				continue
			}
			for _, id := range keys(e.unitIDs) {
				v := e.unit(id)
				if v.HP > 0 && v.Faction == u.Faction && dist(u.X, u.Y, v.X, v.Y) <= 1 {
					if e.data.Items[item].Effect == content.ItemPromote { // one command per class it can make
						o := e.char(v.Character)
						for _, class := range e.data.Promotes(o.Def, o.Class) {
							c := Command{Kind: "item", Actor: actor, Target: id, Item: item, Option: class}
							if t := fromState(e.data, e.Snapshot()); t.execute(c) == nil {
								out.Commands = append(out.Commands, c)
							}
						}
						continue
					}
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
		hit, share = e.victims(u, v, s)
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
			normal, crit := int(math.Floor(e.bulwark(q, d))), int(math.Floor(e.bulwark(q, d*e.data.Rules.CritMultiplier)))
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

// SetXPPct scales the experience every battle action earns to pct percent.
func (e *Engine) SetXPPct(pct int) { e.state.XPPct = pct }

// Restore rebuilds an engine from a state saved with this data; one saved before the data changed fails
// with ChangedContent, which RestoreChanged takes anyway.
func Restore(d *content.Data, s State) (*Engine, error) { return restore(d, copyOf(s), true) }

// RestoreChanged restores a state saved before the data last changed (while the game is still being
// made, a save should outlive an edited line or map); the engine carries the data's hash from then on.
func RestoreChanged(d *content.Data, s State) (*Engine, error) {
	s = copyOf(s)
	s.ContentHash = d.Hash
	if s.Checkpoint != nil {
		s.Checkpoint.ContentHash = d.Hash
	}
	return restore(d, s, true)
}

func restore(d *content.Data, s State, checkpoint bool) (*Engine, error) {
	R := d.Rules
	if s.AI != AIPolicyVersion || s.Difficulty != "normal" || s.Format != 5 || s.Core != CoreVersion || s.Rules != RulesVersion || s.Content != d.Version {
		return nil, fail("IncompatibleSave")
	}
	if s.ContentHash != d.Hash {
		return nil, fail("ChangedContent")
	}
	for i, t := range s.TileStates {
		if s.Stage < 0 || s.Stage >= len(d.Stages) {
			break
		}
		st := d.Stages[s.Stage]
		if _, ok := d.States[t.State]; !ok && t.State != string(content.NoLayer) || t.X < 0 || t.Y < 0 || t.X >= st.Width || t.Y >= st.Height || t.Turns < 0 {
			return nil, fail("CorruptSave")
		}
		for _, o := range s.TileStates[:i] {
			if o.X == t.X && o.Y == t.Y {
				return nil, fail("CorruptSave")
			}
		}
	}
	if s.Stage < 0 || s.Stage >= len(d.Stages) || s.Inventory == nil || s.Warehouse == nil || s.Executed == nil || s.Flags == nil || s.Choices == nil {
		return nil, fail("CorruptSave")
	}
	if !contains([]string{"scenario", "deployment", "battle", "result", "complete"}, s.Phase) {
		return nil, fail("CorruptSave")
	}
	if s.Phase == "scenario" {
		n := d.Node(s.Node)
		if !content.Waiting(n.Kind) || n.Kind == "battle" && d.StageIndex(n.Stage) != s.Stage || n.Kind == "explore" && s.Explore != n.ID {
			return nil, fail("CorruptSave")
		}
	}
	if s.Money < 0 || s.Explore != "" && (s.Phase != "scenario" || d.Node(s.Explore).Kind != "explore" || d.Explores[d.Node(s.Explore).Explore].Areas[s.Spot.Area] == nil) {
		return nil, fail("CorruptSave")
	}
	if s.Phase == "complete" && s.Node != "" {
		return nil, fail("CorruptSave")
	}
	seen := map[string]bool{}
	for _, o := range s.Characters {
		def, ok := d.Characters[o.Def]
		_, cl := d.Classes[o.Class]
		if !ok || !cl || o.Temp != (def.Template || o.Placement) || !o.Temp && o.ID != o.Def || (!o.Placement && !classReachable(d, def.Class, o.Class) || o.Placement && d.ClassFamily(def.Class) != d.ClassFamily(o.Class)) || seen[o.ID] || o.Level < 1 || o.Level > R.MaxLevel ||
			o.XP < 0 || o.XP >= XPPerLevel || o.Points < 0 || o.XPShareRemainder < 0 || o.XPShareRemainder >= 100 || o.Equipment == nil {
			return nil, fail("CorruptSave")
		}
		if t := d.Classes[o.Class].Tier; !o.Placement && t > 0 && o.Level < R.PromoteLevels[t-1] {
			return nil, fail("CorruptSave")
		}
		if o.TraitsOverride != nil {
			set := map[string]bool{}
			for _, id := range *o.TraitsOverride {
				if _, ok := d.Traits[id]; !ok || set[id] {
					return nil, fail("CorruptSave")
				}
				set[id] = true
			}
		}
		seen[o.ID] = true
		learned := map[string]bool{}
		for _, t := range o.Learned {
			if _, ok := d.Traits[t]; !ok || learned[t] {
				return nil, fail("CorruptSave")
			}
			learned[t] = true
		}
		held := append(append([]string{}, innate(&o)...), o.Learned...)
		for slot, eq := range o.Equipment {
			v, ok := d.Equipment[eq]
			if !ok || v.Slot != slot || !d.Carries(d.Classes[o.Class], held, v) {
				return nil, fail("CorruptSave")
			}
		}
	}
	for _, id := range d.Campaign.Roster {
		if !seen[id] {
			return nil, fail("CorruptSave")
		}
	}
	deploy := map[string]bool{}
	if len(s.Deployment) < 1 || len(s.Deployment) > d.Stages[s.Stage].Max && s.Phase != "scenario" {
		return nil, fail("CorruptSave")
	}
	for _, id := range s.Deployment {
		if !seen[id] || deploy[id] || !d.Characters[id].Playable {
			return nil, fail("CorruptSave")
		}
		deploy[id] = true
	}
	for id, c := range d.Characters {
		if c.Lord && seen[id] && !deploy[id] && s.Phase != "scenario" {
			return nil, fail("CorruptSave")
		}
	}
	for id, n := range s.Inventory {
		if _, ok := d.Items[id]; !ok || n < 0 {
			return nil, fail("CorruptSave")
		}
	}
	for id, n := range s.Warehouse {
		if _, ok := d.Equipment[id]; !ok || n < 0 {
			return nil, fail("CorruptSave")
		}
	}
	if s.Checkpoint != nil {
		if !checkpoint || s.Checkpoint.Checkpoint != nil || s.Checkpoint.Phase != "scenario" || s.Checkpoint.Stage != s.Stage || d.Node(s.Checkpoint.Node).Kind != "battle" {
			return nil, fail("CorruptCheckpoint")
		}
		if _, err := restore(d, *s.Checkpoint, false); err != nil {
			return nil, err
		}
	}
	e := fromState(d, s)
	deputies := map[string]bool{}
	for _, o := range s.Characters {
		if o.Deputy == "" {
			continue
		}
		q := e.char(o.Deputy)
		if o.Dead || !e.playable(e.char(o.ID)) || e.fxOf(&o, content.FxDeputyAppoint) == 0 || q == nil || !e.playable(q) || e.def(q).Lord || q.ID == o.ID || q.Dead || q.Deputy != "" || deputies[q.ID] || deploy[q.ID] || len(q.Equipment) > 0 {
			return nil, fail("CorruptDeputy")
		}
		deputies[q.ID] = true
	}
	for _, id := range s.Deployment {
		if e.char(id).Dead && s.Phase != "battle" && s.Phase != "result" {
			return nil, fail("CorruptDeployment")
		}
	}
	for _, id := range s.Learning {
		if !seen[id] || !d.Characters[e.char(id).Def].Playable {
			return nil, fail("CorruptSave")
		}
	}
	ids := map[string]bool{}
	positions := map[content.Point]bool{}
	stage := d.Stages[s.Stage]
	fixed := map[string]string{}
	fixedUnits := map[string]Unit{}
	if s.Phase == "deployment" || s.Phase == "battle" || s.Phase == "result" {
		if s.Checkpoint == nil {
			return nil, fail("CorruptSave")
		}
		expected := fromState(d, copyOf(*s.Checkpoint))
		for _, side := range []struct {
			faction string
			spawns  []content.Spawn
		}{{"ally", stage.Party}, {"enemy", stage.Enemies}} {
			for _, sp := range side.spawns {
				u := expected.spawn(sp.Character, sp.Character, side.faction, sp.X, sp.Y, sp)
				if deploy[u.ID] {
					return nil, fail("CorruptSave")
				}
				fixed[u.ID] = side.faction
				fixedUnits[u.ID] = copyOf(*u)
			}
		}
		if s.Phase != "deployment" {
			for _, event := range stage.Events {
				if s.Executed["battle-"+stage.ID+"-"+event.ID] {
					for _, a := range event.Do {
						for _, sp := range a.Spawns {
							side := "ally"
							if sp.Side == "enemy" {
								side = "enemy"
							}
							u := expected.spawn(sp.Character, sp.Character, side, sp.X, sp.Y, sp)
							fixed[u.ID] = side
							fixedUnits[u.ID] = copyOf(*u)
						}
					}
				}
			}
		}
	}

	for _, u := range s.Units {
		if base, ok := fixedUnits[u.ID]; ok {
			if u.AI != base.AI || u.DuelFirst != base.DuelFirst {
				return nil, fail("CorruptSave")
			}
		} else if u.AI || u.DuelFirst {
			return nil, fail("CorruptSave")
		}
		if !stage.ValidPolicy(u.Policy, u.Goal, u.GoalArea) {
			return nil, fail("CorruptSave") // an event may change a unit's policy, so only its shape is checked
		}
		if u.Facing != "" && !contains([]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}, u.Facing) {
			return nil, fail("CorruptSave")
		}
		if ids[u.ID] || u.ID != u.Character || !seen[u.Character] || u.Status == nil || u.Items == nil || !contains([]string{"ally", "enemy"}, u.Faction) || u.Faction == "ally" && !deploy[u.ID] && fixed[u.ID] != "ally" || u.X < 0 || u.Y < 0 || u.X >= stage.Width || u.Y >= stage.Height || !stage.Boats && e.openWater(u.X, u.Y) {
			return nil, fail("CorruptSave")
		}
		ids[u.ID] = true
		p := content.Point{X: u.X, Y: u.Y}
		if u.HP > 0 && positions[p] {
			return nil, fail("CorruptSave")
		}
		if u.HP > 0 {
			positions[p] = true
		}
		st := e.stats(&u)
		if e.char(u.Character).Dead && u.HP != 0 || u.HP < 0 || u.HP > st.MaxHP || u.MP < 0 || u.MP > st.MaxMP || u.Attacked && !u.Acted || u.Done && (!u.Acted || !u.Moved) {
			return nil, fail("CorruptSave")
		}
		for id, v := range u.Status {
			if !content.Statuses[id] || v.Turns < 1 || v.Turns > 99 || abs(v.Value) > 100 || (v.Source != "" && !ids[v.Source] && e.unit(v.Source) == nil) {
				return nil, fail("CorruptSave")
			}
		}
		for id, n := range u.Items {
			if _, ok := d.Items[id]; !ok || n < 0 {
				return nil, fail("CorruptSave")
			}
		}
	}
	if s.Phase == "deployment" || s.Phase == "battle" || s.Phase == "result" {
		if s.Round < 1 || !contains([]string{"ally", "support", "enemy"}, s.Turn) || s.Checkpoint == nil {
			return nil, fail("CorruptSave")
		}
		for id := range deploy {
			if !ids[id] && s.Phase != "deployment" {
				return nil, fail("CorruptSave")
			}
		}
		for id, faction := range fixed {
			if !ids[id] || e.unit(id).Faction != faction {
				return nil, fail("CorruptSave")
			}
		}
		count := len(fixed)
		for id := range deploy {
			if ids[id] {
				count++
			}
		}
		if len(ids) != count {
			return nil, fail("CorruptSave")
		}
		if s.Phase == "deployment" {
			if s.Round != 1 || s.Turn != "ally" || s.Result != "" || len(s.Learning) != 0 {
				return nil, fail("CorruptSave")
			}
			for _, u := range s.Units {
				if base, ok := fixedUnits[u.ID]; ok {
					if u.X != base.X || u.Y != base.Y || u.HP != base.HP || u.MP != base.MP {
						return nil, fail("CorruptSave")
					}
				}
				if u.Moved || u.Acted || u.Attacked || u.Done || u.HP <= 0 || !maps.Equal(u.Status, e.startStatus(&u)) {
					return nil, fail("CorruptSave")
				}
				if deploy[u.ID] {
					if !containsPoint(stage.Allies, content.Point{X: u.X, Y: u.Y}) || e.terrain(&u, u.X, u.Y).Cost <= 0 || u.HP != e.stats(&u).MaxHP || u.MP != 0 {
						return nil, fail("CorruptSave")
					}
				}
			}
		}
		if s.Phase == "result" && !contains([]string{"victory", "defeat"}, s.Result) {
			return nil, fmt.Errorf("CorruptSave: result")
		}
	}
	return e, nil
}

// CharacterStats is a character's battle stats with their current class, equipment and deputy,
// for screens outside battle.
func (e *Engine) CharacterStats(id string) (Stats, error) {
	if e.char(id) == nil {
		return Stats{}, fail("InvalidCharacter")
	}
	return e.stats(&Unit{ID: id, Character: id, Faction: "ally", Status: map[string]Status{}}), nil
}

// CounterDamage is the most damage the target can return to the actor after c lands,
// or 0 when c is not a physical action or the actor is out of the target's reach.
func (e *Engine) CounterDamage(c Command) int {
	u, v := e.unit(c.Actor), e.unit(c.Target)
	if u == nil || v == nil || v.HP <= 0 || v.Faction == u.Faction {
		return 0
	}
	s, err := e.skillDef(c)
	if err != nil || s.Kind != content.KindPhysical || !e.has(v, content.FxRetaliate) && !e.inRange(v, u, e.basic()) {
		return 0
	}
	return int(math.Floor(e.damage(v, u, e.basic()) * (e.data.Rules.CritMultiplier + e.fx(v, content.FxCritBoost)) * e.counterShare(v)))
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
			best = max(best, e.damage(u, v, s)*e.hitChance(u, v, s)/100)
		}
	}
	return best
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

// Threat is every cell the unit could attack next time it acts: its movement reach from
// a fresh turn, widened by its basic attack and damaging skills.
func (e *Engine) Threat(id string) []content.Point {
	v := e.unit(id)
	if v == nil || v.HP <= 0 || e.state.Phase != "battle" {
		return nil
	}
	u := copyOf(*v)
	u.Moved, u.Done = false, false
	cost, _ := e.reach(&u)
	if len(cost) == 0 {
		cost = map[content.Point]int{{X: u.X, Y: u.Y}: 0}
	}
	reach := map[content.Point]bool{}
	for _, d := range e.offsets(&u, e.basic()) {
		reach[d] = true
	}
	for _, name := range e.skills(e.char(u.Character)) {
		s := e.data.Skills[name]
		if (s.Kind == content.KindPhysical || s.Kind == content.KindMagic || s.Kind == content.KindStatus) && s.Mode != content.ModeSelf {
			for _, d := range e.offsets(&u, s) {
				reach[d] = true
			}
		}
	}
	offsets := sortedCells(reach)
	set := map[content.Point]bool{}
	for p := range cost {
		e.cellsFrom(offsets, p, set)
	}
	return sortedCells(set)
}
