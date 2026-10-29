package core

import (
	"math"
	"slices"

	"srpg/internal/content"
)

// might is the character's base might, the stat the might-comparing traits read.
func (e *Engine) might(u *Unit) int      { return e.unitDef(u).Stats[1] }
func (e *Engine) ratio(a, b int) float64 { return float64(max(1, a)) / float64(max(1, b)) }

// curve interpolates linearly between points sorted by x and holds the end values outside them.
func curve(x float64, pts content.Curve) float64 {
	if x <= pts[0][0] {
		return pts[0][1]
	}
	for i := 1; i < len(pts); i++ {
		if p, q := pts[i-1], pts[i]; x <= q[0] {
			return p[1] + (x-p[0])*(q[1]-p[1])/(q[0]-p[0])
		}
	}
	return pts[len(pts)-1][1]
}
func chance(c content.Chance, r float64) float64 { return clamp(c.Base+c.Per*(r-1), 0, c.Max) }

// damage is the unrounded damage of one blow before critical, share and armor-like reductions.
func (e *Engine) damage(u, v *Unit, s content.Skill) float64 {
	R := e.data.Rules
	a, b := e.stats(u), e.stats(v)
	var base float64
	level := float64(e.char(u.Character).Level + R.StatLevelBase)
	if s.Kind == content.KindMagic {
		base = math.Max(1, float64(a.Mind-b.Mind)+level*R.MagicLevelScale)
	} else {
		tf := e.terrain(v, v.X, v.Y).Factor
		if e.has(u, content.FxIgnoreTerrain) {
			tf = 1
		}
		atk, def := float64(a.Attack)*e.terrain(u, u.X, u.Y).Factor, float64(b.Defense)*tf
		base = math.Max(1, atk-def+level*R.PhysicalLevelScale)
	}
	out := base * s.Coeff
	if s.MagicCoeff > 0 {
		out += math.Max(1, float64(a.Mind-b.Mind)+level*R.MagicLevelScale) * s.MagicCoeff
	}
	if p := e.elementTerrain(v, s.Element); p != 0 && s.Kind == content.KindMagic {
		out *= 1 + float64(p)/100
	}
	if p := e.state.DamagePct; p > 0 && p != 100 {
		out = math.Max(1, out*float64(p)/100)
	}
	return out
}

// elementTerrain is the percent a spell of the element gains on the target's cell: fire in grass and woods,
// water in fords and on the water, earth on badland (Rules.ElementTerrain).
func (e *Engine) elementTerrain(v *Unit, element string) int {
	bonus := e.data.Rules.ElementTerrain[element]
	if bonus == nil {
		return 0
	}
	c := e.Cell(v.X, v.Y)
	if p, ok := bonus[string(c.Structure)]; ok {
		return p
	}
	return bonus[string(c.Ground)]
}

// rangedAttack tells whether the blow comes from a distance: by an archer's weapon or from beyond the eight cells around the target.
func (e *Engine) rangedAttack(u, v *Unit, s content.Skill) bool {
	if s.Mode == content.ModeRanged {
		return true
	}
	return s.Mode == content.ModeAny && (e.unitClass(u).Attack == content.AttackRanged || !adjacent(u.X, u.Y, v.X, v.Y))
}
func (e *Engine) hitChance(u, v *Unit, s content.Skill) float64 {
	R := e.data.Rules
	h := float64(s.Hit)
	switch s.Kind {
	case content.KindPhysical:
		if v.Status["guard"].Turns > 0 && !e.rangedAttack(u, v, s) {
			return 0
		}
		if e.has(v, content.FxRangedImmune) && e.rangedAttack(u, v, s) {
			return 0
		}
		h += curve(e.ratio(e.attackStats(u, s).Agility, e.stats(v).Agility), R.HitCurve) - float64(R.BasicAttack.Hit) - e.fx(v, content.FxEvade) + e.fx(u, content.FxAccuracy)
		return clamp(h, R.HitFloor, R.HitCeil)
	case content.KindMagic, content.KindStatus:
		if e.has(v, content.FxDispel) && e.stats(v).Mind > e.stats(u).Mind {
			return 0
		}
		if s.Kind == content.KindStatus && e.has(v, content.FxStatusImmune) {
			return 0
		}
		if e.has(u, content.FxSpellSure) && e.stats(u).Mind > e.stats(v).Mind || e.fx(u, content.FxSure+s.Element) > 0 {
			return 100
		}
		h += float64(e.unitDef(u).Stats[4]-e.unitDef(v).Stats[4]) * .5
		return clamp(h, R.SpellFloor, R.SpellCeil)
	}
	if s.Mode != content.ModeSelf && s.Hit < 100 {
		h += float64(e.stats(u).Mind)/100 + float64(e.unitDef(u).Stats[4])*.05 + 1
	}
	return clamp(h, R.SpellFloor, R.SpellCeil)
}

