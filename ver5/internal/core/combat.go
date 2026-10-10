package core

import (
	"math"
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
	if s.Kind == content.KindMagic {
		base = math.Max(1, R.MagicPower*float64(a.Mind)-R.MagicGuard*float64(b.Mind))
	} else {
		tf := e.terrain(v, v.X, v.Y).Factor
		if e.has(u, content.FxIgnoreTerrain) {
			tf = 1
		}
		atk, def := float64(a.Attack)*e.terrain(u, u.X, u.Y).Factor, float64(b.Defense)*tf
		base = math.Max(1, atk*curve(atk/math.Max(1, def), R.DamageCurve))
	}
	return base * s.Coeff
}

// rangedAttack tells whether the blow comes from a distance: by an archer's weapon or at range 2 and beyond.
func (e *Engine) rangedAttack(u, v *Unit, s content.Skill) bool {
	if s.Mode == content.ModeRanged {
		return true
	}
	return s.Mode == content.ModeAny && (e.unitClass(u).Attack == content.AttackRanged || dist(u.X, u.Y, v.X, v.Y) > 1)
}
func (e *Engine) hitChance(u, v *Unit, s content.Skill) float64 {
	R := e.data.Rules
	h := float64(s.Hit)
	switch s.Kind {
	case content.KindPhysical:
		if e.has(v, content.FxRangedImmune) && e.rangedAttack(u, v, s) {
			return 0
		}
		h += curve(e.ratio(e.stats(u).Agility, e.stats(v).Agility), R.HitCurve) - float64(R.BasicAttack.Hit)
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
		e.has(u, content.FxCritAmbush) && e.tileAt(u.X, u.Y).Ambush
}

// critChance is the chance in percent that a physical blow lands critically; 100 when forced.
func (e *Engine) critChance(u, v *Unit) float64 {
	if e.critForced(u, v, e.basic()) {
		return 100
	}
	return chance(e.data.Rules.Crit, e.ratio(e.stats(u).Morale, e.stats(v).Morale))
}

// doubleChance is the chance in percent that a physical skill strikes the target twice.
func (e *Engine) doubleChance(u, v *Unit) float64 {
	if e.has(u, content.FxDoubleAlways) {
		return 100
	}
	return chance(e.data.Rules.Double, e.ratio(e.stats(u).Agility, e.stats(v).Agility))
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

// strike is one blow: hit, critical and damage, with a missed blow still landing a share of it for 위압-like traits.
// It reports the damage dealt and whether the blow hit.
func (e *Engine) strike(u, v *Unit, s content.Skill, b blow) (int, bool) {
	if u.HP <= 0 || v.HP <= 0 {
		return 0, false
	}
	R := e.data.Rules
	phys := s.Kind == content.KindPhysical
	d := e.damage(u, v, s) * b.share
	hit := float64(e.rand(100)) < e.hitChance(u, v, s)
	crit := false
	if hit {
		crit = e.critForced(u, v, s)
		if phys && !crit {
			crit = float64(e.rand(100)) < e.critChance(u, v)
		}
		if crit && phys && e.has(v, content.FxCritImmune) {
			hit, crit = false, false
		}
	}
	if !hit {
		e.emit("miss", u.ID, v.ID, 0)
		half := e.fx(u, content.FxMissHalf)
		if !phys || half == 0 {
			return 0, false
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
	return n, hit
}

// leech is the HP a striker takes back from damage it dealt.
func (e *Engine) leech(u *Unit, dealt int) {
	if k := e.fx(u, content.FxLifeSteal); k > 0 && u.HP > 0 && dealt > 0 {
		u.HP = min(e.stats(u).MaxHP, u.HP+int(float64(dealt)*k))
	}
}

// attackOn is one physical attack on one target: the second strike is rolled before the first lands,
// and a miss lets a riposting target strike back.
func (e *Engine) attackOn(u, v *Unit, s content.Skill, b blow) int {
	double := float64(e.rand(100)) < e.doubleChance(u, v)
	total := 0
	for i := 0; i < 1+btoi(double) && u.HP > 0 && v.HP > 0; i++ {
		if i == 1 {
			e.emit("double", u.ID, v.ID, 0)
		}
		n, hit := e.strike(u, v, s, b)
		total += n
		if !hit && !b.derived {
			e.riposte(v, u)
		}
	}
	return total
}
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// riposte is the defender's answer to a missed blow, from anywhere.
func (e *Engine) riposte(v, u *Unit) {
	if k := e.fx(v, content.FxRiposte); k > 0 && v.HP > 0 && u.HP > 0 {
		n, _ := e.strike(v, u, e.basic(), blow{share: k, derived: true})
		e.leech(v, n)
	}
}

// counterBack is the target's counter-attack after it took u's physical action: from a counter stance,
// or by force against a weaker attacker. Its strikes are rolled for a second one like any attack.
func (e *Engine) counterBack(u, v *Unit) {
	if v.HP <= 0 || u.HP <= 0 {
		return
	}
	forced := e.has(v, content.FxCounterForce) && e.might(u) < e.might(v)
	if v.Status["counter"].Turns == 0 && !forced || !e.inRange(v, u, e.basic()) {
		return
	}
	R := e.data.Rules
	double := forced || float64(e.rand(100)) < e.doubleChance(v, u)
	for i := 0; i < 1+btoi(double) && u.HP > 0; i++ {
		n, _ := e.strike(v, u, e.basic(), blow{share: R.CounterShare, derived: true})
		e.leech(v, n)
	}
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
				add(e.occupied(v.X+dx, v.Y+dy), k)
			}
		}
	}
	if k := e.fx(u, content.FxPierce); k > 0 {
		dx, dy := v.X-u.X, v.Y-u.Y
		if abs(dx) >= abs(dy) {
			dx, dy = sign(dx), 0
		} else {
			dx, dy = 0, sign(dy)
		}
		add(e.occupied(v.X+dx, v.Y+dy), k)
	}
	return
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
	if d, ok := e.duelFor(u, v, s); ok {
		e.duel(u, v, d)
		return nil
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
	if s.Kind == content.KindPhysical {
		e.physical(u, v, s, c)
	} else {
		e.spell(u, v, s, c)
	}
	return nil
}
func (e *Engine) physical(u, v *Unit, s content.Skill, c Command) {
	hit, share := e.victims(u, v, s)
	dealt, total := map[string]int{}, 0
	for _, q := range hit {
		dealt[q.ID] = e.attackOn(u, q, s, blow{share: share[q.ID]})
		total += dealt[q.ID]
	}
	for _, id := range keys(e.unitIDs) {
		w := e.unit(id)
		k := e.fx(w, content.FxAssist)
		if k == 0 || w.ID == u.ID || w.HP <= 0 || w.Faction != u.Faction || e.state.Turn != u.Faction {
			continue
		}
		for _, q := range hit {
			if q.HP > 0 && e.inRange(w, q, e.basic()) {
				n := e.attackOn(w, q, e.basic(), blow{share: k, derived: true})
				e.leech(w, n)
			}
		}
	}
	for _, q := range hit {
		e.counterBack(u, q)
		if n := dealt[q.ID]; n > 0 {
			e.emit("attack-hit", u.ID, q.ID, n)
			e.actionXP(u, q, c.Kind == "skill", q.HP == 0)
		}
		if s.Effect == "steal" && q.Items[s.Item] > 0 && e.rand(100) < 30 {
			q.Items[s.Item]--
			if u.Faction == "ally" {
				e.state.Inventory[s.Item]++
			} else {
				u.Items[s.Item]++
			}
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
			n, _ := e.strike(u, q, s, blow{share: 1})
			dealt[q.ID] += n
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
		case "attack", "defense", "morale", "agility":
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
			e.xp(u.Character, e.supportXP(u))
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
	stock := u.Items
	if u.Faction == "ally" {
		stock = e.state.Inventory
	}
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
	}
	return nil
}
func (e *Engine) endFaction() {
	R := e.data.Rules
	ending := e.state.Turn
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.Faction != ending {
			continue
		}
		for _, k := range keys(u.Status) {
			s := u.Status[k]
			if k == "counter" || k == "attack-down" || k == "defense-down" {
				continue // counter ends on its own; the duel debuffs last the battle
			}
			s.Turns--
			if s.Turns <= 0 {
				delete(u.Status, k)
			} else {
				u.Status[k] = s
			}
		}
	}
	if ending == "ally" {
		e.state.Turn = "enemy"
	} else {
		e.state.Turn = "ally"
		e.state.Round++
	}
	starting := e.state.Turn
	// what each unit holds after damage over time, so the regeneration below can be reported
	hp, mp := map[string]int{}, map[string]int{}
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.HP <= 0 || u.Faction != starting {
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
		if u.HP <= 0 {
			continue
		}
		hp[id], mp[id] = u.HP, u.MP
		r := int(float64(st.Recovery) * (1 + e.fx(u, content.FxManaRegen)))
		heal := func(n int) { u.HP = min(st.MaxHP, u.HP+e.healAmount(u, float64(n))) }
		t := e.tileAt(u.X, u.Y)
		heal(st.MaxHP * t.RestHP / 100)
		if t.RestMP > 0 {
			r *= t.RestMP
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
		if v.Faction != starting || v.HP <= 0 {
			continue
		}
		for _, aid := range keys(e.unitIDs) {
			u := e.unit(aid)
			if k := e.fx(u, content.FxAura); k > 0 && u.ID != v.ID && u.HP > 0 && u.Faction == starting && abs(u.X-v.X) <= 1 && abs(u.Y-v.Y) <= 1 {
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
