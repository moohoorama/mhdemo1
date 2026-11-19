package core

import (
	"slices"

	"srpg/internal/content"
)

// eventPasses bounds how often one settle() sweeps the events: an event can make a later (or an
// earlier) one true, but a chain longer than this stops until the next command.
const eventPasses = 8

// Battle events run their actions once their conditions hold; speech is presented by the replay. Like a duel they
// wait for the unit in command to finish its action, so a move that is still being ordered sets none off.
func (e *Engine) battleEvents() {
	if e.midAction() {
		return
	}
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
// time its conditions turn true again, an event with Every each time they hold Every rounds after it
// last ran; an event whose units do not fit is tried again later. The rounds it ran in are kept
// (State.Fired) for since: conditions and for the units a save expects.
func (e *Engine) fireEvent(stage content.Stage, ev content.BattleEvent) bool {
	key := "battle-" + stage.ID + "-" + ev.ID
	on := e.holdsIn(ev.If, true)
	switch {
	case ev.Repeat:
		was := e.state.Executed[key+"~on"]
		e.state.Executed[key+"~on"] = on
		if !on || was {
			return false
		}
		if !e.runEvent(stage, ev) {
			e.state.Executed[key+"~on"] = false
			return false
		}
	case ev.Every > 0:
		rounds := e.state.Fired[key]
		if !on || len(rounds) > 0 && e.state.Round < rounds[len(rounds)-1]+ev.Every || !e.runEvent(stage, ev) {
			return false
		}
	default:
		if e.state.Executed[key] || !on || !e.runEvent(stage, ev) {
			return false
		}
	}
	if e.state.Fired == nil {
		e.state.Fired = map[string][]int{}
	}
	e.state.Fired[key] = append(e.state.Fired[key], e.state.Round)
	if !ev.Repeat {
		e.state.Executed[key] = true
	}
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
				e.faceObjective(u, nil)
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
				if !e.reserved(sp.Character) {
					e.welcome(sp.Character, e.stageOrder())
				} else if e.char(sp.Character) == nil {
					continue // a reserve officer who is not in the roster does not come
				}
				u := e.spawn(sp.Character, sp.Character, "ally", p.X, p.Y, sp)
				e.faceObjective(u, nil)
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
					// an ailment shows only where it took hold (status immunity wards it off)
					if u.HP > 0 && (e.applyStatus(u, a.Stat, turns, a.Value) || a.Value != 0) {
						e.events = append(e.events, Event{Kind: "status", Target: u.ID, Text: a.Stat, Amount: a.Value})
					}
				}
			}
		case "damage":
			for _, t := range a.Targets {
				for _, u := range e.matching(t) {
					// the cut never fells a unit
					if n := min(e.stats(u).MaxHP*a.Value/100, u.HP-1); u.HP > 0 && n > 0 {
						u.HP -= n
						e.emit("damage", "", u.ID, n)
					}
				}
			}
		case "move":
			for _, m := range e.moveTogether(a.Moves) {
				e.events = append(e.events, Event{Kind: "move", Actor: m.u.ID, Path: m.path, Facing: m.facing})
			}
		case "cutscene":
			e.playCutscene(a.Steps)
		case "duel":
			for _, d := range stage.Duels {
				if d.ID != a.Duel || e.state.Executed["duel-"+d.ID] {
					continue
				}
				if x, y := e.fieldUnit(d.Ally), e.unit(d.Enemy); x != nil && y != nil && y.HP > 0 && x.Faction != y.Faction {
					e.duel(x, d.Ally, y, d)
				}
			}
		case "say":
			speech := []content.Speech{}
			for _, line := range a.Lines {
				if u := e.unit(line.Actor); line.Offstage || u != nil && u.HP > 0 {
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
						u.Policy, u.Goal, u.GoalArea = a.Policy, copyGoal(a.Goal), a.GoalArea
					}
				}
			}
		case "goals":
			g := e.battleGoals()
			if a.Victory != nil {
				g.Victory = a.Victory
			}
			if a.Defeat != nil {
				g.Defeat = a.Defeat
			}
			if a.Sub != nil {
				g.Sub = a.Sub
			}
			e.state.Goals = &g
			e.events = append(e.events, Event{Kind: "goals"})
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
				if reserved[p] || e.occupied(x, y) != nil || e.data.Move(e.data.Classes[class].Terrain, e.Cell(x, y), e.Boats()).Cost <= 0 {
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

// retreatSweep is the halberd's answer to a retreat: one more attack at the weakest foe in reach, and
// another each time that foe retreats too.
func (e *Engine) retreatSweep(u, fallen *Unit, derived bool) {
	if derived || u == nil || fallen == nil || u.HP <= 0 || u.Faction == fallen.Faction || !e.has(u, content.FxRetreatSweep) {
		return
	}
	e.emit("retreat-sweep", u.ID, fallen.ID, 0)
	for u.HP > 0 {
		var v *Unit
		for _, id := range keys(e.unitIDs) {
			q := e.unit(id)
			if q.HP > 0 && q.Faction != u.Faction && e.inRange(u, q, e.basic()) && (v == nil || q.HP < v.HP) {
				v = q
			}
		}
		if v == nil {
			return
		}
		e.emit("followup-start", u.ID, v.ID, 0)
		e.physical(u, v, e.basic(), Command{Kind: "attack", Actor: u.ID, Target: v.ID})
		e.emit("followup-end", u.ID, v.ID, 0)
		if v.HP > 0 {
			return
		}
	}
}

// changeTerrain puts the action's state on every cell of its area, replacing an earlier one.
func (e *Engine) changeTerrain(stage content.Stage, a content.Action) {
	for _, area := range stage.Areas {
		if area.ID != a.Area {
			continue
		}
		changed := false
		for y := max(0, area.Rect.Y); y < min(stage.Height, area.Rect.Y+area.Rect.H); y++ {
			for x := max(0, area.Rect.X); x < min(stage.Width, area.Rect.X+area.Rect.W); x++ {
				if a.Vacant && e.occupied(x, y) != nil {
					continue
				}
				if e.data.ValidCell(withState(e.Cell(x, y), a.State)) == "" {
					e.setState(stage, x, y, a.State, a.Turns)
					changed = true
				}
			}
		}
		if changed || !a.Vacant {
			e.emit("terrain", "", a.State, a.Area)
		}
	}
}

// withState is the cell with the state put on it.
func withState(c content.Cell, state string) content.Cell {
	c.State = state[0]
	return c
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

// markCell leaves the mark of a critical spell of the element on the cell for Rules.CritMarkTurns rounds:
// embers, a wet patch or rubble. It lies only where the mark's state may lie, and only over a bare cell, another
// mark or what a mark turns into; what an event put there (a fire, a broken bridge, an open gate) stays.
func (e *Engine) markCell(x, y int, element string) {
	R := e.data.Rules
	state, ok := R.CritMarks[element]
	if !ok {
		return
	}
	c := e.Cell(x, y)
	if c.State != content.NoLayer && !e.markState(string(c.State)) || e.data.ValidCell(withState(c, state)) != "" {
		return
	}
	e.setState(e.data.Stages[e.state.Stage], x, y, state, R.CritMarkTurns)
	e.events = append(e.events, Event{Kind: "mark", Target: state, Path: []content.Point{{X: x, Y: y}}})
}

// clearMarks takes the marks of critical spells (and what they turned into) off the 3x3 cells round (x, y), the
// ground back as the stage has it; it reports whether there was any.
func (e *Engine) clearMarks(x, y int) bool {
	stage := e.data.Stages[e.state.Stage]
	any := false
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			cx, cy := x+dx, y+dy
			if cx < 0 || cy < 0 || cx >= stage.Width || cy >= stage.Height {
				continue
			}
			c := e.Cell(cx, cy)
			if c.State == content.NoLayer || !e.markState(string(c.State)) {
				continue
			}
			base := string(stage.Cell(cx, cy).State)
			e.setState(stage, cx, cy, base, 0)
			e.events = append(e.events, Event{Kind: "mark", Target: base, Path: []content.Point{{X: cx, Y: cy}}})
			any = true
		}
	}
	return any
}

