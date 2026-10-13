package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

type hoverCell struct {
	x, y int
	ok   bool
}

type toast struct {
	text string
	col  color.Color
	t    float64
}

const toastTime = 3.5

func (g *Game) toast(s string, c color.Color) {
	if s == "" {
		return
	}
	g.toasts = append(g.toasts, &toast{text: s, col: c})
	if len(g.toasts) > 4 {
		g.toasts = g.toasts[len(g.toasts)-4:]
	}
}

// updateHover finds the battlefield cell under the cursor (not under a HUD panel).
func (g *Game) updateHover() {
	g.hover = hoverCell{}
	if g.field == nil || g.popupPending() {
		return
	}
	x, y := cursorPos()
	if g.overHUD(x, y) || !image.Pt(x, y).In(g.battleViewport()) {
		return
	}
	cx, cy := g.toCanvas(float64(x), float64(y))
	mx, my := g.field.Cell(cx, cy)
	if g.field.Inside(mx, my) {
		g.hover = hoverCell{mx, my, true}
	}
}

// cursor outlines the hovered cell at ground level, even on a lifted castle wall.
func (g *Game) cursor(dst *ebiten.Image) {
	if !g.hover.ok {
		return
	}
	cx, cy := g.field.Center(float64(g.hover.x), float64(g.hover.y))
	pts := [][2]float64{{cx, cy - 8*K}, {cx + 16*K, cy}, {cx, cy + 8*K}, {cx - 16*K, cy}}
	pulse := uint8(180 + 60*math.Sin(g.clockMS/160))
	if g.deploying() {
		pulse = uint8(150 + 105*(deploymentPulse(g.uiSeconds)-.12)/.18)
	}
	for i := range pts {
		ax, ay := g.toScreen(pts[i][0], pts[i][1])
		bx, by := g.toScreen(pts[(i+1)%4][0], pts[(i+1)%4][1])
		strokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), 2, color.NRGBA{R: 255, G: 240, B: 190, A: pulse})
	}
}

func (g *Game) objective(st *coreStage) (win, lose string) {
	lose = g.lordName() + " 퇴각"
	if st.turnLimit > 0 {
		lose += fmt.Sprintf(" / %d턴 초과", st.turnLimit)
	}
	if st.goal == "rout" {
		return "적군 전멸", lose
	}
	win = st.boss + " 격파"
	if st.threshold > 0 {
		win = fmt.Sprintf("%s 병력 %d%% 이하", st.boss, st.threshold)
	}
	return win, lose
}

// lordName is the first lord of the starting roster, whose fall loses a battle.
func (g *Game) lordName() string {
	for _, id := range g.S.Data.Campaign.Roster {
		if g.S.Data.Characters[id].Lord {
			return g.name(id)
		}
	}
	return "아군 지휘관"
}

type coreStage struct {
	name, boss, goal     string
	threshold, turnLimit int
}

func (g *Game) stageInfo(o core.Observation) *coreStage {
	st := g.S.Data.Stages[o.Stage]
	return &coreStage{st.Name, g.name(st.Boss), st.Goal, st.Threshold, st.TurnLimit}
}

func (g *Game) pending(o core.Observation) (left, total int) {
	for _, u := range o.UnitViews {
		if u.Faction == "ally" && !u.AI && u.HP > 0 {
			total++
			if !u.Done {
				left++
			}
		}
	}
	return left, total
}

// topBar shows the battle, the objective and how many allies have yet to act.
func (g *Game) topBar(dst *ebiten.Image, o core.Observation) {
	r := image.Rect(0, 0, Width, 40)
	rect(dst, 0, 0, Width, 40, color.NRGBA{R: 20, G: 14, B: 12, A: 225})
	rect(dst, 0, 39, Width, 2, gold)
	g.block(r)
	st := g.stageInfo(o)
	g.label(dst, st.name, 16, 8, 18, gold)
	turn := color.NRGBA{R: 120, G: 180, B: 240, A: 255}
	if o.Turn == "support" {
		turn = good
	}
	if o.Turn == "enemy" {
		turn = bad
	}
	x := 24 + g.width(st.name, 18)
	turnText := fmt.Sprintf("%d턴", o.Round)
	if st.turnLimit > 0 {
		turnText = fmt.Sprintf("%d/%d턴", o.Round, st.turnLimit)
	}
	g.label(dst, turnText, x, 9, 16, ink)
	g.label(dst, factionName(o.Turn)+" 진영", x+g.width(turnText, 16)+12, 9, 16, turn)
	win, lose := g.objective(st)
	g.label(dst, "승리  "+win+"     패배  "+lose, x+180, 11, 12, muted)
	if o.Phase == "battle" {
		left, total := g.pending(o)
		g.label(dst, fmt.Sprintf("미행동 %d / %d", left, total), 905, 10, 15, ink)
	}
	if g.speed > 1 {
		g.label(dst, fmt.Sprintf("재생 ×%d", g.speed), 1030, 10, 15, gold)
	}
	g.btn(dst, image.Rect(1176, 5, 1272, 34), "메뉴", func() { g.openOverlay("menu") })
}

