package gui

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// The corner panels stand in the two upper corners of every playing screen, each a head and a row of
// buttons that keep their places from screen to screen: the left one the place and the progress of the
// story or the battle, the right one the party and the menu (in a battle the HUD panel, which adds its tabs).
const (
	cornerHeadH = 34
	cornerBtnH  = 34
	cornerPad   = 8
	cornerGap   = 6
	cornerSlotW = 92 // the narrowest slot that still holds 장면 넘기기
	leftPanelW  = 540
	// cornerRowH is a panel's button row under its head: the rule, the padding and the buttons.
	cornerRowH = 1 + cornerPad + cornerBtnH + cornerPad
)

// cornerSlot is a button of a corner panel; a nil click draws it disabled, a toggle is a switch.
type cornerSlot struct {
	label  string
	click  func()
	toggle bool
	on     bool
}

func (g *Game) cornerButton(dst *ebiten.Image, r image.Rectangle, s *cornerSlot) {
	switch {
	case s.toggle:
		g.toggleBtn(dst, r, s.label, s.on, s.click)
	default:
		g.centeredBtn(dst, r, s.label, s.click)
	}
}

// foldButton is a panel's fold switch, its mark pointing up while the panel is open.
func (g *Game) foldButton(dst *ebiten.Image, x, y int, folded *bool) {
	r := image.Rect(x, y+6, x+24, y+cornerHeadH-6)
	g.btn(dst, r, "", func() { *folded = !*folded })
	g.sortMark(dst, float32(r.Min.X+r.Dx()/2), float32(r.Min.Y+r.Dy()/2), !*folded, gold)
}

// leftPanel is the panel in the upper left: on its head the fold switch, the place, the chapter and the money;
// below it the buttons the screen has, in this order — speed, zoom, auto text, skip scene, battle objective — a nil
// slot left out. The panel is as wide as its buttons and head need; a window too narrow wraps the buttons.
func (g *Game) leftPanel(dst *ebiten.Image, place, sub, money string, all [5]*cornerSlot) {
	slots := []*cornerSlot{}
	for _, s := range all {
		if s != nil {
			slots = append(slots, s)
		}
	}
	const slotW = (leftPanelW - 2*cornerPad - 4*cornerGap) / 5
	need := 2*cornerPad + len(slots)*slotW + max(0, len(slots)-1)*cornerGap
	head := 42 + 12.0
	if place != "" {
		head += g.width(place, 15) + 8
	}
	head += g.width(sub, 13)
	if money != "" {
		head += 16 + g.width(money, 15)
	}
	w := min(max(need, int(head)), uiW-12-12-8-infoW)
	cols := max(1, min(len(slots), (w-2*cornerPad+cornerGap)/(cornerSlotW+cornerGap)))
	rows := max(1, (len(slots)+cols-1)/cols)
	h := cornerHeadH
	if !g.leftFolded && len(slots) > 0 {
		h += cornerRowH + (rows-1)*(cornerBtnH+cornerGap)
	}
	r := image.Rect(12, hudTop(), 12+w, hudTop()+h)
	g.window(dst, r)
	g.block(r)
	g.foldButton(dst, r.Min.X+8, r.Min.Y, &g.leftFolded)
	x := float64(r.Min.X + 42)
	if place != "" {
		g.label(dst, place, x, float64(r.Min.Y+8), 15, gold)
		x += g.width(place, 15) + 8
	}
	g.label(dst, sub, x, float64(r.Min.Y+10), 13, muted)
	if money != "" {
		g.moneyLabel(dst, float64(r.Max.X-12), float64(r.Min.Y+8), 15)
	}
	if g.leftFolded || len(slots) == 0 {
		return
	}
	rect(dst, float32(r.Min.X+8), float32(r.Min.Y+cornerHeadH), float32(w-16), 1, goldDim)
	sw := (w - 2*cornerPad - (cols-1)*cornerGap) / cols
	for i, s := range slots {
		sx := r.Min.X + cornerPad + i%cols*(sw+cornerGap)
		sy := r.Min.Y + cornerHeadH + 1 + cornerPad + i/cols*(cornerBtnH+cornerGap)
		g.cornerButton(dst, image.Rect(sx, sy, sx+sw, sy+cornerBtnH), s)
	}
}

// cornerButtons is the right panel's row under its head at y: the party and the menu.
func (g *Game) cornerButtons(dst *ebiten.Image, x, y int, party func()) {
	rect(dst, float32(x+8), float32(y), float32(infoW-16), 1, goldDim)
	sw := (infoW - 2*cornerPad - cornerGap) / 2
	by := y + 1 + cornerPad
	g.centeredBtn(dst, image.Rect(x+cornerPad, by, x+cornerPad+sw, by+cornerBtnH), "관리", party)
	g.centeredBtn(dst, image.Rect(x+cornerPad+sw+cornerGap, by, x+cornerPad+2*sw+cornerGap, by+cornerBtnH), "메뉴", func() { g.openOverlay("menu") })
}

