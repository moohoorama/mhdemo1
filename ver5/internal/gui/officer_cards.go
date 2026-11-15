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
	radius := max(40, min(84, float64(topH-120)/2, float64(hex.Dx()-220)/2))
	cy := float64(hex.Min.Y+hex.Max.Y)/2 + 4
	if compact { // the labels, two lines above the top corner and below the bottom one, inside the card
		radius = max(24, min(84, (float64(topH)-2*42-12)/2/1.14, float64(hex.Dx()-220)/2))
		cy = float64(hex.Min.Y+hex.Max.Y) / 2
	}
	g.hexChartAttrs(dst, float64(hex.Min.X+hex.Max.X)/2, cy, radius, stats, attrs, g.rosterBest(o))

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
// On a short screen the slot cards take the whole width, the traits small chips beside the grades.
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
	// a short screen (a phone) leaves out the sum, which the slots' grades tell too, and gives the slots the width:
	// a smaller icon, the traits as small chips beside the grades
	rowH := min(128, (r.Dy()-2*10)/3)
	compact := rowH < 120
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
		for i, a := range hexAxes {
			g.label(dst, a.stat, x, y, 13, ink)
			grade := "–"
			if total[i] != 0 {
				grade = bonusGrade(total[i])
			}
			g.label(dst, grade, x+44, y, 13, gold)
			bar(dst, float32(x+76), float32(y+6), float32(sumW-112), 8, max(0, total[i])/most, gold)
			y += 26
		}
		y += 14
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
		iconK, iconW := 1.8, 64.0
		if compact {
			iconK, iconW = 1.3, 48
		}
		ix, iy := float64(c.Min.X+16), float64(c.Min.Y)+(float64(rowH)-iconW)/2
		id, worn := of.Equipment[slot]
		bx := c.Max.X - 14 - 96
		slot := slot
		g.mBtn(dst, image.Rect(bx, c.Min.Y+(rowH-touchH)/2, bx+96, c.Min.Y+(rowH+touchH)/2), "바꾸기", false, func() {
			g.openOverlay("equipment")
			g.openPicker(slot)
		})
		if _, owner, gearSlot, ok := g.gearStep(); ok && owner == of.ID && gearSlot == slot && (g.overlay() == "party" || g.overlay() == "") { // the party window or the sortie preparation
			g.guideRim(dst, image.Rect(bx, c.Min.Y+(rowH-touchH)/2, bx+96, c.Min.Y+(rowH+touchH)/2))
		}
		tx := ix + iconW + 18
		if !worn {
			g.itemIcon(dst, slotIcons[slot], ix+3, iy+3, iconK, true)
			g.label(dst, "비어 있음", tx, float64(c.Min.Y+14), 16, muted)
			g.label(dst, slotNames[slot], tx+g.width("비어 있음", 16)+8, float64(c.Min.Y+18), 12, muted)
			continue
		}
		eq := g.S.Data.Equipment[id]
		g.itemIcon(dst, eq.Icon, ix+3, iy+3, iconK, false)
		top := c.Min.Y + 12
		if compact {
			top = c.Min.Y + 8
		}
		g.label(dst, eq.Name, tx, float64(top), 16, ink)
		g.label(dst, slotNames[slot], tx+g.width(eq.Name, 16)+8, float64(top+4), 12, muted)
		px := tx
		for k, v := range eq.Bonus {
			if v == 0 {
				continue
			}
			px += g.gradePill(dst, hexAxes[k].stat, bonusGrade(v), px, float64(top+28)) + 6
		}
		if px == tx {
			none := "능력 변화 없음"
			if compact && len(eq.Traits) > 0 {
				none = ""
			}
			g.label(dst, none, tx, float64(top+30), 12, faintInk)
		}
		if !compact {
			chips(dst, slot, eq.Name, eq.Traits, int(tx)-4, c.Min.Y+66, bx-10)
			continue
		}
		key, from := slot, eq.Name
		_, placed := g.smallChips(dst, eq.Traits, image.Rect(int(px), top+24, bx-10, top+24+smallChipH), 1, false, nil, func(id string) {
			if p.tip == key+"|"+id {
				p.tip = ""
			} else {
				p.tip = key + "|" + id
			}
		})
		for _, sc := range placed {
			if sc.id != "" && p.tip == key+"|"+sc.id {
				tip = &tipAt{sc.id, from, sc.r}
			}
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

// officerDeputy is the sortie's 부관 tab for a lead going out: the candidates in a table (deputyTable), sorted by
// the weight they would get, and under it a card of the one picked — class, level and weight, what naming it does
// or why it cannot be, and every trait it would bring as a small chip ("+n" when they need more lines than the
// card gets). It gives the actions that dismiss the deputy and appoint the one picked.
func (g *Game) officerDeputy(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) []action {
	note := ""
	switch lead := g.servedBy(o, of.ID); {
	case lead != "":
		note = g.name(lead) + "의 부관입니다. 부관은 부관을 둘 수 없습니다."
	case !slices.Contains(o.Deployment, of.ID):
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
