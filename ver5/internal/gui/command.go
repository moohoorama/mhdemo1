package gui

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
	"srpg/internal/session"
)

// order is the player's command in progress, 조조전 style: pick a unit, move it
// tentatively, then choose an action and its target. The tentative move lives in a
// sandbox engine so it can be taken back; nothing reaches the session until confirmed.
type order struct {
	actor   string
	stage   string // move, menu, sub, target
	sandbox *core.Engine
	move    *core.Command
	sub     string       // skill, item
	pick    core.Command // the chosen action, Target unset
}

// engine is where the order's options are computed: the sandbox after a tentative move.
func (g *Game) engine() *core.Engine {
	if g.ord.sandbox != nil {
		return g.ord.sandbox
	}
	return g.S.Engine
}

func (g *Game) unitView(o core.Observation, id string) *core.UnitView {
	for i := range o.UnitViews {
		if o.UnitViews[i].ID == id {
			return &o.UnitViews[i]
		}
	}
	return nil
}

func (g *Game) clearOrder() { g.ord = order{} }

// begin starts an order for an ally that can still act.
func (g *Game) begin(u core.UnitView) {
	g.ord = order{actor: u.ID, stage: "move"}
	if u.Moved {
		g.ord.stage = "menu"
	}
}

// cancel steps the order back one stage (right click or Esc).
func (g *Game) cancel() bool {
	switch g.ord.stage {
	case "target", "sub":
		g.ord.stage, g.ord.sub = "menu", ""
	case "menu":
		u := g.unitView(g.S.Engine.Observe(), g.ord.actor)
		if g.ord.move == nil && u != nil && u.Moved {
			g.clearOrder()
			return true
		}
		g.ord.sandbox, g.ord.move, g.ord.stage = nil, nil, "move"
		g.player.Skip()
	case "move":
		g.clearOrder()
	default:
		return false
	}
	return true
}

// tentative moves the actor in a sandbox and walks it there.
func (g *Game) tentative(x, y int) {
	c := core.Command{Kind: "move", Actor: g.ord.actor, X: x, Y: y}
	sb, err := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err != nil {
		g.toast(err.Error(), bad)
		return
	}
	r, err := sb.Apply(c)
	if err != nil {
		g.toast(koreanError(err.Error()), bad)
		return
	}
	a := replay.New(g.S.Engine.Observe(), c.Actor)
	a.Add(c, r.Events)
	g.ord.sandbox, g.ord.move, g.ord.stage = sb, &c, "menu"
	g.player.Play([]*replay.Action{a}, sb.Observe())
}

// confirm sends the tentative move followed by the action that completes the unit's turn.
func (g *Game) confirm(c core.Command) {
	move := g.ord.move
	g.clearOrder()
	if move != nil {
		if r := g.send(session.Request{Op: "command", Command: *move}); !r.OK {
			return
		}
	}
	if c.Kind == "wait" {
		g.send(session.Request{Op: "command", Command: c})
		return
	}
	g.request(session.Request{Op: "command", Command: c})
}

// options are the actor's legal commands of the picked kind.
func (g *Game) options(pick core.Command) []core.Command {
	out := []core.Command{}
	for _, c := range g.engine().LegalActions(g.ord.actor).Commands {
		if c.Kind == pick.Kind && c.Skill == pick.Skill && c.Item == pick.Item {
			out = append(out, c)
		}
	}
	return out
}

