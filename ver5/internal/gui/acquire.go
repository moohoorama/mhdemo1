package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
	"srpg/internal/core"
)

// The acquire window is what the party just got outside a battle's result (a story grant, an event's gift, a
// purchase): one card per piece, turned up one after another once, that stays until 확인. A card opens the piece's
// window; a trait on an equipment's window opens the trait's. Those stack on the cards and close one at a time
// without the cards moving again.
const (
	acqCardW, acqCardH, acqCardGap = 180, 268, 22
	acqCardTop                     = 262
	acqFirst, acqEach              = .2, .15 // seconds before the first card turns up, and between cards
	acqTurn                        = .4
	acqPieceW                      = 520
)

type gain struct {
	id    string
	n     int
	first bool // the party owned none before
}

// acquireGains are the grant events as cards, one per piece; first marks a piece the party did not own in before.
func acquireGains(before *core.Observation, events []core.Event) []gain {
	var out []gain
	for _, e := range events {
		if e.Kind != "grant" || e.Amount <= 0 {
			continue
		}
		merged := false
		for i := range out {
			if out[i].id == e.Target {
				out[i].n += e.Amount
				merged = true
			}
		}
		if !merged {
			out = append(out, gain{id: e.Target, n: e.Amount, first: e.Target != content.Money && !owned(before, e.Target)})
		}
	}
	return out
}

func owned(o *core.Observation, id string) bool {
	if o == nil {
		return true
	}
	if o.Inventory[id] > 0 || o.Warehouse[id] > 0 {
		return true
	}
	for _, c := range o.Characters {
		for _, worn := range c.Equipment {
			if worn == id {
				return true
			}
		}
	}
	return false
}

// drawAcquire draws the cards and whatever window is open over them; only 확인 closes the cards.
func (g *Game) drawAcquire(dst *ebiten.Image, m *modal) {
	full := image.Rect(0, 0, uiW, uiH)
	g.addButton(full, "", func() {})
	n := len(m.gains)
	w := n*acqCardW + (n-1)*acqCardGap
	box := image.Rect(Width/2-w/2-24, 190, Width/2+w/2+24, acqCardTop+acqCardH+110)
	g.popup(dst, fmt.Sprintf("modal:%p", m), 150, Width, Height, box, func(dst *ebiten.Image) { g.acquireCards(dst, m) })
	if m.piece == "" {
		return
	}
	g.addButton(full, "", g.closeModal)
	h := g.pieceHeight(m.piece)
	g.popup(dst, fmt.Sprintf("acquire:%p:%s", m, m.piece), 80, acqPieceW+40, h+40, image.Rect(0, 0, acqPieceW+40, h+40), func(dst *ebiten.Image) {
		g.pieceWindow(dst, m, image.Rect(20, 20, 20+acqPieceW, 20+h))
	})
	if m.chip.id != "" {
		g.chipWindow(dst, fmt.Sprintf("acquire:%p:%s:%s", m, m.piece, m.chip.id), m.chip, g.closeModal)
	}
}

// closeAcquire takes off the acquire window's top layer and reports whether there was one under the cards.
func (m *modal) closeAcquire() bool {
	switch {
	case m.chip.id != "":
		m.chip = equipChip{}
	case m.piece != "":
		m.piece = ""
	default:
		return false
	}
	return true
}

func (g *Game) acquireCards(dst *ebiten.Image, m *modal) {
	cx := float64(uiW / 2)
	g.centered(dst, "획  득", cx, 196, 34, gold)
	n := len(m.gains)
	x0 := uiW/2 - (n*acqCardW+(n-1)*acqCardGap)/2
	for i, gn := range m.gains {
		p := phase(m.t, acqFirst+acqEach*float64(i), acqTurn)
		if p <= 0 {
			continue
		}
		up := bezier(.3, 1.4, .5, 1, p)
		r := image.Rect(x0+i*(acqCardW+acqCardGap), acqCardTop, x0+i*(acqCardW+acqCardGap)+acqCardW, acqCardTop+acqCardH).Add(image.Pt(0, int(24*(1-up))))
		g.acquireCard(dst, r, gn, min(1, p*2), m)
	}
	done := phase(m.t, acqFirst+acqEach*float64(n-1)+acqTurn, .3)
	if done <= 0 {
		return
	}
	g.centered(dst, "패를 누르면 자세히 볼 수 있습니다", cx, acqCardTop+acqCardH+16, 13, fade(muted, done))
	g.btn(dst, image.Rect(uiW/2-80, acqCardTop+acqCardH+48, uiW/2+80, acqCardTop+acqCardH+86), "", g.closeModal)
	g.centered(dst, "확인", cx, acqCardTop+acqCardH+56, 15, ink)
}

