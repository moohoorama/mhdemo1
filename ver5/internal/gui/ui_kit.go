package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

// The officer, unit and trait screens are laid out for a 914×411 phone (UI px as dp): whatever is pressed is at
// least touchH high, text is 14 (names 16, the smallest 12), and the buttons of what to do sit in one row at the
// bottom right, the main one last (docs/plan-party-ui.md).
const (
	touchH    = 48
	listRowH  = 56
	chipH     = 40
	sheetMax  = 960
	sheetMaxH = 720
)

var (
	selBg    = color.NRGBA{R: 58, G: 42, B: 30, A: 255}
	cardBg   = color.NRGBA{R: 60, G: 36, B: 28, A: 255}
	cardRim  = color.NRGBA{R: 74, G: 56, B: 40, A: 255}
	faintInk = color.NRGBA{R: 125, G: 115, B: 102, A: 255}
	pillOut  = color.NRGBA{R: 74, G: 58, B: 28, A: 255}
	pillDep  = color.NRGBA{R: 35, G: 56, B: 74, A: 255}
	depRim   = color.NRGBA{R: 106, G: 154, B: 192, A: 255}
	depInk   = color.NRGBA{R: 207, G: 227, B: 242, A: 255}
)

// sheetWindows are the windows that fill the screen up to sheetMax×sheetMaxH, with no title plate and no 닫기 at the
// top: they close from their own action row.
var sheetWindows = map[string]bool{"party": true, "unit": true, "deputy": true, "equipment": true}

func sheetSize() (int, int) { return min(uiW-16, sheetMax), min(uiH-16, sheetMaxH) }

// mBtn is a button of the phone layout: its label 15 and centred, gold when it is the main thing to do.
func (g *Game) mBtn(dst *ebiten.Image, r image.Rectangle, label string, main bool, click func()) {
	fg, _ := g.btnFace(dst, r, label, click)
	if main && click != nil {
		fg = gold
		strokeRect(dst, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx()-1), float32(r.Dy()-1), 1.5, gold)
	}
	g.label(dst, label, float64(r.Min.X)+(float64(r.Dx())-g.width(label, 15))/2, float64(r.Min.Y)+(float64(r.Dy())-15*1.45)/2, 15, fg)
}

// action is one button of an action row; a nil click draws it disabled.
type action struct {
	label string
	main  bool
	click func()
}

// actionRow draws the actions in a row touchH high ending at right, y, the last one rightmost, and gives the row's
// left edge.
func (g *Game) actionRow(dst *ebiten.Image, right, y int, acts ...action) int {
	x := right
	for i := len(acts) - 1; i >= 0; i-- {
		a := acts[i]
		w := max(96, int(g.width(a.label, 15))+36)
		g.mBtn(dst, image.Rect(x-w, y, x, y+touchH), a.label, a.main, a.click)
		x -= w + 8
	}
	return x + 8
}

// closeGap is the space between 닫기 and the actions to its left.
const closeGap = 24

// closeRow is a window's bottom row ending at right, y: 닫기 always at the far right, and left of it, a little
// apart, the window's actions, the main one nearest 닫기. It gives the row's left edge.
func (g *Game) closeRow(dst *ebiten.Image, right, y int, close func(), acts ...action) int {
	return g.endRow(dst, right, y, "닫기", close, acts...)
}

// endRow is closeRow with another word than 닫기 at the far right, 취소 where 닫기 could be read as going ahead.
func (g *Game) endRow(dst *ebiten.Image, right, y int, end string, close func(), acts ...action) int {
	g.mBtn(dst, image.Rect(right-96, y, right, y+touchH), end, false, close)
	if len(acts) == 0 {
		return right - 96
	}
	return g.actionRow(dst, right-96-closeGap, y, acts...)
}

