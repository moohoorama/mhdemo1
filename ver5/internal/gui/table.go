package gui

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// A table is a list of rows under a header whose cells sort it: none, descending, ascending. The
// unit list, the roster window, the deputy window and the learning window draw theirs with it.

const tablePadBelow = 10

// dimInk is the text of a row that cannot be acted on.
var dimInk = color.NRGBA{R: 150, G: 142, B: 128, A: 255}

// tableSort orders a table by a column key: none, descending or ascending.
type tableSort struct {
	key string
	dir int // 0 none, 1 ascending, 2 descending
}

// cycle is a click on a header: a new key starts descending, then ascending, then none.
func (s tableSort) cycle(key string) tableSort {
	switch {
	case s.key != key:
		return tableSort{key, 2}
	case s.dir == 2:
		return tableSort{key, 1}
	}
	return tableSort{}
}

// tableList is the state of one table: its first row shown, its sort, the glide left after a release
// (rows a frame, see updateFlings) and the last row a glide may stop at (set when drawn).
type tableList struct {
	top            int
	order          tableSort
	wheel          float64
	fling          float64
	end            int
	revealSelected bool
}

func (l *tableList) scroll(wheel float64) {
	l.wheel += wheel
	rows := int(l.wheel)
	l.top -= rows
	l.wheel -= float64(rows)
}

// column is one column of a table: how a row fills it and sorts by it.
type column[R any] struct {
	key, label string
	w          int     // 0 takes the width the other columns leave
	text       bool    // sorts by the text rather than the number
	left       bool    // keeps to the left in a centred table
	pad        int     // room before the text, for what after draws there
	size       float64 // the text size against the table's
	get        func(r *R) (string, float64)
	// show is what the cell shows within the width w and in what colour, when not get's text in
	// the row's ink.
	show  func(r *R, ink color.Color, w float64) (string, color.Color)
	bold  func(r *R) bool
	after func(dst *ebiten.Image, r *R, cell image.Rectangle)
}

// table is how a list is laid out and what its rows do. rank keeps rows in bands whatever the sort,
// the lowest first; open says which rows a click picks (nil: all).
type table[R any] struct {
	cols        []column[R]
	rowH, headH int
	size        float64
	center      bool
	empty       string
	pinFirst    bool
	rank        func(r *R) int
	open        func(r *R) bool
	dim         func(r *R) bool
	selected    func(r *R) bool
	pick        func(r *R)
	guide       func(r *R) bool                                   // rows the promotion guide rims in orange
	back        func(dst *ebiten.Image, r *R, rr image.Rectangle) // paints a row's ground under its cells
}

// widths are the columns' widths over w, the free columns sharing what the others leave.
func (t table[R]) widths(w int) []int {
	out, free, rest := make([]int, len(t.cols)), 0, w
	for i, c := range t.cols {
		out[i] = c.w
		rest -= c.w
		if c.w == 0 {
			free++
		}
	}
	for i, c := range t.cols {
		if c.w == 0 {
			out[i] = rest / free
		}
	}
	return out
}

// keepSort drops a sort whose column the table does not show.
func keepSort[R any](l *tableList, cols []column[R]) {
	for _, c := range cols {
		if c.key == l.order.key {
			return
		}
	}
	l.order = tableSort{}
}

// sortRows orders the rows by the column the sort names within the bands rank gives.
func sortRows[R any](rows []R, cols []column[R], s tableSort, rank func(r *R) int) {
	less := func(a, b *R) bool { return false }
	for _, c := range cols {
		if s.dir == 0 || c.key != s.key {
			continue
		}
		less = func(a, b *R) bool {
			at, an := c.get(a)
			bt, bn := c.get(b)
			if c.text {
				return at < bt
			}
			return an < bn
		}
		if s.dir == 2 {
			asc := less
			less = func(a, b *R) bool { return asc(b, a) }
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := &rows[i], &rows[j]
		if rank != nil && rank(a) != rank(b) {
			return rank(a) < rank(b)
		}
		return less(a, b)
	})
}

