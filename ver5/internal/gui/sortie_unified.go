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

// A card lights up at once and dims over sortieDimSeconds.
const sortieDimSeconds = 0.2

type sortieTransition struct {
	from, started float64
	active        bool
}

func (f sortieTransition) value(now float64) float64 {
	if f.active {
		return 1
	}
	return max(0, f.from-(now-f.started)/sortieDimSeconds)
}

func (f *sortieTransition) update(now float64, active bool) float64 {
	if f.active != active {
		*f = sortieTransition{from: f.value(now), started: now, active: active}
	}
	return f.value(now)
}

func (g *Game) sortieGrid(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	g.label(dst, "출전 부대", float64(r.Min.X+4), float64(r.Min.Y+6), 15, gold)
	if slot := g.campaignSlot(); slot != nil {
		g.cornerButton(dst, image.Rect(r.Max.X-90, r.Min.Y, r.Max.X-2, r.Min.Y+30), slot)
	}
	if slot := g.campaignPolicySlot(); slot != nil {
		g.cornerButton(dst, image.Rect(r.Max.X-184, r.Min.Y, r.Max.X-94, r.Min.Y+30), slot)
	}
	ids := g.placeable(o)
	top := r.Min.Y + 34
	slots := min(g.S.Data.Stages[o.Stage].Max, 18)
	area := image.Rect(r.Min.X, top, r.Max.X, r.Max.Y)
	avail := area.Dx() - tableBarW - 4
	cols := sortieColumns(avail)
	cw := max(1, avail/cols)
	g.listRows(dst, &g.party.sortieSlots, area, (slots+cols-1)/cols, sortieCellH, func(row int, rr image.Rectangle) {
		for col := range cols {
			if i := row*cols + col; i < slots {
				g.sortieCell(dst, o, ids, i, rr.Min.X+col*cw+(cw-sortieCellW)/2, rr.Min.Y)
			}
		}
	}, nil)
}

// sortieColumns is how many cards a row holds in avail px: sortieCols at most (5 on the desktop, 3 on the phone), and
// fewer where the window is too narrow for that many.
func sortieColumns(avail int) int { return max(1, min(sortieCols, avail/sortieCellW)) }

// The card is sized in fixed pixels, not for the window: the officer's card, the deputy's medal that overhangs its
// bottom-right corner (sizes per platform in platform_*.go), and the margin around both.
const (
	sortieOutX  = sortieMedal / 11
	sortieOutY  = sortieMedal * 9 / 56
	sortieCellW = 6 + sortieLeadW + sortieOutX + 6
	sortieCellH = 6 + sortieLeadH + sortieOutY + 6
)