// tabRow draws tabs touchH high from x, y; the one on is gold and underlined. It gives the right edge.
func (g *Game) tabRow(dst *ebiten.Image, x, y int, labels []string, on int, pick func(int)) int {
	for i, l := range labels {
		w := max(72, int(g.width(l, 15))+28)
		r := image.Rect(x, y, x+w, y+touchH)
		fg := muted
		if i == on {
			rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(w), touchH, color.NRGBA{R: 46, G: 32, B: 25, A: 255})
			rect(dst, float32(r.Min.X), float32(r.Max.Y-3), float32(w), 3, gold)
			fg = gold
		} else if pick != nil {
			i := i
			g.addButton(r, l, func() { pick(i) })
			if hovered(r) {
				rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(w), touchH, color.NRGBA{R: 255, G: 240, B: 210, A: 14})
			}
		}
		g.label(dst, l, float64(r.Min.X)+(float64(w)-g.width(l, 15))/2, float64(r.Min.Y)+(touchH-15*1.45)/2, 15, fg)
		x += w
	}
	return x
}

// wheelArea is a scroll area: a list the wheel and the press scroll while the pointer is over it, with its rows
// of rowH px. The buttons drawn from begin to end (indexes into g.buttons) belong to it: a press on one is held
// as a tap or a drag (pressButton).
type wheelArea struct {
	r        image.Rectangle
	l        *tableList
	rowH     int
	from, to int
}

// beginScroll opens the scroll area r of l. The returned index is for endScroll, which closes the buttons the area owns.
func (g *Game) beginScroll(r image.Rectangle, l *tableList, rowH int) int {
	g.wheelAreas = append(g.wheelAreas, wheelArea{r: unframed(r), l: l, rowH: rowH, from: len(g.buttons)})
	return len(g.wheelAreas) - 1
}

// pressArea holds a press at (x, y) on the innermost scroll area under it, where no button took the press: a drag
// scrolls it and a release picks nothing. It reports whether an area was under the press.
func (g *Game) pressArea(x, y int) bool {
	for k := len(g.wheelAreas) - 1; k >= 0; k-- {
		if a := g.wheelAreas[k]; image.Pt(x, y).In(a.r) {
			g.pressTable(a.l, a.rowH, nil)
			return true
		}
	}
	return false
}

// endScroll closes the area begun at i.
func (g *Game) endScroll(i int) { g.wheelAreas[i].to = len(g.buttons) }

// scrollAreaOf is the innermost scroll area that owns button i, nil for a button outside every area.
func (g *Game) scrollAreaOf(i int) *wheelArea {
	for k := len(g.wheelAreas) - 1; k >= 0; k-- {
		if a := &g.wheelAreas[k]; i >= a.from && i < a.to {
			return a
		}
	}
	return nil
}

// pressButton handles a press on button i: a button in a scroll area is held until the press is let go (a tap runs
// it, a drag scrolls), any other runs at once.
func (g *Game) pressButton(i int) {
	b := g.buttons[i]
	if a := g.scrollAreaOf(i); a != nil && !b.grab {
		g.pressTable(a.l, a.rowH, func() { g.runButton(b) })
		return
	}
	g.runButton(b)
}

// runButton is a button's click with its sound: the refusal of a denied row or the select sound.
func (g *Game) runButton(b button) {
	if b.denied {
		g.sfx("ui-denied")
	}
	before := g.sfxCalls
	b.click()
	g.sfxUnless(before, "ui-select")
}

// updateWheelLists scrolls the innermost list under the pointer by the wheel.
func (g *Game) updateWheelLists() bool {
	_, wheel := ebiten.Wheel()
	if wheel == 0 {
		return false
	}
	x, y := cursorPos()
	for k := len(g.wheelAreas) - 1; k >= 0; k-- {
		if a := g.wheelAreas[k]; image.Pt(x, y).In(a.r) {
			a.l.scroll(wheel)
			return true
		}
	}
	return false
}

