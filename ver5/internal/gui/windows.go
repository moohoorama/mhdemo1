package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
	"srpg/internal/sprite"
	"srpg/internal/storage"
)

var overlayTitles = map[string]string{"menu": "메뉴", "save": "저장", "load": "불러오기", "settings": "설정",
	"confirm": "확인", "equipment": "장비", "deputy": "부관 임명", "roster": "병력일람", "shop": "상점", "party": "관리", "unit": "부대", "waypoints": "장면 이동", "difficulty": "난이도"}

func (g *Game) popupPending() bool {
	return len(g.overlays) > 0 || len(g.modals) > 0 || len(g.learn) > 0 ||
		(g.S.Screen == "playing" && len(g.S.View().Learners) > 0)
}

// overlay is the top window, the one that takes the input; "" when none is open.
func (g *Game) overlay() string {
	if len(g.overlays) == 0 {
		return ""
	}
	return g.overlays[len(g.overlays)-1]
}

// openOverlay opens a window over the open ones; closing it shows the one beneath again.
func (g *Game) openOverlay(name string) bool {
	if len(g.overlays) == 0 && g.popupPending() {
		return false
	}
	g.stopCampaignAuto()
	g.overlays = append(g.overlays, name)
	g.page = 0
	if name == "equipment" {
		g.equip = equipState{}
	}
	if name == "deputy" {
		g.deputyFresh()
	}
	g.autoplay = false
	if name == "save" || name == "load" {
		g.openSaves(name)
	}
	return true
}

// closeOverlay closes the top window.
func (g *Game) closeOverlay() {
	g.stopCampaignAuto()
	if len(g.overlays) > 0 {
		if !g.accepting {
			g.sfx("close")
		}
		g.overlays = g.overlays[:len(g.overlays)-1]
	}
}

// closeOverlays closes every window.
func (g *Game) closeOverlays() {
	if len(g.overlays) > 0 && !g.accepting {
		g.sfx("close")
	}
	g.overlays = nil
}

// acceptConfirmation lets the approved action sound; window cleanup is silent, with ok as the fallback.
func (g *Game) acceptConfirmation() {
	if g.overlay() != "confirm" {
		return
	}
	before, previous := g.sfxCalls, g.accepting
	g.accepting = true
	defer func() { g.accepting = previous }()
	f := g.answer
	g.closeOverlay()
	if f != nil {
		f()
	}
	g.sfxUnless(before, "ok")
}

// ask opens a confirmation window; ok runs on 확인.
func (g *Game) ask(message string, ok func()) { g.askWith(message, "확인", "취소", ok) }

// askWith is ask with its own button labels; ok runs on yes.
func (g *Game) askWith(message, yes, no string, ok func()) bool {
	if g.font != nil {
		if _, _, fits := g.confirmLayout(message, uiW, uiH); !fits {
			g.toast("확인 문구가 화면에 들어가지 않습니다", bad)
			return false
		}
	}
	if !g.openOverlay("confirm") {
		return false
	}
	g.question, g.yes, g.no, g.answer, g.confirmTitle = message, yes, no, ok, "확인"
	return true
}

// modal is a window shown after a replay, closed with a click.
type modal struct {
	kind   string // objective, levelup, promotion, trait, acquire
	manual bool   // an objective opened from the panel rather than at the battle's start
	level  levelUp
	promo  promotionUp
	trait  string
	gains  []gain    // acquire: the cards
	piece  string    // acquire: the card whose window is open
	chip   equipChip // acquire: the trait open over that window
	heard  bool      // acquire: its sound has played, with the first card
	t      float64
}

type levelUp struct {
	id            string
	from, to      int
	before, after core.Stats
	progress      float64
}

// promotionUp is what the promotion window shows: who, the class left and the one taken, the looks of
// both and the stats before and after.
type promotionUp struct {
	cmd           core.Command // a seal's preview: the command that makes this class
	id, from, to  string
	old, new      *sprite.Art
	before, after core.Stats
}

// drawOverlay draws the open windows bottom first, each dimming the ones beneath; only the top
// one takes clicks and shows the cursor's hover.
func (g *Game) drawOverlay(dst *ebiten.Image) {
	cursor := cursorPos
	defer func() { cursorPos = cursor }()
	for i, name := range g.overlays {
		g.buttons, g.blocks, g.wheelAreas = nil, nil, nil
		cursorPos = cursor
		if i < len(g.overlays)-1 {
			cursorPos = func() (int, int) { return -1000, -1000 }
		}
		g.drawWindow(dst, name)
	}
}

