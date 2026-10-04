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
	sub     string       // skill, item, learn
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

// confirm sends the tentative move and the action, then ends the unit's turn unless it
// can still act (traits such as 국사무쌍) or it only learned a trait.
func (g *Game) confirm(c core.Command) {
	actor, move := g.ord.actor, g.ord.move
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
	if !g.request(session.Request{Op: "command", Command: c}).OK {
		return
	}
	o := g.S.Engine.Observe()
	u := g.unitView(o, actor)
	if o.Phase != "battle" || u == nil || u.HP <= 0 || u.Done {
		return
	}
	if c.Kind == "learn" || g.canStillAct(actor) {
		g.ord = order{actor: actor, stage: "menu"}
		return
	}
	g.send(session.Request{Op: "command", Command: core.Command{Kind: "wait", Actor: actor}})
}

func (g *Game) canStillAct(actor string) bool {
	for _, c := range g.S.Engine.LegalActions(actor).Commands {
		if c.Kind == "attack" || c.Kind == "skill" || c.Kind == "item" || c.Kind == "duel" {
			return true
		}
	}
	return false
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

// menuItems is the action menu: actions that are unavailable stay listed, greyed out.
func (g *Game) menuItems() []menuItem {
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
	items := []menuItem{{"공격", pick("attack")}, {"병법 ›", sub("skill")}, {"도구 ›", sub("item")}}
	if has["duel"] {
		items = append(items, menuItem{"일기토", pick("duel")})
	}
	items = append(items, menuItem{"학습 ›", sub("learn")})
	items = append(items, menuItem{"대기", func() { g.confirm(core.Command{Kind: "wait", Actor: g.ord.actor}) }})
	return items
}

// subItems lists skills, items or traits; each opens targeting or runs at once.
func (g *Game) subItems() []menuItem {
	e := g.engine()
	legal := e.LegalActions(g.ord.actor).Commands
	o := e.Observe()
	u := g.unitView(o, g.ord.actor)
	out := []menuItem{}
	switch g.ord.sub {
	case "skill":
		ok := map[string]bool{}
		for _, c := range legal {
			if c.Kind == "skill" {
				ok[c.Skill] = true
			}
		}
		for _, name := range u.Skills {
			s := g.S.Data.Skills[name]
			label := fmt.Sprintf("%s  %d", name, s.Cost)
			var click func()
			if ok[name] {
				pick := core.Command{Kind: "skill", Skill: name}
				click = func() {
					if s.Mode == "자기" {
						g.confirm(g.options(pick)[0])
						return
					}
					g.ord.pick, g.ord.stage = pick, "target"
				}
			}
			out = append(out, menuItem{label, click})
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
			out = append(out, menuItem{fmt.Sprintf("%s ×%d", id, o.Inventory[id]), click})
		}
	case "learn":
		for _, c := range legal {
			if c.Kind == "learn" {
				cmd := c
				cost := g.S.Data.Traits[c.Trait].Cost
				out = append(out, menuItem{fmt.Sprintf("%s  %d", c.Trait, cost), func() { g.confirm(cmd) }})
			}
		}
	}
	if len(out) == 0 {
		out = append(out, menuItem{"(없음)", nil})
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
	x := int(sx) + 18*g.zoom
	draw := func(items []menuItem, x, y, w int, title string) image.Rectangle {
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
			g.btn(dst, image.Rect(x+7, top+i*30, x+w-7, top+i*30+28), it.label, it.click)
		}
		return r
	}
	r := draw(g.menuItems(), x, int(sy)-110, 128, g.name(g.ord.actor))
	if g.ord.stage == "sub" {
		title := map[string]string{"skill": "병법 · 소모", "item": "도구", "learn": "학습 · 특성치"}[g.ord.sub]
		draw(g.subItems(), r.Max.X+6, r.Min.Y+20, 178, title)
	}
	g.label(dst, "우클릭/Esc: 취소", float64(r.Min.X), float64(r.Max.Y+4), 12, muted)
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
	}
	add(content.Point{X: a.X, Y: a.Y}, color.RGBA{255, 214, 96, 150})
	return out
}