// listRows draws the rows of a list n long in r, rowH each from l.top: draw paints row i in rr, a press on it
// that is let go without dragging picks it, and dragging, the wheel or the scrollbar on the right (shown when
// the rows do not all fit) scrolls. A button draw adds inside the row (a status pill) is registered after the
// row's own and so takes the press first.
func (g *Game) listRows(dst *ebiten.Image, l *tableList, r image.Rectangle, n, rowH int, draw func(i int, rr image.Rectangle), pick func(i int)) {
	fit := max(1, r.Dy()/rowH)
	l.top = max(0, min(l.top, n-fit))
	l.end = max(0, n-fit)
	right := r.Max.X
	if n > fit {
		right -= tableBarW + 4
	}
	area := g.beginScroll(r, l, rowH)
	for i := l.top; i < n && i < l.top+fit; i++ {
		rr := image.Rect(r.Min.X, r.Min.Y+(i-l.top)*rowH, right, r.Min.Y+(i-l.top+1)*rowH)
		if pick != nil {
			i := i
			g.addButton(rr, "", func() { pick(i) })
		}
		draw(i, rr)
	}
	if n > fit {
		g.tableBar(dst, l, image.Rect(r.Max.X-tableBarW, r.Min.Y+2, r.Max.X, r.Min.Y+fit*rowH-2), fit, n)
	}
	g.endScroll(area)
}

// rowBack paints a list row's ground: picked rows lit with a gold bar on the left, and the line under every row.
func rowBack(dst *ebiten.Image, rr image.Rectangle, picked bool) {
	if picked {
		rect(dst, float32(rr.Min.X), float32(rr.Min.Y), float32(rr.Dx()), float32(rr.Dy()), selBg)
		rect(dst, float32(rr.Min.X), float32(rr.Min.Y), 4, float32(rr.Dy()), gold)
	}
	rect(dst, float32(rr.Min.X), float32(rr.Max.Y-1), float32(rr.Dx()), 1, cardRim)
}

// depBand paints the ground of a row that serves or will serve as a deputy: blue, with a blue band on the left, or
// a gold one when the row is picked.
func depBand(dst *ebiten.Image, rr image.Rectangle, picked bool) {
	rect(dst, float32(rr.Min.X), float32(rr.Min.Y), float32(rr.Dx()), float32(rr.Dy()-1), pillDep)
	band := depRim
	if picked {
		band = gold
	}
	rect(dst, float32(rr.Min.X), float32(rr.Min.Y), 4, float32(rr.Dy()-1), band)
}

// depPlate is a small blue plate 20 high with s at 12 from x, y, the mark of a deputy; it gives its width.
func (g *Game) depPlate(dst *ebiten.Image, s string, x, y float64) float64 {
	w := g.width(s, 12) + 12
	rect(dst, float32(x), float32(y), float32(w), 20, pillDep)
	strokeRect(dst, float32(x)+.5, float32(y)+.5, float32(w-1), 19, 1, depRim)
	g.label(dst, s, x+6, y+(20-12*1.45)/2, 12, depInk)
	return w
}

// officerRow is an officer's list row in two lines: the face, the name and level over the class and a note on the
// right. A deputy's row is blue; beside the name a blue plate says whose deputy it is (unless the row's own pill
// says so, served) or which deputy the officer leads. It gives where the class ends, for what the row adds after it.
func (g *Game) officerRow(dst *ebiten.Image, o core.Observation, rr image.Rectangle, c core.Character, picked bool, note string, noteC color.Color) float64 {
	g.officerGround(dst, o, rr, c.ID, picked)
	return g.officerLines(dst, o, rr, c, false, note, noteC)
}

// officerGround paints an officer row's ground: lit with a gold band when picked, blue when the officer is a deputy.
func (g *Game) officerGround(dst *ebiten.Image, o core.Observation, rr image.Rectangle, id string, picked bool) {
	rowBack(dst, rr, picked)
	if g.servedBy(o, id) != "" {
		depBand(dst, rr, picked)
	}
}

