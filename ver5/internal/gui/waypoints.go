package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/session"
)

// The waypoint window lists every chapter's start and every battle on its left; picking one asks the
// difficulty, then starts a game there (core.NewAtWith) to check the story or a battle without playing up to
// it. Its right side sets what that game is told first: the party's level and the flags the battles before
// left (content.StoryFlags).
const (
	waypointW, waypointH = 1160, 700
	waypointListW        = 712 // the list's share of the window; the rest is the setup
	waypointFlagH        = 40
	waypointRowH         = 48
	waypointGap          = 6
)

// waypointList is the window's scroll and press, the way the equipment picker keeps them.
type waypointList struct {
	top      int             // scroll, UI px
	end      int             // the last top the list stops at, as last drawn
	view     image.Rectangle // the list as last drawn, in UI px
	rows     []waypointRow
	press    image.Point
	pressTop int
	pressed  bool
	moved    bool
	last     int             // the pointer's y at the last frame of the press
	speed    float64         // the pointer's recent speed, px a frame
	fling    float64         // the glide left after a release, px a frame
	bar      image.Rectangle // the scrollbar's track as last drawn, empty when the list fits
	barDrag  bool            // a press on the track is dragging the thumb
	barPress int
	barTop   int
	barPerPx float64 // list px per px of the thumb's travel
}

type waypointRow struct {
	r    image.Rectangle
	node string
}

