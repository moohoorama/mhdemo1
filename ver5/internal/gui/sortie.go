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

// drawSortie draws the preparation over the battle node, filling the screen up to sheetMax.
func (g *Game) drawSortie(dst *ebiten.Image, o core.Observation) {
	w, h := sheetSize()
	x, y := (uiW-w)/2, (uiH-h)/2
	g.window(dst, image.Rect(x, y, x+w, y+h))
	g.block(image.Rect(x, y, x+w, y+h))
	ids := g.sortieCards(o)
	if !slices.Contains(ids, g.selected) && len(ids) > 0 {
		g.selected = ids[0]
	}
	st := g.S.Data.Stages[o.Stage]
	listW := min(340, w*9/25)
	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-16), goldDim)
	g.label(dst, "출전 준비", float64(x+16), float64(y+12), 18, gold)
	tx := float64(x+28) + g.width("출전 준비", 18)
	g.label(dst, fmt.Sprintf("출전 %d / %d", len(o.Deployment), st.Max), tx, float64(y+17), 13, muted)
	g.mBtn(dst, image.Rect(x+listW-160, y+4, x+listW-84, y+4+touchH), "목표", false, g.openObjective)
	g.mBtn(dst, image.Rect(x+listW-80, y+4, x+listW-8, y+4+touchH), "메뉴", false, func() { g.openOverlay("menu") })
	rect(dst, float32(x+8), float32(y+56), float32(listW-16), 1, goldDim)
	_, gearOwner, _, guide := g.gearStep()
	guide = guide && g.overlay() == ""
	g.listRows(dst, &g.party.officers, image.Rect(x+6, y+58, x+listW-4, y+h-6), len(ids), listRowH, func(i int, rr image.Rectangle) {
		g.sortieRow(dst, o, rr, ids[i])
		if guide && ids[i] == gearOwner && ids[i] != g.selected {
			g.guideRim(dst, rr.Inset(4))
		}
	}, func(i int) { g.selected, g.party.trait, g.party.tip = ids[i], "", "" })

	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	// the picked officer as in the officers' window, with a 부관 tab
	rx, right := x+listW+16, x+w-16
	g.label(dst, g.name(of.ID), float64(rx), float64(y+12), 18, gold)
	p := &g.party
	p.sortieTab = min(p.sortieTab, len(sortieTabs)-1)
	g.tabRow(dst, rx+int(g.width(g.name(of.ID), 18))+18, y+4, sortieTabs, p.sortieTab, func(i int) { p.sortieTab, p.tip = i, "" })
	if tx := rx + int(g.width(g.name(of.ID), 18)) + 18; guide && of.ID == gearOwner && p.sortieTab != 2 {
		g.guideRim(dst, g.tabRect(tx, y+4, sortieTabs, 2))
	}
	rect(dst, float32(rx), float32(y+52), float32(right-rx), 1, goldDim)
	body := image.Rect(rx, y+60, right, y+h-64)
	var acts []action
	switch p.sortieTab {
	case 0:
		g.officerBio(dst, o, of, body)
	case 1:
		acts = g.officerTraits(dst, of, body)
	case 2:
		if lead := g.servedBy(o, of.ID); lead != "" { // a deputy wears nothing
			c := image.Rect(body.Min.X, body.Min.Y, body.Max.X, body.Min.Y+64)
			g.card(dst, c)
			g.label(dst, g.name(lead)+"의 부관이라 장비를 들지 않습니다. 장비는 창고에 있습니다.", float64(c.Min.X+18), float64(c.Min.Y+22), 14, muted)
		} else {
			g.officerEquipment(dst, of, body)
		}
	case 3:
		acts = g.officerDeputy(dst, o, of, body)
	}
	belongings := action{"소지품", false, func() { // the seals can be used before the battle too
		p.items, p.table, p.insu = true, false, insuState{}
		g.openOverlay("party")
	}}
	g.actionRow(dst, right, y+h-60, append(append(acts, belongings), action{"출진 ›", true, g.startSortie})...)
}

// sortieTabs are the tabs of the picked officer in the sortie preparation: the officers' window's and 부관.
var sortieTabs = append(append([]string{}, officerTabs...), "부관")

// sortieRow is an officer's row: the state cell (출전, 부관 · whom, 대기), pressed to send out or call back, then the
// officer as in the officers' list.
func (g *Game) sortieRow(dst *ebiten.Image, o core.Observation, rr image.Rectangle, id string) {
	of := g.officer(o, id)
	if of == nil {
		return
	}
	picked := id == g.selected
	rowBack(dst, rr, picked)
	label, body, rim, fg := "대기", color.NRGBA{}, color.NRGBA{R: 90, G: 70, B: 48, A: 255}, faintInk
	switch lead := g.servedBy(o, id); {
	case lead != "":
		label, body, rim, fg = "부관 · "+g.name(lead), pillDep, depRim, depInk
	case slices.Contains(o.Deployment, id):
		label, body, rim, fg = "출전", pillOut, gold, gold
		if g.sortieLocked(id) {
			label = "출전 · 필수"
		}
	}
	pill := image.Rect(rr.Min.X+10, rr.Min.Y+(listRowH-36)/2, rr.Min.X+10+96, rr.Min.Y+(listRowH+36)/2)
	if body.A > 0 {
		rect(dst, float32(pill.Min.X), float32(pill.Min.Y), float32(pill.Dx()), float32(pill.Dy()), body)
	}
	strokeRect(dst, float32(pill.Min.X)+.5, float32(pill.Min.Y)+.5, float32(pill.Dx()-1), float32(pill.Dy()-1), 1, rim)
	g.label(dst, ellipsis(g, label, 13, pill.Dx()-8), float64(pill.Min.X)+(float64(pill.Dx())-g.width(ellipsis(g, label, 13, pill.Dx()-8), 13))/2, float64(pill.Min.Y)+(36-13*1.45)/2, 13, fg)
	g.addButton(image.Rect(rr.Min.X, rr.Min.Y, pill.Max.X+6, rr.Max.Y), label, func() { g.togglePrep(id) })
	row := rr
	row.Min.X = pill.Max.X - 2
	note, noteC := fmt.Sprintf("특성치 %d", of.Points), muted
	if g.hasAffordable(*of) {
		noteC = good
	}
	g.officerRow(dst, row, *of, false, note, noteC)
	if g.servedBy(o, id) == "" { // a deputy wears nothing
		g.missingGear(dst, of.Class, of.Equipment, float64(row.Min.X+70)+g.width(g.name(id), 16), float64(row.Min.Y+5))
	}
}

// togglePrep sends a waiting officer out or calls one back; the lord and the stage's required officers always go,
// and a deputy cannot go while serving.
func (g *Game) togglePrep(id string) {
	o := g.S.View()
	g.selected = id
	if lead := g.servedBy(o, id); lead != "" {
		g.toast(g.name(lead)+"의 부관이라 출전할 수 없습니다", bad)
		return
	}
	if g.sortieLocked(id) {
		g.toast("반드시 출전합니다", muted)
		return
	}
	deploy := slices.Clone(o.Deployment)
	if i := slices.Index(deploy, id); i >= 0 {
		deploy = slices.Delete(deploy, i, i+1)
	} else if len(deploy) >= g.S.Data.Stages[o.Stage].Max {
		g.toast("출전 인원이 찼습니다", bad)
		return
	} else {
		deploy = append(deploy, id)
	}
	g.request(session.Request{Op: "command", Command: core.Command{Kind: "deploy", Deployment: deploy}})
}
