package gui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// browserPanel leaves its heading and close row outside the scrollable content.
func (g *Game) browserPanel(dst *ebiten.Image, title string, total int, close func()) image.Rectangle {
	w, h := sheetSize()
	x, y := (uiW-w)/2, (uiH-h)/2
	g.window(dst, image.Rect(x, y, x+w, y+h))
	g.label(dst, title, float64(x+20), float64(y+18), 24, gold)
	if total >= 0 {
		g.label(dst, fmt.Sprintf("전체 %d개", total), float64(x+20), float64(y+54), 14, muted)
	}
	g.closeRow(dst, x+w-20, y+h-touchH-16, close)
	return image.Rect(x+20, y+84, x+w-20, y+h-touchH-28)
}

func (g *Game) editorMapList(dst *ebiten.Image) {
	e := g.editor
	stages := g.S.Data.Stages
	r := g.browserPanel(dst, "전투맵 선택", len(stages), func() { g.editor = nil })
	g.listRows(dst, &e.maps, r, len(stages), listRowH, func(i int, rr image.Rectangle) {
		st := stages[i]
		rowBack(dst, rr, hovered(rr))
		g.label(dst, ellipsis(g, st.Name, 16, rr.Dx()-32), float64(rr.Min.X+12), float64(rr.Min.Y+6), 16, ink)
		g.label(dst, fmt.Sprintf("%s · %d×%d", st.ID, st.Width, st.Height), float64(rr.Min.X+12), float64(rr.Min.Y+31), 12, muted)
	}, g.editorOpen)
	if e.message != "" {
		g.label(dst, ellipsis(g, e.message, 12, r.Dx()-132), float64(r.Min.X), float64(r.Max.Y+22), 12, bad)
	}
}
