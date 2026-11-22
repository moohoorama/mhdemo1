package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// The officer's detail in cards (docs/plan-party-ui.md): 열전 with the face card, the hexagon card of the battle
// stats over the attributes that feed them and the biography card; 특성 with a card for each shelf; 장비 with what
// the gear adds in all and a card for each slot.

// officerBio is the 열전 tab: the face card (class, level, move, range, spell points and their recovery,
// experience) and beside it, as tall, the hexagon card; the biography card fills what is left under them. On a
// short screen (a phone) the face is small with the class and move beside it, the hexagon smaller and the
// biography a size down, so that everything stays inside its card.
func (g *Game) officerBio(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) {
	stats, attrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return
	}
	const faceW, gap = 184, 12
	compact := r.Dy() < 400
	topH := min(290, r.Dy()-120)
	if compact {
		topH = r.Dy() - 96
	}
	face := image.Rect(r.Min.X, r.Min.Y, r.Min.X+faceW, r.Min.Y+topH)
	g.card(dst, face)
	x, y := float64(face.Min.X+16), float64(face.Min.Y+16)
	cl := g.S.Data.Classes[of.Class]
	reach := fmt.Sprint(cl.Range)
	if cl.MinRange > 1 {
		reach = fmt.Sprintf("%d~%d", cl.MinRange, cl.Range)
	}
	if compact {
		const size = 56
		g.portrait(dst, of.ID, "", x, y-2, size)
		g.classRank(dst, of.Def, of.Class, of.Level, fmt.Sprintf(" · Lv%d", of.Level), x+size+10, y-2, 13, muted)
		g.label(dst, fmt.Sprintf("이동 %d", cl.Move), x+size+10, y+17, 13, muted)
		g.label(dst, "사거리 "+reach, x+size+10, y+36, 13, muted)
		y += size + 8
	} else {
		size := float64(max(56, min(150, topH-140)))
		g.portrait(dst, of.ID, "", x, y, size)
		y += size + 12
		g.classRank(dst, of.Def, of.Class, of.Level, fmt.Sprintf(" · Lv%d", of.Level), x, y, 13, muted)
		g.label(dst, fmt.Sprintf("이동 %d · 사거리 %s", cl.Move, reach), x, y+22, 13, muted)
		y += 44
	}
	mp := fmt.Sprintf("병법치 %d", stats.MaxMP)
	g.label(dst, mp, x, y, 13, muted)
	g.label(dst, fmt.Sprintf("(+%d)", stats.Recovery), x+g.width(mp, 13)+5, y, 13, good)
	progress := core.ExperienceProgress(of.XP)
	g.label(dst, fmt.Sprintf("경험치 %.0f / 100", progress), x, y+22, 13, muted)
	bar(dst, float32(x), float32(y+44), float32(faceW-32), 8, progress/100, xpC)

	hex := image.Rect(face.Max.X+gap, r.Min.Y, r.Max.X, r.Min.Y+topH)
	g.card(dst, hex)
	g.statsCard(dst, hex, compact, stats, attrs, g.rosterBest(o))

	bio := image.Rect(r.Min.X, face.Max.Y+gap, r.Max.X, r.Max.Y)
	g.card(dst, bio)
	head, size, lead := 14.0, 15.0, 1.75
	if compact {
		head, size, lead = 10, 14, 1.5
	}
	g.label(dst, "열전", float64(bio.Min.X+18), float64(bio.Min.Y)+head, 13, gold)
	text := g.S.Data.Characters[of.Def].Biography
	if text == "" {
		text = "등록된 열전이 없습니다."
	}
	top := head + 24
	lines := int((float64(bio.Dy()) - top - 8) / (size * lead))
	g.labelSpaced(dst, clipLines(wrapWidth(text, g.face(size), float64(bio.Dx()-36)), lines), float64(bio.Min.X+18), float64(bio.Min.Y)+top, size, lead, ink)
}

// statBarStep is how far apart statBars' lines are.
const statBarStep = 26

