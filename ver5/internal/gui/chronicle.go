package gui

import (
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/session"
	"srpg/internal/storage"
)

// The chronicle window (지난 이야기, docs/system-chronicle.md) lists the chapters and battles this game's lines have reached.
// Each row shows the lord's level and the day it was reached on its right; picking one asks, then plays on
// from the state the player really had there. It has no level or flag to set, and lists only what was reached.
const (
	chronicleW, chronicleH = 860, 640
	chronicleRowH          = 48
	chronicleGap           = 6
	chronicleTabs          = 7 // lines the tab row shows at once
)

// chronicleView is the window's state: the lines as read when it opened, the line shown, the first tab shown and
// the list's scroll.
type chronicleView struct {
	lines []session.ChronicleLine
	line  int
	first int
	list  scroller
	way   map[string]content.Waypoint
}

func (g *Game) openChronicle() {
	v := chronicleView{lines: g.S.Chronicle(), way: map[string]content.Waypoint{}}
	for _, w := range g.S.Data.Waypoints() {
		v.way[w.Node] = w
	}
	v.line = max(0, len(v.lines)-1)
	for i, l := range v.lines {
		if l.Current {
			v.line = i
		}
	}
	v.first = max(0, v.line-chronicleTabs+1)
	v.list.top = 1 << 20 // the newest points show first
	g.chron = v
}

// chronicleName is a point's label: where in the story it is and, for the two tellings that split, which.
func (g *Game) chronicleName(e storage.ChronicleEntry) (string, string) {
	w, ok := g.chron.way[e.Node]
	if !ok {
		return "알 수 없는 지점", ""
	}
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
	g.label(dst, "닿은 장과 전투마다 그때의 모습을 기록합니다. 고르면 지나온 시점부터 다시 시작하며, 지금까지의 진행은 그대로 남습니다.", float64(x+32), float64(y+56), 14, muted)
	if len(v.lines) == 0 {
		g.centered(dst, "아직 기록된 곳이 없습니다.", float64(x+w/2), float64(y+h/2-30), 18, ink)
		g.centered(dst, "새 장이나 전투에 닿으면 여기에 남습니다.", float64(x+w/2), float64(y+h/2), 15, muted)
		return
	}
	top := y + 86
	v.line = max(0, min(v.line, len(v.lines)-1))
	if len(v.lines) > 1 {
		last := min(len(v.lines), v.first+chronicleTabs)
		labels := make([]string, 0, chronicleTabs)
		for i := v.first; i < last; i++ {
			labels = append(labels, fmt.Sprintf("진행 %d", v.lines[i].Number))
		}
		g.tabRow(dst, x+24, top, labels, v.line-v.first, func(i int) { g.pickChronicleLine(v.first + i) })
		if v.first > 0 {
			g.centeredBtn(dst, image.Rect(x+w-24-2*(touchH+6), top, x+w-24-touchH-6, top+touchH), "‹", func() { v.first-- })
		}
		if last < len(v.lines) {
			g.centeredBtn(dst, image.Rect(x+w-24-touchH, top, x+w-24, top+touchH), "›", func() { v.first++ })
		}
		top += touchH + 8
	}
	line := v.lines[v.line]
	note := fmt.Sprintf("진행 %d", line.Number)
	switch {
	case line.From != nil:
		from, _ := g.chronicleName(line.From.Entry)
		note += fmt.Sprintf(" · 진행 %d의 「%s」에서 다시 시작해 갈라진 기록", line.From.Number, from)
	default:
		note += " · 처음부터 이어 온 기록"
	}
	if line.Current {
		note += " · 지금 하는 진행"
	}
	g.label(dst, note, float64(x+32), float64(top+2), 14, gold)
	top += 30

	view := image.Rect(x+24, top, x+w-24, y+h-24)
	total := max(0, len(line.Entries)*(chronicleRowH+chronicleGap)-chronicleGap)
	l := &v.list
	clip := l.begin(dst, view, total)
	if line.Err != "" {
		g.label(clip, "이 기록을 읽을 수 없습니다.", float64(view.Min.X+16), float64(view.Min.Y+16), 16, bad)
	}
	for i, e := range line.Entries {
		rt := view.Min.Y + i*(chronicleRowH+chronicleGap) - l.top
		r := image.Rect(view.Min.X, rt, view.Max.X-14, rt+chronicleRowH)
		if !r.Overlaps(view) {
			continue
		}
		var click func()
		if !e.Bad {
			click = func() { g.askChronicle(line, e) }
		}
		l.row(i, r, view, click)
		body, rim, fg := lacquer2, goldDim, ink
		switch {
		case e.Bad:
			fg = muted
		case hovered(r.Intersect(view)):
			body, rim = hoverC, gold
		}
		rect(clip, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), body)
		strokeRect(clip, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx())-1, float32(r.Dy())-1, 1, rim)
		ty := float64(rt + 14)
		if e.Bad {
			g.label(clip, "읽을 수 없는 기록", float64(r.Min.X+16), ty, 16, fg)
			continue
		}
		name, tag := g.chronicleName(e)
		g.label(clip, name, float64(r.Min.X+16), ty, 16, fg)
		tx := float64(r.Min.X+16) + g.width(name, 16) + 12
		if tag != "" {
			g.label(clip, tag, tx, ty+2, 13, gold)
			tx += g.width(tag, 13) + 10
		}
		if w := g.chron.way[e.Node]; w.Stage != "" {
			status := "도전 중"
			if i < len(line.Entries)-1 {
				status = "승리"
			}
			g.label(clip, status, tx, ty+2, 13, muted)
			tx += g.width(status, 13) + 10
		}
		if e.Line != line.ID {
			g.label(clip, fmt.Sprintf("진행 %d와 같음", g.chronicleNumber(e.Line)), tx, ty+2, 13, muted)
		}
		date := e.At.Format("01/02 15:04")
		right := float64(r.Max.X - 16)
		g.label(clip, date, right-g.width(date, 15), ty+1, 15, muted)
		lv := fmt.Sprintf("Lv %d", e.Level)
		g.label(clip, lv, right-g.width(date, 15)-24-g.width(lv, 16), ty, 16, fg)
	}
	l.drawBar(dst, view, total)
}

func (g *Game) chronicleNumber(line string) int {
	for _, l := range g.chron.lines {
		if l.ID == line {
			return l.Number
		}
	}
	return 0
}

func (g *Game) pickChronicleLine(i int) {
	v := &g.chron
	v.line, v.list = i, scroller{top: 1 << 20}
}

// askChronicle asks before playing on from a point, and says what stays.
func (g *Game) askChronicle(line session.ChronicleLine, e storage.ChronicleEntry) {
	name, _ := g.chronicleName(e)
	g.askWith(fmt.Sprintf("「%s」 (Lv %d · %s)에서 다시 시작합니다.\n지금 진행과 저장은 그대로 남습니다.", name, e.Level, e.At.Format("01/02 15:04")), "시작", "취소", func() {
		g.request(session.Request{Op: "chronicle_load", Line: line.ID, Seq: e.Seq})
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
