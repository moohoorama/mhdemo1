package gui

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The command menu panels (docs/ui-mockup, idea 6): a lacquered list that unrolls from
// the top with its rows sliding in, a gold bar that fills the hovered row from the left,
// small icons on the main commands and, on a sub-list, a header and a description strip.
const (
	panelOpen  = .22 // the panel unrolls
	rowSlide   = .22 // each row slides in
	rowStagger = .04
	rowSweep   = .18 // the hover bar fills
	rowHeight  = 30
	panelPad   = 5
)

var (
	panelTop    = color.NRGBA{R: 43, G: 29, B: 20, A: 250}
	panelBottom = color.NRGBA{R: 27, G: 18, B: 13, A: 250}
	rowOn       = color.NRGBA{R: 255, G: 246, B: 221, A: 255}
	rowOff      = color.NRGBA{R: 119, G: 105, B: 90, A: 255}
)

// panelAnim remembers when a panel opened and which row the pointer entered when.
type panelAnim struct {
	key      string
	openedAt float64
	hot      int
	hotAt    float64
}

func (a *panelAnim) touch(key string, now float64) {
	if a.key != key {
		*a = panelAnim{key: key, openedAt: now, hot: -1}
	}
}

type panelSpec struct {
	items      []entry
	header     string  // gold header line; empty for none
	gauge      float64 // MP fraction shown in the header, < 0 for none
	descLines  int     // reserved lines of the description strip; 0 for none
	x, y, w    int
	anim       *panelAnim
	hoveredOut *string // receives the hovered row's help when there is no strip
	tipOut     *string // receives the hovered row's tip
	held       int     // row kept lit while its sub-list is open, -1 for none
}

func (g *Game) panelHeight(p panelSpec) int {
	h := panelPad*2 + len(p.items)*rowHeight
	if p.header != "" {
		h += 24
	}
	if p.descLines > 0 {
		h += 10 + p.descLines*17
	}
	return h
}

// panelBounds is where command panels must stay: on screen below the top bar, and left of
// the HUD panel.
func (g *Game) panelBounds() image.Rectangle {
	b := uiBounds(8)
	if !g.hudRect.Empty() {
		b.Max.X = g.hudRect.Min.X - 8
	}
	return b
}

