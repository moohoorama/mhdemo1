package core

import (
	"fmt"
	"math"
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
	for _, id := range keys(e.unitIDs) {
		c := Command{Kind: "duel", Actor: actor, Target: id}
		if _, err := e.duelDefinition(u, c); err == nil {
			out.Commands = append(out.Commands, c)
		}
	}
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
		for _, t := range e.traits(u, true) {
			o := e.officer(u.Officer)
			if !contains(o.Learned, t) && o.Points >= e.data.Traits[t].Cost {
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
	if c.Kind == "duel" {
		d, err := e.duelDefinition(u, c)
		if err != nil {
			return Preview{}, err
		}
		return Preview{Effect: d.Outcome, Hit: 100, Targets: []string{c.Target}}, nil
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
	for _, q := range e.targets(u, v, s) {
		p.Targets = append(p.Targets, q.ID)
		if s.Effect == "heal" {
			p.Healing += min(e.stats(q).MaxHP-q.HP, e.healAmount(u, s))
		}
		if s.Kind == "physical" || s.Kind == "magic" {
			d := e.damage(u, q, s)
			if s.Kind == "physical" {
				d *= 1.5
			}
			p.MaxDamage += int(math.Floor(d))
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
			if !contains([]string{"speed", "charge", "counter", "range", "morale", "attack", "defense", "poison", "burn", "confusion", "seal", "root", "weak"}, id) || v.Turns < 1 || v.Turns > 10 || v.Value < 0 || (v.Source != "" && !ids[v.Source] && e.unit(v.Source) == nil) {
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
		if s.Round < 1 || !contains([]string{"ally", "enemy"}, s.Turn) || !ids[stage.Boss] || s.Checkpoint == nil {
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