func (g *Game) drawWindow(dst *ebiten.Image, name string) {
	w, h := 560, 440
	switch name {
	case "confirm":
		w, h, _ = g.confirmLayout(g.question, uiW, uiH)
	case "save":
		w, h = saveW, saveH-46
	case "load":
		w, h = saveW, saveH
	case "equipment":
		w, h = 760, 80+3*(equipCardH+14)+10
	case "deputy":
		w, h = deputyW, deputyH
	case "settings":
		w, h = 640, 668+64
		if g.Debug {
			h += 128
		}
	case "roster":
		w, h = uiW-100, uiH-90
	case "shop":
		w, h = 880, 720
	case "party":
		w, h = 1100, 720
	case "waypoints":
		w, h = waypointW, waypointH
	case "difficulty":
		w, h = 640, 84+len(g.difficulties())*difficultyRowH
	}
	// the window in a frame with room for its title plate and shadow; a sheet has no plate
	pad, strip := 40, closeStrip
	switch {
	case sheetWindows[name]:
		w, h = sheetSize()
		pad, strip = 16, 0
	case name == "confirm":
		strip = 0
	}
	g.popup(dst, "overlay:"+name, 150, w+pad, h+strip+pad, image.Rect(0, 0, w+pad, h+strip+pad), func(dst *ebiten.Image) { g.overlayWindow(dst, name, w, h, strip) })
	if name == "equipment" && g.equip.info.id != "" {
		g.drawEquipInfo(dst)
	}
	if name == "party" && len(g.party.insu.preview) > 0 {
		g.drawInsuPreview(dst)
	}
	if name == "shop" && g.shopView.info.id != "" {
		g.chipWindow(dst, "overlay:shop-info", g.shopView.info, func() { g.shopView.info = equipChip{} })
	}
	partyDeputy := name == "party" && !g.party.items && !g.party.table && g.party.tab == 3
	if (name == "deputy" || partyDeputy) && g.deputyInfo.id != "" {
		g.chipWindow(dst, "overlay:deputy-info", g.deputyInfo, func() { g.deputyInfo = equipChip{} })
	}
}

// closeStrip is the row under an older window's content that holds 닫기 at the bottom right.
const closeStrip = touchH + 16