// markState reports whether a critical spell may cover the state: a mark, or what one turns into.
func (e *Engine) markState(k string) bool {
	for _, m := range e.data.Rules.CritMarks {
		if k == m || k == e.data.States[m].Then {
			return true
		}
	}
	return false
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

// shifted is where a forced move took a unit: the cells it walked and the way it faces there.
type shifted struct {
	u      *Unit
	path   []content.Point
	facing string
}

// moveTogether carries out forced moves that start at once. The cells the other movers still stand
// on count as free, since they are leaving them, so a line can step forward into itself. Each mover
// ends turned to its move's facing, or the way its last step went when the move names none.
func (e *Engine) moveTogether(moves []content.Move) []shifted {
	type order struct {
		u  *Unit
		mv content.Move
	}
	var orders []order
	leaving := map[*Unit]bool{}
	for _, mv := range moves {
		for _, u := range e.matching(mv.Unit) {
			if u.HP > 0 {
				orders = append(orders, order{u, mv})
				leaving[u] = true
			}
		}
	}
	var out []shifted
	for _, o := range orders {
		delete(leaving, o.u)
		path := e.shift(o.u, o.mv.To, leaving)
		if path == nil {
			continue
		}
		facing := o.mv.Facing
		if facing == "" && len(path) > 1 {
			facing = content.Toward(path[len(path)-2], path[len(path)-1])
		}
		if facing != "" {
			o.u.Facing = facing
		}
		out = append(out, shifted{o.u, path, facing})
	}
	return out
}

// shift puts the unit on the free cell nearest to a point, whatever its move range or the cost of the
// ground between; units in leaving do not hold their cells. It returns the cells it passes through,
// or nil if it cannot stand anywhere.
func (e *Engine) shift(u *Unit, to content.Point, leaving map[*Unit]bool) []content.Point {
	stage := e.data.Stages[e.state.Stage]
	held := map[content.Point]bool{}
	for _, id := range keys(e.unitIDs) {
		if o := e.unit(id); o != u && o.HP > 0 && !leaving[o] {
			held[content.Point{X: o.X, Y: o.Y}] = true
		}
	}
	best, score, near := content.Point{X: -1, Y: -1}, stage.Width+stage.Height+1, 0
	for y := 0; y < stage.Height; y++ {
		for x := 0; x < stage.Width; x++ {
			if held[content.Point{X: x, Y: y}] || e.terrain(u, x, y).Cost <= 0 {
				continue
			}
			if n, m := dist(x, y, to.X, to.Y), dist(x, y, u.X, u.Y); n < score || n == score && m < near {
				best, score, near = content.Point{X: x, Y: y}, n, m
			}
		}
	}
	if best.X < 0 {
		return nil
	}
	path := e.walk(u, best)
	u.X, u.Y = best.X, best.Y
	return path
}

// walk is the shortest way to a cell over ground the unit can walk, other units aside (a forced move
// goes round fences and walls through their gaps), or a straight walk when no way leads there.
func (e *Engine) walk(u *Unit, to content.Point) []content.Point {
	stage := e.data.Stages[e.state.Stage]
	from := content.Point{X: u.X, Y: u.Y}
	prev := map[content.Point]content.Point{from: from}
	for todo := []content.Point{from}; len(todo) > 0 && to != from; todo = todo[1:] {
		p := todo[0]
		if p == to {
			break
		}
		for _, d := range []content.Point{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			q := content.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if _, seen := prev[q]; seen || q.X < 0 || q.Y < 0 || q.X >= stage.Width || q.Y >= stage.Height || e.terrain(u, q.X, q.Y).Cost <= 0 {
				continue
			}
			prev[q] = p
			todo = append(todo, q)
		}
	}
	if _, ok := prev[to]; !ok {
		return straightPath(from, to)
	}
	path := []content.Point{to}
	for p := to; p != from; {
		p = prev[p]
		path = append(path, p)
	}
	slices.Reverse(path)
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

// leave walks a unit to the nearest edge cell it can stand on and takes it off the field as if it had
// fled there, returning the cells it walks.
func (e *Engine) leave(u *Unit) []content.Point {
	stage := e.data.Stages[e.state.Stage]
	best, score := content.Point{X: u.X, Y: u.Y}, stage.Width+stage.Height+1
	for y := 0; y < stage.Height; y++ {
		for x := 0; x < stage.Width; x++ {
			if x > 0 && y > 0 && x < stage.Width-1 && y < stage.Height-1 || e.terrain(u, x, y).Cost <= 0 {
				continue
			}
			if n := dist(x, y, u.X, u.Y); n < score {
				best, score = content.Point{X: x, Y: y}, n
			}
		}
	}
	path := e.walk(u, best)
	u.X, u.Y, u.HP = best.X, best.Y, 0
	if len(path) > 1 {
		u.Facing = content.Toward(path[len(path)-2], path[len(path)-1])
	}
	e.state.Executed["fled-"+stage.ID+"-"+u.ID] = true
	return path
}

// standing is the first unit still on the field that a cutscene target names, or nil.
func (e *Engine) standing(target string) *Unit {
	if target == "" {
		return nil
	}
	for _, u := range e.matching(target) {
		if u.HP > 0 {
			return u
		}
	}
	return nil
}

// playCutscene applies a cutscene's end state (positions and facings; hit points and the rest stay
// as they are) and sends the replay what it came to, step by step. Steps about units that are not on
// the field are left out; a group's moves are carried out together.
func (e *Engine) playCutscene(groups [][]content.Step) {
	var out [][]CutStep
	for _, group := range groups {
		var moves []content.Move
		for _, st := range group {
			if st.Kind != "move" {
				continue
			}
			if st.To != nil {
				moves = append(moves, content.Move{Unit: st.Unit, To: *st.To, Facing: st.Facing})
			} else if by := e.standing(st.By); by != nil {
				moves = append(moves, content.Move{Unit: st.Unit, To: content.Point{X: by.X, Y: by.Y}, Facing: st.Facing})
			}
		}
		moved := map[*Unit]shifted{}
		for _, m := range e.moveTogether(moves) {
			moved[m.u] = m
		}
		for _, st := range group {
			if by := e.standing(st.By); st.Kind == "move" && st.Facing == "" && by != nil {
				for _, u := range e.matching(st.Unit) {
					if m, ok := moved[u]; ok && u != by {
						u.Facing = content.Toward(content.Point{X: u.X, Y: u.Y}, content.Point{X: by.X, Y: by.Y})
						m.facing = u.Facing
						moved[u] = m
					}
				}
			}
		}
		var steps []CutStep
		for _, st := range group {
			for _, u := range e.matching(st.Unit) {
				if u.HP <= 0 && st.Kind != "wait" && st.Kind != "look" {
					continue
				}
				switch st.Kind {
				case "move":
					if m, ok := moved[u]; ok {
						steps = append(steps, CutStep{Kind: "move", Unit: u.ID, Path: m.path, Facing: m.facing})
					}
				case "play":
					steps = append(steps, CutStep{Kind: "play", Unit: u.ID, Motion: st.Motion, Seconds: st.Seconds})
				case "face":
					if by := e.standing(st.By); by != nil && by != u {
						u.Facing = content.Toward(content.Point{X: u.X, Y: u.Y}, content.Point{X: by.X, Y: by.Y})
					} else if st.By == "" {
						u.Facing = st.Facing
					}
					steps = append(steps, CutStep{Kind: "face", Unit: u.ID, Facing: u.Facing})
				case "say":
					steps = append(steps, CutStep{Kind: "say", Unit: u.ID, Text: st.Text})
				case "leave":
					steps = append(steps, CutStep{Kind: "leave", Unit: u.ID, Path: e.leave(u)})
				}
			}
			if st.Kind == "wait" {
				steps = append(steps, CutStep{Kind: "wait", Seconds: st.Seconds})
			}
			if st.Kind == "look" {
				steps = append(steps, CutStep{Kind: "look", Path: []content.Point{*st.To}, Seconds: st.Seconds})
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

// battleGoals are the current battle's win and loss goals and bonus objectives: the stage's, until an
// event changed them.
func (e *Engine) battleGoals() content.Goals {
	if e.state.Goals != nil {
		return *e.state.Goals
	}
	s := e.data.Stages[e.state.Stage]
	return content.Goals{Victory: s.Victory, Defeat: s.Defeat, Sub: s.Sub}
}