// sortieCell draws the sortie slot i, an officer's card with its deputy's medal, with its corner at (x, y).
// Rule: a picture selects that person; the swap mark (or an empty medal's +) opens the lead's deputy window.
func (g *Game) sortieCell(dst *ebiten.Image, o core.Observation, ids []string, i, x, y int) {
	lead := image.Rect(x+6, y+6, x+6+sortieLeadW, y+6+sortieLeadH)
	if i >= len(ids) {
		g.sortieSlot(dst, lead)
		return
	}
	id := ids[i]
	of := g.officer(o, id)
	if of == nil {
		g.sortieSlot(dst, lead)
		return
	}

	var medal, swap image.Rectangle
	var deputy *core.Character
	canDeputy := of.Deputy != "" || g.S.Engine.CanDeputize(id)
	if canDeputy {
		medal = image.Rect(lead.Max.X+sortieOutX-sortieMedal, lead.Max.Y+sortieOutY-sortieMedal, lead.Max.X+sortieOutX, lead.Max.Y+sortieOutY)
		deputy = g.officer(o, of.Deputy)
		c := medal.Min.Add(image.Pt(sortieMedal/2, sortieMedal/2))
		off := sortieMedal * 35 / 100 // the medal rim at 45 degrees
		swap = image.Rect(c.X-off-sortieSwap/2, c.Y-off-sortieSwap/2, c.X-off+sortieSwap/2, c.Y-off+sortieSwap/2)
	}

	idle := g.overlay() == "" && len(g.modals) == 0 && !touchUI
	onMedal := idle && canDeputy && (hovered(medal) || deputy != nil && hovered(swap))
	showSwap := deputy != nil && (onMedal || touchUI && deputy.ID == g.selected)
	g.sortieSlot(dst, lead)
	lit := id == g.selected || idle && hovered(lead) && !onMedal
	g.sortieLeadArt(dst, of, lead, g.sortieLight(id, lit))
	if g.sortieLocked(id) {
		x, y := float64(lead.Min.X+2), float64(lead.Max.Y-16)
		write(dst, "필수", g.boldFace(13), x+1, y+1, color.Black)
		write(dst, "필수", g.boldFace(13), x, y, gold)
	}
	g.sortieLeadName(dst, lead, id, lit)
	g.addButton(lead, "", func() { g.selected, g.party.trait, g.party.tip = id, "", "" })

	if canDeputy {
		openDeputy := func() {
			if g.openOverlay("deputy") {
				g.selected, g.party.trait, g.party.tip = id, "", ""
			}
		}
		hit := medal
		if touchUI {
			hit = hit.Inset(-6)
		}
		if deputy != nil {
			g.sortieMedalPlate(dst, medal, deputy, g.sortieLight(id+"/deputy", deputy.ID == g.selected || id == g.selected || onMedal))
			g.addButton(hit, g.name(deputy.ID), func() {
				g.selected, g.party.trait, g.party.tip = deputy.ID, "", ""
				g.party.officerList.revealSelected = g.party.sortieView < 3
			})
			if showSwap {
				g.sortieSwapMark(dst, swap)
				g.addButton(swap.Inset(-6), g.name(id)+" 부관 교체", openDeputy)
			}
		} else {
			g.sortieMedalPlate(dst, medal, nil, g.sortieLight(id+"/deputy", onMedal))
			g.addButton(hit, g.name(id)+" 부관 선택", openDeputy)
		}
	}
	switch {
	case id == g.selected:
		g.sortieSlotRim(dst, lead, gold)
	case deputy != nil && deputy.ID == g.selected:
		g.sortieRoundRim(dst, medal, gold)
	}
}