// dock holds the battle-wide commands in the lower right corner.
func (g *Game) dock(dst *ebiten.Image, o core.Observation) {
	type entry struct {
		label string
		click func()
	}
	idle := !g.busy() && o.Phase == "battle"
	entries := []entry{
		{"부대 일람 U", func() { g.openOverlay("units") }},
		{"위협 범위 T" + map[bool]string{true: " ●", false: ""}[g.threatAll], func() { g.threatAll = !g.threatAll }},
		{"기록 L", func() { g.openOverlay("log") }},
		{"자동 진행 P" + map[bool]string{true: " ●", false: ""}[g.autoplay], func() { g.autoplay = !g.autoplay }},
	}
	if o.Turn == "ally" {
		var end func()
		if idle {
			end = g.endPhase
		}
		entries = append(entries, entry{"진영 종료 E", end})
	} else if o.Turn == "enemy" && g.S.Settings.AI != "auto" {
		var one, all func()
		if idle {
			one = func() { g.request(session.Request{Op: "ai"}) }
			all = g.resolvePhase
		}
		entries = append(entries, entry{"적 한 명령", one}, entry{"적 진영 진행", all})
	}
	w, h := 148, 30
	x := Width - w - 12
	y := Height - 12 - len(entries)*(h+4) - 12
	r := image.Rect(x-8, y-8, Width-4, Height-4)
	window(dst, r)
	g.block(r)
	for i, e := range entries {
		g.btn(dst, image.Rect(x, y+i*(h+4), x+w-4, y+i*(h+4)+h), e.label, e.click)
	}
}

// endPhase ends the allied phase, asking first when some allies have not acted.
func (g *Game) endPhase() {
	if g.popupPending() {
		return
	}
	if left, _ := g.pending(g.S.Engine.Observe()); left > 0 {
		g.ask(fmt.Sprintf("아직 행동하지 않은 부대가 %d개 있습니다.\n진영을 종료하시겠습니까?", left), g.endTurn)
		return
	}
	g.endTurn()
}

func (g *Game) endTurn() {
	if g.popupPending() {
		return
	}
	g.clearOrder()
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "end"}})
}

// infoUnit is the unit whose terrain the panel rates: the hovered one, else the acting or selected one.
func (g *Game) infoUnit(o core.Observation) *core.UnitView {
	if u := g.hoveredUnit(o); u != nil {
		return u
	}
	for _, id := range []string{g.ord.actor, g.selected} {
		if u := g.unitView(o, id); u != nil && u.HP > 0 {
			return u
		}
	}
	return nil
}

func (g *Game) hoveredUnit(o core.Observation) *core.UnitView {
	if !g.hover.ok {
		return nil
	}
	for i := range o.UnitViews {
		if u := &o.UnitViews[i]; u.HP > 0 && u.X == g.hover.x && u.Y == g.hover.y {
			return u
		}
	}
	return nil
}

func teamColor(u *core.UnitView) color.NRGBA {
	if u.AI && u.Faction == "ally" {
		return good
	}
	if u.Faction == "enemy" {
		return bad
	}
	return color.NRGBA{R: 120, G: 180, B: 240, A: 255}
}

