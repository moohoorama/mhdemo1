package core

import (
	"math"
	"slices"
	"strings"

	"srpg/internal/content"
)

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

// damage is the unrounded damage of one blow before its terrain and critical boost, share and armor-like reductions.
func (e *Engine) damage(u, v *Unit, s content.Skill) float64 {
	R := e.data.Rules
	a, b := e.stats(u), e.stats(v)
	var base float64
	level := float64(e.char(u.Character).Level + R.StatLevelBase)
	if s.Kind == content.KindMagic {
		base = math.Max(1, float64(a.Mind-b.Mind)+level*R.MagicLevelScale)
	} else {
		af, tf := e.terrain(u, u.X, u.Y).Factor, e.terrain(v, v.X, v.Y).Factor
		if e.has(u, content.FxTerrainFloor) {
			af = math.Max(af, 1)
		}
		if e.has(v, content.FxTerrainFloor) {
			tf = math.Max(tf, 1)
		}
		if e.has(u, content.FxTerrainCap) {
			tf = math.Min(tf, 1)
		}
		base = math.Max(1, float64(a.Attack)*af-float64(b.Defense)*tf+level*R.PhysicalLevelScale)
	}
	out := base * s.Coeff
	if s.MagicCoeff > 0 {
		out += math.Max(1, float64(a.Mind-b.Mind)+level*R.MagicLevelScale) * s.MagicCoeff
	}
	if s.Kind == content.KindMagic {
		out *= R.MagicPower
		if R.MagicWardSpan > 0 {
			out *= 1 - clamp((e.attributes(e.char(v.Character))[2]-R.MagicWardFrom)/R.MagicWardSpan, 0, 1)
		}
		out *= 1 + e.fx(u, content.FxPower+s.Element)
		if mp := e.castingMP(u); mp > 50 && e.has(u, content.FxPatience) {
			out *= 1 + float64(mp-50)/100
		}
		if k := e.fx(u, content.FxFinisher); k > 0 {
			out *= 1 + k*(1-float64(v.HP)/float64(max(1, b.MaxHP)))
		}
	}
	if k := e.fx(v, content.FxRangedWard); k > 0 && s.Kind == content.KindPhysical && e.rangedAttack(u, v, s) {
		out *= 1 - min(k, 1) // shares from different traits (own 원거리피해감소, 거울) add up
	}
	if k := e.fx(v, content.FxMagicWard); k > 0 && s.Kind == content.KindMagic {
		out *= 1 - k
	}
	if k := e.fx(v, content.FxWard+s.Element); k > 0 && s.Kind == content.KindMagic {
		out *= 1 - min(k, 1) // 수영: water spells
	}
	if k := e.fx(u, content.FxOpening); k > 0 {
		out *= 1 + k*math.Max(0, float64(v.HP)/float64(max(1, b.MaxHP))-.5)
	}
	if k := e.fx(u, content.FxDesperate); k > 0 && e.wounded(u) {
		out *= 1 + k
	}
	if k := e.fx(v, content.FxGrit); k > 0 && e.wounded(v) {
		out *= 1 - k
	}
	if s := v.Status[holdGround]; s.Turns > 0 {
		out *= 1 - float64(s.Value)/100
	}
	if k := e.fx(v, content.FxScorn); k > 0 && s.Kind == content.KindPhysical && e.def(e.char(u.Character)).Stats[1]+10 <= e.def(e.char(v.Character)).Stats[1] {
		out *= 1 - k // 안하무인: a foe ten 무력 below it
	}
	if d := e.difficulty(); u.Faction != v.Faction { // 쉬움: the player's side hits harder and is hit softer
		if p := d.DealtPct; p > 0 && u.Faction == "ally" {
			out *= float64(p) / 100
		}
		if p := d.TakenPct; p > 0 && v.Faction == "ally" {
			out *= float64(p) / 100
		}
	}
	if p := e.state.DamagePct; p > 0 && p != 100 {
		out = math.Max(1, out*float64(p)/100)
	}
	return out
}

// boost is what a landed blow is multiplied by: its element's terrain bonus and, when critical, the critical
// bonus, added together rather than multiplied.
func (e *Engine) boost(u, v *Unit, s content.Skill, crit bool) float64 {
	k := 1.0
	if s.Kind == content.KindMagic {
		k += float64(e.data.ElementTerrain(e.Cell(v.X, v.Y), s.Element)) / 100
	}
	if crit {
		k += e.data.Rules.CritMultiplier - 1 + e.fx(u, content.FxCritBoost)
	}
	return k
}

// wounded tells whether the unit is down to half its HP or less.
func (e *Engine) wounded(u *Unit) bool { return u.HP*2 <= e.stats(u).MaxHP }

// castingMP is the spell points the unit had when it began the skill it is using now (what 인내 reads),
// or what it holds when it is not casting.
func (e *Engine) castingMP(u *Unit) int {
	if e.caster == u.ID {
		return e.castMP
	}
	return u.MP
}

