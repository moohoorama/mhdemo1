package core

import "srpg/internal/content"

// Battle events are one-shot rules events; dialogue is presented by the replay.
func (e *Engine) battleEvents() {
	stage := e.data.Stages[e.state.Stage]
	for _, event := range stage.Events {
		key := "battle-" + stage.ID + "-" + event.ID
		if e.state.Executed[key] {
			continue
		}
		ready := false
		switch event.Trigger {
		case "boss-alone":
			boss := e.unit(stage.Boss)
			ready = boss != nil && boss.HP > 0
			for _, id := range keys(e.unitIDs) {
				u := e.unit(id)
				if u.Faction == "enemy" && u.HP > 0 && u.ID != stage.Boss {
					ready = false
				}
			}
		case "duel":
			for _, id := range event.Duels {
				ready = ready || e.state.Executed["duel-"+id]
			}
		}
		if !ready {
			continue
		}
		// Reserve every position before spawning so a crowded entry cannot create a partial wave.
		reserved := map[content.Point]bool{}
		placements := make([]content.Point, 0, len(event.Spawns))
		for _, sp := range event.Spawns {
			class := sp.Class
			if class == "" {
				class = e.data.Characters[sp.Character].Class
			}
			best := content.Point{X: -1, Y: -1}
			score := stage.Width + stage.Height + 1
			for y := 0; y < stage.Height; y++ {
				for x := 0; x < stage.Width; x++ {
					p := content.Point{X: x, Y: y}
					if reserved[p] || e.occupied(x, y) != nil || e.data.Terrain[e.data.Classes[class].Terrain][string(stage.Tiles[y][x])].Cost <= 0 {
						continue
					}
					if n := dist(x, y, sp.X, sp.Y); n < score {
						best, score = p, n
					}
				}
			}
			if best.X < 0 {
				break
			}
			placements = append(placements, best)
			reserved[best] = true
		}
		if len(placements) != len(event.Spawns) {
			continue
		}
		e.state.Executed[key] = true
		for i, sp := range event.Spawns {
			p := placements[i]
			u := e.spawn(sp.Character, sp.Character, "ally", p.X, p.Y, sp)
			snapshot := copyOf(*u)
			e.events = append(e.events, Event{Kind: "unit-arrival", Actor: u.ID, Unit: &snapshot})
		}
		for _, group := range event.Dialogue {
			speech := []content.Speech{}
			for _, line := range group {
				if u := e.unit(line.Actor); u != nil && u.HP > 0 {
					speech = append(speech, line)
				}
			}
			if len(speech) > 0 {
				e.events = append(e.events, Event{Kind: "speech", Speech: speech})
			}
		}
	}
}

func (e *Engine) retreatSweep(u, fallen *Unit, derived bool) {
	if derived || u == nil || fallen == nil || u.HP <= 0 || u.Faction == fallen.Faction || !e.has(u, content.FxRetreatSweep) {
		return
	}
	e.emit("retreat-sweep", u.ID, fallen.ID, 0)
	for _, id := range keys(e.unitIDs) {
		v := e.unit(id)
		if u.HP <= 0 {
			break
		}
		if v.HP <= 0 || v.Faction == u.Faction || v.ID == fallen.ID || !e.inRange(u, v, e.basic()) {
			continue
		}
		e.emit("followup-start", u.ID, v.ID, 0)
		e.physical(u, v, e.basic(), Command{Kind: "attack", Actor: u.ID, Target: v.ID})
		e.emit("followup-end", u.ID, v.ID, 0)
	}
}
