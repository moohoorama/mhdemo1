package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// rosterTable lays the roster window's list out: the name, then the view's cells sharing the
// rest equally, in text larger than the panel's.
func (g *Game) rosterTable() unitTable {
	t := unitTable{cells: unitViewKinds[g.rosterList.kind].cells, nameW: 150, rowH: 36, headH: 36, size: 17, center: true}
	t.pick = func(id string) {
		g.closeOverlays()
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
	o := g.S.View()
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
	g.unitTable(dst, o, &g.rosterList, g.rosterTable(), image.Rect(left, y+98, right, y+h-20))
}
