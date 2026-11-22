package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

const sortieTransitionSeconds = 0.1

type sortieTransition struct {
	from, started float64
	active        bool
}

func (f sortieTransition) value(now float64) float64 {
	elapsed := (now - f.started) / sortieTransitionSeconds
	if f.active {
		return min(1, f.from+elapsed)
	}
	return max(0, f.from-elapsed)
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
	cw := max(1, r.Dx()/3)
	ch := max(1, (r.Max.Y-top)/6)
	size := min(cw, ch) * 10 / 11
	for i := range min(g.S.Data.Stages[o.Stage].Max, 18) {
		x := r.Min.X + (i%3)*cw + (cw-size*11/10)/2
		y := top + (i/3)*ch + (ch-size*11/10)/2
		lead := image.Rect(x+size/10, y+size/10, x+size*9/10, y+size*9/10)
		if i >= len(ids) {
			g.sortieSlot(dst, lead)
			continue
		}
		id := ids[i]
		of := g.officer(o, id)
		if of == nil {
			g.sortieSlot(dst, lead)
			continue
		}

		var slot image.Rectangle
		var deputy *core.Character
		var deputyBrightness float64
		if of.Deputy != "" || g.S.Engine.CanDeputize(id) {
			slot = image.Rect(x+size*7/10, y+size*7/10, x+size*11/10, y+size*11/10)
			g.sortieSlot(dst, slot)
			deputy = g.officer(o, of.Deputy)
			if deputy != nil {
				deputyBrightness = g.sortiePortraitBrightness(deputy.ID)
				g.sortiePortrait(dst, deputy, slot, deputyBrightness, 1)
			}
		}

		g.sortieSlot(dst, lead)
		g.sortieLeadSprite(dst, of, lead, g.sortiePortraitBrightness(id))
		if g.sortieLocked(id) {
			x, y := float64(lead.Min.X+2), float64(lead.Min.Y+1)
			write(dst, "필수", g.boldFace(9), x+1, y+1, color.Black)
			write(dst, "필수", g.boldFace(9), x, y, gold)
		}
		g.sortiePortraitName(dst, lead, id)
		g.addButton(lead, "", func() { g.selected, g.party.trait, g.party.tip = id, "", "" })

		if deputy != nil {
			key := [2]string{id, deputy.ID}
			if g.party.sortieFades == nil {
				g.party.sortieFades = make(map[[2]string]sortieTransition)
			}
			transition := g.party.sortieFades[key]
			active := g.overlay() == "" && len(g.modals) == 0 && hovered(slot)
			opacity := transition.update(g.uiSeconds, active || deputy.ID == g.selected)
			g.party.sortieFades[key] = transition
			if overlap := slot.Intersect(lead); opacity > 0 && !overlap.Empty() {
				layer := clipped(dst, overlap)
				g.sortiePortrait(layer, deputy, slot, deputyBrightness, opacity)
				g.sortieSlotRim(layer, slot, fade(cardRim, opacity))
			}
			g.sortiePortraitName(dst, slot, deputy.ID)
			g.addButton(slot, g.name(deputy.ID), func() {
				if g.party.sortieView == 5 && g.party.depLead == g.selected &&
					g.S.Engine.CanDeputize(g.selected) && slices.Contains(o.Deployment, g.selected) {
					g.deputyPick, g.deputyInfo, g.party.depAll = deputy.ID, equipChip{}, false
					return
				}
				g.selected, g.party.trait, g.party.tip = deputy.ID, "", ""
				g.party.officerList.revealSelected = g.party.sortieView < 3
			})
		} else if !slot.Empty() {
			g.addButton(slot, g.name(id)+" 부관 선택", func() {
				if g.openOverlay("deputy") {
					g.selected, g.party.trait, g.party.tip = id, "", ""
				}
			})
		}
		switch {
		case id == g.selected:
			g.sortieSlotRim(dst, lead, gold)
		case deputy != nil && deputy.ID == g.selected:
			g.sortieSlotRim(dst, slot, gold)
		}
	}
}

func (g *Game) sortieLeadSprite(dst *ebiten.Image, officer *core.Character, r image.Rectangle, brightness float64) {
	weapon := g.S.Data.Equipment[officer.Equipment["weapon"]].Type
	faction := g.S.Data.Characters[officer.Def].Look.Faction
	art := g.battleClassArt(officer.Def, officer.Class, weapon, g.mountOf(*officer), faction)
	if art == nil {
		return
	}
	scale := min(1, float64(max(1, r.Dy()-23))/max(1, art.TopOffset()))
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

func (g *Game) sortiePortraitName(dst *ebiten.Image, r image.Rectangle, id string) {
	size := min(10, float64(r.Dy())/3)
	name := ellipsis(g, g.name(id), size, r.Dx()-4)
	g.outlined(dst, name, float64(r.Min.X+r.Dx()/2)-g.width(name, size)/2, float64(r.Max.Y)-size-2, size)
}

func (g *Game) sortieSlot(dst *ebiten.Image, r image.Rectangle) {
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.NRGBA{R: 31, G: 24, B: 20, A: 255})
	g.sortieSlotRim(dst, r, cardRim)
}

func (g *Game) sortieSlotRim(dst *ebiten.Image, r image.Rectangle, rim color.Color) {
	strokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, rim)
}

func (g *Game) sortiePortraitBrightness(id string) float64 {
	if g.party.sortieBrightness == nil {
		g.party.sortieBrightness = make(map[string]sortieTransition)
	}
	transition := g.party.sortieBrightness[id]
	amount := transition.update(g.uiSeconds, id == g.selected)
	g.party.sortieBrightness[id] = transition
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
	g.wheelAreas = append(g.wheelAreas, wheelArea{unframed(tableRect), &p.officerList})
	drawTable(g, dst, &p.officerList, table[row]{cols: cols, rowH: touchH, headH: 32, size: 13, center: true, dim: func(r *row) bool { return r.deputy }, selected: func(r *row) bool { return r.c.ID == g.selected }, back: func(dst *ebiten.Image, r *row, rr image.Rectangle) {
		if slices.Contains(o.Deployment, r.c.ID) || g.sortieReserve(o, r.c.ID) {
			rect(dst, float32(rr.Min.X+1), float32(rr.Min.Y+1), float32(rr.Dx()-3), float32(rr.Dy()-2), color.NRGBA{R: 78, G: 144, B: 98, A: 46})
		}
	}, pick: func(r *row) { g.selected = r.c.ID; p.trait = ""; p.tip = "" }}, rows, tableRect)
}