// critForced tells whether the blow is a certain critical.
func (e *Engine) critForced(u, v *Unit, s content.Skill) bool {
	if s.Kind == content.KindMagic {
		return e.has(u, content.FxSpellCrit) || e.fx(u, content.FxCritElement+s.Element) > 0
	}
	return e.has(u, content.FxCritAlways) || e.has(u, content.FxCritWeaker) && e.might(u) > e.might(v) ||
		e.has(u, content.FxCritAmbush) && e.CellProps(u.X, u.Y).Ambush
}

// critChance is the chance in percent that a blow lands critically, by the morale of both sides; 100 when forced.
func (e *Engine) critChance(u, v *Unit, s content.Skill) float64 {
	if e.critForced(u, v, s) {
		return 100
	}
	return chance(e.data.Rules.Crit, e.ratio(e.attackStats(u, s).Morale, e.stats(v).Morale))
}

// doubleChance is the chance in percent that a physical skill strikes the target twice.
func (e *Engine) doubleChance(u, v *Unit, s content.Skill) float64 {
	if e.has(u, content.FxDoubleAlways) {
		return 100
	}
	return chance(e.data.Rules.Double, e.ratio(e.attackStats(u, s).Agility, e.stats(v).Agility))
}

// attackStats are the attacker's stats for a blow of the skill: an aimed shot raises agility and morale.
func (e *Engine) attackStats(u *Unit, s content.Skill) Stats {
	st := e.stats(u)
	if s.Effect == "aim" {
		p := e.data.Rules.BuffPercent
		st.Agility, st.Morale = percent(st.Agility, p), percent(st.Morale, p)
	}
	return st
}

// applyStatus puts a status on v. A stat percentage replaces an opposite one and otherwise keeps the stronger.
func (e *Engine) applyStatus(v *Unit, id string, turns, value int) {
	if content.Harmful[id] && e.has(v, content.FxStatusImmune) {
		return
	}
	old := v.Status[id]
	if old.Turns > 0 && (value >= 0) == (old.Value >= 0) && abs(old.Value) > abs(value) {
		value = old.Value
	}
	v.Status[id] = Status{Turns: turns, Value: value, Source: old.Source}
}
func (e *Engine) hurt(v *Unit, n int, u *Unit) int {
	n = min(v.HP, max(0, n))
	if n >= v.HP && n > 0 && !v.Spared && e.has(v, content.FxLastStand) {
		n = v.HP - 1
		v.Spared = true
	}
	v.HP -= n
	e.emit("damage", u.ID, v.ID, n)
	if v.HP == 0 && n > 0 {
		e.emit("retreat", v.ID, "", 0)
	}
	return n
}

// blow is how one strike is made: its share of full damage, and whether it was set off by another attack
// (a counter, assist, riposte or reflection), which never sets off another.
type blow struct {
	share   float64
	derived bool
}