// rangedAttack tells whether the blow is a shot, by the weapon that makes it: a bow, or a rider shooting
// with 기사 at a cell its melee weapon does not reach. Distance alone never makes a blow ranged.
func (e *Engine) rangedAttack(u, v *Unit, s content.Skill) bool {
	switch {
	case s.Mode == content.ModeRanged:
		return true
	case s.Mode == content.ModeMelee:
		return false
	case e.unitClass(u).Attack == content.AttackRanged:
		return true
	}
	_, ranged := e.weaponReach(u, s, content.Point{X: u.X, Y: u.Y}, v.X, v.Y)
	return ranged
}
func (e *Engine) hitChance(u, v *Unit, s content.Skill) float64 {
	R := e.data.Rules
	h := float64(s.Hit)
	switch s.Kind {
	case content.KindPhysical:
		ranged := e.rangedAttack(u, v, s)
		if v.Status["guard"].Turns > 0 && !ranged && !confused(v) {
			return 0
		}
		if e.has(v, content.FxRangedImmune) && ranged {
			return 0
		}
		if e.has(u, content.FxRangedSure) && ranged {
			return 100
		}
		h += curve(e.ratio(e.stats(u).Agility, e.stats(v).Agility), R.HitCurve) - float64(R.BasicAttack.Hit) - e.fx(v, content.FxEvade) + e.fx(u, content.FxAccuracy) +
			e.fx(u, content.FxChargeAccuracy)*float64(u.Steps)
		return clamp(h, R.HitFloor, R.HitCeil)
	case content.KindMagic, content.KindStatus:
		if e.has(v, content.FxDispel) && v.Faction != u.Faction {
			return 0
		}
		if s.Kind == content.KindStatus && e.statusImmune(v) && v.Faction != u.Faction {
			return 0
		}
		if s.Sure || e.has(u, content.FxSpellSure) || e.fx(u, content.FxSure+s.Element) > 0 {
			return 100
		}
		h += float64(e.unitDef(u).Stats[4]-e.unitDef(v).Stats[4]) * .5
		if v.Faction != u.Faction {
			h -= e.fx(v, content.FxSpellWard)
		}
		return clamp(h, R.SpellFloor, R.SpellCeil)
	}
	if s.Mode != content.ModeSelf && s.Hit < 100 {
		h += float64(e.stats(u).Mind)/100 + float64(e.unitDef(u).Stats[4])*.05 + 1
	}
	return clamp(h, R.SpellFloor, R.SpellCeil)
}

// critForced tells whether the blow is a certain critical.
func (e *Engine) critForced(u, v *Unit, s content.Skill) bool {
	if e.has(u, content.FxCritWounded) && e.wounded(v) || e.has(u, content.FxLoneCritLeader) && e.alone(u) && (v.Rank == "commander" || v.Rank == "boss") ||
		e.has(u, content.FxCritAiling) && ailing(v) {
		return true
	}
	if s.Kind == content.KindMagic {
		return e.has(u, content.FxSpellCrit) || e.fx(u, content.FxCritElement+s.Element) > 0
	}
	return e.has(u, content.FxCritAlways) || u.Status["sure_crit"].Turns > 0 ||
		e.has(u, content.FxCritAmbush) && e.CellProps(u.X, u.Y).Ambush ||
		e.has(u, content.FxCritScholar) && contains(e.unitClass(v).Tags, content.TagCivil) ||
		e.has(u, content.FxCritMounted) && e.mounted(v)
}

// alone tells whether no ally of the unit stands in the eight cells around it (일기당천).
func (e *Engine) alone(u *Unit) bool { return e.around(u, true) == 0 }

// around counts the living allies (or foes) of the unit in the eight cells around it.
func (e *Engine) around(u *Unit, allies bool) int {
	n := 0
	for _, p := range ring {
		if q := e.occupied(u.X+p.X, u.Y+p.Y); q != nil && (q.Faction == u.Faction) == allies {
			n++
		}
	}
	return n
}

// critChance is the chance in percent that a blow lands critically, by the morale of both sides; 100 when forced.
func (e *Engine) critChance(u, v *Unit, s content.Skill) float64 {
	if e.critForced(u, v, s) {
		return 100
	}
	return chance(e.data.Rules.Crit, e.ratio(e.stats(u).Morale, e.stats(v).Morale))
}

// doubleChance is the chance in percent that a physical skill strikes the target twice.
func (e *Engine) doubleChance(u, v *Unit, s content.Skill) float64 {
	if e.has(u, content.FxDoubleAlways) {
		return 100
	}
	return chance(e.data.Rules.Double, e.ratio(e.stats(u).Agility, e.stats(v).Agility))
}

// statusImmune tells whether harmful statuses fail on the unit: its own 군율, or an adjacent ally's 엄명.
func (e *Engine) statusImmune(v *Unit) bool {
	if e.has(v, content.FxStatusImmune) {
		return true
	}
	for _, p := range ring {
		if q := e.occupied(v.X+p.X, v.Y+p.Y); q != nil && q.Faction == v.Faction && e.has(q, content.FxStatusWard) {
			return true
		}
	}
	return false
}

// applyStatus puts a status on v. A stat percentage replaces an opposite one and otherwise keeps the stronger.
// applyStatus puts status id on v and reports whether that changed anything. A buff and a debuff of one stat cancel
// out, the later one ending the earlier, so a stat steps from down to even to up; one of the same side keeps the
// stronger value and the longer of the two durations.
func (e *Engine) applyStatus(v *Unit, id string, turns, value int) bool {
	if content.Harmful[id] && e.statusImmune(v) {
		return false
	}
	old := v.Status[id]
	if old.Turns > 0 && value != 0 && old.Value != 0 && (value > 0) != (old.Value > 0) {
		delete(v.Status, id)
		return true
	}
	if old.Turns > 0 && (value >= 0) == (old.Value >= 0) {
		if abs(old.Value) > abs(value) {
			value = old.Value
		}
		turns = max(turns, old.Turns)
	}
	v.Status[id] = Status{Turns: turns, Value: value, Source: old.Source}
	return old.Turns <= 0 || value != old.Value || turns > old.Turns
}
func (e *Engine) hurt(v *Unit, n int, u *Unit) int {
	n = min(v.HP, max(0, n))
	if n >= v.HP && n > 0 && !v.Spared && e.has(v, content.FxLastStand) {
		n = v.HP - 1
		v.Spared = true
	}
	if e.undying(v) {
		n = min(n, v.HP-1)
	}
	v.HP -= n
	e.emit("damage", u.ID, v.ID, n)
	if v.HP == 0 && n > 0 {
		e.emit("retreat", v.ID, "", 0)
	}
	return n
}

