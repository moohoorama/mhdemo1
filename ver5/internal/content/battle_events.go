package content

import "fmt"

// validateBattleEvents checks a stage's areas and events and returns the characters its events
// spawn on the allies' side.
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
	targets := map[string]bool{}
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