// overlayWindow draws a window: its frame (strip px taller than the content, for 닫기 and the actions the content
// put in overlayActs), the content in w×h and the close row.
func (g *Game) overlayWindow(dst *ebiten.Image, name string, w, h, strip int) {
	x, y := uiW/2-w/2, uiH/2-(h+strip)/2
	r := image.Rect(x, y, x+w, y+h+strip)
	g.overlayActs = nil
	defer func() {
		if strip > 0 {
			g.closeRow(dst, x+w-20, y+h+strip-touchH-14, g.closeOverlay, g.overlayActs...)
		}
	}()
	switch {
	case sheetWindows[name]:
		g.window(dst, r)
	case name == "confirm":
		g.titledPattern(dst, r, g.confirmTitle)
	default:
		g.titled(dst, r, overlayTitles[name])
	}
	left := x + 32
	switch name {
	case "confirm":
		g.label(dst, wrapWidth(g.question, g.face(18), float64(w-64)), float64(left), float64(y+44), 18, ink)
		g.btn(dst, image.Rect(left, y+h-62, left+190, y+h-28), g.yes, g.acceptConfirmation)
		g.btn(dst, image.Rect(x+w-222, y+h-62, x+w-32, y+h-28), g.no, g.closeOverlay)
	case "menu":
		items := []menuItem{
			{"게임으로", func() { g.closeOverlays(); g.request(session.Request{Op: "resume"}) }},
			{"저장", func() { g.openOverlay("save") }},
			{"불러오기", func() { g.openOverlay("load") }},
			{"설정", func() { g.openOverlay("settings") }},
			{"시작 메뉴", func() {
				g.request(session.Request{Op: "title"})
			}},
			{"종료", func() { g.request(session.Request{Op: "quit"}) }},
		}
		if g.S.Engine == nil {
			items[1].click = nil
		}
		for i, it := range items {
			g.btn(dst, image.Rect(left, y+64+i*54, x+w-32, y+64+i*54+42), it.label, it.click)
		}
	case "save", "load":
		g.saveWindow(dst, name, x, y, w, h)
	case "waypoints":
		g.waypointWindow(dst, x, y, w, h)
	case "difficulty":
		g.difficultyWindow(dst, x, y, w)
	case "settings":
		s := g.S.Settings
		set := func(change func(v *storage.Settings)) func() {
			return func() {
				v := g.S.Settings
				change(&v)
				g.request(session.Request{Op: "settings", Settings: &v})
			}
		}
		speeds, scales, damages, xps := []choice{}, []choice{}, []choice{}, []choice{}
		for _, n := range []int{1, 2, 4, 8} {
			n := n
			speeds = append(speeds, choice{fmt.Sprintf("×%d", n), g.speed == n, func() { g.setSpeed(n) }})
		}
		for _, n := range uiScales {
			n := n
			scales = append(scales, choice{fmt.Sprintf("%d%%", n), int(math.Round(g.uiScale()*100)) == n, set(func(v *storage.Settings) { v.UIScale = n })})
		}
		for _, n := range storage.ScalePcts {
			n := n
			damages = append(damages, choice{fmt.Sprintf("%d%%", n), s.DamageScale() == n, set(func(v *storage.Settings) { v.DamagePct = n })})
			xps = append(xps, choice{fmt.Sprintf("%d%%", n), s.XPScale() == n, set(func(v *storage.Settings) { v.XPPct = n })})
		}
		var camW, camH []choice
		zw, zh := s.CamZone()
		for _, n := range storage.CamZonePcts {
			n := n
			camW = append(camW, choice{fmt.Sprintf("%d%%", n), zw == n, set(func(v *storage.Settings) { v.CamZoneW = n })})
			camH = append(camH, choice{fmt.Sprintf("%d%%", n), zh == n, set(func(v *storage.Settings) { v.CamZoneH = n })})
		}
		type row struct {
			title string
			opts  []choice
		}
		rows := []row{
			{"AI 동시 진행", []choice{
				{"항상", s.Replay == "together", set(func(v *storage.Settings) { v.Replay = "together" })},
				{"보통", s.Replay == "spells", set(func(v *storage.Settings) { v.Replay = "spells" })},
				{"안 함", s.Replay == "one", set(func(v *storage.Settings) { v.Replay = "one" })},
			}},
			{"전투 기록", []choice{
				{"상세", s.Detail, set(func(v *storage.Settings) { v.Detail = true })},
				{"주요 사건만", !s.Detail, set(func(v *storage.Settings) { v.Detail = false })},
			}},
			{"길 안내", []choice{
				{"켜기", !s.NoGuide, set(func(v *storage.Settings) { v.NoGuide = false })},
				{"끄기", s.NoGuide, set(func(v *storage.Settings) { v.NoGuide = true })},
			}},
			{"대사 속도", []choice{
				{"한 글자씩", !s.InstantText, set(func(v *storage.Settings) { v.InstantText = false })},
				{"즉시", s.InstantText, set(func(v *storage.Settings) { v.InstantText = true })},
			}},
			{"소리", []choice{
				{"모두", s.Sound == "", set(func(v *storage.Settings) { v.Sound = "" })},
				{"효과음만", s.Sound == "sfx", set(func(v *storage.Settings) { v.Sound = "sfx" })},
				{"음악만", s.Sound == "music", set(func(v *storage.Settings) { v.Sound = "music" })},
				{"끄기", s.Sound == "off", set(func(v *storage.Settings) { v.Sound = "off" })},
			}},
			{"재생 속도", speeds},
			{"UI 크기", scales},
		}
		if g.Debug {
			rows = append(rows, row{"데미지 배율", damages}, row{"경험치 배율", xps}, row{"장면 카메라 가로", camW}, row{"장면 카메라 세로", camH})
		}
		for i, it := range rows {
			g.choiceRow(dst, left, y+64+i*64, x+w-32-left, it.title, it.opts)
		}
		spaceHint := ""
		if g.Debug {
			spaceHint = "Space 추천 한 명령 · "
		}
		g.label(dst, "조작: 클릭 선택 · 우클릭/Esc 취소 · 드래그/가장자리 화살표 클릭/방향키 화면 이동 · 휠 확대\n"+
			spaceHint+"F 재생 속도 · Enter 재생 건너뛰기 · F5/F9 저장/불러오기",
			float64(left), float64(y+64+len(rows)*64+4), 13, muted)
	case "equipment":
		g.equipmentWindow(dst, x, y, w, h)
	case "deputy":
		g.deputyWindow(dst, x, y, w, h)
	case "roster":
		g.rosterWindow(dst, x, y, w, h)
	case "shop":
		g.shopWindow(dst, x, y, w, h)
	case "party":
		g.partyWindow(dst, x, y, w, h)
	case "unit":
		g.unitWindow(dst, x, y, w, h)
	}
}