// mapClick handles a left click on the battlefield cell (x, y).
func (g *Game) mapClick(x, y int) {
	o := g.S.Engine.Observe()
	var here *core.UnitView
	for i := range o.UnitViews {
		if u := &o.UnitViews[i]; u.HP > 0 && u.X == x && u.Y == y && u.ID != g.ord.actor {
			here = u
		}
	}
	if g.ord.actor != "" {
		if a := g.unitView(g.engine().Observe(), g.ord.actor); a != nil && a.X == x && a.Y == y && g.ord.stage != "target" {
			if g.ord.stage == "move" {
				g.ord.stage = "menu"
			}
			return
		}
	}
	switch g.ord.stage {
	case "move":
		if g.S.Engine.Deploying() && containsCell(o.Map.Allies, x, y) {
			g.command(core.Command{Kind: "place", Actor: g.ord.actor, X: x, Y: y})
			g.clearOrder()
			return
		}
		for _, c := range g.S.Engine.LegalActions(g.ord.actor).Commands {
			if c.Kind == "move" && c.X == x && c.Y == y {
				g.tentative(x, y)
				return
			}
		}
	case "target":
		for _, c := range g.options(g.ord.pick) {
			if t := g.unitView(g.engine().Observe(), c.Target); t != nil && t.X == x && t.Y == y {
				g.confirm(c)
				return
			}
		}
		return
	case "menu", "sub":
		g.cancel()
		return
	}
	g.clearOrder()
	if here == nil {
		g.selected = ""
		return
	}
	g.selected = here.ID
	if here.Faction == "ally" && o.Turn == "ally" && !here.Done {
		g.begin(*here)
	}
}

type menuItem struct {
	label string
	click func()
}

// entry is an action-menu line with the explanation shown while it is hovered.
type entry struct {
	menuItem
	help string
}

// menuItems is the action menu: actions that are unavailable stay listed, greyed out.
// Duels are not a command: they start when the paired officers attack each other.
func (g *Game) menuItems() []entry {
	legal := g.engine().LegalActions(g.ord.actor).Commands
	has := map[string]bool{}
	for _, c := range legal {
		has[c.Kind] = true
	}
	pick := func(kind string) func() {
		if !has[kind] {
			return nil
		}
		return func() { g.ord.pick, g.ord.stage = core.Command{Kind: kind}, "target" }
	}
	sub := func(kind string) func() {
		if !has[kind] {
			return nil
		}
		return func() { g.ord.sub, g.ord.stage = kind, "sub" }
	}
	attack := "무기로 공격합니다. 정해진 상대와 맞붙으면 일기토가 벌어집니다."
	if a := g.unitView(g.engine().Observe(), g.ord.actor); a != nil {
		lo, hi := g.engine().AttackRange(a.ID)
		attack = fmt.Sprintf("무기로 공격합니다(사거리 %d–%d).\n정해진 상대와 맞붙으면 일기토가 벌어집니다.", lo, hi)
	}
	return []entry{
		{menuItem{"공격", pick("attack")}, attack},
		{menuItem{"병법 ›", sub("skill")}, "병법치를 써서 병법을 사용합니다."},
		{menuItem{"도구 ›", sub("item")}, "군량·병법단 등 도구를 자신이나 인접 부대에 씁니다."},
		{menuItem{"대기", func() { g.confirm(core.Command{Kind: "wait", Actor: g.ord.actor}) }}, "이번 턴 행동을 마칩니다."},
	}
}

// subItems lists skills, items or traits; each opens targeting or runs at once.
func (g *Game) subItems() []entry {
	e := g.engine()
	legal := e.LegalActions(g.ord.actor).Commands
	o := e.Observe()
	u := g.unitView(o, g.ord.actor)
	out := []entry{}
	switch g.ord.sub {
	case "skill":
		ok := map[string]bool{}
		for _, c := range legal {
			if c.Kind == "skill" {
				ok[c.Skill] = true
			}
		}
		for _, id := range u.Skills {
			s := g.S.Data.Skills[id]
			label := fmt.Sprintf("%s  %d", s.Name, s.Cost)
			var click func()
			if ok[id] {
				pick := core.Command{Kind: "skill", Skill: id}
				click = func() {
					if s.Mode == content.ModeSelf {
						g.confirm(g.options(pick)[0])
						return
					}
					g.ord.pick, g.ord.stage = pick, "target"
				}
			}
			out = append(out, entry{menuItem{label, click}, g.skillHelp(id, u)})
		}
	case "item":
		ok := map[string]bool{}
		for _, c := range legal {
			if c.Kind == "item" {
				ok[c.Item] = true
			}
		}
		for _, id := range sorted(o.Inventory) {
			if o.Inventory[id] < 1 {
				continue
			}
			pick := core.Command{Kind: "item", Item: id}
			var click func()
			if ok[id] {
				click = func() { g.ord.pick, g.ord.stage = pick, "target" }
			}
			out = append(out, entry{menuItem{fmt.Sprintf("%s ×%d", g.itemName(id), o.Inventory[id]), click}, g.itemHelp(id)})
		}
	}
	if len(out) == 0 {
		out = append(out, entry{menuItem: menuItem{"(없음)", nil}})
	}
	return out
}

