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
	v[0] = max(1, percent(v[0], u.Status["attack"].Value))
	v[1] = max(1, percent(v[1], u.Status["defense"].Value))
	v[3] = max(1, percent(v[3], u.Status["agility"].Value))
	v[4] = max(1, percent(v[4], u.Status["morale"].Value))
	v[0] = max(1, percent(v[0], -u.Status["attack-down"].Value))
	v[1] = max(1, percent(v[1], -u.Status["defense-down"].Value))
	return Stats{v[0], v[1], v[2], v[3], v[4], v[5], int(float64(c.MPBase) + a[0]*c.MPCoeff), int(float64(c.RecoveryBase) + a[2]*c.RecoveryCoeff)}
}
func (e *Engine) tile(x, y int) string {
	s := e.data.Stages[e.state.Stage]
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return e.data.Rules.EdgeTile
	}
	return string(s.Tiles[y][x])
}
func (e *Engine) tileAt(x, y int) content.Tile { return e.data.Tiles[e.tile(x, y)] }
func (e *Engine) terrain(u *Unit, x, y int) content.Terrain {
	t := e.data.Terrain[e.unitClass(u).Terrain][e.tile(x, y)]
	if e.has(u, content.FxIgnoreMoveCost) && t.Cost > 0 {
		t.Cost = 1
	}
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
	out := []content.Point{}
	start := content.Point{X: u.X, Y: u.Y}
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
	n := cl.Move + u.Status["speed"].Value + int(e.fx(u, content.FxMove))
	if contains(cl.Tags, content.TagLight) {
		n += int(e.fx(u, content.FxLightMove))
	}
	return n
}

// reach returns the movement cost of every cell the unit can stop on and the cell each was
// entered from. Allies can be walked through but not stopped on.
func (e *Engine) reach(u *Unit) (map[content.Point]int, map[content.Point]content.Point) {
	start := content.Point{X: u.X, Y: u.Y}
	cost := map[content.Point]int{start: 0}
	prev := map[content.Point]content.Point{}
	if u.Moved || u.Done || u.Status["confusion"].Turns > 0 || u.Status["root"].Turns > 0 {
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

// cost is what the skill spends from the unit's spell points.
func (e *Engine) cost(u *Unit, s content.Skill) int {
	n := s.Cost
	if e.has(u, content.FxCostHalf) && n > 0 {
		n = max(1, (n+1)/2)
	}
	return n
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
func (e *Engine) mounted(u *Unit) bool { return contains(e.unitClass(u).Tags, content.TagMounted) }

// canRange and canMelee tell which attack modes the unit's class and traits allow.
func (e *Engine) canRange(u *Unit) bool {
	return e.unitClass(u).Attack == content.AttackRanged || e.mounted(u) && e.fx(u, content.FxMountedRange) > 0
}
func (e *Engine) canMelee(u *Unit) bool { return e.unitClass(u).Attack == content.AttackMelee }
func (e *Engine) rangeFor(u *Unit, s content.Skill) (int, int) {
	if s.Mode == content.ModeSelf {
		return 0, 0
	}
	cl := e.unitClass(u)
	lo, hi := s.Min, s.Max
	if hi == 0 {
		hi = cl.Range
		if e.mounted(u) {
			hi = max(hi, int(e.fx(u, content.FxMountedRange)))
		}
		if cl.Attack == content.AttackRanged && !e.has(u, content.FxNoMinRange) {
			lo = max(lo, cl.MinRange)
		}
	}
	hi += u.Status["range"].Value
	if s.Kind != content.KindPhysical {
		hi += int(e.fx(u, content.FxSpellRange))
	}
	return lo, hi
}
func (e *Engine) inRange(u, v *Unit, s content.Skill) bool {
	lo, hi := e.rangeFor(u, s)
	d := dist(u.X, u.Y, v.X, v.Y)
	return d >= lo && d <= hi
}

// waterNear reports whether a water tile touches (x, y).
func (e *Engine) waterNear(x, y int) bool {
	for _, p := range []content.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		if e.tileAt(x+p.X, y+p.Y).Water {
			return true
		}
	}
	return false
}
func (e *Engine) validateSkill(u *Unit, c Command, s content.Skill) (*Unit, error) {
	if u.Status["confusion"].Turns > 0 {
		return nil, fail("Confused")
	}
	if s.Kind == content.KindPhysical && u.Status["disarm"].Turns > 0 {
		return nil, fail("Disarmed")
	}
	if c.Kind == "attack" {
		if u.Acted || u.Attacked {
			return nil, fail("ActionSpent")
		}
	} else {
		if !contains(e.skills(e.char(u.Character)), c.Skill) {
			return nil, fail("UnavailableSkill")
		}
		if u.Status["seal"].Turns > 0 && s.Kind != content.KindPhysical {
			return nil, fail("Sealed")
		}
		if u.Acted {
			return nil, fail("ActionSpent")
		}
	}
	if _, ok := e.payable(u, e.cost(u, s)); !ok {
		return nil, fail("InsufficientMP")
	}
	if s.Mode == content.ModeMelee && !e.canMelee(u) || s.Mode == content.ModeRanged && !e.canRange(u) {
		return nil, fail("WrongAttackMode")
	}
	v := e.unit(c.Target)
	if s.Mode == content.ModeSelf {
		v = u
	}
	if v == nil || v.HP <= 0 {
		return nil, fail("InvalidTarget")
	}
	friendly := s.Kind == content.KindHeal || s.Kind == content.KindBuff
	if friendly != (v.Faction == u.Faction) {
		return nil, fail("InvalidTarget")
	}
	if !e.inRange(u, v, s) {
		return nil, fail("OutOfRange")
	}
	if s.Condition == "water" && !e.waterNear(u.X, u.Y) && !e.waterNear(v.X, v.Y) {
		return nil, fail("TerrainCondition")
	}
	return v, nil
}