// strike is one blow: hit, critical and damage, with a missed blow still landing a share of it for 급소-like traits.
// It reports the damage dealt, whether the blow hit and whether it was critical.
func (e *Engine) strike(u, v *Unit, s content.Skill, b blow) (int, bool, bool) {
	if u.HP <= 0 || v.HP <= 0 {
		return 0, false, false
	}
	R := e.data.Rules
	phys := s.Kind == content.KindPhysical
	d := e.damage(u, v, s) * b.share
	hit := float64(e.rand(100)) < e.hitChance(u, v, s)
	crit := false
	if hit {
		crit = e.critForced(u, v, s)
		if !crit {
			crit = float64(e.rand(100)) < e.critChance(u, v, s)
		}
		if crit && e.has(v, content.FxCritImmune) {
			hit, crit = false, false
		}
	}
	e.defenseXP(v, u, s.Kind == content.KindMagic)
	if !hit {
		e.emit("miss", u.ID, v.ID, 0)
		half := e.fx(u, content.FxMissHalf)
		if !phys || half == 0 {
			return 0, false, false
		}
		d *= half
	} else if crit {
		d *= R.CritMultiplier
		e.emit("critical", u.ID, v.ID, 0)
	}
	if k := e.fx(v, content.FxBulwark); k > 0 {
		limit := float64(e.stats(v).MaxHP) * k
		if d > limit {
			d = limit + (d-limit)/2
		}
	}
	n := e.hurt(v, int(math.Floor(d)), u)
	if hit && n > 0 && phys && e.has(u, content.FxPoison) {
		e.applyStatus(v, "poison", 3, 0)
		if status, ok := v.Status["poison"]; ok {
			status.Source = u.ID
			v.Status["poison"] = status
		}
	}
	return n, hit, crit
}

// leech is the HP a striker takes back from damage it dealt.
func (e *Engine) leech(u *Unit, dealt int) {
	if k := e.fx(u, content.FxLifeSteal); k > 0 && u.HP > 0 && dealt > 0 {
		u.HP = min(e.stats(u).MaxHP, u.HP+int(float64(dealt)*k))
	}
}

// attackOn is one physical attack on its victims. Whether it strikes twice is rolled once, against the
// target the attack was aimed at, and holds for every victim unless twice forces it; all take the first
// strike before any takes the second, so a splash lands with the first swing. A miss lets a riposting
// victim strike back.
func (e *Engine) attackOn(u, aim *Unit, vs []*Unit, s content.Skill, share map[string]float64, derived, twice bool) map[string]int {
	var again []*Unit
	if twice || float64(e.rand(100)) < e.doubleChance(u, aim, s) || s.Strikes > 1 {
		again = vs
	}
	dealt := map[string]int{}
	for i, round := range [][]*Unit{vs, again} {
		for _, v := range round {
			if u.HP <= 0 {
				return dealt
			}
			if v.HP <= 0 {
				continue
			}
			b := blow{share: share[v.ID], derived: derived}
			if i == 1 {
				e.emit("double", u.ID, v.ID, 0)
				if !e.has(u, content.FxFullShare) {
					b.share *= e.data.Rules.DoubleShare
				}
			}
			n, hit, _ := e.strike(u, v, s, b)
			dealt[v.ID] += n
			if !hit && !derived {
				e.riposte(v, u)
			}
		}
	}
	return dealt
}
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// strikeBack is a basic attack set off by another one (a counter, assist or riposte) at share k of a
// normal blow. It is aimed at the first of aims and also hits the rest, each widened by u's area traits,
// and strikes twice like any attack of u's, or surely when twice is set. u heals by what it deals.
func (e *Engine) strikeBack(u *Unit, aims []*Unit, k float64, twice bool) {
	var vs []*Unit
	share := map[string]float64{}
	for _, aim := range aims {
		hit, sh := e.victims(u, aim, e.basic())
		for _, q := range hit {
			if share[q.ID] == 0 {
				vs = append(vs, q)
			}
			share[q.ID] = math.Max(share[q.ID], sh[q.ID]*k)
		}
	}
	for _, n := range e.attackOn(u, aims[0], vs, e.basic(), share, true, twice) {
		e.leech(u, n)
	}
}

// riposte is the defender's answer to a missed blow, from anywhere.
func (e *Engine) riposte(v, u *Unit) {
	if k := e.fx(v, content.FxRiposte); k > 0 && v.HP > 0 && u.HP > 0 {
		e.strikeBack(v, []*Unit{u}, k, false)
	}
}

// counterShare is the share of a normal blow v strikes back at u with: full for a 위압 unit, CounterShare
// from a counter stance or trait, PlainCounterShare otherwise. forced is a counter by force against a
// weaker attacker, which surely strikes twice.
func (e *Engine) counterShare(u, v *Unit) (k float64, forced bool) {
	forced = e.has(v, content.FxCounterForce) && e.might(u) < e.might(v)
	if e.has(v, content.FxFullShare) {
		return 1, forced
	}
	if v.Status["counter"].Turns > 0 || forced || e.has(v, content.FxCounterAlways) {
		return e.data.Rules.CounterShare, forced
	}
	return e.data.Rules.PlainCounterShare, false
}