// drawMenu draws the action menu (and an open sub-list) beside the acting unit.
func (g *Game) drawMenu(dst *ebiten.Image) {
	if g.ord.stage != "menu" && g.ord.stage != "sub" || g.busy() {
		return
	}
	v := g.vis[g.ord.actor]
	if v == nil {
		return
	}
	cx, cy := g.field.Center(v.u, v.v)
	sx, sy := g.toScreen(cx, cy-v.lift)
	x := int(sx + 16*g.zoom)
	help := ""
	var helpAt image.Rectangle
	draw := func(items []entry, x, y, w int, title string) image.Rectangle {
		h := 14 + len(items)*30
		if title != "" {
			h += 22
		}
		x = max(8, min(Width-w-8, x))
		y = max(48, min(Height-h-8, y))
		r := image.Rect(x, y, x+w, y+h)
		window(dst, r)
		g.block(r)
		top := y + 7
		if title != "" {
			g.label(dst, title, float64(x+12), float64(y+5), 13, gold)
			top += 22
		}
		for i, it := range items {
			b := image.Rect(x+7, top+i*30, x+w-7, top+i*30+28)
			g.btn(dst, b, it.label, it.click)
			if hovered(b) && it.help != "" {
				help, helpAt = it.help, image.Rect(r.Max.X+6, b.Min.Y, r.Max.X+6, b.Min.Y)
			}
		}
		return r
	}
	r := draw(g.menuItems(), x, int(sy)-110, 128, g.name(g.ord.actor))
	if g.ord.stage == "sub" {
		title := map[string]string{"skill": "병법 · 소모", "item": "도구"}[g.ord.sub]
		draw(g.subItems(), r.Max.X+6, r.Min.Y+20, 178, title)
	}
	g.label(dst, "우클릭/Esc: 취소", float64(r.Min.X), float64(r.Max.Y+4), 12, muted)
	if help != "" {
		g.tooltip(dst, help, helpAt.Min.X, helpAt.Min.Y)
	}
}

// marks are the cells tinted on the field for the current order or selection.
func (g *Game) marks(o core.Observation) []Highlight {
	out := []Highlight{}
	add := func(p content.Point, c color.RGBA) { out = append(out, Highlight{p.X, p.Y, c}) }
	if g.threatAll {
		for _, p := range g.threatUnion(o) {
			add(p, color.RGBA{235, 40, 30, 92})
		}
	}
	if sel := g.unitView(o, g.selected); sel != nil && sel.HP > 0 && sel.Faction == "enemy" && g.ord.actor == "" {
		for _, p := range g.threat(sel.ID) {
			add(p, color.RGBA{240, 60, 30, 110})
		}
		add(content.Point{X: sel.X, Y: sel.Y}, color.RGBA{255, 120, 90, 150})
	}
	if g.S.Engine.Deploying() && !g.autoplay && (g.ord.actor == "" || g.ord.stage == "move") {
		for _, p := range o.Map.Allies {
			add(p, color.RGBA{240, 196, 80, 90})
		}
	}
	if g.busy() || g.ord.actor == "" {
		return out
	}
	e := g.engine()
	eo := e.Observe()
	a := g.unitView(eo, g.ord.actor)
	if a == nil {
		return out
	}
	switch g.ord.stage {
	case "move":
		for _, c := range g.S.Engine.LegalActions(g.ord.actor).Commands {
			if c.Kind == "move" {
				add(content.Point{X: c.X, Y: c.Y}, color.RGBA{60, 130, 230, 110})
			}
		}
	case "target":
		lo, hi := g.reach(*a, g.ord.pick)
		for y := a.Y - hi; y <= a.Y+hi; y++ {
			for x := a.X - hi; x <= a.X+hi; x++ {
				d := abs(x-a.X) + abs(y-a.Y)
				if d >= lo && d <= hi && g.field.Inside(x, y) && g.field.Tile(x, y) != '~' {
					add(content.Point{X: x, Y: y}, color.RGBA{230, 90, 60, 60})
				}
			}
		}
		for _, c := range g.options(g.ord.pick) {
			if t := g.unitView(eo, c.Target); t != nil {
				add(content.Point{X: t.X, Y: t.Y}, color.RGBA{240, 70, 50, 150})
			}
		}
		for _, p := range g.areaMarks(e, eo, *a) {
			add(p.cell, p.color)
		}
	}
	add(content.Point{X: a.X, Y: a.Y}, color.RGBA{255, 214, 96, 150})
	return out
}