// brief is the small card right of the cursor while it rests on a unit. While a target
// is being picked the forecast takes its place.
func (g *Game) brief(dst *ebiten.Image, o core.Observation) {
	u := g.hoveredUnit(o)
	if u == nil || g.ord.stage == "target" {
		return
	}
	w, h := 214, 82
	if u.Faction == "ally" {
		h += 16
	}
	x, y := cursorPos()
	x += 22
	if x+w > Width-8 {
		x -= w + 44
	}
	y = max(48, min(Height-h-8, y-h/2))
	window(dst, image.Rect(x, y, x+w, y+h))
	g.label(dst, u.Name, float64(x+14), float64(y+9), 15, teamColor(u))
	g.label(dst, fmt.Sprintf("Lv%d", u.Level), float64(x+22)+g.width(u.Name, 15), float64(y+12), 12, ink)
	row := func(i int, name string, now, maxV int, k float64, c color.Color) {
		ry := float64(y + 36 + i*16)
		g.label(dst, name, float64(x+14), ry-4, 11, muted)
		bar(dst, float32(x+60), float32(ry), 90, 7, k, c)
		g.label(dst, fmt.Sprintf("%d/%d", now, maxV), float64(x+156), ry-4, 11, ink)
	}
	k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
	row(0, "병력", u.HP, u.Stats.MaxHP, k, hpColor(k))
	row(1, "병법치", u.MP, u.Stats.MaxMP, float64(u.MP)/float64(max(1, u.Stats.MaxMP)), mpC)
	if u.Faction == "ally" {
		row(2, "경험치", int(u.XPProgress), 100, u.XPProgress/100, xpC)
	}
}

// detail describes the selected unit under the minimap: gauges, a hexagon of its unit
// stats against the battle's best, and its owned or learnable traits; a trait opens its
// explanation in the middle of the screen.
func (g *Game) detail(dst *ebiten.Image, o core.Observation) {
	if g.selected == "" {
		return
	}
	if g.ord.sandbox != nil {
		o = g.ord.sandbox.Observe()
	}
	u := g.unitView(o, g.selected)
	if u == nil || u.HP <= 0 {
		return
	}
	if g.traitOf != u.ID {
		g.traitOf, g.traitTop = u.ID, 0
		g.detailTab, g.detailPage = 0, 0
	}
	const w = 320
	x, y := Width-12-w, g.miniRect.Max.Y+10
	h := Height - 240 - y
	r := image.Rect(x, y, x+w, y+h)
	window(dst, r)
	g.block(r)
	g.portrait(dst, u.ID, float64(x+14), float64(y+14), 56)
	g.label(dst, u.Name, float64(x+82), float64(y+10), 19, teamColor(u))
	g.label(dst, fmt.Sprintf("%s  Lv%d", g.className(u.Class), u.Level), float64(x+82), float64(y+40), 13, ink)
	state := ""
	switch {
	case u.AI:
		state = "동맹군 · AI"
	case !u.ActsIn(o.Turn):
	case u.Done:
		state = "행동 완료"
	case u.Moved:
		state = "이동함"
	}
	g.label(dst, state, float64(x+w-16)-g.width(state, 12), float64(y+14), 12, muted)
	points := fmt.Sprintf("특성치 %d", u.Points)
	g.label(dst, points, float64(x+w-16)-g.width(points, 13), float64(y+40), 13, gold)
	recovery := g.unitRecovery(u.ID, o.Revision)
	gauge := func(by int, name string, now, maxV int, k float64, c color.Color, recovery int, showRecovery bool) {
		v := fmt.Sprintf("%d / %d", now, maxV)
		right := float64(x + w - 16)
		if showRecovery {
			extra := fmt.Sprintf("+%d", recovery)
			left := right - g.width(extra, 12)
			g.label(dst, extra, left, float64(by), 12, healC)
			right = left - 8
		}
		left := right - g.width(v, 12)
		g.label(dst, name, float64(x+16), float64(by), 12, muted)
		bar(dst, float32(x+70), float32(by+6), float32(max(20, left-float64(x+80))), 7, k, c)
		g.label(dst, v, left, float64(by), 12, ink)
	}
	k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
	gauge(y+78, "병력", u.HP, u.Stats.MaxHP, k, hpColor(k), recovery.HP, recovery.TerrainHP || recovery.HP > 0)
	gauge(y+96, "병법치", u.MP, u.Stats.MaxMP, float64(u.MP)/float64(max(1, u.Stats.MaxMP)), mpC, recovery.MP, true)
	if u.Faction == "ally" {
		gauge(y+114, "경험치", int(u.XPProgress), 100, u.XPProgress/100, xpC, 0, false)
	}
	g.label(dst, "+수치: 다음 자기 턴 회복 · 현재 위치", float64(x+82), float64(y+62), 10, muted)
	for i, title := range []string{"능력", "특성", "장비", "열전"} {
		bx := x + 12 + i*75
		var click func()
		if g.detailTab != i {
			click = func() { g.detailTab, g.detailPage = i, 0 }
		}
		g.btn(dst, image.Rect(bx, y+138, bx+72, y+168), title, click)
		if g.detailTab == i {
			rect(dst, float32(bx), float32(y+166), 72, 2, gold)
		}
	}
	body := image.Rect(x+16, y+180, x+w-16, y+h-12)
	switch g.detailTab {
	case 0:
		g.statHexagon(dst, o, u, float64(x+w/2), float64(body.Min.Y+95), 48)
		status := []string{}
		for _, k := range sorted(u.Status) {
			status = append(status, statusText(k, u.Status[k]))
		}
		if len(status) == 0 {
			status = append(status, "상태 이상 없음")
		}
		g.detailText(dst, image.Rect(body.Min.X, body.Min.Y+180, body.Max.X, body.Max.Y), strings.Join(status, " · "))
	case 1:
		g.detailTraits(dst, body, u.Traits)
	case 2:
		lines := []string{}
		for _, c := range o.Characters {
			if c.ID != u.Character {
				continue
			}
			for _, slot := range []string{"weapon", "armor", "accessory"} {
				eq, ok := g.S.Data.Equipment[c.Equipment[slot]]
				name := map[string]string{"weapon": "무기", "armor": "방어구", "accessory": "장신구"}[slot]
				if !ok {
					lines = append(lines, name+" · 없음", "")
					continue
				}
				lines = append(lines, name+" · "+eq.Name, eq.Desc)
				for _, id := range eq.Traits {
					lines = append(lines, g.traitName(id)+": "+g.traitDesc(id))
				}
				lines = append(lines, "")
			}
			break
		}
		g.detailText(dst, body, strings.Join(lines, "\n"))
	case 3:
		for i, name := range []string{"통솔", "무력", "지력", "민첩", "운", "매력"} {
			g.label(dst, fmt.Sprintf("%s  %d", name, u.Attributes[i]), float64(body.Min.X+i%2*144), float64(body.Min.Y+i/2*24), 14, ink)
		}
		g.label(dst, "열전", float64(body.Min.X), float64(body.Min.Y+80), 14, gold)
		bio := g.S.Data.Characters[u.Def].Biography
		if bio == "" {
			bio = "등록된 열전이 없습니다."
		}
		g.detailText(dst, image.Rect(body.Min.X, body.Min.Y+108, body.Max.X, body.Max.Y), bio)
	}
}

