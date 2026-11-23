package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// The sortie preparation comes between a battle node and its deployment: one row per officer with whether they go
// out, serve someone as deputy or wait, and the picked officer beside it with their gear, deputy and traits. 출진
// leaves for the deployment, where only places are set (docs/plan-party-ui.md).

// sortieForced tells whether the battle of node o skips the preparation: the stage sends everyone at once.
func (g *Game) sortieForced(o core.Observation) bool {
	return g.S.Data.Stages[o.Stage].Forced
}

// The preparation's sheet is as wide as sortieCols cards and the detail panel take, and never narrower than sheetMax;
// where the window is narrower the grid gives up columns first (sortieColumns), the detail panel keeps sortieDetailW.
const (
	sortieSideW   = 68  // the ◀ ▶ buttons and the margins to the sheet's right edge
	sortieDetailW = 440 // the detail panel, wide enough for its six tabs
)

// sortieSheetSize is the preparation's sheet in UI px.
func sortieSheetSize() (int, int) {
	_, h := sheetSize()
	w := 8 + sortieCols*sortieCellW + tableBarW + 4 + sortieSideW + sortieDetailW
	return min(uiW-16, max(sheetMax, w)), h
}

// drawSortie draws the preparation over the battle node, filling the screen up to sheetMax.
func (g *Game) drawSortie(dst *ebiten.Image, o core.Observation) {
	w, h := sortieSheetSize()
	x, y := (uiW-w)/2, (uiH-h)/2
	g.window(dst, image.Rect(x, y, x+w, y+h))
	g.block(image.Rect(x, y, x+w, y+h))
	ids := g.sortieCards(o)
	if !slices.Contains(ids, g.selected) && len(ids) > 0 {
		g.selected = ids[0]
	}
	st := g.S.Data.Stages[o.Stage]
	g.label(dst, "출전 준비", float64(x+16), float64(y+12), 18, gold)
	count := fmt.Sprintf("출전 %d / %d", g.sortieCount(o, o.Deployment), st.Max)
	if len(st.Reserve) > 0 {
		count += " (복병 제외)"
	}
	g.label(dst, count, float64(x+120), float64(y+17), 13, muted)
	foot := y + h - 60
	leftMax := min(x+w-sortieSideW-sortieDetailW, x+8+sortieCols*sortieCellW+tableBarW+4) // the cards a row, the detail panel keeps its room
	left := image.Rect(x+8, y+58, leftMax, foot-8)
	g.sortieGrid(dst, o, left)
	g.mBtn(dst, image.Rect(x+8, foot, x+96, foot+touchH), "목표", false, g.openObjective)
	g.mBtn(dst, image.Rect(x+104, foot, x+192, foot+touchH), "메뉴", false, func() { g.openOverlay("menu") })
	ax := left.Max.X + 4
	var add, remove func()
	lead := g.servedBy(o, g.selected)
	if lead == "" {
		if slices.Contains(o.Deployment, g.selected) {
			remove = func() { g.togglePrep(g.selected) }
		} else {
			add = func() { g.togglePrep(g.selected) }
		}
	}
	g.mBtn(dst, image.Rect(ax, y+62, ax+40, y+102), "◀", false, add)
	g.mBtn(dst, image.Rect(ax, foot-48, ax+40, foot-8), "▶", false, remove)
	rx, right := ax+48, x+w-16
	p := &g.party
	views := []string{"기본", "전투력", "열전", "장비", "특성", "부관"}
	p.sortieView = min(p.sortieView, len(views)-1)
	g.tabRow(dst, rx, y+4, views, p.sortieView, func(i int) { p.sortieView = i; p.officerList.order = tableSort{}; p.tip = "" })
	if g.overlay() == "" && p.sortieView != 3 && len(g.equipmentAdvice(o, g.selected)) > 0 {
		recommendationRim(dst, g.tabRect(rx, y+4, views, 3).Inset(4))
	}
	rect(dst, float32(rx), float32(y+52), float32(right-rx), 1, goldDim)
	body := image.Rect(rx, y+60, right, foot-4)
	var acts []action
	if p.sortieView < 3 {
		g.sortieTable(dst, o, ids, body)
	} else if of := g.officer(o, g.selected); of != nil {
		nameInk := gold
		if lead != "" {
			nameInk = dimInk
		}
		g.label(dst, g.name(of.ID), float64(body.Min.X+8), float64(body.Min.Y+4), 16, nameInk)
		body.Min.Y += 30
		switch p.sortieView {
		case 3:
			p.sortieTab = 2
			if lead != "" {
				g.label(dst, g.name(lead)+"의 부관이라 장비를 들지 않습니다.", float64(body.Min.X), float64(body.Min.Y+16), 14, muted)
			} else {
				g.officerEquipment(dst, of, body)
			}
		case 4:
			p.sortieTab = 1
			acts = g.officerTraits(dst, of, body)
		case 5:
			p.sortieTab = 3
			acts = g.officerDeputy(dst, o, of, body)
		}
	}
	belongings := action{"소지품", false, func() { // the seals can be used before the battle too
		p.items, p.table, p.insu = true, false, insuState{}
		g.openOverlay("party")
	}}
	// the tab's actions on the left, apart from 소지품 and 출진 at the right
	row := []action{belongings, {"출진 ›", true, g.startSortie}}
	if left := g.actionRow(dst, right, y+h-60, row...); len(acts) > 0 {
		g.actionRow(dst, left-closeGap, y+h-60, acts...)
	}
	if _, ok := g.promoteGuide(); ok && g.overlay() == "" {
		g.promoteRim(dst, g.rowActionRect(right, y+h-60, belongings.label, row...))
	}
	if g.overlay() == "" && g.deputyInfo.id != "" {
		g.chipWindow(dst, "sortie:deputy-info", g.deputyInfo, func() { g.deputyInfo = equipChip{} })
	}
}

