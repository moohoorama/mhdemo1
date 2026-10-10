package core

import "srpg/internal/content"

func (e *Engine) Deploying() bool { return e.state.Phase == "deployment" }

func (e *Engine) deploymentCell(id string, p content.Point) bool {
	c := e.char(id)
	st := e.data.Stages[e.state.Stage]
	if c == nil || c.Dead || !contains(e.state.Deployment, id) || p.X < 0 || p.Y < 0 || p.X >= st.Width || p.Y >= st.Height || !containsPoint(st.Allies, p) || e.occupied(p.X, p.Y) != nil {
		return false
	}
	return e.terrain(&Unit{Character: id, Status: map[string]Status{}}, p.X, p.Y).Cost > 0
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
		e.state.Phase = "battle"
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
		e.world.RemoveEntity(e.unitIDs[c.Actor])
		delete(e.unitIDs, c.Actor)
		e.emit("unplace", c.Actor, "", 0)
		return nil
	}
	if u != nil {
		return fail("AlreadyPlaced")
	}
	if !e.deploymentCell(c.Actor, content.Point{X: c.X, Y: c.Y}) {
		return fail("InvalidPlacement")
	}
	// Deployment has no resource-changing commands: every placement starts at full HP, zero MP.
	e.spawn(c.Actor, e.char(c.Actor).Def, "ally", c.X, c.Y, content.Spawn{})
	e.emit("place", c.Actor, "", 0)
	return nil
}

// DeploymentPlan assigns all remaining officers before returning any commands.
// Reassigning an earlier match avoids trapping a unit with fewer usable terrains.
func (e *Engine) DeploymentPlan() ([]Command, error) {
	if !e.Deploying() {
		return nil, fail("WrongPhase")
	}
	cells := e.data.Stages[e.state.Stage].Allies
	owner := map[int]string{}
	var assign func(string, map[int]bool) bool
	assign = func(id string, seen map[int]bool) bool {
		for i, p := range cells {
			if seen[i] || !e.deploymentCell(id, p) {
				continue
			}
			seen[i] = true
			old := owner[i]
			if old == "" || assign(old, seen) {
				owner[i] = id
				return true
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
	return append(out, Command{Kind: "deployment_finish"}), nil
}