// blow is how one strike is made: its share of full damage, whether it was set off by another attack
// (a counter, assist, guard or reflection), which never sets off another, and whether it is an attack's second strike.
type blow struct {
	share   float64
	derived bool
	second  bool
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
	if s.Kind == content.KindMagic && e.has(v, content.FxImmune+s.Element) {
		e.hurt(v, 0, u) // 화덕성군: the fire does not touch it
		return 0, true, false
	}
	hit := float64(e.rand(100)) < e.hitChance(u, v, s)
	crit := false
	if hit {
		crit = e.critForced(u, v, s)
		if !crit {
			crit = float64(e.rand(100)) < e.critChance(u, v, s)
		}
		if crit && e.has(v, content.FxCritImmune) || b.second && e.has(v, content.FxDoubleWard) {
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
	} else {
		d *= e.boost(u, v, s, crit)
		if crit {
			e.emit("critical", u.ID, v.ID, 0)
		}
	}
	n := e.hurt(v, int(math.Floor(e.bulwark(v, d))), u)
	if hit && n > 0 && phys {
		if k := e.fx(u, content.FxPlunder); k > 0 && !u.NPC() {
			e.state.Money += int(k)
		}
		if e.has(u, content.FxPoison) {
			e.applyStatus(v, "poison", 3, 0)
			if status, ok := v.Status["poison"]; ok {
				status.Source = u.ID
				v.Status["poison"] = status
			}
		}
		for id := range content.StatStatuses {
			if e.has(u, content.FxHitDebuff+id) {
				e.applyStatus(v, id, R.BuffTurns, -R.BuffPercent)
			}
		}
		for id := range content.Harmful {
			if k := int(e.fx(u, content.FxHitStatus+id)); k > 0 {
				e.applyStatus(v, id, k, 0)
			}
		}
	}
	return n, hit, crit
}

// holdGround is the status 진지고수 leaves on a unit that ended its own turn without walking: blows on it deal
// Value percent less until its next turn starts.
const holdGround = "hold_ground"

// standFast gives u 진지고수's guard when it ends its turn where the turn began.
func (e *Engine) standFast(u *Unit) {
	if k := e.fx(u, content.FxHoldGround); k > 0 && u.Steps == 0 && u.HP > 0 {
		if u.Status == nil {
			u.Status = map[string]Status{}
		}
		u.Status[holdGround] = Status{Turns: 1, Value: int(math.Round(k * 100))}
	}
}

// bulwark is what a blow of d leaves after v's 철벽 or 철갑, which halve the part above a share of max HP;
// holding both, the lower share counts.
func (e *Engine) bulwark(v *Unit, d float64) float64 {
	k := 0.0
	for _, t := range e.traits(e.char(v.Character)) {
		if s := e.data.Traits[t].Effects[content.FxBulwark]; s > 0 && (k == 0 || s < k) {
			k = s
		}
	}
	if k > 0 {
		if limit := float64(e.stats(v).MaxHP) * k; d > limit {
			return limit + (d-limit)/2
		}
	}
	return d
}

// leech is the HP a striker takes back from damage it dealt.
func (e *Engine) leech(u *Unit, dealt int) {
	if k := e.fx(u, content.FxLifeSteal); k > 0 && u.HP > 0 && dealt > 0 {
		u.HP = min(e.stats(u).MaxHP, u.HP+int(float64(dealt)*k))
	}
}

// attackOn is one physical attack on its victims. Whether it strikes twice is rolled once, against the
// target the attack was aimed at, and holds for every victim unless twice forces it; all take the first
// strike before any takes the second, so a splash lands with the first swing. With 돌파 and knock set (an
// attack of its own or a counter) a landed first blow on the aim pushes it a cell back, or deals more where
// it cannot be pushed. A marking skill leaves its element's mark under every victim it hit, after all
// its strikes, so its own second blow does not gain from it.
func (e *Engine) attackOn(u, aim *Unit, vs []*Unit, s content.Skill, share map[string]float64, derived, knocks, twice bool) map[string]int {
	var again []*Unit
	if twice || float64(e.rand(100)) < e.doubleChance(u, aim, s) || s.Strikes > 1 {
		again = vs
	}
	dealt := map[string]int{}
	var marked []*Unit
	for i, round := range [][]*Unit{vs, again} {
		for _, v := range round {
			if u.HP <= 0 {
				return dealt
			}
			if v.HP <= 0 {
				continue
			}
			b := blow{share: share[v.ID], derived: derived, second: i == 1}
			if i == 1 {
				e.emit("double", u.ID, v.ID, 0)
				b.share *= e.data.Rules.DoubleShare + e.fx(u, content.FxHeavy)
			}
			push, canPush := content.Point{}, false
			knock := i == 0 && v == aim && knocks && s.Kind == content.KindPhysical && e.has(u, content.FxKnockback)
			if knock {
				if push, canPush = e.pushCell(u, v); !canPush {
					b.share *= 1 + e.fx(u, content.FxKnockback)
				}
			}
			n, hit, _ := e.strike(u, v, s, b)
			dealt[v.ID] += n
			if hit && s.Mark && !slices.Contains(marked, v) {
				marked = append(marked, v)
			}
			if knock && canPush && hit && v.HP > 0 {
				from := content.Point{X: v.X, Y: v.Y}
				v.X, v.Y = push.X, push.Y
				e.events = append(e.events, Event{Kind: "knockback", Actor: u.ID, Target: v.ID, Path: []content.Point{from, push}})
			}
		}
	}
	for _, v := range marked {
		e.markCell(v.X, v.Y, s.Element)
	}
	return dealt
}
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// pushCell is the cell behind v as u strikes it (away from u, along the longer axis, or diagonally on
// a diagonal blow) and whether v can be pushed there: a cell its class can enter, with nobody on it.
func (e *Engine) pushCell(u, v *Unit) (content.Point, bool) {
	dx, dy := v.X-u.X, v.Y-u.Y
	switch {
	case abs(dx) > abs(dy):
		dx, dy = sign(dx), 0
	case abs(dy) > abs(dx):
		dx, dy = 0, sign(dy)
	default:
		dx, dy = sign(dx), sign(dy)
	}
	p := content.Point{X: v.X + dx, Y: v.Y + dy}
	s := e.data.Stages[e.state.Stage]
	if p.X < 0 || p.Y < 0 || p.X >= s.Width || p.Y >= s.Height || e.terrain(v, p.X, p.Y).Cost == 0 || e.occupied(p.X, p.Y) != nil {
		return p, false
	}
	return p, true
}

// strikeBack is a basic attack set off by another one (a counter, assist or guard) at share k of a
// normal blow. It is aimed at the first of aims and also hits the rest, each widened by u's area traits,
// and strikes twice like any attack of u's, or surely when twice is set; a counter pushes like u's own
// attack (돌파). u heals by what it deals.
func (e *Engine) strikeBack(u *Unit, aims []*Unit, k float64, counter, twice bool) {
	var vs []*Unit
	var area []content.Point
	share := map[string]float64{}
	for _, aim := range aims {
		hit, sh, ar := e.victims(u, aim, e.basic())
		for _, q := range hit {
			if share[q.ID] == 0 {
				vs = append(vs, q)
			}
			share[q.ID] = math.Max(share[q.ID], sh[q.ID]*k)
		}
		area = append(area, ar...)
	}
	e.emitArea(u, area)
	for _, n := range e.attackOn(u, aims[0], vs, e.basic(), share, true, counter, twice) {
		e.leech(u, n)
	}
}

// counterShare is the share of a normal blow v strikes back at u with: CounterShare in a counter stance,
// PlainCounterShare otherwise, either raised by 참격 and 반격강화.
func (e *Engine) counterShare(v *Unit) float64 {
	k := e.data.Rules.PlainCounterShare
	if v.Status["counter"].Turns > 0 {
		k = e.data.Rules.CounterShare
	}
	return k + e.fx(v, content.FxHeavy) + e.fx(v, content.FxCounterBoost)
}

// counterBack is the target's counter-attack after it took u's physical action: every unit strikes back
// at an attacker in its reach, and one with 응징 at an attacker anywhere.
func (e *Engine) counterBack(u, v *Unit) {
	if v.HP <= 0 || u.HP <= 0 || confused(v) || !e.has(v, content.FxRetaliate) && !e.inRange(v, u, e.basic()) {
		return
	}
	e.strikeBack(v, []*Unit{u}, e.counterTaken(u, e.counterShare(v)), true, false)
}

// counterTaken is the share of a counter struck at share k that lands on u: its 반격피해감소 takes points off.
func (e *Engine) counterTaken(u *Unit, k float64) float64 {
	return math.Max(0, k-e.fx(u, content.FxCounterWard))
}

// guard is what 호위 does after u's melee attack on the struck: each ally next to one of them that has
// 호위 and reaches u strikes back once at its share.
func (e *Engine) guard(u *Unit, struck []*Unit) {
	done := map[string]bool{}
	for _, q := range struck {
		for _, p := range ring {
			w := e.occupied(q.X+p.X, q.Y+p.Y)
			if w == nil || done[w.ID] || w.Faction != q.Faction || u.HP <= 0 || confused(w) {
				continue
			}
			if k := e.fx(w, content.FxGuardian); k > 0 && e.inRange(w, u, e.basic()) {
				done[w.ID] = true
				e.strikeBack(w, []*Unit{u}, e.counterTaken(u, k), false, false)
			}
		}
	}
}

// victims are the units a physical action strikes with the share of full damage each takes, and the area:
// every cell its blow reaches besides its target's (splash, flank, pierce), taken or not.
func (e *Engine) victims(u, v *Unit, s content.Skill) (out []*Unit, share map[string]float64, area []content.Point) {
	share = map[string]float64{}
	add := func(q *Unit, k float64) {
		if q != nil && q.HP > 0 && q.Faction != u.Faction && k > share[q.ID] {
			if share[q.ID] == 0 {
				out = append(out, q)
			}
			share[q.ID] = k
		}
	}
	side := func(x, y int, k float64) { // struck besides the target: never more than the rules' share, raised by 참격
		if p := (content.Point{X: x, Y: y}); !slices.Contains(area, p) {
			area = append(area, p)
		}
		if m := e.data.Rules.SideShareMax; m > 0 {
			k = math.Min(k, m)
		}
		add(e.occupied(x, y), k+e.fx(u, content.FxHeavy))
	}
	if s.SideCoeff > 0 {
		add(v, 1)
		side(v.X-1, v.Y, s.SideCoeff)
		side(v.X+1, v.Y, s.SideCoeff)
		side(v.X, v.Y-1, s.SideCoeff)
		side(v.X, v.Y+1, s.SideCoeff)
		return
	}
	if s.Shape != content.ShapeSingle {
		for _, aim := range e.targets(u, v, s) {
			hit, sh, ar := e.victims(u, aim, single(s))
			for _, q := range hit {
				add(q, sh[q.ID])
			}
			for _, p := range ar {
				if !slices.Contains(area, p) {
					area = append(area, p)
				}
			}
		}
		return
	}
	add(v, 1)
	cross, square := e.fx(u, content.FxSplashCross), e.fx(u, content.FxSplashSquare)
	if e.rangedAttack(u, v, s) {
		cross = math.Max(cross, e.fx(u, content.FxSplashCrossRanged))
		square = math.Max(square, e.fx(u, content.FxSplashSquareRanged))
	}
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
				side(v.X+dx, v.Y+dy, k)
			}
		}
	}
	if k := e.fx(u, content.FxFlank); k > 0 && !e.rangedAttack(u, v, s) {
		dx, dy := sign(v.X-u.X), sign(v.Y-u.Y)
		side(v.X-dy, v.Y+dx, k)
		side(v.X+dy, v.Y-dx, k)
	}
	k := math.Max(e.fx(u, content.FxPierce), s.Pierce)
	if k > 0 {
		dx, dy := v.X-u.X, v.Y-u.Y // onward in the blow's direction, diagonal when it comes diagonally
		switch {
		case abs(dx) > abs(dy):
			dx, dy = sign(dx), 0
		case abs(dx) < abs(dy):
			dx, dy = 0, sign(dy)
		default:
			dx, dy = sign(dx), sign(dy)
		}
		side(v.X+dx, v.Y+dy, k)
	}
	return
}

