package gui

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
	"srpg/internal/storage"
)

var overlayTitles = map[string]string{"menu": "메뉴", "save": "저장", "load": "불러오기", "settings": "설정",
	"confirm": "확인", "equipment": "장비", "deputy": "부관", "roster": "병력일람", "shop": "상점", "party": "부대"}

func (g *Game) popupPending() bool {
	return g.overlay != "" || len(g.modals) > 0 || len(g.learn) > 0 ||
		(g.S.Engine != nil && g.S.Screen == "playing" && len(g.S.View().Learners) > 0)
}

func (g *Game) openOverlay(name string) bool {
	if g.overlay == "" && g.popupPending() {
		return false
	}
	g.overlay, g.page = name, 0
	g.equip = equipState{}
	g.autoplay = false
	if name == "save" || name == "load" {
		g.slotInfo = g.readSlots()
	}
	return true
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
	kind   string // objective, levelup, trait, acquire
	manual bool   // an objective opened from the panel rather than at the battle's start
	level  levelUp
	trait  string
	gains  []gain    // acquire: the cards
	piece  string    // acquire: the card whose window is open
	chip   equipChip // acquire: the trait open over that window
	t      float64
}

type levelUp struct {
	id            string
	from, to      int
	before, after core.Stats
	progress      float64
}

func (g *Game) drawOverlay(dst *ebiten.Image) {
	g.buttons, g.blocks = nil, nil
	w, h := 560, 440
	switch g.overlay {
	case "confirm":
		w, h, _ = g.confirmLayout(g.question, uiW, uiH)
	case "save":
		w, h = 640, 480
	case "load":
		w, h = 760, 660
	case "equipment":
		w, h = 760, 80+3*(equipCardH+14)+10
	case "deputy":
		w, h = 640, 560
	case "settings":
		w, h = 640, 732
	case "roster":
		w, h = uiW-100, uiH-90
	case "shop":
		w, h = 820, 640
	case "party":
		w, h = 1100, 720
	}
	// the window in a frame with room for its title plate and shadow
	g.popup(dst, "overlay:"+g.overlay, 150, w+40, h+40, image.Rect(0, 0, w+40, h+40), func(dst *ebiten.Image) { g.overlayWindow(dst, w, h) })
	if g.overlay == "equipment" && g.equip.slot != "" {
		g.drawEquipPicker(dst)
	}
	if g.overlay == "equipment" && g.equip.info.id != "" {
		g.drawEquipInfo(dst)
	}
}