// officerLines is officerRow without the ground; served leaves out whose deputy the officer is.
func (g *Game) officerLines(dst *ebiten.Image, o core.Observation, rr image.Rectangle, c core.Character, served bool, note string, noteC color.Color) float64 {
	lead := g.servedBy(o, c.ID)
	x, y := float64(rr.Min.X+12), float64(rr.Min.Y)
	right := float64(rr.Max.X - 12)
	g.portrait(dst, c.ID, c.Class, x, y+8, 40)
	name := g.name(c.ID)
	g.label(dst, name, x+50, y+6, 16, ink)
	lv := fmt.Sprintf("Lv%d", c.Level)
	g.label(dst, lv, right-g.width(lv, 12), y+10, 12, muted)
	plate := ""
	switch {
	case lead != "" && !served:
		plate = "부관 · " + g.name(lead)
	case c.Deputy != "":
		plate = "부관 " + g.name(c.Deputy)
	}
	if px := x + 58 + g.width(name, 16); plate != "" {
		if room := int(right - g.width(lv, 12) - 8 - px - 12); room > 24 {
			g.depPlate(dst, ellipsis(g, plate, 12, room), px, y+9)
		}
	}
	w := g.classRank(dst, c.Def, c.Class, c.Level, "", x+50, y+31, 12, muted)
	if note != "" {
		g.label(dst, note, right-g.width(note, 12), y+31, 12, noteC)
	}
	return x + 50 + w
}

// Small trait chips show what a deputy brings: 30 high on lines 36 apart, wrapping, and when they do not all fit,
// a last chip "+n" that shows them all (then "접기").
const (
	smallChipH     = 30
	smallChipPitch = 36
)

func (g *Game) smallChipW(name string) int {
	return int(math.Ceil(8 + 18 + 5 + g.width(name, 13) + 10))
}

// smallChipAt is a placed chip: its trait, or "" for the chip that shows more or fewer, and where, from y 0.
type smallChipAt struct {
	id, label string
	r         image.Rectangle
}

// layoutSmallChips places the chips of ids from x0 to right on at most lines lines, the "+n" chip last when some
// are left out; when all, every chip on as many lines as they take, then "접기" (when fold) if they take more than
// lines. It gives the chips and the lines used.
func (g *Game) layoutSmallChips(ids []string, x0, right, lines int, all, fold bool) ([]smallChipAt, int) {
	place := func(labels []string, keys []string) ([]smallChipAt, int) {
		out := []smallChipAt{}
		x, line := x0, 0
		for i, l := range labels {
			w := g.smallChipW(l)
			if keys[i] == "" {
				w -= 18 + 5
			}
			if x+w > right && x > x0 {
				x, line = x0, line+1
			}
			out = append(out, smallChipAt{keys[i], l, image.Rect(x, line*smallChipPitch, x+w, line*smallChipPitch+smallChipH)})
			x += w + 6
		}
		return out, line + 1
	}
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = g.traitName(id)
	}
	out, used := place(names, ids)
	if used <= lines {
		return out, used
	}
	if all {
		if fold {
			return place(append(names, "접기"), append(slices.Clone(ids), ""))
		}
		return out, used
	}
	for n := len(ids) - 1; n >= 0; n-- {
		more := fmt.Sprintf("+%d", len(ids)-n)
		out, used = place(append(slices.Clone(names[:n]), more), append(slices.Clone(ids[:n]), ""))
		if used <= lines {
			return out, used
		}
	}
	return out, used
}

