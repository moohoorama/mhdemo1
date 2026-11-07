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
	sheetMax  = 1180
	sheetMaxH = 720
)

var (
	selBg    = color.NRGBA{R: 58, G: 42, B: 30, A: 255}
	cardBg   = color.NRGBA{R: 34, G: 24, B: 18, A: 255}
	cardRim  = color.NRGBA{R: 74, G: 56, B: 40, A: 255}
	faintInk = color.NRGBA{R: 125, G: 115, B: 102, A: 255}
	pillOut  = color.NRGBA{R: 74, G: 58, B: 28, A: 255}
	pillDep  = color.NRGBA{R: 35, G: 56, B: 74, A: 255}
	depRim   = color.NRGBA{R: 106, G: 154, B: 192, A: 255}
	depInk   = color.NRGBA{R: 207, G: 227, B: 242, A: 255}
)

// sheetWindows are the windows that fill the screen up to sheetMax×sheetMaxH, with no title plate and no 닫기 at the
// top: they close from their own action row.
var sheetWindows = map[string]bool{"party": true, "unit": true}

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

// wheelArea is a list the wheel scrolls while the pointer is over it.
type wheelArea struct {
	r image.Rectangle
	l *tableList
}

// updateWheelLists scrolls the list under the pointer by the wheel.
func (g *Game) updateWheelLists() bool {
	_, wheel := ebiten.Wheel()
	if wheel == 0 {
		return false
	}
	x, y := cursorPos()
	for _, a := range g.wheelAreas {
		if image.Pt(x, y).In(a.r) {
			a.l.top -= int(wheel)
			return true
		}
	}
	return false
}

// listRows draws the rows of a list n long in r, rowH each from l.top: draw paints row i in rr, a press on it
// that is let go without dragging picks it, and dragging or the wheel scrolls.
func (g *Game) listRows(dst *ebiten.Image, l *tableList, r image.Rectangle, n, rowH int, draw func(i int, rr image.Rectangle), pick func(i int)) {
	fit := max(1, r.Dy()/rowH)
	l.top = max(0, min(l.top, n-fit))
	for i := l.top; i < n && i < l.top+fit; i++ {
		rr := image.Rect(r.Min.X, r.Min.Y+(i-l.top)*rowH, r.Max.X, r.Min.Y+(i-l.top+1)*rowH)
		draw(i, rr)
		if pick != nil {
			i := i
			g.addButton(rr, "", func() { g.pressTable(l, rowH, func() { pick(i) }) })
		}
	}
	if n > fit {
		track := image.Rect(r.Max.X-4, r.Min.Y+2, r.Max.X-1, r.Min.Y+fit*rowH-2)
		th := max(24, track.Dy()*fit/n)
		ty := track.Min.Y + (track.Dy()-th)*l.top/(n-fit)
		rect(dst, float32(track.Min.X), float32(ty), 3, float32(th), color.NRGBA{R: 160, G: 130, B: 90, A: 200})
	}
	g.wheelAreas = append(g.wheelAreas, wheelArea{unframed(r), l})
}

// rowBack paints a list row's ground: picked rows lit with a gold bar on the left, and the line under every row.
func rowBack(dst *ebiten.Image, rr image.Rectangle, picked bool) {
	if picked {
		rect(dst, float32(rr.Min.X), float32(rr.Min.Y), float32(rr.Dx()), float32(rr.Dy()), selBg)
		rect(dst, float32(rr.Min.X), float32(rr.Min.Y), 4, float32(rr.Dy()), gold)
	}
	rect(dst, float32(rr.Min.X), float32(rr.Max.Y-1), float32(rr.Dx()), 1, cardRim)
}

// officerRow is an officer's list row: the face, the name over the class and level, and right a note.
func (g *Game) officerRow(dst *ebiten.Image, rr image.Rectangle, c core.Character, picked bool, note string, noteC color.Color) {
	rowBack(dst, rr, picked)
	x, y := float64(rr.Min.X+12), float64(rr.Min.Y)
	g.portrait(dst, c.ID, c.Class, x, y+8, 40)
	g.label(dst, g.name(c.ID), x+50, y+7, 16, ink)
	g.label(dst, fmt.Sprintf("%s · Lv%d", g.className(c.Class), c.Level), x+50, y+31, 12, muted)
	if note != "" {
		g.label(dst, note, float64(rr.Max.X-14)-g.width(note, 13), y+19, 13, noteC)
	}
}

// card paints a framed panel.
func card(dst *ebiten.Image, r image.Rectangle) {
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), cardBg)
	strokeRect(dst, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, cardRim)
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

// traitCard describes trait id in r: its icon, name, kind and where it comes from, what it does, and a note.
func (g *Game) traitCard(dst *ebiten.Image, r image.Rectangle, id, from, note string) {
	card(dst, r)
	x, y := float64(r.Min.X+12), float64(r.Min.Y+10)
	g.traitIconOf(dst, id, x, y+2, 32)
	name := g.traitName(id)
	g.label(dst, name, x+44, y-2, 18, gold)
	g.label(dst, g.traitKindName(id)+" · "+from, x+52+g.width(name, 18), y+4, 12, muted)
	desc := wrapWidth(g.traitDesc(id), g.face(14), float64(r.Dx()-68))
	g.label(dst, desc, x+44, y+26, 14, ink)
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
	corner := func(i int, k float64) (float32, float32) {
		t := -math.Pi/2 + float64(i)*math.Pi/3
		return float32(cx + radius*k*math.Cos(t)), float32(cy + radius*k*math.Sin(t))
	}
	ring := func(k float64) *vector.Path {
		var p vector.Path
		for i := range hexAxes {
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
	for _, k := range []float64{1, 2.0 / 3, 1.0 / 3} {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(goldDim)
		strokePath(dst, ring(k), vector.StrokeOptions{Width: 1}, op)
	}
	shape := func(s core.Stats, c color.NRGBA, fill bool) {
		var p vector.Path
		for i, a := range hexAxes {
			px, py := corner(i, max(.04, min(1, float64(a.get(s))/float64(max(1, best[i])))))
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
	if alone != nil && *alone != cur {
		shape(*alone, gold, false)
	}
	shape(cur, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, true)
	for i, a := range hexAxes {
		s := fmt.Sprintf("%s %d", a.stat, a.get(cur))
		ex, ey := corner(i, 1.14)
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