// counterBack is the target's counter-attack after it took u's physical action; every unit strikes back
// at an attacker in its reach.
func (e *Engine) counterBack(u, v *Unit) {
	if v.HP <= 0 || u.HP <= 0 || !e.inRange(v, u, e.basic()) {
		return
	}
	k, forced := e.counterShare(u, v)
	e.strikeBack(v, []*Unit{u}, k, forced)
}

// victims are the units a physical action strikes with the share of full damage each takes.
func (e *Engine) victims(u, v *Unit, s content.Skill) (out []*Unit, share map[string]float64) {
	share = map[string]float64{}
	add := func(q *Unit, k float64) {
		if q != nil && q.HP > 0 && q.Faction != u.Faction && k > share[q.ID] {
			if share[q.ID] == 0 {
				out = append(out, q)
			}
			share[q.ID] = k
		}
	}
	full := e.has(u, content.FxFullShare)
	side := func(q *Unit, k float64) { // struck besides the target: never more than the rules' share, or in full for 위압
		if full {
			k = 1
		} else if m := e.data.Rules.SideShareMax; m > 0 {
			k = math.Min(k, m)
		}
		add(q, k)
	}
	if s.Each {
		for _, aim := range e.targets(u, v, s) {
			hit, sh := e.victims(u, aim, single(s))
			for _, q := range hit {
				add(q, sh[q.ID])
			}
		}
		return
	}
	for _, q := range e.targets(u, v, s) {
		add(q, 1)
	}
	if s.Shape != content.ShapeSingle {
		return
	}
	cross, square := e.fx(u, content.FxSplashCross), e.fx(u, content.FxSplashSquare)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			k := square
			if dx == 0 || dy == 0 {
				k = math.Max(k, cross)
			}
			if k > 0 {
				side(e.occupied(v.X+dx, v.Y+dy), k)
			}
		}
	}
	if k := e.fx(u, content.FxSweepArc); k > 0 && adjacent(u.X, u.Y, v.X, v.Y) && !e.rangedAttack(u, v, s) {
		for _, d := range arcAhead(sign(v.X-u.X), sign(v.Y-u.Y)) {
			side(e.occupied(u.X+d.X, u.Y+d.Y), k)
		}
	}
	k := e.fx(u, content.FxPierce)
	if s.Pierce > 0 {
		k = math.Max(k, s.Pierce+e.fx(u, content.FxPierceBoost))
	}
	if k > 0 {
		dx, dy := v.X-u.X, v.Y-u.Y
		if abs(dx) >= abs(dy) {
			dx, dy = sign(dx), 0
		} else {
			dx, dy = 0, sign(dy)
		}
		side(e.occupied(v.X+dx, v.Y+dy), k)
	}
	return
}

// ring is the eight cells around a unit in compass order on the screen: N NE E SE S SW W NW.
var ring = [8]content.Point{{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1}, {X: 1}, {X: 1, Y: 1}, {Y: 1}, {X: -1, Y: 1}, {X: -1}}

// arcAhead is the five cells of the ring ahead of a strike toward (dx, dy): that cell and two to each side of it.
func arcAhead(dx, dy int) []content.Point {
	for i, p := range ring {
		if p.X == dx && p.Y == dy {
			out := make([]content.Point, 0, 5)
			for k := -2; k <= 2; k++ {
				out = append(out, ring[(i+k+8)%8])
			}
			return out
		}
	}
	return nil
}

