package core

import "srpg/internal/content"

// eventPasses bounds how often one settle() sweeps the events: an event can make a later (or an
// earlier) one true, but a chain longer than this stops until the next command.
const eventPasses = 8

// Battle events run their actions once their conditions hold; speech is presented by the replay.
func (e *Engine) battleEvents() {
	stage := e.data.Stages[e.state.Stage]
	for pass := 0; pass < eventPasses; pass++ {
		fired := false
		for _, ev := range stage.Events {
			if e.fireEvent(stage, ev) {
				fired = true
			}
		}
		if !fired {
			return
		}
	}
}

// fireEvent runs the event if it is due and reports whether it ran. A repeating event is due each
// time its conditions turn true again; an event whose units do not fit is tried again later.
func (e *Engine) fireEvent(stage content.Stage, ev content.BattleEvent) bool {
	key := "battle-" + stage.ID + "-" + ev.ID
	on := e.holdsIn(ev.If, true)
	if ev.Repeat {
		was := e.state.Executed[key+"~on"]
		e.state.Executed[key+"~on"] = on
		if !on || was {
			return false
		}
		if !e.runEvent(stage, ev) {
			e.state.Executed[key+"~on"] = false
			return false
		}
		return true
	}
	if e.state.Executed[key] || !on || !e.runEvent(stage, ev) {
		return false
	}
	e.state.Executed[key] = true
	return true
}

func (e *Engine) runEvent(stage content.Stage, ev content.BattleEvent) bool {
	var spawns []content.Spawn
	for _, a := range ev.Do {
		if a.Kind == "spawn" {
			spawns = append(spawns, a.Spawns...)
		}
	}
	placements, ok := e.placeSpawns(stage, spawns)
	if !ok {
		return false
	}
	next := 0
	for _, a := range ev.Do {
		switch a.Kind {
		case "spawn":
			for _, sp := range a.Spawns {
				p := placements[next]
				next++
				side := "ally"
				if sp.Side == "enemy" {
					side = "enemy"
				}
				u := e.spawn(sp.Character, sp.Character, side, p.X, p.Y, sp)
				snapshot := copyOf(*u)
				e.events = append(e.events, Event{Kind: "unit-arrival", Actor: u.ID, Unit: &snapshot})
			}
		case "say":
			speech := []content.Speech{}
			for _, line := range a.Lines {
				if u := e.unit(line.Actor); u != nil && u.HP > 0 {
					speech = append(speech, line)
				}
			}
			if len(speech) > 0 {
				e.events = append(e.events, Event{Kind: "speech", Speech: speech})
			}
		case "remove":
			for _, t := range a.Targets {
				for _, u := range e.matching(t) {
					if u.HP > 0 {
						u.HP = 0
						e.emit("retreat", u.ID, "", 0)
					}
				}
			}
		case "flag":
			if e.state.PendingFlags == nil {
				e.state.PendingFlags = map[string]int{}
			}
			for k, v := range a.Flags {
				e.state.PendingFlags[k] = v
			}
		}
	}
	return true
}

// placeSpawns reserves a cell for every spawn before any appears, so a crowded entry cannot
// create a partial wave. Each takes the nearest free cell its class can stand on.
func (e *Engine) placeSpawns(stage content.Stage, spawns []content.Spawn) ([]content.Point, bool) {
	reserved := map[content.Point]bool{}
	placements := make([]content.Point, 0, len(spawns))
	for _, sp := range spawns {
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
			return nil, false
		}
		placements = append(placements, best)
		reserved[best] = true
	}
	return placements, true
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