// statBars draws hexAxes' six stats a line each from x, y, w wide: the stat's name, its value in valueC, a bar of
// its share from barX on and, when given, a tail in gold at the right end.
func (g *Game) statBars(dst *ebiten.Image, x, y, w, barX float64, values, tails [6]string, valueC color.Color, shares [6]float64) {
	tailW := 0.0
	for _, t := range tails {
		if t != "" {
			tailW = max(tailW, g.width(t, 13)+10)
		}
	}
	for i, a := range hexAxes {
		g.label(dst, a.stat, x, y, 13, ink)
		g.label(dst, values[i], x+44, y, 13, valueC)
		bar(dst, float32(x+barX), float32(y+6), float32(w-barX-tailW), 8, shares[i], gold)
		if tails[i] != "" {
			g.label(dst, tails[i], x+w-g.width(tails[i], 13), y, 13, gold)
		}
		y += statBarStep
	}
}

// statsCard fills the 열전 tab's card c with the battle stats over the attributes that feed them: a hexagon with its
// labels beside the corners, or, when c is too narrow for the labels to clear the hexagon, a bar a stat as in the
// 장비 tab's sum.
func (g *Game) statsCard(dst *ebiten.Image, c image.Rectangle, compact bool, stats core.Stats, attrs, best [6]int) {
	side := 0.0 // the widest label beside a corner: the stat over its attribute
	for i, a := range hexAxes {
		if i != 0 && i != 3 {
			side = max(side, g.width(fmt.Sprintf("%s %d", a.stat, a.get(stats)), 14), g.width(fmt.Sprintf("%s %d", a.attr, attrs[a.attrIdx]), 13))
		}
	}
	// a side label starts 2 past its corner, 1.14 radius out at 30°
	byWidth := (float64(c.Dx())/2 - 16 - 2 - side) / (1.14 * math.Cos(math.Pi/6))
	radius := min(84, float64(c.Dy()-120)/2, byWidth)
	cy := float64(c.Min.Y+c.Max.Y)/2 + 4
	if compact { // the labels, two lines above the top corner and below the bottom one, inside the card
		radius = min(84, (float64(c.Dy())-2*42-12)/2/1.14, byWidth)
		cy = float64(c.Min.Y+c.Max.Y) / 2
	}
	if radius >= 44 {
		g.hexChartAttrs(dst, float64(c.Min.X+c.Max.X)/2, cy, radius, stats, attrs, best)
		return
	}
	var values, tails [6]string
	var shares [6]float64
	valueW := 0.0
	for i, a := range hexAxes {
		v := a.get(stats)
		values[i], tails[i] = fmt.Sprint(v), fmt.Sprintf("%s %d", a.attr, attrs[a.attrIdx])
		shares[i] = float64(v) / float64(max(1, best[i]))
		valueW = max(valueW, g.width(values[i], 13))
	}
	const head = 30
	w := float64(c.Dx() - 36)
	h := head + 6*statBarStep - (statBarStep - 18)
	x, y := float64(c.Min.X+18), float64(c.Min.Y)+max(16, float64(c.Dy()-h)/2)
	g.label(dst, "능력", x, y, 13, gold)
	g.statBars(dst, x, y+head, w, 44+valueW+10, values, tails, ink, shares)
}

// labelSpaced draws s line by line, lead times the size apart.
func (g *Game) labelSpaced(dst *ebiten.Image, s string, x, y, size, lead float64, c color.Color) {
	for i, line := range strings.Split(s, "\n") {
		g.label(dst, line, x, y+float64(i)*size*lead, size, c)
	}
}

// shelfStep is how far one step of the wheel or of dragging moves the trait shelves.
const shelfStep = 24