// emitArea tells the replay which cells u's coming blow reaches besides its target's, so that nobody is
// shown walking into them while it lands.
func (e *Engine) emitArea(u *Unit, area []content.Point) {
	if len(area) > 0 {
		e.events = append(e.events, Event{Kind: "area", Actor: u.ID, Path: area})
	}
}

// ring is the eight cells around a unit.
var ring = [8]content.Point{{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1}, {X: 1}, {X: 1, Y: 1}, {Y: 1}, {X: -1, Y: 1}, {X: -1}}

// targets are the units a skill reaches, by its shape: those of the opposite side for harm, the caster's
// side for help, both for a mixed skill.
func (e *Engine) targets(u, v *Unit, s content.Skill) []*Unit {
	shape := s.Shape
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
		if friendly(s) != (q.Faction == u.Faction) && s.Kind != content.KindMixed {
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

// friendly tells whether the skill helps those it reaches.
func friendly(s content.Skill) bool { return s.Kind == content.KindHeal || s.Kind == content.KindBuff }

func (e *Engine) useSkill(u *Unit, c Command) error {
	s, err := e.skillDef(c)
	if err != nil {
		return err
	}
	v, err := e.validateSkill(u, c, s)
	if err != nil {
		return err
	}
	e.caster, e.castMP = u.ID, u.MP
	defer func() { e.caster = "" }()
	if n := e.cost(u, s); n > 0 {
		hp, _ := e.payable(u, n)
		u.MP = max(0, u.MP-n)
		if hp > 0 {
			u.HP -= hp
			e.emit("damage", u.ID, u.ID, hp)
		}
		e.emit("cost", u.ID, "", n)
	}
	if hp := e.hpCost(u, s); hp > 0 {
		u.HP -= hp
		e.emit("damage", u.ID, u.ID, hp)
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
				e.crush(e.unit(hit.Actor), e.unit(event.Actor))
				e.rout(e.unit(hit.Actor), e.unit(event.Actor))
				e.absorbMana(e.unit(hit.Actor), e.unit(event.Actor))
				e.retreatSweep(e.unit(hit.Actor), e.unit(event.Actor), false)
				break
			}
		}
	}
	return nil
}