// reach is the distance band a picked action covers, as the core measures it.
func (g *Game) reach(a core.UnitView, pick core.Command) (int, int) {
	class := g.S.Data.Classes[a.Class]
	lo, hi := 1, class.Range
	switch pick.Kind {
	case "item", "duel":
		return 0, 1
	case "skill":
		s := g.S.Data.Skills[pick.Skill]
		lo, hi = s.Min, s.Max
		if hi == 0 {
			hi = class.Range
		}
	}
	if a.Status["range"].Turns > 0 {
		hi++
	}
	return lo, hi
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

// forecast shows the expected outcome when the cursor is on a valid target.
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
	w, h := 470, 196
	x, y := Width/2-w/2, 52
	r := image.Rect(x, y, x+w, y+h)
	title := map[string]string{"attack": "공격", "skill": c.Skill, "item": c.Item, "duel": "일기토"}[c.Kind]
	g.titled(dst, r, title+" 예측")
	g.block(r)
	side := func(u *core.UnitView, px int, dmg int, heal int) {
		g.portrait(dst, u.ID, float64(px), float64(y+22), 56)
		g.label(dst, u.Name, float64(px+66), float64(y+20), 17, ink)
		g.label(dst, fmt.Sprintf("%s · %s", u.Class, terrainNames[g.field.Tile(u.X, u.Y)]), float64(px+66), float64(y+44), 12, muted)
		k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
		bx, by := float32(px), float32(y+86)
		bar(dst, bx, by, 190, 10, k, hpColor(k))
		if dmg > 0 {
			lost := float64(min(u.HP, dmg)) / float64(max(1, u.Stats.MaxHP))
			rect(dst, bx+1+188*float32(k-lost), by+1, 188*float32(lost), 8, color.NRGBA{R: 240, G: 70, B: 50, A: 220})
		}
		if heal > 0 {
			gain := float64(min(u.Stats.MaxHP-u.HP, heal)) / float64(max(1, u.Stats.MaxHP))
			rect(dst, bx+1+188*float32(k), by+1, 188*float32(gain), 8, color.NRGBA{R: 160, G: 240, B: 150, A: 220})
		}
		g.label(dst, fmt.Sprintf("병력 %d / %d", u.HP, u.Stats.MaxHP), float64(px), float64(y+98), 12, muted)
	}
	counter := e.CounterDamage(*c)
	dmg := p.MaxDamage
	if c.Kind == "attack" || c.Kind == "skill" && g.S.Data.Skills[c.Skill].Kind == "physical" {
		dmg = p.MaxDamage * 2 / 3 // Preview's maximum includes the critical blow
	}
	side(a, x+18, counter, 0)
	if t.ID != a.ID {
		side(t, x+w/2+14, dmg, p.Healing)
	}
	lines := []string{}
	switch {
	case c.Kind == "duel":
		lines = append(lines, "일기토 결과: "+p.Effect)
	case p.Healing > 0:
		lines = append(lines, fmt.Sprintf("회복 +%d", p.Healing))
	case p.MaxDamage > 0:
		s := fmt.Sprintf("명중 %.0f%%   피해 약 %d", p.Hit, dmg)
		if dmg != p.MaxDamage {
			s += fmt.Sprintf(" (치명 %d)", p.MaxDamage)
		}
		lines = append(lines, s)
	default:
		effect := statusNames[p.Effect]
		if effect == "" {
			effect = p.Effect
		}
		lines = append(lines, fmt.Sprintf("성공 %.0f%%   효과 %s", p.Hit, effect))
	}
	if p.Cost > 0 {
		lines[0] += fmt.Sprintf("   병법치 -%d", p.Cost)
	}
	if counter > 0 {
		lines = append(lines, fmt.Sprintf("반격 받음: 최대 %d", counter))
	} else if c.Kind == "attack" || c.Kind == "skill" {
		lines = append(lines, "반격 없음")
	}
	if len(p.Targets) > 1 {
		lines = append(lines, "대상 "+strings.Join(names(g, p.Targets), " · "))
	}
	g.label(dst, strings.Join(lines, "\n"), float64(x+18), float64(y+124), 15, ink)
}

func names(g *Game, ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = g.name(id)
	}
	return out
}
