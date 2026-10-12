package content

import "fmt"

func (d *Data) validateBattleEvents(s Stage) (map[string]bool, error) {
	allies := map[string]bool{}
	actors := map[string]bool{}
	for id, c := range d.Characters {
		if c.Playable {
			actors[id] = true
		}
	}
	for _, sp := range append(append([]Spawn{}, s.Party...), s.Enemies...) {
		actors[sp.Character] = true
	}
	ids := map[string]bool{}
	for _, event := range s.Events {
		if event.ID == "" || ids[event.ID] || !in(event.Trigger, "boss-alone", "duel") {
			return nil, fmt.Errorf("battle event %s", event.ID)
		}
		ids[event.ID] = true
		if event.Trigger == "duel" {
			if len(event.Duels) == 0 {
				return nil, fmt.Errorf("battle event %s missing duel", event.ID)
			}
			for _, id := range event.Duels {
				found := false
				for _, duel := range s.Duels {
					found = found || duel.ID == id
				}
				if !found {
					return nil, fmt.Errorf("battle event %s unknown duel %s", event.ID, id)
				}
			}
		}
		for _, sp := range event.Spawns {
			if err := d.ValidateSpawn(sp); err != nil {
				return nil, err
			}
			if !sp.AI || sp.X < 0 || sp.Y < 0 || sp.X >= s.Width || sp.Y >= s.Height || d.Characters[sp.Character].Playable || d.Characters[sp.Character].Lord {
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
			allies[sp.Character] = true
			actors[sp.Character] = true
		}
	}
	for _, event := range s.Events {
		for _, group := range event.Dialogue {
			if len(group) == 0 {
				return nil, fmt.Errorf("empty battle dialogue %s", event.ID)
			}
			seen := map[string]bool{}
			for _, line := range group {
				if !actors[line.Actor] || line.Text == "" || seen[line.Actor] {
					return nil, fmt.Errorf("battle dialogue %s", event.ID)
				}
				seen[line.Actor] = true
			}
		}
	}
	return allies, nil
}
