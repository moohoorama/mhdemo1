package core

import (
	"slices"

	"srpg/internal/content"
)

func (e *Engine) playable(c *Character) bool { return c != nil && e.def(c).Playable }
func (e *Engine) isDeputy(id string) bool {
	for _, key := range keys(e.charIDs) {
		if e.char(key).Deputy == id {
			return true
		}
	}
	return false
}

// CanDeputize reports whether the character may name a deputy.
func (e *Engine) CanDeputize(id string) bool {
	o := e.char(id)
	return o != nil && e.fxOf(o, content.FxDeputyAppoint) != 0
}
func (e *Engine) assignDeputy(c Command) error {
	o := e.char(c.Actor)
	if !e.playable(o) || o.Dead || e.isDeputy(o.ID) {
		return fail("InvalidActor")
	}
	if c.Kind == "undeputy" {
		o.Deputy = ""
		return nil
	}
	if err := e.DeputyCheck(o.ID, c.Target); err != nil {
		return err
	}
	d := e.char(c.Target)
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

// releaseRequired frees the officers a stage requires from serving as deputies, so they go into the battle.
func (e *Engine) releaseRequired(s content.Stage) {
	for _, id := range keys(e.charIDs) {
		if o := e.char(id); o.Deputy != "" && contains(s.Required, o.Deputy) {
			e.emit("undeputy", o.ID, o.Deputy, 0)
			o.Deputy = ""
		}
	}
}

// DeputyCheck reports why the target cannot become the officer's deputy now, or nil: the officer
// must be able to name one, and the target be a living officer who is neither the lord, nor one
// the coming battle requires, nor one with or serving as a deputy.
func (e *Engine) DeputyCheck(actor, target string) error {
	if !e.CanDeputize(actor) {
		return fail("DeputyUnavailable")
	}
	d := e.char(target)
	if !e.playable(d) || d.ID == actor || d.Dead || d.Deputy != "" || e.isDeputy(d.ID) {
		return fail("InvalidDeputy")
	}
	if e.def(d).Lord {
		return fail("LordDeputy")
	}
	if e.state.Stage >= 0 && e.state.Stage < len(e.data.Stages) && (e.Deploying() || e.state.Phase == "scenario" && e.data.Node(e.state.Node).Kind == "battle") &&
		contains(e.data.Stages[e.state.Stage].Required, d.ID) {
		return fail("RequiredDeputy")
	}
	return nil
}

// DeputyStats are the officer's stats and six attributes were the target its deputy ("" for none),
// leaving the state as it is.
func (e *Engine) DeputyStats(actor, target string) (Stats, [6]int, error) {
	o := e.char(actor)
	if o == nil || target != "" && e.char(target) == nil {
		return Stats{}, [6]int{}, fail("InvalidCharacter")
	}
	old := o.Deputy
	o.Deputy = target
	defer func() { o.Deputy = old }()
	return e.stats(&Unit{ID: actor, Character: actor, Faction: "ally", Status: map[string]Status{}}), attributes(e.attributes(o)), nil
}

// DeputyTraits are the traits an officer brings to the one it serves as deputy.
func (e *Engine) DeputyTraits(id string) []string {
	d := e.char(id)
	if d == nil {
		return nil
	}
	return append(slices.Clone(e.def(d).Traits), d.Learned...)
}

// attributes are the character's six attributes with the deputy's blended in where the deputy's are higher.
func (e *Engine) attributes(o *Character) [6]float64 {
	R := e.data.Rules
	a := e.def(o).Stats
	var out [6]float64
	for i, n := range a {
		out[i] = float64(n)
	}
	if d := e.char(o.Deputy); d != nil && !d.Dead {
		b := e.def(d).Stats
		charm := float64(a[5]) + e.ownFx(o, content.FxDeputyCharm)
		r := (charm*float64(R.DeputyWeight) + float64(b[5])) / float64(R.DeputyDivisor) // past 1 with 용인술, on purpose
		for i := range a {
			if b[i] > a[i] {
				out[i] = float64(a[i])*(1-r) + float64(b[i])*r
			}
		}
	}
	return out
}
func (e *Engine) promote(u *Unit, c Command, it content.Item) error {
	if c.Actor != c.Target || u.Faction != "ally" {
		return fail("InvalidTarget")
	}
	o := e.char(u.Character)
	cl := e.class(o)
	next := e.data.Promotes(o.Def, o.Class)
	class := c.Option
	if class == "" && len(next) == 1 {
		class = next[0]
	}
	if it.Tier != cl.Tier+1 || !contains(next, class) {
		return fail("InvalidPromotion")
	}
	required := e.data.Rules.PromoteLevels[cl.Tier]
	if o.Level < required {
		return fail("InsufficientLevel")
	}
	if e.state.Inventory[c.Item] < 1 {
		return fail("MissingItem")
	}
	if u.Acted {
		return fail("ActionSpent")
	}
	e.state.Inventory[c.Item]--
	o.Class = class
	st := e.stats(u)
	u.HP = st.MaxHP
	u.MP = st.MaxMP
	e.emit("promotion", u.ID, class, required)
	return nil
}

// pendingDuel is the stage duel still waiting between two living units of opposite sides, wherever they stand.
func (e *Engine) pendingDuel(u, v *Unit) (content.Duel, bool) {
	if u == nil || v == nil || u.HP <= 0 || v.HP <= 0 || u.Faction == v.Faction {
		return content.Duel{}, false
	}
	for _, d := range e.data.Stages[e.state.Stage].Duels {
		pair := duelist(d, u.Character) && d.Enemy == v.Character || d.Enemy == u.Character && duelist(d, v.Character)
		if pair && !e.state.Executed["duel-"+d.ID] && e.duelAlly(d) != nil {
			return d, true
		}
	}
	return content.Duel{}, false
}

// DuelPending reports whether a stage duel is still waiting between the two units, wherever they stand.
func (e *Engine) DuelPending(a, b string) bool {
	_, ok := e.pendingDuel(e.unit(a), e.unit(b))
	return ok
}

// DuelAt is the stage duel the unit would set off by standing at (x, y): a pending duel whose
// other officer is alive, on the other side and next to that cell. Empty when there is none.
func (e *Engine) DuelAt(id string, x, y int) string {
	u := e.unit(id)
	if u == nil || u.HP <= 0 || e.state.Phase != "battle" {
		return ""
	}
	for _, d := range e.data.Stages[e.state.Stage].Duels {
		others := []string{d.Enemy}
		if d.Enemy == u.Character {
			others = append([]string{d.Ally}, d.Supports...)
		} else if !duelist(d, u.Character) {
			continue
		}
		for _, other := range others {
			if v := e.unit(other); v != nil && v.HP > 0 && v.Faction != u.Faction && dist(x, y, v.X, v.Y) <= 1 && !e.state.Executed["duel-"+d.ID] && e.duelAlly(d) != nil {
				return d.ID
			}
		}
	}
	return ""
}

// duelist reports whether the character starts the duel on the ally side by meeting its enemy:
// the duel's ally or one of the supports who join it.
func duelist(d content.Duel, character string) bool {
	return d.Ally == character || slices.Contains(d.Supports, character)
}

// duelAlly is the officer who fights the duel on the ally side, whoever on his side met the enemy:
// the duel's ally, or when he has left the field the first of its supports still on it.
func (e *Engine) duelAlly(d content.Duel) *Unit {
	for _, id := range append([]string{d.Ally}, d.Supports...) {
		if a := e.unit(id); a != nil && a.HP > 0 {
			return a
		}
	}
	return nil
}

// autoDuels holds every pending stage duel whose enemy stands next to its ally or one of its
// supports. It runs at each settle() but waits while the unit that took the command is still in
// the middle of its turn: a move is not a meeting until the action that follows it (wait, attack,
// skill, item) has been carried out. Spawns and the start of the battle set duels off at once.
// One duel can change who stands where, so the stage's duels are looked at again after each.
func (e *Engine) autoDuels() {
	if m := e.unit(e.lastActor); m != nil && m.HP > 0 && !m.Done {
		return
	}
	duels := e.data.Stages[e.state.Stage].Duels
	for again := true; again; {
		again = false
		for _, d := range duels {
			a, b := e.duelAlly(d), e.unit(d.Enemy)
			if a == nil || b == nil || b.HP <= 0 || a.Faction == b.Faction || e.state.Executed["duel-"+d.ID] {
				continue
			}
			for _, id := range append([]string{d.Ally}, d.Supports...) {
				if m := e.unit(id); m != nil && m.HP > 0 && m.Faction == a.Faction && dist(m.X, m.Y, b.X, b.Y) <= 1 {
					e.duel(a, b, d)
					again = true
					break
				}
			}
			if again {
				break
			}
		}
	}
}

// duel resolves a stage duel between the two officers. The event names the ally first and
// the one who moved to meet the other (if the command was one of theirs) as First.
func (e *Engine) duel(ally, enemy *Unit, d content.Duel) {
	first := ""
	if m := e.unit(e.lastActor); m == ally || m == enemy || m != nil && slices.Contains(d.Supports, m.Character) {
		first = m.ID
	}
	e.state.Executed["duel-"+d.ID] = true
	e.events = append(e.events, Event{Kind: "duel", DuelID: d.ID, Outcome: d.Outcome, Actor: ally.ID, Target: enemy.ID, First: first, Text: d.Text + " [" + d.Outcome + "]"})
	outcome := func(q *Unit, result string) {
		if result == "retreat" || result == "death" {
			q.HP = 0
			e.emit(result, q.ID, "", 0)
			if result == "death" {
				o := e.char(q.Character)
				o.Dead = true
				q.MP = 0
				// The surviving deputy returns to the roster.
				o.Deputy = ""
				if e.playable(o) {
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
	if ally.HP == 0 && enemy.HP > 0 {
		e.retreatSweep(enemy, ally, false)
	}
	if enemy.HP == 0 && ally.HP > 0 {
		e.retreatSweep(ally, enemy, false)
	}
	if ally.HP > 0 && !ally.AI {
		e.xp(ally.Character, XPPerLevel)
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