// officerTraits is the 특성 tab: a card for each shelf (고유, 병종, 장비) over the picked trait's card, which stays
// at the bottom while the shelves scroll above it. It gives the action that learns the picked trait.
func (g *Game) officerTraits(dst *ebiten.Image, of *core.Character, r image.Rectangle) []action {
	p := &g.party
	g.label(dst, fmt.Sprintf("남은 특성치 %d", of.Points), float64(r.Min.X+8), float64(r.Min.Y+4), 15, gold)
	r.Min.Y += 30
	shelves := g.shelvesOf(of)
	all := []traitEntry{}
	for _, s := range shelves {
		all = append(all, s.traits...)
	}
	var picked *traitEntry
	if i := slices.IndexFunc(all, func(e traitEntry) bool { return e.id == p.trait }); i >= 0 {
		picked = &all[i]
	} else if i := slices.IndexFunc(all, func(e traitEntry) bool { return e.affordable }); i >= 0 {
		picked = &all[i]
	} else if len(all) > 0 {
		picked = &all[0]
	}
	note, cardH := "", 0
	if picked != nil {
		switch {
		case picked.have:
		case !picked.learnable && picked.requires != "":
			note = "선행 특성 " + picked.requires + "을 먼저 배워야 합니다"
		case !picked.affordable:
			note = fmt.Sprintf("특성치가 %d 모자랍니다 (%d 필요, %d 있음)", picked.cost-of.Points, picked.cost, of.Points)
		case !g.canLearnNow(of.ID):
			note = "탐험이나 출전 준비 중에 배울 수 있습니다"
		default:
			note = fmt.Sprintf("특성치 %d → %d", of.Points, of.Points-picked.cost)
		}
		cardH = g.traitCardHeight(picked.id, note, r.Dx()) + 10
	}
	pick := ""
	if picked != nil {
		pick = picked.id
	}
	g.shelfCards(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-cardH), shelves, pick, &p.shelves, func(id string) { p.trait = id })
	if picked == nil {
		return nil
	}
	g.traitCard(dst, image.Rect(r.Min.X, r.Max.Y-cardH+10, r.Max.X, r.Max.Y), picked.id, picked.group, note)
	if picked.have || !picked.learnable {
		return nil
	}
	var learn func()
	if picked.affordable && g.canLearnNow(of.ID) {
		id, trait := of.ID, picked.id
		learn = func() { g.command(core.Command{Kind: "learn", Actor: id, Trait: trait}) }
	}
	return []action{{fmt.Sprintf("%s 배우기 · %d", g.traitName(picked.id), picked.cost), true, learn}}
}