// targets are the units a skill reaches, by its shape: those of the opposite side for harm, the caster's side for help.
func (e *Engine) targets(u, v *Unit, s content.Skill) []*Unit {
	shape := s.Shape
	if shape == content.ShapeSingle && s.Kind != content.KindPhysical && s.Mode != content.ModeSelf && u.Status["eightfold"].Turns > 0 {
		shape = content.ShapeSquare
	}
	dx, dy := sign(v.X-u.X), 0
	if abs(v.Y-u.Y) > abs(v.X-u.X) {
		dx, dy = 0, sign(v.Y-u.Y)
	}
	out := []*Unit{}
	for _, id := range keys(e.unitIDs) {
		q := e.unit(id)
		if q.HP <= 0 {
			continue
		}
		friendly := s.Kind == content.KindHeal || s.Kind == content.KindBuff
		if friendly != (q.Faction == u.Faction) {
			continue
		}
		in := false
		switch shape {
		case content.ShapeAll:
			in = e.inRange(u, q, s)
		case content.ShapeCross:
			in = dist(q.X, q.Y, v.X, v.Y) <= 1
		case content.ShapeSquare:
			in = abs(q.X-v.X) <= 1 && abs(q.Y-v.Y) <= 1
		case content.ShapeLine:
			for i := 1; i <= s.Length; i++ {
				in = in || q.X == u.X+dx*i && q.Y == u.Y+dy*i
			}
		default:
			in = q.ID == v.ID
		}
		if in {
			out = append(out, q)
		}
	}
	return out
}
func (e *Engine) useSkill(u *Unit, c Command) error {
	s, err := e.skillDef(c)
	if err != nil {
		return err
	}
	v, err := e.validateSkill(u, c, s)
	if err != nil {
		return err
	}
	if n := e.cost(u, s); n > 0 {
		hp, _ := e.payable(u, n)
		u.MP = max(0, u.MP-n)
		if hp > 0 {
			u.HP -= hp
			e.emit("damage", u.ID, u.ID, hp)
		}
		e.emit("cost", u.ID, "", n)
	}
	if c.Kind == "attack" {
		u.Attacked = true
	}
	start := len(e.events)
	if s.Kind == content.KindPhysical {
		e.physical(u, v, s, c)
	} else {
		e.spell(u, v, s, c)
	}
	// Finish the original exchange before resolving followups; followup kills do not chain.
	events := append([]Event(nil), e.events[start:]...)
	for i, event := range events {
		if event.Kind != "retreat" {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if hit := events[j]; hit.Kind == "damage" && hit.Target == event.Actor {
				e.retreatSweep(e.unit(hit.Actor), e.unit(event.Actor), false)
				break
			}
		}
	}
	return nil
}

// single is the attack a skill that strikes each target on its own makes on one of them, widened
// by the attacker's area traits like a basic attack.
func single(s content.Skill) content.Skill {
	s.Shape, s.Each = content.ShapeSingle, false
	return s
}

// physical resolves a physical action: one attack on all its victims, or, for a skill that strikes each
// target on its own, a blow with its own exchange on each of them in turn, the aimed one first.
func (e *Engine) physical(u, v *Unit, s content.Skill, c Command) {
	if !s.Each {
		hit, share := e.victims(u, v, s)
		e.exchange(u, v, hit, share, s, c)
		return
	}
	aims := e.targets(u, v, s)
	slices.SortStableFunc(aims, func(a, b *Unit) int { return btoi(b == v) - btoi(a == v) })
	for i, q := range aims {
		if u.HP <= 0 {
			return
		}
		if q.HP <= 0 {
			continue
		}
		if i > 0 {
			e.emit("next-strike", u.ID, q.ID, 0)
		}
		hit, share := e.victims(u, q, single(s))
		e.exchange(u, q, hit, share, s, c)
	}
}

// exchange is one attack on its victims with what answers it: allies' assists and the victims' counters.
func (e *Engine) exchange(u, v *Unit, hit []*Unit, share map[string]float64, s content.Skill, c Command) {
	dealt, total := e.attackOn(u, v, hit, s, share, false, false), 0
	for _, q := range hit {
		total += dealt[q.ID]
		if !e.rangedAttack(u, q, s) {
			delete(q.Status, "guard")
		}
		if s.Effect == "provoke" && dealt[q.ID] > 0 {
			e.applyStatus(q, "defense", e.data.Rules.BuffTurns, -e.data.Rules.BuffPercent)
		}
	}
	for _, id := range keys(e.unitIDs) {
		w := e.unit(id)
		k := e.fx(w, content.FxAssist)
		if k == 0 || w.ID == u.ID || w.HP <= 0 || w.Faction != u.Faction || !u.ActsIn(e.state.Turn, e.state.Army) {
			continue
		}
		var reach []*Unit
		for _, q := range hit {
			if q.HP > 0 && e.inRange(w, q, e.basic()) {
				reach = append(reach, q)
			}
		}
		if len(reach) == 0 {
			continue
		}
		slices.SortStableFunc(reach, func(a, b *Unit) int { return btoi(b == v) - btoi(a == v) })
		e.strikeBack(w, reach, k, false)
	}
	for _, q := range hit {
		e.counterBack(u, q)
		if n := dealt[q.ID]; n > 0 {
			e.emit("attack-hit", u.ID, q.ID, n)
			e.actionXP(u, q, c.Kind == "skill", q.HP == 0)
		}
		if s.Effect == "steal" && q.Items[s.Item] > 0 && e.rand(100) < 30 {
			q.Items[s.Item]--
			e.stock(u)[s.Item]++
		}
	}
	e.leech(u, total)
}