func (g *Game) detailTraits(dst *ebiten.Image, r image.Rectangle, traits []string) {
	const rowH, arrowH = 28, 18
	rows := max(1, (r.Dy()-28-2*arrowH-4)/rowH)
	g.traitTop = max(0, min(max(0, len(traits)-rows), g.traitTop))
	g.label(dst, fmt.Sprintf("보유 특성 %d", len(traits)), float64(r.Min.X), float64(r.Min.Y), 13, gold)
	var up, down func()
	if g.traitTop > 0 {
		up = func() { g.traitTop-- }
	}
	if g.traitTop+rows < len(traits) {
		down = func() { g.traitTop++ }
	}
	y := r.Min.Y + 26
	g.arrow(dst, image.Rect(r.Min.X, y, r.Max.X, y+arrowH), true, up)
	y += arrowH + 2
	for i := 0; i < min(rows, len(traits)); i++ {
		id := traits[g.traitTop+i]
		g.btn(dst, image.Rect(r.Min.X, y+i*rowH, r.Max.X, y+(i+1)*rowH-2), g.traitName(id), func() {
			if !g.popupPending() {
				g.modals = append(g.modals, &modal{kind: "trait", trait: id})
			}
		})
	}
	if len(traits) == 0 {
		g.label(dst, "(없음)", float64(r.Min.X+8), float64(y+4), 13, muted)
	}
	g.arrow(dst, image.Rect(r.Min.X, y+rows*rowH, r.Max.X, y+rows*rowH+arrowH), false, down)
}

