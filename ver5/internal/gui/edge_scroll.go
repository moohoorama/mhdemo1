package gui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// edgeWays are the ways the map scrolls, in the order of the edge_arrow frames.
var edgeWays = [8][2]int{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}

// edgeHoldDelay is how long a press on the edge glides one step before it scrolls on while held.
const edgeHoldDelay = .3

// edgeArrow is the way the battle map would scroll from where the cursor is, as an edge_arrow frame,
// if any: the cursor must be in the band along the viewport's edge (5% of the window's width), off
// the HUD, and the camera able to go that way. Story scenes and the walk follow Liu Bei instead.
func (g *Game) edgeArrow(locked bool) (int, bool) {
	if g.field == nil || locked || pointer.used || len(g.Shots) > 0 || g.dragging {
		return 0, false
	}
	if o := g.S.View(); o.Phase != "battle" && o.Phase != "deployment" || g.exploring() || g.scene != nil {
		return 0, false
	}
	x, y := cursorPos()
	wx, wy := cursorWorld()
	vp := g.battleViewport()
	if !image.Pt(int(wx), int(wy)).In(vp) || g.overHUD(x, y) {
		return 0, false
	}
	edge := float64(worldW) * .05
	dx, dy := 0, 0
	switch {
	case wx < float64(vp.Min.X)+edge && g.camCanGo(-2, 0):
		dx = -1
	case wx >= float64(vp.Max.X)-edge && g.camCanGo(2, 0):
		dx = 1
	}
	switch {
	case wy < float64(vp.Min.Y)+edge && g.camCanGo(0, -2):
		dy = -1
	case wy >= float64(vp.Max.Y)-edge && g.camCanGo(0, 2):
		dy = 1
	}
	for i, w := range edgeWays {
		if w == [2]int{dx, dy} {
			return i, true
		}
	}
	return 0, false
}

// edgeScroll moves the camera while the left button is pressed on an edge arrow: a press glides a
// quarter of the view that way, holding it on scrolls on at the arrow keys' speed.
func (g *Game) edgeScroll(step float64) {
	if !g.edgeOn || !mousePressed(ebiten.MouseButtonLeft) {
		g.edgeHeld = 0
		return
	}
	w := edgeWays[g.edgeDir]
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		vp := g.battleViewport()
		g.panTo(g.camX+float64(w[0]*vp.Dx())/4/g.zoom, g.camY+float64(w[1]*vp.Dy())/4/g.zoom)
	}
	if g.edgeHeld += 1.0 / 60; g.edgeHeld > edgeHoldDelay {
		g.pan = nil
		g.camX += float64(w[0]) * step
		g.camY += float64(w[1]) * step
	}
}

// drawEdgeArrow draws the edge arrow, two UI px a dot on whole device pixels so its dots stay
// square, in place of the mouse cursor, which it hides meanwhile.
func (g *Game) drawEdgeArrow(dst *ebiten.Image) {
	hide := g.edgeOn
	if hide != g.cursorHidden {
		g.cursorHidden = hide
		mode := ebiten.CursorModeVisible
		if hide {
			mode = ebiten.CursorModeHidden
		}
		ebiten.SetCursorMode(mode)
	}
	if !hide {
		return
	}
	s := g.assets.UI["edge_arrow"]
	dot := max(1, math.Round(2*drawPx)) // device px a dot
	// the tip, 9 dots from the centre straight on and 8.5 each way on a diagonal, sits on the mouse
	w := edgeWays[g.edgeDir]
	tip := 9.0
	if w[0] != 0 && w[1] != 0 {
		tip = 8.5
	}
	x, y := cursorPos()
	cx := math.Round((float64(x)-float64(w[0])*tip*2)*drawPx + drawOX)
	cy := math.Round((float64(y)-float64(w[1])*tip*2)*drawPx + drawOY)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(-s.Anchor[0], -s.Anchor[1])
	op.GeoM.Scale(dot/4, dot/4)
	op.GeoM.Translate(cx, cy)
	dst.DrawImage(s.Frames[g.edgeDir], op)
}
