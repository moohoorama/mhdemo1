package gui

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
	"srpg/internal/storage"
)

const (
	saveW, saveH = 860, 640
	saveRowH     = 36
)

// saveRow is one slot as the table shows it; ok is false for an empty slot, bad for one that cannot be read.
type saveRow struct {
	slot, kind, place string
	group             storage.SlotGroup
	num, chapter, lv  int
	at                time.Time
	ok, bad, legacy   bool
}

// saveList is the state of the save and load windows: the tab, the slot picked and the table.
type saveList struct {
	group storage.SlotGroup
	pick  string
	list  tableList
	rows  map[string]saveRow
}

// openSaves reads the slots and picks, for saving, the slot last used; for loading, the newest save, in its tab.
func (g *Game) openSaves(op string) {
	g.saves = saveList{rows: g.readSlots()}
	if op == "save" {
		g.saves.pick = fmt.Sprint(g.slot)
		return
	}
	var newest saveRow
	for _, r := range g.saves.rows {
		if r.ok && !r.bad && r.at.After(newest.at) {
			newest = r
		}
	}
	g.showSaveTab(newest.group)
	g.saves.pick = newest.slot
}

// showSaveTab switches the load window's tab; the automatic saves sort newest first.
func (g *Game) showSaveTab(group storage.SlotGroup) {
	g.saves.group, g.saves.pick, g.saves.list = group, "", tableList{}
	if group != storage.Manual {
		g.saves.list.order = tableSort{"at", 2}
	}
}

// readSlots describes every slot: story or battle, the chapter, where, when it was saved and the lord's level.
func (g *Game) readSlots() map[string]saveRow {
	d := g.S.Data
	out := map[string]saveRow{}
	for _, group := range []storage.SlotGroup{storage.Manual, storage.AutoScenario, storage.AutoBattle, storage.LegacyAuto} {
		for i, slot := range group.Slots() {
			r := saveRow{slot: slot, num: i + 1, group: group, legacy: group == storage.LegacyAuto}
			if !g.S.Store.Exists(slot) {
				if group != storage.LegacyAuto {
					out[slot] = r
				}
				continue
			}
			r.ok, r.at = true, g.S.Store.SavedAt(slot)
			s, err := g.S.Store.Load(slot)
			if err != nil {
				r.bad, r.place = true, "읽을 수 없음"
				if group == storage.LegacyAuto {
					r.group = storage.AutoBattle
				}
				out[slot] = r
				continue
			}
			if group == storage.LegacyAuto {
				r.group = storage.AutoGroup(s)
			}
			r.kind, r.place = "스토리", "이야기"
			switch s.Phase {
			case "deployment", "battle", "result":
				r.kind = "전투"
				r.place = d.Stages[min(max(0, s.Stage), len(d.Stages)-1)].Name
				switch s.Phase {
				case "deployment":
					r.place += " · 배치"
				case "battle":
					r.place += fmt.Sprintf(" · %d턴", s.Round)
				case "result":
					r.place += " · 결과"
				}
			case "complete":
				r.place = "완료"
			default:
				if x, ok := d.Explores[d.Node(s.Explore).Explore]; s.Explore != "" && ok {
					if a := x.Areas[s.Spot.Area]; a != nil {
						r.place = a.Name
					}
				} else if s.Place != "" {
					r.place = s.Place
				}
			}
			r.chapter = d.ChapterNumber(s.Node)
			r.lv = lordLevel(d.Characters, s)
			out[slot] = r
		}
	}
	return out
}

// lordLevel is the level of the lord (유비) in a saved game, 0 when he is not in it.
func lordLevel(defs map[string]content.Character, s core.State) int {
	for _, c := range s.Characters {
		if defs[c.Def].Lord {
			return c.Level
		}
	}
	return 0
}

