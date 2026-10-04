package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

var overlayTitles = map[string]string{"menu": "메뉴", "save": "저장", "load": "불러오기", "settings": "설정",
	"confirm": "확인", "units": "부대 일람", "log": "전투 기록", "equipment": "장비", "deputy": "부관"}

func (g *Game) openOverlay(name string) {
	g.overlay, g.page = name, 0
	g.autoplay = false
	if name == "save" || name == "load" {
		g.slotInfo = g.readSlots()
	}
}

// ask opens a confirmation window; ok runs on 확인.
func (g *Game) ask(message string, ok func()) {
	g.question, g.answer = message, ok
	g.openOverlay("confirm")
}

// modal is a window shown after a replay, closed with a click.
type modal struct {
	kind  string // objective, levelup
	level levelUp
	t     float64
}

type levelUp struct {
	id            string
	from, to      int
	before, after core.Stats
	progress      float64
}

func (g *Game) drawOverlay(dst *ebiten.Image) {
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 150})
	g.buttons, g.blocks = nil, nil
	w, h := 560, 440
	switch g.overlay {
	case "confirm":
		w, h = 480, 210
	case "units", "log":
		w, h = 860, 640
	case "save", "load":
		w, h = 640, 480
	case "equipment", "deputy":
		w, h = 640, 560
	}
	x, y := Width/2-w/2, Height/2-h/2
	r := image.Rect(x, y, x+w, y+h)
	g.titled(dst, r, overlayTitles[g.overlay])
	g.btn(dst, image.Rect(x+w-96, y+16, x+w-18, y+44), "닫기", func() { g.overlay = "" })
	left := x + 32
	switch g.overlay {
	case "confirm":
		g.label(dst, g.question, float64(left), float64(y+44), 18, ink)
		g.btn(dst, image.Rect(left, y+h-62, left+190, y+h-28), "확인", func() {
			f := g.answer
			g.overlay = ""
			if f != nil {
				f()
			}
		})
		g.btn(dst, image.Rect(x+w-222, y+h-62, x+w-32, y+h-28), "취소", func() { g.overlay = "" })
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
		slots := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
		if op == "load" {
			slots = append([]string{"auto"}, slots...)
		}
		for i, s := range slots {
			slot := s
			info, ok := g.slotInfo[slot]
			name := "슬롯 " + slot
			if slot == "auto" {
				name = "자동 저장"
			}
			label := name + "   —   비어 있음"
			if ok {
				label = name + "   " + info
			}
			var click func()
			if op == "save" || ok {
				click = func() {
					if slot != "auto" {
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
		rows := []menuItem{
			{"적 AI: " + map[string]string{"auto": "자동 진행", "step": "한 명령씩"}[s.AI], func() {
				v := g.S.Settings
				v.AI = map[string]string{"auto": "step", "step": "auto"}[v.AI]
				g.request(session.Request{Op: "settings", Settings: &v})
			}},
			{"전투 기록: " + map[bool]string{true: "상세", false: "주요 사건만"}[s.Detail], func() {
				v := g.S.Settings
				v.Detail = !v.Detail
				g.request(session.Request{Op: "settings", Settings: &v})
			}},
			{"대사 속도: " + map[bool]string{true: "즉시", false: "한 글자씩"}[s.TextDelayMS == 0], func() {
				v := g.S.Settings
				v.TextDelayMS = map[bool]int{true: 20, false: 0}[v.TextDelayMS == 0]
				g.request(session.Request{Op: "settings", Settings: &v})
			}},
			{fmt.Sprintf("재생 속도: ×%d", g.speed), func() { g.speed = map[int]int{1: 2, 2: 4, 4: 1}[g.speed] }},
		}
		for i, it := range rows {
			g.btn(dst, image.Rect(left, y+70+i*54, x+w-32, y+70+i*54+42), it.label, it.click)
		}
		g.label(dst, "조작: 클릭 선택 · 우클릭/Esc 취소 · 우클릭 드래그/화살표 화면 이동 · 휠 확대\n"+
			"U 부대 일람 · T 위협 범위 · L 기록 · P 자동 진행 · E 진영 종료\n"+
			"Space 추천 한 명령 · F 재생 속도 · Enter 재생 건너뛰기 · F5/F9 저장/불러오기",
			float64(left), float64(y+300), 13, muted)
	case "units":
		g.unitList(dst, x, y, w, h)
	case "log":
		lines := g.history
		if len(lines) > 26 {
			lines = lines[len(lines)-26:]
		}
		g.label(dst, strings.Join(lines, "\n"), float64(left), float64(y+56), 14, ink)
	case "equipment", "deputy":
		g.equipmentWindow(dst, x, y, w, h)
	}
}

func (g *Game) unitList(dst *ebiten.Image, x, y, w, h int) {
	o := g.S.Engine.Observe()
	tabs := []string{"아군", "적군"}
	for i, t := range tabs {
		tab := i
		label := t
		if g.page == i {
			label = "▶ " + t
		}
		g.btn(dst, image.Rect(x+32+i*130, y+40, x+152+i*130, y+68), label, func() { g.page = tab })
	}
	faction := []string{"ally", "enemy"}[min(g.page, 1)]
	cols := []int{40, 160, 256, 296, 478, 560, 640}
	for i, h := range []string{"이름", "병종", "Lv", "병력", "병법치", "위치", "상태"} {
		g.label(dst, h, float64(x+cols[i]), float64(y+80), 13, muted)
	}
	for i, u := range unitsByFaction(o, faction) {
		if i >= 16 {
			break
		}
		row := y + 104 + i*31
		unit := u
		state := ""
		switch {
		case u.HP <= 0:
			state = "퇴각"
		case u.Done:
			state = "행동 완료"
		}
		for _, k := range sorted(u.Status) {
			state += " " + statusNames[k]
		}
		r := image.Rect(x+28, row-2, x+w-28, row+27)
		var click func()
		if u.HP > 0 {
			click = func() {
				g.overlay = ""
				g.selected = unit.ID
				g.focus(unit.X, unit.Y)
			}
		}
		g.btn(dst, r, "", click)
		c := ink
		if u.HP <= 0 {
			c = muted
		}
		g.label(dst, u.Name, float64(x+40), float64(row+3), 14, c)
		g.label(dst, u.Class, float64(x+160), float64(row+3), 14, c)
		g.label(dst, fmt.Sprint(u.Level), float64(x+256), float64(row+3), 14, c)
		k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
		bar(dst, float32(x+296), float32(row+10), 110, 8, k, hpColor(k))
		g.label(dst, fmt.Sprintf("%d", u.HP), float64(x+414), float64(row+3), 13, c)
		g.label(dst, fmt.Sprintf("%d/%d", u.MP, u.Stats.MaxMP), float64(x+478), float64(row+3), 13, c)
		g.label(dst, fmt.Sprintf("(%d,%d)", u.X, u.Y), float64(x+560), float64(row+3), 13, c)
		g.label(dst, state, float64(x+640), float64(row+3), 13, gold)
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
		where := map[string]string{"scenario": "이야기", "result": "전투 결과", "complete": "완료"}[s.Phase]
		if s.Phase == "battle" {
			where = fmt.Sprintf("%d턴", s.Round)
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
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 110})
	g.buttons, g.blocks = nil, nil
	g.buttons = append(g.buttons, button{image.Rect(0, 0, Width, Height), "", g.closeModal})
	switch m.kind {
	case "objective":
		o := g.S.Engine.Observe()
		st := g.stageInfo(o)
		win, lose := objective(st)
		r := image.Rect(Width/2-260, 230, Width/2+260, 470)
		g.titled(dst, r, "전투 목표")
		g.centered(dst, st.name, float64(Width/2), 262, 30, gold)
		g.label(dst, "승리 조건", float64(r.Min.X+48), 326, 16, muted)
		g.label(dst, win, float64(r.Min.X+160), 324, 19, ink)
		g.label(dst, "패배 조건", float64(r.Min.X+48), 362, 16, muted)
		g.label(dst, lose, float64(r.Min.X+160), 360, 19, ink)
		g.centered(dst, "클릭하여 시작", float64(Width/2), 424, 13, muted)
	case "levelup":
		l := m.level
		r := image.Rect(Width/2-250, 200, Width/2+250, 540)
		g.titled(dst, r, "레벨 업")
		g.portrait(dst, l.id, float64(r.Min.X+32), 236, 110)
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
		g.centered(dst, "클릭하여 계속", float64(Width/2), 512, 13, muted)
	}
}

func (g *Game) closeModal() {
	if len(g.modals) > 0 {
		g.modals = g.modals[1:]
	}
}

// result is the end-of-battle window.
func (g *Game) result(dst *ebiten.Image, o core.Observation) {
	g.buttons, g.blocks = nil, nil
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 120})
	st := g.stageInfo(o)
	r := image.Rect(Width/2-330, 130, Width/2+330, 720)
	if o.Result != "victory" {
		r = image.Rect(Width/2-280, 250, Width/2+280, 560)
		g.titled(dst, r, "패배")
		g.centered(dst, st.name, float64(Width/2), 290, 28, bad)
		g.centered(dst, "유비가 퇴각했습니다.", float64(Width/2), 346, 19, ink)
		g.btn(dst, image.Rect(r.Min.X+40, 420, r.Max.X-40, 456), "출전 준비부터 재도전", func() { g.request(session.Request{Op: "retry"}) })
		g.btn(dst, image.Rect(r.Min.X+40, 466, r.Min.X+260, 502), "불러오기", func() { g.openOverlay("load") })
		g.btn(dst, image.Rect(r.Max.X-260, 466, r.Max.X-40, 502), "시작 메뉴", func() { g.request(session.Request{Op: "title"}) })
		return
	}
	g.titled(dst, r, "승리")
	g.centered(dst, st.name+" 승리", float64(Width/2), 160, 30, gold)
	allies := unitsByFaction(o, "ally")
	best, bestXP := "", 0
	for _, u := range allies {
		if g.battleXP[u.ID] > bestXP {
			best, bestXP = u.ID, g.battleXP[u.ID]
		}
	}
	for i, u := range allies {
		y := 222 + i*76
		g.portrait(dst, u.ID, float64(r.Min.X+40), float64(y), 60)
		g.label(dst, u.Name, float64(r.Min.X+116), float64(y), 19, ink)
		lv := fmt.Sprintf("Lv %d", u.Level)
		if from, ok := g.startLevel[u.ID]; ok && from < u.Level {
			lv = fmt.Sprintf("Lv %d → %d", from, u.Level)
		}
		g.label(dst, lv, float64(r.Min.X+116), float64(y+28), 15, gold)
		g.label(dst, fmt.Sprintf("경험치 +%d", g.battleXP[u.ID]), float64(r.Min.X+260), float64(y+4), 15, ink)
		bar(dst, float32(r.Min.X+260), float32(y+32), 220, 8, u.XPProgress/100, xpC)
		if u.ID == best {
			g.label(dst, "최다 공훈", float64(r.Min.X+510), float64(y+14), 16, good)
		}
	}
	rewards := []string{}
	reward := g.S.Data.Stages[o.Stage].Reward
	keys := make([]string, 0, len(reward))
	for k := range reward {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rewards = append(rewards, fmt.Sprintf("%s ×%d", k, reward[k]))
	}
	g.label(dst, "획득 물품   "+strings.Join(rewards, "  ·  "), float64(r.Min.X+40), float64(r.Max.Y-110), 16, ink)
	g.btn(dst, image.Rect(r.Min.X+40, r.Max.Y-64, r.Max.X-40, r.Max.Y-26), "다음 이야기", func() {
		g.command(core.Command{Kind: "continue"})
	})
}
