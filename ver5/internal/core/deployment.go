package core

import "srpg/internal/content"

func (e *Engine) Deploying() bool { return e.state.Phase == "deployment" }

func (e *Engine) deploymentCell(id string, p content.Point) bool {
	return e.deploymentGround(id, p) && e.occupied(p.X, p.Y) == nil
}

// deploymentGround reports whether the officer of the sortie could stand on the ally cell, were it empty.
func (e *Engine) deploymentGround(id string, p content.Point) bool {
	c := e.char(id)
	st := e.data.Stages[e.state.Stage]
	if c == nil || c.Dead || !contains(e.state.Deployment, id) || p.X < 0 || p.Y < 0 || p.X >= st.Width || p.Y >= st.Height || !containsPoint(st.Allies, p) {
		return false
	}
	return e.terrain(&Unit{Character: id, Status: map[string]Status{}}, p.X, p.Y).Cost > 0
}

// canReposition reports whether the placed officer may move to the ally cell, trading cells with an
// officer of the sortie standing there.
func (e *Engine) canReposition(id string, p content.Point) bool {
	u := e.unit(id)
	if u == nil || u.X == p.X && u.Y == p.Y || !e.deploymentGround(id, p) {
		return false
	}
	other := e.occupied(p.X, p.Y)
	return other == nil || other.Faction == "ally" && e.deploymentGround(other.ID, content.Point{X: u.X, Y: u.Y})
}

func (e *Engine) deploymentReady() bool {
	for _, id := range e.state.Deployment {
		u := e.unit(id)
		if u == nil || u.HP <= 0 || u.Faction != "ally" {
			return false
		}
	}
	return len(e.state.Deployment) > 0
}

func (e *Engine) deployCommand(c Command) error {
	if !e.Deploying() {
		return fail("WrongPhase")
	}
	if c.Kind == "deployment_finish" {
		if !e.deploymentReady() {
			return fail("DeploymentIncomplete")
		}
		e.keepSortie()
		e.state.Phase = "battle"
		e.turnStarted = "ally"
		e.emit("deployment_finish", "", "", 0)
		return nil
	}
	if !contains(e.state.Deployment, c.Actor) || e.char(c.Actor) == nil || e.char(c.Actor).Dead {
		return fail("InvalidActor")
	}
	u := e.unit(c.Actor)
	if c.Kind == "unplace" {
		if u == nil {
			return fail("InvalidActor")
		}
		e.unplace(c.Actor)
		return nil
	}
	if c.Kind == "reposition" {
		return e.reposition(c)
	}
	if u != nil {
		return fail("AlreadyPlaced")
	}
	if !e.deploymentCell(c.Actor, content.Point{X: c.X, Y: c.Y}) {
		return fail("InvalidPlacement")
	}
	e.place(c.Actor, c.X, c.Y)
	return nil
}

// place stands an officer of the sortie on a cell. Deployment has no resource-changing commands:
// every placement starts at full HP, zero MP.
func (e *Engine) place(id string, x, y int) {
	e.faceObjective(e.spawn(id, e.char(id).Def, "ally", x, y, content.Spawn{}), e.data.Stages[e.state.Stage].Allies)
	e.emit("place", id, "", 0)
}
func (e *Engine) unplace(id string) {
	e.world.RemoveEntity(e.unitIDs[id])
	delete(e.unitIDs, id)
	e.emit("unplace", id, "", 0)
}

// reposition moves a placed officer to another ally cell; an officer of the sortie standing there
// takes the cell the first one left, so both must be able to stand where they go.
func (e *Engine) reposition(c Command) error {
	p := content.Point{X: c.X, Y: c.Y}
	if !e.canReposition(c.Actor, p) {
		return fail("InvalidPlacement")
	}
	u := e.unit(c.Actor)
	from := content.Point{X: u.X, Y: u.Y}
	other := e.occupied(p.X, p.Y)
	e.unplace(c.Actor)
	if other != nil {
		id := other.ID
		e.unplace(id)
		e.place(id, from.X, from.Y)
	}
	e.place(c.Actor, p.X, p.Y)
	return nil
}

// redeploy runs a preparation command while the officers stand on the field: the sortie, equipment
// and deputies change there, those who leave the sortie leave the field, those who join it stand on
// the first free ally cells, and those whose gear or deputy changed stand again where they were.
func (e *Engine) redeploy(c Command) error {
	if c.Kind == "start" {
		return fail("WrongPhase")
	}
	cells := map[string]content.Point{}
	for _, id := range e.state.Deployment {
		if u := e.unit(id); u != nil {
			cells[id] = content.Point{X: u.X, Y: u.Y}
		}
	}
	if err := e.prepare(c); err != nil {
		return err
	}
	for id, p := range cells {
		e.unplace(id)
		if contains(e.state.Deployment, id) {
			e.place(id, p.X, p.Y)
		}
	}
	return e.placeRest()
}

// placeRest stands every officer of the sortie who is not on the field yet, keeping the cells' order.
func (e *Engine) placeRest() error {
	plan, err := e.placements()
	if err != nil {
		return err
	}
	for _, c := range plan {
		e.place(c.Actor, c.X, c.Y)
	}
	return nil
}

// keepSortie writes the sortie, gear and deputies the battle begins with into the checkpoint, so
// a retry after defeat comes back to the field as the player last left it.
func (e *Engine) keepSortie() {
	cp := e.state.Checkpoint
	if cp == nil {
		return
	}
	cp.Characters = nil
	for _, id := range keys(e.charIDs) {
		if o := e.char(id); !o.Temp {
			cp.Characters = append(cp.Characters, copyOf(*o))
		}
	}
	cp.Warehouse = copyOf(e.state.Warehouse)
	cp.Deployment = append([]string{}, e.state.Deployment...)
}

// DeploymentPlan assigns all remaining officers before returning any commands.
// Reassigning an earlier match avoids trapping a unit with fewer usable terrains. A battle already
// under way (a forced sortie places its officers itself) has nothing left to place.
func (e *Engine) DeploymentPlan() ([]Command, error) {
	if e.state.Phase == "battle" {
		return nil, nil
	}
	if !e.Deploying() {
		return nil, fail("WrongPhase")
	}
	out, err := e.placements()
	if err != nil {
		return nil, err
	}
	return append(out, Command{Kind: "deployment_finish"}), nil
}

// placements are the place commands that stand every officer not yet on the field.
func (e *Engine) placements() ([]Command, error) {
	cells := e.data.Stages[e.state.Stage].Allies
	owner := map[int]string{}
	var assign func(string, map[int]bool) bool
	assign = func(id string, seen map[int]bool) bool {
		for _, free := range []bool{true, false} { // a free cell first, so the officers keep the cells' order
			for i, p := range cells {
				if seen[i] || (owner[i] == "") != free || !e.deploymentCell(id, p) {
					continue
				}
				seen[i] = true
				old := owner[i]
				if old == "" || assign(old, seen) {
					owner[i] = id
					return true
				}
			}
		}
		return false
	}
	for _, id := range e.state.Deployment {
		if e.unit(id) == nil && !assign(id, map[int]bool{}) {
			return nil, fail("NoDeploymentSpace")
		}
	}
	out := []Command{}
	for _, id := range e.state.Deployment {
		if e.unit(id) != nil {
			continue
		}
		for i, p := range cells {
			if owner[i] == id {
				out = append(out, Command{Kind: "place", Actor: id, X: p.X, Y: p.Y})
				break
			}
		}
	}
	return out, nil
}