// sortieLeadArt fills the card with the officer's portrait, cropped from the top to the card's shape; an officer
// without one keeps the dot sprite.
func (g *Game) sortieLeadArt(dst *ebiten.Image, of *core.Character, r image.Rectangle, brightness float64) {
	look := g.laterLook(g.S.Data.Characters[of.Def].Look)
	var src *ebiten.Image
	if !look.NoPortrait {
		src = g.assets.portrait(look.Portrait)
	}
	if src == nil {
		g.sortieLeadSprite(dst, of, r, brightness)
		return
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w*r.Dy() > h*r.Dx() {
		w = h * r.Dx() / r.Dy()
	} else {
		h = w * r.Dy() / r.Dx()
	}
	x0 := b.Min.X + (b.Dx()-w)/2
	sub := src.SubImage(image.Rect(x0, b.Min.Y, x0+w, b.Min.Y+h)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(r.Dx())/float64(w), float64(r.Dy())/float64(h))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	shade := float32(brightness)
	op.ColorScale.Scale(shade, shade, shade, 1)
	drawScaled(dst, sub, op)
}

func (g *Game) sortieLeadSprite(dst *ebiten.Image, officer *core.Character, r image.Rectangle, brightness float64) {
	weapon := g.S.Data.Equipment[officer.Equipment["weapon"]].Type
	faction := g.S.Data.Characters[officer.Def].Look.Faction
	art := g.battleClassArt(officer.Def, officer.Class, weapon, g.mountOf(*officer), faction)
	if art == nil {
		return
	}
	const scale = 2.0
	x, y := float64(r.Min.X+r.Dx()/2), float64(r.Max.Y-15)
	frame := art.FrameIndex("idle", g.uiSeconds*1000)
	dst = clipped(dst, r)
	draw := func(src *ebiten.Image, px, py float64) {
		if src == nil {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x-px*scale, y-py*scale)
		shade := float32(brightness)
		op.ColorScale.Scale(shade, shade, shade, 1)
		drawScaled(dst, src, op)
	}
	draw(art.Shadow("SW", "idle", frame), art.ShadowPivot[0], art.ShadowPivot[1])
	draw(art.Frame("SW", "idle", frame), art.Pivot[0], art.Pivot[1])
}

// sortieLeadName writes the officer's name at the top of the card, centred, over a shade, and the class small at
// its bottom.
func (g *Game) sortieLeadName(dst *ebiten.Image, r image.Rectangle, id string, lit bool) {
	const size = 14
	for i := range 20 {
		rect(dst, float32(r.Min.X), float32(r.Min.Y+i), float32(r.Dx()), 1, color.NRGBA{A: uint8(150 * (20 - i) / 20)})
	}
	name := ellipsis(g, g.name(id), size, r.Dx()-4)
	g.outlined(dst, name, float64(r.Min.X+r.Dx()/2)-g.width(name, size)/2, float64(r.Min.Y)+3, size)
}

// sortieMedalName writes the deputy's name inside the lower part of its medal.
func (g *Game) sortieMedalName(dst *ebiten.Image, r image.Rectangle, id string) {
	size := min(15, float64(r.Dx())/5)
	name := ellipsis(g, g.name(id), size, int(float64(r.Dx())*.8))
	g.outlined(dst, name, float64(r.Min.X+r.Dx()/2)-g.width(name, size)/2, float64(r.Max.Y)-size*1.9, size)
}

// sortieMedalPlate draws the deputy's round medal: the portrait cropped square from the top, or a + on an empty one.
func (g *Game) sortieMedalPlate(dst *ebiten.Image, r image.Rectangle, deputy *core.Character, brightness float64) {
	const k = 4
	w, h := r.Dx()*k, r.Dy()*k
	p := &g.party
	if p.deputyLayer == nil || p.deputyLayer.Bounds().Dx() != w {
		p.deputyLayer, p.deputyMask = ebiten.NewImage(w, h), ebiten.NewImage(w, h)
		var path vector.Path
		roundedPath(&path, 0, 0, float32(w), float32(h), float32(w)/2)
		vector.FillPath(p.deputyMask, &path, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScaleWhite()})
	}
	layer := p.deputyLayer
	layer.Fill(color.NRGBA{R: 31, G: 24, B: 20, A: 255})
	if deputy != nil {
		if look := g.laterLook(g.S.Data.Characters[deputy.Def].Look); !look.NoPortrait {
			if src := g.assets.portrait(look.Portrait); src != nil {
				b := src.Bounds()
				side := min(b.Dx(), b.Dy())
				x0 := b.Min.X + (b.Dx()-side)/2
				sub := src.SubImage(image.Rect(x0, b.Min.Y, x0+side, b.Min.Y+side)).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
				op.GeoM.Scale(float64(w)/float64(side), float64(h)/float64(side))
				shade := float32(brightness)
				op.ColorScale.Scale(shade, shade, shade, 1)
				layer.DrawImage(sub, op)
			}
		}
	} else {
		c := color.NRGBA{R: 138, G: 122, B: 96, A: 255}
		m := float32(w) / 2
		vector.StrokeLine(layer, m, m-float32(w)/6, m, m+float32(w)/6, 3*k, c, true)
		vector.StrokeLine(layer, m-float32(w)/6, m, m+float32(w)/6, m, 3*k, c, true)
	}
	layer.DrawImage(p.deputyMask, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/k, 1.0/k)
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	drawScaled(dst, layer, op)
	if deputy != nil {
		g.sortieMedalName(dst, r, deputy.ID)
	}
	g.sortieRoundRim(dst, r, cardRim)
}

// sortieSwapMark draws the swap mark (two opposite arrows on a round chip) in r.
func (g *Game) sortieSwapMark(dst *ebiten.Image, r image.Rectangle) {
	px := float32(drawPx)
	cx := (float32(r.Min.X)+float32(r.Dx())/2)*px + float32(drawOX)
	cy := (float32(r.Min.Y)+float32(r.Dy())/2)*px + float32(drawOY)
	rad := float32(r.Dx()) / 2 * px
	vector.DrawFilledCircle(dst, cx, cy, rad, color.NRGBA{R: 58, G: 40, B: 29, A: 255}, true)
	vector.StrokeCircle(dst, cx, cy, rad, px, color.NRGBA{R: 160, G: 124, B: 60, A: 255}, true)
	half, gap, head, width := rad*.5, rad*.24, rad*.22, max(px, rad*.1)
	arrow := func(y, dir float32) {
		vector.StrokeLine(dst, cx-half, y, cx+half, y, width, gold, true)
		tip := cx + dir*half
		vector.StrokeLine(dst, tip, y, tip-dir*head, y-head, width, gold, true)
		vector.StrokeLine(dst, tip, y, tip-dir*head, y+head, width, gold, true)
	}
	arrow(cy-gap, 1)
	arrow(cy+gap, -1)
}