// shelfCards draws the shelves in r, a card each 10 apart: the head, where the traits come from and how many are
// learned on the left, the chips wrapping to the right, pick ringed. When the cards are taller than r they scroll
// by the wheel, by dragging or by the scrollbar on the right (l.top in shelfSteps), clipped to r.
func (g *Game) shelfCards(dst *ebiten.Image, r image.Rectangle, shelves []shelf, pick string, l *tableList, choose func(id string)) {
	const headW, pad, gap = 92, 10, 10
	right := r.Max.X - tableBarW - 4 // room for the scrollbar
	chipsX := r.Min.X + pad + headW
	lines := func(s shelf) int {
		n, x := 1, chipsX
		for _, e := range s.traits {
			if w := g.traitChipW(e); x+w > right-pad && x > chipsX {
				n, x = n+1, chipsX
			}
			x += g.traitChipW(e) + 6
		}
		return n
	}
	heights := make([]int, len(shelves))
	total := 0
	for i, s := range shelves {
		heights[i] = max(68, lines(s)*touchH+2*pad)
		total += heights[i] + gap
	}
	total -= gap
	steps := max(0, int(math.Ceil(float64(total-r.Dy())/shelfStep)))
	l.top = max(0, min(l.top, steps))
	if steps > 0 {
		g.addButton(r, "", func() { g.pressTable(l, shelfStep, nil) })
		g.wheelAreas = append(g.wheelAreas, wheelArea{unframed(r), l})
	} else {
		right = r.Max.X
	}
	clip := clipped(dst, r)
	g.buttonClip = r
	y := r.Min.Y - l.top*shelfStep
	for i, s := range shelves {
		c := image.Rect(r.Min.X, y, right, y+heights[i])
		if c.Max.Y > r.Min.Y && c.Min.Y < r.Max.Y {
			g.card(clip, c)
			hx, hy := float64(c.Min.X+pad+4), float64(c.Min.Y+pad+2)
			g.label(clip, s.head, hx, hy, 14, ink)
			if s.sub != "" {
				g.label(clip, ellipsis(g, s.sub, 12, headW-10), hx, hy+20, 12, muted)
			}
			learned := 0
			for _, e := range s.traits {
				if e.have {
					learned++
				}
			}
			if len(s.traits) > 0 {
				g.label(clip, fmt.Sprintf("%d / %d 배움", learned, len(s.traits)), hx, hy+38, 11, faintInk)
			}
			x, cy := chipsX, c.Min.Y+pad
			if len(s.traits) == 0 {
				g.label(clip, s.empty, float64(x), float64(cy+14), 12, faintInk)
			}
			for _, e := range s.traits {
				if w := g.traitChipW(e); x+w > c.Max.X-pad && x > chipsX {
					x, cy = chipsX, cy+touchH
				}
				id := e.id
				x += g.traitChip(clip, x, cy, e, e.id == pick, func() { g.pressTable(l, shelfStep, func() { choose(id) }) }) + 6
			}
		}
		y += heights[i] + gap
	}
	g.buttonClip = image.Rectangle{}
	if steps > 0 {
		fit := max(1, r.Dy()/shelfStep)
		g.tableBar(dst, l, image.Rect(r.Max.X-tableBarW, r.Min.Y, r.Max.X, r.Max.Y), fit, fit+steps)
	}
}

// clipped is dst cut to r (in UI px), so that what is drawn on it past r is dropped.
func clipped(dst *ebiten.Image, r image.Rectangle) *ebiten.Image {
	dev := image.Rect(int(math.Floor(float64(r.Min.X)*drawPx+drawOX)), int(math.Floor(float64(r.Min.Y)*drawPx+drawOY)),
		int(math.Ceil(float64(r.Max.X)*drawPx+drawOX)), int(math.Ceil(float64(r.Max.Y)*drawPx+drawOY)))
	return dst.SubImage(dev.Intersect(dst.Bounds())).(*ebiten.Image)
}

