package gui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// The battle HUD's one panel in the upper right corner (docs_old/plans/PLAN_hud-panel.md): a head
// line with the side to move and the end-of-phase button, a row of tabs, and the tab's content,
// which sets the panel's height. Selecting an officer shows the unit tab; letting go returns to
// the tab chosen by hand, and an AI army's turn shows the log.
const (
	hudHeaderH = cornerHeadH
	hudTabsH   = 32
)

// hudTop is the top edge of the panel and of the corner panels beside it.
func hudTop() int { return 10 }

type hudTab struct{ id, title string }

var hudTabs = []hudTab{{"minimap", "미니맵"}, {"unit", "장수"}, {"units", "일람"}, {"log", "기록"}}

// syncHudFolds keeps two sets of folds for the battle's three panels, one while the player commands (their turn, no
// event playing; the deployment too) and one while others act or an event plays. Defaults: all open to command, all
// folded to watch. Folding or opening a panel changes the set of the state the battle is in, for the next time.
func (g *Game) syncHudFolds(o core.Observation) {
	idle := false
	if o.Phase != "deployment" {
		idle = g.player.shownTurn(o).Turn != "ally" || g.player.eventPlaying()
	}
	cur := [3]bool{g.leftFolded, g.hudFolded, g.infoFolded}
	if !g.hudModeSet || idle != g.hudIdle {
		g.hudIdle, g.hudModeSet = idle, true
		g.hudApplied = g.S.Settings.HudFolds(idle)
		g.leftFolded, g.hudFolded, g.infoFolded = g.hudApplied[0], g.hudApplied[1], g.hudApplied[2]
		return
	}
	if cur != g.hudApplied {
		v := g.S.Settings
		v.SetHudFolds(idle, cur)
		g.hudApplied = cur
		g.request(session.Request{Op: "settings", Settings: &v})
	}
}

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
	case "unit":
		return detailH
	}
	_, h := g.miniSize()
	return int(h) + 40
}

// hudFadeAlpha is how opaque the panel stays while the pointer is away from it.
const hudFadeAlpha = .55

// hudGripH is the strip along the top of the info panel that drags the log's height.
const hudGripH = 12