// spell resolves a magic, status, buff or heal skill on every unit it reaches.
func (e *Engine) spell(u, v *Unit, s content.Skill, c Command) {
	R := e.data.Rules
	dealt := map[string]int{}
	var struck []*Unit
	apply := func(q *Unit, derived bool) {
		if s.Kind == content.KindMagic {
			struck = append(struck, q)
			n, _, crit := e.strike(u, q, s, blow{share: 1})
			dealt[q.ID] += n
			if crit {
				e.markCell(q.X, q.Y, s.Element)
			}
			if !derived && e.has(q, content.FxReflect) && q.HP > 0 && u.HP > 0 {
				e.strike(q, u, s, blow{share: 1, derived: true})
			}
			return
		}
		if float64(e.rand(100)) >= e.hitChance(u, q, s) {
			e.emit("miss", u.ID, q.ID, 0)
			return
		}
		useful := false
		recovered := 0
		turns := s.Duration
		if s.Kind == content.KindBuff && turns >= 2 {
			turns += int(e.fx(u, content.FxBuffExtend))
		}
		switch s.Effect {
		case "heal":
			n := e.healAmount(u, float64(e.stats(u).Mind)*s.Coeff)
			actual := min(e.stats(q).MaxHP-q.HP, n)
			q.HP += actual
			useful = actual > 0
			e.emit("heal", u.ID, q.ID, actual)
		case "cure":
			for k := range content.Harmful {
				delete(q.Status, k)
			}
		case "refill":
			recovered = min(e.stats(q).MaxMP-q.MP, int(s.Coeff))
			q.MP += recovered
		case "attack", "defense", "mind", "morale", "agility", "disrupt":
			p := R.BuffPercent
			if s.Kind == content.KindStatus {
				p = -p
			}
			e.applyStatus(q, s.Effect, turns, p)
		case "speed", "range":
			e.applyStatus(q, s.Effect, turns, int(s.Coeff))
		default:
			e.applyStatus(q, s.Effect, turns, 0)
		}
		e.emit("effect", u.ID, q.ID, recovered)
		// Support spells provide training too. Full-health healing does not.
		if u.Faction == "ally" && (s.Effect != "heal" || useful) {
			e.xp(u.Character, e.supportXP(u, q))
		}
	}
	for _, q := range e.targets(u, v, s) {
		apply(q, false)
	}
	if s.Kind == content.KindMagic && s.Shape != content.ShapeAll && s.Mode != content.ModeSelf && e.has(u, content.FxChain) {
		near := []*Unit{}
		for _, id := range keys(e.unitIDs) {
			q := e.unit(id)
			if q.ID != v.ID && q.Faction == v.Faction && q.HP > 0 && abs(q.X-v.X) <= 1 && abs(q.Y-v.Y) <= 1 {
				near = append(near, q)
			}
		}
		if len(near) > 0 {
			apply(near[e.rand(len(near))], false)
		}
	}
	for _, q := range struck {
		if n := dealt[q.ID]; n > 0 {
			e.emit("spell-hit", u.ID, q.ID, n)
			e.actionXP(u, q, true, q.HP == 0)
		}
	}
}