// absorbMana is what 격파 does when u defeats v: u takes the spell points v had left.
func (e *Engine) absorbMana(u, v *Unit) {
	if u == nil || v == nil || u.HP <= 0 || !e.has(u, content.FxAbsorbMana) {
		return
	}
	n := min(e.stats(u).MaxMP-u.MP, v.MP)
	if n <= 0 {
		return
	}
	u.MP += n
	v.MP -= n
	e.emit("effect", u.ID, u.ID, n)
}

// ailing tells whether the unit carries a harmful status (약점공략).
// confused tells whether the unit is under 혼란: it neither moves nor acts, and answers no blow (counter, assist,
// guardian, reflection, 경계, 역계).
func confused(u *Unit) bool { return u.Status["confusion"].Turns > 0 }

// cure lifts every harmful status off the unit.
func cure(v *Unit) {
	for k := range content.Harmful {
		delete(v.Status, k)
	}
}

func ailing(v *Unit) bool {
	for k, st := range v.Status {
		if content.Harmful[k] && st.Turns > 0 {
			return true
		}
	}
	return false
}

// rout is what 위압 and 위풍 do when u defeats v: the foes of u in the eight cells around v lose heart.
func (e *Engine) rout(u, v *Unit) {
	if u == nil || v == nil || u.HP <= 0 {
		return
	}
	R := e.data.Rules
	for id := range content.StatStatuses {
		if !e.has(u, content.FxRout+id) {
			continue
		}
		for _, p := range ring {
			if q := e.occupied(v.X+p.X, v.Y+p.Y); q != nil && q.Faction != u.Faction {
				e.applyStatus(q, id, R.BuffTurns, -R.BuffPercent)
				e.emit("effect", u.ID, q.ID, 0)
			}
		}
	}
}