// focus centres the camera on a cell.
func (g *Game) focus(u, v int) {
	if g.field != nil {
		g.camX, g.camY = g.field.Center(float64(u), float64(v))
	}
}

// drawModal draws the first queued modal; any click closes it.
func (g *Game) drawModal(dst *ebiten.Image) {
	m := g.modals[0]
	g.buttons, g.blocks = nil, nil
	if m.kind == "acquire" {
		g.drawAcquire(dst, m)
		return
	}
	if m.kind == "objective" {

		box := g.objectiveBox(uiW)
		g.popup(dst, fmt.Sprintf("modal:%p", m), 110, uiW, uiH, box.Inset(-24), func(dst *ebiten.Image) { g.modalWindow(dst, m) })
		return
	}
	if m.kind != "levelup" || !g.levelUpChoice(m) {
		g.addButton(image.Rect(0, 0, uiW, uiH), "", g.closeModal)
	}
	box := map[string]image.Rectangle{
		"levelup":   levelUpBox,
		"promotion": g.promotionBox(m.promo, Width, 0),
		"trait":     image.Rect(Width/2-300, 220, Width/2+300, 560),
	}[m.kind]
	g.popup(dst, fmt.Sprintf("modal:%p", m), 110, Width, Height, box.Inset(-24), func(dst *ebiten.Image) { g.modalWindow(dst, m) })
}

// modalWindow draws a modal laid out on the Width×Height design.
func (g *Game) modalWindow(dst *ebiten.Image, m *modal) {
	switch m.kind {
	case "objective":
		o := g.S.View()
		st := g.S.Data.Stages[o.Stage]
		brief := g.objective(o)
		r := g.objectiveBox(uiW)
		if _, fits := g.objectiveLayout(g.objectiveRows(o, brief), uiW, uiH); !fits {
			g.titledPattern(dst, r, "전투 목표")
			g.label(dst, "표시 공간이 부족합니다. UI 크기를 줄이세요.", float64(r.Min.X+32), float64(r.Min.Y+56), 18, ink)
			g.objectiveButtons(dst, r)
			return
		}
		g.titledPattern(dst, r, "전투 목표")
		g.centered(dst, st.Name, float64(uiW/2), float64(r.Min.Y+32), 30, gold)
		y := float64(r.Min.Y + 96)
		for _, row := range g.objectiveRows(o, brief) {
			tw := float64(r.Dx() - 208)
			if row.state != "" {
				tw -= subChipW
			}
			wrapped := wrapWidth(row.text, g.face(19), tw)
			g.label(dst, row.name, float64(r.Min.X+48), y+2, 16, muted)
			tx := float64(r.Min.X + 160)
			if row.duel > 0 {
				duelSwatch(dst, tx+12, y+14, duelColor(row.duel-1))
				tx += 32
			}
			if row.state != "" {
				g.subChip(dst, tx, y, row.state, row.lit)
				tx += subChipW
			}
			g.label(dst, wrapped, tx, y, 19, ink)
			y += max(36, float64(strings.Count(wrapped, "\n")+1)*19*1.45+9)
		}
		g.objectiveButtons(dst, r)
	case "levelup":
		g.levelUpWindow(dst, m)
	case "promotion":
		g.promotionWindow(dst, m.promo, m.t)
	case "trait":
		r := image.Rect(uiW/2-300, 220, uiW/2+300, 560)
		g.titled(dst, r, "특성")
		name := g.traitName(m.trait)
		nx := float64(uiW/2) - (g.width(name, 34)+52)/2
		g.traitIconOf(dst, m.trait, nx, 256, 40)
		g.label(dst, name, nx+52, 252, 34, gold)
		if t, ok := g.S.Data.Traits[m.trait]; ok {
			g.centered(dst, fmt.Sprintf("%s · 습득 특성치 %d", g.traitKindName(m.trait), t.Cost), float64(uiW/2), 304, 14, muted)
		}
		g.label(dst, wrap(g.traitDesc(m.trait), 34), float64(r.Min.X+48), 342, 18, ink)
		g.closeRow(dst, r.Max.X-16, r.Max.Y-touchH-12, g.closeModal)
	}
}