func containsCell(cells []content.Point, x, y int) bool {
	for _, p := range cells {
		if p.X == x && p.Y == y {
			return true
		}
	}
	return false
}

// reach is the distance band a picked action covers, as the core measures it (traits that
// lengthen a spell's reach are not shown).
func (g *Game) reach(a core.UnitView, pick core.Command) (int, int) {
	lo, hi := g.engine().AttackRange(a.ID)
	switch pick.Kind {
	case "item":
		return 0, 1
	case "skill":
		s := g.S.Data.Skills[pick.Skill]
		if s.Mode == content.ModeSelf {
			return 0, 0
		}
		if s.Max > 0 {
			lo, hi = s.Min, s.Max+a.Status["range"].Value
		} else {
			lo = max(lo, s.Min)
		}
	}
	return lo, hi
}

type areaMark struct {
	cell  content.Point
	color color.RGBA
}

// areaMarks tints the cells the picked skill covers around the cell under the cursor: the
// 3x3 of a square, the cross, the line from the caster toward the target; units it hits are
// tinted stronger (the core's preview names them).
func (g *Game) areaMarks(e *core.Engine, o core.Observation, a core.UnitView) []areaMark {
	if !g.hover.ok || g.ord.pick.Kind != "skill" {
		return nil
	}
	var c *core.Command
	for _, opt := range g.options(g.ord.pick) {
		if t := g.unitView(o, opt.Target); t != nil && t.X == g.hover.x && t.Y == g.hover.y {
			cmd := opt
			c = &cmd
		}
	}
	if c == nil {
		return nil
	}
	s := g.S.Data.Skills[c.Skill]
	shape := s.Shape
	if shape == content.ShapeSingle && s.Kind != content.KindPhysical && s.Mode != content.ModeSelf && a.Status["eightfold"].Turns > 0 {
		shape = content.ShapeSquare
	}
	hx, hy := g.hover.x, g.hover.y
	var cells []content.Point
	switch shape {
	case content.ShapeSquare:
		for y := hy - 1; y <= hy+1; y++ {
			for x := hx - 1; x <= hx+1; x++ {
				cells = append(cells, content.Point{X: x, Y: y})
			}
		}
	case content.ShapeCross:
		cells = []content.Point{{X: hx, Y: hy}, {X: hx + 1, Y: hy}, {X: hx - 1, Y: hy}, {X: hx, Y: hy + 1}, {X: hx, Y: hy - 1}}
	case content.ShapeLine:
		dx, dy := sign(hx-a.X), 0
		if abs(hy-a.Y) > abs(hx-a.X) {
			dx, dy = 0, sign(hy-a.Y)
		}
		for i := 1; i <= s.Length; i++ {
			cells = append(cells, content.Point{X: a.X + dx*i, Y: a.Y + dy*i})
		}
	}
	out := []areaMark{}
	for _, p := range cells {
		if g.field.Inside(p.X, p.Y) {
			out = append(out, areaMark{p, color.RGBA{255, 170, 60, 90}})
		}
	}
	if p, err := e.Preview(*c); err == nil {
		for _, id := range p.Targets {
			if t := g.unitView(o, id); t != nil {
				out = append(out, areaMark{content.Point{X: t.X, Y: t.Y}, color.RGBA{255, 120, 40, 140}})
			}
		}
	}
	return out
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (g *Game) threat(id string) []content.Point {
	key := fmt.Sprint(g.S.Engine.Observe().Revision, id)
	if p, ok := g.threats[key]; ok {
		return p
	}
	if len(g.threats) > 64 {
		g.threats = map[string][]content.Point{}
	}
	p := g.S.Engine.Threat(id)
	g.threats[key] = p
	return p
}

func (g *Game) threatUnion(o core.Observation) []content.Point {
	set := map[content.Point]bool{}
	for _, u := range o.UnitViews {
		if u.Faction == "enemy" && u.HP > 0 {
			for _, p := range g.threat(u.ID) {
				set[p] = true
			}
		}
	}
	out := make([]content.Point, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Y*1000+out[i].X < out[j].Y*1000+out[j].X })
	return out
}

