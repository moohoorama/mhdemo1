package content

import "fmt"

// validateBattleEvents checks a stage's areas, events and objectives and returns the characters
// its events spawn on the allies' side.
func (d *Data) validateBattleEvents(s Stage) (map[string]bool, error) {
	areas := map[int]bool{}
	for _, a := range s.Areas {
		r := a.Rect
		if a.ID < 1 || areas[a.ID] || a.Name == "" || r.W < 1 || r.H < 1 || r.X+r.W <= 0 || r.Y+r.H <= 0 || r.X >= s.Width || r.Y >= s.Height {
			return nil, fmt.Errorf("area %d", a.ID)
		}
		areas[a.ID] = true
	}
	allies := map[string]bool{}
	actors := map[string]bool{}
	targets := map[string]bool{"ally": true, "enemy": true} // every unit of a side
	for id, c := range d.Characters {
		if c.Playable {
			actors[id] = true
			targets[id] = true
		}
	}
	note := func(sp Spawn) {
		actors[sp.Character] = true
		if !d.Characters[sp.Character].Template {
			targets[sp.Character] = true
		}
		if sp.Group != "" {
			targets[sp.Group] = true
		}
	}
	for _, sp := range append(append([]Spawn{}, s.Party...), s.Enemies...) {
		if sp.Group == "ally" || sp.Group == "enemy" || sp.Group == "support" {
			return nil, fmt.Errorf("group name %q is reserved", sp.Group)
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
	for _, event := range s.Events {
		if event.ID == "" || ids[event.ID] || len(event.Do) == 0 {
			return nil, fmt.Errorf("battle event %s", event.ID)
		}
		ids[event.ID] = true
		for _, a := range event.Do {
			switch a.Kind {
			case "spawn":
				if event.Repeat || len(a.Spawns) == 0 {
					return nil, fmt.Errorf("battle event %s: spawn without units", event.ID)
				}
				for _, sp := range a.Spawns {
					if err := d.ValidateSpawn(sp); err != nil {
						return nil, err
					}
					enemy := sp.Side == "enemy"
					if sp.Side != "" && sp.Side != "ally" && !enemy || !enemy && !sp.AI || sp.X < 0 || sp.Y < 0 || sp.X >= s.Width || sp.Y >= s.Height || d.Characters[sp.Character].Playable || d.Characters[sp.Character].Lord {
						return nil, fmt.Errorf("battle event spawn %s", sp.Character)
					}
					if sp.Faction != "" {
						if _, ok := d.Factions[sp.Faction]; !ok {
							return nil, fmt.Errorf("battle event faction %s", sp.Faction)
						}
					}
					if !d.Characters[sp.Character].Template && actors[sp.Character] {
						return nil, fmt.Errorf("duplicate event officer %s", sp.Character)
					}
					if !enemy {
						allies[sp.Character] = true
					}
					note(sp)
				}
			case "say", "remove", "flag":
			default:
				return nil, fmt.Errorf("battle event %s: action %q", event.ID, a.Kind)
			}
		}
	}
	refs := &battleRefs{targets: targets, areas: areas, duels: duels}
	if err := d.validateObjectives(s, refs); err != nil {
		return nil, err
	}
	for _, event := range s.Events {
		for _, cond := range event.If {
			if err := d.validateCond(cond, refs); err != nil {
				return nil, fmt.Errorf("battle event %s: %w", event.ID, err)
			}
		}
		for _, a := range event.Do {
			switch a.Kind {
			case "say":
				if len(a.Lines) == 0 {
					return nil, fmt.Errorf("empty battle dialogue %s", event.ID)
				}
				seen := map[string]bool{}
				for _, line := range a.Lines {
					if !actors[line.Actor] || line.Text == "" || seen[line.Actor] {
						return nil, fmt.Errorf("battle dialogue %s", event.ID)
					}
					seen[line.Actor] = true
				}
			case "remove":
				if len(a.Targets) == 0 {
					return nil, fmt.Errorf("battle event %s: remove without targets", event.ID)
				}
				for _, t := range a.Targets {
					if !refs.targets[t] {
						return nil, fmt.Errorf("battle event %s: remove %s", event.ID, t)
					}
				}
			case "flag":
				if len(a.Flags) == 0 {
					return nil, fmt.Errorf("battle event %s: flag without flags", event.ID)
				}
				for k := range a.Flags {
					if k == "" {
						return nil, fmt.Errorf("battle event %s: empty flag name", event.ID)
					}
				}
			}
		}
	}
	return allies, nil
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
	if len(s.Required) > s.Max {
		return fmt.Errorf("stage %s: %d required officers for %d places", s.ID, len(s.Required), s.Max)
	}
	if len(s.Victory) == 0 {
		return fmt.Errorf("stage %s: no victory goal", s.ID)
	}
	goals := func(kind string, list []Goal, needText bool) error {
		for _, g := range list {
			if len(g.If) == 0 || needText && g.Text == "" {
				return fmt.Errorf("stage %s: %s goal %q", s.ID, kind, g.Text)
			}
			for _, cond := range g.If {
				if err := d.validateCond(cond, refs); err != nil {
					return fmt.Errorf("stage %s: %s goal: %w", s.ID, kind, err)
				}
			}
		}
		return nil
	}
	if err := goals("victory", s.Victory, true); err != nil {
		return err
	}
	if err := goals("defeat", s.Defeat, false); err != nil {
		return err
	}
	ids := map[string]bool{}
	for _, o := range s.Sub {
		if o.ID == "" || ids[o.ID] || o.Text == "" || len(o.If) == 0 {
			return fmt.Errorf("stage %s: objective %q", s.ID, o.ID)
		}
		ids[o.ID] = true
		for _, cond := range o.If {
			if err := d.validateCond(cond, refs); err != nil {
				return fmt.Errorf("stage %s: objective %s: %w", s.ID, o.ID, err)
			}
		}
		for item, n := range o.Reward {
			if _, ok := d.Items[item]; !ok || n < 1 {
				return fmt.Errorf("stage %s: objective %s reward %s", s.ID, o.ID, item)
			}
		}
	}
	return nil
}