// healAmount is a heal of base points grown by the healer's bonus.
func (e *Engine) healAmount(u *Unit, base float64) int {
	return int(base * (1 + e.fx(u, content.FxHealBonus)))
}
func (e *Engine) useItem(u *Unit, c Command) error {
	it, ok := e.data.Items[c.Item]
	if !ok {
		return fail("InvalidItem")
	}
	if it.Effect == content.ItemPromote {
		return e.promote(u, c, it)
	}
	if u.Acted {
		return fail("ActionSpent")
	}
	v := e.unit(c.Target)
	if v == nil || v.HP <= 0 || v.Faction != u.Faction || dist(u.X, u.Y, v.X, v.Y) > 1 {
		return fail("InvalidTarget")
	}
	stock := e.stock(u)
	if stock[c.Item] < 1 {
		return fail("MissingItem")
	}
	stock[c.Item]--
	users := []*Unit{v}
	if e.has(u, content.FxItemShare) {
		for _, id := range keys(e.unitIDs) {
			if w := e.unit(id); w.ID != v.ID && w.HP > 0 && w.Faction == u.Faction && dist(w.X, w.Y, v.X, v.Y) == 1 {
				users = append(users, w)
				break
			}
		}
	}
	n := e.healAmount(u, float64(it.Amount))
	for _, w := range users {
		actual := 0
		if it.Effect == content.ItemMP {
			actual = min(e.stats(w).MaxMP-w.MP, it.Amount)
			w.MP += actual
		} else {
			actual = min(e.stats(w).MaxHP-w.HP, n)
			w.HP += actual
		}
		e.emit("item", u.ID, w.ID, actual)
		if actual > 0 && u.Faction == "ally" {
			e.xp(u.Character, e.supportXP(u, w)) // like a support spell; a full bar teaches nothing
		}
	}
	return nil
}

// armyOrder lists the armies (faction colors) of a turn's side in the order they move: the stage's
// Armies first, then any other color in the order its first unit appears. alive tells which still
// have a unit on the field.
func (e *Engine) armyOrder(turn string) (order []string, alive map[string]bool) {
	alive = map[string]bool{}
	seen := map[string]bool{}
	var found []string
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if !u.ActsIn(turn, "") {
			continue
		}
		if !seen[u.Color] {
			seen[u.Color] = true
			found = append(found, u.Color)
		}
		if u.HP > 0 {
			alive[u.Color] = true
		}
	}
	for _, a := range e.data.Stages[e.state.Stage].Armies {
		if seen[a.Faction] {
			order = append(order, a.Faction)
			delete(seen, a.Faction)
		}
	}
	for _, c := range found {
		if seen[c] {
			order = append(order, c)
		}
	}
	return order, alive
}

// nextArmy is the first army of the side after the one given ("" for the first) that still has a unit.
func (e *Engine) nextArmy(turn, after string) (string, bool) {
	order, alive := e.armyOrder(turn)
	passed := after == ""
	for _, a := range order {
		if passed && alive[a] {
			return a, true
		}
		if a == after {
			passed = true
		}
	}
	return "", false
}