// smallChips draws the chips laid out by layoutSmallChips from r's top left within r's width: pressing a trait
// calls press, pressing "+n" or "접기" calls toggle. It gives the height used and the chips where they were drawn.
func (g *Game) smallChips(dst *ebiten.Image, ids []string, r image.Rectangle, lines int, all bool, toggle func(), press func(id string)) (int, []smallChipAt) {
	chips, used := g.layoutSmallChips(ids, r.Min.X, r.Max.X, lines, all, toggle != nil)
	for i := range chips {
		chips[i].r = chips[i].r.Add(image.Pt(0, r.Min.Y))
		c := chips[i]
		cr := c.r
		rect(dst, float32(cr.Min.X), float32(cr.Min.Y), float32(cr.Dx()), smallChipH, color.NRGBA{R: 46, G: 32, B: 25, A: 255})
		hit := image.Rect(cr.Min.X, cr.Min.Y-(smallChipPitch-smallChipH)/2, cr.Max.X, cr.Max.Y+(smallChipPitch-smallChipH)/2)
		ty := float64(cr.Min.Y) + (smallChipH-13*1.45)/2
		if c.id == "" {
			strokeRect(dst, float32(cr.Min.X)+.5, float32(cr.Min.Y)+.5, float32(cr.Dx()-1), smallChipH-1, 1, muted)
			g.label(dst, c.label, float64(cr.Min.X+8), ty, 13, muted)
			if toggle != nil {
				g.addButton(hit, c.label, toggle)
			}
			continue
		}
		strokeRect(dst, float32(cr.Min.X)+.5, float32(cr.Min.Y)+.5, float32(cr.Dx()-1), smallChipH-1, 1, goldDim)
		g.traitIconOf(dst, c.id, float64(cr.Min.X+8), float64(cr.Min.Y)+(smallChipH-18)/2, 18)
		g.label(dst, c.label, float64(cr.Min.X+31), ty, 13, ink)
		if press != nil {
			id := c.id
			g.addButton(hit, c.label, func() { press(id) })
		}
	}
	return used*smallChipPitch - (smallChipPitch - smallChipH), chips
}

// card paints a framed panel: a drop shadow, the ruyi clouds at half contrast on a red-brown, a gold rim and
// inside it a dark line and a faint gold one.
func (g *Game) card(dst *ebiten.Image, r image.Rectangle) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	rect(dst, x, y+h, w, 3, color.NRGBA{A: 90})
	g.ruyiPanel(dst, float64(x), float64(y), float64(w), float64(h), cardBg, .5)
	strokeRect(dst, x+.5, y+.5, w-1, h-1, 1, goldDim)
	strokeRect(dst, x+2.5, y+2.5, w-5, h-5, 3, color.NRGBA{A: 70})
	strokeRect(dst, x+4.5, y+4.5, w-9, h-9, 1, color.NRGBA{R: 226, G: 182, B: 93, A: 46})
}

// dashedRect strokes r with a dashed line, for what can be learned.
func dashedRect(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	x0, y0, x1, y1 := float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X-1), float32(r.Max.Y-1)
	for x := x0; x < x1; x += 6 {
		w := min(3, x1-x)
		rect(dst, x, y0, w, 1, c)
		rect(dst, x, y1, w, 1, c)
	}
	for y := y0; y < y1; y += 6 {
		h := min(3, y1-y)
		rect(dst, x0, y, 1, h, c)
		rect(dst, x1, y, 1, h, c)
	}
}

// traitChipW is how wide a trait chip is: the icon, the name and, when not held, the cost.
func (g *Game) traitChipW(e traitEntry) int {
	w := 10 + traitIconSize + 6 + g.width(g.traitName(e.id), 14) + 12
	if !e.have {
		w += g.width(fmt.Sprint(e.cost), 12) + 6
	}
	return int(math.Ceil(w))
}

// traitChip draws a trait at x, y in a band touchH high, chipH of it seen: held ones solid gold rimmed, learnable
// ones (prerequisites met and points enough) green and dashed, the rest dark without a rim. It gives its width.
func (g *Game) traitChip(dst *ebiten.Image, x, y int, e traitEntry, picked bool, click func()) int {
	w := g.traitChipW(e)
	r := image.Rect(x, y+(touchH-chipH)/2, x+w, y+(touchH+chipH)/2)
	fg := faintInk
	switch {
	case e.have:
		rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(w), chipH, color.NRGBA{R: 46, G: 32, B: 25, A: 255})
		strokeRect(dst, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(w-1), chipH-1, 1, gold)
		fg = ink
	case e.affordable:
		dashedRect(dst, r, good)
		fg = good
	}
	if picked {
		strokeRect(dst, float32(r.Min.X)-3, float32(r.Min.Y)-3, float32(w+6), chipH+6, 2, color.NRGBA{R: 255, G: 248, B: 225, A: 230})
	}
	iy := float64(r.Min.Y) + (chipH-traitIconSize)/2
	if e.have || e.affordable {
		g.traitIconOf(dst, e.id, float64(x+10), iy, traitIconSize)
	} else {
		g.drawDimmedTraitIcon(dst, e.id, float64(x+10), iy, traitIconSize)
	}
	name := g.traitName(e.id)
	nx := float64(x + 10 + traitIconSize + 6)
	g.label(dst, name, nx, float64(r.Min.Y)+(chipH-14*1.45)/2, 14, fg)
	if !e.have {
		g.label(dst, fmt.Sprint(e.cost), nx+g.width(name, 14)+6, float64(r.Min.Y)+(chipH-12*1.45)/2+1, 12, fg)
	}
	if click != nil {
		g.addButton(image.Rect(x, y, x+w, y+touchH), name, click)
	}
	return w
}