// battleHUDPanel draws the HUD in two: at the top right the side to move with the 관리 and 메뉴
// buttons, and at the bottom right the info panel with its tabs, whose height eases to the tab's. Folding leaves
// the head; the info panel fades while the pointer is elsewhere.
func (g *Game) battleHUDPanel(dst *ebiten.Image, o core.Observation) {
	g.hudFollowTurn(o)
	tab := g.hudCurrent(o)
	x, y := uiW-12-infoW, hudTop()
	topH := hudHeaderH + int(cornerRowH*g.foldK(&g.hudFold, g.hudFolded)+.5)
	top := image.Rect(x, y, x+infoW, y+topH)
	g.window(dst, top)
	g.block(top)
	g.hudHead(dst, o, top)
	if topH > hudHeaderH {
		g.cornerButtons(clipTo(dst, top), x, y+hudHeaderH, func() { g.openOverlay("roster") })
	}
	g.hudRect, g.hudShown, g.hudBody = top, "", image.Rectangle{}
	grip := 4
	if tab == "log" {
		grip = hudGripH
	}
	if g.infoFolded {
		grip = 0 // folded, the panel is its tab row alone, which stays where it is whatever tab is shown
	}
	room := uiH - 12 - (top.Max.Y + 6) - grip - hudTabsH
	h := grip + hudTabsH
	if !g.infoFolded {
		h += min(room, g.hudBodyHeight(tab, o, room))
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
	g.hudH = min(g.hudH, float64(grip+hudTabsH+room)) // easing down from a taller tab never climbs over the top panel
	info := image.Rect(x, uiH-12-h, x+infoW, uiH-12)
	shown := image.Rect(x, uiH-12-int(g.hudH+.5), x+infoW, uiH-12)
	g.hudRect = image.Rect(x, y, x+infoW, info.Max.Y)
	g.hudInfo, g.hudShown, g.hudBody = info, tab, image.Rect(x, shown.Min.Y+grip, x+infoW, info.Max.Y-hudTabsH) // the content follows the eased top
	if g.infoFolded {
		g.hudShown = ""
	}
	target := 1.0
	if cx, cy := cursorPos(); !pointer.used && !image.Pt(cx, cy).In(info) && !g.logDragging && !g.logThumbDragging { // a touch screen has no pointer to come back
		target = hudFadeAlpha
	}
	if !g.hudAnimated {
		g.hudAlpha = 1
	}
	g.hudAlpha += (target - g.hudAlpha) * k
	g.hudAnimated, g.hudAnimAt = true, now

	clip := image.Rect(int(float64(x-2)*drawPx+drawOX), int(float64(shown.Min.Y-2)*drawPx+drawOY), int(float64(x+infoW+8)*drawPx+drawOX), int(float64(shown.Max.Y+8)*drawPx+drawOY))
	canvas := dst
	faded := g.hudAlpha < .995
	if faded {
		if g.hudLayer == nil || g.hudLayer.Bounds().Size() != dst.Bounds().Size() {
			if g.hudLayer != nil {
				g.hudLayer.Deallocate()
			}
			g.hudLayer = newLayer(dst.Bounds().Dx(), dst.Bounds().Dy())
		}
		g.hudLayer.Clear()
		canvas = g.hudLayer
	}
	cd := canvas.SubImage(clip.Intersect(canvas.Bounds())).(*ebiten.Image)
	g.window(cd, shown)
	g.block(info)
	if tab == "log" && !g.infoFolded {
		grip := image.Rect(x+infoW/2-18, shown.Min.Y+5, x+infoW/2+18, shown.Min.Y+8)
		rect(cd, float32(grip.Min.X), float32(grip.Min.Y), float32(grip.Dx()), 3, goldDim)
	}
	g.hudTabRow(cd, o, tab, x, shown.Max.Y-hudTabsH)
	switch g.hudShown {
	case "log":
		g.drawBattleLog(cd)
	case "units":
		g.hudUnits(cd, o, g.hudBody)
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

// hudHead is the side to move with the turn and the fold switch.
func (g *Game) hudHead(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	side := g.armyName(o.Turn, o.Army) + " 차례"
	c := g.armyColor(o.Turn, o.Army)
	turn := fmt.Sprintf("%d턴", o.Round)
	if st := g.S.Data.Stages[o.Stage]; st.TurnLimit > 0 {
		turn = fmt.Sprintf("%d/%d턴", o.Round, st.TurnLimit)
	}
	g.label(dst, side, float64(r.Min.X+12), float64(r.Min.Y+8), 15, c)
	g.label(dst, turn, float64(r.Min.X+12)+g.width(side, 15)+8, float64(r.Min.Y+10), 13, muted)
	g.foldButton(dst, r.Max.X-32, r.Min.Y, &g.hudFolded)
}

// hudTabRow draws the tabs along the bottom of the info panel, so that they stay in place when it folds: the unit tab is
// always there and shows itself with a selection, and a click on another tab while one is selected only peeks at it.
// A click on any tab of a folded panel opens it; the fold switch ends the row.
func (g *Game) hudTabRow(dst *ebiten.Image, o core.Observation, shown string, x, y int) {
	selected := g.detailUnit(o) != nil
	w := (infoW - 20 - 28) / len(hudTabs)
	for i, t := range hudTabs {
		id := t.id
		bx := x + 10 + i*w
		var click func()
		switch {
		case id == shown && !g.infoFolded:
		case id == "unit" && selected:
			click = func() { g.hudPeek, g.infoFolded = "", false }
		case selected:
			click = func() { g.hudPeek, g.infoFolded = id, false }
		default:
			click = func() { g.hudBase, g.infoFolded = id, false }
		}
		g.btn(dst, image.Rect(bx, y+2, bx+w-2, y+hudTabsH-2), t.title, click)
		if id == shown && !g.infoFolded {
			rect(dst, float32(bx), float32(y+hudTabsH-4), float32(w-2), 2, gold)
		}
	}
	r := image.Rect(x+infoW-10-24, y+4, x+infoW-10, y+hudTabsH-4)
	g.btn(dst, r, "", func() { g.infoFolded = !g.infoFolded })
	g.sortMark(dst, float32(r.Min.X+r.Dx()/2), float32(r.Min.Y+r.Dy()/2), g.infoFolded, gold) // open: points down, away from the content it hides
}
