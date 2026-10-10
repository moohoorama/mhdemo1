package core

import (
	"fmt"
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
	u := e.unit(actor)
	if e.state.Phase != "battle" || u == nil || u.HP <= 0 || u.Faction != e.state.Turn || u.Done {
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
		stock := u.Items
		if u.Faction == "ally" {
			stock = e.state.Inventory
		}
		for _, item := range keys(stock) {
			if stock[item] < 1 {
				continue
			}
			for _, id := range keys(e.unitIDs) {
				v := e.unit(id)
				if v.HP > 0 && v.Faction == u.Faction && dist(u.X, u.Y, v.X, v.Y) <= 1 {
					if e.data.Items[item].Effect == content.ItemPromote {
						t := fromState(e.data, e.Snapshot())
						if t.execute(Command{Kind: "item", Actor: actor, Target: id, Item: item}) != nil {
							continue
						}
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
	if u.Faction != e.state.Turn {
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
		return Preview{Healing: v.HP - hp, MPRecovery: v.MP - mp, Hit: 100, Targets: []string{v.ID}}, nil
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
	if phys {
		p.Crit = e.critChance(u, v)
		p.Double = e.doubleChance(u, v)
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
			normal, crit := int(math.Floor(d)), int(math.Floor(d))
			if phys {
				crit = int(math.Floor(d * e.data.Rules.CritMultiplier))
			}
			p.MaxDamage += crit
			p.XP += e.actionXPAmount(u, q, c.Kind == "skill", normal >= q.HP)
			p.CritXP += e.actionXPAmount(u, q, c.Kind == "skill", crit >= q.HP)
		} else if u.Faction == "ally" && (s.Effect != "heal" || heal > 0) {
			p.XP += e.supportXP(u)
			p.CritXP += e.supportXP(u)
		}
	}
	return p, nil
}
func Restore(d *content.Data, s State) (*Engine, error) { return restore(d, copyOf(s), true) }
func restore(d *content.Data, s State, checkpoint bool) (*Engine, error) {
	R := d.Rules
	if s.AI != AIPolicyVersion || s.Difficulty != "normal" || s.Format != 2 || s.Core != CoreVersion || s.Rules != RulesVersion || s.Content != d.Version || s.ContentHash != d.Hash {
		return nil, fail("IncompatibleSave")
	}
	if s.Stage < 0 || s.Stage >= len(d.Stages) || s.Inventory == nil || s.Warehouse == nil || s.Executed == nil {
		return nil, fail("CorruptSave")
	}
	if !contains([]string{"scenario", "battle", "result", "complete"}, s.Phase) {
		return nil, fail("CorruptSave")
	}
	if s.Phase == "scenario" {
		n := d.Node(s.Node)
		if !contains([]string{"dialogue", "choice", "preparation"}, n.Kind) || n.Kind == "preparation" && d.StageIndex(n.Stage) != s.Stage {
			return nil, fail("CorruptSave")
		}
	}
	if s.Phase == "complete" && d.Stages[s.Stage].Next != "" {
		return nil, fail("CorruptSave")
	}
	seen := map[string]bool{}
	for _, o := range s.Characters {
		def, ok := d.Characters[o.Def]
		_, cl := d.Classes[o.Class]
		if !ok || !cl || o.Temp != def.Template || !o.Temp && o.ID != o.Def || !classReachable(d, def.Class, o.Class) || seen[o.ID] || o.Level < 1 || o.Level > R.MaxLevel ||
			o.XP < 0 || o.XP >= ExperienceRequired(d, o.Level) || o.Points < 0 || o.XPShareRemainder < 0 || o.XPShareRemainder >= 100 || o.Equipment == nil {
			return nil, fail("CorruptSave")
		}
		if t := d.Classes[o.Class].Tier; t > 0 && o.Level < R.PromoteLevels[t-1] {
			return nil, fail("CorruptSave")
		}
		seen[o.ID] = true
		learned := map[string]bool{}
		for _, t := range o.Learned {
			if _, ok := d.Traits[t]; !ok || learned[t] {
				return nil, fail("CorruptSave")
			}
			learned[t] = true
		}
		for slot, eq := range o.Equipment {
			v, ok := d.Equipment[eq]
			c := d.Classes[o.Class]
			if !ok || v.Slot != slot || slot == "weapon" && v.Type != c.Weapon || slot == "armor" && v.Type != c.Armor {
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
	if len(s.Deployment) < 1 || len(s.Deployment) > d.Stages[s.Stage].Limit && s.Phase != "scenario" {
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
		if !checkpoint || s.Checkpoint.Checkpoint != nil || s.Checkpoint.Phase != "scenario" || s.Checkpoint.Stage != s.Stage || d.Node(s.Checkpoint.Node).Kind != "preparation" {
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
		if o.Dead || !e.playable(e.char(o.ID)) || d.Classes[o.Class].Tier < R.DeputyMinTier || q == nil || !e.playable(q) || e.def(q).Lord || q.ID == o.ID || q.Dead || q.Deputy != "" || deputies[q.ID] || deploy[q.ID] || len(q.Equipment) > 0 {
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
	for _, u := range s.Units {
		if ids[u.ID] || u.ID != u.Character || !seen[u.Character] || u.Status == nil || u.Items == nil || !contains([]string{"ally", "enemy"}, u.Faction) || u.Faction == "ally" && !deploy[u.ID] || u.X < 0 || u.Y < 0 || u.X >= stage.Width || u.Y >= stage.Height || e.tileAt(u.X, u.Y).Water {
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
			if !content.Statuses[id] || v.Turns < 1 || v.Turns > 10 || abs(v.Value) > 100 || (v.Source != "" && !ids[v.Source] && e.unit(v.Source) == nil) {
				return nil, fail("CorruptSave")
			}
		}
		for id, n := range u.Items {
			if _, ok := d.Items[id]; !ok || n < 0 {
				return nil, fail("CorruptSave")
			}
		}
	}
	if s.Phase == "battle" || s.Phase == "result" {
		if s.Round < 1 || !contains([]string{"ally", "enemy"}, s.Turn) || stage.Goal != "rout" && !seen[stage.Boss] || s.Checkpoint == nil {
			return nil, fail("CorruptSave")
		}
		for id := range deploy {
			if !ids[id] {
				return nil, fail("CorruptSave")
			}
		}
		if len(ids) != len(deploy)+len(stage.Enemies) {
			return nil, fail("CorruptSave")
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
// or 0 when the target has no counter stance or the actor is out of its reach.
func (e *Engine) CounterDamage(c Command) int {
	u, v := e.unit(c.Actor), e.unit(c.Target)
	if u == nil || v == nil || v.HP <= 0 || v.Faction == u.Faction {
		return 0
	}
	forced := e.has(v, content.FxCounterForce) && e.might(u) < e.might(v)
	if v.Status["counter"].Turns == 0 && !forced {
		return 0
	}
	s, err := e.skillDef(c)
	if err != nil || s.Kind != content.KindPhysical || !e.inRange(v, u, e.basic()) {
		return 0
	}
	n := int(math.Floor(e.damage(v, u, e.basic()) * e.data.Rules.CritMultiplier * e.data.Rules.CounterShare))
	if forced {
		n *= 2
	}
	return n
}

// AttackRange is the reach of the unit's basic attack.
func (e *Engine) AttackRange(id string) (lo, hi int) {
	u := e.unit(id)
	if u == nil {
		return 0, 0
	}
	return e.rangeFor(u, e.basic())
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
	lo, hi := e.rangeFor(&u, e.basic())
	for _, name := range e.skills(e.char(u.Character)) {
		s := e.data.Skills[name]
		if s.Kind == content.KindPhysical || s.Kind == content.KindMagic || s.Kind == content.KindStatus {
			l, h := e.rangeFor(&u, s)
			lo, hi = min(lo, l), max(hi, h)
		}
	}
	stage := e.data.Stages[e.state.Stage]
	set := map[content.Point]bool{}
	for p := range cost {
		for y := p.Y - hi; y <= p.Y+hi; y++ {
			for x := p.X - hi; x <= p.X+hi; x++ {
				d := dist(p.X, p.Y, x, y)
				if d >= max(1, lo) && d <= hi && x >= 0 && y >= 0 && x < stage.Width && y < stage.Height {
					set[content.Point{X: x, Y: y}] = true
				}
			}
		}
	}
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
