package gui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// A popup window (an overlay, a modal, the result or learning window) comes down from just
// above with a small overshoot while it unrolls from its middle line; when it goes away it
// sinks a little as it rolls up again. The screen behind dims with it. Clicks reach the
// window's final place from the first frame, so the motion never delays play.
const (
	windowUnroll  = .38 // seconds
	windowDrop    = .42
	windowDropBy  = 40.0 // UI px above its place it starts from
	windowDim     = .22
	windowClose   = .24
	windowSinkBy  = 14.0
	windowDimFade = .16
	windowSliver  = .04 // fraction of its height a window shows when rolled up
)

// windowAnim is one popup's motion, by key; img keeps its last frame for the closing.
type windowAnim struct {
	openedAt, closedAt float64
	closing, seen      bool
	img                *ebiten.Image
	at                 image.Point
	dim                uint8
}

// popup lays out a popup window as inFrame does, to be shown over the screen dimmed to alpha
// dim at the end of the frame (showWindows). key names the window; a new key opens a new one.
func (g *Game) popup(dst *ebiten.Image, key string, dim uint8, w, h int, box image.Rectangle, draw func(dst *ebiten.Image)) {
	if g.windows == nil {
		g.windows = map[string]*windowAnim{}
	}
	a := g.windows[key]
	if a == nil || a.closing {
		a = &windowAnim{openedAt: g.uiSeconds}
		g.windows[key] = a
	}
	a.seen, a.dim = true, dim
	_, a.at = g.renderFrame(dst, w, h, box, 0, &a.img, draw)
	g.shown = append(g.shown, a)
}

// showWindows draws the popup windows once at the end of each frame: first those closing
// (every window not laid out this frame starts to close), then the open ones in the order
// they were laid out.
func (g *Game) showWindows(dst *ebiten.Image) {
	for key, a := range g.windows {
		if !a.seen && !a.closing {
			a.closing, a.closedAt = true, g.uiSeconds
		}
		if a.closing {
			if g.uiSeconds-a.closedAt >= windowClose {
				a.img.Deallocate()
				delete(g.windows, key)
				continue
			}
			a.drawOn(dst, g.uiSeconds)
		}
		a.seen = false
	}
	for _, a := range g.shown {
		a.drawOn(dst, g.uiSeconds)
	}
	g.shown = g.shown[:0]
}

// drawOn draws the dim and the window at time now.
func (a *windowAnim) drawOn(dst *ebiten.Image, now float64) {
	var open, dy, dimK float64
	if a.closing {
		p := phase(now-a.closedAt, 0, windowClose)
		open = 1 - bezier(.6, 0, .8, .4, p)
		dy = windowSinkBy * p * p
		dimK = 1 - phase(now-a.closedAt, 0, windowDimFade)
	} else {
		t := now - a.openedAt
		open = bezier(.3, .7, .2, 1, phase(t, 0, windowUnroll))
		dy = -windowDropBy * (1 - bezier(.34, 1.56, .64, 1, phase(t, 0, windowDrop)))
		dimK = phase(t, 0, windowDim)
	}
	if a.dim > 0 {
		rect(dst, 0, 0, float32(uiW), float32(uiH), color.NRGBA{A: uint8(float64(a.dim) * dimK)})
	}
	b := a.img.Bounds()
	mid := float64(b.Dy()) / 2
	half := mid * (windowSliver + (1-windowSliver)*open)
	top, bottom := int(mid-half), int(mid+half)
	if bottom <= top {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(a.at.X), float64(a.at.Y+top)+dy*uiPx)
	dst.DrawImage(a.img.SubImage(image.Rect(0, top, b.Dx(), bottom)).(*ebiten.Image), op)
}

// dropPopup takes popup key off the screen at once, without the closing roll that would show its last frame over
// whatever the click just changed.
func (g *Game) dropPopup(key string) {
	if a := g.windows[key]; a != nil {
		a.img.Deallocate()
		delete(g.windows, key)
	}
}
