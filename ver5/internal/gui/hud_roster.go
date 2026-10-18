package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// rosterTable lays the roster window's list out over the width w: the name, then the view's
// cells sharing the rest equally, in text larger than the panel's.
func (g *Game) rosterTable(w int) unitTable {
	cells := unitViewKinds[g.rosterList.kind].cells
	t := unitTable{cells: cells, nameW: 150, rowH: 36, headH: 36, size: 17, center: true}
	t.widths = make([]int, len(cells))
	for i := range t.widths {
		t.widths[i] = (w - t.nameW) / len(cells)
	}
	t.pick = func(id string) {
		g.overlay = ""
		g.focusOfficer(id)
	}
	return t
}

// rosterWindow is the roster overlay: the filter and view buttons over one table that fills the
// window.
func (g *Game) rosterWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.Engine.Observe()
	left, right := x+32, x+w-32
	// the seven buttons shrink with the window so the filters and the views never meet
	bw := min(100, (right-left-12-18-24)/7)
	g.unitFilterRow(dst, &g.rosterList, image.Rect(left, y+54, left+3*bw+12, y+84))
	kx := right - len(unitViewKinds)*bw - 18
	for i, k := range unitViewKinds {
		idx := i
		var click func()
		if g.rosterList.kind != idx {
			click = func() { g.rosterList.kind, g.rosterList.top = idx, 0 }
		}
		bx := kx + i*(bw+6)
		g.btn(dst, image.Rect(bx, y+54, bx+bw, y+84), k.title, click)
	}
	g.unitTable(dst, o, &g.rosterList, g.rosterTable(right-left), image.Rect(left, y+98, right, y+h-20))
}
