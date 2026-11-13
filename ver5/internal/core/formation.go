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
	return append(slices.Clone(innate(d)), d.Learned...)
}

// DeputyBlend is how much of deputy's attributes actor takes in where deputy's are higher, and the charm of actor
// it counts (인덕 included); zero when either is missing.
func (e *Engine) DeputyBlend(actor, deputy string) (r, charm float64) {
	o, d := e.char(actor), e.char(deputy)
	if o == nil || d == nil {
		return 0, 0
	}
	R := e.data.Rules
	charm = float64(e.def(o).Stats[5]) + e.ownFx(o, content.FxDeputyCharm)
	return (charm*float64(R.DeputyWeight) + float64(e.def(d).Stats[5])) / float64(R.DeputyDivisor), charm
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
		r := (charm*float64(R.DeputyWeight) + float64(b[5])) / float64(R.DeputyDivisor) // past 1 with 인덕 (deputy_charm), on purpose
		for i := range a {
			if b[i] > a[i] {
				out[i] = float64(a[i])*(1-r) + float64(b[i])*r
			}
		}
	}
	return out
}

// promotion is the class an 인수 makes officer o by command c (its Option, or the only class there is), or why it
// cannot: the seal must be of the tier after o's class, o's level must reach that promotion and the party must
// carry one. It gives the level the promotion asks for too.
func (e *Engine) promotion(o *Character, c Command, it content.Item) (string, int, error) {
	cl := e.class(o)
	next := e.data.Promotes(o.Def, o.Class)
	class := c.Option
	if class == "" && len(next) == 1 {
		class = next[0]
	}
	if it.Effect != content.ItemPromote || it.Tier != cl.Tier+1 || !contains(next, class) {
		return "", 0, fail("InvalidPromotion")
	}
	required := e.data.Rules.PromoteLevels[cl.Tier]
	if o.Level < required {
		return "", 0, fail("InsufficientLevel")
	}
	if e.state.Inventory[c.Item] < 1 {
		return "", 0, fail("MissingItem")
	}
	return class, required, nil
}

// promoteAway spends an 인수 outside a battle, where officers may change their gear (the walk and a battle's
// preparation, CanLearn's places): Target is the officer promoted, Option the class picked. An officer already
// on the deployment's field gets the new class's full HP and MP. A piece the new class cannot carry (a sword on a
// 장창병, a spear on a 궁기병) goes back to the warehouse with an unequip event. The promotion event names the
// officer as Actor and Text and the new class as Target.
func (e *Engine) promoteAway(c Command) error {
	o, class, required, err := e.awayPromotion(c)
	if err != nil {
		return err
	}
	e.state.Inventory[c.Item]--
	o.Class = class
	if u := e.unit(o.ID); u != nil {
		st := e.stats(u)
		u.HP, u.MP = st.MaxHP, st.MaxMP
	}
	e.events = append(e.events, Event{Kind: "promotion", Actor: o.ID, Target: class, Text: o.ID, Amount: required})
	for _, slot := range keys(o.Equipment) {
		if id := o.Equipment[slot]; !e.carries(o, e.data.Equipment[id]) {
			delete(o.Equipment, slot)
			e.state.Warehouse[id]++
			e.events = append(e.events, Event{Kind: "unequip", Actor: o.ID, Target: id})
		}
	}
	return nil
}

// awayPromotion is the officer promoteAway would promote, into which class at which level, or why it cannot.
func (e *Engine) awayPromotion(c Command) (*Character, string, int, error) {
	it, ok := e.data.Items[c.Item]
	if !ok || it.Effect != content.ItemPromote || !e.preparing() {
		return nil, "", 0, fail("WrongPhase") // off the field only a seal is used
	}
	o := e.char(c.Target)
	if !e.playable(o) || o.Dead || o.Temp {
		return nil, "", 0, fail("InvalidTarget")
	}
	class, required, err := e.promotion(o, c, it)
	return o, class, required, err
}

// PromotionsAway are the classes the 인수 would make the officer now outside a battle (promoteAway), or why it
// cannot be used on them.
func (e *Engine) PromotionsAway(item, id string) ([]string, error) {
	o := e.char(id)
	if o == nil {
		return nil, fail("InvalidTarget")
	}
	var out []string
	var last error = fail("InvalidPromotion")
	for _, class := range e.data.Promotes(o.Def, o.Class) {
		if _, _, _, err := e.awayPromotion(Command{Kind: "item", Target: id, Item: item, Option: class}); err != nil {
			last = err
			continue
		}
		out = append(out, class)
	}
	if len(out) == 0 {
		return nil, last
	}
	return out, nil
}

