package gui

import (
	"image"
	"image/color"
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

// tableList is the state of one table: its first row shown and its sort.
type tableList struct {
	top   int
	order tableSort
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
	after func(dst *ebiten.Image, r *R, cell image.Rectangle)
}

// table is how a list is laid out and what its rows do. first puts rows ahead whatever the sort;
// open says which rows a click picks (nil: all).
type table[R any] struct {
	cols        []column[R]
	rowH, headH int
	size        float64
	center      bool
	empty       string
	first       func(r *R) bool
	open        func(r *R) bool
	dim         func(r *R) bool
	selected    func(r *R) bool
	pick        func(r *R)
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

// sortRows orders the rows by the column the sort names, those first says first ahead.
func sortRows[R any](rows []R, cols []column[R], s tableSort, first func(r *R) bool) {
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
		if first != nil && first(a) != first(b) {
			return first(a)
		}
		return less(a, b)
	})
}

// drawTable sorts the rows and draws the header and as many rows as fit in r, with a scrollbar
// when more exist.
func drawTable[R any](g *Game, dst *ebiten.Image, l *tableList, t table[R], rows []R, r image.Rectangle) {
	keepSort(l, t.cols)
	sortRows(rows, t.cols, l.order, t.first)
	ws := t.widths(r.Dx())
	x, hy := r.Min.X, r.Min.Y
	for i, c := range t.cols {
		g.tableHeader(dst, l, c.key, c.label, image.Rect(x, hy, x+ws[i], hy+t.headH), c.pad, t.size-1, t.center && !c.left)
		x += ws[i]
	}
	rect(dst, float32(r.Min.X), float32(hy+t.headH-1), float32(r.Dx()), 1, goldDim)

	fit := max(1, (r.Dy()-t.headH-tablePadBelow)/t.rowH)
	l.top = max(0, min(l.top, len(rows)-fit))
	ry := hy + t.headH
	for i := l.top; i < len(rows) && i < l.top+fit; i++ {
		tableRow(g, dst, l, t, ws, &rows[i], image.Rect(r.Min.X, ry+(i-l.top)*t.rowH, r.Max.X, ry+(i-l.top+1)*t.rowH))
	}
	if len(rows) == 0 && t.empty != "" {
		g.label(dst, t.empty, float64(r.Min.X+4), float64(ry+8), t.size, muted)
	}
	if len(rows) > fit {
		track := image.Rect(r.Max.X-6, ry, r.Max.X, ry+fit*t.rowH)
		rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
		th := max(18, track.Dy()*fit/len(rows))
		ty := track.Min.Y + (track.Dy()-th)*l.top/(len(rows)-fit)
		rect(dst, float32(track.Min.X), float32(ty), float32(track.Dx()), float32(th), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
	}
}

// tableRow draws one row over rr and registers its click.
func tableRow[R any](g *Game, dst *ebiten.Image, l *tableList, t table[R], ws []int, row *R, rr image.Rectangle) {
	open := t.open == nil || t.open(row)
	if open && t.pick != nil {
		g.addButton(rr, "", func() { t.pick(row) })
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
		if l.order.key == c.key && l.order.dir != 0 {
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

// boldFace is the sans face at the weight used for the values a table is sorted by.
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
