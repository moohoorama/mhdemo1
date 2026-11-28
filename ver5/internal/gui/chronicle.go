package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/session"
	"srpg/internal/storage"
)

// The chronicle window (지난 이야기, docs/system-chronicle.md) lists the story's chapters and battles in order, with the
// lord's level and the day each was reached on its right; picking a point asks, then plays on from the state the
// player really had there. Where the story forks (content.Fork) a header offers the arms and the rows below are
// the chosen arm's, the ones never reached dimmed. It has no level or flag to set.
const (
	chronicleW, chronicleH = 860, 640
	chronicleRowH          = 48
	chronicleGap           = 6
	chronicleHeadH         = 70 + touchH + 12 // the fork header: name, note, the arms' chips
)

// chronicleRow is a row of the list: a fork's header (fork set, w empty) or a waypoint on the arm route.
type chronicleRow struct {
	fork  *content.Fork
	w     content.Waypoint
	route string
	top   int
	h     int
	entry *storage.ChronicleEntry
}

// chronicleView is the window's state: the points as read when it opened, the arm chosen on each fork, the rows
// built from them and the list's scroll.
type chronicleView struct {
	points map[string]storage.ChronicleEntry
	arm    map[string]string
	rows   []chronicleRow
	list   scroller
	total  int
}

func chronicleKey(node, route string) string { return node + "@" + route }

func (g *Game) openChronicle() {
	v := chronicleView{points: map[string]storage.ChronicleEntry{}, arm: map[string]string{}}
	latest := map[string]time.Time{} // fork -> the newest point of the arm the default is
	var newest time.Time
	focus := ""
	for _, e := range g.S.Chronicle() {
		v.points[chronicleKey(e.Node, e.Route)] = e
		if e.Bad {
			continue
		}
		if f := g.S.Data.ForkAt(e.Node); f != nil && e.At.After(latest[f.ID]) {
			latest[f.ID], v.arm[f.ID] = e.At, e.Route
		}
		if e.At.After(newest) {
			newest, focus = e.At, e.Node
		}
	}
	g.chron = v
	g.buildChronicle()
	g.chron.list.top = 0
	for _, r := range g.chron.rows {
		if r.fork == nil && r.w.Node == focus {
			g.chron.list.top = max(0, r.top-(chronicleH-160)/2)
		}
	}
}

// buildChronicle lays the rows out for the arms chosen.
func (g *Game) buildChronicle() {
	v := &g.chron
	v.rows = v.rows[:0]
	y := 0
	var in *content.Fork
	for _, w := range g.S.Data.Waypoints() {
		f := g.S.Data.ForkAt(w.Node)
		if f != in && f != nil {
			if !slices.ContainsFunc(f.Arms, func(a content.Arm) bool { return a.ID == v.arm[f.ID] }) {
				v.arm[f.ID] = f.Arms[0].ID
			}
			v.rows = append(v.rows, chronicleRow{fork: f, top: y, h: chronicleHeadH})
			y += chronicleHeadH + chronicleGap
		}
		in = f
		route := ""
		if f != nil {
			arm := f.Arms[slices.IndexFunc(f.Arms, func(a content.Arm) bool { return a.ID == v.arm[f.ID] })]
			if !arm.Shows(w) {
				continue
			}
			route = arm.ID
		}
		r := chronicleRow{w: w, route: route, top: y, h: chronicleRowH}
		if e, ok := v.points[chronicleKey(w.Node, route)]; ok {
			r.entry = &e
		}
		v.rows = append(v.rows, r)
		y += chronicleRowH + chronicleGap
	}
	v.total = max(0, y-chronicleGap)
}

// chronicleName is a waypoint's label: where in the story it is and, for the two tellings that split, which.
func (g *Game) chronicleName(w content.Waypoint) (string, string) {
	tag := ""
	if _, mode, found := strings.Cut(w.Act, "-"); found {
		tag = mode
	}
	if w.Stage != "" {
		return "전투 · " + g.S.Data.Stages[g.S.Data.StageIndex(w.Stage)].Name, tag
	}
	name := fmt.Sprintf("제 %d 장", w.Chapter)
	if w.Place != "" {
		name += " · " + w.Place
	}
	return name, tag
}

func (g *Game) chronicleWindow(dst *ebiten.Image, x, y, w, h int) {
	v := &g.chron
	g.label(dst, "닿은 장과 전투마다 그때의 모습을 기록합니다. 고르면 그 시점부터 다시 시작하며, 같은 곳에 다시 닿으면 기록이 새로 덮입니다.", float64(x+32), float64(y+56), 14, muted)
	if len(v.points) == 0 {
		g.centered(dst, "아직 기록된 곳이 없습니다.", float64(x+w/2), float64(y+h/2-30), 18, ink)
		g.centered(dst, "새 장이나 전투에 닿으면 여기에 남습니다.", float64(x+w/2), float64(y+h/2), 15, muted)
		return
	}
	view := image.Rect(x+24, y+86, x+w-24, y+h-24)
	l := &v.list
	clip := l.begin(dst, view, v.total)
	for i, r := range v.rows {
		rt := view.Min.Y + r.top - l.top
		rr := image.Rect(view.Min.X, rt, view.Max.X-14, rt+r.h)
		if !rr.Overlaps(view) {
			continue
		}
		if r.fork != nil {
			g.chronicleFork(clip, r, rr, view, i)
			continue
		}
		g.chronicleRow(clip, r, rr, view, i)
	}
	l.drawBar(dst, view, v.total)
}

