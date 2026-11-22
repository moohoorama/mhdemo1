package core

import (
	"math"
	"sort"
	"srpg/internal/content"
)

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func dist(x, y, a, b int) int         { return abs(x-a) + abs(y-b) }
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func sign(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}
func containsPoint(ps []content.Point, p content.Point) bool {
	for _, q := range ps {
		if q == p {
			return true
		}
	}
	return false
}

// percent scales n by a signed percentage.
func percent(n, p int) int { return n * (100 + p) / 100 }
func (e *Engine) stats(u *Unit) Stats {
	o := e.char(u.Character)
	R := e.data.Rules
	a := e.attributes(o)
	c := e.class(o)
	b := c.Bonus
	for _, eq := range o.Equipment {
		for i, n := range e.data.Equipment[eq].Bonus {
			b[i] += n
		}
	}
	var v [6]int
	for i, at := range []float64{a[1], a[0], a[2], a[3], a[4], a[5]} {
		k := 1
		if i == 5 {
			k = c.HPFactor
		}
		v[i] = int(math.Floor(float64(k*(o.Level+R.StatLevelBase)) * (R.StatNumerator/(R.StatDenominator-at) + b[i])))
	}
	attack, defense, mind := v[0], v[1], v[2]
	if e.has(u, content.FxBlendBonus) {
		v[0] += max(0, (mind-attack)/2)
		v[2] += max(0, (attack-mind)/2)
	}
	if e.has(u, content.FxDefenseMindBalance) {
		v[1] += max(0, (mind-defense)/2)
		v[2] = max(v[2], mind+(defense-mind)/2)
	}
	if u.HP > 0 { // standing on the field: who is around, how far it walked
		foes, allies := e.around(u, false), e.around(u, true)
		k := e.fx(u, content.FxOutnumbered)*float64(foes) + e.fx(u, content.FxCompanions)*float64(allies)
		if allies == 0 {
			k += e.fx(u, content.FxLone)
		}
		v[0] = int(float64(v[0]) * (1 + k + e.fx(u, content.FxChargePower)*float64(u.Steps)))
		v[1] = int(float64(v[1]) * (1 + k))
	}
	v[0] = max(1, percent(v[0], u.Status["attack"].Value))
	v[1] = max(1, percent(v[1], u.Status["defense"].Value))
	v[2] = max(1, percent(v[2], u.Status["mind"].Value))
	v[3] = max(1, percent(v[3], u.Status["agility"].Value))
	v[4] = max(1, percent(v[4], u.Status["morale"].Value))
	v[3] = max(1, percent(v[3], u.Status["disrupt"].Value))
	v[4] = max(1, percent(v[4], u.Status["disrupt"].Value))
	return Stats{v[0], v[1], v[2], v[3], v[4], v[5],
		int(float64(c.MPBase)+R.ManaMaxFactor*R.StatNumerator/(R.StatDenominator-a[0])) + int(e.fx(u, content.FxManaMax)),
		int(float64(c.RecoveryBase)+R.ManaRecoveryFactor*R.StatNumerator/(R.StatDenominator-a[2])) + int(e.fx(u, content.FxManaPlus))}
}

// Cell is the cell at (x, y) with the state an event put on it; beyond the map's edge it is the edge cell.
func (e *Engine) Cell(x, y int) content.Cell {
	s := e.data.Stages[e.state.Stage]
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		c, _ := content.ParseCell(e.data.Rules.EdgeTile)
		return c
	}
	c := s.Cell(x, y)
	for _, o := range e.state.TileStates {
		if o.X == x && o.Y == y {
			c.State = o.State[0]
		}
	}
	return c
}

// openWater reports whether the cell is water nothing bridges: nobody stands there.
func (e *Engine) openWater(x, y int) bool {
	c := e.Cell(x, y)
	return c.Structure == content.NoLayer && e.data.Grounds[string(c.Ground)].Water
}

// Boats reports whether this stage crosses water in boats.
func (e *Engine) Boats() bool { return e.data.Stages[e.state.Stage].Boats }