// acquireCard is one card at alpha a: the picture, the name and count, the kind, and the equipment's traits or the
// item's use. A piece the party did not own before is marked 처음 and glows.
func (g *Game) acquireCard(dst *ebiten.Image, r image.Rectangle, gn gain, a float64, m *modal) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	open := gn.id != content.Money
	hover := open && hovered(r) && a >= 1
	rim := goldDim
	if gn.first || hover {
		rim = gold
	}
	if gn.first {
		for i := float32(1); i <= 6; i++ {
			strokeRect(dst, x-i*2, y-i*2, w+i*4, h+i*4, 2, fade(color.NRGBA{R: 226, G: 182, B: 93, A: uint8(70 - 10*i)}, a))
		}
	}
	body := lacquer
	if hover {
		body = lacquer2
	}
	body.A = 255 // the story's lines stay under the cards, not through them
	rect(dst, x, y, w, h, fade(body, a))
	strokeRect(dst, x+1, y+1, w-2, h-2, 2, fade(rim, a))
	cx := float64(r.Min.X + r.Dx()/2)
	if gn.first {
		tw := float32(g.width("처음", 12) + 16)
		rect(dst, float32(cx)-tw/2, y-10, tw, 20, fade(color.NRGBA{R: 108, G: 30, B: 26, A: 255}, a))
		strokeRect(dst, float32(cx)-tw/2, y-10, tw, 20, 1, fade(gold, a))
		g.centered(dst, "처음", cx, float64(y)-10, 12, fade(ink, a))
	}
	g.pieceIcon(dst, gn.id, cx-44, float64(y)+22, 88, a)
	g.centered(dst, g.pieceName(gn.id), cx, float64(y)+120, 20, fade(ink, a))
	count := fmt.Sprintf("×%d", gn.n)
	if gn.id == content.Money {
		count = fmt.Sprintf("+%d", gn.n)
	}
	g.centered(dst, count, cx, float64(y)+150, 15, fade(gold, a))
	g.centered(dst, g.pieceKind(gn.id), cx, float64(y)+174, 12, fade(muted, a))
	ty := float64(y) + 200
	if eq, ok := g.S.Data.Equipment[gn.id]; ok {
		for _, c := range itemChips(eq.Traits, eq.Learn, eq.Skills) {
			name := g.chipName(c)
			cw := traitIconSize*.8 + 6 + g.width(name, 13)
			kind, tone := g.chipLook(c)
			rect(dst, float32(cx-cw/2-6), float32(ty-3), float32(cw+12), 24, fade(trough, a))
			g.traitIcon(dst, kind, tone, cx-cw/2, ty, traitIconSize*.8)
			g.label(dst, name, cx-cw/2+traitIconSize*.8+6, ty, 13, fade(ink, a))
			ty += 28
		}
	} else if it, ok := g.S.Data.Items[gn.id]; ok {
		for _, line := range strings.Split(wrapWidth(it.Desc, g.face(12), acqCardW-24), "\n") {
			g.centered(dst, line, cx, ty, 12, fade(ink, a))
			ty += 18
		}
	}
	if open && a >= 1 {
		g.addButton(r, "", func() { m.piece = gn.id })
	}
}

// pieceIcon draws a piece's picture in an s-wide slot at (x, y): the equipment's icon, a coin for money, an empty
// slot for an item that has no picture yet.
func (g *Game) pieceIcon(dst *ebiten.Image, id string, x, y, s, a float64) {
	rect(dst, float32(x), float32(y), float32(s), float32(s), fade(trough, a))
	strokeRect(dst, float32(x)+.5, float32(y)+.5, float32(s)-1, float32(s)-1, 1, fade(goldDim, a))
	if id == content.Money {
		px, ox, oy := float32(drawPx), float32(drawOX), float32(drawOY)
		cx, cy, rad := float32(x+s/2)*px+ox, float32(y+s/2)*px+oy, float32(s*.26)*px
		vector.FillCircle(dst, cx, cy, rad, fade(gold, a), true)
		vector.StrokeCircle(dst, cx, cy, rad, 2*px, fade(goldDim, a), true)
		vector.FillRect(dst, cx-rad*.3, cy-rad*.3, rad*.6, rad*.6, fade(trough, a), false)
		return
	}
	if eq, ok := g.S.Data.Equipment[id]; ok {
		if _, ok := g.assets.UI["item_"+eq.Icon]; ok {
			g.drawUI(dst, "item_"+eq.Icon, 0, x+6, y+6, (s-12)/32, fade(white, a))
		}
	}
}

