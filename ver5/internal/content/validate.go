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
	if len(r.HitCurve) < 2 || r.StatDenominator <= 0 || r.PhysicalLevelScale <= 0 || r.MagicLevelScale <= 0 || r.ManaMaxFactor <= 0 || r.ManaRecoveryFactor <= 0 || r.MaxLevel < 1 || len(r.PromoteLevels) == 0 ||
		r.BasicAttack.Kind != KindPhysical || r.ExpGrowth <= 0 || r.DeputyDivisor <= 0 {
		return fmt.Errorf("rules incomplete")
	}
	_, troop := r.RankLead["troop"]
	_, officer := r.RankLead["officer"]
	if r.StageLevelBase > 0 && (r.StageLevelStep < 0 || !troop || !officer) {
		return fmt.Errorf("rules: a stage level needs a step and the troop and officer ranks")
	}
	growth := func(r Rules) bool { return r.DuelXPPct > 0 && r.ExpRatioPower >= 0 && r.ExpMinFactor <= r.ExpMaxFactor }
	if !growth(r) {
		return fmt.Errorf("rules: growth incomplete")
	}
	for _, v := range r.Difficulties {
		if v.ID == "" || v.Name == "" || v.XPPct <= 0 || v.DealtPct < 0 || v.TakenPct < 0 {
			return fmt.Errorf("rules: difficulty %q incomplete", v.ID)
		}
		if v.rules != nil && (!growth(*v.rules) || v.rules.StageLevelStep < 0) {
			return fmt.Errorf("rules: difficulty %q: growth incomplete", v.ID)
		}
	}
	if _, ok := r.DifficultyOf("normal"); !ok && len(r.Difficulties) > 0 {
		return fmt.Errorf("rules: no normal difficulty")
	}
	if len(d.Classes) == 0 || len(d.Characters) == 0 || len(d.Terrain) == 0 || len(d.Grounds) == 0 || len(d.Stages) == 0 || len(d.Nodes) == 0 {
		return fmt.Errorf("missing content definitions")
	}
	if why := d.validateTerrain(); why != "" {
		return fmt.Errorf("%s", why)
	}
	for id, f := range d.Factions {
		if len(f.Colors) != 3 || len(f.OfficerColors) != 0 && len(f.OfficerColors) != 3 {
			return fmt.Errorf("faction %s needs three colors, and three for its officers if any", id)
		}
	}
	for id, t := range d.Traits {
		if t.Name == "" || t.Cost < 0 {
			return fmt.Errorf("trait %s", id)
		}
		if !in(t.Kind, "strategy", "tactic", "talent") || !in(t.Tone, "attack", "heal", "buff", "debuff", "guard", "move", "craft") {
			return fmt.Errorf("trait %s: kind %q or tone %q", id, t.Kind, t.Tone)
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
		if s.Name == "" || !in(s.Kind, KindPhysical, KindMagic, KindStatus, KindBuff, KindHeal, KindMixed) || !in(s.Mode, ModeMelee, ModeRanged, ModeSelf, ModeAny) ||
			!in(s.Shape, ShapeSingle, ShapeCross, ShapeSquare, ShapeLine, ShapeAll) || s.Cost < 0 || s.Hit < 0 || s.Min < 0 || s.Max < 0 ||
			s.Shape == ShapeLine && s.Length < 1 || s.Strikes < 0 || s.Strikes > 2 || s.Reach < 0 || !in(s.Tab, "", TabMartial) || s.Sure && s.Kind != KindStatus || s.Each && (s.Kind != KindPhysical || s.Shape == ShapeSingle) || s.Pierce < 0 || s.Pierce > 1 || s.Pierce > 0 && s.Shape != ShapeSingle || s.MagicCoeff < 0 || s.MagicCoeff > 0 && s.Kind != KindPhysical || s.Effect != "" && !validSkillEffect(s) || s.Kind != KindPhysical && s.Kind != KindMagic && s.Effect == "" {
			return fmt.Errorf("skill %s", id)
		}
		for _, t := range s.Requires {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("skill %s: unknown trait %s", id, t)
			}
		}
	}
	for id, c := range d.Classes {
		if _, ok := d.Terrain[c.Terrain]; !ok || !in(c.Attack, AttackMelee, AttackRanged) || c.Move < 1 || c.Range < 0 || c.HPFactor < 1 ||
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
		for _, t := range c.Pool {
			if d.Traits[t].Cost < 1 {
				return fmt.Errorf("class %s: pool trait %s is not learnable", id, t)
			}
		}
		if !in(c.Prefer, "", KindMagic, KindStatus, KindHeal) {
			return fmt.Errorf("class %s: unknown preference %s", id, c.Prefer)
		}
		if s, ok := d.Classes[c.Shoot]; c.Shoot != "" && (!ok || s.Attack != AttackRanged) {
			return fmt.Errorf("class %s: shoot form %s is not a ranged class", id, c.Shoot)
		}
	}
	for i, g := range r.GearTiers {
		if g.Tier < 1 || g.Chapter < 1 || i > 0 && (g.Chapter <= r.GearTiers[i-1].Chapter || g.Tier < r.GearTiers[i-1].Tier) {
			return fmt.Errorf("gear tier step %d: chapter %d tier %d", i, g.Chapter, g.Tier)
		}
	}
	for id, e := range d.Equipment {
		if e.Name == "" || !in(e.Slot, "weapon", "armor", "accessory") {
			return fmt.Errorf("equipment %s", id)
		}
		if (e.Line == "") != (e.Tier == 0) || e.Tier < 0 || e.Tier > r.maxGearTier() {
			return fmt.Errorf("equipment %s: line %q tier %d", id, e.Line, e.Tier)
		}
		for other, o := range d.Equipment {
			if other != id && e.Line != "" && o.Line == e.Line && (o.Tier == e.Tier || o.Slot != e.Slot || o.Type != e.Type) {
				return fmt.Errorf("equipment %s: line %s clashes with %s", id, e.Line, other)
			}
		}
		for _, t := range e.Traits {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("equipment %s: unknown trait %s", id, t)
			}
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
		if o, ok := d.Characters[e.Owner]; e.Owner != "" && (!ok || !o.Playable || !d.Carries(d.Classes[o.Class], o.Traits, e)) {
			return fmt.Errorf("equipment %s: owner %s", id, e.Owner)
		}
	}
	have := map[string]map[int]bool{}
	for _, e := range d.Equipment {
		if e.Line != "" {
			if have[e.Line] == nil {
				have[e.Line] = map[int]bool{}
			}
			have[e.Line][e.Tier] = true
		}
	}
	for line, tiers := range have {
		for _, g := range r.GearTiers {
			if !tiers[g.Tier] {
				return fmt.Errorf("equipment line %s has no tier %d", line, g.Tier)
			}
		}
	}
	for id, it := range d.Items {
		switch it.Effect {
		case ItemHP, ItemMP:
			if it.Amount < 1 {
				return fmt.Errorf("item %s", id)
			}
		case ItemCure:
		case ItemPromote:
			if it.Tier < 1 || it.Tier > len(r.PromoteLevels) {
				return fmt.Errorf("item %s: tier %d", id, it.Tier)
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
		for t, l := range c.TraitAt {
			if d.Traits[t].Name == "" || l < 1 || l > r.MaxLevel {
				return fmt.Errorf("character %s: trait %s at level %d", id, t, l)
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
			if !ok || !d.Carries(cl, c.Traits, e) {
				return fmt.Errorf("equipment %s on %s", eq, id)
			}
		}
		for key := range c.Look.Sprites {
			class, weapon, hasWeapon := strings.Cut(key, ":")
			if _, ok := d.Classes[class]; !ok || hasWeapon && weapon == "" {
				return fmt.Errorf("character %s: sprite for unknown class %s", id, key)
			}
		}
		for _, l := range c.Look.Later {
			if len(l.If) == 0 {
				return fmt.Errorf("character %s: a later look without a condition", id)
			}
			for _, cond := range l.If {
				if err := d.validateCond(cond, nil); err != nil {
					return fmt.Errorf("character %s: later look: %w", id, err)
				}
			}
		}
		if f := c.Look.Faction; f != "" {
			if _, ok := d.Factions[f]; !ok {
				return fmt.Errorf("character %s: faction %s", id, f)
			}
		}
	}
	if r.TraitPlanStep < 1 {
		return fmt.Errorf("trait plan step %d", r.TraitPlanStep)
	}
	for class, cl := range d.Classes {
		for _, lt := range cl.Plan {
			if _, ok := d.Traits[lt.Trait]; !ok || lt.Level < 1 {
				return fmt.Errorf("trait plan of %s: %s from level %d", class, lt.Trait, lt.Level)
			}
			for _, q := range d.Traits[lt.Trait].Requires {
				for _, pre := range cl.Plan {
					if pre.Trait == q && pre.Level > lt.Level {
						return fmt.Errorf("trait plan of %s: %s at level %d before its prerequisite %s at %d", class, lt.Trait, lt.Level, q, pre.Level)
					}
				}
			}
		}
	}
	for _, stock := range []map[string]int{r.EnemyStock, r.AllyStock} {
		for item, n := range stock {
			if _, ok := d.Items[item]; !ok || n < 0 {
				return fmt.Errorf("npc stock %s", item)
			}
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
	for id, n := range c.Warehouse {
		if _, ok := d.Equipment[id]; !ok || n < 0 {
			return fmt.Errorf("campaign equipment %s", id)
		}
	}
	if c.Money < 0 {
		return fmt.Errorf("campaign money %d", c.Money)
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
	chapters := map[string]bool{}
	for _, ch := range d.Chapters {
		if ch.ID == "" || ch.File == "" || chapters[ch.ID] || len(ch.Nodes) == 0 {
			return fmt.Errorf("chapter %q", ch.ID)
		}
		chapters[ch.ID] = true
	}
	if err := d.checkNodeLoops(nodes); err != nil {
		return err
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
		if n.Card && n.Kind != "narrate" {
			return fmt.Errorf("node %s: only a narrate node is a card", n.ID)
		}
		switch n.Kind {
		case "battle":
			if d.StageIndex(n.Stage) < 0 {
				return fmt.Errorf("node %s stage %s", n.ID, n.Stage)
			}
		case "say":
			if n.Text == "" || n.Speaker == "" {
				return fmt.Errorf("node %s: say needs a speaker and text", n.ID)
			}
			for _, id := range strings.Split(n.Speaker, "·") {
				if !d.Has(id) {
					return fmt.Errorf("node %s speaker %s", n.ID, id)
				}
			}
		case "narrate":
			if n.Text == "" || n.Speaker != "" {
				return fmt.Errorf("node %s: narrate needs text and no speaker", n.ID)
			}
		case "scene":
			if n.Place == "" {
				return fmt.Errorf("node %s: scene needs a place", n.ID)
			}
		case "set":
			if len(n.Flags) == 0 {
				return fmt.Errorf("node %s: set needs flags", n.ID)
			}
			for k := range n.Flags {
				if k == "" {
					return fmt.Errorf("node %s: empty flag name", n.ID)
				}
			}
		case "branch":
			if len(n.Cases) == 0 || n.Else != "" && nodes[n.Else].ID == "" {
				return fmt.Errorf("node %s: branch", n.ID)
			}
			for _, c := range n.Cases {
				if len(c.If) == 0 || nodes[c.Next].ID == "" {
					return fmt.Errorf("node %s: branch case %q", n.ID, c.Next)
				}
				for _, cond := range c.If {
					if err := d.validateCond(cond, nil); err != nil {
						return fmt.Errorf("node %s: %w", n.ID, err)
					}
				}
			}
		case "end":
		case "join", "leave":
			if !d.Characters[n.Character].Playable {
				return fmt.Errorf("node %s joins %s", n.ID, n.Character)
			}
		case "grant":
			if !d.Gainable(n.Item) {
				return fmt.Errorf("node %s grants %s", n.ID, n.Item)
			}
			if _, piece := d.Equipment[n.Item]; n.Character != "" && (!piece || !d.Characters[n.Character].Playable) {
				return fmt.Errorf("node %s puts %s on %s", n.ID, n.Item, n.Character)
			}
		case "choice", "explore", "back":
		default:
			return fmt.Errorf("node %s kind %s", n.ID, n.Kind)
		}
	}
	if err := d.validateExplores(nodes); err != nil {
		return err
	}
	duelIDs, stageIDs := map[string]bool{}, map[string]bool{}
	for _, s := range d.Stages {
		if stageIDs[s.ID] || s.Width < 4 || s.Height < 4 || len(s.Tiles) != s.Height || s.Max < 1 || len(s.Allies) < s.Max || s.TurnLimit < 0 {
			return fmt.Errorf("map %s", s.ID)
		}
		stageIDs[s.ID] = true
		for item, n := range s.Reward {
			if !d.Gainable(item) || n < 1 {
				return fmt.Errorf("stage %s reward %s", s.ID, item)
			}
		}
		if s.Faction != "" {
			if _, ok := d.Factions[s.Faction]; !ok {
				return fmt.Errorf("stage %s faction %s", s.ID, s.Faction)
			}
		}
		seen := map[Point]bool{}
		for _, row := range s.Tiles {
			if len(row) != s.Width*CellWidth {
				return fmt.Errorf("row %s", s.ID)
			}
			for x := 0; x < s.Width; x++ {
				if why := d.ValidCell(Cell{row[x*CellWidth], row[x*CellWidth+1], row[x*CellWidth+2]}); why != "" {
					return fmt.Errorf("stage %s: %s", s.ID, why)
				}
			}
		}
		officers := map[string]bool{}
		for i, side := range [][]Spawn{s.Enemies, s.Party} {
			for _, e := range side {
				if (i == 0 || e.AI) && e.Policy == "" {
					return fmt.Errorf("stage %s spawn %s: an AI-driven unit needs a policy", s.ID, e.Character)
				}
				if err := d.ValidateSpawn(e); err != nil {
					return fmt.Errorf("stage %s: %w", s.ID, err)
				}
				if !d.Characters[e.Character].Template {
					if officers[e.Character] {
						return fmt.Errorf("duplicate officer %s", e.Character)
					}
					officers[e.Character] = true
				}
				p := Point{e.X, e.Y}
				if !d.Has(e.Character) || e.X < 0 || e.Y < 0 || e.X >= s.Width || e.Y >= s.Height || seen[p] || e.Level < 0 || e.Level > r.MaxLevel || !s.ValidPolicy(e.Policy, e.Goal, e.GoalArea) {
					return fmt.Errorf("stage %s spawn %s", s.ID, e.Character)
				}
				if e.Faction != "" {
					if _, ok := d.Factions[e.Faction]; !ok {
						return fmt.Errorf("stage %s spawn %s faction %s", s.ID, e.Character, e.Faction)
					}
				}
				seen[p] = true
			}
		}
		eventAllies, eventEnemies, err := d.validateBattleEvents(s)
		if err != nil {
			return fmt.Errorf("stage %s: %w", s.ID, err)
		}
		for _, event := range s.Duels {
			enemy, ally := eventEnemies[event.Enemy], d.Characters[event.Ally].Playable || eventAllies[event.Ally]
			for _, sp := range s.Enemies {
				enemy = enemy || sp.Character == event.Enemy
			}
			for _, sp := range s.Party {
				ally = ally || sp.Character == event.Ally
			}
			if event.ID == "" || duelIDs[event.ID] || !enemy || !ally || !in(event.Outcome, "victory", "defeat", "draw") ||
				!in(event.AllyResult, "stay", "retreat", "death") || !in(event.EnemyResult, "stay", "retreat", "death") {
				return fmt.Errorf("duel %s", event.ID)
			}
			if event.Outcome == "draw" && (event.AllyResult != "stay" || event.EnemyResult != "stay") {
				return fmt.Errorf("duel %s: a draw leaves both officers in place (stay)", event.ID)
			}
			duelIDs[event.ID] = true
		}
		for _, p := range s.Allies {
			if p.X < 0 || p.Y < 0 || p.X >= s.Width || p.Y >= s.Height || seen[p] {
				return fmt.Errorf("ally spawn")
			}
			seen[p] = true
		}
		for id, p := range s.Recommend {
			if !d.Characters[id].Playable || !seen[p] {
				return fmt.Errorf("stage %s recommend %s %d,%d: a playable officer on one of the Allies cells", s.ID, id, p.X, p.Y)
			}
		}
		if err := validateGroups(s); err != nil {
			return fmt.Errorf("stage %s: %w", s.ID, err)
		}
		for _, p := range s.Lights {
			if p.X < 0 || p.Y < 0 || p.X >= s.Width || p.Y >= s.Height {
				return fmt.Errorf("stage %s light %d,%d", s.ID, p.X, p.Y)
			}
		}
		armies := map[string]bool{}
		allied := map[string]bool{}
		for _, a := range s.Armies {
			if _, ok := d.Factions[a.Faction]; !ok || armies[a.Faction] || a.Name == "" {
				return fmt.Errorf("stage %s army %q", s.ID, a.Faction)
			}
			armies[a.Faction], allied[a.Faction] = true, a.Ally
		}
		// an army is on one side: no enemy wears the color of an army on the player's side
		for _, e := range s.Enemies {
			color := e.Faction
			if color == "" {
				color = d.Characters[e.Character].Look.Faction
			}
			if color == "" {
				color = s.Faction
			}
			if armies[color] && allied[color] {
				return fmt.Errorf("stage %s: enemy %s in the allied army %s", s.ID, e.Character, color)
			}
		}
	}
	return nil
}

// checkNodeLoops rejects a cycle made only of nodes that never wait for the player.
func (d *Data) checkNodeLoops(nodes map[string]Node) error {
	next := func(n Node) []string {
		if n.Kind != "branch" {
			return []string{d.After(n.ID)}
		}
		out := []string{}
		for _, c := range n.Cases {
			out = append(out, c.Next)
		}
		if n.Else != "" {
			return append(out, n.Else)
		}
		return append(out, d.After(n.ID))
	}
	state := map[string]int{} // 1 on the path, 2 done
	var visit func(id string) error
	visit = func(id string) error {
		n := nodes[id]
		if n.ID == "" || Waiting(n.Kind) || state[id] == 2 {
			return nil
		}
		if state[id] == 1 {
			return fmt.Errorf("node %s loops without waiting", id)
		}
		state[id] = 1
		for _, to := range next(n) {
			if err := visit(to); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, id := range sortedKeys(nodes) {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

// validSkillEffect tells whether the skill's effect is one the engine knows: one effect, or for a buff,
// status or mixed skill several stat statuses joined with '+'.
func validSkillEffect(s Skill) bool {
	if SkillEffects[s.Effect] {
		return true
	}
	parts := strings.Split(s.Effect, "+")
	if len(parts) < 2 || !in(s.Kind, KindBuff, KindStatus, KindMixed) {
		return false
	}
	for _, p := range parts {
		if !StatStatuses[p] {
			return false
		}
	}
	return true
}

// validateGroups checks a split sortie's start groups: each names one of the stage's areas holding Allies cells
// no other group holds, and its required officers are officers the stage requires, each in one group only, no
// more than its cells, recommended (if at all) to one of its cells.
func validateGroups(s Stage) error {
	names, owner := map[string]bool{}, map[string]string{}
	taken := map[Point]string{}
	for _, g := range s.Groups {
		if g.Name == "" || names[g.Name] {
			return fmt.Errorf("start group %q", g.Name)
		}
		names[g.Name] = true
		cells := 0
		for _, p := range s.Allies {
			if s.InArea(p, g.Area) {
				if other, ok := taken[p]; ok {
					return fmt.Errorf("start groups %s and %s share the ally cell %d,%d", other, g.Name, p.X, p.Y)
				}
				taken[p] = g.Name
				cells++
			}
		}
		if cells == 0 || len(g.Required) > cells {
			return fmt.Errorf("start group %s: %d ally cells in area %d for %d required officers", g.Name, cells, g.Area, len(g.Required))
		}
		for _, id := range g.Required {
			if !contains(s.Required, id) || owner[id] != "" {
				return fmt.Errorf("start group %s: %s must be required by the stage and by one group only", g.Name, id)
			}
			owner[id] = g.Name
			if p, ok := s.Recommend[id]; ok && !s.InArea(p, g.Area) {
				return fmt.Errorf("start group %s: %s recommended off its cells", g.Name, id)
			}
		}
	}
	return nil
}