// sortieRoundRim outlines the deputy's round medal.
func (g *Game) sortieRoundRim(dst *ebiten.Image, r image.Rectangle, rim color.Color) {
	var path vector.Path
	px := float32(drawPx)
	roundedPath(&path, float32(r.Min.X)*px+float32(drawOX), float32(r.Min.Y)*px+float32(drawOY), float32(r.Dx())*px, float32(r.Dy())*px, float32(r.Dx())*px/2)
	var rimScale ebiten.ColorScale
	rimScale.ScaleWithColor(rim)
	vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: 2 * px}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: rimScale})
}

// roundedPath adds a rectangle with rounded corners of radius rad to path.
func roundedPath(path *vector.Path, x, y, w, h, rad float32) {
	path.MoveTo(x+rad, y)
	path.LineTo(x+w-rad, y)
	path.Arc(x+w-rad, y+rad, rad, -math.Pi/2, 0, vector.Clockwise)
	path.LineTo(x+w, y+h-rad)
	path.Arc(x+w-rad, y+h-rad, rad, 0, math.Pi/2, vector.Clockwise)
	path.LineTo(x+rad, y+h)
	path.Arc(x+rad, y+h-rad, rad, math.Pi/2, math.Pi, vector.Clockwise)
	path.LineTo(x, y+rad)
	path.Arc(x+rad, y+rad, rad, math.Pi, 3*math.Pi/2, vector.Clockwise)
	path.Close()
}

func (g *Game) sortieSlot(dst *ebiten.Image, r image.Rectangle) {
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.NRGBA{R: 31, G: 24, B: 20, A: 255})
	g.sortieSlotRim(dst, r, cardRim)
}

func (g *Game) sortieSlotRim(dst *ebiten.Image, r image.Rectangle, rim color.Color) {
	strokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, rim)
}

// sortieLight is how bright a card or deputy slot (key) is drawn: dim at rest, full while active.
func (g *Game) sortieLight(key string, active bool) float64 {
	if g.party.sortieBrightness == nil {
		g.party.sortieBrightness = make(map[string]sortieTransition)
	}
	transition := g.party.sortieBrightness[key]
	amount := transition.update(g.uiSeconds, active)
	g.party.sortieBrightness[key] = transition
	return 0.7 + 0.3*amount
}

func (g *Game) sortiePortrait(dst *ebiten.Image, officer *core.Character, r image.Rectangle, brightness, opacity float64) {
	look := g.laterLook(g.S.Data.Characters[officer.Def].Look)
	if look.NoPortrait {
		return
	}
	if src := g.assets.portrait(look.Portrait); src != nil {
		shade := uint8(brightness*255 + 0.5)
		tint := color.NRGBA{R: shade, G: shade, B: shade, A: uint8(opacity*255 + 0.5)}
		blit(dst, src, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), tint)
	}
}