func (e *Engine) endFaction() {
	R := e.data.Rules
	ending, endingArmy := e.state.Turn, e.state.Army
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if !u.ActsIn(ending, endingArmy) {
			continue
		}
		for _, k := range keys(u.Status) {
			s := u.Status[k]
			if k == "counter" {
				continue // counter ends on its own
			}
			s.Turns--
			if s.Turns <= 0 {
				delete(u.Status, k)
			} else {
				u.Status[k] = s
			}
		}
	}
	// the player's army, then each allied AI army, then each enemy army, one at a time; an AI turn
	// without an army (a save from before armies) was the whole side
	next := func(turn, after string) bool {
		if turn == ending && after == "" {
			return false
		}
		if a, ok := e.nextArmy(turn, after); ok {
			e.state.Turn, e.state.Army = turn, a
			return true
		}
		return false
	}
	switch {
	case ending == "ally" && (next("support", "") || next("enemy", "")):
	case ending == "support" && (next("support", endingArmy) || next("enemy", "")):
	case ending == "enemy" && next("enemy", endingArmy):
	default:
		e.state.Turn, e.state.Army = "ally", ""
		e.state.Round++
		e.ageTerrain()
	}
	starting := e.state.Turn
	e.turnStarted = ""
	if starting != ending {
		e.turnStarted = starting // turnstart conditions hold when a side begins, not for each of its armies
	}
	// what each unit holds after damage over time, so the regeneration below can be reported
	hp, mp := map[string]int{}, map[string]int{}
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.HP <= 0 || !u.ActsIn(starting, e.state.Army) {
			continue
		}
		if e.has(u, content.FxCleanse) {
			for k := range content.Harmful {
				delete(u.Status, k)
			}
		}
		delete(u.Status, "counter")
		st := e.stats(u)
		if u.Status["poison"].Turns > 0 {
			e.dot(u, "poison", st.MaxHP*R.PoisonPercent/100)
		}
		if u.Status["burn"].Turns > 0 {
			e.dot(u, "burn", st.MaxHP*R.BurnPercent/100)
		}
		if t := e.CellProps(u.X, u.Y); t.RestHP < 0 {
			e.dot(u, "terrain", st.MaxHP*-t.RestHP/100) // the ground itself hurts: standing in fire
		}
		if u.HP <= 0 {
			continue
		}
		hp[id], mp[id] = u.HP, u.MP
		r := int(float64(st.Recovery) * (1 + e.fx(u, content.FxManaRegen)))
		heal := func(n int) { u.HP = min(st.MaxHP, u.HP+e.healAmount(u, float64(n))) }
		t := e.CellProps(u.X, u.Y)
		if t.RestHP > 0 {
			heal(st.MaxHP * t.RestHP / 100)
		}
		if t.RestMP > 0 {
			r *= t.RestMP
		}
		if k := int(e.fx(u, content.FxTurnDefense)); k > 0 {
			e.applyStatus(u, "defense", R.BuffTurns, k)
		}
		if e.has(u, content.FxFortify) && t.Castle {
			e.applyStatus(u, "defense", R.BuffTurns, R.BuffPercent)
		}
		if k := e.fx(u, content.FxRegen); k > 0 {
			heal(int(float64(st.MaxHP) * k))
		}
		u.MP = min(st.MaxMP, u.MP+r)
		if k := int(e.fx(u, content.FxHaste)); k > 0 {
			e.applyStatus(u, "speed", R.BuffTurns, k)
		}
		if k := int(e.fx(u, content.FxFarSight)); k > 0 {
			e.applyStatus(u, "range", R.BuffTurns, k)
		}
		u.Moved = false
		u.Acted = false
		u.Attacked = false
		u.Done = false
	}
	// Auras are deduplicated per recipient, independent of component iteration order.
	for _, id := range keys(e.unitIDs) {
		v := e.unit(id)
		if !v.ActsIn(starting, e.state.Army) || v.HP <= 0 {
			continue
		}
		for _, aid := range keys(e.unitIDs) {
			u := e.unit(aid)
			if k := e.fx(u, content.FxAura); k > 0 && u.ID != v.ID && u.HP > 0 && u.Faction == v.Faction && abs(u.X-v.X) <= 1 && abs(u.Y-v.Y) <= 1 {
				v.HP = min(e.stats(v).MaxHP, v.HP+e.healAmount(v, float64(e.stats(v).MaxHP)*k))
				break
			}
		}
	}
	for _, id := range keys(hp) {
		u := e.unit(id)
		if n := u.HP - hp[id]; n > 0 {
			e.emit("recover-hp", id, "", n)
		}
		if n := u.MP - mp[id]; n > 0 {
			e.emit("recover-mp", id, "", n)
		}
	}
	e.emit("turn", "", starting, e.state.Round)
	e.events[len(e.events)-1].Text = e.state.Army
}
func (e *Engine) dot(v *Unit, id string, n int) {
	if v.HP <= 0 {
		return
	}
	if source := e.unit(v.Status[id].Source); source != nil {
		actual := e.hurt(v, n, source)
		if actual > 0 && v.HP == 0 {
			e.actionXP(source, v, false, true)
		}
		return
	}
	n = min(v.HP, n)
	v.HP -= n
	e.emit("damage", "", v.ID, n)
	if v.HP == 0 {
		e.emit("retreat", v.ID, "", 0)
	}
}

// stock is the item pouch the unit draws from: its own for NPCs, the party's for player units.
func (e *Engine) stock(u *Unit) map[string]int {
	if u.NPC() {
		return u.Items
	}
	return e.state.Inventory
}