// crush is what 위진화하 does when u fells v: every stat of the foes in the eight cells around v drops.
func (e *Engine) crush(u, v *Unit) {
	if u == nil || v == nil || u.HP <= 0 || !e.has(u, content.FxCrush) {
		return
	}
	R := e.data.Rules
	for _, p := range ring {
		if q := e.occupied(v.X+p.X, v.Y+p.Y); q != nil && q.Faction != u.Faction && q.HP > 0 {
			for _, id := range keys(content.StatStatuses) {
				e.applyStatus(q, id, R.BuffTurns, -R.BuffPercent)
			}
			e.emit("effect", u.ID, q.ID, 0)
		}
	}
}

// single is the attack an area skill makes on each of its targets, widened by the attacker's area traits
// like a basic attack.
func single(s content.Skill) content.Skill {
	s.Shape, s.Each = content.ShapeSingle, false
	return s
}

// physical resolves a physical action: one attack on all its victims, or, for a skill that strikes each
// target on its own, a blow with its own exchange on each of them in turn, the aimed one first.
func (e *Engine) physical(u, v *Unit, s content.Skill, c Command) {
	if !s.Each {
		hit, share, area := e.victims(u, v, s)
		e.emitArea(u, area)
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
		hit, share, area := e.victims(u, q, single(s))
		e.emitArea(u, area)
		e.exchange(u, q, hit, share, s, c)
	}
}