// shelf is one row of the trait shelves: its head, under it where the traits come from, and the traits.
type shelf struct {
	head, sub string
	traits    []traitEntry
	empty     string
}

// traitShelves draws the shelves in r, the heads in a column on the left and the chips wrapping to the right,
// pick ringed; pressing a chip calls choose. It gives the height used.
func (g *Game) traitShelves(dst *ebiten.Image, r image.Rectangle, shelves []shelf, pick string, choose func(id string)) int {
	const headW = 84
	y := r.Min.Y
	for _, s := range shelves {
		g.label(dst, s.head, float64(r.Min.X), float64(y+8), 13, muted)
		if s.sub != "" {
			g.label(dst, ellipsis(g, s.sub, 12, headW-6), float64(r.Min.X), float64(y+27), 12, faintInk)
		}
		x := r.Min.X + headW
		if len(s.traits) == 0 {
			g.label(dst, s.empty, float64(x), float64(y+8), 12, faintInk)
			y += 36
			continue
		}
		for _, e := range s.traits {
			if w := g.traitChipW(e); x+w > r.Max.X && x > r.Min.X+headW {
				x, y = r.Min.X+headW, y+touchH
			}
			id := e.id
			x += g.traitChip(dst, x, y, e, e.id == pick, func() { choose(id) }) + 6
		}
		y += touchH
	}
	return y - r.Min.Y
}