func (g *Game) sortieTable(dst *ebiten.Image, o core.Observation, ids []string, r image.Rectangle) {
	p := &g.party
	type row struct {
		c                core.Character
		s, base          core.Stats
		attrs, baseAttrs [6]int
		deputy           bool
	}
	rows := make([]row, 0, len(ids))
	for _, id := range ids {
		if c := g.officer(o, id); c != nil {
			s, a, _ := g.S.Engine.DeputyStats(id, c.Deputy)
			base, baseAttrs := s, a
			if c.Deputy != "" {
				base, baseAttrs, _ = g.S.Engine.DeputyStats(id, "")
			}
			rows = append(rows, row{*c, s, base, a, baseAttrs, g.servedBy(o, id) != ""})
		}
	}
	num := func(key, label string, f func(*row) int) column[row] {
		return column[row]{key: key, label: label, get: func(r *row) (string, float64) { v := f(r); return fmt.Sprint(v), float64(v) }}
	}
	deputyNum := func(key, label string, f, base func(*row) int) column[row] {
		col := num(key, label, f)
		col.show = func(r *row, fg color.Color, _ float64) (string, color.Color) {
			value := f(r)
			if delta := value - base(r); delta != 0 {
				return fmt.Sprintf("%d (%+d)", value, delta), gold
			}
			return fmt.Sprint(value), fg
		}
		return col
	}
	statCol := func(key, label string, get func(core.Stats) int) column[row] {
		return deputyNum(key, label, func(r *row) int { return get(r.s) }, func(r *row) int { return get(r.base) })
	}
	cols := []column[row]{{key: "name", label: "장수", w: 95, text: true, left: true, get: func(r *row) (string, float64) { return g.name(r.c.ID), 0 }, after: func(dst *ebiten.Image, r *row, cell image.Rectangle) {
		if g.overlay() == "" && len(g.equipmentAdvice(o, r.c.ID)) > 0 {
			rim := cell.Inset(4)
			rim.Min.X -= 3
			rim.Max.X = min(rim.Max.X, rim.Min.X+int(g.width(g.name(r.c.ID), 13))+12)
			recommendationRim(dst, rim)
		}
	}}}
	switch p.sortieView {
	case 0:
		state := column[row]{key: "state", label: "상태", w: 55, text: true, get: func(r *row) (string, float64) {
			if g.sortieReserve(o, r.c.ID) {
				return "복병", 0
			}
			if r.deputy {
				return "부관", 0
			}
			if g.sortieLocked(r.c.ID) {
				return "필수", 0
			}
			if slices.Contains(o.Deployment, r.c.ID) {
				return "출전", 0
			}
			return "대기", 0
		}}
		state.show = func(r *row, fg color.Color, _ float64) (string, color.Color) {
			label, _ := state.get(r)
			if label == "부관" {
				fg = color.White
			}
			return label, fg
		}
		state.bold = func(r *row) bool {
			label, _ := state.get(r)
			return label == "부관" || label == "필수"
		}
		cols = append(cols, state, column[row]{key: "class", label: "병종", text: true, get: func(r *row) (string, float64) { return g.className(r.c.Class), 0 }}, num("level", "Lv", func(r *row) int { return r.c.Level }), statCol("hp", "병력", statRows[0].get), statCol("mp", "병법치", statRows[1].get), num("points", "특성치", func(r *row) int { return r.c.Points }))
	case 1:
		for _, v := range statRows[2:] {
			v := v
			cols = append(cols, statCol(v.name, v.name, v.get))
		}
	case 2:
		for i, n := range attrNames {
			i := i
			cols = append(cols, deputyNum(n, n, func(r *row) int { return r.attrs[i] }, func(r *row) int { return r.baseAttrs[i] }))
		}
	case 3:
		for _, slot := range equipSlots {
			slot := slot
			cols = append(cols, column[row]{key: slot, label: slotNames[slot], text: true, get: func(r *row) (string, float64) {
				if id := r.c.Equipment[slot]; id != "" {
					return g.itemName(id), 0
				}
				return "—", 0
			}})
		}
	}
	tableRect := r
	drawTable(g, dst, &p.officerList, table[row]{cols: cols, rowH: touchH, headH: 32, size: 13, center: true, dim: func(r *row) bool { return r.deputy }, selected: func(r *row) bool { return r.c.ID == g.selected }, back: func(dst *ebiten.Image, r *row, rr image.Rectangle) {
		if slices.Contains(o.Deployment, r.c.ID) || g.sortieReserve(o, r.c.ID) {
			rect(dst, float32(rr.Min.X+1), float32(rr.Min.Y+1), float32(rr.Dx()-3), float32(rr.Dy()-2), color.NRGBA{R: 78, G: 144, B: 98, A: 46})
		}
	}, pick: func(r *row) { g.selected = r.c.ID; p.trait = ""; p.tip = "" }}, rows, tableRect)
}
