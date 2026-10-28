package gui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// The battle HUD's one panel in the upper right corner (docs_old/plans/PLAN_hud-panel.md): a head
// line with the side to move and the end-of-phase button, a row of tabs, and the tab's content,
// which sets the panel's height. Selecting an officer shows the unit tab; letting go returns to
// the tab chosen by hand, and an AI army's turn shows the log.
const (
	hudHeaderH = 36
	hudTabsH   = 32
)

// hudTop is the panel's top edge.
func hudTop() int { return 8 }

// menuButtonRect is the menu button in the top right corner of the screens without the panel.
func menuButtonRect() image.Rectangle { return image.Rect(uiW-104, 6, uiW-8, 35) }

type hudTab struct{ id, title string }

var hudTabs = []hudTab{{"minimap", "미니맵"}, {"unit", "장수"}, {"units", "일람"}, {"log", "기록"}, {"misc", "기타"}}

// hudFollowTurn shows the log for the AI armies' turns (enemy and allied) and the previous tab
// again afterwards.
func (g *Game) hudFollowTurn(o core.Observation) {
	if o.Turn == g.hudTurn {
		return
	}
	was := g.hudTurn != "" && g.hudTurn != "ally"
	g.hudTurn = o.Turn
	switch now := o.Turn != "ally"; {
	case now && !was:
		g.hudBefore, g.hudBase, g.hudPeek = g.hudBase, "log", ""
	case !now && was:
		g.hudBase = g.hudBefore
	}
}

