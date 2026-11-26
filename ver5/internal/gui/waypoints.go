package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
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
	flagCauseH           = 18 // a line of a flag's battle and objective (12 px text)
	flagEffectH          = 21 // a line of what it changes later (14 px text)
	waypointRowH         = 48
	waypointGap          = 6
)

// waypointList is the window's state: the act whose waypoints the list shows (each act keeps its scroll while
// another is shown) and the two scrolling areas, the waypoints on the left and the battle flags on the right.
type waypointList struct {
	act   string
	tops  map[string]int
	list  scroller
	flags scroller
}

// scroller is a scrolling area's scroll and press, the way the equipment picker keeps them: a press let go
// without dragging picks the row under it, a drag or the wheel scrolls, the thumb of the bar can be dragged.
type scroller struct {
	top      int             // scroll, UI px
	end      int             // the last top the area stops at, as last drawn
	view     image.Rectangle // the area as last drawn, in UI px
	rows     []scrollRow
	press    image.Point
	pressTop int
	pressed  bool
	moved    bool
	last     int             // the pointer's y at the last frame of the press
	speed    float64         // the pointer's recent speed, px a frame
	fling    float64         // the glide left after a release, px a frame
	bar      image.Rectangle // the scrollbar's track as last drawn, empty when the area fits
	barDrag  bool            // a press on the track is dragging the thumb
	barPress int
	barTop   int
	barPerPx float64 // area px per px of the thumb's travel
}

// scrollRow is a row shown in a scroller: where it is on the screen, and what picking it does.
type scrollRow struct {
	r     image.Rectangle
	index int // the row's place among the area's rows, drawn or not
	click func()
}

// begin sizes the area to view for content px of rows: it clamps the scroll and forgets last frame's rows. It
// gives the clip to draw them in.
func (s *scroller) begin(dst *ebiten.Image, view image.Rectangle, content int) *ebiten.Image {
	s.end = max(0, content-view.Dy())
	s.top = max(0, min(s.top, s.end))
	s.view, s.rows = unframed(view), s.rows[:0]
	return dst.SubImage(image.Rect(int(float64(view.Min.X)*drawPx+drawOX), int(float64(view.Min.Y)*drawPx+drawOY),
		int(float64(view.Max.X)*drawPx+drawOX), int(float64(view.Max.Y)*drawPx+drawOY))).(*ebiten.Image)
}

