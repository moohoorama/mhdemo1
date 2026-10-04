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
	if g.field == nil || g.overlay != "" || len(g.modals) > 0 {
		return
	}
	x, y := cursorPos()
	if g.overHUD(x, y) {
		return
	}
	cx, cy := g.toCanvas(float64(x), float64(y))
	mx, my := g.field.Cell(cx, cy)
	// a castle wall's surface is drawn lifted; prefer it when the cursor is on it
	if lu, lv := g.field.Cell(cx, cy+float64(g.assets.Rise*K)); g.field.Tile(lu, lv) == 'c' {
		mx, my = lu, lv
	}
	if g.field.Inside(mx, my) {
		g.hover = hoverCell{mx, my, true}
	}
}

// cursor outlines the hovered cell.
func (g *Game) cursor(dst *ebiten.Image) {
	if !g.hover.ok {
		return
	}
	cx, cy := g.field.Center(float64(g.hover.x), float64(g.hover.y))
	cy -= g.field.Lift(g.hover.x, g.hover.y)
	pts := [][2]float64{{cx, cy - 8*K}, {cx + 16*K, cy}, {cx, cy + 8*K}, {cx - 16*K, cy}}
	pulse := uint8(180 + 60*math.Sin(g.clockMS/160))
	for i := range pts {
		ax, ay := g.toScreen(pts[i][0], pts[i][1])
		bx, by := g.toScreen(pts[(i+1)%4][0], pts[(i+1)%4][1])
		vector.StrokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), 2, color.NRGBA{R: 255, G: 240, B: 190, A: pulse}, true)
	}
}

func objective(st *coreStage) (win, lose string) {
	win = st.boss + " 격파"
	if st.threshold > 0 {
		win = fmt.Sprintf("%s 병력 %d%% 이하", st.boss, st.threshold)
	}
	return win, "유비 퇴각"
}

type coreStage struct {
	name, boss string
	threshold  int
}

func (g *Game) stageInfo(o core.Observation) *coreStage {
	st := g.S.Data.Stages[o.Stage]
	return &coreStage{st.Name, g.name(st.Boss), st.Threshold}
}