// objectiveButtons are the briefing's bottom row: the objective guide's switch, pressed in and gold
// while the guide shows, and 확인, which closes it.
func (g *Game) objectiveButtons(dst *ebiten.Image, r image.Rectangle) {
	y := r.Max.Y - 58
	on := !g.S.Settings.NoGoalGuide
	toggle := func() {
		v := g.S.Settings
		v.NoGoalGuide = on
		g.request(session.Request{Op: "settings", Settings: &v})
	}
	g.toggleBtn(dst, image.Rect(r.Min.X+48, y, r.Min.X+48+150, y+36), "목표 안내", on, toggle)
	g.mBtn(dst, image.Rect(r.Max.X-48-96, y-6, r.Max.X-48, y-6+touchH), "닫기", false, g.closeModal)
}

// subChip is a bonus goal's state before its line, drawn like the guide's switch: pressed in and gold while
// the goal holds, dark when it does not.
func (g *Game) subChip(dst *ebiten.Image, x, y float64, state string, lit bool) {
	cx, cy, w, h := float32(x), float32(y)+1, float32(subChipW-12), float32(26)
	tx, ty := x+(float64(w)-g.width(state, 14))/2, float64(cy)+(float64(h)-14*1.45)/2+1
	if lit {
		rect(dst, cx, cy, w, h, color.NRGBA{R: 132, G: 98, B: 34, A: 255})
		rect(dst, cx+1, cy+1, w-2, 3, color.NRGBA{A: 110})
		strokeRect(dst, cx+.5, cy+.5, w-1, h-1, 1.5, gold)
		g.label(dst, state, tx, ty, 14, color.NRGBA{R: 255, G: 238, B: 170, A: 255})
		return
	}
	col := muted
	if state == "실패" {
		col = bad
	}
	rect(dst, cx, cy, w, h, color.NRGBA{R: 24, G: 18, B: 14, A: 255})
	strokeRect(dst, cx+.5, cy+.5, w-1, h-1, 1, goldDim)
	g.label(dst, state, tx, ty, 14, col)
}

// closeModal closes the first modal, or the top window stacked inside it.
func (g *Game) closeModal() {
	if len(g.modals) == 0 {
		return
	}
	m := g.modals[0]
	if m.closeAcquire() {
		g.sfx("close")
		return
	}
	g.sfx("ok")
	g.modals = g.modals[1:]
}

// result is the end-of-battle window.
func (g *Game) result(dst *ebiten.Image, o core.Observation) {
	g.buttons, g.blocks = nil, nil
	if g.windows["result:"+o.Result] == nil {
		g.meritHeard = 0
		if o.Result == "victory" {
			g.sfx("victory")
		} else {
			g.sfx("defeat")
		}
	}
	box := image.Rect(Width/2-330, 130, Width/2+330, 760)
	if o.Result != "victory" {
		box = image.Rect(Width/2-280, 250, Width/2+280, 560)
	}
	g.popup(dst, "result:"+o.Result, 120, Width, Height, box.Inset(-24), func(dst *ebiten.Image) { g.resultWindow(dst, o) })
}

