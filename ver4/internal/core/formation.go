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
func (e *Engine) duelDefinition(u *Unit, c Command) (content.Duel, error) {
	v := e.unit(c.Target)
	if u.Faction != "ally" || u.Acted || v == nil || v.HP <= 0 || v.Faction != "enemy" || dist(u.X, u.Y, v.X, v.Y) > 1 {
		return content.Duel{}, fail("InvalidDuel")
	}
	for _, d := range e.data.Stages[e.state.Stage].Duels {
		if d.Ally == u.Officer && d.Enemy == v.Officer && !e.state.Executed["duel-"+d.ID] {
			return d, nil
		}
	}
	return content.Duel{}, fail("InvalidDuel")
}
func (e *Engine) duel(u *Unit, c Command) error {
	d, err := e.duelDefinition(u, c)
	if err != nil {
		return err
	}
	v := e.unit(c.Target)
	e.state.Executed["duel-"+d.ID] = true
	e.events = append(e.events, Event{Kind: "duel", Actor: u.ID, Target: v.ID, Text: d.Text + " [" + d.Outcome + "]"})
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
	outcome(u, d.AllyResult)
	outcome(v, d.EnemyResult)
	u.Acted = true
	if u.HP > 0 {
		e.xp(u.Officer, ExperienceRequired(e.officer(u.Officer).Level))
	}
	return nil
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