func (g *Game) waypointWindow(dst *ebiten.Image, x, y, w, h int) {
	g.label(dst, "오른쪽에서 레벨과 앞선 전투 결과를 정한 뒤, 왼쪽에서 시작할 곳을 고릅니다. 아이템은 모두 가진 채입니다.", float64(x+32), float64(y+56), 14, muted)
	view := image.Rect(x+24, y+86, x+24+waypointListW, y+h-24)
	g.waypointSetup(dst, image.Rect(view.Max.X+24, view.Min.Y, x+w-24, view.Max.Y))
	ways := g.S.Data.Waypoints()
	content := len(ways)*(waypointRowH+waypointGap) - waypointGap
	l := &g.waypoints
	l.top = max(0, min(l.top, content-view.Dy()))
	l.end = max(0, content-view.Dy())
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
		level := g.jumpLevel
		if level == 0 {
			level = g.S.Data.Rules.StageLevel(g.S.Data.NextBattleOrder(wp.Node))
		}
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
	l.bar = image.Rectangle{}
	if content > view.Dy() {
		track := image.Rect(view.Max.X-6, view.Min.Y, view.Max.X, view.Max.Y)
		l.bar = unframed(track)
		rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
		th := max(24, view.Dy()*view.Dy()/content)
		l.barPerPx = float64(content-view.Dy()) / float64(max(1, view.Dy()-th))
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
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		l.fling = 0
	}
	if l.barDrag {
		if mousePressed(ebiten.MouseButtonLeft) {
			l.top = max(0, min(l.end, l.barTop+int(float64(p.Y-l.barPress)*l.barPerPx+.5)))
		} else {
			l.barDrag = false
		}
		return true
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) && p.In(l.bar) {
		l.barDrag, l.barPress, l.barTop = true, p.Y, l.top
		return true
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) && p.In(l.view) {
		l.pressed, l.moved, l.press, l.pressTop, l.last, l.speed = true, false, p, l.top, p.Y, 0
	}
	if !l.pressed {
		g.glideWaypoints()
		return false
	}
	if mousePressed(ebiten.MouseButtonLeft) {
		if abs(p.Y-l.press.Y) > dragStartPx {
			l.moved = true
		}
		if l.moved {
			l.top = max(0, l.pressTop-(p.Y-l.press.Y))
			l.speed = .6*l.speed + .4*float64(p.Y-l.last)
		}
		l.last = p.Y
		return true
	}
	l.pressed = false
	if l.moved && math.Abs(l.speed) >= flingMinPx {
		l.fling = -l.speed
	}
	if !l.moved {
		for _, r := range l.rows {
			if l.press.In(r.r) {
				if len(g.S.Data.Rules.Difficulties) == 0 {
					g.closeOverlays()
					g.jump(r.node, "")
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

// waypointLevels are the levels the setup offers at a press; the steps beside them reach the ones between.
var waypointLevels = []int{10, 20, 30, 40, 50, 60}

// waypointSetup draws the setup in r: the party's level (none is each waypoint's own) over the battle flags,
// each a switch that names its battle and objective over what it changes later. A flag of a battle after the picked waypoint
// is not used.
func (g *Game) waypointSetup(dst *ebiten.Image, r image.Rectangle) {
	x, y, w := r.Min.X, r.Min.Y, r.Dx()
	levels := []choice{{"기본", g.jumpLevel == 0, func() { g.jumpLevel = 0 }}}
	for _, n := range waypointLevels {
		levels = append(levels, choice{fmt.Sprint(n), g.jumpLevel == n, func() { g.jumpLevel = n }})
	}
	g.choiceRow(dst, x, y, w, "아군 레벨", levels)
	step := func(d int) func() {
		if g.jumpLevel == 0 {
			return nil
		}
		return func() { g.jumpLevel = max(1, min(g.jumpLevel+d, g.S.Data.Rules.MaxLevel)) }
	}
	now := "다음 전투 기준"
	if g.jumpLevel > 0 {
		now = fmt.Sprintf("Lv %d", g.jumpLevel)
	}
	g.centeredBtn(dst, image.Rect(x, y+64, x+56, y+96), "-1", step(-1))
	g.centeredBtn(dst, image.Rect(x+w-56, y+64, x+w, y+96), "+1", step(1))
	g.centered(dst, now, float64(x+w/2), float64(y+69), 16, ink)

	flags := g.S.Data.StoryFlags()
	top := y + 120
	g.label(dst, "앞선 전투 결과", float64(x), float64(top), 14, muted)
	g.centeredBtn(dst, image.Rect(x+w-136, top-6, x+w-72, top+22), "모두", func() {
		g.jumpFlags = map[string]bool{}
		for _, f := range flags {
			g.jumpFlags[f.Flag] = true
		}
	})
	g.centeredBtn(dst, image.Rect(x+w-64, top-6, x+w, top+22), "없음", func() { g.jumpFlags = nil })
	for i, f := range flags {
		b := image.Rect(x, top+30+i*(waypointFlagH+4), x+w, top+30+i*(waypointFlagH+4)+waypointFlagH)
		on := g.jumpFlags[f.Flag]
		toggle := func() {
			if g.jumpFlags == nil {
				g.jumpFlags = map[string]bool{}
			}
			g.jumpFlags[f.Flag] = !on
		}
		g.choiceBtn(dst, b, "", on, toggle)
		fg := ink
		if on {
			g.addButton(b, "", toggle) // a lit choice is not a button of its own
			fg = gold
			rect(dst, float32(b.Min.X+13), float32(b.Min.Y+13), 14, 14, gold)
		}
		strokeRect(dst, float32(b.Min.X)+12.5, float32(b.Min.Y)+12.5, 15, 15, 1, goldDim)
		later := f.Later
		if later == "" {
			later = "뒤에 달라지는 것 없음"
		}
		g.label(dst, g.S.Data.Stages[g.S.Data.StageIndex(f.Stage)].Name+" · "+f.Text, float64(b.Min.X+40), float64(b.Min.Y+3), 12, muted)
		g.label(dst, later, float64(b.Min.X+40), float64(b.Min.Y+17), 14, fg)
	}
}

// jump starts a game at the waypoint as the setup says.
func (g *Game) jump(node, difficulty string) {
	var flags []string
	for _, f := range g.S.Data.StoryFlags() {
		if g.jumpFlags[f.Flag] {
			flags = append(flags, f.Flag)
		}
	}
	g.request(session.Request{Op: "jump", Node: node, Seed: g.seed, Difficulty: difficulty, Flags: flags, Level: g.jumpLevel})
}

// glideWaypoints carries on a glide left by a release, px a frame, stopping at the first or last row.
func (g *Game) glideWaypoints() {
	l := &g.waypoints
	if l.fling == 0 {
		return
	}
	l.top = max(0, min(l.end, l.top+int(l.fling)))
	l.fling *= flingFriction
	if math.Abs(l.fling) < flingRest || l.top == 0 && l.fling < 0 || l.top == l.end && l.fling > 0 {
		l.fling = 0
	}
}
