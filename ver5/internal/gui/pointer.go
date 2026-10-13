package gui

import (
	"math"
	"sort"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type touchPoint struct {
	id   ebiten.TouchID
	x, y int
}

type touchPointer struct {
	x, y                                    int
	held, justPressed, used, blocked, multi bool
	distance, ratio, dx, dy                 float64
}

var pointer touchPointer
var backRequested atomic.Bool
var backPressed bool

// RequestBack is safe to call from the Android UI thread.
func RequestBack() { backRequested.Store(true) }

func keyJustPressed(key ebiten.Key) bool {
	return key == ebiten.KeyEscape && backPressed || inpututil.IsKeyJustPressed(key)
}

func mouseJustPressed(button ebiten.MouseButton) bool {
	return button == ebiten.MouseButtonLeft && pointer.justPressed || inpututil.IsMouseButtonJustPressed(button)
}

func mousePressed(button ebiten.MouseButton) bool {
	return button == ebiten.MouseButtonLeft && pointer.held || ebiten.IsMouseButtonPressed(button)
}

func (p *touchPointer) update(points []touchPoint) {
	wasHeld, wasMulti := p.held, p.multi
	oldX, oldY, oldDistance := p.x, p.y, p.distance
	p.held, p.justPressed, p.multi = false, false, len(points) >= 2
	p.ratio, p.dx, p.dy = 1, 0, 0
	if len(points) == 0 {
		p.blocked, p.distance = false, 0
		return
	}
	p.used = true
	p.x, p.y = points[0].x, points[0].y
	if p.multi {
		p.blocked = true
		p.x, p.y = (points[0].x+points[1].x)/2, (points[0].y+points[1].y)/2
		p.distance = math.Hypot(float64(points[1].x-points[0].x), float64(points[1].y-points[0].y))
		if wasMulti {
			p.dx, p.dy = float64(p.x-oldX), float64(p.y-oldY)
			if oldDistance > 0 && p.distance > 0 {
				p.ratio = p.distance / oldDistance
			}
		}
		return
	}
	if !p.blocked {
		p.held, p.justPressed = true, !wasHeld
	}
}

func (g *Game) updatePointer() {
	backPressed = backRequested.Swap(false)
	ids := ebiten.AppendTouchIDs(nil)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	points := make([]touchPoint, 0, len(ids))
	for _, id := range ids {
		x, y := ebiten.TouchPosition(id)
		points = append(points, touchPoint{id, x, y})
	}
	pointer.update(points)
	if !pointer.multi {
		return
	}
	g.dragging = false
	x, y := cursorPos()
	if g.editor != nil {
		e := g.editor
		if e.doc == nil || e.field == nil || e.popup != "" || g.overlay != "" || e.resizing || x < editorViewport.Min.X || x >= editorViewport.Max.X || y < editorViewport.Min.Y || y >= editorViewport.Max.Y {
			return
		}
		e.zoom = max(.1, min(3, e.zoom*pointer.ratio))
		e.cx -= pointer.dx / scale / e.zoom
		e.cy -= pointer.dy / scale / e.zoom
		e.clampCamera()
	} else if g.field != nil && !g.popupPending() && !g.input && !g.overHUD(x, y) && x >= 0 && x < Width && y >= g.battleViewport().Min.Y && y < g.battleViewport().Max.Y {
		g.zoom = max(.5, min(2, g.zoom*pointer.ratio))
		g.camX -= pointer.dx / scale / g.zoom
		g.camY -= pointer.dy / scale / g.zoom
	}
}
