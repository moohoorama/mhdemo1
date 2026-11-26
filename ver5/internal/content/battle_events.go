package content

import "fmt"

// validateBattleEvents checks a stage's areas, events and objectives and returns the characters
// its events spawn on the allies' side and on the enemy side.
func (d *Data) validateBattleEvents(s Stage) (map[string]bool, map[string]bool, error) {
	areas := map[int]bool{}
	for _, a := range s.Areas {
		r := a.Rect
		if a.ID < 1 || areas[a.ID] || a.Name == "" || r.W < 1 || r.H < 1 || r.X+r.W <= 0 || r.Y+r.H <= 0 || r.X >= s.Width || r.Y >= s.Height {
			return nil, nil, fmt.Errorf("area %d", a.ID)
		}
		areas[a.ID] = true
	}
	allies := map[string]bool{}
	enemies := map[string]bool{}
	actors := map[string]bool{}
	targets := map[string]bool{"ally": true, "enemy": true, "player": true} // every unit of a side; player: the ally units the player commands
	for id, c := range d.Characters {
		if c.Playable {
			actors[id] = true
			targets[id] = true
		}
	}
	placed := map[string]bool{} // named officers some spawn puts on the map
	note := func(sp Spawn) {
		actors[sp.Character] = true
		if !d.Characters[sp.Character].Template {
			placed[sp.Character] = true
		}
		if !d.Characters[sp.Character].Template {
			targets[sp.Character] = true
		}
		if sp.Group != "" {
			targets[sp.Group] = true
		}
	}
	for _, sp := range append(append([]Spawn{}, s.Party...), s.Enemies...) {
		if sp.Group == "ally" || sp.Group == "enemy" || sp.Group == "support" || sp.Group == "player" {
			return nil, nil, fmt.Errorf("group name %q is reserved", sp.Group)
		}
	}
	for _, sp := range append(append([]Spawn{}, s.Party...), s.Enemies...) {
		note(sp)
	}
	duels := map[string]bool{}
	for _, duel := range s.Duels {
		duels[duel.ID] = true
	}
	ids := map[string]bool{}
	joins := map[string]bool{} // officers a join action calls in
	for _, event := range s.Events {
		if event.ID == "" || ids[event.ID] || len(event.Do) == 0 || event.Every < 0 || event.Every > 0 && event.Repeat {
			return nil, nil, fmt.Errorf("battle event %s", event.ID)
		}
		ids[event.ID] = true
		for _, a := range event.Do {
			switch a.Kind {
			case "join":
				if event.Every > 0 {
					return nil, nil, fmt.Errorf("battle event %s: join in an event that comes back", event.ID)
				}
				if event.Repeat || len(a.Spawns) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: join without officers", event.ID)
				}
				for _, sp := range a.Spawns {
					c := d.Characters[sp.Character]
					if err := d.ValidateSpawn(sp); err != nil {
						return nil, nil, err
					}
					if !c.Playable || c.Lord || c.Template || sp.AI || sp.Side != "" || sp.Policy != "" || sp.X < 0 || sp.Y < 0 || sp.X >= s.Width || sp.Y >= s.Height || placed[sp.Character] {
						return nil, nil, fmt.Errorf("battle event %s: join %s", event.ID, sp.Character)
					}
					allies[sp.Character] = true
					joins[sp.Character] = true
					if contains(s.Reserve, sp.Character) && (sp.Class != "" || sp.Level != 0 || sp.Equipment != nil || sp.Traits != nil) {
						return nil, nil, fmt.Errorf("battle event %s: join %s keeps the class, level, gear and traits of the preparation", event.ID, sp.Character)
					}
					note(sp)
				}
			case "spawn":
				if event.Repeat || len(a.Spawns) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: spawn without units", event.ID)
				}
				for _, sp := range a.Spawns {
					if err := d.ValidateSpawn(sp); err != nil {
						return nil, nil, err
					}
					enemy := sp.Side == "enemy"
					if sp.Policy == "" || !s.ValidPolicy(sp.Policy, sp.Goal, sp.GoalArea) {
						return nil, nil, fmt.Errorf("battle event spawn %s: policy", sp.Character)
					}
					if sp.Side != "" && sp.Side != "ally" && !enemy || !enemy && !sp.AI || sp.X < 0 || sp.Y < 0 || sp.X >= s.Width || sp.Y >= s.Height || d.Characters[sp.Character].Playable || d.Characters[sp.Character].Lord {
						return nil, nil, fmt.Errorf("battle event spawn %s", sp.Character)
					}
					if sp.Faction != "" {
						if _, ok := d.Factions[sp.Faction]; !ok {
							return nil, nil, fmt.Errorf("battle event faction %s", sp.Faction)
						}
					}
					if !d.Characters[sp.Character].Template && actors[sp.Character] {
						return nil, nil, fmt.Errorf("duplicate event officer %s", sp.Character)
					}
					if enemy {
						enemies[sp.Character] = true
					} else {
						allies[sp.Character] = true
					}
					note(sp)
				}
			case "ai":
				if a.Policy == "" || len(a.Targets) == 0 || !s.ValidPolicy(a.Policy, a.Goal, a.GoalArea) {
					return nil, nil, fmt.Errorf("battle event %s: ai action", event.ID)
				}
			case "terrain":
				if _, ok := d.States[a.State]; !ok && a.State != string(NoLayer) || !areas[a.Area] || a.Turns < 0 {
					return nil, nil, fmt.Errorf("battle event %s: terrain action", event.ID)
				}
			case "buff":
				ailment := contains(BuffAilments, a.Stat)
				if !ailment && !contains(BuffStats, a.Stat) || ailment != (a.Value == 0) || a.Value < -100 || a.Value > 100 || a.Turns < 0 || a.Turns > 99 || len(a.Targets) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: buff action", event.ID)
				}
			case "damage":
				if a.Value < 1 || a.Value > 99 || len(a.Targets) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: damage action", event.ID)
				}
			case "lockmove":
				if a.Value != 0 && a.Value != 1 || len(a.Targets) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: lockmove action", event.ID)
				}
			case "move":
				if len(a.Moves) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: move without units", event.ID)
				}
				for _, mv := range a.Moves {
					if mv.To.X < 0 || mv.To.Y < 0 || mv.To.X >= s.Width || mv.To.Y >= s.Height {
						return nil, nil, fmt.Errorf("battle event %s: move %s off the map", event.ID, mv.Unit)
					}
					if mv.Facing != "" && !in(mv.Facing, "NE", "SE", "SW", "NW") {
						return nil, nil, fmt.Errorf("battle event %s: move %s facing %q", event.ID, mv.Unit, mv.Facing)
					}
				}
			case "cutscene":
				if len(a.Steps) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: empty cutscene", event.ID)
				}
			case "duel":
				if !duels[a.Duel] {
					return nil, nil, fmt.Errorf("battle event %s: no duel %q", event.ID, a.Duel)
				}
			case "say", "remove", "flag", "goals":
			default:
				return nil, nil, fmt.Errorf("battle event %s: action %q", event.ID, a.Kind)
			}
		}
	}
	if err := d.validateReserve(s, joins); err != nil {
		return nil, nil, err
	}
	refs := &battleRefs{targets: targets, areas: areas, duels: duels, events: ids}
	if err := d.validateObjectives(s, refs); err != nil {
		return nil, nil, err
	}
	for _, duel := range s.Duels {
		for _, cond := range duel.If {
			if err := d.validateCond(cond, refs); err != nil {
				return nil, nil, fmt.Errorf("duel %s: %w", duel.ID, err)
			}
		}
	}
	for _, event := range s.Events {
		for _, cond := range event.If {
			if err := d.validateCond(cond, refs); err != nil {
				return nil, nil, fmt.Errorf("battle event %s: %w", event.ID, err)
			}
		}
		for _, a := range event.Do {
			switch a.Kind {
			case "say":
				if len(a.Lines) == 0 {
					return nil, nil, fmt.Errorf("empty battle dialogue %s", event.ID)
				}
				seen := map[string]bool{}
				for _, line := range a.Lines {
					if _, known := d.Characters[line.Actor]; !(actors[line.Actor] || line.Offstage && known) || line.Text == "" || seen[line.Actor] {
						return nil, nil, fmt.Errorf("battle dialogue %s", event.ID)
					}
					seen[line.Actor] = true
				}
			case "move":
				for _, mv := range a.Moves {
					if !refs.targets[mv.Unit] {
						return nil, nil, fmt.Errorf("battle event %s: move %s", event.ID, mv.Unit)
					}
				}
			case "cutscene":
				if err := validateCutscene(s, event.ID, a.Steps, refs); err != nil {
					return nil, nil, err
				}
			case "remove", "ai", "buff", "damage", "lockmove":
				if len(a.Targets) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: %s without targets", event.ID, a.Kind)
				}
				for _, t := range a.Targets {
					if !refs.validTarget(t) {
						return nil, nil, fmt.Errorf("battle event %s: %s %s", event.ID, a.Kind, t)
					}
				}
			case "goals":
				if a.Victory == nil && a.Defeat == nil && a.Sub == nil {
					return nil, nil, fmt.Errorf("battle event %s: goals without goals", event.ID)
				}
				if err := d.validateGoals(s.ID, "victory", a.Victory, true, refs); err != nil {
					return nil, nil, fmt.Errorf("battle event %s: %w", event.ID, err)
				}
				if err := d.validateGoals(s.ID, "defeat", a.Defeat, false, refs); err != nil {
					return nil, nil, fmt.Errorf("battle event %s: %w", event.ID, err)
				}
				if err := d.validateSubs(s.ID, a.Sub, refs); err != nil {
					return nil, nil, fmt.Errorf("battle event %s: %w", event.ID, err)
				}
			case "flag":
				if len(a.Flags) == 0 {
					return nil, nil, fmt.Errorf("battle event %s: flag without flags", event.ID)
				}
				for k := range a.Flags {
					if k == "" {
						return nil, nil, fmt.Errorf("battle event %s: empty flag name", event.ID)
					}
				}
			}
		}
	}
	return allies, enemies, nil
}