// drawPanel draws a command panel and registers its rows as buttons.
func (g *Game) drawPanel(dst *ebiten.Image, p panelSpec) image.Rectangle {
	now := g.uiSeconds
	t := now - p.anim.openedAt
	h := g.panelHeight(p)
	bound := g.panelBounds()
	x := max(bound.Min.X, min(bound.Max.X-p.w, p.x))
	y := max(bound.Min.Y, min(bound.Max.Y-h, p.y))
	r := image.Rect(x, y, x+p.w, y+h)
	g.block(r)
	open := bezier(.2, .8, .3, 1, phase(t, 0, panelOpen))
	clip := image.Rect(int(float64(x-4)*drawPx), int(float64(y-2)*drawPx), int(float64(x+p.w+8)*drawPx), int((float64(y)+float64(h)*open+8)*drawPx))
	cd := dst.SubImage(clip).(*ebiten.Image)
	fx, fy, fw, fh := float32(x), float32(y), float32(p.w), float32(h)
	rect(cd, fx+3, fy+5, fw, fh, color.NRGBA{A: 110})
	for i := float32(0); i < 8; i++ {
		k := float64(i) / 7
		c := color.NRGBA{R: uint8(float64(panelTop.R)*(1-k) + float64(panelBottom.R)*k), G: uint8(float64(panelTop.G)*(1-k) + float64(panelBottom.G)*k), B: uint8(float64(panelTop.B)*(1-k) + float64(panelBottom.B)*k), A: 250}
		rect(cd, fx, fy+fh*i/8, fw, fh/8+1, c)
	}
	strokeRect(cd, fx+.5, fy+.5, fw-1, fh-1, 1, color.NRGBA{R: 11, G: 7, B: 5, A: 255})
	strokeRect(cd, fx+1.5, fy+1.5, fw-3, fh-3, 1, goldDim)

	slide := func(i int) (float64, uint8) {
		k := bezier(.2, .8, .2, 1, phase(t, .04+float64(i)*rowStagger, rowSlide))
		return -10 * (1 - k), uint8(255 * k)
	}
	top := y + panelPad
	n := 0
	if p.header != "" {
		dx, a := slide(n)
		n++
		c := gold
		c.A = a
		g.label(cd, p.header, float64(x+11)+dx, float64(top+2), 12, c)
		if p.gauge >= 0 {
			gx, gw := float32(x)+fw-82+float32(dx), float32(68)
			rect(cd, gx, float32(top+8), gw, 6, color.NRGBA{A: a})
			rect(cd, gx+1, float32(top+9), gw-2, 4, color.NRGBA{R: 16, G: 23, B: 34, A: a})
			fill := color.NRGBA{R: 93, G: 154, B: 216, A: a}
			if p.gauge > 1 {
				fill = mpColor(2, 1, g.uiSeconds)
				fill.A = a
			}
			rect(cd, gx+1, float32(top+9), (gw-2)*float32(min(1, p.gauge)), 4, fill)
		}
		rect(cd, float32(x+6), float32(top+21), fw-12, 1, color.NRGBA{R: 61, G: 44, B: 31, A: a})
		top += 24
	}
	help := ""
	for i, it := range p.items {
		b := image.Rect(x+panelPad, top+i*rowHeight, x+p.w-panelPad, top+i*rowHeight+rowHeight-2)
		hot := it.click != nil && hovered(b)
		if i == p.held && !hot {
			hot = true
			if p.anim.hot != i {
				p.anim.hot, p.anim.hotAt = i, now-rowSweep
			}
		}
		if hovered(b) {
			help = it.help
			if p.tipOut != nil {
				*p.tipOut = it.tip
			}
		}
		if hot && p.anim.hot != i {
			p.anim.hot, p.anim.hotAt = i, now
		} else if !hot && p.anim.hot == i && i != p.held {
			p.anim.hot = -1
		}
		if it.click != nil {
			g.addButton(b, it.label, it.click)
		} else {
			g.buttons = append(g.buttons, button{rect: unframed(b), label: it.label, click: func() {}, denied: true})
		}
		dx, a := slide(n)
		n++
		g.drawRow(cd, b, it, dx, a, hot, now-p.anim.hotAt)
	}
	if p.descLines > 0 {
		dy := top + len(p.items)*rowHeight + 2
		dx, a := slide(n)
		rect(cd, float32(x+6), float32(dy), fw-12, 1, color.NRGBA{R: 61, G: 44, B: 31, A: a})
		if help == "" && len(p.items) > 0 {
			help = p.items[0].help
		}
		c := muted
		c.A = a
		g.label(cd, clampLines(wrap(help, 17), p.descLines), float64(x+11)+dx, float64(dy+6), 12, c)
	} else if p.hoveredOut != nil && help != "" {
		*p.hoveredOut = help
	}
	return r
}

func clampLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// drawRow is one command row: hover bar, icon, label and the note on the right.
func (g *Game) drawRow(dst *ebiten.Image, b image.Rectangle, it entry, dx float64, a uint8, hot bool, hotFor float64) {
	x, y, w, h := float32(b.Min.X)+float32(dx), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy())
	if hot {
		k := float32(bezier(0, 0, .58, 1, min(1, hotFor/rowSweep)))
		for i := float32(0); i < 12; i++ {
			s := i / 12
			c := color.NRGBA{R: 214, G: 169, B: 87, A: uint8((97 - 82*s) * float32(a) / 255)}
			rect(dst, x+w*k*s, y, w*k/12+1, h, c)
		}
		rect(dst, x, y+h*.15, 3, h*.7, color.NRGBA{R: gold.R, G: gold.G, B: gold.B, A: a})
	}
	fg := ink
	switch {
	case it.click == nil:
		fg = rowOff
	case hot:
		fg = rowOn
	}
	fg.A = a
	lx := float64(x) + 9
	if it.icon != "" {
		g.drawIcon(dst, it.icon, float32(lx), y+h/2-6, 12, fg)
		lx += 19
	}
	g.label(dst, it.label, lx, float64(y)+(float64(h)-15*1.45)/2, 15, fg)
	if it.note != "" {
		nc := muted
		if it.click == nil {
			nc = rowOff
		}
		nc.A = a
		g.label(dst, it.note, float64(x+w)-9-g.width(it.note, 12), float64(y)+(float64(h)-12*1.45)/2, 12, nc)
	}
}

// menuIcons are the command menu's icons in the UI sheet's order.
var menuIcons = map[string]int{"attack": 0, "skill": 1, "item": 2, "wait": 3}

// drawIcon draws a 12-unit icon at (x, y) sized s.
func (g *Game) drawIcon(dst *ebiten.Image, name string, x, y, s float32, c color.NRGBA) {
	if i, ok := menuIcons[name]; ok {
		g.drawUI(dst, "icon", i, float64(x), float64(y), float64(s)/12, c)
	}
}