// ellipsis cuts s to fit w at size, ending in "…".
func ellipsis(g *Game, s string, size float64, w int) string {
	if g.width(s, size) <= float64(w) {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && g.width(string(r)+"…", size) > float64(w) {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// traitCardBelow draws traitCard for trait id under the used height of r, as tall as its description and note
// need, raised to stay inside r.
func (g *Game) traitCardBelow(dst *ebiten.Image, r image.Rectangle, used int, id, from, note string) {
	h := g.traitCardHeight(id, note, r.Dx())
	y := min(r.Min.Y+used+6, r.Max.Y-h)
	g.traitCard(dst, image.Rect(r.Min.X, y, r.Max.X, y+h), id, from, note)
}

// traitCardHeight is the height traitCard needs at width w: the head, every line of the description and the note.
func (g *Game) traitCardHeight(id, note string, w int) int {
	_, h := measureLines(g, g.traitCardDesc(id, w), 14)
	h += 36 + 10
	if note != "" {
		h += 20
	}
	return max(84, int(math.Ceil(h)))
}

func (g *Game) traitCardDesc(id string, w int) string {
	return wrapWidth(g.traitDesc(id), g.face(14), float64(w-68))
}

// traitCard describes trait id in r: its icon, name, kind and where it comes from, what it does, and a note.
func (g *Game) traitCard(dst *ebiten.Image, r image.Rectangle, id, from, note string) {
	g.card(dst, r)
	x, y := float64(r.Min.X+12), float64(r.Min.Y+10)
	g.traitIconOf(dst, id, x, y+2, 32)
	name := g.traitName(id)
	g.label(dst, name, x+44, y-2, 18, gold)
	g.label(dst, g.traitKindName(id)+" · "+from, x+52+g.width(name, 18), y+4, 12, muted)
	g.label(dst, g.traitCardDesc(id, r.Dx()), x+44, y+26, 14, ink)
	if note != "" {
		g.label(dst, note, x+44, float64(r.Max.Y-24), 12, muted)
	}
}

// shelvesOf sorts the trait catalog of an officer into the shelves 고유, 병종 and 장비: the equipment shelf holds
// what the pieces worn give or teach even when the same trait is on another shelf too.
func (g *Game) shelvesOf(of *core.Character) []shelf {
	equip := []string{}
	for _, slot := range equipSlots {
		if id, ok := of.Equipment[slot]; ok {
			equip = append(equip, id)
		}
	}
	have := slices.Clone(of.Learned)
	for _, id := range equip {
		have = append(have, g.S.Data.Equipment[id].Traits...)
	}
	rows := g.traitCatalog(of.Def, of.Class, equip, have)
	if e := g.S.Engine; e != nil {
		learnable := e.Learnable(of.ID)
		for i := range rows {
			rows[i].learnable = !rows[i].have && slices.Contains(learnable, rows[i].id)
			rows[i].affordable = rows[i].learnable && rows[i].cost <= of.Points
		}
	}
	byID := map[string]traitEntry{}
	out := []shelf{{head: "고유", sub: g.name(of.ID)}, {head: "병종", sub: g.className(of.Class)}, {head: "장비", empty: "장비에서 오는 특성이 없습니다"}}
	for _, e := range rows {
		byID[e.id] = e
		switch e.group {
		case "고유":
			out[0].traits = append(out[0].traits, e)
		case "병종":
			out[1].traits = append(out[1].traits, e)
		}
	}
	names := []string{}
	for _, id := range equip {
		eq := g.S.Data.Equipment[id]
		names = append(names, eq.Name)
		for _, t := range append(slices.Clone(eq.Traits), eq.Learn...) {
			if e, ok := byID[t]; ok && !slices.ContainsFunc(out[2].traits, func(x traitEntry) bool { return x.id == t }) {
				out[2].traits = append(out[2].traits, e)
			}
		}
	}
	out[2].sub = joinNames(names)
	return out
}

func joinNames(s []string) string {
	out := ""
	for i, n := range s {
		if i > 0 {
			out += " · "
		}
		out += n
	}
	return out
}

// hexChart is the six battle stats on a hexagon at cx, cy: rim at the best of the roster, cur in white and, when
// given, alone (the lead without the deputy) dashed in gold. Labels are 12.
func (g *Game) hexChart(dst *ebiten.Image, cx, cy, radius float64, cur core.Stats, alone *core.Stats, best [6]int) {
	var names [6]string
	var c, al [6]int
	for i, a := range hexAxes {
		names[i], c[i] = a.stat, a.get(cur)
		if alone != nil {
			al[i] = a.get(*alone)
		}
	}
	var ap *[6]int
	if alone != nil && *alone != cur {
		ap = &al
	}
	g.hexagon(dst, cx, cy, radius, names, c, ap, best, []float64{1, 2.0 / 3, 1.0 / 3}, nil)
}

// hexChartAttrs is hexChart of the officer alone with, under each battle stat, the attribute that feeds it in gold
// (공격 over 무력, 방어 over 통솔, …), the labels a size up.
func (g *Game) hexChartAttrs(dst *ebiten.Image, cx, cy, radius float64, cur core.Stats, attrs [6]int, best [6]int) {
	var names [6]string
	var c [6]int
	var subs hexSubs
	for i, a := range hexAxes {
		names[i], c[i] = a.stat, a.get(cur)
		subs.text[i], subs.c[i] = fmt.Sprintf("%s %d", a.attr, attrs[a.attrIdx]), gold
	}
	g.hexagon(dst, cx, cy, radius, names, c, nil, best, []float64{1, 2.0 / 3, 1.0 / 3}, &subs)
}

// hexSubs are the second lines of a hexagon's labels and their colours.
type hexSubs struct {
	text [6]string
	c    [6]color.Color
}

// hexagon draws six values clockwise from the top on a hexagon at cx, cy, each against its rim value, with guide
// rings at the given shares of the rim: cur filled in white and, when given, other outlined in gold. Each corner is
// labelled with its name and value at 12, or, given subs, at 14 over its sub line at 13.
func (g *Game) hexagon(dst *ebiten.Image, cx, cy, radius float64, names [6]string, cur [6]int, other *[6]int, rim [6]int, rings []float64, subs *hexSubs) {
	corner := func(i int, k float64) (float32, float32) {
		t := -math.Pi/2 + float64(i)*math.Pi/3
		return float32(cx + radius*k*math.Cos(t)), float32(cy + radius*k*math.Sin(t))
	}
	ring := func(k float64) *vector.Path {
		var p vector.Path
		for i := range 6 {
			px, py := corner(i, k)
			if i == 0 {
				p.MoveTo(px, py)
			} else {
				p.LineTo(px, py)
			}
		}
		p.Close()
		return &p
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.NRGBA{R: 18, G: 14, B: 12, A: 200})
	fillPath(dst, ring(1), op)
	for _, k := range rings {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(goldDim)
		strokePath(dst, ring(k), vector.StrokeOptions{Width: 1}, op)
	}
	shape := func(v [6]int, c color.NRGBA, fill bool) {
		var p vector.Path
		for i := range 6 {
			px, py := corner(i, max(.04, min(1, float64(v[i])/float64(max(1, rim[i])))))
			if i == 0 {
				p.MoveTo(px, py)
			} else {
				p.LineTo(px, py)
			}
		}
		p.Close()
		if fill {
			f := &vector.DrawPathOptions{AntiAlias: true}
			f.ColorScale.ScaleWithColor(color.NRGBA{R: c.R, G: c.G, B: c.B, A: 35})
			fillPath(dst, &p, f)
		}
		e := &vector.DrawPathOptions{AntiAlias: true}
		e.ColorScale.ScaleWithColor(c)
		strokePath(dst, &p, vector.StrokeOptions{Width: 1.6, LineJoin: vector.LineJoinRound}, e)
	}
	if other != nil {
		shape(*other, gold, false)
	}
	shape(cur, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, true)
	for i := range 6 {
		s := fmt.Sprintf("%s %d", names[i], cur[i])
		ex, ey := corner(i, 1.14)
		if subs == nil {
			lx, ly := float64(ex), float64(ey)
			tw := g.width(s, 12)
			switch i {
			case 0:
				lx, ly = lx-tw/2, ly-16
			case 3:
				lx, ly = lx-tw/2, ly-2
			case 1, 2:
				ly -= 9
			default:
				lx, ly = lx-tw, ly-9
			}
			g.label(dst, s, lx, ly, 12, ink)
			continue
		}
		// two lines, 20 + 19 high: above the top corner, below the bottom one, beside the others
		sub := subs.text[i]
		at := func(t string, size float64) float64 {
			switch i {
			case 0, 3:
				return float64(ex) - g.width(t, size)/2
			case 1, 2:
				return float64(ex) + 2
			}
			return float64(ex) - 2 - g.width(t, size)
		}
		ly := float64(ey) - 20
		switch i {
		case 0:
			ly = float64(ey) - 40
		case 3:
			ly = float64(ey)
		}
		g.label(dst, s, at(s, 14), ly, 14, ink)
		g.label(dst, sub, at(sub, 13), ly+20, 13, subs.c[i])
	}
}

// rosterBest is the highest of each hexagon stat over the roster, the rim of hexChart.
func (g *Game) rosterBest(o core.Observation) [6]int {
	var best [6]int
	for _, c := range g.roster(o) {
		s, _, err := g.S.Engine.DeputyStats(c.ID, c.Deputy)
		if err != nil {
			continue
		}
		for i, a := range hexAxes {
			best[i] = max(best[i], a.get(s))
		}
	}
	return best
}