// validateReserve checks the officers who wait out of the battle's start: playable, not otherwise bound to the
// sortie or its cells, and called in by one of the stage's join actions.
func (d *Data) validateReserve(s Stage, joins map[string]bool) error {
	seen := map[string]bool{}
	for _, id := range s.Reserve {
		_, recommended := s.Recommend[id]
		if !d.Characters[id].Playable || d.Characters[id].Lord || seen[id] || contains(s.Required, id) || recommended {
			return fmt.Errorf("stage %s: reserve officer %s", s.ID, id)
		}
		if !joins[id] {
			return fmt.Errorf("stage %s: reserve officer %s is called in by no join action", s.ID, id)
		}
		seen[id] = true
	}
	return nil
}

// validateGoals checks a list of win (needText) or loss goals.
func (d *Data) validateGoals(stage, kind string, list []Goal, needText bool, refs *battleRefs) error {
	for _, g := range list {
		if len(g.If) == 0 || needText && g.Text == "" {
			return fmt.Errorf("stage %s: %s goal %q", stage, kind, g.Text)
		}
		for _, cond := range g.If {
			if err := d.validateCond(cond, refs); err != nil {
				return fmt.Errorf("stage %s: %s goal: %w", stage, kind, err)
			}
		}
	}
	return nil
}

