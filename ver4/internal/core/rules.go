package core

import (
	"math"
	"sort"
	"srpg/internal/content"
	"strings"
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
func (e *Engine) stats(u *Unit) Stats {
	o := e.officer(u.Officer)
	a := e.attributes(o)
	c := e.data.Classes[o.Class]
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
		v[i] = int(math.Floor(float64(k*(o.Level+10)) * (400/(140-at) + b[i])))
	}
	for _, k := range []string{"attack", "defense", "morale", "weak"} {
		s := u.Status[k]
		switch k {
		case "attack":
			v[0] += s.Value
		case "defense":
			v[1] += s.Value
		case "morale":
			v[4] += s.Value
		case "weak":
			v[0] = max(1, v[0]-s.Value)
			v[1] = max(1, v[1]-s.Value)
		}
	}
	return Stats{v[0], v[1], v[2], v[3], v[4], v[5], int(float64(c.MPBase) + a[0]*c.MPCoeff), int(float64(c.RecoveryBase) + a[2]*c.RecoveryCoeff)}
}
func (e *Engine) tile(x, y int) string {
	s := e.data.Stages[e.state.Stage]
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return "~"
	}
	return string(s.Tiles[y][x])
}
func (e *Engine) terrain(u *Unit, x, y int) content.Terrain {
	c := e.data.Classes[e.officer(u.Officer).Class]
	t := e.data.Terrain[c.Family][e.tile(x, y)]
	if e.has(u, "행군") && e.tile(x, y) == "d" && t.Cost > 0 {
		t.Cost = 1
	}
	if e.has(u, "강행") && contains([]string{"d", "s", "f"}, e.tile(x, y)) && t.Cost > 0 {
		t.Cost = max(1, t.Cost-1)
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

// reach returns the movement cost of every reachable cell and the cell each was entered from.
func (e *Engine) reach(u *Unit) (map[content.Point]int, map[content.Point]content.Point) {
	start := content.Point{X: u.X, Y: u.Y}
	cost := map[content.Point]int{start: 0}
	prev := map[content.Point]content.Point{}
	if u.Moved || u.Done || u.Status["confusion"].Turns > 0 || u.Status["root"].Turns > 0 {
		return map[content.Point]int{}, prev
	}
	budget := e.data.Classes[e.officer(u.Officer).Class].Move
	if u.Status["speed"].Turns > 0 {
		budget += 2
	}
	todo := []content.Point{start}
	for len(todo) > 0 {
		p := todo[0]
		todo = todo[1:]
		if p != start && u.Status["charge"].Turns == 0 && e.zoc(u, p.X, p.Y) {
			continue
		}
		for _, delta := range []content.Point{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
			q := content.Point{X: p.X + delta.X, Y: p.Y + delta.Y}
			t := e.terrain(u, q.X, q.Y)
			if t.Cost == 0 || e.occupied(q.X, q.Y) != nil {
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
		return content.Skill{Kind: "physical", Mode: "전체", Cost: 0, Hit: 90, Min: 1, Coeff: 1, Shape: "single"}, nil
	}
	s, ok := e.data.Skills[c.Skill]
	if !ok {
		return s, fail("InvalidSkill")
	}
	return s, nil
}
func (e *Engine) cost(u *Unit, s content.Skill) int {
	n := s.Cost
	if e.has(u, "백출") && n > 0 {
		n = max(1, (n+1)/2)
	}
	return n
}
func (e *Engine) rangeFor(u *Unit, s content.Skill) (int, int) {
	hi := s.Max
	if hi == 0 {
		if s.Mode == "자기" {
			return 0, 0
		}
		hi = e.data.Classes[e.officer(u.Officer).Class].Range
	}
	if u.Status["range"].Turns > 0 && s.Mode != "자기" {
		hi++
	}
	return s.Min, hi
}
func (e *Engine) inRange(u, v *Unit, s content.Skill) bool {
	lo, hi := e.rangeFor(u, s)
	d := dist(u.X, u.Y, v.X, v.Y)
	return d >= lo && d <= hi
}
func (e *Engine) water(x, y int) bool {
	for _, p := range []content.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		if e.tile(x+p.X, y+p.Y) == "~" {
			return true
		}
	}
	return false
}
func (e *Engine) validateSkill(u *Unit, c Command, s content.Skill) (*Unit, error) {
	if u.Status["confusion"].Turns > 0 {
		return nil, fail("Confused")
	}
	if c.Kind == "attack" {
		if u.Acted || u.Attacked {
			return nil, fail("ActionSpent")
		}
	} else {
		if !contains(e.skills(u), c.Skill) {
			return nil, fail("UnavailableSkill")
		}
		if u.Status["seal"].Turns > 0 {
			return nil, fail("Sealed")
		}
		if u.Acted && !e.has(u, "국사무쌍") {
			return nil, fail("ActionSpent")
		}
	}
	if u.MP < e.cost(u, s) {
		return nil, fail("InsufficientMP")
	}
	cl := e.data.Classes[e.officer(u.Officer).Class]
	if s.Mode == "근접" && cl.Weapon == "활" || s.Mode == "원거리" && cl.Weapon != "활" {
		return nil, fail("WrongAttackMode")
	}
	v := e.unit(c.Target)
	if s.Mode == "자기" {
		v = u
	}
	if v == nil || v.HP <= 0 {
		return nil, fail("InvalidTarget")
	}
	friendly := s.Kind == "heal" || s.Kind == "buff"
	if s.Kind != "heal" && friendly != (v.Faction == u.Faction) {
		return nil, fail("InvalidTarget")
	}
	if !e.inRange(u, v, s) {
		return nil, fail("OutOfRange")
	}
	if s.Condition == "water" && !e.water(u.X, u.Y) && !e.water(v.X, v.Y) {
		return nil, fail("TerrainCondition")
	}
	return v, nil
}
func (e *Engine) multiplier(u, v *Unit, magic bool) float64 {
	m := 1.0
	if !magic {
		if e.has(u, "호걸") {
			m *= 1.1
		}
		if e.has(u, "용장") && e.data.Classes[e.officer(u.Officer).Class].Weapon != "활" && e.data.Officers[u.Officer].Stats[1] > e.data.Officers[v.Officer].Stats[1] {
			m *= 1.2
		}
		if e.has(u, "투신") && u.HP*2 <= e.stats(u).MaxHP {
			m *= 1.3
		}
		if e.has(u, "패왕") && e.data.Classes[e.officer(u.Officer).Class].Weapon != "활" {
			m *= 1.25
		}
		family := e.data.Classes[e.officer(u.Officer).Class].Family
		for t, f := range map[string]string{"기병숙련": "기병", "보병숙련": "보병", "궁병숙련": "궁병", "도적숙련": "도적"} {
			if family == f && e.has(u, t) {
				m *= 1.15
			}
		}
		if e.has(u, "협동") {
			n := 0
			for _, id := range keys(e.unitIDs) {
				q := e.unit(id)
				if q.ID != u.ID && q.HP > 0 && q.Faction == u.Faction && abs(q.X-u.X) <= 1 && abs(q.Y-u.Y) <= 1 {
					n++
				}
			}
			if n >= 2 {
				m *= 1.15
			}
		}
		if e.has(v, "철벽") {
			m *= .85
		}
		if e.has(v, "원거리방어") && e.data.Classes[e.officer(u.Officer).Class].Weapon == "활" {
			m *= .8
		}
	} else if e.has(v, "책략방어") {
		m *= .8
	}
	if e.has(v, "불굴") && v.HP*100 <= e.stats(v).MaxHP*30 {
		m *= .75
	}
	return m
}
func (e *Engine) damage(u, v *Unit, s content.Skill) float64 {
	a, b := e.stats(u), e.stats(v)
	base := math.Max(1, 1.5*float64(a.Attack)*e.terrain(u, u.X, u.Y).Factor-.75*float64(b.Defense)*e.terrain(v, v.X, v.Y).Factor)
	if s.Kind == "magic" {
		base = math.Max(1, 2*float64(a.Mind)-.5*float64(b.Mind))
	}
	return base * s.Coeff * e.multiplier(u, v, s.Kind == "magic")
}
func (e *Engine) hitChance(u, v *Unit, s content.Skill) float64 {
	a, b := e.stats(u), e.stats(v)
	h := float64(s.Hit)
	if s.Kind == "physical" {
		h += float64(a.Agility-b.Agility) / 20
		if e.has(u, "군신") && e.data.Classes[e.officer(u.Officer).Class].Weapon != "활" && e.data.Officers[u.Officer].Stats[1] > e.data.Officers[v.Officer].Stats[1] {
			h = 100
		}
	} else if s.Kind == "magic" || s.Kind == "status" {
		h += float64(e.data.Officers[u.Officer].Stats[4]-e.data.Officers[v.Officer].Stats[4]) * .5
	} else if s.Mode != "자기" && s.Hit < 100 {
		h += float64(a.Mind)/100 + float64(e.data.Officers[u.Officer].Stats[4])*.05 + 1
	}
	if e.has(u, "행운") && (s.Kind == "magic" || s.Kind == "physical") {
		h += 5
	}
	return clamp(h, 5, 100)
}

// critChance is the chance in percent that a physical blow lands 1.5×; 100 when forced.
func (e *Engine) critChance(u, v *Unit) float64 {
	if e.has(u, "필살") || e.has(u, "군신") && e.data.Classes[e.officer(u.Officer).Class].Weapon != "활" && e.data.Officers[u.Officer].Stats[1] > e.data.Officers[v.Officer].Stats[1] {
		return 100
	}
	a, b := e.stats(u), e.stats(v)
	crit := clamp(10+float64(a.Morale-b.Morale)/20, 5, 40)
	if e.has(u, "행운") {
		crit += 5
	}
	return crit
}
func (e *Engine) applyStatus(v *Unit, id string, turns, value int) {
	if contains([]string{"poison", "confusion", "seal", "root", "weak", "burn"}, id) && e.has(v, "군율") {
		turns = max(1, turns-1)
	}
	old := v.Status[id]
	v.Status[id] = Status{Turns: turns, Value: max(old.Value, value), Source: old.Source}
}
func (e *Engine) hurt(v *Unit, n int, u *Unit) int {
	n = min(v.HP, max(0, n))
	v.HP -= n
	e.emit("damage", u.ID, v.ID, n)
	if v.HP == 0 && n > 0 {
		e.emit("retreat", v.ID, "", 0)
	}
	return n
}
func (e *Engine) strike(u, v *Unit, s content.Skill, counter bool) int {
	if u.HP <= 0 || v.HP <= 0 {
		return 0
	}
	if float64(e.rand(100)) >= e.hitChance(u, v, s) {
		e.emit("miss", u.ID, v.ID, 0)
		return 0
	}
	d := e.damage(u, v, s)
	if s.Kind == "physical" {
		a, b := e.stats(u), e.stats(v)
		roll := e.rand(100)
		if float64(roll) < e.critChance(u, v) {
			d *= 1.5
			e.emit("critical", u.ID, v.ID, 0)
		}
		guard := clamp(10+float64(b.Agility-a.Agility)/50, 5, 25)
		if e.has(v, "방패") {
			guard += 10
		}
		roll = e.rand(100)
		if float64(roll) < guard && !e.has(u, "패왕") {
			if e.has(u, "위압") {
				d *= .5
			} else {
				d = 0
			}
		}
	}
	if counter {
		d *= .7
	}
	if d > 0 && e.has(v, "천명") && e.state.Turn != v.Faction && !v.FirstHitUsed {
		d *= .5
		v.FirstHitUsed = true
	}
	n := e.hurt(v, int(math.Floor(d)), u)
	if n > 0 && s.Kind == "physical" && e.has(u, "독공") {
		e.applyStatus(v, "poison", 3, 0)
		status := v.Status["poison"]
		status.Source = u.ID
		v.Status["poison"] = status
	}
	if !counter && v.HP > 0 && u.HP > 0 && v.Status["counter"].Turns > 0 {
		basic, _ := e.skillDef(Command{Kind: "attack"})
		if e.inRange(v, u, basic) {
			r := e.strike(v, u, basic, true)
			if e.has(v, "심공") && v.HP > 0 {
				v.HP = min(e.stats(v).MaxHP, v.HP+r/4)
			}
		}
	}
	return n
}
func (e *Engine) targets(u, v *Unit, s content.Skill) []*Unit {
	out := []*Unit{}
	for _, id := range keys(e.unitIDs) {
		q := e.unit(id)
		if q.HP <= 0 {
			continue
		}
		friendly := s.Kind == "heal" || s.Kind == "buff"
		if s.Kind != "heal" && friendly != (q.Faction == u.Faction) {
			continue
		}
		if s.Shape == "all" && e.inRange(u, q, s) || s.Shape == "cross" && dist(q.X, q.Y, v.X, v.Y) <= 1 || s.Shape == "single" && q.ID == v.ID {
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
		u.MP -= n
		e.emit("cost", u.ID, "", n)
	}
	if c.Kind == "attack" {
		u.Attacked = true
		u.Acted = true
	} else if !e.has(u, "국사무쌍") {
		u.Acted = true
	}
	targets := e.targets(u, v, s)
	total := 0
	apply := func(q *Unit) {
		if s.Kind == "physical" || s.Kind == "magic" {
			n := e.strike(u, q, s, false)
			total += n
			if n > 0 {
				kind := "attack-hit"
				if c.Kind == "skill" {
					kind = "spell-hit"
				}
				e.emit(kind, u.ID, q.ID, n)
				e.actionXP(u, q, 24, c.Kind == "skill", q.HP == 0)
			}
			if c.Skill == "탈취" && q.Items["소군량"] > 0 && e.rand(100) < 30 {
				q.Items["소군량"]--
				if u.Faction == "ally" {
					e.state.Inventory["소군량"]++
				} else {
					u.Items["소군량"]++
				}
			}
			return
		}
		if float64(e.rand(100)) >= e.hitChance(u, q, s) {
			e.emit("miss", u.ID, q.ID, 0)
			return
		}
		actualSupport := false
		switch s.Effect {
		case "heal":
			n := e.healAmount(u, s)
			actual := min(e.stats(q).MaxHP-q.HP, n)
			q.HP += actual
			actualSupport = actual > 0
			e.emit("heal", u.ID, q.ID, actual)
		case "curepoison":
			delete(q.Status, "poison")
			delete(q.Status, "burn")
		case "curemental":
			for _, k := range []string{"confusion", "seal", "root"} {
				delete(q.Status, k)
			}
		case "cure":
			for _, k := range []string{"poison", "burn", "weak", "confusion", "seal", "root"} {
				delete(q.Status, k)
			}
		case "sealroot":
			e.applyStatus(q, "seal", s.Duration, 0)
			e.applyStatus(q, "root", s.Duration, 0)
		default:
			val := int(float64(e.stats(u).Mind) * s.Coeff)
			e.applyStatus(q, s.Effect, s.Duration, val)
		}
		e.emit("effect", u.ID, q.ID, 0)
		// Support spells provide training too. Full-health healing does not.
		if u.Faction == "ally" && (s.Effect != "heal" || actualSupport) {
			e.xp(u.Officer, e.supportXP(u))
		}
	}
	for _, q := range targets {
		apply(q)
	}
	if c.Kind == "skill" && s.Mode != "자기" && s.Shape != "all" && e.has(u, "연환") {
		near := []*Unit{}
		for _, id := range keys(e.unitIDs) {
			q := e.unit(id)
			if q.ID != v.ID && q.Faction == v.Faction && q.HP > 0 && abs(q.X-v.X) <= 1 && abs(q.Y-v.Y) <= 1 {
				near = append(near, q)
			}
		}
		if len(near) > 0 {
			apply(near[e.rand(len(near))])
		}
	}
	if e.has(u, "심공") && u.HP > 0 && s.Kind == "physical" {
		u.HP = min(e.stats(u).MaxHP, u.HP+total/4)
	}
	return nil
}
func (e *Engine) useItem(u *Unit, c Command) error {
	if strings.HasPrefix(c.Item, "승급:") {
		return e.promote(u, c)
	}
	if u.Acted && !e.has(u, "부호") {
		return fail("ActionSpent")
	}
	v := e.unit(c.Target)
	if v == nil || v.HP <= 0 || (v.Faction != u.Faction && c.Item == "소병법단") || dist(u.X, u.Y, v.X, v.Y) > 1 {
		return fail("InvalidTarget")
	}
	n, ok := e.data.Items[c.Item]
	if !ok {
		return fail("InvalidItem")
	}
	stock := u.Items
	if u.Faction == "ally" {
		stock = e.state.Inventory
	}
	if stock[c.Item] < 1 {
		return fail("MissingItem")
	}
	stock[c.Item]--
	if e.has(u, "보급") {
		n = int(float64(n) * 1.25)
	}
	if c.Item == "소병법단" {
		v.MP = min(e.stats(v).MaxMP, v.MP+n)
	} else {
		v.HP = min(e.stats(v).MaxHP, v.HP+n)
	}
	if !e.has(u, "부호") {
		u.Acted = true
	}
	e.emit("item", u.ID, v.ID, n)
	return nil
}
func (e *Engine) endFaction() {
	ending := e.state.Turn
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.Faction != ending {
			continue
		}
		for _, k := range keys(u.Status) {
			s := u.Status[k]
			if k == "counter" {
				continue
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
	for _, id := range keys(e.unitIDs) {
		e.unit(id).FirstHitUsed = false
	}
	// what each unit holds after damage over time, so the regeneration below can be reported
	hp, mp := map[string]int{}, map[string]int{}
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.HP <= 0 || u.Faction != starting {
			continue
		}
		if e.has(u, "침착") {
			for _, k := range []string{"poison", "burn", "weak", "confusion", "seal", "root"} {
				delete(u.Status, k)
			}
		}
		delete(u.Status, "counter")
		st := e.stats(u)
		if u.Status["poison"].Turns > 0 {
			e.dot(u, "poison", st.MaxHP/20)
		}
		if u.Status["burn"].Turns > 0 {
			e.dot(u, "burn", st.MaxHP*4/100)
		}
		if u.HP <= 0 {
			continue
		}
		hp[id], mp[id] = u.HP, u.MP
		r := st.Recovery
		if e.has(u, "집중") {
			r += 2
		}
		if e.tile(u.X, u.Y) == "v" || e.tile(u.X, u.Y) == "i" {
			r *= 2
			u.HP = min(st.MaxHP, u.HP+st.MaxHP*8/100)
		}
		if e.has(u, "양생") {
			u.HP = min(st.MaxHP, u.HP+st.MaxHP*3/100)
		}
		u.MP = min(st.MaxMP, u.MP+r)
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
			if u.ID != v.ID && u.HP > 0 && u.Faction == starting && e.has(u, "인덕") && abs(u.X-v.X) <= 1 && abs(u.Y-v.Y) <= 1 {
				v.HP = min(e.stats(v).MaxHP, v.HP+e.stats(v).MaxHP*3/100)
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

func (e *Engine) healAmount(u *Unit, s content.Skill) int {
	n := float64(e.stats(u).Mind) * s.Coeff
	if e.has(u, "의술") {
		n *= 1.25
	}
	return int(n)
}

func (e *Engine) dot(v *Unit, id string, n int) {
	if v.HP <= 0 {
		return
	}
	if source := e.unit(v.Status[id].Source); source != nil {
		actual := e.hurt(v, n, source)
		if actual > 0 && v.HP == 0 {
			e.actionXP(source, v, 24, false, true)
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