// chronicleFork draws a fork's header: its name, how the chosen arm is come to, and a chip for each arm.
func (g *Game) chronicleFork(clip *ebiten.Image, r chronicleRow, rr, view image.Rectangle, index int) {
	v := &g.chron
	f, cur := r.fork, v.arm[r.fork.ID]
	g.label(clip, "갈림길 · "+f.Name, float64(rr.Min.X+4), float64(rr.Min.Y+4), 16, gold)
	note := ""
	for _, a := range f.Arms {
		if a.ID == cur {
			note = a.Note
			if !g.armKept(f, a) {
				note += " 아직 가 보지 않은 길입니다. 갈림길 앞의 기록에서 다시 시작해 보세요."
			}
		}
	}
	g.label(clip, note, float64(rr.Min.X+4), float64(rr.Min.Y+30), 13, muted)
	x := rr.Min.X
	cy := rr.Min.Y + 70
	for _, a := range f.Arms {
		cw := max(96, int(g.width(a.Name, 15))+32)
		c := image.Rect(x, cy, x+cw, cy+touchH)
		fg := muted
		switch {
		case a.ID == cur:
			rect(clip, float32(c.Min.X), float32(c.Min.Y), float32(cw), touchH, color.NRGBA{R: 46, G: 32, B: 25, A: 255})
			rect(clip, float32(c.Min.X), float32(c.Max.Y-3), float32(cw), 3, gold)
			fg = gold
		case hovered(c.Intersect(view)):
			rect(clip, float32(c.Min.X), float32(c.Min.Y), float32(cw), touchH, color.NRGBA{R: 255, G: 240, B: 210, A: 14})
		}
		if !g.armKept(f, a) && a.ID != cur {
			fg = goldDim
		}
		id := a.ID
		var click func()
		if a.ID != cur {
			click = func() { v.arm[f.ID] = id; g.buildChronicle() }
		}
		v.list.row(index*8+len(v.list.rows), c, view, click)
		g.label(clip, a.Name, float64(c.Min.X)+(float64(cw)-g.width(a.Name, 15))/2, float64(c.Min.Y)+(touchH-15*1.45)/2, 15, fg)
		x += cw + 6
	}
}

// armKept reports whether any waypoint of the arm has a point.
func (g *Game) armKept(f *content.Fork, a content.Arm) bool {
	for k := range g.chron.points {
		node, route, _ := strings.Cut(k, "@")
		if route == a.ID && g.S.Data.ForkAt(node) == f {
			return true
		}
	}
	return false
}

func (g *Game) chronicleRow(clip *ebiten.Image, r chronicleRow, rr, view image.Rectangle, index int) {
	l := &g.chron.list
	e := r.entry
	var click func()
	if e != nil && !e.Bad {
		click = func() { g.askChronicle(r) }
	}
	l.row(index, rr, view, click)
	body, rim, fg := lacquer2, goldDim, ink
	switch {
	case e == nil || e.Bad:
		fg = muted
	case hovered(rr.Intersect(view)):
		body, rim = hoverC, gold
	}
	rect(clip, float32(rr.Min.X), float32(rr.Min.Y), float32(rr.Dx()), float32(rr.Dy()), body)
	strokeRect(clip, float32(rr.Min.X)+.5, float32(rr.Min.Y)+.5, float32(rr.Dx())-1, float32(rr.Dy())-1, 1, rim)
	ty := float64(rr.Min.Y + 14)
	name, tag := g.chronicleName(r.w)
	g.label(clip, name, float64(rr.Min.X+16), ty, 16, fg)
	if tag != "" {
		g.label(clip, tag, float64(rr.Min.X+16)+g.width(name, 16)+12, ty+2, 13, gold)
	}
	right := float64(rr.Max.X - 16)
	switch {
	case e == nil:
		g.label(clip, "아직 가 보지 않음", right-g.width("아직 가 보지 않음", 14), ty+1, 14, muted)
	case e.Bad:
		g.label(clip, "읽을 수 없는 기록", right-g.width("읽을 수 없는 기록", 14), ty+1, 14, muted)
	default:
		date := e.At.Format("01/02 15:04")
		g.label(clip, date, right-g.width(date, 15), ty+1, 15, muted)
		lv := fmt.Sprintf("Lv %d", e.Level)
		g.label(clip, lv, right-g.width(date, 15)-24-g.width(lv, 16), ty, 16, fg)
	}
}

// askChronicle asks before playing on from a point, and says what stays.
func (g *Game) askChronicle(r chronicleRow) {
	name, _ := g.chronicleName(r.w)
	e := r.entry
	g.askWith(fmt.Sprintf("「%s」 (Lv %d · %s)에서 다시 시작합니다.\n지금 진행과 저장은 그대로 남습니다.", name, e.Level, e.At.Format("01/02 15:04")), "시작", "취소", func() {
		g.request(session.Request{Op: "chronicle_load", Node: e.Node, Route: e.Route})
	})
}

// updateChronicle scrolls the list by the wheel or a drag and picks the row a press is let go on without dragging.
func (g *Game) updateChronicle() bool {
	if g.overlay() != "chronicle" {
		g.chron.list.pressed = false
		return false
	}
	x, y := cursorPos()
	return g.chron.list.update(image.Pt(x, y))
}