// sortieTabs are the tabs of the picked officer in the sortie preparation: the officers' window's and 부관.
var sortieTabs = officerManagementTabs

// sortieRow is an officer's row: the state pill (출전, 필수, 부관 over whom, 대기), pressed to send out or call
// back, then the officer as in the officers' list, the missing gear after the class.
func (g *Game) sortieRow(dst *ebiten.Image, o core.Observation, rr image.Rectangle, id string) {
	of := g.officer(o, id)
	if of == nil {
		return
	}
	picked := id == g.selected
	lead := g.servedBy(o, id)
	label, body, rim, fg := "대기", color.NRGBA{}, color.NRGBA{R: 90, G: 70, B: 48, A: 255}, faintInk
	switch {
	case lead != "":
		label, body, rim, fg = "부관", pillDep, depRim, depInk
	case slices.Contains(o.Deployment, id):
		label, body, rim, fg = "출전", pillOut, gold, gold
		if g.sortieReserve(o, id) {
			label = "복병"
		} else if g.sortieLocked(id) {
			label = "필수"
		}
	}
	// the officer's lines right of the state pill
	pill := image.Rect(rr.Min.X+10, rr.Min.Y+(listRowH-40)/2, rr.Min.X+10+sortiePillW, rr.Min.Y+(listRowH+40)/2)
	row := rr
	row.Min.X = pill.Max.X - 2
	g.officerGround(dst, o, rr, id, picked)
	note, noteC := fmt.Sprintf("특성치 %d", of.Points), muted
	if g.hasAffordable(*of) {
		noteC = good
	}
	if g.sortieReserve(o, id) && lead == "" {
		note, noteC = "복병 · 전투 중 출현", gold
	}
	classEnd := g.officerLines(dst, o, row, *of, true, note, noteC)
	if lead == "" { // a deputy wears nothing
		g.missingGear(dst, of.Class, of.Equipment, classEnd+6, float64(row.Min.Y+28))
	}
	if body.A > 0 {
		rect(dst, float32(pill.Min.X), float32(pill.Min.Y), float32(pill.Dx()), float32(pill.Dy()), body)
	}
	strokeRect(dst, float32(pill.Min.X)+.5, float32(pill.Min.Y)+.5, float32(pill.Dx()-1), float32(pill.Dy()-1), 1, rim)
	cx := float64(pill.Min.X+pill.Max.X) / 2
	if lead != "" { // 부관 over whom it serves
		who := ellipsis(g, g.name(lead), 12, pill.Dx()-8)
		g.label(dst, label, cx-g.width(label, 13)/2, float64(pill.Min.Y+2), 13, fg)
		g.label(dst, who, cx-g.width(who, 12)/2, float64(pill.Min.Y+21), 12, fg)
	} else {
		g.label(dst, label, cx-g.width(label, 13)/2, float64(pill.Min.Y)+(40-13*1.45)/2, 13, fg)
	}
	tag := label
	if lead != "" {
		tag = "부관 · " + g.name(lead)
	}
	g.addButton(image.Rect(rr.Min.X, rr.Min.Y, pill.Max.X+6, rr.Max.Y), tag, func() { g.togglePrep(id) })
}

// sortiePillW is how wide the state pill of a sortie row is.
const sortiePillW = 64

// togglePrep sends a waiting officer out or calls one back; the lord and the stage's required officers always go,
// and a deputy cannot go while serving.
func (g *Game) togglePrep(id string) {
	o := g.S.View()
	g.selected = id
	if lead := g.servedBy(o, id); lead != "" {
		g.toast(g.name(lead)+"의 부관이라 출전할 수 없습니다", bad)
		return
	}
	if g.sortieReserve(o, id) {
		g.toast("복병은 전투 중에 나타나 반드시 출전합니다", muted)
		return
	}
	if g.sortieLocked(id) {
		g.toast("반드시 출전합니다", muted)
		return
	}
	deploy := slices.Clone(o.Deployment)
	if i := slices.Index(deploy, id); i >= 0 {
		deploy = slices.Delete(deploy, i, i+1)
	} else if g.sortieCount(o, deploy) >= g.S.Data.Stages[o.Stage].Max {
		g.toast("출전 인원이 찼습니다", bad)
		return
	} else {
		deploy = append(deploy, id)
	}
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "deploy", Deployment: deploy}})
}