// pendingDuel is the stage duel still waiting between two living units of opposite sides, wherever they stand.
func (e *Engine) pendingDuel(u, v *Unit) (content.Duel, bool) {
	if u == nil || v == nil || u.HP <= 0 || v.HP <= 0 || u.Faction == v.Faction {
		return content.Duel{}, false
	}
	for _, d := range e.data.Stages[e.state.Stage].Duels {
		pair := e.duelistOn(d, u) != "" && d.Enemy == v.Character || d.Enemy == u.Character && e.duelistOn(d, v) != ""
		if a, _ := e.duelAlly(d); pair && e.duelOpen(d) && a != nil {
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
		} else if e.duelistOn(d, u) == "" {
			continue
		}
		for _, other := range others {
			if v := e.fieldUnit(other); v != nil && v.Faction != u.Faction && dist(x, y, v.X, v.Y) <= 1 && e.duelOpen(d) {
				if a, _ := e.duelAlly(d); a != nil {
					return d.ID
				}
			}
		}
	}
	return ""
}

// duelOpen reports whether the stage duel is still to be fought and its conditions hold.
func (e *Engine) duelOpen(d content.Duel) bool {
	return !e.state.Executed["duel-"+d.ID] && e.holdsIn(d.If, true)
}

// duelist reports whether the character starts the duel on the ally side by meeting its enemy:
// the duel's ally or one of the supports who join it.
func duelist(d content.Duel, character string) bool {
	return d.Ally == character || slices.Contains(d.Supports, character)
}

// duelistOn is the character of unit u who starts the duel on the ally side by meeting its enemy (the duel's
// ally or one of its supports): the unit's own officer, or the deputy riding with it. Empty when neither is.
func (e *Engine) duelistOn(d content.Duel, u *Unit) string {
	if duelist(d, u.Character) {
		return u.Character
	}
	if c := e.char(u.Character); c != nil {
		if q := e.char(c.Deputy); q != nil && !q.Dead && duelist(d, q.ID) {
			return q.ID
		}
	}
	return ""
}

// fieldUnit is the living unit that carries the character onto the field: its own, or, for a deputy, the
// unit of the officer it rides with.
func (e *Engine) fieldUnit(char string) *Unit {
	if u := e.unit(char); u != nil && u.HP > 0 {
		return u
	}
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.HP > 0 && !u.AI {
			if c := e.char(u.Character); c != nil && c.Deputy == char && !e.char(char).Dead {
				return u
			}
		}
	}
	return nil
}

// duelAlly is the officer who fights the duel on the ally side, whoever on his side met the enemy: the duel's
// ally, or when he has left the field the first of its supports still on it; with the unit that carries him,
// his own or, for a deputy, his officer's.
func (e *Engine) duelAlly(d content.Duel) (*Unit, string) {
	for _, id := range append([]string{d.Ally}, d.Supports...) {
		if a := e.fieldUnit(id); a != nil {
			return a, id
		}
	}
	return nil, ""
}

// autoDuels holds every pending stage duel whose enemy stands next to its ally or one of its
// supports. It runs at each settle() but waits while the unit that took the command is still in
// the middle of its turn: a move is not a meeting until the action that follows it (wait, attack,
// skill, item) has been carried out. Spawns and the start of the battle set duels off at once.
// One duel can change who stands where, so the stage's duels are looked at again after each.
func (e *Engine) autoDuels() {
	if e.midAction() {
		return
	}
	duels := e.data.Stages[e.state.Stage].Duels
	for again := true; again; {
		again = false
		for _, d := range duels {
			a, fighter := e.duelAlly(d)
			b := e.unit(d.Enemy)
			if a == nil || b == nil || b.HP <= 0 || a.Faction == b.Faction || !e.duelOpen(d) {
				continue
			}
			for _, id := range append([]string{d.Ally}, d.Supports...) {
				if m := e.fieldUnit(id); m != nil && m.Faction == a.Faction && dist(m.X, m.Y, b.X, b.Y) <= 1 {
					e.duel(a, fighter, b, d)
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

// duel resolves a stage duel between the two officers. The event names the ally's unit first, the
// officer who fights for it as Fighter when that is the deputy riding with it, and the one who moved
// to meet the other (if the command was one of theirs) as First.
func (e *Engine) duel(ally *Unit, fighter string, enemy *Unit, d content.Duel) {
	first := ""
	if m := e.unit(e.lastActor); m == ally || m == enemy || m != nil && slices.Contains(d.Supports, m.Character) {
		first = m.ID
	}
	e.state.Executed["duel-"+d.ID] = true
	if fighter == ally.Character {
		fighter = ""
	}
	e.events = append(e.events, Event{Kind: "duel", DuelID: d.ID, Outcome: d.Outcome, Actor: ally.ID, Target: enemy.ID, First: first, Fighter: fighter, Text: d.Text + " [" + d.Outcome + "]"})
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
		who := ally.Character
		if fighter != "" {
			who = fighter
		}
		e.gainXP(who, XPPerLevel*e.data.Rules.DuelXPPct/100) // the duellist alone, never shared between an officer and its deputy
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