// exchange is one attack on its victims with what answers it: allies' assists and the victims' counters.
func (e *Engine) exchange(u, v *Unit, hit []*Unit, share map[string]float64, s content.Skill, c Command) {
	hp := u.HP
	melee := map[string]bool{}
	for _, q := range hit {
		melee[q.ID] = !e.rangedAttack(u, q, s)
	}
	dealt, total := e.attackOn(u, v, hit, s, share, false, true, false), 0
	delete(u.Status, "sure_crit")
	for _, q := range hit {
		total += dealt[q.ID]
		if melee[q.ID] {
			delete(q.Status, "guard")
		}
	}
	if k := int(e.fx(u, content.FxChargeShock)); k > 0 && u.Steps >= k && melee[v.ID] && dealt[v.ID] > 0 && v.HP > 0 {
		e.applyStatus(v, "defense", e.data.Rules.BuffTurns, -e.data.Rules.BuffPercent)
	}
	for _, id := range keys(e.unitIDs) {
		w := e.unit(id)
		k := e.fx(w, content.FxAssist)
		if k == 0 || w.ID == u.ID || w.HP <= 0 || w.Faction != u.Faction || confused(w) || !u.ActsIn(e.state.Turn, e.state.Army) {
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
		e.strikeBack(w, reach, k, false, false)
	}
	var struck []*Unit
	for _, q := range hit {
		if melee[q.ID] {
			struck = append(struck, q)
		}
	}
	for _, q := range hit {
		e.counterBack(u, q)
	}
	e.guard(u, struck)
	for _, q := range hit {
		if n := dealt[q.ID]; n > 0 {
			e.emit("attack-hit", u.ID, q.ID, n)
			e.actionXP(u, q, c.Kind == "skill", q.HP == 0)
		}
	}
	e.leech(u, total)
	for _, q := range hit {
		if dealt[q.ID] > 0 {
			e.pouchHeal(q)
		}
	}
	if u.HP < hp {
		e.pouchHeal(u) // hurt by a counter
	}
}

// pouchHeal is the bean a unit with a ration pouch eats once an attack has hurt it and left it standing:
// only when a whole bean's heal fits under its full strength, and only while a bean is left to eat. Like an
// item it uses, the bean heals more with 물심양면 and is shared with the weakest ally around with 보급.
func (e *Engine) pouchHeal(v *Unit) {
	n := int(e.fx(v, content.FxPouchHeal))
	if v.HP <= 0 || n <= 0 || e.stats(v).MaxHP-v.HP < n {
		return
	}
	st, item := e.stock(v), e.data.Rules.PouchItem
	if st[item] < 1 {
		return
	}
	st[item]--
	n = int(float64(n) * (1 + e.fx(v, content.FxItemPower)))
	actual := min(e.stats(v).MaxHP-v.HP, n)
	v.HP += actual
	e.emit("pouch-heal", v.ID, "", actual)
	if !e.has(v, content.FxItemShare) {
		return
	}
	if w := e.weakestAround(v, v.Faction); w != nil {
		actual := min(e.stats(w).MaxHP-w.HP, n)
		w.HP += actual
		e.emit("pouch-heal", w.ID, v.ID, actual) // the target is the one whose pouch it was
	}
}

// spell resolves a magic, status, buff, heal or mixed skill on every unit it reaches.
func (e *Engine) spell(u, v *Unit, s content.Skill, c Command) {
	R := e.data.Rules
	hp := u.HP
	dealt := map[string]int{}
	var struck []*Unit
	apply := func(q *Unit, derived bool) {
		if s.Kind == content.KindMagic {
			struck = append(struck, q)
			n, hit, crit := e.strike(u, q, s, blow{share: 1})
			dealt[q.ID] += n
			if crit {
				e.markCell(q.X, q.Y, s.Element)
			}
			if hit && q.HP > 0 && e.has(u, content.FxDispelBuffs) {
				stripBuffs(q)
			}
			if !derived && e.has(q, content.FxReflect) && q.HP > 0 && u.HP > 0 && !confused(q) {
				e.strike(q, u, s, blow{share: 1, derived: true})
			}
			return
		}
		if float64(e.rand(100)) >= e.hitChance(u, q, s) {
			e.emit("miss", u.ID, q.ID, 0)
			return
		}
		if s.Kind == content.KindStatus && !derived && q.Faction != u.Faction && e.has(q, content.FxReflectStatus) && !confused(q) {
			e.emit("effect", q.ID, u.ID, 0)
			q = u // 역계: the status turns back on its caster
		}
		useful := false
		recovered := 0
		turns := s.Duration
		if s.Kind == content.KindBuff && turns >= 2 {
			turns += int(e.fx(u, content.FxBuffExtend))
		}
		if s.Kind == content.KindStatus && q.Faction != u.Faction && e.has(u, content.FxStatusDouble) {
			turns *= 2
		}
		switch s.Effect {
		case "heal":
			n := e.healAmount(u, float64(e.stats(u).Mind)*s.Coeff)
			actual := min(e.stats(q).MaxHP-q.HP, n)
			q.HP += actual
			useful = actual > 0
			e.emit("heal", u.ID, q.ID, actual)
		case "cure":
			useful = ailing(q)
			cure(q)
		case "refill":
			recovered = max(0, min(e.stats(q).MaxMP-q.MP, int(s.Coeff)))
			q.MP += recovered
			useful = recovered > 0
		case "meditate": // Coeff is the share of its max spell points it gathers
			recovered = max(0, min(e.stats(q).MaxMP-q.MP, int(float64(e.stats(q).MaxMP)*s.Coeff)))
			q.MP += recovered
			useful = recovered > 0
		case "invert":
			for k, st := range q.Status {
				if content.StatStatuses[k] && st.Value > 0 {
					st.Value = -st.Value
					q.Status[k] = st
					useful = true
				}
			}
		case "speed", "range":
			useful = e.applyStatus(q, s.Effect, turns, int(s.Coeff))
		default:
			if parts := strings.Split(s.Effect, "+"); content.StatStatuses[parts[0]] || s.Effect == "disrupt" {
				p := R.BuffPercent
				if s.Kind == content.KindStatus || s.Kind == content.KindMixed && q.Faction != u.Faction {
					p = -p
				}
				for _, id := range parts {
					useful = e.applyStatus(q, id, turns, p) || useful
				}
			} else {
				useful = e.applyStatus(q, s.Effect, turns, 0)
			}
		}
		if s.Kind == content.KindStatus && q.Faction != u.Faction && e.has(u, content.FxDispelBuffs) {
			useful = stripBuffs(q) || useful
		}
		if q.Faction == u.Faction && s.Kind != content.KindStatus && e.has(u, content.FxSpellCure) {
			useful = ailing(q) || useful
			cure(q)
		}
		e.emit("effect", u.ID, q.ID, recovered)
		// Support spells provide training only when they did something: a heal on a full unit, a buff on one that
		// already has it as long, a debuff an enemy already carries teach nothing.
		if u.Faction == "ally" && useful {
			e.xp(u.Character, e.supportXP(u, q))
		}
	}
	if s.Effect == "clear_marks" { // 지형정비: the embers, wet patches and rubble in the 3x3 round the target go
		cleared := e.clearMarks(v.X, v.Y)
		e.emit("effect", u.ID, v.ID, 0)
		if cleared && u.Faction == "ally" {
			e.xp(u.Character, e.supportXP(u, v))
		}
		return
	}
	reached := e.targets(u, v, s)
	for _, q := range reached {
		apply(q, false)
	}
	if s.Shape == content.ShapeSingle && s.Mode != content.ModeSelf && e.has(u, content.FxChain) {
		if q := e.weakestAround(v, v.Faction); q != nil {
			apply(q, false)
		}
	}
	for _, q := range struck {
		if n := dealt[q.ID]; n > 0 {
			e.emit("spell-hit", u.ID, q.ID, n)
			e.actionXP(u, q, true, q.HP == 0)
			e.pouchHeal(q)
		}
	}
	if k := e.fx(u, content.FxSpellLeech); k > 0 && s.Kind == content.KindMagic && u.HP > 0 {
		n := min(e.stats(u).MaxHP-u.HP, int(float64(e.stats(u).MaxHP)*k))
		u.HP += n
		e.emit("heal", u.ID, u.ID, n)
	}
	if u.HP < hp {
		e.pouchHeal(u) // hurt by a reflected spell
	}
}

// weakestAround is the living unit of the faction in the eight cells around v with the least HP (the first
// in id order on a tie), or nil: the one 연환 and 보급 also reach.
func (e *Engine) weakestAround(v *Unit, faction string) *Unit {
	var best *Unit
	for _, id := range keys(e.unitIDs) {
		w := e.unit(id)
		if w.ID == v.ID || w.HP <= 0 || w.Faction != faction || abs(w.X-v.X) > 1 || abs(w.Y-v.Y) > 1 {
			continue
		}
		if best == nil || w.HP < best.HP {
			best = w
		}
	}
	return best
}

// buffStatuses are the statuses that help their holder besides stat raises: 파진 strips them with those.
var buffStatuses = map[string]bool{"speed": true, "range": true, "counter": true, "guard": true, "sure_crit": true, holdGround: true,
	"stance_pierce": true, "stance_cross": true, "stance_square": true}

// stripBuffs removes every buff the unit holds.
// stripBuffs ends q's buffs, reporting whether it had any.
func stripBuffs(q *Unit) bool {
	any := false
	for k, st := range q.Status {
		if buffStatuses[k] || content.StatStatuses[k] && st.Value > 0 {
			delete(q.Status, k)
			any = true
		}
	}
	return any
}

// healAmount is a heal spell's base points grown by the caster's medicine.
func (e *Engine) healAmount(u *Unit, base float64) int {
	return int(base * (1 + e.fx(u, content.FxHealBonus)))
}
func (e *Engine) useItem(u *Unit, c Command) error {
	it, ok := e.data.Items[c.Item]
	if !ok {
		return fail("InvalidItem")
	}
	if it.Effect == content.ItemPromote {
		return fail("PromoteAway") // a seal is used off the field (promoteAway), where the officer can change weapons
	}
	if u.Acted {
		return fail("ActionSpent")
	}
	if confused(u) {
		return fail("Confused")
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
		if w := e.weakestAround(v, u.Faction); w != nil {
			users = append(users, w)
		}
	}
	n := int(float64(it.Amount) * (1 + e.fx(u, content.FxItemPower)))
	for _, w := range users {
		actual, useful := 0, false
		switch it.Effect {
		case content.ItemMP: // a drink pours past the brim: what it holds is all added
			actual = n
			w.MP += actual
		case content.ItemCure:
			useful = ailing(w)
			cure(w)
		default:
			actual = min(e.stats(w).MaxHP-w.HP, n)
			w.HP += actual
		}
		e.emit("item", u.ID, w.ID, actual)
		if (actual > 0 || useful) && u.Faction == "ally" {
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
		if !u.Done {
			e.standFast(u) // a unit left standing when its side ends the turn holds its ground too
		}
		u.Steps = 0 // 돌진 and 기습 count the walk of the unit's own turn: no charged counters on the other side's
		for _, k := range keys(u.Status) {
			s := u.Status[k]
			if k == "counter" || k == holdGround {
				continue // counter and 진지고수 end on their own, at the unit's next turn
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
		if e.has(u, content.FxPassAilments) {
			e.passAilments(u)
		}
		delete(u.Status, "counter")
		delete(u.Status, holdGround)
		u.Steps = 0
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
		lost := st.MaxHP - u.HP
		r := int(float64(st.Recovery) * (1 + e.fx(u, content.FxManaRegen)))
		heal := func(n int) { u.HP = min(st.MaxHP, u.HP+n) }
		t := e.CellProps(u.X, u.Y)
		if t.RestHP > 0 {
			heal(st.MaxHP * t.RestHP / 100)
		}
		if t.RestMP > 0 {
			r *= t.RestMP
		}
		for id, k := range e.turnBuffs(u) {
			e.applyStatus(u, id, R.BuffTurns, k)
		}
		if k := e.fx(u, content.FxRegen); k > 0 {
			heal(int(float64(st.MaxHP) * k))
		}
		if k := e.fx(u, content.FxWoundRegen); k > 0 {
			heal(int(float64(lost) * k))
		}
		u.MP = max(u.MP, min(st.MaxMP, u.MP+r)) // MP over the max from a drink stays
		if k := int(e.fx(u, content.FxHaste)); k > 0 {
			e.applyStatus(u, "speed", R.BuffTurns, k)
		}
		u.Moved = false
		u.Acted = false
		u.Attacked = false
		u.Done = false
	}
	// Auras are deduplicated per recipient, independent of component iteration order: each kind
	// comes once, from the first neighbour that has it.
	for _, id := range keys(e.unitIDs) {
		v := e.unit(id)
		if !v.ActsIn(starting, e.state.Army) || v.HP <= 0 {
			continue
		}
		got := map[string]bool{}
		for _, aid := range keys(e.unitIDs) {
			u := e.unit(aid)
			if u.ID == v.ID || u.HP <= 0 || u.Faction != v.Faction || abs(u.X-v.X) > 1 || abs(u.Y-v.Y) > 1 {
				continue
			}
			once := func(key string) float64 {
				k := e.fx(u, key)
				if k <= 0 || got[key] {
					return 0
				}
				got[key] = true
				return k
			}
			st := e.stats(v)
			if k := once(content.FxAura); k > 0 {
				v.HP = min(st.MaxHP, v.HP+int(float64(st.MaxHP)*k))
			}
			if k := once(content.FxAuraMana); k > 0 {
				v.MP = max(v.MP, min(st.MaxMP, v.MP+int(float64(st.MaxMP)*k)))
			}
			if k := once(content.FxAuraHaste); k > 0 {
				e.applyStatus(v, "speed", R.BuffTurns, int(k))
			}
			for sid := range content.StatStatuses {
				if k := once(content.FxAuraBuff + sid); k > 0 {
					e.applyStatus(v, sid, R.BuffTurns, int(k))
				}
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

// passAilments is 전가: the unit's harmful statuses and stat debuffs pass to one adjacent ally.
func (e *Engine) passAilments(u *Unit) {
	for _, p := range ring {
		w := e.occupied(u.X+p.X, u.Y+p.Y)
		if w == nil || w.Faction != u.Faction {
			continue
		}
		moved := false
		for _, k := range keys(u.Status) {
			if st := u.Status[k]; content.Harmful[k] || content.StatStatuses[k] && st.Value < 0 {
				w.Status[k] = st
				delete(u.Status, k)
				moved = true
			}
		}
		if moved {
			e.emit("effect", u.ID, w.ID, 0)
		}
		return
	}
}
func (e *Engine) dot(v *Unit, id string, n int) {
	if v.HP <= 0 {
		return
	}
	if source := e.unit(v.Status[id].Source); source != nil {
		if id == "poison" {
			n = int(float64(n) * (1 + e.fx(source, content.FxPoisonBoost)))
		}
		actual := e.hurt(v, n, source)
		if actual > 0 && v.HP == 0 {
			e.actionXP(source, v, false, true)
		}
		return
	}
	n = min(v.HP, n)
	if e.undying(v) {
		n = min(n, v.HP-1)
	}
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