// forecast shows the expected outcome when the cursor is on a valid target: each side's
// 병력, 병법치 and 경험치 after a normal and a critical blow, and the odds. The panel sits
// beside the target, clear of every unit, and never takes the cursor from the map.
func (g *Game) forecast(dst *ebiten.Image) {
	if g.ord.stage != "target" || !g.hover.ok || g.busy() {
		return
	}
	e := g.engine()
	o := e.Observe()
	var c *core.Command
	for _, opt := range g.options(g.ord.pick) {
		if t := g.unitView(o, opt.Target); t != nil && t.X == g.hover.x && t.Y == g.hover.y {
			cmd := opt
			c = &cmd
		}
	}
	if c == nil {
		return
	}
	p, err := e.Preview(*c)
	if err != nil {
		return
	}
	a, t := g.unitView(o, c.Actor), g.unitView(o, c.Target)
	duel := e.DuelFor(*c)
	counter := 0
	dmg := p.MaxDamage
	if !duel {
		counter = e.CounterDamage(*c) * 2 / 3 // CounterDamage's maximum includes the critical blow
		if c.Kind == "attack" || c.Kind == "skill" && g.S.Data.Skills[c.Skill].Kind == "physical" {
			dmg = p.MaxDamage * 2 / 3 // Preview's maximum includes the critical blow
		}
	} else {
		dmg, p.MaxDamage, p.Cost, p.Healing, p.MPRecovery, p.XP, p.CritXP = 0, 0, 0, 0, 0, 0, 0
	}
	lines := []string{}
	switch {
	case duel:
		lines = append(lines, "일기토 발생! 공격 대신 일기토 결과가 적용됩니다.")
	case dmg >= t.HP:
		lines = append(lines, "격파!")
	case p.MaxDamage >= t.HP:
		lines = append(lines, "치명타 시 격파!")
	}
	if p.Effect != "" && p.MaxDamage == 0 && p.Healing == 0 {
		if effect := statusLabels[p.Effect]; effect != "" {
			lines = append(lines, "효과: "+effect)
		}
	}
	if counter > 0 {
		lines = append(lines, fmt.Sprintf("반격을 받음 (최대 %d)", e.CounterDamage(*c)))
	}
	if len(p.Targets) > 1 {
		lines = append(lines, "대상 "+strings.Join(names(g, p.Targets), " · "))
	}
	self := t.ID == a.ID
	w, h := 336, 58+len(lines)*18
	h += sideHeight(a)
	if !self {
		h += sideHeight(t) + 8
	}
	r := g.besideTarget(o, t, w, h)
	title := map[string]string{"attack": "공격", "skill": g.skillName(c.Skill), "item": g.itemName(c.Item)}[c.Kind]
	g.titled(dst, r, title)
	x, y := r.Min.X+16, r.Min.Y+18
	if self {
		y = g.side(dst, a, x, y, sideChange{hp: [2]int{p.Healing, p.Healing}, mp: p.MPRecovery - p.Cost, xp: [2]int{p.XP, p.CritXP}})
	} else {
		// a counter only comes back when the target is still standing
		back := [2]int{-counter, -counter}
		if dmg >= t.HP {
			back[0] = 0
		}
		if p.MaxDamage >= t.HP {
			back[1] = 0
		}
		y = g.side(dst, a, x, y, sideChange{hp: back, mp: -p.Cost, xp: [2]int{p.XP, p.CritXP}})
		rect(dst, float32(x), float32(y+3), float32(w-32), 1, goldDim)
		y = g.side(dst, t, x, y+8, sideChange{hp: [2]int{p.Healing - dmg, p.Healing - p.MaxDamage}, mp: p.MPRecovery})
	}
	rect(dst, float32(x), float32(y+3), float32(w-32), 1, goldDim)
	if !duel {
		odds := fmt.Sprintf("명중 %.0f%%", p.Hit)
		if c.Kind == "item" {
			odds = "효과 확정"
		} else if p.Crit > 0 {
			odds += fmt.Sprintf("     치명 %.0f%%", p.Crit)
		}
		if p.Double > 0 {
			odds += fmt.Sprintf("     2회 %.0f%%", p.Double)
		}
		g.label(dst, odds, float64(x), float64(y+8), 16, gold)
	}
	g.label(dst, strings.Join(lines, "\n"), float64(x), float64(y+34), 12, ink)
}