// officerEquipment is the 장비 tab: on the left a card of what the gear adds in all, each battle stat's grade with a
// bar, and the traits it gives; on the right a card for each slot with the piece, its grades, its traits and 바꾸기.
// On a short or narrow screen the slot cards take the whole width. Inside a slot card the icon shrinks with the
// card's width and the grades and traits wrap short of 바꾸기, as small chips when touch chips do not fit, and with
// 바꾸기 at the bottom when even those do not.
// Pressing a gear trait shows its description in a card under it until pressed again or elsewhere.
func (g *Game) officerEquipment(dst *ebiten.Image, of *core.Character, r image.Rectangle) {
	p := &g.party
	if p.tip != "" {
		g.addButton(r, "", func() { p.tip = "" })
	}
	type tipAt struct {
		id, from string
		chip     image.Rectangle
	}
	var tip *tipAt
	chips := func(dst *ebiten.Image, key, from string, ids []string, x, y, right int) int {
		x0 := x
		for _, id := range ids {
			e := traitEntry{id: id, have: true}
			if w := g.traitChipW(e); x+w > right && x > x0 {
				x, y = x0, y+touchH
			}
			k := key + "|" + id
			on := p.tip == k
			w := g.traitChip(dst, x, y, e, on, func() {
				if p.tip == k {
					p.tip = ""
				} else {
					p.tip = k
				}
			})
			if on {
				tip = &tipAt{id, from, image.Rect(x, y+(touchH-chipH)/2, x+w, y+(touchH+chipH)/2)}
			}
			x += w + 6
		}
		return y + touchH
	}

	const sumW, gap = 210, 12
	sum := image.Rect(r.Min.X, r.Min.Y, r.Min.X+sumW, r.Max.Y)
	var total [6]float64
	var gearTraits []string
	var gearFrom []string
	for _, slot := range equipSlots {
		if id, ok := of.Equipment[slot]; ok {
			eq := g.S.Data.Equipment[id]
			for i, v := range eq.Bonus {
				total[i] += v
			}
			for _, t := range eq.Traits {
				gearTraits, gearFrom = append(gearTraits, t), append(gearFrom, eq.Name)
			}
		}
	}
	// a short or narrow screen (a phone) leaves out the sum, which the slots' grades tell too, and gives the slots
	// the width
	rowH := min(128, (r.Dy()-2*10)/3)
	compact := rowH < 120 || r.Dx()-sumW-gap < 300
	rows := image.Rect(sum.Max.X+gap, r.Min.Y, r.Max.X, r.Max.Y)
	if compact {
		rows = r
	} else {
		g.card(dst, sum)
		x, y := float64(sum.Min.X+18), float64(sum.Min.Y+16)
		g.label(dst, "장비로 오른 능력", x, y, 13, gold)
		y += 30
		most := 5.0
		for _, v := range total {
			most = max(most, v)
		}
		var grades, none [6]string
		var shares [6]float64
		for i := range hexAxes {
			grades[i], shares[i] = "–", max(0, total[i])/most
			if total[i] != 0 {
				grades[i] = bonusGrade(total[i])
			}
		}
		g.statBars(dst, x, y, sumW-36, 76, grades, none, gold, shares)
		y += 6*statBarStep + 14
		g.label(dst, "장비 특성", x, y, 13, gold)
		y += 22
		if len(gearTraits) == 0 {
			g.label(dst, "없음", x, y, 12, faintInk)
		}
		for i, t := range gearTraits { // one chip a line keeps each one's source
			chips(dst, "sum", gearFrom[i], []string{t}, int(x)-4, int(y), sum.Max.X-12)
			y += touchH
		}
	}
	for i, slot := range equipSlots {
		c := image.Rect(rows.Min.X, rows.Min.Y+i*(rowH+10), rows.Max.X, rows.Min.Y+i*(rowH+10)+rowH)
		g.card(dst, c)
		// the icon a share of the card's width, so that a narrow card keeps room for the name and the chips
		iconW := max(40, min(64, c.Dx()*16/100, rowH-24))
		ix, iy := c.Min.X+14, c.Min.Y+(rowH-iconW)/2
		tx := ix + iconW + max(10, min(18, c.Dx()/24))
		id, worn := of.Equipment[slot]
		eq := g.S.Data.Equipment[id]
		name, nameC, icon := eq.Name, ink, eq.Icon
		if !worn {
			name, nameC, icon = "비어 있음", muted, slotIcons[slot]
		}
		g.itemIcon(dst, icon, float64(ix+3), float64(iy+3), float64(iconW-6)/32, !worn)
		var grades []gearPiece
		for k, v := range eq.Bonus {
			if v == 0 {
				continue
			}
			stat, grade := hexAxes[k].stat, bonusGrade(v)
			grades = append(grades, gearPiece{int(math.Ceil(g.width(stat, 12) + g.width(grade, 12) + 18)), 22, func(x, y int) {
				g.gradePill(dst, stat, grade, float64(x), float64(y))
			}})
		}
		if worn && len(grades) == 0 {
			grades = append(grades, gearPiece{int(math.Ceil(g.width("능력 변화 없음", 12))), 22, func(x, y int) {
				g.label(dst, "능력 변화 없음", float64(x), float64(y+2), 12, faintInk)
			}})
		}
		big, small := slices.Clone(grades), slices.Clone(grades)
		key, from := slot, eq.Name
		for _, t := range eq.Traits {
			big = append(big, gearPiece{g.traitChipW(traitEntry{id: t, have: true}), touchH, func(x, y int) {
				chips(dst, key, from, []string{t}, x, y, x)
			}})
			w := g.smallChipW(g.traitName(t))
			small = append(small, gearPiece{w, smallChipH, func(x, y int) {
				_, placed := g.smallChips(dst, []string{t}, image.Rect(x, y, x+w, y+smallChipH), 1, false, nil, func(id string) {
					if p.tip == key+"|"+id {
						p.tip = ""
					} else {
						p.tip = key + "|" + id
					}
				})
				if len(placed) > 0 && p.tip == key+"|"+t {
					tip = &tipAt{t, from, placed[0].r}
				}
			}})
		}
		// 바꾸기 beside the lines as long as they fit there, the traits as touch chips, else as small chips, else
		// with 바꾸기 at the bottom so that the lines above it take the card's whole width
		top := c.Min.Y + 12
		if rowH < 120 {
			top = c.Min.Y + 8
		}
		bx := c.Max.X - 14 - 96
		var btn image.Rectangle
		var at []image.Point
		var right func(y0, y1 int) int
		var pieces []gearPiece
		for _, try := range []struct {
			pieces []gearPiece
			btnY   int
		}{{big, (rowH - touchH) / 2}, {small, (rowH - touchH) / 2}, {small, rowH - 10 - touchH}} {
			btn = image.Rect(bx, c.Min.Y+try.btnY, bx+96, c.Min.Y+try.btnY+touchH)
			right = func(y0, y1 int) int {
				if y0 < btn.Max.Y+4 && y1 > btn.Min.Y-4 {
					return bx - 10
				}
				return c.Max.X - 14
			}
			pieces = try.pieces
			var bottom int
			var fits bool
			at, bottom, fits = gearLines(pieces, tx, top+28, right)
			if fits && bottom <= c.Max.Y-6 && g.width(name, 16) <= float64(right(top, top+22)-tx) {
				break
			}
		}
		g.mBtn(dst, btn, "바꾸기", false, func() {
			g.openOverlay("equipment")
			g.openPicker(slot)
		})
		if (g.overlay() == "party" || g.overlay() == "") && len(g.equipmentAdvice(g.S.View(), of.ID)[slot]) > 0 {
			recommendationRim(dst, btn.Inset(3))
		}
		nameW := right(top, top+22) - tx
		shown := ellipsis(g, name, 16, nameW)
		g.label(dst, shown, float64(tx), float64(top), 16, nameC)
		if kx := float64(tx) + g.width(shown, 16) + 8; kx+g.width(slotNames[slot], 12) <= float64(tx+nameW) {
			g.label(dst, slotNames[slot], kx, float64(top+4), 12, muted)
		}
		for k, pc := range pieces {
			pc.draw(at[k].X, at[k].Y)
		}
	}
	if tip != nil {
		note := tip.from + "에 딸린 특성 · 벗으면 사라집니다"
		w := 340
		h := g.traitCardHeight(tip.id, note, w)
		tx := min(tip.chip.Min.X, r.Max.X-w)
		ty := tip.chip.Max.Y + 6
		if ty+h > r.Max.Y+56 {
			ty = tip.chip.Min.Y - 6 - h
		}
		box := image.Rect(tx, ty, tx+w, ty+h)
		rect(dst, float32(box.Min.X+4), float32(box.Min.Y+6), float32(w), float32(h), color.NRGBA{A: 110})
		g.traitCard(dst, box, tip.id, "장비", note)
		strokeRect(dst, float32(box.Min.X)+.5, float32(box.Min.Y)+.5, float32(w-1), float32(h-1), 1, gold)
		g.addButton(box, "", func() {})
	}
}