// resultWindow draws the result window, laid out on the Width×Height design.
func (g *Game) resultWindow(dst *ebiten.Image, o core.Observation) {
	st := g.S.Data.Stages[o.Stage]
	r := image.Rect(uiW/2-330, 130, uiW/2+330, 760)
	if o.Result != "victory" {
		r = image.Rect(uiW/2-280, 250, uiW/2+280, 560)
		g.titled(dst, r, "패배")
		g.centered(dst, st.Name, float64(uiW/2), 290, 28, bad)
		reason := "전투 목표를 달성하지 못함"
		if losses := g.lordDefeat(st); len(losses) > 0 {
			reason = losses[0] + " 또는 아군 전멸"
		}
		if st.TurnLimit > 0 && o.Round > st.TurnLimit {
			reason = fmt.Sprintf("%d턴 제한 초과", st.TurnLimit)
		}
		g.centered(dst, reason, float64(uiW/2), 346, 19, ink)
		g.btn(dst, image.Rect(r.Min.X+40, 420, r.Max.X-40, 456), "배치부터 재도전", func() { g.request(session.Request{Op: "retry"}) })
		g.btn(dst, image.Rect(r.Min.X+40, 466, r.Min.X+260, 502), "불러오기", func() { g.openOverlay("load") })
		g.btn(dst, image.Rect(r.Max.X-260, 466, r.Max.X-40, 502), "시작 메뉴", func() { g.request(session.Request{Op: "title"}) })
		return
	}
	g.victoryWindow(dst, o, r)
}

func resultUnits(o core.Observation) []core.UnitView {
	units := []core.UnitView{}
	for _, u := range unitsByFaction(o, "ally") {
		if !u.AI {
			units = append(units, u)
		}
	}
	return units
}

// uiScales are the UI size steps the settings window offers.
var uiScales = []int{80, 100, 115, 130, 145, 160}

// choice is one button of a choiceRow; the one that is on stays pressed.
type choice struct {
	label string
	on    bool
	click func()
}

// choiceRow draws a setting's name with its choices as a row of buttons below it that share
// the width w.
func (g *Game) choiceRow(dst *ebiten.Image, x, y, w int, title string, opts []choice) {
	g.label(dst, title, float64(x), float64(y), 14, muted)
	bw := (w - (len(opts)-1)*6) / len(opts)
	for i, o := range opts {
		bx := x + i*(bw+6)
		g.choiceBtn(dst, image.Rect(bx, y+24, bx+bw, y+56), o.label, o.on, o.click)
	}
}

// objectiveDuels names the stage's duels as "ally·supports 대 enemy", each with its guide colour, the rest counted.
func (g *Game) objectiveDuels(o core.Observation) []objectiveRow {
	st := g.S.Data.Stages[o.Stage]
	groups := st.DuelGroups()
	out := []objectiveRow{}
	for i, d := range st.Duels {
		if d.Unlisted {
			continue
		}
		if len(out) == 3 {
			out = append(out, objectiveRow{text: fmt.Sprintf("외 %d건", len(st.Duels)-3)})
			break
		}
		names := []string{}
		for _, id := range append([]string{d.Ally}, d.Supports...) {
			names = append(names, g.name(id))
		}
		out = append(out, objectiveRow{text: strings.Join(names, "·") + " 대 " + g.name(d.Enemy), duel: 1 + groups[i]})
	}
	return out
}

// objectiveRow is a line of the briefing; a duel's carries 1 + its duel group, for the guide's swatch, and a
// bonus goal's its state for the chip before it.
type objectiveRow struct {
	name, text, state string
	lit               bool
	duel              int
}

// subChipW is the width a bonus goal's state chip takes before its text, the gap included.
const subChipW = 96

func (g *Game) objectiveRows(o core.Observation, brief briefing) []objectiveRow {
	st := g.S.Data.Stages[o.Stage]
	rows := []objectiveRow{{name: "승리 조건", text: brief.win}, {name: "패배 조건", text: brief.lose}}
	for i, sub := range brief.subs {
		name := ""
		if i == 0 {
			name = "부목표"
		}
		rows = append(rows, objectiveRow{name: name, text: sub.text, state: sub.state, lit: sub.lit})
	}
	turn := fmt.Sprintf("%d턴째 · 제한 없음", o.Round)
	if st.TurnLimit > 0 {
		turn = fmt.Sprintf("%d턴째 · %d턴까지", o.Round, st.TurnLimit)
	}
	rows = append(rows, objectiveRow{name: "턴", text: turn})
	duels := g.objectiveDuels(o)
	if len(duels) == 0 {
		duels = []objectiveRow{{text: "없음"}}
	}
	duels[0].name = "일기토"
	rows = append(rows, duels...)
	return rows
}