// CellProps is what the cell at (x, y) does besides costing steps: recovery, water, ambush, castle.
func (e *Engine) CellProps(x, y int) content.CellProps { return e.data.Props(e.Cell(x, y)) }
func (e *Engine) terrain(u *Unit, x, y int) content.Terrain {
	if s := e.data.Stages[e.state.Stage]; x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return content.Terrain{} // the edge cell is only scenery: nobody enters it, even by boat
	}
	c := e.Cell(x, y)
	t := e.data.Move(e.unitClass(u).Terrain, c, e.Boats())
	if e.has(u, content.FxIgnoreMoveCost) && t.Cost > 0 {
		t.Cost = 1
	}
	if c.Ground == content.GroundGrass && c.Structure == content.NoLayer {
		t.Factor += e.fx(u, content.FxGrassFactor)
	}
	if k := e.fx(u, content.FxWetFactor); k > 0 && e.data.Props(c).Wet {
		t.Factor += k
	}
	if k := e.fx(u, content.FxAmbushFactor); k > 0 && e.data.Props(c).Ambush {
		t.Factor += k
	}
	t.Factor += e.fx(u, content.FxTerrainFactor)
	return t
}
func (e *Engine) occupied(x, y int) *Unit {
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.HP > 0 && u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}
func (e *Engine) zoc(u *Unit, x, y int) bool {
	for _, id := range keys(e.unitIDs) {
		v := e.unit(id)
		if v.HP > 0 && v.Faction != u.Faction && dist(x, y, v.X, v.Y) == 1 {
			return true
		}
	}
	return false
}
func (e *Engine) moves(u *Unit) []content.Point {
	cost, _ := e.reach(u)
	return moveDestinations(content.Point{X: u.X, Y: u.Y}, cost)
}

func moveDestinations(start content.Point, cost map[content.Point]int) []content.Point {
	out := make([]content.Point, 0, len(cost))
	for p := range cost {
		if p != start {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Y == out[j].Y {
			return out[i].X < out[j].X
		}
		return out[i].Y < out[j].Y
	})
	return out
}

// movement is the unit's movement points this turn.
func (e *Engine) movement(u *Unit) int {
	cl := e.unitClass(u)
	return cl.Move + u.Status["speed"].Value + int(e.fx(u, content.FxMove))
}

// reach returns the movement cost of every cell the unit can stop on and the cell each was
// entered from. Allies can be walked through but not stopped on.
func (e *Engine) reach(u *Unit) (map[content.Point]int, map[content.Point]content.Point) {
	start := content.Point{X: u.X, Y: u.Y}
	cost := map[content.Point]int{start: 0}
	prev := map[content.Point]content.Point{}
	if u.Policy == "hold" || u.Moved || u.Done || confused(u) || u.Status["root"].Turns > 0 {
		return map[content.Point]int{}, prev
	}
	budget := e.movement(u)
	ignoreZOC := e.has(u, content.FxIgnoreZOC)
	todo := []content.Point{start}
	for len(todo) > 0 {
		p := todo[0]
		todo = todo[1:]
		if p != start && !ignoreZOC && e.zoc(u, p.X, p.Y) {
			continue
		}
		for _, delta := range []content.Point{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			q := content.Point{X: p.X + delta.X, Y: p.Y + delta.Y}
			t := e.terrain(u, q.X, q.Y)
			if t.Cost == 0 {
				continue
			}
			if o := e.occupied(q.X, q.Y); o != nil && o.Faction != u.Faction {
				continue
			}
			n := cost[p] + t.Cost
			if n > budget {
				continue
			}
			old, ok := cost[q]
			if ok && old <= n {
				continue
			}
			cost[q] = n
			prev[q] = p
			todo = append(todo, q)
		}
	}
	for p := range cost {
		if p != start && e.occupied(p.X, p.Y) != nil {
			delete(cost, p)
		}
	}
	return cost, prev
}