// drawTable sorts the rows and draws the header and as many rows as fit in r, with a scrollbar on
// the right when more exist. The rows scroll by the wheel (the window's update), a drag on them or
// on the scrollbar, and a press on the bar's track pages; a row is picked when a press on it is let
// go without dragging.
func drawTable[R any](g *Game, dst *ebiten.Image, l *tableList, t table[R], rows []R, r image.Rectangle) {
	area := g.beginScroll(r, l, t.rowH)
	keepSort(l, t.cols)
	sortRows(rows, t.cols, l.order, t.rank)
	pinned := 0
	if t.pinFirst && len(rows) > 0 {
		pinned = 1
	}
	fit := max(1, (r.Dy()-t.headH-tablePadBelow)/t.rowH-pinned)
	if l.revealSelected {
		l.revealSelected = false
		if t.selected != nil {
			for i := range rows[pinned:] {
				if t.selected(&rows[pinned+i]) {
					l.top = max(i-fit+1, min(l.top, i))
					break
				}
			}
		}
	}
	l.top = max(0, min(l.top, len(rows)-pinned-fit))
	l.end = max(0, len(rows)-pinned-fit)
	scroll := len(rows)-pinned > fit
	tw := r.Dx()
	if scroll {
		tw -= tableBarW + 4
	}
	ws := t.widths(tw)
	x, hy := r.Min.X, r.Min.Y
	for i, c := range t.cols {
		g.tableHeader(dst, l, c.key, c.label, image.Rect(x, hy, x+ws[i], hy+t.headH), c.pad, t.size-1, t.center && !c.left)
		x += ws[i]
	}
	rect(dst, float32(r.Min.X), float32(hy+t.headH-1), float32(r.Dx()), 1, goldDim)

	ry := hy + t.headH
	if pinned > 0 {
		tableRow(g, dst, l, t, ws, &rows[0], image.Rect(r.Min.X, ry, r.Min.X+tw, ry+t.rowH))
		ry += t.rowH
	}
	start := pinned + l.top
	for i := start; i < len(rows) && i < start+fit; i++ {
		rr := image.Rect(r.Min.X, ry+(i-start)*t.rowH, r.Min.X+tw, ry+(i-start+1)*t.rowH)
		tableRow(g, dst, l, t, ws, &rows[i], rr)
		if t.guide != nil && t.guide(&rows[i]) {
			g.promoteRim(dst, rr.Inset(3))
		}
	}
	if len(rows) == 0 && t.empty != "" {
		g.label(dst, t.empty, float64(r.Min.X+4), float64(ry+8), t.size, muted)
	}
	if scroll {
		g.tableBar(dst, l, image.Rect(r.Max.X-tableBarW, ry, r.Max.X, ry+fit*t.rowH), fit, len(rows)-pinned)
	}
	g.endScroll(area)
}

const tableBarW = 10

// addGrab registers a button whose click runs at once, even inside a scroll area: the scrollbar's own.
func (g *Game) addGrab(r image.Rectangle, click func()) {
	n := len(g.buttons)
	g.addButton(r, "", click)
	if len(g.buttons) > n {
		g.buttons[n].grab = true
	}
}

// tableBar draws the scrollbar over track for fit of n rows and registers its press: on the thumb
// it drags the thumb, elsewhere it pages toward the press.
func (g *Game) tableBar(dst *ebiten.Image, l *tableList, track image.Rectangle, fit, n int) {
	th := max(24, track.Dy()*fit/n)
	ty := track.Min.Y + (track.Dy()-th)*l.top/(n-fit)
	thumb := image.Rect(track.Min.X, ty, track.Max.X, ty+th)
	c := color.NRGBA{R: 185, G: 170, B: 135, A: 220}
	if d := g.tableDrag; hovered(thumb) || d != nil && d.l == l && d.perPx > 0 {
		c = color.NRGBA{R: 225, G: 205, B: 150, A: 240}
	}
	scrollBarStyles[scrollBarStyle](g, dst, track, thumb, c)
	g.addGrab(track, func() {
		x, y := cursorPos()
		switch p := image.Pt(x, y); {
		case p.In(unframed(thumb)):
			g.tableDrag = &tableDrag{l: l, press: p, top: l.top, perPx: float64(n-fit) / float64(max(1, track.Dy()-th))}
		case p.Y < unframed(thumb).Min.Y:
			l.top -= fit
		default:
			l.top += fit
		}
	})
}

// tableDrag is a press on a table being held: dragging the rows or the scrollbar's thumb, or a row
// to pick when it is let go without moving. speed is the pointer's recent speed in px a frame.
type tableDrag struct {
	l     *tableList
	rowH  int
	press image.Point
	top   int
	moved bool
	perPx float64 // rows per px of the thumb's travel, while dragging the thumb
	pick  func()
	last  int
	speed float64
}

// Glide after a release: the rows keep going a little, each frame less, and stop at the first or last row. A
// press stops it at once.
const (
	flingFriction = .92
	flingRest     = .02 // rows a frame
	flingMinPx    = 3   // px a frame the pointer must have been moving at the release
)

// pressTable starts holding a press on l's rows; pick runs if it is let go without dragging.
func (g *Game) pressTable(l *tableList, rowH int, pick func()) {
	x, y := cursorPos()
	g.tableDrag = &tableDrag{l: l, rowH: rowH, press: image.Pt(x, y), top: l.top, pick: pick, last: y}
}

// updateTableDrag follows the press held on a table: the rows follow the pointer once it moves
// past dragStartPx, the thumb at once; letting go of a row that did not move picks it, and letting
// go of rows that moved fast leaves them gliding.
func (g *Game) updateTableDrag() bool {
	d := g.tableDrag
	if d == nil {
		return false
	}
	_, y := cursorPos()
	dy := y - d.press.Y
	if mousePressed(ebiten.MouseButtonLeft) {
		switch {
		case d.perPx > 0:
			d.l.top = d.top + int(float64(dy)*d.perPx+.5)
		case d.moved || abs(dy) > dragStartPx:
			d.moved = true
			d.l.top = d.top - dy/d.rowH
			d.speed = .6*d.speed + .4*float64(y-d.last)
		}
		d.last = y
		return true
	}
	g.tableDrag = nil
	if d.moved && d.perPx == 0 && math.Abs(d.speed) >= flingMinPx {
		d.l.fling = -d.speed / float64(d.rowH)
		g.flinging = d.l
	}
	if !d.moved && d.pick != nil {
		d.pick()
	}
	return true
}

