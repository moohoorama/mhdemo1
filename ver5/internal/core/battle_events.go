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
		if a.Kind == "spawn" || a.Kind == "join" {
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
		case "join":
			for _, sp := range a.Spawns {
				p := placements[next]
				next++
				if o := e.char(sp.Character); e.unit(sp.Character) != nil || o != nil && o.Dead {
					continue // already on the map, or fallen
				}
				u := e.spawn(sp.Character, sp.Character, "ally", p.X, p.Y, sp)
				snapshot := copyOf(*u)
				e.events = append(e.events, Event{Kind: "unit-arrival", Actor: u.ID, Unit: &snapshot})
			}
		case "buff":
			turns := a.Turns
			if turns == 0 {
				turns = 99 // for the rest of the battle
			}
			for _, t := range a.Targets {
				for _, u := range e.matching(t) {
					if u.HP > 0 {
						e.applyStatus(u, a.Stat, turns, a.Value)
					}
				}
			}
		case "move":
			for _, mv := range a.Moves {
				for _, u := range e.matching(mv.Unit) {
					if path := e.shift(u, mv.To); path != nil {
						e.events = append(e.events, Event{Kind: "move", Actor: u.ID, Path: path})
					}
				}
			}
		case "cutscene":
			e.playCutscene(a.Steps)
		case "duel":
			for _, d := range stage.Duels {
				if d.ID != a.Duel || e.state.Executed["duel-"+d.ID] {
					continue
				}
				if x, y := e.unit(d.Ally), e.unit(d.Enemy); x != nil && y != nil && x.HP > 0 && y.HP > 0 && x.Faction != y.Faction {
					e.duel(x, y, d)
				}
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
		case "terrain":
			e.changeTerrain(stage, a)
		case "ai":
			for _, t := range a.Targets {
				for _, u := range e.matching(t) {
					if u.HP > 0 {
						u.Policy, u.Goal = a.Policy, copyGoal(a.Goal)
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
				if reserved[p] || e.occupied(x, y) != nil || e.data.Move(e.data.Classes[class].Terrain, e.Cell(x, y)).Cost <= 0 {
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

// changeTerrain puts the action's state on every cell of its area, replacing an earlier one.
func (e *Engine) changeTerrain(stage content.Stage, a content.Action) {
	for _, area := range stage.Areas {
		if area.ID != a.Area {
			continue
		}
		for y := max(0, area.Rect.Y); y < min(stage.Height, area.Rect.Y+area.Rect.H); y++ {
			for x := max(0, area.Rect.X); x < min(stage.Width, area.Rect.X+area.Rect.W); x++ {
				e.setState(stage, x, y, a.State, a.Turns)
			}
		}
		e.emit("terrain", "", a.State, a.Area)
	}
}

// setState gives a cell a state for turns rounds (0: for good). A state the stage gave the cell
// itself needs no entry.
func (e *Engine) setState(stage content.Stage, x, y int, state string, turns int) {
	kept := e.state.TileStates[:0:0]
	for _, o := range e.state.TileStates {
		if o.X != x || o.Y != y {
			kept = append(kept, o)
		}
	}
	e.state.TileStates = kept
	if string(stage.Cell(x, y).State) != state || turns > 0 {
		e.state.TileStates = append(e.state.TileStates, TileState{X: x, Y: y, State: state, Turns: turns})
	}
}

// ageTerrain counts a round off every temporary state; the ones that run out turn into what the data says.
func (e *Engine) ageTerrain() {
	stage := e.data.Stages[e.state.Stage]
	var done []TileState
	kept := e.state.TileStates[:0:0]
	for _, o := range e.state.TileStates {
		if o.Turns > 0 {
			if o.Turns--; o.Turns == 0 {
				done = append(done, o)
				continue
			}
		}
		kept = append(kept, o)
	}
	e.state.TileStates = kept
	for _, o := range done {
		next := e.data.States[o.State].Then
		if next == "" {
			next = string(content.NoLayer)
		}
		e.setState(stage, o.X, o.Y, next, 0)
	}
}

// shift puts the unit on the free cell nearest to a point, whatever its move range or the cost of the
// ground between; it returns the cells it passes through (a straight walk), or nil if it cannot stand anywhere.
func (e *Engine) shift(u *Unit, to content.Point) []content.Point {
	stage := e.data.Stages[e.state.Stage]
	best, score := content.Point{X: -1, Y: -1}, stage.Width+stage.Height+1
	for y := 0; y < stage.Height; y++ {
		for x := 0; x < stage.Width; x++ {
			if o := e.occupied(x, y); (o != nil && o != u) || e.terrain(u, x, y).Cost <= 0 {
				continue
			}
			if n := dist(x, y, to.X, to.Y); n < score {
				best, score = content.Point{X: x, Y: y}, n
			}
		}
	}
	if best.X < 0 {
		return nil
	}
	path := straightPath(content.Point{X: u.X, Y: u.Y}, best)
	u.X, u.Y = best.X, best.Y
	return path
}

// straightPath lists the cells from one point to another, along x first and then y, both ends included.
func straightPath(from, to content.Point) []content.Point {
	path := []content.Point{from}
	for from.X != to.X {
		from.X += sign(to.X - from.X)
		path = append(path, from)
	}
	for from.Y != to.Y {
		from.Y += sign(to.Y - from.Y)
		path = append(path, from)
	}
	return path
}

// playCutscene applies a cutscene's end state (positions and facings; hit points and the rest stay
// as they are) and sends the replay what it came to, step by step. Steps about units that are not on
// the field are left out.
func (e *Engine) playCutscene(groups [][]content.Step) {
	var out [][]CutStep
	for _, group := range groups {
		var steps []CutStep
		for _, st := range group {
			for _, u := range e.matching(st.Unit) {
				if u.HP <= 0 && st.Kind != "wait" {
					continue
				}
				switch st.Kind {
				case "move":
					if path := e.shift(u, *st.To); path != nil {
						steps = append(steps, CutStep{Kind: "move", Unit: u.ID, Path: path})
					}
				case "play":
					steps = append(steps, CutStep{Kind: "play", Unit: u.ID, Motion: st.Motion, Seconds: st.Seconds})
				case "face":
					u.Facing = st.Facing
					steps = append(steps, CutStep{Kind: "face", Unit: u.ID, Facing: st.Facing})
				case "say":
					steps = append(steps, CutStep{Kind: "say", Unit: u.ID, Text: st.Text})
				}
			}
			if st.Kind == "wait" {
				steps = append(steps, CutStep{Kind: "wait", Seconds: st.Seconds})
			}
		}
		if len(steps) > 0 {
			out = append(out, steps)
		}
	}
	if len(out) > 0 {
		e.events = append(e.events, Event{Kind: "cutscene", Cutscene: out})
	}
}