func (g *Game) objectiveLayout(rows []objectiveRow, width, height int) (image.Rectangle, bool) {
	maxW, maxH := width*4/5, height*4/5
	fallback := image.Rectangle{}
	for w := 520; w <= maxW; w += 16 {
		h := 96 + 80 // title above, the buttons below
		longest := 0
		for _, row := range rows {
			tw := float64(w - 208)
			if row.state != "" {
				tw -= subChipW
			}
			wrapped := wrapWidth(row.text, g.face(19), tw)
			lines := strings.Count(wrapped, "\n") + 1
			longest = max(longest, lines)
			h += int(max(36, float64(lines)*19*1.45+9))
		}
		if h <= maxH {
			r := image.Rect(width/2-w/2, height/2-h/2, width/2+w/2, height/2+h/2)
			if fallback.Empty() {
				fallback = r
			}
			if longest <= 2 {
				return r, true
			}
		}
	}
	return fallback, !fallback.Empty()
}

func (g *Game) confirmLayout(message string, width, height int) (int, int, bool) {
	fallbackW, fallbackH := 0, 0
	for w := 480; w <= width*4/5; w += 16 {
		wrapped := wrapWidth(message, g.face(18), float64(w-64))
		lines := strings.Count(wrapped, "\n") + 1
		h := max(210, 44+lines*27+90)
		if h <= height*4/5 {
			if fallbackW == 0 {
				fallbackW, fallbackH = w, h
			}
			if lines <= 3 {
				return w, h, true
			}
		}
	}
	return fallbackW, fallbackH, fallbackW > 0
}

// objectiveBox fits the complete rendered briefing within 80% of the UI canvas.
func (g *Game) objectiveBox(w int) image.Rectangle {
	if g.S.Engine == nil {
		return image.Rect(w/2-260, 190, w/2+260, 510)
	}
	o := g.S.View()
	r, ok := g.objectiveLayout(g.objectiveRows(o, g.objective(o)), w, uiH)
	if !ok {
		return image.Rect(w/10, uiH/2-100, w*9/10, uiH/2+100)
	}
	return r
}

// promotionW is the promotion window's width.
const promotionW = 560

// promotionChipRows are the traits the new class teaches, in the rows of chips they take inside the window.
func (g *Game) promotionChipRows(p promotionUp) [][]string {
	var rows [][]string
	room := float64(promotionW - 80)
	x := room
	for _, id := range g.S.Data.Classes[p.to].Pool {
		w := traitIconSize + 6 + g.width(g.traitName(id), 14)
		if x+w > room {
			rows, x = append(rows, nil), 0
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], id)
		x += w + 18
	}
	return rows
}

// promotionTabsH is the room the class tabs of a seal's preview take under the title.
const promotionTabsH = 64

// promotionBox is the promotion window on a screen screenW wide, as tall as head (the tabs) and its rows of trait
// chips need.
func (g *Game) promotionBox(p promotionUp, screenW, head int) image.Rectangle {
	h := head + 340
	if n := len(g.promotionChipRows(p)); n > 0 {
		h = head + 376 + n*30
	}
	h += touchH + 12
	top := 150 - head/2
	return image.Rect(screenW/2-promotionW/2, top, screenW/2+promotionW/2, top+h)
}

// promotionWindow is the window of a promotion made: its body and 닫기.
func (g *Game) promotionWindow(dst *ebiten.Image, p promotionUp, t float64) {
	r := g.promotionBox(p, uiW, 0)
	g.titled(dst, r, "승급")
	g.promotionBody(dst, p, t, r, float64(r.Min.Y))
	g.closeRow(dst, r.Max.X-16, r.Max.Y-touchH-12, g.closeModal)
}