func (g *Game) detailText(dst *ebiten.Image, r image.Rectangle, text string) {
	lines := strings.Split(wrap(text, 23), "\n")
	rows := max(1, (r.Dy()-28)/20)
	pages := max(1, (len(lines)+rows-1)/rows)
	g.detailPage = max(0, min(pages-1, g.detailPage))
	start := g.detailPage * rows
	g.label(dst, strings.Join(lines[start:min(len(lines), start+rows)], "\n"), float64(r.Min.X), float64(r.Min.Y), 12, ink)
	if pages <= 1 {
		return
	}
	var prev, next func()
	if g.detailPage > 0 {
		prev = func() { g.detailPage-- }
	}
	if g.detailPage+1 < pages {
		next = func() { g.detailPage++ }
	}
	g.btn(dst, image.Rect(r.Min.X, r.Max.Y-26, r.Min.X+64, r.Max.Y), "이전", prev)
	g.btn(dst, image.Rect(r.Max.X-64, r.Max.Y-26, r.Max.X, r.Max.Y), "다음", next)
	g.centered(dst, fmt.Sprintf("%d / %d", g.detailPage+1, pages), float64(r.Min.X+r.Dx()/2), float64(r.Max.Y-22), 12, muted)

}

// hexAxes are the hexagon's corners clockwise from the top: each unit stat with the
// officer attribute it grows from (core/rules.go: stats).
var hexAxes = []struct {
	stat, attr string
	attrIdx    int
	get        func(core.Stats) int
}{
	{"공격", "무력", 1, func(s core.Stats) int { return s.Attack }},
	{"방어", "통솔", 0, func(s core.Stats) int { return s.Defense }},
	{"정신", "지력", 2, func(s core.Stats) int { return s.Mind }},
	{"순발", "민첩", 3, func(s core.Stats) int { return s.Agility }},
	{"사기", "운", 4, func(s core.Stats) int { return s.Morale }},
	{"병력", "매력", 5, func(s core.Stats) int { return s.MaxHP }},
}

func (g *Game) originalStats(u *core.UnitView) core.Stats {
	base, err := g.S.Engine.CharacterStats(u.ID)
	if err != nil {
		return u.Stats
	}
	return base
}

