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

// The save and load windows list the slots in a table that sorts by its headers: the load window in two tabs, the
// automatic saves (before each sortie and at each allied turn of a battle) and the player's own; the save window
// only the player's. A row is picked first and 확인 saves or loads it.

const (
	saveW, saveH = 860, 640
	saveRowH     = 36
)

// saveRow is one slot as the table shows it; ok is false for an empty slot, bad for one that cannot be read.
type saveRow struct {
	slot, kind, place string
	num, chapter, lv  int
	at                time.Time
	ok, bad           bool
}

// saveList is the state of the save and load windows: the tab, the slot picked and the table.
type saveList struct {
	auto bool
	pick string
	list tableList
	rows map[string]saveRow
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
	g.showSaveTab(storage.IsAutoSlot(newest.slot))
	g.saves.pick = newest.slot
}

// showSaveTab switches the load window's tab; the automatic saves sort newest first.
func (g *Game) showSaveTab(auto bool) {
	g.saves.auto, g.saves.pick, g.saves.list = auto, "", tableList{}
	if auto {
		g.saves.list.order = tableSort{"at", 2}
	}
}

// readSlots describes every slot: story or battle, the chapter, where, when it was saved and the lord's level.
func (g *Game) readSlots() map[string]saveRow {
	d := g.S.Data
	out := map[string]saveRow{}
	for _, ids := range [][]string{storage.ManualSlots(), storage.AutoSlots()} {
		for i, slot := range ids {
			r := saveRow{slot: slot, num: i + 1}
			out[slot] = r
			if !g.S.Store.Exists(slot) {
				continue
			}
			r.ok, r.at = true, g.S.Store.SavedAt(slot)
			s, err := g.S.Store.Load(slot)
			if err != nil {
				r.bad, r.place = true, "읽을 수 없음"
				out[slot] = r
				continue
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
			r.chapter = d.ChapterIndex(s.Node) + 1
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
		{key: "slot", label: "칸", w: 60, get: func(r *saveRow) (string, float64) { return fmt.Sprint(r.num), float64(r.num) }},
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
		g.choiceBtn(dst, image.Rect(left, top, left+140, top+34), "자동 저장", v.auto, func() { g.showSaveTab(true) })
		g.choiceBtn(dst, image.Rect(left+146, top, left+286, top+34), "일반 저장", !v.auto, func() { g.showSaveTab(false) })
		top += 46
	}
	note := map[string]string{"save": "저장할 칸을 고르고 확인을 누르십시오.", "load": "불러올 저장을 고르고 확인을 누르십시오."}[op]
	if op == "load" && v.auto {
		note = "출진 직전과 전투의 아군 차례마다 저절로 저장됩니다."
	}
	g.label(dst, note, float64(left), float64(top), 14, muted)
	top += 30
	ids := storage.ManualSlots()
	if op == "load" && v.auto {
		ids = storage.AutoSlots()
	}
	rows := make([]saveRow, len(ids))
	for i, id := range ids {
		rows[i] = v.rows[id]
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
	g.btn(dst, image.Rect(x+w/2-200, y+h-62, x+w/2-10, y+h-24), "확인", ok)
	g.btn(dst, image.Rect(x+w/2+10, y+h-62, x+w/2+200, y+h-24), "취소", g.closeOverlay)
}
