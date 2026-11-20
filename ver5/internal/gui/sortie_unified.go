package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// Guan Yu and Zhang Fei stand about 45 source pixels above their ground pivots.
const sortieSpriteHeadroom = 48

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
	z := min(1, float64(max(1, ch-25))/sortieSpriteHeadroom)
	for i := 0; i < 18; i++ {
		x, y := r.Min.X+(i%3)*cw, top+(i/3)*ch
		cell := image.Rect(x+2, y+2, x+cw-2, y+ch-2)
		if cell.Max.Y > r.Max.Y {
			break
		}
		strokeRect(dst, float32(cell.Min.X), float32(cell.Min.Y), float32(cell.Dx()), float32(cell.Dy()), 1, cardRim)
		if i >= len(ids) {
			continue
		}
		id := ids[i]
		of := g.officer(o, id)
		if of == nil {
			continue
		}
		if id == g.selected {
			strokeRect(dst, float32(cell.Min.X+1), float32(cell.Min.Y+1), float32(cell.Dx()-2), float32(cell.Dy()-2), 1, gold)
		}
		weapon := g.S.Data.Equipment[of.Equipment["weapon"]].Type
		a := g.battleClassArt(of.Def, of.Class, weapon, g.mountOf(*of), g.S.Data.Characters[of.Def].Look.Faction)
		if a != nil {
			preview := clipped(dst, cell)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(float64(cell.Min.X+cell.Dx()/2)-a.Pivot[0]*z, float64(cell.Max.Y-19)-a.Pivot[1]*z)
			frame := a.FrameIndex("idle", g.uiSeconds*1000)
			shadow := &ebiten.DrawImageOptions{}
			shadow.GeoM.Scale(z, z)
			shadow.GeoM.Translate(float64(cell.Min.X+cell.Dx()/2)-a.ShadowPivot[0]*z, float64(cell.Max.Y-19)-a.ShadowPivot[1]*z)
			if sh := a.Shadow("SW", "idle", frame); sh != nil {
				drawScaled(preview, sh, shadow)
			}
			drawScaled(preview, a.Frame("SW", "idle", frame), op)
		}
		if g.sortieLocked(id) {
			g.label(dst, "필수", float64(cell.Min.X+3), float64(cell.Min.Y+2), 10, gold)
		}
		g.centered(dst, ellipsis(g, g.name(id), 12, cell.Dx()-4), float64(cell.Min.X+cell.Dx()/2), float64(cell.Max.Y-18), 12, ink)
		g.addButton(cell, "", func() { g.selected = id; g.party.trait = "" })
	}
}

func (g *Game) sortieTable(dst *ebiten.Image, o core.Observation, ids []string, r image.Rectangle) {
	p := &g.party
	type row struct {
		c                core.Character
		s, base          core.Stats
		attrs, baseAttrs [6]int
	}
	rows := make([]row, 0, len(ids))
	for _, id := range ids {
		if c := g.officer(o, id); c != nil {
			s, a, _ := g.S.Engine.DeputyStats(id, c.Deputy)
			base, baseAttrs := s, a
			if c.Deputy != "" {
				base, baseAttrs, _ = g.S.Engine.DeputyStats(id, "")
			}
			rows = append(rows, row{*c, s, base, a, baseAttrs})
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
		if _, owner, _, guide := g.gearStep(); guide && g.overlay() == "" && owner == r.c.ID && owner != g.selected {
			g.guideRim(dst, cell.Inset(2))
		}
	}}}
	switch p.sortieView {
	case 0:
		cols = append(cols, column[row]{key: "state", label: "상태", w: 55, text: true, get: func(r *row) (string, float64) {
			if g.sortieReserve(o, r.c.ID) {
				return "복병", 0
			}
			if g.servedBy(o, r.c.ID) != "" {
				return "부관", 0
			}
			if g.sortieLocked(r.c.ID) {
				return "필수", 0
			}
			if slices.Contains(o.Deployment, r.c.ID) {
				return "출전", 0
			}
			return "대기", 0
		}}, column[row]{key: "class", label: "병종", text: true, get: func(r *row) (string, float64) { return g.className(r.c.Class), 0 }}, num("level", "Lv", func(r *row) int { return r.c.Level }), statCol("hp", "병력", statRows[0].get), statCol("mp", "병법치", statRows[1].get), num("points", "특성치", func(r *row) int { return r.c.Points }))
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
	drawTable(g, dst, &p.officerList, table[row]{cols: cols, rowH: touchH, headH: 32, size: 13, center: true, selected: func(r *row) bool { return r.c.ID == g.selected }, back: func(dst *ebiten.Image, r *row, rr image.Rectangle) {
		if slices.Contains(o.Deployment, r.c.ID) || g.sortieReserve(o, r.c.ID) {
			rect(dst, float32(rr.Min.X+1), float32(rr.Min.Y+1), float32(rr.Dx()-3), float32(rr.Dy()-2), color.NRGBA{R: 78, G: 144, B: 98, A: 46})
		}
	}, pick: func(r *row) { g.selected = r.c.ID; p.trait = ""; p.tip = "" }}, rows, tableRect)
}