// gradePill is a small plate with a stat's name and its grade in gold, from x, y; it gives its width.
func (g *Game) gradePill(dst *ebiten.Image, name, grade string, x, y float64) float64 {
	w := g.width(name, 12) + g.width(grade, 12) + 18
	rect(dst, float32(x), float32(y), float32(w), 22, color.NRGBA{A: 80})
	strokeRect(dst, float32(x)+.5, float32(y)+.5, float32(w-1), 21, 1, goldDim)
	g.label(dst, name, x+7, y+3, 12, ink)
	g.label(dst, grade, x+11+g.width(name, 12), y+3, 12, gold)
	return w
}

// gearPiece is one thing on a slot card's lines under the piece's name: a grade pill, the note of no grades or a
// trait chip, w by h from x, y.
type gearPiece struct {
	w, h int
	draw func(x, y int)
}

// gearLines flows the pieces from x0, y0 in lines 4 apart, each piece centred in its line's height, a line ending
// at right of its top and bottom (short of a button beside it). It gives where each piece goes, the bottom of the
// last line and whether every piece fits within its line.
func gearLines(pieces []gearPiece, x0, y0 int, right func(y0, y1 int) int) ([]image.Point, int, bool) {
	at := make([]image.Point, len(pieces))
	x, top, h, first, fits := x0, y0, 0, 0, true
	centre := func(end int) {
		for k := first; k < end; k++ {
			at[k].Y = top + (h-pieces[k].h)/2
		}
	}
	for i, pc := range pieces {
		if x > x0 && x+pc.w > right(top, top+max(h, pc.h)) {
			centre(i)
			x, top, h, first = x0, top+h+4, 0, i
		}
		if x+pc.w > right(top, top+max(h, pc.h)) {
			fits = false
		}
		at[i] = image.Pt(x, top)
		x, h = x+pc.w+6, max(h, pc.h)
	}
	centre(len(pieces))
	return at, top + h, fits
}