// sideChange is what an action does to one unit: 병력 after a normal and a critical
// blow, 병법치, and raw experience after a normal and a critical blow.
type sideChange struct {
	hp [2]int
	mp int
	xp [2]int
}

func sideHeight(u *core.UnitView) int {
	if u.Faction == "ally" {
		return 84
	}
	return 64
}

// side is one unit's half of the forecast: name, level and gauges with the change marked.
// It returns the y below it.
func (g *Game) side(dst *ebiten.Image, u *core.UnitView, x, y int, ch sideChange) int {
	team := color.NRGBA{R: 120, G: 180, B: 240, A: 255}
	if u.Faction == "enemy" {
		team = bad
	}
	g.label(dst, u.Name, float64(x), float64(y), 16, team)
	g.label(dst, fmt.Sprintf("Lv%d  %s", u.Level, g.className(u.Class)), float64(x)+g.width(u.Name, 16)+8, float64(y+3), 12, muted)
	y += 26
	clamp := func(v, hi int) int { return max(0, min(hi, v)) }
	hp := u.Stats.MaxHP
	g.gauge(dst, x, y, "병력", u.HP, clamp(u.HP+ch.hp[0], hp), clamp(u.HP+ch.hp[1], hp), hp, hpColor(float64(u.HP)/float64(max(1, hp))), "")
	mp := clamp(u.MP+ch.mp, u.Stats.MaxMP)
	g.gauge(dst, x, y+20, "병법치", u.MP, mp, mp, u.Stats.MaxMP, mpC, "")
	if u.Faction == "ally" {
		now := int(u.XPProgress)
		gain := func(n int) int { return now + n*100/max(1, u.XPRequired) }
		g.gauge(dst, x, y+40, "경험치", now, gain(ch.xp[0]), gain(ch.xp[1]), 100, xpC, "LV UP")
		return y + 58
	}
	return y + 38
}

