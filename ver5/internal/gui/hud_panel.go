package gui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// The battle HUD's one panel in the upper right corner (docs/plans/PLAN_hud-panel.md): a head
// line with the side to move and the end-of-phase button, a row of tabs, and the tab's content,
// which sets the panel's height. Selecting an officer shows the unit tab; letting go returns to
// the tab chosen by hand, and the enemy's turn shows the log.
const (
	hudHeaderH = 36
	hudTabsH   = 32
	hudAIRowH  = 34
)

// hudTop is the panel's top edge: under the name tags that hang from the top bar.
func hudTop() int { return tagH + 6 }

type hudTab struct{ id, title string }

var hudTabs = []hudTab{{"minimap", "미니맵"}, {"log", "기록"}, {"misc", "기타"}, {"unit", "장수"}}

// hudManualAI reports whether the enemy's turn waits for the player to advance it.
func (g *Game) hudManualAI(o core.Observation) bool {
	return o.Turn == "enemy" && g.S.Settings.AI != "auto"
}

// hudFollowTurn shows the log for the enemy's turn and the previous tab again afterwards.
func (g *Game) hudFollowTurn(o core.Observation) {
	if o.Turn == g.hudTurn {
		return
	}
	was := g.hudTurn == "enemy"
	g.hudTurn = o.Turn
	switch {
	case o.Turn == "enemy":
		g.hudBefore, g.hudBase, g.hudPeek = g.hudBase, "log", ""
	case was:
		g.hudBase = g.hudBefore
	}
}

// hudCurrent is the tab to show.
func (g *Game) hudCurrent(o core.Observation) string {
	if g.detailUnit(o) == nil {
		g.hudPeek = ""
		if g.hudBase == "" || g.hudBase == "unit" {
			return "minimap"
		}
		return g.hudBase
	}
	if g.hudPeek != "" {
		return g.hudPeek
	}
	return "unit"
}

// hudBodyHeight is the content height of a tab.
func (g *Game) hudBodyHeight(tab string) int {
	switch tab {
	case "log":
		return g.logBodyHeight()
	case "misc":
		return 4*40 + 20
	case "unit":
		return detailH
	}
	_, h := g.miniSize()
	return int(h) + 40
}

// battleHUDPanel draws the panel and registers its buttons.
func (g *Game) battleHUDPanel(dst *ebiten.Image, o core.Observation) {
	g.hudFollowTurn(o)
	tab := g.hudCurrent(o)
	head := hudHeaderH
	if g.hudManualAI(o) {
		head += hudAIRowH
	}
	x, y := uiW-12-infoW, hudTop()
	h := min(head+hudTabsH+g.hudBodyHeight(tab), uiH-8-y)
	r := image.Rect(x, y, x+infoW, y+h)
	g.hudRect, g.hudShown = r, tab
	g.hudBody = image.Rect(x, y+head+hudTabsH, x+infoW, y+h)
	window(dst, r)
	g.block(r)
	g.hudHead(dst, o, r)
	g.hudTabRow(dst, o, tab, x, y+head)
	switch tab {
	case "log":
		g.drawBattleLog(dst)
	case "misc":
		g.hudMisc(dst, g.hudBody)
	case "unit":
		g.detail(dst, o)
	default:
		w, mh := g.miniSize()
		g.drawMinimap(dst, o, float64(g.hudBody.Min.X)+(infoW-w)/2, float64(g.hudBody.Min.Y)+(float64(g.hudBody.Dy())-mh)/2)
	}
}

// hudHead is the side to move with the turn, the end-of-phase button, and on the enemy's turn
// that waits for the player the buttons that advance it.
func (g *Game) hudHead(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	side := g.lordName() + "군 차례"
	switch o.Turn {
	case "enemy":
		side = "적군 차례"
	case "support":
		side = "동맹군 차례"
	}
	c := turnColor(o.Turn)
	turn := fmt.Sprintf("%d턴", o.Round)
	if st := g.stageInfo(o); st.turnLimit > 0 {
		turn = fmt.Sprintf("%d/%d턴", o.Round, st.turnLimit)
	}
	glow := c
	glow.A = 70
	fillCircle(dst, float32(r.Min.X+18), float32(r.Min.Y+hudHeaderH/2), 7, glow)
	fillCircle(dst, float32(r.Min.X+18), float32(r.Min.Y+hudHeaderH/2), 4, c)
	g.label(dst, side, float64(r.Min.X+32), float64(r.Min.Y+8), 15, c)
	g.label(dst, turn, float64(r.Min.X+32)+g.width(side, 15)+10, float64(r.Min.Y+10), 13, muted)

	idle := !g.busy() && o.Phase == "battle"
	var end func()
	if idle && o.Turn == "ally" {
		end = g.endPhase
	}
	g.btn(dst, image.Rect(r.Max.X-108, r.Min.Y+5, r.Max.X-10, r.Min.Y+hudHeaderH-5), "진영 종료", end)
	if g.hudManualAI(o) {
		var one, all func()
		if idle {
			one = func() { g.request(session.Request{Op: "ai"}) }
			all = g.resolvePhase
		}
		y := r.Min.Y + hudHeaderH
		half := (r.Dx() - 30) / 2
		g.btn(dst, image.Rect(r.Min.X+10, y, r.Min.X+10+half, y+hudAIRowH-6), "적 한 명령", one)
		g.btn(dst, image.Rect(r.Min.X+20+half, y, r.Min.X+20+2*half, y+hudAIRowH-6), "적 진영 진행", all)
	}
}

// hudTabRow draws the tabs: the unit tab lights up with a selection, and a click on another
// tab while one is selected only peeks at it.
func (g *Game) hudTabRow(dst *ebiten.Image, o core.Observation, shown string, x, y int) {
	selected := g.detailUnit(o) != nil
	w := (infoW - 20) / len(hudTabs)
	for i, t := range hudTabs {
		id := t.id
		bx := x + 10 + i*w
		var click func()
		switch {
		case id == shown:
		case id == "unit":
			if selected {
				click = func() { g.hudPeek = "" }
			}
		case selected:
			click = func() { g.hudPeek = id }
		default:
			click = func() { g.hudBase = id }
		}
		g.btn(dst, image.Rect(bx, y+2, bx+w-2, y+hudTabsH-2), t.title, click)
		if id == shown {
			rect(dst, float32(bx), float32(y+hudTabsH-4), float32(w-2), 2, gold)
		}
	}
}

// hudMisc is the tab of switches and the windows that rarely open.
func (g *Game) hudMisc(dst *ebiten.Image, r image.Rectangle) {
	mark := func(on bool) string {
		if on {
			return " ●"
		}
		return ""
	}
	items := []struct {
		label string
		click func()
	}{
		{"위협 범위" + mark(g.threatAll), func() { g.threatAll = !g.threatAll }},
		{"자동 진행" + mark(g.autoplay), func() { g.autoplay = !g.autoplay }},
		{"작전 개요", func() { g.modals = append(g.modals, &modal{kind: "objective"}) }},
		{"메뉴", func() { g.openOverlay("menu") }},
	}
	for i, it := range items {
		y := r.Min.Y + 10 + i*40
		g.btn(dst, image.Rect(r.Min.X+10, y, r.Max.X-10, y+34), it.label, it.click)
	}
}
