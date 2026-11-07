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
	listW := min(470, w/2)
	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-16), goldDim)
	g.label(dst, "출전 준비", float64(x+16), float64(y+12), 18, gold)
	tx := float64(x+28) + g.width("출전 준비", 18)
	g.label(dst, fmt.Sprintf("출전 %d / %d", len(o.Deployment), st.Max), tx, float64(y+17), 13, muted)
	g.mBtn(dst, image.Rect(x+listW-168, y+4, x+listW-86, y+4+touchH), "목표", false, g.openObjective)
	g.mBtn(dst, image.Rect(x+listW-80, y+4, x+listW-8, y+4+touchH), "메뉴", false, func() { g.openOverlay("menu") })
	rect(dst, float32(x+8), float32(y+56), float32(listW-16), 1, goldDim)
	g.listRows(dst, &g.party.officers, image.Rect(x+6, y+58, x+listW-4, y+h-6), len(ids), listRowH, func(i int, rr image.Rectangle) {
		g.sortieRow(dst, o, rr, ids[i])
	}, func(i int) { g.selected = ids[i] })

	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	rx, right := x+listW+16, x+w-16
	g.sortieBrief(dst, o, of, image.Rect(rx, y+10, right, y+h-128))
	going := slices.Contains(o.Deployment, of.ID)
	var deputy func()
	if going && g.S.Engine.CanDeputize(of.ID) {
		deputy = func() { g.openOverlay("deputy") }
	}
	var equip, traits func()
	if g.servedBy(o, of.ID) == "" {
		equip = func() { g.openOverlay("equipment") }
	}
	traits = func() {
		g.party.items, g.party.table, g.party.tab, g.party.trait = false, false, 1, ""
		g.openOverlay("party")
	}
	detail := func() {
		g.party.items, g.party.table, g.party.tab = false, false, 0
		g.openOverlay("party")
	}
	bw := (right - rx - 18) / 4
	for i, b := range []struct {
		label string
		click func()
	}{{"장비", equip}, {"부관", deputy}, {"특성", traits}, {"상세", detail}} {
		bx := rx + i*(bw+6)
		g.mBtn(dst, image.Rect(bx, y+h-120, bx+bw, y+h-120+touchH), b.label, false, b.click)
	}
	g.actionRow(dst, right, y+h-60, action{"출진 ›", true, g.startSortie})
}

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

// sortieBrief is the picked officer in short: the face, name and state, the deputy's face, the hexagon (the lead
// alone dashed in gold when a deputy serves) and the traits the unit carries.
func (g *Game) sortieBrief(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) {
	x, y := float64(r.Min.X), float64(r.Min.Y)
	g.portrait(dst, of.ID, of.Class, x, y, 48)
	g.label(dst, g.name(of.ID), x+60, y-2, 20, gold)
	state := "대기"
	switch {
	case g.servedBy(o, of.ID) != "":
		state = g.name(g.servedBy(o, of.ID)) + "의 부관"
	case slices.Contains(o.Deployment, of.ID):
		state = "출전"
	}
	g.label(dst, fmt.Sprintf("%s · Lv%d · %s", g.className(of.Class), of.Level, state), x+60, y+28, 12, muted)
	if of.Deputy != "" {
		d := g.officer(o, of.Deputy)
		dx := float64(r.Max.X) - 36 - g.width(g.name(of.Deputy), 14) - 8
		if d != nil {
			g.portrait(dst, d.ID, d.Class, dx, y+6, 36)
		}
		g.label(dst, "부관", dx+44, y+4, 12, muted)
		g.label(dst, g.name(of.Deputy), dx+44, y+20, 14, ink)
	}
	stats, _, err := g.S.Engine.DeputyStats(of.ID, of.Deputy)
	if err != nil {
		return
	}
	var alone *core.Stats
	if of.Deputy != "" {
		if s, _, err := g.S.Engine.DeputyStats(of.ID, ""); err == nil {
			alone = &s
		}
	}
	radius := min(46.0, float64(r.Dy()-90)/2)
	g.hexChart(dst, x+radius+60, y+radius+84, radius, stats, alone, g.rosterBest(o))
	tx, ty := x+2*radius+140, y+64
	traits := slices.Clone(of.Learned)
	for _, slot := range equipSlots {
		if id, ok := of.Equipment[slot]; ok {
			traits = append(traits, g.S.Data.Equipment[id].Traits...)
		}
	}
	if of.Deputy != "" {
		for _, t := range g.S.Engine.DeputyTraits(of.Deputy) {
			if !slices.Contains(traits, t) {
				traits = append(traits, t)
			}
		}
	}
	g.label(dst, fmt.Sprintf("특성 %d", len(traits)), tx, ty, 12, muted)
	ty += 22
	for _, t := range traits {
		if ty+22 > float64(r.Max.Y) {
			break
		}
		g.traitIconOf(dst, t, tx, ty, 20)
		g.label(dst, g.traitName(t), tx+26, ty, 13, ink)
		ty += 23
	}
}
