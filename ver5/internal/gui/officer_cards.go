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
// experience) and beside it, as tall, the hexagon card; the biography card fills what is left under them.
func (g *Game) officerBio(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) {
	stats, attrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return
	}
	const faceW, gap = 184, 12
	topH := min(290, r.Dy()-120)
	face := image.Rect(r.Min.X, r.Min.Y, r.Min.X+faceW, r.Min.Y+topH)
	g.card(dst, face)
	size := float64(max(56, min(150, topH-140)))
	x, y := float64(face.Min.X+16), float64(face.Min.Y+16)
	g.portrait(dst, of.ID, "", x, y, size)
	y += size + 12
	g.classRank(dst, of.Def, of.Class, of.Level, fmt.Sprintf(" · Lv%d", of.Level), x, y, 13, muted)
	cl := g.S.Data.Classes[of.Class]
	reach := fmt.Sprint(cl.Range)
	if cl.MinRange > 1 {
		reach = fmt.Sprintf("%d~%d", cl.MinRange, cl.Range)
	}
	g.label(dst, fmt.Sprintf("이동 %d · 사거리 %s", cl.Move, reach), x, y+22, 13, muted)
	mp := fmt.Sprintf("병법치 %d", stats.MaxMP)
	g.label(dst, mp, x, y+44, 13, muted)
	g.label(dst, fmt.Sprintf("(+%d)", stats.Recovery), x+g.width(mp, 13)+5, y+44, 13, good)
	progress := core.ExperienceProgress(of.XP)
	g.label(dst, fmt.Sprintf("경험치 %.0f / 100", progress), x, y+66, 13, muted)
	bar(dst, float32(x), float32(y+88), float32(faceW-32), 8, progress/100, xpC)

	hex := image.Rect(face.Max.X+gap, r.Min.Y, r.Max.X, r.Min.Y+topH)
	g.card(dst, hex)
	radius := max(40, min(84, float64(topH-120)/2, float64(hex.Dx()-220)/2))
	g.hexChartAttrs(dst, float64(hex.Min.X+hex.Max.X)/2, float64(hex.Min.Y+hex.Max.Y)/2+4, radius, stats, attrs, g.rosterBest(o))

	bio := image.Rect(r.Min.X, face.Max.Y+gap, r.Max.X, r.Max.Y)
	g.card(dst, bio)
	g.label(dst, "열전", float64(bio.Min.X+18), float64(bio.Min.Y+14), 13, gold)
	text := g.S.Data.Characters[of.Def].Biography
	if text == "" {
		text = "등록된 열전이 없습니다."
	}
	lines := int((float64(bio.Dy()) - 50) / (15 * 1.75))
	g.labelSpaced(dst, clipLines(wrapWidth(text, g.face(15), float64(bio.Dx()-36)), lines), float64(bio.Min.X+18), float64(bio.Min.Y+38), 15, 1.75, ink)
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

	rows := image.Rect(sum.Max.X+gap, r.Min.Y, r.Max.X, r.Max.Y)
	rowH := min(128, (rows.Dy()-2*10)/3)
	for i, slot := range equipSlots {
		c := image.Rect(rows.Min.X, rows.Min.Y+i*(rowH+10), rows.Max.X, rows.Min.Y+i*(rowH+10)+rowH)
		g.card(dst, c)
		ix, iy := float64(c.Min.X+16), float64(c.Min.Y+(rowH-64)/2)
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
		tx := ix + 64 + 18
		if !worn {
			g.itemIcon(dst, slotIcons[slot], ix+3, iy+3, 1.8, true)
			g.label(dst, "비어 있음", tx, float64(c.Min.Y+14), 16, muted)
			g.label(dst, slotNames[slot], tx+g.width("비어 있음", 16)+8, float64(c.Min.Y+18), 12, muted)
			continue
		}
		eq := g.S.Data.Equipment[id]
		g.itemIcon(dst, eq.Icon, ix+3, iy+3, 1.8, false)
		g.label(dst, eq.Name, tx, float64(c.Min.Y+12), 16, ink)
		g.label(dst, slotNames[slot], tx+g.width(eq.Name, 16)+8, float64(c.Min.Y+16), 12, muted)
		px := tx
		for k, v := range eq.Bonus {
			if v == 0 {
				continue
			}
			px += g.gradePill(dst, hexAxes[k].stat, bonusGrade(v), px, float64(c.Min.Y+40)) + 6
		}
		if px == tx {
			g.label(dst, "능력 변화 없음", tx, float64(c.Min.Y+42), 12, faintInk)
		}
		chips(dst, slot, eq.Name, eq.Traits, int(tx)-4, c.Min.Y+66, bx-10)
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

// officerDeputy is the sortie's 부관 tab for a lead going out: on the left a card listing who could serve, each
// with where they are, their charm and the weight they would get; on the right a card of the one picked — what it
// raises, the unit's hexagon with it (white) over the lead alone (dashed gold) with each stat's bonus, and the
// traits it would bring. It gives the actions that dismiss the deputy and appoint the one picked.
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
	alone, aloneAttrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return nil
	}
	p := &g.party
	if p.depLead != of.ID {
		p.depLead, g.deputyPick = of.ID, ""
	}
	rows := g.deputyRows(o)
	slices.SortStableFunc(rows, func(a, b deputyRow) int {
		if a.current != b.current {
			return b2i(b.current) - b2i(a.current)
		}
		return b2i(a.reason != "") - b2i(b.reason != "")
	})
	pick := g.deputyPick
	if pick == "" {
		pick = of.Deputy
	}
	at := slices.IndexFunc(rows, func(r deputyRow) bool { return r.id == pick })
	if at < 0 && len(rows) > 0 {
		at, pick = 0, rows[0].id
	}
	const listW, gap, rowH = 210, 12, 52
	list := image.Rect(r.Min.X, r.Min.Y, r.Min.X+listW, r.Max.Y)
	g.card(dst, list)
	g.listRows(dst, &p.deputies, list.Inset(5), len(rows), rowH, func(i int, rr image.Rectangle) {
		d := rows[i]
		c := g.officer(o, d.id)
		if c == nil {
			return
		}
		rowBack(dst, rr, d.id == pick)
		fg := color.Color(ink)
		if d.reason != "" {
			fg = faintInk
		}
		x, y := float64(rr.Min.X+10), float64(rr.Min.Y)
		g.portrait(dst, c.ID, c.Class, x, y+8, 36)
		g.label(dst, d.name, x+44, y+6, 14, fg)
		if d.current {
			g.label(dst, "지금", x+50+g.width(d.name, 14), y+9, 11, depInk)
		}
		where := "대기"
		switch {
		case d.current:
			where = g.name(of.ID) + " 부관"
		case g.servedBy(o, d.id) != "":
			where = g.name(g.servedBy(o, d.id)) + " 부관"
		case slices.Contains(o.Deployment, d.id):
			where = "출전"
		}
		_, depAttrs, _ := g.S.Engine.DeputyStats(d.id, "")
		g.label(dst, ellipsis(g, fmt.Sprintf("%s · 매력 %d", where, depAttrs[5]), 11, rr.Dx()-110), x+44, y+28, 11, muted)
		if d.reason == "" {
			wr, _ := g.S.Engine.DeputyBlend(of.ID, d.id)
			ws := fmt.Sprintf("%.0f%%", wr*100)
			g.label(dst, ws, float64(rr.Max.X-10)-g.width(ws, 15), y+15, 15, gold)
		}
	}, func(i int) { g.deputyPick = rows[i].id })

	main := image.Rect(list.Max.X+gap, r.Min.Y, r.Max.X, r.Max.Y)
	g.card(dst, main)
	var picked *deputyRow
	if at >= 0 {
		picked = &rows[at]
	}
	if picked == nil {
		g.label(dst, "부관이 될 장수가 없습니다.", float64(main.Min.X+18), float64(main.Min.Y+20), 14, muted)
		return nil
	}
	x, y := float64(main.Min.X+16), float64(main.Min.Y+16)
	if c := g.officer(o, picked.id); c != nil {
		g.portrait(dst, c.ID, c.Class, x, y, 52)
		g.label(dst, picked.name, x+64, y, 17, ink)
		g.classRank(dst, c.Def, c.Class, c.Level, fmt.Sprintf(" · Lv%d", c.Level), x+72+g.width(picked.name, 17), y+4, 12, muted)
	}
	ups := ""
	for k, a := range attrNames {
		if d := picked.attrs[k] - aloneAttrs[k]; d > 0 {
			if ups != "" {
				ups += " · "
			}
			ups += fmt.Sprintf("%s +%d", a, d)
		}
	}
	upsC := color.Color(good)
	switch {
	case picked.reason != "":
		ups, upsC = picked.reason, faintInk
	case ups == "":
		ups, upsC = "오르는 능력 없음", faintInk
	}
	lines := clipLines(wrapWidth(ups, g.face(13), float64(main.Dx()-170)), 2)
	g.label(dst, lines, x+64, y+24, 13, upsC)
	if picked.note != "" {
		_, lh := measureLines(g, lines, 13)
		g.label(dst, ellipsis(g, picked.note, 12, main.Dx()-170), x+64, y+26+lh, 12, color.NRGBA{R: 224, G: 160, B: 112, A: 255})
	}
	if picked.reason == "" {
		wr, _ := g.S.Engine.DeputyBlend(of.ID, picked.id)
		ws := fmt.Sprintf("%.0f%%", wr*100)
		g.label(dst, ws, float64(main.Max.X-18)-g.width(ws, 24), y-2, 24, gold)
		g.label(dst, "가중치", float64(main.Max.X-18)-g.width("가중치", 11), y+30, 11, muted)
	}

	traitsH := 84
	hexTop, hexBot := y+90, float64(main.Max.Y-traitsH-24)
	var names [6]string
	var unit [6]int
	var dashed [6]int
	var subs hexSubs
	for i, a := range hexAxes {
		names[i], unit[i], dashed[i] = a.stat, a.get(picked.stats), a.get(alone)
		if d := unit[i] - dashed[i]; d != 0 {
			subs.text[i], subs.c[i] = fmt.Sprintf("%+d", d), good
		} else {
			subs.text[i], subs.c[i] = "—", faintInk
		}
	}
	var other *[6]int
	if unit != dashed {
		other = &dashed
	}
	best := g.rosterBest(o)
	for i := range best {
		best[i] = max(best[i], unit[i])
	}
	radius := max(40, min(96, (hexBot-hexTop-90)/2, float64(main.Dx()-190)/2))
	cy := (hexTop+hexBot)/2 + 4
	g.hexagon(dst, float64(main.Min.X+main.Max.X)/2, cy, radius, names, unit, other, best, []float64{1, 2.0 / 3, 1.0 / 3}, &subs)
	ly := hexBot + 2
	lx := x
	rect(dst, float32(lx), float32(ly+8), 18, 2, white)
	g.label(dst, "부대", lx+24, ly, 12, muted)
	lx += 24 + g.width("부대", 12) + 16
	for k := 0; k < 18; k += 6 {
		rect(dst, float32(lx)+float32(k), float32(ly+8), 3, 2, gold)
	}
	g.label(dst, g.name(of.ID)+" 혼자", lx+24, ly, 12, muted)
	lx += 24 + g.width(g.name(of.ID)+" 혼자", 12) + 16
	mp := fmt.Sprintf("병법치 %d → ", alone.MaxMP)
	g.label(dst, mp, lx, ly, 12, muted)
	g.label(dst, fmt.Sprint(picked.stats.MaxMP), lx+g.width(mp, 12), ly, 12, gold)

	ty := float64(main.Max.Y - traitsH)
	g.label(dst, "부대에 더해지는 특성", x, ty, 12, muted)
	tx := int(x) - 4
	if len(picked.traits) == 0 {
		g.label(dst, "없음", x, ty+24, 12, faintInk)
	}
	for _, t := range picked.traits {
		e := traitEntry{id: t, have: true}
		if tx+g.traitChipW(e) > main.Max.X-12 {
			break
		}
		tx += g.traitChip(dst, tx, int(ty)+18, e, false, nil) + 6
	}

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