func (g *Game) statHexagon(dst *ebiten.Image, o core.Observation, u *core.UnitView, cx, cy, radius float64) {
	base := g.originalStats(u)
	changed := false
	for _, a := range hexAxes {
		changed = changed || a.get(base) != a.get(u.Stats)
	}
	best := make([]int, len(hexAxes))
	for _, v := range o.UnitViews {
		original := g.originalStats(&v)
		for i, a := range hexAxes {
			best[i] = max(best[i], a.get(v.Stats), a.get(original))
		}
	}
	corner := func(i int, k float64) (float32, float32) {
		t := -math.Pi/2 + float64(i)*math.Pi/3
		return float32(cx + radius*k*math.Cos(t)), float32(cy + radius*k*math.Sin(t))
	}
	ring := func(k float64) *vector.Path {
		var p vector.Path
		for i := range hexAxes {
			px, py := corner(i, k)
			if i == 0 {
				p.MoveTo(px, py)
			} else {
				p.LineTo(px, py)
			}
		}
		p.Close()
		return &p
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.NRGBA{R: 18, G: 14, B: 12, A: 200})
	fillPath(dst, ring(1), op)
	for _, k := range []float64{1, 2.0 / 3, 1.0 / 3} {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(goldDim)
		strokePath(dst, ring(k), vector.StrokeOptions{Width: 1}, op)
	}
	for i := range hexAxes {
		ex, ey := corner(i, 1)
		strokeLine(dst, float32(cx), float32(cy), ex, ey, 1, goldDim)
	}
	drawShape := func(stats core.Stats, c color.NRGBA) {
		var shape vector.Path
		for i, a := range hexAxes {
			px, py := corner(i, max(.04, min(1, float64(a.get(stats))/float64(max(1, best[i])))))
			if i == 0 {
				shape.MoveTo(px, py)
			} else {
				shape.LineTo(px, py)
			}
		}
		shape.Close()
		fill := &vector.DrawPathOptions{AntiAlias: true}
		fill.ColorScale.ScaleWithColor(color.NRGBA{R: c.R, G: c.G, B: c.B, A: 35})
		fillPath(dst, &shape, fill)
		edge := &vector.DrawPathOptions{AntiAlias: true}
		edge.ColorScale.ScaleWithColor(c)
		strokePath(dst, &shape, vector.StrokeOptions{Width: 1.6, LineJoin: vector.LineJoinRound}, edge)
	}
	if changed {
		drawShape(base, color.NRGBA{R: 145, G: 145, B: 145, A: 255})
	}
	drawShape(u.Stats, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	tip := []string{"바깥선: 전투 부대의 현재·원래 최고치", "흰색: 현재 능력치"}
	if changed {
		tip = append(tip, "회색: 상태 효과 적용 전 능력치")
	}
	for i, a := range hexAxes {
		top := fmt.Sprintf("%s %d", a.stat, a.get(u.Stats))
		sub := fmt.Sprintf("%s %d", a.attr, u.Attributes[a.attrIdx])
		ex, ey := corner(i, 1)
		lx, ly := float64(ex), float64(ey)
		tw := max(g.width(top, 12), g.width(sub, 11))
		switch i {
		case 0:
			lx, ly = lx-tw/2, ly-36
		case 3:
			lx, ly = lx-tw/2, ly+3
		case 1, 2:
			lx, ly = lx+8, ly-16
		default:
			lx, ly = lx-8-tw, ly-16
		}
		g.label(dst, top, lx, ly, 12, ink)
		g.label(dst, sub, lx, ly+16, 11, muted)
		tip = append(tip, fmt.Sprintf("%s %d (원래 %d) / 최고 %d", a.stat, a.get(u.Stats), a.get(base), best[i]))
	}
	area := image.Rect(int(cx-radius), int(cy-radius), int(cx+radius), int(cy+radius))
	if hovered(area) {
		tw, th := 270, 14+len(tip)*19
		tx, ty := int(cx-radius)-110-tw, int(cy)-th/2
		window(dst, image.Rect(tx, ty, tx+tw, ty+th))
		g.label(dst, strings.Join(tip, "\n"), float64(tx+14), float64(ty+7), 12, ink)
	}
}

// terrainPanel names the hovered cell and what it does for the described unit's family.
func (g *Game) terrainPanel(dst *ebiten.Image, o core.Observation) {
	if !g.hover.ok {
		return
	}
	tile := g.field.Tile(g.hover.x, g.hover.y)
	family, class := "", "병종"
	if u := g.infoUnit(o); u != nil {
		c := g.S.Data.Classes[u.Class]
		family, class = c.Terrain, c.Name
	}
	lines := terrainInfoLines(g.S.Data, string(tile), family, class)
	w, h := 238, 44+len(lines)*19
	x, y := 12, g.battleLogRect().Min.Y-8-h
	window(dst, image.Rect(x, y, x+w, y+h))
	g.label(dst, fmt.Sprintf("%s  (%d,%d)", g.S.Data.Tiles[string(tile)].Name, g.hover.x, g.hover.y), float64(x+14), float64(y+10), 16, gold)
	g.label(dst, strings.Join(lines, "\n"), float64(x+14), float64(y+36), 13, ink)
}

var miniColors = map[byte]color.NRGBA{
	'.': {104, 150, 72, 255}, 'd': {164, 142, 96, 255}, 's': {120, 104, 88, 255}, 'f': {54, 100, 52, 255},
	'c': {150, 150, 156, 255}, 'i': {190, 180, 160, 255}, 'v': {206, 150, 72, 255}, '~': {60, 110, 170, 255},
	'b': {150, 100, 60, 255}, 'g': {110, 110, 118, 255}, 'k': {196, 120, 70, 255},
}

const miniScale = 4.0 // minimap pixels per half cell width

// minimap shows the whole field, unit dots and the visible area; a click moves the camera.
func (g *Game) minimap(dst *ebiten.Image, o core.Observation) {
	m := g.field.Map
	w := float64(m.W+m.H) * miniScale
	h := w / 2
	x0 := float64(Width) - 12 - w - 14
	y0 := 52.0
	r := image.Rect(int(x0)-10, int(y0)-10, int(x0+w)+10, int(y0+h)+10)
	g.miniRect = r
	window(dst, r)
	ox := x0 + float64(m.H)*miniScale
	at := func(u, v float64) (float32, float32) {
		return float32(ox + (u-v)*miniScale), float32(y0 + (u+v)*miniScale/2)
	}
	for v, row := range m.Tiles {
		for u := range row {
			px, py := at(float64(u), float64(v))
			c := miniColors[row[u]]
			rect(dst, px, py, float32(miniScale*2), float32(miniScale), c)
		}
	}
	for _, u := range o.UnitViews {
		if u.HP <= 0 {
			continue
		}
		c := color.NRGBA{R: 120, G: 190, B: 255, A: 255}
		if u.Faction == "enemy" {
			c = color.NRGBA{R: 240, G: 80, B: 64, A: 255}
		}
		px, py := at(float64(u.X), float64(u.Y))
		fillCircle(dst, px+float32(miniScale), py+float32(miniScale/2), 2.5, c)
	}
	// visible area: screen corners → canvas → minimap (both are linear in the iso projection)
	toMini := func(sx, sy float64) (float32, float32) {
		cx, cy := g.toCanvas(sx, sy)
		mx, my := ToMap(cx, cy)
		return float32(ox + (mx-g.field.OriginX)/16*miniScale), float32(y0 + (my-g.field.OriginY)/16*miniScale)
	}
	ax, ay := toMini(0, 40)
	vp := g.battleViewport()
	bx, by := toMini(float64(vp.Max.X), float64(vp.Max.Y))
	ax, ay = max(ax, float32(x0)), max(ay, float32(y0))
	bx, by = min(bx, float32(x0+w)), min(by, float32(y0+h))
	if bx > ax && by > ay {
		strokeRect(dst, ax, ay, bx-ax, by-ay, 1, ink)
	}
	g.buttons = append(g.buttons, button{r, "", func() {
		mx, my := cursorPos()
		g.camX = ((float64(mx)-ox)/miniScale*16 + g.field.OriginX) * K
		g.camY = ((float64(my)-y0)/miniScale*16 + g.field.OriginY) * K
	}})
	g.block(r)
}

// messages fades recent log lines in under the top bar.
func (g *Game) messages(dst *ebiten.Image) {
	y := 50.0
	if g.ord.stage == "target" {
		y = 256
	}
	for _, t := range g.toasts {
		a := min(1, (toastTime-t.t)*2)
		w := g.width(t.text, 16)
		rect(dst, float32(Width/2-w/2-14), float32(y), float32(w+28), 28, color.NRGBA{A: uint8(150 * a)})
		c := color.NRGBA{}
		r, gg, b, _ := t.col.RGBA()
		c.R, c.G, c.B, c.A = uint8(r>>8), uint8(gg>>8), uint8(b>>8), uint8(255*a)
		g.label(dst, t.text, float64(Width/2)-w/2, y+3, 16, c)
		y += 32
	}
}

// unitsByFaction lists live units of one faction, the leader first.
func unitsByFaction(o core.Observation, faction string) []core.UnitView {
	out := []core.UnitView{}
	for _, u := range o.UnitViews {
		if u.Faction == faction {
			out = append(out, u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].HP > 0 && out[j].HP <= 0 })
	return out
}

func (g *Game) unitRecovery(id string, revision uint64) core.RecoveryPreview {
	e := g.engine()
	if e != g.recoveryEngine || revision != g.recoveryRevision || id != g.recoveryID {
		g.recovery = e.NextTurnRecovery(id)
		g.recoveryEngine, g.recoveryRevision, g.recoveryID = e, revision, id
	}
	return g.recovery
}

func terrainInfoLines(data *content.Data, tile, family, class string) []string {
	d := data.Tiles[tile]
	var lines []string
	if family == "" {
		lines = []string{"이동·보정: 유닛 선택 시 표시"}
	} else {
		t := data.Terrain[family][tile]
		move := "진입 불가"
		if t.Cost > 0 {
			move = fmt.Sprintf("이동 비용 %d", t.Cost)
		}
		lines = []string{fmt.Sprintf("%s 보정 %.0f%%", class, t.Factor*100), move}
	}
	if d.RestHP > 0 {
		lines = append(lines, fmt.Sprintf("병력 회복 +%d%%", d.RestHP))
	} else if d.RestMP > 1 {
		lines = append(lines, "병력 회복 없음")
	}
	if d.RestMP > 1 {
		lines = append(lines, fmt.Sprintf("병법치 회복 ×%d", d.RestMP))
	}
	if d.RestHP == 0 && d.RestMP <= 1 {
		lines = append(lines, "지형 회복 효과 없음")
	}
	return lines
}