// officerDeputy is the 부관 tab: the candidates in a table (deputyTable), sorted by
// the weight they would get, and under it a card of the one picked — class, level and weight, what naming it does
// or why it cannot be, and every trait it would bring as a small chip ("+n" when they need more lines than the
// card gets). It gives the actions that dismiss the deputy and appoint the one picked.
func (g *Game) officerDeputy(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) []action {
	note := ""
	switch lead := g.servedBy(o, of.ID); {
	case lead != "":
		note = g.name(lead) + "의 부관입니다. 부관은 부관을 둘 수 없습니다."
	case o.Dialogue.Kind == "battle" && !slices.Contains(o.Deployment, of.ID):
		note = "출전하는 장수만 부관을 둘 수 있습니다."
	case !g.S.Engine.CanDeputize(of.ID):
		note = "지금은 부관을 둘 수 없습니다."
	}
	if note != "" {
		c := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+64)
		g.card(dst, c)
		g.label(dst, note, float64(c.Min.X+18), float64(c.Min.Y+22), 14, muted)
		return nil
	}
	p := &g.party
	if p.depLead != of.ID {
		p.depLead, p.depAll = of.ID, false
		g.deputyFresh()
	}
	rows := g.deputyRows(o)
	// the card of the one picked takes what the table leaves, its chips on the lines that fit in half of r unless
	// all are asked for; the pick comes from the table, so the card is first sized for the pick of the last frame
	var last *deputyRow
	if i := slices.IndexFunc(rows, func(d deputyRow) bool { return d.id == g.deputyShown(of) }); i >= 0 {
		last = &rows[i]
	}
	lines := func(d *deputyRow) int {
		return max(1, (r.Dy()/2-deputyCardTop(d)-8+smallChipPitch-smallChipH)/smallChipPitch)
	}
	cardH := g.deputyCardHeight(last, r.Dx(), lines(last), p.depAll)
	pick := g.deputyTable(dst, &p.deputies, of, rows, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-cardH-6))
	at := slices.IndexFunc(rows, func(d deputyRow) bool { return d.id == pick })
	if at < 0 {
		return nil
	}
	picked := &rows[at]
	if picked != last {
		cardH = g.deputyCardHeight(picked, r.Dx(), lines(picked), p.depAll)
	}
	g.deputyCard(dst, o, picked, image.Rect(r.Min.X, r.Max.Y-cardH, r.Max.X, r.Max.Y), lines(picked), p.depAll)

	var appoint, release func()
	if pick != of.Deputy && picked.reason == "" {
		lead, dep := of.ID, pick
		appoint = func() {
			if g.request(session.Request{Op: "command", Command: core.Command{Kind: "deputy", Actor: lead, Target: dep}}).OK {
				g.deputyPick = ""
			}
		}
	}
	if of.Deputy != "" {
		lead := of.ID
		release = func() {
			if g.request(session.Request{Op: "command", Command: core.Command{Kind: "undeputy", Actor: lead}}).OK {
				g.deputyPick = ""
			}
		}
	}
	label := picked.name + " 임명"
	if picked.current {
		label = "지금 부관"
	}
	return []action{{"부관 해임", false, release}, {label, true, appoint}}
}

