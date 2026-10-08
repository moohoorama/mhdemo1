package core

import (
	"fmt"
	"math"
	"sort"
	"srpg/internal/content"
	"strings"
)

func (e *Engine) LegalActions(actor string) Options {
	out := Options{Commands: []Command{}}
	u := e.unit(actor)
	if e.state.Phase != "battle" || u == nil || u.HP <= 0 || u.Faction != e.state.Turn || u.Done {
		return out
	}
	out.Commands = append(out.Commands, Command{Kind: "wait", Actor: actor})
	for _, p := range e.moves(u) {
		out.Commands = append(out.Commands, Command{Kind: "move", Actor: actor, X: p.X, Y: p.Y})
	}
	kinds := []string{""}
	kinds = append(kinds, e.skills(u)...)
	for _, sk := range kinds {
		k := "skill"
		if sk == "" {
			k = "attack"
		}
		for _, id := range keys(e.unitIDs) {
			c := Command{Kind: k, Actor: actor, Target: id, Skill: sk}
			s, _ := e.skillDef(c)
			if s.Mode == "자기" && id != actor {
				continue
			}
			if _, err := e.validateSkill(u, c, s); err == nil {
				out.Commands = append(out.Commands, c)
			}
		}
	}
	if !u.Acted {
		for _, t := range e.learnable(u) {
			if e.officer(u.Officer).Points >= e.data.Traits[t].Cost {
				out.Commands = append(out.Commands, Command{Kind: "learn", Actor: actor, Trait: t})
			}
		}
	}
	if !u.Acted || e.has(u, "부호") {
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
				if v.HP > 0 && (v.Faction == u.Faction || item != "소병법단") && dist(u.X, u.Y, v.X, v.Y) <= 1 {
					if strings.HasPrefix(item, "승급:") {
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
	if s.Kind == "physical" {
		p.Crit = e.critChance(u, v)
		p.Double = e.doubleChance(u, v)
	}
	for _, q := range e.targets(u, v, s) {
		p.Targets = append(p.Targets, q.ID)
		heal := 0
		if s.Effect == "heal" {
			heal = min(e.stats(q).MaxHP-q.HP, e.healAmount(u, s))
			p.Healing += heal
		}
		if s.Kind == "physical" || s.Kind == "magic" {
			d := e.damage(u, q, s)
			normal, crit := int(math.Floor(d)), int(math.Floor(d))
			if s.Kind == "physical" {
				crit = int(math.Floor(d * 1.5))
			}
			p.MaxDamage += crit
			p.XP += e.actionXPAmount(u, q, 24, c.Kind == "skill", normal >= q.HP)
			p.CritXP += e.actionXPAmount(u, q, 24, c.Kind == "skill", crit >= q.HP)
		} else if u.Faction == "ally" && (s.Effect != "heal" || heal > 0) {
			p.XP += e.supportXP(u)
			p.CritXP += e.supportXP(u)
		}
	}
	return p, nil
}
func Restore(d *content.Data, s State) (*Engine, error) { return restore(d, copyOf(s), true) }
func restore(d *content.Data, s State, checkpoint bool) (*Engine, error) {
	if s.AI != AIPolicyVersion || s.Difficulty != "normal" || s.Format != 1 || s.Core != CoreVersion || s.Rules != RulesVersion || s.Content != d.Version || s.ContentHash != d.Hash {
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
		if !contains([]string{"dialogue", "choice", "preparation"}, n.Kind) {
			return nil, fail("CorruptSave")
		}
	}
	if s.Phase == "complete" && s.Stage != 2 {
		return nil, fail("CorruptSave")
	}
	seen := map[string]bool{}
	for _, o := range s.Officers {
		def, ok := d.Officers[o.ID]
		_, cl := d.Classes[o.Class]
		if !ok || !cl || !classReachable(d, def.Class, o.Class) || seen[o.ID] || o.Level < 1 || o.Level > 99 || o.XP < 0 || o.XP >= ExperienceRequired(o.Level) || o.Points < 0 || d.Classes[o.Class].Tier == 1 && o.Level < 15 || d.Classes[o.Class].Tier == 2 && o.Level < 30 || o.XPShareRemainder < 0 || o.XPShareRemainder >= 100 || o.Equipment == nil {
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
	if !seen["유비"] || !seen["관우"] || !seen["장비"] {
		return nil, fail("CorruptSave")
	}
	deploy := map[string]bool{}
	if len(s.Deployment) < 1 || len(s.Deployment) > d.Stages[s.Stage].Limit || !contains(s.Deployment, "유비") {
		return nil, fail("CorruptSave")
	}
	for _, id := range s.Deployment {
		if !seen[id] || deploy[id] || !contains([]string{"유비", "관우", "장비", "간옹"}, id) {
			return nil, fail("CorruptSave")
		}
		deploy[id] = true
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
	for _, o := range s.Officers {
		if o.Deputy == "" {
			continue
		}
		q := e.officer(o.Deputy)
		if o.Dead || !party(o.ID) || d.Classes[o.Class].Tier < 1 || q == nil || !party(q.ID) || q.ID == "유비" || q.ID == o.ID || q.Dead || q.Deputy != "" || deputies[q.ID] || deploy[q.ID] || len(q.Equipment) > 0 {
			return nil, fail("CorruptDeputy")
		}
		deputies[q.ID] = true
	}
	for _, id := range s.Deployment {
		if e.officer(id).Dead && s.Phase != "battle" && s.Phase != "result" {
			return nil, fail("CorruptDeployment")
		}
	}
	ids := map[string]bool{}
	positions := map[content.Point]bool{}
	stage := d.Stages[s.Stage]
	for _, u := range s.Units {
		if ids[u.ID] || u.ID != u.Officer || !seen[u.Officer] || u.Status == nil || u.Items == nil || !contains([]string{"ally", "enemy"}, u.Faction) || u.Faction == "ally" && !deploy[u.ID] || u.X < 0 || u.Y < 0 || u.X >= stage.Width || u.Y >= stage.Height || e.tile(u.X, u.Y) == "~" {
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
		if e.officer(u.Officer).Dead && u.HP != 0 || u.HP < 0 || u.HP > st.MaxHP || u.MP < 0 || u.MP > st.MaxMP || u.Attacked && !u.Acted || u.Done && (!u.Acted || !u.Moved) {
			return nil, fail("CorruptSave")
		}
		for id, v := range u.Status {
			if !contains([]string{"speed", "charge", "counter", "range", "morale", "attack", "defense", "poison", "burn", "confusion", "seal", "root", "weak", "attack-down", "defense-down"}, id) || v.Turns < 1 || v.Turns > 10 || v.Value < 0 || strings.HasSuffix(id, "-down") && v.Value > 90 || (v.Source != "" && !ids[v.Source] && e.unit(v.Source) == nil) {
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
		if s.Round < 1 || !contains([]string{"ally", "enemy"}, s.Turn) || stage.Goal != "rout" && !ids[stage.Boss] || s.Checkpoint == nil {
			return nil, fail("CorruptSave")
		}
		for id := range deploy {
			if !ids[id] {
				return nil, fail("CorruptSave")
			}
		}
		for _, sp := range stage.Enemies {
			if !ids[sp.Officer] {
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

// OfficerStats is an officer's battle stats with their current class, equipment and deputy,
// for screens outside battle.
func (e *Engine) OfficerStats(id string) (Stats, error) {
	if e.officer(id) == nil {
		return Stats{}, fail("InvalidOfficer")
	}
	return e.stats(&Unit{ID: id, Officer: id, Faction: "ally", Status: map[string]Status{}}), nil
}

// CounterDamage is the most damage the target can return to the actor after c lands,
// or 0 when the target has no counter stance or the actor is out of its reach.
func (e *Engine) CounterDamage(c Command) int {
	u, v := e.unit(c.Actor), e.unit(c.Target)
	if u == nil || v == nil || v.HP <= 0 || v.Faction == u.Faction || v.Status["counter"].Turns == 0 {
		return 0
	}
	s, err := e.skillDef(c)
	if err != nil || s.Kind != "physical" {
		return 0
	}
	basic, _ := e.skillDef(Command{Kind: "attack"})
	if !e.inRange(v, u, basic) {
		return 0
	}
	return int(math.Floor(e.damage(v, u, basic) * 1.5 * .7))
}

// AttackRange is the reach of the unit's basic attack.
func (e *Engine) AttackRange(id string) (lo, hi int) {
	u := e.unit(id)
	if u == nil {
		return 0, 0
	}
	return e.rangeFor(u, content.Skill{Min: 1})
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
	lo, hi := e.rangeFor(&u, content.Skill{Min: 1})
	for _, name := range e.skills(&u) {
		s := e.data.Skills[name]
		if s.Kind == "physical" || s.Kind == "magic" || s.Kind == "status" {
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