func (g *Game) overlayWindow(dst *ebiten.Image, w, h int) {
	x, y := uiW/2-w/2, uiH/2-h/2
	r := image.Rect(x, y, x+w, y+h)
	if g.overlay == "confirm" {
		g.titledPattern(dst, r, g.confirmTitle)
	} else {
		g.titled(dst, r, overlayTitles[g.overlay])
	}
	g.btn(dst, image.Rect(x+w-96, y+16, x+w-18, y+44), "닫기", func() { g.overlay = "" })
	left := x + 32
	switch g.overlay {
	case "confirm":
		g.label(dst, wrapWidth(g.question, g.face(18), float64(w-64)), float64(left), float64(y+44), 18, ink)
		g.btn(dst, image.Rect(left, y+h-62, left+190, y+h-28), g.yes, func() {
			f := g.answer
			g.overlay = ""
			if f != nil {
				f()
			}
		})
		g.btn(dst, image.Rect(x+w-222, y+h-62, x+w-32, y+h-28), g.no, func() { g.overlay = "" })
	case "menu":
		items := []menuItem{
			{"게임으로", func() { g.overlay = ""; g.request(session.Request{Op: "resume"}) }},
			{"저장", func() { g.openOverlay("save") }},
			{"불러오기", func() { g.openOverlay("load") }},
			{"설정", func() { g.openOverlay("settings") }},
			{"시작 메뉴", func() {
				g.request(session.Request{Op: "title"})
				if g.S.Screen == "title" {
					g.overlay = ""
				}
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
		op := g.overlay
		g.label(dst, map[string]string{"save": "저장할 슬롯을 고르십시오. 덮어쓸 때는 확인합니다.", "load": "불러올 슬롯을 고르십시오."}[op], float64(left), float64(y+56), 14, muted)
		slots := storage.ManualSlots()
		if op == "load" {
			slots = append(storage.AutoSlots(), slots...)
		}
		for i, s := range slots {
			slot := s
			info, ok := g.slotInfo[slot]
			name := "슬롯 " + slot
			if !storage.IsManualSlot(slot) {
				name = "자동 저장 " + strings.TrimPrefix(slot, "auto")
			}
			label := name + "   —   비어 있음"
			if ok {
				label = name + "   " + info
			}
			var click func()
			if op == "save" || ok {
				click = func() {
					if storage.IsManualSlot(slot) {
						fmt.Sscan(slot, &g.slot)
					}
					g.request(session.Request{Op: op, Slot: slot})
					if g.overlay != "confirm" {
						g.overlay = ""
					}
				}
			}
			g.btn(dst, image.Rect(left, y+84+i*32, x+w-32, y+84+i*32+28), label, click)
		}
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
			speeds = append(speeds, choice{fmt.Sprintf("×%d", n), g.speed == n, func() { g.speed = n }})
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
		rows := []struct {
			title string
			opts  []choice
		}{
			{"적 AI", []choice{
				{"자동 진행", s.AI == "auto", set(func(v *storage.Settings) { v.AI = "auto" })},
				{"한 명령씩", s.AI == "step", set(func(v *storage.Settings) { v.AI = "step" })},
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
			{"재생 속도", speeds},
			{"UI 크기", scales},
			{"데미지 배율", damages},
			{"경험치 배율", xps},
			{"전원 반격", []choice{
				{"켜기", s.CounterAll, set(func(v *storage.Settings) { v.CounterAll = true })},
				{"끄기", !s.CounterAll, set(func(v *storage.Settings) { v.CounterAll = false })},
			}},
		}
		for i, it := range rows {
			g.choiceRow(dst, left, y+64+i*64, x+w-32-left, it.title, it.opts)
		}
		g.label(dst, "조작: 클릭 선택 · 우클릭/Esc 취소 · 드래그/화면 가장자리/화살표 화면 이동 · 휠 확대\n"+
			"Space 추천 한 명령 · F 재생 속도 · Enter 재생 건너뛰기 · F5/F9 저장/불러오기",
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
	}
}

// focus centres the camera on a cell.
func (g *Game) focus(u, v int) {
	if g.field != nil {
		g.camX, g.camY = g.field.Center(float64(u), float64(v))
	}
}

// readSlots describes each existing save: chapter, battle or story, and when it was saved.
func (g *Game) readSlots() map[string]string {
	out := map[string]string{}
	for _, slot := range g.S.Store.Slots() {
		s, err := g.S.Store.Load(slot)
		if err != nil {
			out[slot] = "읽을 수 없음"
			continue
		}
		stage := g.S.Data.Stages[min(max(0, s.Stage), len(g.S.Data.Stages)-1)]
		where := map[string]string{"scenario": "이야기", "deployment": "아군 배치", "result": "전투 결과", "complete": "완료"}[s.Phase]
		if s.Phase == "battle" {
			where = fmt.Sprintf("%d턴 · %s", s.Round, factionName(s.Turn))
		}
		when := ""
		if fi, err := os.Stat(filepath.Join(g.S.Store.Dir, "slot-"+slot+".json")); err == nil {
			when = fi.ModTime().Format("01/02 15:04")
		}
		out[slot] = fmt.Sprintf("제%d장 %s · %s · %s", s.Stage+1, stage.Name, where, when)
	}
	return out
}

// drawModal draws the first queued modal; any click closes it.
func (g *Game) drawModal(dst *ebiten.Image) {
	m := g.modals[0]
	g.buttons, g.blocks = nil, nil
	if m.kind == "acquire" {
		g.drawAcquire(dst, m)
		return
	}
	g.addButton(image.Rect(0, 0, uiW, uiH), "", g.closeModal)
	if m.kind == "objective" {
		box := g.objectiveBox(uiW)
		g.popup(dst, fmt.Sprintf("modal:%p", m), 110, uiW, uiH, box.Inset(-24), func(dst *ebiten.Image) { g.modalWindow(dst, m) })
		return
	}
	box := map[string]image.Rectangle{
		"levelup": image.Rect(Width/2-250, 200, Width/2+250, 540),
		"trait":   image.Rect(Width/2-300, 220, Width/2+300, 560),
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
			g.titledPattern(dst, r, "작전 개요")
			g.label(dst, "표시 공간이 부족합니다. UI 크기를 줄이세요.", float64(r.Min.X+32), float64(r.Min.Y+56), 18, ink)
			return
		}
		g.titledPattern(dst, r, "작전 개요")
		g.centered(dst, st.Name, float64(uiW/2), float64(r.Min.Y+32), 30, gold)
		y := float64(r.Min.Y + 96)
		for _, row := range g.objectiveRows(o, brief) {
			wrapped := wrapWidth(row.text, g.face(19), float64(r.Dx()-208))
			g.label(dst, row.name, float64(r.Min.X+48), y+2, 16, muted)
			g.label(dst, wrapped, float64(r.Min.X+160), y, 19, ink)
			y += max(36, float64(strings.Count(wrapped, "\n")+1)*19*1.45+9)
		}
		hint := "클릭하여 시작"
		if m.manual {
			hint = "클릭하여 닫기"
		}
		g.centered(dst, hint, float64(uiW/2), float64(r.Max.Y-34), 13, muted)
	case "levelup":
		l := m.level
		r := image.Rect(uiW/2-250, 200, uiW/2+250, 540)
		g.titled(dst, r, "레벨 업")
		g.portrait(dst, l.id, "", float64(r.Min.X+32), 236, 110)
		g.label(dst, g.name(l.id), float64(r.Min.X+164), 232, 26, gold)
		g.label(dst, fmt.Sprintf("Lv %d  →  Lv %d", l.from, l.to), float64(r.Min.X+164), 274, 20, ink)
		g.label(dst, fmt.Sprintf("특성치 +%d", 100*(l.to-l.from)), float64(r.Min.X+164), 306, 15, good)
		k := min(1, m.t/.8) * l.progress / 100
		bar(dst, float32(r.Min.X+164), 334, 220, 8, k, xpC)
		g.label(dst, fmt.Sprintf("EXP %.0f / 100", l.progress), float64(r.Min.X+392), 326, 13, muted)
		for i, s := range statRows {
			a, b := s.get(l.before), s.get(l.after)
			col, row := i%2, i/2
			px, py := float64(r.Min.X+40+col*230), float64(366+row*30)
			g.label(dst, s.name, px, py, 15, muted)
			g.label(dst, fmt.Sprint(b), px+70, py, 15, ink)
			if b != a {
				g.label(dst, fmt.Sprintf("+%d", b-a), px+130, py, 15, good)
			}
		}
		g.centered(dst, "클릭하여 계속", float64(uiW/2), 512, 13, muted)
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
		g.centered(dst, "클릭하여 닫기", float64(uiW/2), float64(r.Max.Y-36), 13, muted)
	}
}

// closeModal closes the first modal, or the top window stacked inside it.
func (g *Game) closeModal() {
	if len(g.modals) > 0 && g.modals[0].closeAcquire() {
		return
	}
	if len(g.modals) > 0 {
		g.modals = g.modals[1:]
	}
}

// result is the end-of-battle window.
func (g *Game) result(dst *ebiten.Image, o core.Observation) {
	g.buttons, g.blocks = nil, nil
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
		reason := g.lordName() + " 퇴각 또는 아군 전멸"
		if st.TurnLimit > 0 && o.Round > st.TurnLimit {
			reason = fmt.Sprintf("%d턴 제한 초과", st.TurnLimit)
		}
		g.centered(dst, reason, float64(uiW/2), 346, 19, ink)
		g.btn(dst, image.Rect(r.Min.X+40, 420, r.Max.X-40, 456), "출전 준비부터 재도전", func() { g.request(session.Request{Op: "retry"}) })
		g.btn(dst, image.Rect(r.Min.X+40, 466, r.Min.X+260, 502), "불러오기", func() { g.openOverlay("load") })
		g.btn(dst, image.Rect(r.Max.X-260, 466, r.Max.X-40, 502), "시작 메뉴", func() { g.request(session.Request{Op: "title"}) })
		return
	}
	g.titled(dst, r, "승리")
	g.centered(dst, st.Name+" 승리", float64(uiW/2), 160, 30, gold)
	allies := resultUnits(o)
	best, bestXP := "", 0
	for _, u := range allies {
		if g.battleXP[u.ID] > bestXP {
			best, bestXP = u.ID, g.battleXP[u.ID]
		}
	}
	pages := max(1, (len(allies)+4)/5)
	g.resultPage = max(0, min(g.resultPage, pages-1))
	start := g.resultPage * 5
	for i, u := range allies[start:min(start+5, len(allies))] {
		y := 222 + i*76
		g.portrait(dst, u.ID, u.Class, float64(r.Min.X+40), float64(y), 60)
		g.label(dst, u.Name, float64(r.Min.X+116), float64(y), 19, ink)
		lv := fmt.Sprintf("Lv %d", u.Level)
		if from, ok := g.startLevel[u.ID]; ok && from < u.Level {
			lv = fmt.Sprintf("Lv %d → %d", from, u.Level)
		}
		g.label(dst, lv, float64(r.Min.X+116), float64(y+28), 15, gold)
		g.label(dst, fmt.Sprintf("경험치 +%s", xpText(g.battleXP[u.ID])), float64(r.Min.X+260), float64(y+4), 15, ink)
		bar(dst, float32(r.Min.X+260), float32(y+32), 220, 8, u.XPProgress/100, xpC)
		if u.ID == best {
			g.label(dst, "최다 공훈", float64(r.Min.X+510), float64(y+14), 16, good)
		}
	}
	if pages > 1 {
		g.btn(dst, image.Rect(r.Min.X+40, 604, r.Min.X+140, 636), "이전", func() { g.resultPage = max(0, g.resultPage-1) })
		g.centered(dst, fmt.Sprintf("%d / %d", g.resultPage+1, pages), float64(uiW/2), 610, 14, muted)
		g.btn(dst, image.Rect(r.Max.X-140, 604, r.Max.X-40, 636), "다음", func() { g.resultPage = min(pages-1, g.resultPage+1) })
	}
	rewards := []string{}
	reward := g.S.Data.Stages[o.Stage].Reward
	keys := make([]string, 0, len(reward))
	for k := range reward {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rewards = append(rewards, fmt.Sprintf("%s ×%d", g.itemName(k), reward[k]))
	}
	g.label(dst, "획득 물품   "+strings.Join(rewards, "  ·  "), float64(r.Min.X+40), float64(r.Max.Y-98), 14, ink)
	g.btn(dst, image.Rect(r.Min.X+40, r.Max.Y-64, r.Max.X-40, r.Max.Y-26), "다음 이야기", func() {
		g.command(core.Command{Kind: "continue"})
	})
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

// objectiveDuels names the stage's duels as "ally 대 enemy", the rest counted.
func (g *Game) objectiveDuels(o core.Observation) []string {
	out := []string{}
	for _, d := range g.S.Data.Stages[o.Stage].Duels {
		if len(out) == 3 {
			out = append(out, fmt.Sprintf("외 %d건", len(g.S.Data.Stages[o.Stage].Duels)-3))
			break
		}
		out = append(out, g.name(d.Ally)+" 대 "+g.name(d.Enemy))
	}
	return out
}

type objectiveRow struct{ name, text string }

func (g *Game) objectiveRows(o core.Observation, brief briefing) []objectiveRow {
	st := g.S.Data.Stages[o.Stage]
	rows := []objectiveRow{{"승리 조건", brief.win}, {"패배 조건", brief.lose}}
	for i, sub := range brief.subs {
		name, mark := "", ""
		if i == 0 {
			name = "부목표"
		}
		if sub.done {
			mark = " (달성)"
		}
		rows = append(rows, objectiveRow{name, sub.text + mark})
	}
	turn := fmt.Sprintf("%d턴째 · 제한 없음", o.Round)
	if st.TurnLimit > 0 {
		turn = fmt.Sprintf("%d턴째 · %d턴까지", o.Round, st.TurnLimit)
	}
	rows = append(rows, objectiveRow{"턴", turn})
	duels := g.objectiveDuels(o)
	if len(duels) == 0 {
		duels = []string{"없음"}
	}
	for i, duel := range duels {
		name := ""
		if i == 0 {
			name = "일기토"
		}
		rows = append(rows, objectiveRow{name, duel})
	}
	return rows
}

func (g *Game) objectiveLayout(rows []objectiveRow, width, height int) (image.Rectangle, bool) {
	maxW, maxH := width*4/5, height*4/5
	fallback := image.Rectangle{}
	for w := 520; w <= maxW; w += 16 {
		h := 96 + 56
		longest := 0
		for _, row := range rows {
			wrapped := wrapWidth(row.text, g.face(19), float64(w-208))
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