// validateObjectives checks who must sortie and the victory, defeat and bonus goals.
func (d *Data) validateObjectives(s Stage, refs *battleRefs) error {
	seen := map[string]bool{}
	for _, id := range s.Required {
		if !d.Characters[id].Playable || seen[id] {
			return fmt.Errorf("stage %s: required officer %s", s.ID, id)
		}
		seen[id] = true
	}
	for _, id := range s.Excluded {
		_, recommended := s.Recommend[id]
		if !d.Characters[id].Playable || seen[id] || contains(s.Reserve, id) || recommended {
			return fmt.Errorf("stage %s: excluded officer %s", s.ID, id)
		}
		seen[id] = true
	}
	if len(s.Required) > s.Max {
		return fmt.Errorf("stage %s: %d required officers for %d places", s.ID, len(s.Required), s.Max)
	}
	if len(s.Victory) == 0 {
		return fmt.Errorf("stage %s: no victory goal", s.ID)
	}
	if err := d.validateGoals(s.ID, "victory", s.Victory, true, refs); err != nil {
		return err
	}
	if err := d.validateGoals(s.ID, "defeat", s.Defeat, false, refs); err != nil {
		return err
	}
	return d.validateSubs(s.ID, s.Sub, refs)
}

// validateSubs checks a list of bonus objectives: the stage's own, or those a goals action puts in place.
func (d *Data) validateSubs(stage string, subs []Objective, refs *battleRefs) error {
	ids := map[string]bool{}
	for _, o := range subs {
		if o.ID == "" || ids[o.ID] || o.Text == "" || len(o.If) == 0 {
			return fmt.Errorf("stage %s: objective %q", stage, o.ID)
		}
		ids[o.ID] = true
		for _, cond := range o.If {
			if err := d.validateCond(cond, refs); err != nil {
				return fmt.Errorf("stage %s: objective %s: %w", stage, o.ID, err)
			}
		}
		for item, n := range o.Reward {
			if !d.Gainable(item) || n < 1 {
				return fmt.Errorf("stage %s: objective %s reward %s", stage, o.ID, item)
			}
		}
	}
	return nil
}