// hudCurrent is the tab to show.
func (g *Game) hudCurrent(o core.Observation) string {
	if g.detailUnit(o) == nil {
		g.hudPeek = ""
		if g.hudBase == "" {
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
func (g *Game) hudBodyHeight(tab string, o core.Observation, room int) int {
	switch tab {
	case "units":
		return g.unitListHeight(o, room)
	case "log":
		return g.logBodyHeight()
	case "misc":
		return 5*40 + 20
	case "unit":
		return detailH
	}
	_, h := g.miniSize()
	return int(h) + 40
}

// hudFadeAlpha is how opaque the panel stays while the pointer is away from it.
const hudFadeAlpha = .55

// battleHUDPanel draws the panel and registers its buttons. Its height eases to the tab's,
// it folds to the head and the tabs, and it fades while the pointer is elsewhere.
func (g *Game) battleHUDPanel(dst *ebiten.Image, o core.Observation) {
	g.hudFollowTurn(o)
	tab := g.hudCurrent(o)
	head := hudHeaderH
	x, y := uiW-12-infoW, hudTop()
	room := uiH - 8 - y - head - hudTabsH
	bodyH := 0
	if !g.hudFolded {
		bodyH = g.hudBodyHeight(tab, o, room)
	}
	h := min(head+hudTabsH+bodyH, uiH-8-y)
	r := image.Rect(x, y, x+infoW, y+h)
	g.hudRect, g.hudShown, g.hudBody = r, tab, image.Rect(x, y+head+hudTabsH, x+infoW, y+h)
	if g.hudFolded {
		g.hudShown, g.hudBody = "", image.Rectangle{}
	}

	now := g.uiSeconds
	k := 1.0
	if g.hudAnimated {
		k = min(1, (now-g.hudAnimAt)*16)
	}
	g.hudH += (float64(h) - g.hudH) * k
	if !g.hudAnimated || abs(int(g.hudH*2)-h*2) < 1 {
		g.hudH = float64(h)
	}
	target := 1.0
	if cx, cy := cursorPos(); !image.Pt(cx, cy).In(r) && !g.logDragging && !g.logThumbDragging {
		target = hudFadeAlpha
	}
	if !g.hudAnimated {
		g.hudAlpha = 1
	}
	g.hudAlpha += (target - g.hudAlpha) * k
	g.hudAnimated, g.hudAnimAt = true, now

	shown := image.Rect(x, y, x+infoW, y+int(g.hudH+.5))
	clip := image.Rect(int(float64(x-2)*drawPx+drawOX), int(float64(y-2)*drawPx+drawOY), int(float64(x+infoW+8)*drawPx+drawOX), int(float64(shown.Max.Y+8)*drawPx+drawOY))
	canvas := dst
	faded := g.hudAlpha < .995
	if faded {
		if g.hudLayer == nil || g.hudLayer.Bounds().Size() != dst.Bounds().Size() {
			g.hudLayer = ebiten.NewImage(dst.Bounds().Dx(), dst.Bounds().Dy())
		}
		g.hudLayer.Clear()
		canvas = g.hudLayer
	}
	cd := canvas.SubImage(clip.Intersect(canvas.Bounds())).(*ebiten.Image)
	g.window(cd, shown)
	g.block(r)
	g.hudHead(cd, o, r)
	g.hudTabRow(cd, o, tab, x, y+head)
	switch g.hudShown {
	case "log":
		g.drawBattleLog(cd)
	case "units":
		g.hudUnits(cd, o, g.hudBody)
	case "misc":
		g.hudMisc(cd, g.hudBody)
	case "unit":
		if g.detailUnit(o) == nil {
			g.label(cd, "선택한 장수가 없습니다.", float64(g.hudBody.Min.X+14), float64(g.hudBody.Min.Y+16), 14, muted)
		}
		g.detail(cd, o)
	case "minimap":
		w, mh := g.miniSize()
		g.drawMinimap(cd, o, float64(g.hudBody.Min.X)+(infoW-w)/2, float64(g.hudBody.Min.Y)+(float64(g.hudBody.Dy())-mh)/2)
	}
	if faded {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(cd.Bounds().Min.X), float64(cd.Bounds().Min.Y))
		op.ColorScale.ScaleAlpha(float32(g.hudAlpha))
		dst.DrawImage(cd, op)
	}
}

// hudHead is the side to move with the turn, the end-of-phase button, and on the enemy's turn
// that waits for the player the buttons that advance it.
func (g *Game) hudHead(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	side := g.armyName(o.Turn, o.Army) + " 차례"
	c := g.armyColor(o.Turn, o.Army)
	turn := fmt.Sprintf("%d턴", o.Round)
	if st := g.S.Data.Stages[o.Stage]; st.TurnLimit > 0 {
		turn = fmt.Sprintf("%d/%d턴", o.Round, st.TurnLimit)
	}
	fold := image.Rect(r.Min.X+8, r.Min.Y+6, r.Min.X+32, r.Min.Y+hudHeaderH-6)
	g.btn(dst, fold, "", func() { g.hudFolded = !g.hudFolded })
	g.sortMark(dst, float32(fold.Min.X+fold.Dx()/2), float32(fold.Min.Y+fold.Dy()/2), !g.hudFolded, gold)
	g.label(dst, side, float64(r.Min.X+42), float64(r.Min.Y+8), 15, c)
	g.label(dst, turn, float64(r.Min.X+42)+g.width(side, 15)+8, float64(r.Min.Y+10), 13, muted)

	idle := !g.busy() && o.Phase == "battle"
	var end func()
	if idle && o.Turn == "ally" {
		end = g.endPhase
	}
	g.btn(dst, image.Rect(r.Max.X-108, r.Min.Y+5, r.Max.X-10, r.Min.Y+hudHeaderH-5), "진영 종료", end)
}

// hudTabRow draws the tabs: the unit tab is always there and shows itself with a selection, and a click on another
// tab while one is selected only peeks at it.
func (g *Game) hudTabRow(dst *ebiten.Image, o core.Observation, shown string, x, y int) {
	selected := g.detailUnit(o) != nil
	w := (infoW - 20) / len(hudTabs)
	for i, t := range hudTabs {
		id := t.id
		bx := x + 10 + i*w
		var click func()
		switch {
		case id == shown && !g.hudFolded:
		case id == "unit" && selected:
			click = func() { g.hudPeek, g.hudFolded = "", false }
		case selected:
			click = func() { g.hudPeek, g.hudFolded = id, false }
		default:
			click = func() { g.hudBase, g.hudFolded = id, false }
		}
		g.btn(dst, image.Rect(bx, y+2, bx+w-2, y+hudTabsH-2), t.title, click)
		if id == shown && !g.hudFolded {
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
		{"자동 전투" + mark(g.autoplay), func() { g.autoplay = !g.autoplay }},
		{"작전 개요", func() { g.modals = append(g.modals, &modal{kind: "objective", manual: true}) }},
		{"병력일람", func() { g.openOverlay("roster") }},
		{"메뉴", func() { g.openOverlay("menu") }},
	}
	for i, it := range items {
		y := r.Min.Y + 10 + i*40
		g.btn(dst, image.Rect(r.Min.X+10, y, r.Max.X-10, y+34), it.label, it.click)
	}
}