func (g *Game) saveColumns() []column[saveRow] {
	num := func(key, label string, w int, v func(r *saveRow) int) column[saveRow] {
		return column[saveRow]{key: key, label: label, w: w, get: func(r *saveRow) (string, float64) {
			if !r.ok || r.bad || v(r) == 0 {
				return "", float64(v(r))
			}
			return fmt.Sprint(v(r)), float64(v(r))
		}}
	}
	return []column[saveRow]{
		{key: "slot", label: "칸", w: 80, get: func(r *saveRow) (string, float64) {
			if r.legacy {
				return fmt.Sprintf("이전 %d", r.num), float64(r.num)
			}
			return fmt.Sprint(r.num), float64(r.num)
		}},
		{key: "kind", label: "구분", w: 90, text: true, get: func(r *saveRow) (string, float64) { return r.kind, 0 }},
		num("chapter", "장", 70, func(r *saveRow) int { return r.chapter }),
		{key: "place", label: "장소", text: true, left: true, get: func(r *saveRow) (string, float64) {
			if !r.ok {
				return "비어 있음", 0
			}
			return r.place, 0
		}},
		{key: "at", label: "저장 시간", w: 170, get: func(r *saveRow) (string, float64) {
			if !r.ok {
				return "", 0
			}
			return r.at.Format("01/02 15:04"), float64(r.at.Unix())
		}},
		num("lv", "유비 Lv", 100, func(r *saveRow) int { return r.lv }),
	}
}

// saveWindow draws the save or load window (op) in the frame at (x, y).
func (g *Game) saveWindow(dst *ebiten.Image, op string, x, y, w, h int) {
	v := &g.saves
	left, top := x+32, y+60
	if op == "load" {
		for i, tab := range []struct {
			group storage.SlotGroup
			label string
		}{{storage.AutoScenario, "자동 시나리오"}, {storage.AutoBattle, "자동 전투"}, {storage.Manual, "수동"}} {
			tx := left + i*146
			g.choiceBtn(dst, image.Rect(tx, top, tx+140, top+34), tab.label, v.group == tab.group, func() { g.showSaveTab(tab.group) })
		}
		top += 46
	}
	group := storage.Manual
	if op == "load" {
		group = v.group
	}
	note := map[string]string{"save": "저장할 칸을 고르고 저장을 누르십시오.", "load": "불러올 저장을 고르고 불러오기를 누르십시오."}[op]
	switch group {
	case storage.AutoScenario:
		note = "출진 직전의 시나리오 진행을 저장합니다."
	case storage.AutoBattle:
		note = "전투 시작과 매 아군 턴 시작에 저장합니다."
	}
	g.label(dst, note, float64(left), float64(top), 14, muted)
	top += 30
	rows := make([]saveRow, 0, storage.SlotCount)
	for _, id := range group.Slots() {
		rows = append(rows, v.rows[id])
	}
	if group != storage.Manual {
		for _, id := range storage.LegacyAuto.Slots() {
			if r, ok := v.rows[id]; ok && r.group == group {
				rows = append(rows, r)
			}
		}
	}
	usable := func(r *saveRow) bool { return op == "save" || r.ok && !r.bad }
	drawTable(g, dst, &v.list, table[saveRow]{
		cols: g.saveColumns(), rowH: saveRowH, headH: 32, size: 15, center: true,
		rank:     func(r *saveRow) int { return b2i(!r.ok) },
		open:     usable,
		dim:      func(r *saveRow) bool { return !r.ok || r.bad },
		selected: func(r *saveRow) bool { return r.slot == v.pick },
		pick:     func(r *saveRow) { v.pick = r.slot },
	}, rows, image.Rect(left, top, x+w-32, y+h-80))

	var ok func()
	if r, found := v.rows[v.pick]; found && usable(&r) && (op == "load" || storage.IsManualSlot(r.slot)) {
		slot := v.pick
		ok = func() {
			if storage.IsManualSlot(slot) {
				fmt.Sscan(slot, &g.slot)
			}
			g.request(session.Request{Op: op, Slot: slot})
		}
	}
	label := "불러오기"
	if op == "save" {
		label = "저장"
	}
	g.overlayActs = append(g.overlayActs, action{label, true, ok})
}