// updateFlings carries on a glide left by a release. It never takes the input: a press stops it and goes on as
// a press.
func (g *Game) updateFlings() bool {
	l := g.flinging
	if l == nil {
		return false
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) || g.tableDrag != nil {
		l.fling, g.flinging = 0, nil
		return false
	}
	l.scroll(-l.fling)
	l.fling *= flingFriction
	l.top = max(0, min(l.end, l.top))
	if math.Abs(l.fling) < flingRest || l.top == 0 && l.fling < 0 || l.top == l.end && l.fling > 0 {
		l.fling, g.flinging = 0, nil
	}
	return false
}

// tableRow draws one row over rr and registers its click.
func tableRow[R any](g *Game, dst *ebiten.Image, l *tableList, t table[R], ws []int, row *R, rr image.Rectangle) {
	open := t.open == nil || t.open(row)
	var pick func()
	if open && t.pick != nil {
		pick = func() { t.pick(row) }
	}
	g.addButton(rr, "", func() {
		if pick != nil {
			pick()
		}
	})
	if t.back != nil {
		t.back(dst, row, rr)
	}
	switch {
	case t.selected != nil && t.selected(row):
		strokeRect(dst, float32(rr.Min.X), float32(rr.Min.Y+1), float32(rr.Dx()-2), float32(rr.Dy()-2), 1.5, gold)
	case open && hovered(rr):
		rect(dst, float32(rr.Min.X), float32(rr.Min.Y+1), float32(rr.Dx()-2), float32(rr.Dy()-2), color.NRGBA{R: 255, G: 255, B: 255, A: 16})
	}
	rect(dst, float32(rr.Min.X), float32(rr.Max.Y-1), float32(rr.Dx()-2), 1, color.NRGBA{R: 255, G: 255, B: 255, A: 14})

	ink2 := color.Color(ink)
	if t.dim != nil && t.dim(row) {
		ink2 = dimInk
	}
	x := rr.Min.X
	for i, c := range t.cols {
		w, size := ws[i], t.size+c.size
		s, _ := c.get(row)
		tc := ink2
		if c.show != nil {
			s, tc = c.show(row, ink2, float64(w-6-c.pad))
		}
		s = g.clip(s, size, float64(w-6-c.pad))
		tx := float64(x + 4 + c.pad)
		if t.center && !c.left {
			tx = float64(x) + (float64(w)-g.width(s, size))/2
		}
		ty := float64(rr.Min.Y) + (float64(t.rowH)-size*1.25)/2
		if l.order.key == c.key && l.order.dir != 0 || c.bold != nil && c.bold(row) {
			write(dst, s, g.boldFace(size), tx, ty, tc)
		} else {
			g.label(dst, s, tx, ty, size, tc)
		}
		if c.after != nil {
			c.after(dst, row, image.Rect(x, rr.Min.Y, x+w, rr.Max.Y))
		}
		x += w
	}
}

// tableHeader draws a header cell with the sort mark and registers its click.
func (g *Game) tableHeader(dst *ebiten.Image, l *tableList, key, label string, r image.Rectangle, pad int, size float64, center bool) {
	g.addButton(r, label, func() { l.order = l.order.cycle(key) })
	c := muted
	sorted := l.order.key == key && l.order.dir != 0
	if sorted {
		c = gold
		rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()-2), float32(r.Dy()), color.NRGBA{R: 255, G: 210, B: 120, A: 18})
	} else if hovered(r) {
		c = ink
	}
	ty := float64(r.Min.Y) + (float64(r.Dy())-size*1.25)/2
	lx := float64(r.Min.X + 4 + pad)
	if center {
		lx = float64(r.Min.X) + (float64(r.Dx())-g.width(label, size))/2
	}
	g.label(dst, label, lx, ty, size, c)
	if sorted {
		g.sortMark(dst, float32(lx+g.width(label, size)+9), float32(ty+size*.6), l.order.dir == 1, gold)
	}
}

// sortMark is a small triangle centred on (cx, cy): its tip up for ascending, down for
// descending.
func (g *Game) sortMark(dst *ebiten.Image, cx, cy float32, up bool, c color.NRGBA) {
	frame := 1
	if up {
		frame = 0
	}
	g.drawUI(dst, "sort_mark", frame, float64(cx), float64(cy), 1, c)
}

// boldFace is the sans face for emphasized values.
func (g *Game) boldFace(size float64) *text.GoTextFace {
	f := &text.GoTextFace{Source: g.font, Size: size}
	f.SetVariation(text.MustParseTag("wght"), 800)
	return f
}

// clip shortens s with an ellipsis to the width w at the size.
func (g *Game) clip(s string, size, w float64) string {
	if g.width(s, size) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && g.width(string(r)+"…", size) > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// b2i is 1 for true, for ranks.
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