// validateCutscene checks a cutscene's steps: who they name, where they go and how long they wait.
func validateCutscene(s Stage, event string, groups [][]Step, refs *battleRefs) error {
	for _, group := range groups {
		if len(group) == 0 {
			return fmt.Errorf("battle event %s: empty cutscene group", event)
		}
		for _, st := range group {
			ok := true
			switch st.Kind {
			case "move":
				ok = refs.targets[st.Unit] && (st.By != "") != (st.To != nil) && (st.By == "" || refs.targets[st.By]) &&
					(st.To == nil || st.To.X >= 0 && st.To.Y >= 0 && st.To.X < s.Width && st.To.Y < s.Height) &&
					(st.Facing == "" || in(st.Facing, "NE", "SE", "SW", "NW"))
			case "play":
				ok = refs.targets[st.Unit] && contains([]string{"attack", "hit", "block", "exhausted", "idle"}, st.Motion)
			case "face":
				ok = refs.targets[st.Unit] && (st.By == "") == in(st.Facing, "NE", "SE", "SW", "NW") && (st.By == "" || refs.targets[st.By])
			case "say":
				ok = refs.targets[st.Unit] && st.Text != "" && st.Seconds >= 0 && st.Seconds <= 30
			case "leave", "fall":
				ok = refs.targets[st.Unit]
			case "look", "dust":
				ok = st.To != nil && st.To.X >= 0 && st.To.Y >= 0 && st.To.X < s.Width && st.To.Y < s.Height && st.Seconds >= 0 && st.Seconds <= 30
			case "wait":
				ok = st.Seconds > 0 && st.Seconds <= 30
			case "surge":
				ok = st.Text != ""
			default:
				ok = false
			}
			if !ok {
				return fmt.Errorf("battle event %s: cutscene step %q %q", event, st.Kind, st.Unit)
			}
		}
	}
	return nil
}
