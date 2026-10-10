package content

import (
	"fmt"
	"strings"
)

func in(s string, list ...string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Validate checks that the bundle is complete and consistent, so the engine can trust it.
func (d *Data) Validate() error {
	r := d.Rules
	if len(r.DamageCurve) < 2 || len(r.HitCurve) < 2 || r.StatDenominator <= 0 || r.MaxLevel < 1 || len(r.PromoteLevels) == 0 ||
		r.BasicAttack.Kind != KindPhysical || r.ExpBase <= 0 || r.ExpGrowth <= 0 || r.DeputyDivisor <= 0 {
		return fmt.Errorf("rules incomplete")
	}
	if len(d.Classes) == 0 || len(d.Characters) == 0 || len(d.Terrain) == 0 || len(d.Tiles) == 0 || len(d.Stages) == 0 || len(d.Nodes) == 0 {
		return fmt.Errorf("missing content definitions")
	}
	if _, ok := d.Tiles[r.EdgeTile]; !ok {
		return fmt.Errorf("edge tile %q", r.EdgeTile)
	}
	for group, table := range d.Terrain {
		for tile := range d.Tiles {
			if _, ok := table[tile]; !ok {
				return fmt.Errorf("terrain %s lacks tile %s", group, tile)
			}
		}
		for tile := range table {
			if _, ok := d.Tiles[tile]; !ok {
				return fmt.Errorf("terrain %s: unknown tile %s", group, tile)
			}
		}
	}
	for id, f := range d.Factions {
		if len(f.Colors) != 3 {
			return fmt.Errorf("faction %s needs three colors", id)
		}
	}
	for id, t := range d.Traits {
		if t.Name == "" || t.Cost < 0 {
			return fmt.Errorf("trait %s", id)
		}
		for k := range t.Effects {
			if !TraitEffects[k] && !(strings.Contains(k, ":") && TraitEffects[k[:strings.Index(k, ":")+1]]) {
				return fmt.Errorf("trait %s: unknown effect %s", id, k)
			}
		}
		for _, q := range t.Requires {
			if _, ok := d.Traits[q]; !ok {
				return fmt.Errorf("trait %s: unknown prerequisite %s", id, q)
			}
		}
	}
	for id := range d.Traits {
		seen := map[string]bool{}
		var cycle func(string) bool
		cycle = func(t string) bool {
			if seen[t] {
				return t == id
			}
			seen[t] = true
			for _, q := range d.Traits[t].Requires {
				if q == id || cycle(q) {
					return true
				}
			}
			return false
		}
		if cycle(id) {
			return fmt.Errorf("trait %s: prerequisite cycle", id)
		}
	}
	for id, s := range d.Skills {
		if s.Name == "" || !in(s.Kind, KindPhysical, KindMagic, KindStatus, KindBuff, KindHeal) || !in(s.Mode, ModeMelee, ModeRanged, ModeSelf, ModeAny) ||
			!in(s.Shape, ShapeSingle, ShapeCross, ShapeSquare, ShapeLine, ShapeAll) || s.Cost < 0 || s.Hit < 0 || s.Min < 0 || s.Max < 0 ||
			s.Shape == ShapeLine && s.Length < 1 || s.Effect != "" && !SkillEffects[s.Effect] || s.Kind != KindPhysical && s.Kind != KindMagic && s.Effect == "" {
			return fmt.Errorf("skill %s", id)
		}
		for _, t := range s.Requires {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("skill %s: unknown trait %s", id, t)
			}
		}
		if s.Item != "" {
			if _, ok := d.Items[s.Item]; !ok {
				return fmt.Errorf("skill %s: unknown item %s", id, s.Item)
			}
		}
	}
	granted := map[string]bool{}
	for id, c := range d.Classes {
		if _, ok := d.Terrain[c.Terrain]; !ok || !in(c.Attack, AttackMelee, AttackRanged) || c.Move < 1 || c.Range < 1 || c.HPFactor < 1 ||
			c.Tier < 0 || c.Tier >= len(r.PromoteLevels)+1 || c.Name == "" {
			return fmt.Errorf("class %s", id)
		}
		for _, next := range c.Promotes {
			n, ok := d.Classes[next]
			if !ok || n.Tier != c.Tier+1 {
				return fmt.Errorf("promotion %s -> %s", id, next)
			}
		}
		for _, s := range c.Skills {
			if _, ok := d.Skills[s]; !ok {
				return fmt.Errorf("class %s: unknown skill %s", id, s)
			}
		}
		for _, t := range c.Grants {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("class %s: unknown trait %s", id, t)
			}
			granted[t] = true
		}
		for _, t := range c.Pool {
			if d.Traits[t].Cost < 1 {
				return fmt.Errorf("class %s: pool trait %s is not learnable", id, t)
			}
		}
	}
	for id, e := range d.Equipment {
		if e.Name == "" || !in(e.Slot, "weapon", "armor", "accessory") {
			return fmt.Errorf("equipment %s", id)
		}
		for _, t := range e.Traits {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("equipment %s: unknown trait %s", id, t)
			}
			granted[t] = true
		}
		for _, t := range e.Learn {
			if d.Traits[t].Cost < 1 {
				return fmt.Errorf("equipment %s: learn trait %s is not learnable", id, t)
			}
		}
		for _, s := range e.Skills {
			if _, ok := d.Skills[s]; !ok {
				return fmt.Errorf("equipment %s: unknown skill %s", id, s)
			}
		}
	}
	for t := range granted {
		if d.Traits[t].Cost > 0 {
			return fmt.Errorf("trait %s is learnable and also granted", t)
		}
	}
	for id, it := range d.Items {
		switch it.Effect {
		case ItemHP, ItemMP:
			if it.Amount < 1 {
				return fmt.Errorf("item %s", id)
			}
		case ItemPromote:
			if c, ok := d.Classes[it.Class]; !ok || c.Tier < 1 {
				return fmt.Errorf("item %s: class %s", id, it.Class)
			}
		default:
			return fmt.Errorf("item %s: effect %q", id, it.Effect)
		}
	}
	for id, c := range d.Characters {
		cl, ok := d.Classes[c.Class]
		if !ok || c.Name == "" || c.Level < 1 || c.Level > r.MaxLevel || c.Lord && !c.Playable || c.Template && (c.Lord || c.Playable) {
			return fmt.Errorf("character %s", id)
		}
		for _, s := range c.Stats {
			if s < 1 || s > 100 {
				return fmt.Errorf("stats %s", id)
			}
		}
		for _, t := range append(append([]string{}, c.Traits...), c.Learn...) {
			if d.Traits[t].Name == "" {
				return fmt.Errorf("character %s: unknown trait %s", id, t)
			}
		}
		for _, t := range c.Learn {
			if d.Traits[t].Cost < 1 {
				return fmt.Errorf("character %s: learn trait %s is not learnable", id, t)
			}
		}
		for _, eq := range c.Equipment {
			e, ok := d.Equipment[eq]
			if !ok || (e.Slot == "weapon" && e.Type != cl.Weapon) || (e.Slot == "armor" && e.Type != cl.Armor) {
				return fmt.Errorf("equipment %s on %s", eq, id)
			}
		}
		if f := c.Look.Faction; f != "" {
			if _, ok := d.Factions[f]; !ok {
				return fmt.Errorf("character %s: faction %s", id, f)
			}
		}
	}
	for item, n := range r.EnemyStock {
		if _, ok := d.Items[item]; !ok || n < 0 {
			return fmt.Errorf("enemy stock %s", item)
		}
	}
	c := d.Campaign
	for _, id := range append(append([]string{}, c.Roster...), c.Deployment...) {
		if !d.Characters[id].Playable {
			return fmt.Errorf("campaign: %s is not playable", id)
		}
	}
	for id, n := range c.Inventory {
		if _, ok := d.Items[id]; !ok || n < 0 {
			return fmt.Errorf("campaign item %s", id)
		}
	}
	nodes := map[string]Node{}
	for _, n := range d.Nodes {
		if n.ID == "" || nodes[n.ID].ID != "" {
			return fmt.Errorf("node id %q", n.ID)
		}
		nodes[n.ID] = n
	}
	if nodes[c.Start].ID == "" {
		return fmt.Errorf("campaign start %q", c.Start)
	}
	for _, n := range nodes {
		if n.Next != "" && nodes[n.Next].ID == "" {
			return fmt.Errorf("node %s next %s", n.ID, n.Next)
		}
		for _, v := range n.Choices {
			if nodes[v].ID == "" {
				return fmt.Errorf("node %s choice %s", n.ID, v)
			}
		}
		switch n.Kind {
		case "preparation":
			if d.StageIndex(n.Stage) < 0 {
				return fmt.Errorf("node %s stage %s", n.ID, n.Stage)
			}
		case "join":
			if !d.Characters[n.Character].Playable {
				return fmt.Errorf("node %s joins %s", n.ID, n.Character)
			}
		case "grant":
			if _, ok := d.Items[n.Item]; !ok {
				if _, ok = d.Equipment[n.Item]; !ok {
					return fmt.Errorf("node %s grants %s", n.ID, n.Item)
				}
			}
		case "dialogue", "choice":
		default:
			return fmt.Errorf("node %s kind %s", n.ID, n.Kind)
		}
	}
	duelIDs, stageIDs := map[string]bool{}, map[string]bool{}
	for _, s := range d.Stages {
		if stageIDs[s.ID] || s.Width < 8 || s.Height < 8 || len(s.Tiles) != s.Height || len(s.Allies) < s.Limit {
			return fmt.Errorf("map %s", s.ID)
		}
		stageIDs[s.ID] = true
		if s.Next != "" && nodes[s.Next].ID == "" {
			return fmt.Errorf("stage %s next %s", s.ID, s.Next)
		}
		if s.Faction != "" {
			if _, ok := d.Factions[s.Faction]; !ok {
				return fmt.Errorf("stage %s faction %s", s.ID, s.Faction)
			}
		}
		seen := map[Point]bool{}
		boss := false
		for _, row := range s.Tiles {
			if len(row) != s.Width {
				return fmt.Errorf("row %s", s.ID)
			}
			for _, v := range row {
				if _, ok := d.Tiles[string(v)]; !ok {
					return fmt.Errorf("stage %s: tile %c", s.ID, v)
				}
			}
		}
		for _, side := range [][]Spawn{s.Enemies} {
			for _, e := range side {
				p := Point{e.X, e.Y}
				if !d.Has(e.Character) || e.X < 0 || e.Y < 0 || e.X >= s.Width || e.Y >= s.Height || seen[p] || e.Level < 0 || e.Level > r.MaxLevel {
					return fmt.Errorf("stage %s spawn %s", s.ID, e.Character)
				}
				if e.Faction != "" {
					if _, ok := d.Factions[e.Faction]; !ok {
						return fmt.Errorf("stage %s spawn %s faction %s", s.ID, e.Character, e.Faction)
					}
				}
				seen[p] = true
				boss = boss || e.Character == s.Boss
			}
		}
		for _, event := range s.Duels {
			enemy := false
			for _, sp := range s.Enemies {
				enemy = enemy || sp.Character == event.Enemy
			}
			if event.ID == "" || duelIDs[event.ID] || !enemy || !d.Characters[event.Ally].Playable || event.AttackDown < 0 || event.AttackDown > 90 ||
				event.DefenseDown < 0 || event.DefenseDown > 90 || !in(event.Outcome, "victory", "defeat", "draw") ||
				!in(event.AllyResult, "stay", "retreat", "death") || !in(event.EnemyResult, "stay", "retreat", "death") {
				return fmt.Errorf("duel %s", event.ID)
			}
			duelIDs[event.ID] = true
		}
		if !boss && s.Goal != "rout" {
			return fmt.Errorf("boss %s", s.ID)
		}
		for _, p := range s.Allies {
			if p.X < 0 || p.Y < 0 || p.X >= s.Width || p.Y >= s.Height || seen[p] {
				return fmt.Errorf("ally spawn")
			}
			seen[p] = true
		}
	}
	return nil
}