// gauge draws one forecast bar: the value now and after a normal and a critical blow.
// Losses are red (the extra a critical blow takes darker), gains pale. full replaces a
// value that reaches maxV (a level up).
func (g *Game) gauge(dst *ebiten.Image, x, y int, name string, now, normal, crit, maxV int, c color.NRGBA, full string) {
	g.label(dst, name, float64(x), float64(y-3), 12, muted)
	bx, bw := float32(x+48), float32(120)
	at := func(v int) float32 { return bx + 1 + (bw-2)*float32(min(v, maxV))/float32(max(1, maxV)) }
	span := func(from, to int, c color.Color) {
		if to < from {
			from, to = to, from
		}
		rect(dst, at(from), float32(y+1), at(to)-at(from), 8, c)
	}
	bar(dst, bx, float32(y), bw, 10, float64(min(now, normal, crit))/float64(max(1, maxV)), c)
	if crit < normal {
		span(crit, normal, color.NRGBA{R: 150, G: 30, B: 24, A: 255})
	}
	if normal < now {
		span(normal, now, color.NRGBA{R: 240, G: 70, B: 50, A: 255})
	}
	if normal > now {
		span(now, normal, color.NRGBA{R: 200, G: 245, B: 170, A: 255})
	}
	if crit > normal {
		span(normal, crit, color.NRGBA{R: 255, G: 250, B: 210, A: 255})
	}
	show := func(v int) string {
		if full != "" && v >= maxV {
			return full
		}
		return fmt.Sprint(v)
	}
	s, col := fmt.Sprintf("%d / %d", now, maxV), ink
	if full != "" {
		s = fmt.Sprint(now)
	}
	if normal != now || crit != now {
		s, col = show(now)+" → "+show(normal), gold
		if crit != normal {
			s += "  (치명 " + show(crit) + ")"
		}
	}
	g.label(dst, s, float64(bx+bw+8), float64(y-3), 12, col)
}

// unitBox is the screen area a unit's sprite covers.
func (g *Game) unitBox(v *unitVis) image.Rectangle {
	cx, cy := g.field.Center(v.u, v.v)
	sx, sy := g.toScreen(cx, cy-v.lift)
	half, top := 16*K*g.zoom, (v.top+8)*g.zoom
	return image.Rect(int(sx-half), int(sy-top), int(sx+half), int(sy+8*K*g.zoom))
}

// besideTarget places a w×h panel to the right of the target, else to its left, moving
// it up or down until it covers no unit; failing that, no unit but the HUD may be covered.
func (g *Game) besideTarget(o core.Observation, t *core.UnitView, w, h int) image.Rectangle {
	boxes := []image.Rectangle{}
	for _, u := range o.UnitViews {
		if v := g.vis[u.ID]; v != nil && u.HP > 0 {
			boxes = append(boxes, g.unitBox(v))
		}
	}
	tb := image.Rect(Width/2, Height/2, Width/2, Height/2)
	if v := g.vis[t.ID]; v != nil {
		tb = g.unitBox(v)
	}
	const gap = 10
	clear := func(r image.Rectangle, avoid []image.Rectangle) bool {
		for _, b := range avoid {
			if r.Overlaps(b) {
				return false
			}
		}
		return true
	}
	mid := (tb.Min.Y+tb.Max.Y)/2 - h/2
	for _, hud := range []bool{false, true} {
		avoid := boxes
		if !hud {
			avoid = append(append([]image.Rectangle{}, boxes...), g.blocks...)
		}
		for dy := 0; dy <= Height; dy += 24 {
			for _, y := range []int{mid - dy, mid + dy} {
				y = max(48, min(Height-h-8, y))
				for _, x := range []int{tb.Max.X + gap, tb.Min.X - gap - w} {
					if r := image.Rect(x, y, x+w, y+h); x >= 8 && x+w <= Width-8 && clear(r, avoid) {
						return r
					}
				}
			}
		}
	}
	x := tb.Max.X + gap
	if x+w > Width-8 {
		x = tb.Min.X - gap - w
	}
	x = max(8, min(Width-w-8, x))
	y := max(48, min(Height-h-8, mid))
	return image.Rect(x, y, x+w, y+h)
}

func names(g *Game, ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = g.name(id)
	}
	return out
}