// deputyShown is the candidate the table showed picked: the one previewed, else the deputy now.
func (g *Game) deputyShown(of *core.Character) string {
	if g.deputyPick != "" {
		return g.deputyPick
	}
	return of.Deputy
}

// deputyCardTop is the height of the picked candidate's card above its chips: the name line and, when there is
// one, the line of what naming it does or why it cannot be.
func deputyCardTop(d *deputyRow) int {
	if d != nil && (d.reason != "" || d.note != "") {
		return 8 + 24 + 20 + 4
	}
	return 8 + 24 + 4
}

// deputyCardHeight is how tall deputyCard is at width w for d, its chips on at most lines lines unless all.
func (g *Game) deputyCardHeight(d *deputyRow, w, lines int, all bool) int {
	if d == nil || len(d.traits) == 0 {
		return deputyCardTop(d) + 20 + 8
	}
	_, used := g.layoutSmallChips(d.traits, 16, w-16, lines, all, true)
	return deputyCardTop(d) + used*smallChipPitch - (smallChipPitch - smallChipH) + 8
}

// deputyCard is the candidate picked in the 부관 tab: name, class and level, the weight in gold, the traits it would
// bring counted on the right; what naming it does or why it cannot be; then the traits as small chips, each opening
// its description, "+n" showing the rest.
func (g *Game) deputyCard(dst *ebiten.Image, o core.Observation, d *deputyRow, r image.Rectangle, lines int, all bool) {
	p := &g.party
	g.card(dst, r)
	x, y := float64(r.Min.X+16), float64(r.Min.Y+8)
	g.label(dst, d.name, x, y, 16, ink)
	lx := x + g.width(d.name, 16) + 10
	if c := g.officer(o, d.id); c != nil {
		lx += g.classRank(dst, c.Def, c.Class, c.Level, fmt.Sprintf(" · Lv%d", c.Level), lx, y+4, 12, muted)
	}
	if d.reason == "" {
		ws := fmt.Sprintf("가중치 %.0f%%", d.weight*100)
		g.label(dst, ws, lx+12, y+3, 13, gold)
	}
	count := fmt.Sprintf("더해지는 특성 %d", len(d.traits))
	g.label(dst, count, float64(r.Max.X-16)-g.width(count, 12), y+4, 12, muted)
	y += 24
	switch {
	case d.reason != "":
		g.label(dst, ellipsis(g, d.reason, 12, r.Dx()-32), x, y, 12, faintInk)
		y += 20
	case d.note != "":
		g.label(dst, ellipsis(g, d.note, 12, r.Dx()-32), x, y, 12, color.NRGBA{R: 224, G: 160, B: 112, A: 255})
		y += 20
	}
	y += 4
	if len(d.traits) == 0 {
		g.label(dst, "더해지는 특성이 없습니다", x, y, 12, faintInk)
		return
	}
	g.smallChips(dst, d.traits, image.Rect(r.Min.X+16, int(y), r.Max.X-16, r.Max.Y), lines, all,
		func() { p.depAll = !p.depAll },
		func(id string) { g.deputyInfo = equipChip{"deputy", id} })
}
