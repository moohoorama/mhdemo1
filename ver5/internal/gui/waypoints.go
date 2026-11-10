package gui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/session"
)

// The waypoint window lists every chapter's start and every battle; picking one asks the difficulty, then
// starts a game there (core.NewAt) to check the story or a battle without playing up to it.
const (
	waypointW, waypointH = 760, 700
	waypointRowH         = 48
	waypointGap          = 6
)

// waypointList is the window's scroll and press, the way the equipment picker keeps them.
type waypointList struct {
	top      int             // scroll, UI px
	view     image.Rectangle // the list as last drawn, in UI px
	rows     []waypointRow
	press    image.Point
	pressTop int
	pressed  bool
	moved    bool
}

type waypointRow struct {
	r    image.Rectangle
	node string
}

func (g *Game) waypointWindow(dst *ebiten.Image, x, y, w, h int) {
	g.label(dst, "고른 곳에서 새로 시작합니다. 레벨·승급은 다음 전투 기준, 아이템은 모두 가진 채입니다.", float64(x+32), float64(y+56), 14, muted)
	view := image.Rect(x+24, y+86, x+w-24, y+h-24)
	ways := g.S.Data.Waypoints()
	content := len(ways)*(waypointRowH+waypointGap) - waypointGap
	l := &g.waypoints
	l.top = max(0, min(l.top, content-view.Dy()))
	l.view, l.rows = unframed(view), l.rows[:0]
	clip := dst.SubImage(image.Rect(int(float64(view.Min.X)*drawPx+drawOX), int(float64(view.Min.Y)*drawPx+drawOY),
		int(float64(view.Max.X)*drawPx+drawOX), int(float64(view.Max.Y)*drawPx+drawOY))).(*ebiten.Image)
	for i, wp := range ways {
		top := view.Min.Y + i*(waypointRowH+waypointGap) - l.top
		r := image.Rect(view.Min.X, top, view.Max.X-14, top+waypointRowH)
		if !r.Overlaps(view) {
			continue
		}
		l.rows = append(l.rows, waypointRow{unframed(r.Intersect(view)), wp.Node})
		body, rim := lacquer2, goldDim
		if hovered(r.Intersect(view)) {
			body, rim = hoverC, gold
		}
		rect(clip, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), body)
		strokeRect(clip, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx())-1, float32(r.Dy())-1, 1, rim)
		level := g.S.Data.Rules.StageLevel(g.S.Data.NextBattleOrder(wp.Node))
		ty := float64(top + 13)
		if wp.Stage == "" {
			g.label(clip, fmt.Sprintf("제 %d 장", wp.Chapter), float64(r.Min.X+16), ty, 17, gold)
			g.label(clip, wp.Place, float64(r.Min.X+110), ty+1, 16, ink)
		} else {
			g.label(clip, "전투", float64(r.Min.X+44), ty+1, 15, muted)
			g.label(clip, g.S.Data.Stages[g.S.Data.StageIndex(wp.Stage)].Name, float64(r.Min.X+110), ty+1, 16, ink)
		}
		lv := fmt.Sprintf("Lv %d", level)
		g.label(clip, lv, float64(r.Max.X-20)-g.width(lv, 15), ty+1, 15, muted)
	}
	if content > view.Dy() {
		track := image.Rect(view.Max.X-6, view.Min.Y, view.Max.X, view.Max.Y)
		rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
		th := max(24, view.Dy()*view.Dy()/content)
		ty := view.Min.Y + (view.Dy()-th)*l.top/(content-view.Dy())
		rect(dst, float32(track.Min.X), float32(ty), float32(track.Dx()), float32(th), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
	}
}

// updateWaypoints scrolls the list by the wheel or a drag and starts at the row a press is let go on
// without dragging.
func (g *Game) updateWaypoints() bool {
	l := &g.waypoints
	if g.overlay() != "waypoints" {
		l.pressed = false
		return false
	}
	x, y := cursorPos()
	p := image.Pt(x, y)
	if _, wheel := ebiten.Wheel(); wheel != 0 && p.In(l.view) {
		l.top = max(0, l.top-int(wheel*40))
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) && p.In(l.view) {
		l.pressed, l.moved, l.press, l.pressTop = true, false, p, l.top
	}
	if !l.pressed {
		return false
	}
	if mousePressed(ebiten.MouseButtonLeft) {
		if abs(p.Y-l.press.Y) > dragStartPx {
			l.moved = true
		}
		if l.moved {
			l.top = max(0, l.pressTop-(p.Y-l.press.Y))
		}
		return true
	}
	l.pressed = false
	if !l.moved {
		for _, r := range l.rows {
			if l.press.In(r.r) {
				if len(g.S.Data.Rules.Difficulties) == 0 {
					g.closeOverlays()
					g.request(session.Request{Op: "jump", Node: r.node, Seed: g.seed})
					break
				}
				g.jumpNode = r.node
				g.openOverlay("difficulty")
				break
			}
		}
	}
	return true
}