// rightPanel is the panel in the upper right of the screens without the battle's HUD panel: a summary of the
// party on its head and the fold switch at its end, the party and the menu below. It gives its bottom.
func (g *Game) rightPanel(dst *ebiten.Image, summary string, party func()) int {
	h := cornerHeadH
	if !g.hudFolded {
		h += cornerRowH
	}
	r := image.Rect(uiW-12-infoW, hudTop(), uiW-12, hudTop()+h)
	g.window(dst, r)
	g.block(r)
	g.label(dst, summary, float64(r.Min.X+12), float64(r.Min.Y+10), 13, muted)
	g.foldButton(dst, r.Max.X-32, r.Min.Y, &g.hudFolded)
	if !g.hudFolded {
		g.cornerButtons(dst, r.Min.X, r.Min.Y+cornerHeadH, party)
	}
	return r.Max.Y
}

func (g *Game) speedSlot() *cornerSlot {
	return &cornerSlot{label: fmt.Sprintf("배속 ×%d", g.speed), click: func() { g.speed = nextSpeed[g.speed] }}
}

// zoomSlot steps the camera through the zooms as the wheel does; nil click while the camera waits.
func (g *Game) zoomSlot(locked bool) *cornerSlot {
	cur := g.zoom
	if !g.exploring() && g.zoomT < .5 {
		cur = g.zoomTo
	}
	s := &cornerSlot{label: fmt.Sprintf("확대 %d배", int(math.Round(cur)))}
	if !locked {
		s.click = func() {
			next := zooms[0]
			for _, z := range zooms {
				if z > cur+.01 {
					next = z
					break
				}
			}
			if g.exploring() {
				g.zoom = next
			} else {
				g.setZoom(next)
			}
		}
	}
	return s
}

func (g *Game) autoSlot() *cornerSlot {
	s := g.S.Settings
	return &cornerSlot{label: "자동 넘김", toggle: true, on: s.AutoText, click: func() {
		s.AutoText = !s.AutoText
		g.request(session.Request{Op: "settings", Settings: &s})
	}}
}

// skipGuard is how long clicks wait after a skip stops, so a second press does not run on into the next scene.
const skipGuard = 1.0

// skipSlot skips the rest of the scene on screen.
func (g *Game) skipSlot() *cornerSlot {
	s := &cornerSlot{label: "장면 넘기기"}
	if g.scene != nil && g.skipping == nil && g.uiSeconds >= g.skipUntil {
		s.click = func() { g.skipping = g.scene.def }
	}
	return s
}

func (g *Game) goalSlot() *cornerSlot {
	return &cornerSlot{label: "전투 목표", click: g.openObjective}
}

// chapterName is "N장" for the chapter holding the node, or nothing.
func (g *Game) chapterName(node string) string {
	if i := g.S.Data.ChapterIndex(node); i >= 0 {
		return fmt.Sprintf("%d장", i+1)
	}
	return ""
}

// partySummary is the right panel's head outside a battle: how many officers and kinds of things the party has.
func (g *Game) partySummary(o core.Observation) string {
	return fmt.Sprintf("장수 %d · 소지품 %d", len(g.roster(o)), len(g.belongings(o)))
}

// storyPanels are the corner panels of the story: a scene's or the walk's place, its chapter, and on the walk the
// money; the zoom only on the walk, where no scene sets the camera, and the skip only in a scene.
func (g *Game) storyPanels(dst *ebiten.Image, o core.Observation) {
	place, money := "", ""
	var zoom, skip *cornerSlot
	if o.Dialogue.Kind == "explore" && g.explore != nil {
		place = g.S.Engine.ExploreFile().Areas[g.explore.area].Name
		money = fmt.Sprintf("돈 %d", o.Money)
		zoom = g.zoomSlot(false)
	} else if g.scene != nil {
		if art := g.assets.Maps[g.scene.def.Map]; art != nil {
			place, _, _ = strings.Cut(art.Name, "(")
			place = strings.TrimSpace(place)
		}
		skip = g.skipSlot()
	}
	g.leftPanel(dst, place, g.chapterName(o.Node), money, [5]*cornerSlot{g.speedSlot(), zoom, g.autoSlot(), skip, nil})
	var party func()
	if o.Dialogue.Kind == "explore" && g.explore != nil && g.explore.talk == nil {
		party = func() { g.openOverlay("party") }
	}
	g.rightPanel(dst, g.partySummary(o), party)
}

// battleLeftPanel is the left panel of a battle and of its deployment, where the head counts who is placed. A
// battle's lines pass on their own, so it has no auto text.
func (g *Game) battleLeftPanel(dst *ebiten.Image, o core.Observation, sub string) {
	g.leftPanel(dst, g.S.Data.Stages[o.Stage].Name, sub, "", [5]*cornerSlot{g.speedSlot(), g.zoomSlot(g.cameraLocked()), nil, nil, g.goalSlot()})
}