func (g *Game) pending(o core.Observation) (left, total int) {
	for _, u := range o.UnitViews {
		if u.Faction == "ally" && u.HP > 0 {
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
	if o.Turn == "enemy" {
		turn = bad
	}
	x := 24 + g.width(st.name, 18)
	g.label(dst, fmt.Sprintf("%d턴", o.Round), x, 9, 16, ink)
	g.label(dst, factionName(o.Turn)+" 진영", x+52, 9, 16, turn)
	win, lose := objective(st)
	g.label(dst, "승리  "+win+"     패배  "+lose, x+150, 11, 14, muted)
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
	} else if g.S.Settings.AI != "auto" {
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
	if left, _ := g.pending(g.S.Engine.Observe()); left > 0 {
		g.ask(fmt.Sprintf("아직 행동하지 않은 부대가 %d개 있습니다.\n진영을 종료하시겠습니까?", left), g.endTurn)
		return
	}
	g.endTurn()
}

func (g *Game) endTurn() {
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

// detail describes the selected unit under the minimap, its traits listed below; a trait
// opens its explanation in the middle of the screen.
func (g *Game) detail(dst *ebiten.Image, o core.Observation) {
	u := g.unitView(o, g.selected)
	if u == nil || u.HP <= 0 {
		return
	}
	if g.traitOf != u.ID {
		g.traitOf, g.traitTop = u.ID, 0
	}
	const w, rowH, arrowH = 300, 26, 20
	x, y := Width-12-w, g.miniRect.Max.Y+10
	bottom := Height - 260 // clear of the dock
	rows := min(len(u.Traits), max(2, (bottom-y-232-2*arrowH)/rowH))
	h := 232 + 2*arrowH + max(1, rows)*rowH + 12
	r := image.Rect(x, y, x+w, y+h)
	window(dst, r)
	g.block(r)
	g.portrait(dst, u.ID, float64(x+16), float64(y+16), 64)
	g.label(dst, u.Name, float64(x+94), float64(y+12), 19, teamColor(u))
	g.label(dst, fmt.Sprintf("%s  Lv%d", u.Class, u.Level), float64(x+94), float64(y+42), 13, ink)
	state := ""
	switch {
	case u.Faction != o.Turn:
	case u.Done:
		state = "행동 완료"
	case u.Moved:
		state = "이동함"
	}
	g.label(dst, state, float64(x+w-80), float64(y+16), 12, muted)
	gauge := func(by int, name string, now, maxV int, k float64, c color.Color) {
		g.label(dst, name, float64(x+16), float64(by), 12, muted)
		g.label(dst, fmt.Sprintf("%d / %d", now, maxV), float64(x+w-16)-g.width(fmt.Sprintf("%d / %d", now, maxV), 12), float64(by), 12, ink)
		bar(dst, float32(x+16), float32(by+18), float32(w-32), 7, k, c)
	}
	k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
	gauge(y+88, "병력", u.HP, u.Stats.MaxHP, k, hpColor(k))
	gauge(y+116, "병법치", u.MP, u.Stats.MaxMP, float64(u.MP)/float64(max(1, u.Stats.MaxMP)), mpC)
	if u.Faction == "ally" {
		gauge(y+144, "경험치", int(u.XPProgress), 100, u.XPProgress/100, xpC)
	}
	st := u.Stats
	g.label(dst, fmt.Sprintf("공격 %d  방어 %d  정신 %d  순발 %d  사기 %d", st.Attack, st.Defense, st.Mind, st.Agility, st.Morale),
		float64(x+16), float64(y+174), 12, ink)
	status := []string{}
	for _, k := range sorted(u.Status) {
		status = append(status, fmt.Sprintf("%s %d", statusNames[k], u.Status[k].Turns))
	}
	if len(status) > 0 {
		g.label(dst, "상태  "+strings.Join(status, " · "), float64(x+16), float64(y+192), 12, gold)
	}
	g.label(dst, "특성", float64(x+16), float64(y+210), 14, gold)
	g.traitTop = max(0, min(len(u.Traits)-rows, g.traitTop))
	var up, down func()
	if g.traitTop > 0 {
		up = func() { g.traitTop-- }
	}
	if g.traitTop+rows < len(u.Traits) {
		down = func() { g.traitTop++ }
	}
	ly := y + 232
	g.arrow(dst, image.Rect(x+16, ly, x+w-16, ly+arrowH), true, up)
	ly += arrowH + 2
	for i := 0; i < rows; i++ {
		trait := u.Traits[g.traitTop+i]
		g.btn(dst, image.Rect(x+16, ly+i*rowH, x+w-16, ly+i*rowH+rowH-2), trait, func() {
			g.modals = append(g.modals, &modal{kind: "trait", trait: trait})
		})
	}
	if rows == 0 {
		g.label(dst, "(없음)", float64(x+24), float64(ly+4), 13, muted)
	}
	ly += max(1, rows) * rowH
	g.arrow(dst, image.Rect(x+16, ly, x+w-16, ly+arrowH), false, down)
}

// terrainPanel names the hovered cell and what it does for the described unit's family.
func (g *Game) terrainPanel(dst *ebiten.Image, o core.Observation) {
	if !g.hover.ok {
		return
	}
	tile := g.field.Tile(g.hover.x, g.hover.y)
	family := "보병"
	if u := g.infoUnit(o); u != nil {
		family = g.S.Data.Classes[u.Class].Family
	}
	t := g.S.Data.Terrain[family][string(tile)]
	x, y, w, h := 12, Height-12-86, 196, 86
	window(dst, image.Rect(x, y, x+w, y+h))
	g.label(dst, fmt.Sprintf("%s  (%d,%d)", terrainNames[tile], g.hover.x, g.hover.y), float64(x+14), float64(y+10), 16, gold)
	move := "진입 불가"
	if t.Cost > 0 {
		move = fmt.Sprintf("이동 비용 %d", t.Cost)
	}
	lines := []string{fmt.Sprintf("%s 보정 %.0f%%", family, t.Factor*100), move}
	if tile == 'v' || tile == 'i' {
		lines[1] += " · 회복"
	}
	g.label(dst, strings.Join(lines, "\n"), float64(x+14), float64(y+36), 13, ink)
}

var miniColors = map[byte]color.NRGBA{
	'.': {104, 150, 72, 255}, 'd': {164, 142, 96, 255}, 's': {120, 104, 88, 255}, 'f': {54, 100, 52, 255},
	'c': {150, 150, 156, 255}, 'i': {190, 180, 160, 255}, 'v': {206, 150, 72, 255}, '~': {60, 110, 170, 255},
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
		vector.FillCircle(dst, px+float32(miniScale), py+float32(miniScale/2), 2.5, c, true)
	}
	// visible area: screen corners → canvas → minimap (both are linear in the iso projection)
	toMini := func(sx, sy float64) (float32, float32) {
		cx, cy := g.toCanvas(sx, sy)
		mx, my := ToMap(cx, cy)
		return float32(ox + (mx-g.field.OriginX)/16*miniScale), float32(y0 + (my-g.field.OriginY)/16*miniScale)
	}
	ax, ay := toMini(0, 40)
	bx, by := toMini(Width, Height)
	ax, ay = max(ax, float32(x0)), max(ay, float32(y0))
	bx, by = min(bx, float32(x0+w)), min(by, float32(y0+h))
	if bx > ax && by > ay {
		vector.StrokeRect(dst, ax, ay, bx-ax, by-ay, 1, ink, false)
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