// drawBar draws the scrollbar along the right edge of view when the rows do not fit.
func (s *scroller) drawBar(dst *ebiten.Image, view image.Rectangle, content int) {
	s.bar = image.Rectangle{}
	if content <= view.Dy() {
		return
	}
	track := image.Rect(view.Max.X-6, view.Min.Y, view.Max.X, view.Max.Y)
	s.bar = unframed(track)
	rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
	th := max(24, view.Dy()*view.Dy()/content)
	s.barPerPx = float64(content-view.Dy()) / float64(max(1, view.Dy()-th))
	ty := view.Min.Y + (view.Dy()-th)*s.top/(content-view.Dy())
	rect(dst, float32(track.Min.X), float32(ty), float32(track.Dx()), float32(th), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
}

// row adds the index-th row, drawn at r (in the clip), that picking runs click on.
func (s *scroller) row(index int, r, view image.Rectangle, click func()) {
	s.rows = append(s.rows, scrollRow{unframed(r.Intersect(view)), index, click})
}

// update scrolls by the wheel or a drag and picks the row a press is let go on without dragging. It tells
// whether the area took the press.
func (s *scroller) update(p image.Point) bool {
	pressedHere := mouseJustPressed(ebiten.MouseButtonLeft) && (p.In(s.view) || p.In(s.bar))
	if _, wheel := ebiten.Wheel(); wheel != 0 && p.In(s.view) {
		s.top = max(0, min(s.end, s.top-int(wheel*40)))
	}
	if pressedHere {
		s.fling = 0
	}
	if s.barDrag {
		if mousePressed(ebiten.MouseButtonLeft) {
			s.top = max(0, min(s.end, s.barTop+int(float64(p.Y-s.barPress)*s.barPerPx+.5)))
		} else {
			s.barDrag = false
		}
		return true
	}
	if pressedHere && p.In(s.bar) {
		s.barDrag, s.barPress, s.barTop = true, p.Y, s.top
		return true
	}
	if pressedHere {
		s.pressed, s.moved, s.press, s.pressTop, s.last, s.speed = true, false, p, s.top, p.Y, 0
	}
	if !s.pressed {
		s.glide()
		return false
	}
	if mousePressed(ebiten.MouseButtonLeft) {
		if abs(p.Y-s.press.Y) > dragStartPx {
			s.moved = true
		}
		if s.moved {
			s.top = max(0, min(s.end, s.pressTop-(p.Y-s.press.Y)))
			s.speed = .6*s.speed + .4*float64(p.Y-s.last)
		}
		s.last = p.Y
		return true
	}
	s.pressed = false
	if s.moved && math.Abs(s.speed) >= flingMinPx {
		s.fling = -s.speed
	}
	if !s.moved {
		for _, r := range s.rows {
			if s.press.In(r.r) {
				r.click()
				break
			}
		}
	}
	return true
}

// glide carries on a glide left by a release, px a frame, stopping at the first or last row.
func (s *scroller) glide() {
	if s.fling == 0 {
		return
	}
	s.top = max(0, min(s.end, s.top+int(s.fling)))
	s.fling *= flingFriction
	if math.Abs(s.fling) < flingRest || s.top == 0 && s.fling < 0 || s.top == s.end && s.fling > 0 {
		s.fling = 0
	}
}

// waypointActFor is the act the list shows: the one picked, the first when that is none of the story's.
func (g *Game) waypointActFor(acts []string) string {
	if slices.Contains(acts, g.waypoints.act) || len(acts) == 0 {
		return g.waypoints.act
	}
	return acts[0]
}

// pickWaypointAct shows another act's waypoints, each act with the scroll it was left at.
func (g *Game) pickWaypointAct(from, to string) {
	w := &g.waypoints
	if w.tops == nil {
		w.tops = map[string]int{}
	}
	w.tops[from] = w.list.top
	w.act, w.list.top, w.list.fling = to, w.tops[to], 0
}

func (g *Game) waypointWindow(dst *ebiten.Image, x, y, w, h int) {
	g.label(dst, "오른쪽에서 레벨과 앞선 전투 결과를 정한 뒤, 왼쪽에서 시작할 곳을 고릅니다. 아이템은 모두 가진 채입니다.", float64(x+32), float64(y+56), 14, muted)
	acts := g.S.Data.WaypointActs()
	act := g.waypointActFor(acts)
	top := y + 86
	if len(acts) > 1 { // a scenario with no acts is one list
		g.tabRow(dst, x+24, y+80, acts, slices.Index(acts, act), func(i int) { g.pickWaypointAct(act, acts[i]) })
		top = y + 80 + touchH + 8
	}
	view := image.Rect(x+24, top, x+24+waypointListW, y+h-24)
	g.waypointSetup(dst, image.Rect(view.Max.X+24, y+86, x+w-24, view.Max.Y))
	ways := slices.DeleteFunc(g.S.Data.Waypoints(), func(wp content.Waypoint) bool { return wp.Act != act })
	total := len(ways)*(waypointRowH+waypointGap) - waypointGap
	l := &g.waypoints.list
	clip := l.begin(dst, view, total)
	for i, wp := range ways {
		top := view.Min.Y + i*(waypointRowH+waypointGap) - l.top
		r := image.Rect(view.Min.X, top, view.Max.X-14, top+waypointRowH)
		if !r.Overlaps(view) {
			continue
		}
		l.row(i, r, view, func() { g.pickWaypoint(wp.Node) })
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
	l.drawBar(dst, view, total)
}

// pickWaypoint asks the difficulty for the waypoint, or starts there when the scenario has none.
func (g *Game) pickWaypoint(node string) {
	if len(g.S.Data.Rules.Difficulties) == 0 {
		g.closeOverlays()
		g.jump(node, "")
		return
	}
	g.jumpNode = node
	g.openOverlay("difficulty")
}

// updateWaypoints scrolls the list and the flags by the wheel or a drag and starts at the row a press is let go
// on without dragging, or switches the flag the press is let go on.
func (g *Game) updateWaypoints() bool {
	w := &g.waypoints
	if g.overlay() != "waypoints" {
		w.list.pressed, w.flags.pressed = false, false
		return false
	}
	x, y := cursorPos()
	p := image.Pt(x, y)
	list, flags := w.list.update(p), w.flags.update(p)
	return list || flags
}

// waypointLevels are the levels the setup offers at a press; the steps beside them reach the ones between.
var waypointLevels = []int{10, 20, 30, 40, 50, 60}

// waypointSetup draws the setup in r: the party's level (none is each waypoint's own) over the battle flags,
// each a switch that names its battle and objective over what it changes later, wrapped to the row's width and
// scrolled when they do not fit. A flag of a battle after the picked waypoint is not used.
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
	lit := 0
	for _, f := range flags {
		if g.jumpFlags[f.Flag] {
			lit++
		}
	}
	g.label(dst, fmt.Sprintf("앞선 전투 결과 %d/%d", lit, len(flags)), float64(x), float64(top), 14, muted)
	g.centeredBtn(dst, image.Rect(x+w-136, top-6, x+w-72, top+22), "모두", func() {
		g.jumpFlags = map[string]bool{}
		for _, f := range flags {
			g.jumpFlags[f.Flag] = true
		}
	})
	g.centeredBtn(dst, image.Rect(x+w-64, top-6, x+w, top+22), "없음", func() { g.jumpFlags = nil })
	view := image.Rect(x, top+30, r.Max.X, r.Max.Y)
	textW := float64(w - 14 - 40 - 10)
	type flagRow struct {
		f      content.StoryFlag
		cause  []string
		effect []string
		h      int
	}
	rows, total := make([]flagRow, len(flags)), 0
	for i, f := range flags {
		later := f.Later
		if later == "" {
			later = "뒤에 달라지는 것 없음"
		}
		cause := g.wrapLines(g.S.Data.Stages[g.S.Data.StageIndex(f.Stage)].Name+" · "+f.Text, 12, textW)
		effect := g.wrapLines(later, 14, textW)
		rows[i] = flagRow{f, cause, effect, max(waypointFlagH, 8+len(cause)*flagCauseH+len(effect)*flagEffectH)}
		total += rows[i].h + 4
	}
	total = max(0, total-4)
	l := &g.waypoints.flags
	clip := l.begin(dst, view, total)
	top = view.Min.Y - l.top
	for i, row := range rows {
		b := image.Rect(view.Min.X, top, view.Max.X-14, top+row.h)
		top += row.h + 4
		if !b.Overlaps(view) {
			continue
		}
		f := row.f
		l.row(i, b, view, func() {
			if g.jumpFlags == nil {
				g.jumpFlags = map[string]bool{}
			}
			g.jumpFlags[f.Flag] = !g.jumpFlags[f.Flag]
		})
		on := g.jumpFlags[f.Flag]
		body, rim, fg := lacquer2, goldDim, ink
		switch {
		case on:
			body, rim, fg = color.NRGBA{R: 26, G: 19, B: 15, A: 255}, gold, gold
		case hovered(b.Intersect(view)):
			body, rim = hoverC, gold
		}
		rect(clip, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), body)
		strokeRect(clip, float32(b.Min.X)+.5, float32(b.Min.Y)+.5, float32(b.Dx())-1, float32(b.Dy())-1, 1, rim)
		if on {
			rect(clip, float32(b.Min.X+13), float32(b.Min.Y+13), 14, 14, gold)
		}
		strokeRect(clip, float32(b.Min.X)+12.5, float32(b.Min.Y)+12.5, 15, 15, 1, goldDim)
		ty := float64(b.Min.Y + 3)
		for _, line := range row.cause {
			g.label(clip, line, float64(b.Min.X+40), ty, 12, muted)
			ty += flagCauseH
		}
		for _, line := range row.effect {
			g.label(clip, line, float64(b.Min.X+40), ty, 14, fg)
			ty += flagEffectH
		}
	}
	l.drawBar(dst, view, total)
}

// wrapLines breaks s into lines no wider than w at size, at spaces where it can.
func (g *Game) wrapLines(s string, size, w float64) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		if line != "" && g.width(line+" "+word, size) <= w {
			line += " " + word
			continue
		}
		if line != "" {
			out = append(out, line)
		}
		line = word
		for g.width(line, size) > w { // a word wider than the line
			r := []rune(line)
			cut := len(r) - 1
			for cut > 1 && g.width(string(r[:cut]), size) > w {
				cut--
			}
			out = append(out, string(r[:cut]))
			line = string(r[cut:])
		}
	}
	return append(out, line)
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