// route is the cheapest walk to a reachable cell, both ends included.
func (e *Engine) route(u *Unit, to content.Point) []content.Point {
	_, prev := e.reach(u)
	start := content.Point{X: u.X, Y: u.Y}
	out := []content.Point{to}
	for p := to; p != start; {
		p = prev[p]
		out = append(out, p)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func (e *Engine) skillDef(c Command) (content.Skill, error) {
	if c.Kind == "attack" {
		return e.data.Rules.BasicAttack, nil
	}
	s, ok := e.data.Skills[c.Skill]
	if !ok {
		return s, fail("InvalidSkill")
	}
	return s, nil
}
func (e *Engine) basic() content.Skill { return e.data.Rules.BasicAttack }

// cost is what the skill spends from the unit's spell points, halved by 백출.
func (e *Engine) cost(u *Unit, s content.Skill) int {
	n := s.Cost
	if e.has(u, content.FxCostHalf) && n > 0 {
		n = max(1, (n+1)/2)
	}
	return n
}

// SkillCost is the spell points the actor pays for the skill now.
func (e *Engine) SkillCost(actor, skill string) int {
	u := e.unit(actor)
	if u == nil {
		return 0
	}
	return e.cost(u, e.data.Skills[skill])
}

// HPCost is the HP the skill takes from the actor (명상).
func (e *Engine) HPCost(actor, skill string) int {
	u := e.unit(actor)
	if u == nil {
		return 0
	}
	return e.hpCost(u, e.data.Skills[skill])
}

// hpCost is the HP the skill takes from its user (명상), at least 1 when it asks any.
func (e *Engine) hpCost(u *Unit, s content.Skill) int {
	if s.HPCost <= 0 {
		return 0
	}
	return max(1, int(math.Ceil(float64(e.stats(u).MaxHP)*s.HPCost)))
}

// payable reports whether the unit can pay n spell points, with HP when it has blood cost; the HP owed is returned.
func (e *Engine) payable(u *Unit, n int) (hp int, ok bool) {
	if u.MP >= n {
		return 0, true
	}
	k := int(e.fx(u, content.FxBloodCost))
	hp = (n - u.MP) * k
	return hp, k > 0 && u.HP > hp
}

// weaponType is the kind of weapon the unit has equipped.
func (e *Engine) weaponType(u *Unit) string {
	return e.data.Equipment[e.char(u.Character).Equipment["weapon"]].Type
}
func (e *Engine) mounted(u *Unit) bool { return contains(e.unitClass(u).Tags, content.TagMounted) }

// mount is the equipped accessory whose colours the unit's horse takes, or "" on foot or without one.
func (e *Engine) mount(u *Unit) string {
	c := e.char(u.Character)
	if c == nil || !e.mounted(u) {
		return ""
	}
	if id := c.Equipment["accessory"]; len(e.data.Equipment[id].Palette) > 0 {
		return id
	}
	return ""
}

// shootClass is the class whose look a mounted unit takes when its basic attack shoots from range (FxMountedRange), or "".
func (e *Engine) shootClass(u *Unit) string {
	if !e.mounted(u) || e.fx(u, content.FxMountedRange) <= 0 {
		return ""
	}
	return e.unitClass(u).Shoot
}

// canRange and canMelee tell which attack modes the unit's class and traits allow.
func (e *Engine) canRange(u *Unit) bool {
	return e.unitClass(u).Attack == content.AttackRanged || e.mounted(u) && e.fx(u, content.FxMountedRange) > 0
}
func (e *Engine) canMelee(u *Unit) bool { return e.unitClass(u).Attack == content.AttackMelee }

// meleeReach tells whether a melee blow reaches the cell (dx, dy) away: the four straight neighbours, the
// diagonals too with 대각공격, two cells along the straight lines with 장창·장풍, and the diamond of two with
// both. A range buff or a skill's reach lengthens the lines (and the diamond).
func (e *Engine) meleeReach(u *Unit, s content.Skill, dx, dy int) bool {
	r := e.unitClass(u).Range + u.Status["range"].Value + s.Reach + int(e.fx(u, content.FxLongReach))
	d := abs(dx) + abs(dy)
	if d == 0 || r < 1 {
		return false
	}
	diagonal := e.has(u, content.FxDiagonal)
	switch {
	case diagonal && r >= 2:
		return d <= r
	case diagonal && abs(dx) <= 1 && abs(dy) <= 1:
		return true
	}
	return (dx == 0 || dy == 0) && d <= r
}

// rangedBand is the nearest and farthest a shot reaches: a bow or crossbow's class range, or the
// distance 기사 lets a rider shoot at; hi < lo when the unit cannot shoot.
func (e *Engine) rangedBand(u *Unit, s content.Skill) (lo, hi int) {
	cl := e.unitClass(u)
	more := u.Status["range"].Value + s.Reach + int(e.fx(u, content.FxRangedReach))
	if cl.Attack == content.AttackRanged {
		return max(1, s.Min, cl.MinRange), cl.Range + more
	}
	if k := int(e.fx(u, content.FxMountedRange)); k > 0 && e.mounted(u) {
		return 2, k + more
	}
	return 1, 0
}

// weaponReach tells whether a weapon blow from `from` reaches (x, y), and whether it would be a shot:
// a melee weapon's own reach comes first, a bow (or 기사) shoots wherever its band reaches.
func (e *Engine) weaponReach(u *Unit, s content.Skill, from content.Point, x, y int) (ok, ranged bool) {
	dx, dy := x-from.X, y-from.Y
	if s.Mode != content.ModeRanged && e.canMelee(u) && e.meleeReach(u, s, dx, dy) {
		return true, false
	}
	if s.Mode != content.ModeMelee {
		lo, hi := e.rangedBand(u, s)
		if d := abs(dx) + abs(dy); d >= lo && d <= hi {
			return true, true
		}
	}
	return false, false
}

// skillBand is the distance band of a skill with a reach of its own (spells and the like).
func (e *Engine) skillBand(u *Unit, s content.Skill) (int, int) {
	hi := s.Max + u.Status["range"].Value + s.Reach
	if s.Kind != content.KindPhysical {
		hi += int(e.fx(u, content.FxSpellRange))
	}
	return s.Min, hi
}

// reachesFrom tells whether the unit standing at from reaches (x, y) with the skill.
func (e *Engine) reachesFrom(u *Unit, s content.Skill, from content.Point, x, y int) bool {
	if s.Mode == content.ModeSelf {
		return x == from.X && y == from.Y
	}
	if s.Max == 0 {
		ok, _ := e.weaponReach(u, s, from, x, y)
		return ok
	}
	lo, hi := e.skillBand(u, s)
	d := dist(from.X, from.Y, x, y)
	return d >= lo && d <= hi
}
func (e *Engine) inRange(u, v *Unit, s content.Skill) bool {
	return e.reachesFrom(u, s, content.Point{X: u.X, Y: u.Y}, v.X, v.Y)
}

// rangeFor is the distance band the skill's reach falls in, for screens and AI estimates; the exact cells
// are reachesFrom's.
func (e *Engine) rangeFor(u *Unit, s content.Skill) (int, int) {
	if s.Mode == content.ModeSelf {
		return 0, 0
	}
	if s.Max > 0 {
		return e.skillBand(u, s)
	}
	lo, hi := 0, 0
	if s.Mode != content.ModeRanged && e.canMelee(u) {
		r := e.unitClass(u).Range + u.Status["range"].Value + s.Reach + int(e.fx(u, content.FxLongReach))
		if r > 0 {
			lo, hi = 1, r
			if e.has(u, content.FxDiagonal) && r == 1 {
				hi = 2
			}
		}
	}
	if s.Mode != content.ModeMelee {
		if l, h := e.rangedBand(u, s); h >= l {
			if hi == 0 {
				lo = l
			}
			hi = max(hi, h)
		}
	}
	return lo, hi
}

func (e *Engine) validateSkill(u *Unit, c Command, s content.Skill) (*Unit, error) {
	if err := e.validateSkillUse(u, c, s); err != nil {
		return nil, err
	}
	v := e.unit(c.Target)
	if s.Mode == content.ModeSelf {
		v = u
	}
	if v == nil || v.HP <= 0 {
		return nil, fail("InvalidTarget")
	}
	if friendly(s) != (v.Faction == u.Faction) && s.Kind != content.KindMixed && s.Mode != content.ModeSelf {
		return nil, fail("InvalidTarget")
	}
	if !e.inRange(u, v, s) {
		return nil, fail("OutOfRange")
	}
	return v, nil
}

func (e *Engine) validateSkillUse(u *Unit, c Command, s content.Skill) error {
	if confused(u) {
		return fail("Confused")
	}
	if s.Kind == content.KindPhysical && u.Status["disarm"].Turns > 0 {
		return fail("Disarmed")
	}
	if c.Kind == "attack" {
		if u.Acted || u.Attacked {
			return fail("ActionSpent")
		}
	} else {
		if !contains(e.skills(e.char(u.Character)), c.Skill) {
			return fail("UnavailableSkill")
		}
		if u.Status["seal"].Turns > 0 && s.Kind != content.KindPhysical {
			return fail("Sealed")
		}
		if u.Acted {
			return fail("ActionSpent")
		}
	}
	if _, ok := e.payable(u, e.cost(u, s)); !ok {
		return fail("InsufficientMP")
	}
	if hp := e.hpCost(u, s); hp > 0 && u.HP <= hp {
		return fail("InsufficientHP")
	}
	if s.Mode == content.ModeMelee && !e.canMelee(u) || s.Mode == content.ModeRanged && !e.canRange(u) {
		return fail("WrongAttackMode")
	}
	if s.Weapon != "" && e.weaponType(u) != s.Weapon {
		return fail("WrongWeapon")
	}
	return nil
}
