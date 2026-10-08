package core

import (
	"srpg/internal/content"
	"strings"
)

func party(id string) bool { return contains([]string{"유비", "관우", "장비", "간옹"}, id) }
func (e *Engine) isDeputy(id string) bool {
	for _, key := range keys(e.officerIDs) {
		if e.officer(key).Deputy == id {
			return true
		}
	}
	return false
}
func (e *Engine) assignDeputy(c Command) error {
	o := e.officer(c.Actor)
	if o == nil || !party(o.ID) || o.Dead || e.isDeputy(o.ID) {
		return fail("InvalidActor")
	}
	if c.Kind == "undeputy" {
		o.Deputy = ""
		return nil
	}
	d := e.officer(c.Target)
	if e.data.Classes[o.Class].Tier < 1 {
		return fail("DeputyUnavailable")
	}
	if d == nil || !party(d.ID) || d.ID == o.ID || d.ID == "유비" || d.Dead || d.Deputy != "" || e.isDeputy(d.ID) {
		return fail("InvalidDeputy")
	}
	// Assignment retires the independent deployment and warehouses equipment.
	for _, item := range d.Equipment {
		e.state.Warehouse[item]++
	}
	d.Equipment = map[string]string{}
	o.Deputy = d.ID
	deploy := []string{}
	for _, id := range e.state.Deployment {
		if id != d.ID {
			deploy = append(deploy, id)
		}
	}
	e.state.Deployment = deploy
	e.emit("deputy", o.ID, d.ID, 0)
	return nil
}
func (e *Engine) attributes(o *Officer) [6]float64 {
	a := e.data.Officers[o.ID].Stats
	var out [6]float64
	for i, n := range a {
		out[i] = float64(n)
	}
	if d := e.officer(o.Deputy); d != nil && !d.Dead {
		b := e.data.Officers[d.ID].Stats
		r := float64(a[5]*2+b[5]) / 300
		for i := range a {
			if b[i] > a[i] {
				out[i] = float64(a[i])*(1-r) + float64(b[i])*r
			}
		}
	}
	return out
}
func (e *Engine) promote(u *Unit, c Command) error {
	if c.Actor != c.Target || u.Faction != "ally" {
		return fail("InvalidTarget")
	}
	o := e.officer(u.Officer)
	target := strings.TrimPrefix(c.Item, "승급:")
	cl := e.data.Classes[o.Class]
	if !contains(cl.Promotes, target) {
		return fail("InvalidPromotion")
	}
	required := 15
	if cl.Tier > 0 {
		required = 30
	}
	if o.Level < required {
		return fail("InsufficientLevel")
	}
	if e.state.Inventory[c.Item] < 1 {
		return fail("MissingItem")
	}
	if u.Acted && !e.has(u, "부호") {
		return fail("ActionSpent")
	}
	e.state.Inventory[c.Item]--
	o.Class = target
	st := e.stats(u)
	u.HP = st.MaxHP
	u.MP = st.MaxMP
	if !e.has(u, "부호") {
		u.Acted = true
	}
	e.emit("promotion", u.ID, target, required)
	return nil
}

// duelFor is the stage duel an attack sets off: the two officers of a pending duel meet
// when either attacks the other at close range with a weapon (영걸전 style). The duel
// replaces the attack.
func (e *Engine) duelFor(u, v *Unit, s content.Skill) (content.Duel, bool) {
	if v == nil || u.HP <= 0 || v.HP <= 0 || s.Kind != "physical" || dist(u.X, u.Y, v.X, v.Y) > 1 {
		return content.Duel{}, false
	}
	for _, d := range e.data.Stages[e.state.Stage].Duels {
		pair := d.Ally == u.Officer && d.Enemy == v.Officer || d.Enemy == u.Officer && d.Ally == v.Officer
		if pair && u.Faction != v.Faction && !e.state.Executed["duel-"+d.ID] {
			return d, true
		}
	}
	return content.Duel{}, false
}

// DuelFor reports whether the attack c would set off a stage duel.
func (e *Engine) DuelFor(c Command) bool {
	u, v := e.unit(c.Actor), e.unit(c.Target)
	if u == nil {
		return false
	}
	s, err := e.skillDef(c)
	if err != nil {
		return false
	}
	_, ok := e.duelFor(u, v, s)
	return ok
}

// duel resolves a stage duel in place of the attacker's action. The event names the
// ally first whichever side attacked.
func (e *Engine) duel(attacker, defender *Unit, d content.Duel) {
	ally, enemy := attacker, defender
	if ally.Faction != "ally" {
		ally, enemy = defender, attacker
	}
	e.state.Executed["duel-"+d.ID] = true
	e.events = append(e.events, Event{Kind: "duel", Actor: ally.ID, Target: enemy.ID, Text: d.Text + " [" + d.Outcome + "]"})
	outcome := func(q *Unit, result string) {
		if result == "retreat" || result == "death" {
			q.HP = 0
			e.emit(result, q.ID, "", 0)
			if result == "death" {
				o := e.officer(q.Officer)
				o.Dead = true
				q.MP = 0
				// The surviving deputy returns to the roster.
				o.Deputy = ""
				if party(o.ID) {
					for _, item := range o.Equipment {
						e.state.Warehouse[item]++
					}
				}
				o.Equipment = map[string]string{}
			}
		}
	}
	outcome(ally, d.AllyResult)
	outcome(enemy, d.EnemyResult)
	for _, down := range []struct {
		status string
		n      int
	}{{"attack-down", d.AttackDown}, {"defense-down", d.DefenseDown}} {
		if d.Outcome != "victory" || down.n == 0 {
			continue
		}
		for _, id := range keys(e.unitIDs) {
			if q := e.unit(id); q.Faction == "enemy" && q.HP > 0 {
				q.Status[down.status] = Status{Turns: 1, Value: min(90, q.Status[down.status].Value+down.n)}
			}
		}
		e.emit(down.status, ally.ID, "", down.n)
	}
	attacker.Acted = true
	attacker.Attacked = true
	if ally.HP > 0 {
		e.xp(ally.Officer, ExperienceRequired(e.officer(ally.Officer).Level))
	}
}

func classReachable(d *content.Data, base, target string) bool {
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(c string) bool {
		if c == target {
			return true
		}
		if seen[c] {
			return false
		}
		seen[c] = true
		for _, next := range d.Classes[c].Promotes {
			if visit(next) {
				return true
			}
		}
		return false
	}
	return visit(base)
}

// Deploying is the start of a battle, before any ally has acted: allies may still be
// placed on the stage's deployment cells.
func (e *Engine) Deploying() bool {
	if e.state.Phase != "battle" || e.state.Round != 1 || e.state.Turn != "ally" {
		return false
	}
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.Faction == "ally" && (u.Moved || u.Acted || u.Done) {
			return false
		}
	}
	return true
}

// place moves an ally to a free deployment cell, or swaps it with the ally standing there.
func (e *Engine) place(u *Unit, c Command) error {
	if u.Faction != "ally" || !e.Deploying() {
		return fail("WrongPhase")
	}
	p := content.Point{X: c.X, Y: c.Y}
	if !containsPoint(e.data.Stages[e.state.Stage].Allies, p) {
		return fail("OutOfRange")
	}
	if q := e.occupied(c.X, c.Y); q != nil {
		if q.Faction != "ally" {
			return fail("Occupied")
		}
		q.X, q.Y = u.X, u.Y
	}
	from := content.Point{X: u.X, Y: u.Y}
	u.X, u.Y = c.X, c.Y
	e.events = append(e.events, Event{Kind: "place", Actor: u.ID, Path: []content.Point{from, p}})
	return nil
}