func (g *Game) pieceName(id string) string {
	if id == content.Money {
		return "돈"
	}
	return g.itemName(id)
}

// pieceKind is the card's small line: the equipment's slot, an item, or the money now held.
func (g *Game) pieceKind(id string) string {
	if id == content.Money {
		return fmt.Sprintf("소지금 %d", g.S.View().Money)
	}
	if eq, ok := g.S.Data.Equipment[id]; ok {
		return slotNames[eq.Slot]
	}
	return "도구"
}

func (g *Game) pieceHeight(id string) int {
	if _, ok := g.S.Data.Equipment[id]; ok {
		return 330
	}
	return 250
}

// pieceWindow is a card's window: an equipment's description, stat bonus and traits (each opening its own window),
// or an item's use and how many the party holds.
func (g *Game) pieceWindow(dst *ebiten.Image, m *modal, r image.Rectangle) {
	eq, isEq := g.S.Data.Equipment[m.piece]
	title := "도구"
	if isEq {
		title = "장비"
	}
	g.titled(dst, r, title)
	x, y := float64(r.Min.X+28), float64(r.Min.Y+30)
	g.pieceIcon(dst, m.piece, x, y, 72, 1)
	g.label(dst, g.pieceName(m.piece), x+90, y+4, 26, gold)
	g.label(dst, g.pieceKind(m.piece), x+90, y+44, 14, muted)
	y += 96
	row := func(name string) {
		g.label(dst, name, x, y, 15, muted)
	}
	if isEq {
		g.label(dst, wrapWidth(eq.Desc, g.face(15), float64(r.Dx()-56)), x, y, 15, ink)
		y += 44
		strokeLine(dst, float32(x), float32(y), float32(r.Max.X-28), float32(y), 1, goldDim)
		y += 14
		row("보정")
		g.label(dst, bonusText(eq.Bonus), x+90, y, 15, good)
		y += 36
		row("특성")
		cx := x + 90
		chips := itemChips(eq.Traits, eq.Learn, eq.Skills)
		for _, c := range chips {
			c := c
			cx = g.chip(dst, c, cx, y, func() { m.chip = c }) + 14
		}
		if len(chips) == 0 {
			g.label(dst, "없음", cx, y, 15, ink)
		}
	} else {
		it := g.S.Data.Items[m.piece]
		strokeLine(dst, float32(x), float32(y), float32(r.Max.X-28), float32(y), 1, goldDim)
		y += 14
		row("효과")
		g.label(dst, wrapWidth(it.Desc, g.face(15), float64(r.Dx()-146)), x+90, y, 15, good)
		y += 36
		row("가진 수")
		g.label(dst, fmt.Sprintf("%d개", g.S.View().Inventory[m.piece]), x+90, y, 15, ink)
	}
	g.centered(dst, "창 밖을 누르면 닫힙니다", float64(r.Min.X+r.Dx()/2), float64(r.Max.Y-34), 13, muted)
}

// bonusText is an equipment's stat bonus in the order of its six growth terms (hexAxes), "없음" when it has none.
func bonusText(b [6]float64) string {
	var parts []string
	for i, v := range b {
		if v == 0 {
			continue
		}
		n := fmt.Sprintf("%+g", v)
		if v == math.Trunc(v) {
			n = fmt.Sprintf("%+d", int(v))
		}
		parts = append(parts, hexAxes[i].stat+" "+n)
	}
	if len(parts) == 0 {
		return "없음"
	}
	return strings.Join(parts, " · ")
}

// fade is c with its alpha scaled by a.
func fade(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(float64(c.A) * max(0, min(1, a)))
	return c
}