// promotionBody lays in window r from top the old look beside the new, the new class's name and description with
// its move and reach, the stats before and after with the gain, and the traits the new class teaches, each
// opening its description.
func (g *Game) promotionBody(dst *ebiten.Image, p promotionUp, t float64, r image.Rectangle, top float64) {
	from, to := g.S.Data.Classes[p.from], g.S.Data.Classes[p.to]
	x := float64(r.Min.X)
	look := func(a *sprite.Art, cx, alpha float64) {
		if a == nil {
			return
		}
		z := min(3, 96/max(1, a.TopOffset()))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(z, z)
		op.GeoM.Translate(math.Round(cx-a.Pivot[0]*z), math.Round(top+136-a.Pivot[1]*z))
		op.ColorScale.ScaleAlpha(float32(alpha))
		drawScaled(dst, a.Frame("SW", "idle", 0), op)
	}
	look(p.old, x+80, .55)
	g.centered(dst, "→", x+150, top+80, 24, gold)
	look(p.new, x+220, min(1, t/.4))
	g.label(dst, g.name(p.id), x+300, top+36, 18, muted)
	g.label(dst, from.Name+"  →", x+300, top+66, 16, muted)
	g.label(dst, to.Name, x+300, top+88, 30, gold)
	g.label(dst, wrapWidth(to.Desc, g.face(15), float64(r.Dx()-80)), x+40, top+154, 15, ink)
	move := fmt.Sprintf("이동 %d → %d", from.Move, to.Move)
	reach := fmt.Sprintf("사거리 %d → %d", from.Range, to.Range)
	g.label(dst, move, x+40, top+182, 14, muted)
	g.label(dst, reach, x+200, top+182, 14, muted)
	for i, s := range statRows {
		a, b := s.get(p.before), s.get(p.after)
		col, row := i%2, i/2
		px, py := x+40+float64(col*250), top+216+float64(row*28)
		g.label(dst, s.name, px, py, 15, muted)
		g.label(dst, fmt.Sprintf("%d → %d", a, b), px+70, py, 15, ink)
		if b != a {
			g.label(dst, fmt.Sprintf("%+d", b-a), px+180, py, 15, gainColor(b-a))
		}
	}
	if rows := g.promotionChipRows(p); len(rows) > 0 {
		rect(dst, float32(x+40), float32(top+332), float32(r.Dx()-80), 1, goldDim)
		g.label(dst, "익힐 수 있는 특성 · 눌러서 설명", x+40, top+342, 13, muted)
		for i, row := range rows {
			cx := x + 40
			for _, id := range row {
				cx = g.chip(dst, equipChip{"trait", id}, cx, top+370+float64(i*30), func() {
					g.modals = append([]*modal{{kind: "trait", trait: id}}, g.modals...)
				}) + 18
			}
		}
	}
}

// branches are the promotions the commands would make, each worked out on a copy of the game.
func (g *Game) branches(cmds []core.Command) []promotionUp {
	var out []promotionUp
	before := g.view()
	for _, c := range cmds {
		sb, err := core.Restore(g.S.Data, g.engine().Snapshot())
		if err != nil {
			continue
		}
		if _, err = sb.Apply(c); err != nil {
			continue
		}
		p := g.promotionBetween(&before, sb.Observe(), c.Target)
		p.cmd = c
		out = append(out, p)
	}
	return out
}

// gainColor is how a stat change reads: green up, red down.
func gainColor(d int) color.NRGBA {
	if d < 0 {
		return bad
	}
	return good
}

const difficultyRowH = 96

// difficulties are the difficulties a new game offers: the debug ones only in the debug mode.
func (g *Game) difficulties() []content.Difficulty {
	var out []content.Difficulty
	for _, v := range g.S.Data.Rules.Difficulties {
		if !v.Debug || g.Debug {
			out = append(out, v)
		}
	}
	return out
}

// difficultyWindow picks the difficulty of a new game, or of one started at the picked waypoint: each one a
// button with its name over its description.
func (g *Game) difficultyWindow(dst *ebiten.Image, x, y, w int) {
	for i, v := range g.difficulties() {
		id := v.ID
		top := y + 64 + i*difficultyRowH
		g.btn(dst, image.Rect(x+32, top, x+w-32, top+difficultyRowH-14), "", func() {
			g.closeOverlays()
			if g.jumpNode != "" {
				g.jump(g.jumpNode, id)
				return
			}
			g.request(session.Request{Op: "new", Seed: g.seed, Difficulty: id})
		})
		g.label(dst, v.Name, float64(x+52), float64(top+10), 22, ink)
		g.label(dst, wrapWidth(v.Desc, g.face(16), float64(w-104)), float64(x+52), float64(top+42), 16, muted)
	}
}
