package gui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// The equipment window shows the selected officer's three slots as cards: each piece's icon,
// name, what it adds to the officer's stats and the traits it brings. A card opens the picker,
// a table of the pieces in the warehouse the officer can wear in the slot, each with how it changes
// the stats against what is worn now; the picked one is shown in full below and put on with 확인. A trait badge opens its description over both.

var equipSlots = []string{"weapon", "armor", "accessory"}

var slotNames = map[string]string{"weapon": "무기", "armor": "방어구", "accessory": "장신구"}

// slotIcons stand in, dimmed, for an empty slot.
var slotIcons = map[string]string{"weapon": "sword", "armor": "robe", "accessory": "ration"}

const (
	equipCardH   = 132
	equipInfoW   = 460
	equipInfoH   = 250
	dragStartPx  = 8
	pickNothing  = "-" // the picker's "take it off" row
	equipCurrent = ""  // no pick yet
)

// equipState is the equipment window's state beyond the overlay name.
type equipState struct {
	slot  string // the slot whose picker is open, "" when closed
	pick  string // the picked piece, pickNothing to take the slot's piece off
	list  tableList
	info  equipChip
	alone bool // the picker was opened from the deployment's officer window: closing it closes the window too
}

// equipRow is one row of the picker: a piece, or taking the slot's piece off (id pickNothing).
type equipRow struct {
	id, name string
	bonus    [6]float64
	chips    []equipChip
	worn     bool
}

// equipChip is one trait or skill a piece brings: given while worn, learnable while worn, or
// a skill usable while worn.
type equipChip struct {
	from, id string // trait, learn, skill, deputy
}

// itemEffect is what a piece does for the officer: its stat gain over the empty slot and the
// change from what is worn now, or why it cannot be worn.
type itemEffect struct {
	gain, change []statDelta
	err          string
}

type statDelta struct {
	name string
	d    int
}

func (g *Game) officer(o core.Observation, id string) *core.Character {
	for i := range o.Characters {
		if o.Characters[i].ID == id {
			return &o.Characters[i]
		}
	}
	return nil
}

// equipmentWindow is the officer's gear, filling the screen: the three slots on the left, the pieces that fit the
// slot picked in the middle (taking the worn one off first), and the piece picked on the right with what it adds,
// what it changes against the worn one and the traits it brings. The action row puts it on.
func (g *Game) equipmentWindow(dst *ebiten.Image, x, y, w, h int) {
	o := g.S.View()
	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	if g.equip.slot == "" {
		g.openPicker("weapon")
	}
	slot := g.equip.slot
	g.label(dst, "장비", float64(x+16), float64(y+12), 18, gold)
	g.label(dst, fmt.Sprintf("%s · %s · %s 고르기", g.name(of.ID), g.className(of.Class), slotNames[slot]), float64(x+28)+g.width("장비", 18), float64(y+17), 13, muted)
	rect(dst, float32(x+8), float32(y+56), float32(w-16), 1, goldDim)
	const slotW = 230
	for i, sl := range equipSlots {
		rr := image.Rect(x+6, y+58+i*72, x+slotW, y+58+(i+1)*72)
		rowBack(dst, rr, sl == slot)
		sl := sl
		if !of.Dead && sl != slot {
			g.addButton(rr, slotNames[sl], func() { g.openPicker(sl) })
		}
		px, py := float64(rr.Min.X+14), float64(rr.Min.Y+14)
		g.label(dst, slotNames[sl], px+56, py-4, 12, muted)
		if id, ok := of.Equipment[sl]; ok {
			eq := g.S.Data.Equipment[id]
			g.itemIcon(dst, eq.Icon, px+3, py+3, 1.25, false)
			g.label(dst, ellipsis(g, eq.Name, 16, slotW-90), px+56, py+12, 16, ink)
			g.label(dst, ellipsis(g, bonusText(eq.Bonus, nil), 12, slotW-90), px+56, py+34, 12, muted)
		} else {
			g.itemIcon(dst, slotIcons[sl], px+3, py+3, 1.25, true)
			g.label(dst, "비어 있음", px+56, py+12, 16, muted)
		}
	}
	rect(dst, float32(x+slotW+2), float32(y+58), 1, float32(h-66), cardRim)
	rows := g.pickable(o, of, slot)
	picked := func(r *equipRow) bool { return r.id == g.equip.pick || g.equip.pick == equipCurrent && r.worn }
	listX := x + slotW + 4
	listW := (w - slotW - 32) / 2
	if len(rows) == 0 {
		g.label(dst, "창고에 이 칸에 맞는 장비가 없습니다.", float64(listX+14), float64(y+76), 14, muted)
	}
	g.listRows(dst, &g.equip.list, image.Rect(listX, y+58, listX+listW, y+h-6), len(rows), listRowH, func(i int, rr image.Rectangle) {
		r := rows[i]
		rowBack(dst, rr, picked(&r))
		px, py := float64(rr.Min.X+12), float64(rr.Min.Y)
		if r.id == pickNothing {
			g.itemIcon(dst, slotIcons[slot], px+3, py+12, 1, true)
			g.label(dst, "장비 벗기", px+46, py+16, 16, ink)
			return
		}
		eq := g.S.Data.Equipment[r.id]
		g.itemIcon(dst, eq.Icon, px+3, py+12, 1, false)
		name, c := eq.Name, ink
		if r.worn {
			name, c = name+" (착용)", gold
		}
		g.label(dst, ellipsis(g, name, 16, listW-70), px+46, py+7, 16, c)
		sub := bonusText(eq.Bonus, nil)
		for _, ch := range r.chips {
			sub += " · " + g.chipName(ch)
		}
		g.label(dst, ellipsis(g, sub, 12, listW-70), px+46, py+31, 12, muted)
	}, func(i int) { g.equip.pick = rows[i].id })
	rect(dst, float32(listX+listW+2), float32(y+58), 1, float32(h-66), cardRim)

	rx, right := listX+listW+16, x+w-16
	for i := range rows {
		r := &rows[i]
		if !picked(r) {
			continue
		}
		dx, dy := float64(rx), float64(y+66)
		if r.id == pickNothing {
			g.itemIcon(dst, slotIcons[slot], dx+3, dy+3, 1.5, true)
			g.label(dst, "장비 벗기", dx+64, dy+4, 20, gold)
			g.label(dst, wrapWidth(g.itemName(of.Equipment[slot])+"을(를) 창고로 돌립니다.", g.face(14), float64(right-rx)), dx, dy+64, 14, ink)
			break
		}
		eq := g.S.Data.Equipment[r.id]
		g.itemIcon(dst, eq.Icon, dx+3, dy+3, 1.5, false)
		g.label(dst, eq.Name, dx+64, dy-2, 20, gold)
		note := slotNames[slot]
		if r.worn {
			note += " · 착용 중"
		}
		g.label(dst, note, dx+64, dy+28, 12, muted)
		dy += 62
		e := g.itemEffect(of.ID, slot, r.id)
		g.label(dst, wrapWidth("보정  "+bonusText(eq.Bonus, e.gain), g.face(14), float64(right-rx)), dx, dy, 14, ink)
		dy += 26
		if !r.worn && len(e.change) > 0 {
			cx := dx
			for _, d := range e.change {
				s := fmt.Sprintf("%s %+d", d.name, d.d)
				c := good
				if d.d < 0 {
					c = bad
				}
				if cx+g.width(s, 13) > float64(right) {
					cx, dy = dx, dy+20
				}
				g.label(dst, s, cx, dy, 13, c)
				cx += g.width(s, 13) + 12
			}
			dy += 24
		}
		cx := int(dx)
		for _, ch := range r.chips {
			ch := ch
			e := traitEntry{id: ch.id, have: ch.from == "trait", affordable: ch.from != "trait"}
			if ch.from == "skill" {
				g.chip(dst, ch, float64(cx), dy+14, func() { g.equip.info = ch })
				cx += int(g.width(g.chipName(ch), 14)) + 50
				continue
			}
			if cx+g.traitChipW(e) > right && cx > rx {
				cx, dy = rx, dy+touchH
			}
			cx += g.traitChip(dst, cx, int(dy), e, false, func() { g.equip.info = ch }) + 6
		}
		if len(r.chips) == 0 {
			g.label(dst, "특성 없음", dx, dy+12, 13, muted)
		}
	}
	label := "바꾸기"
	switch {
	case g.equip.pick == pickNothing:
		label = "장비 벗기"
	case g.equip.pick != equipCurrent && g.equip.pick != of.Equipment[slot]:
		label = g.itemName(g.equip.pick) + "로 바꾸기"
	}
	var confirm func()
	if c, ok := g.pickCommand(of, slot); ok {
		confirm = func() {
			g.command(c)
			g.equip.pick = equipCurrent
		}
	}
	g.actionRow(dst, right, y+h-60, action{"닫기", false, g.closeOverlay}, action{label, true, confirm})
}

func itemChips(traits, learn, skills []string) []equipChip {
	out := []equipChip{}
	for _, id := range traits {
		out = append(out, equipChip{"trait", id})
	}
	for _, id := range learn {
		out = append(out, equipChip{"learn", id})
	}
	for _, id := range skills {
		out = append(out, equipChip{"skill", id})
	}
	return out
}

// chip draws the icon and the name after it (a learnable trait marked 배움) as one button that
// opens the description, and returns where it ends.
func (g *Game) chip(dst *ebiten.Image, c equipChip, x, y float64, click func()) float64 {
	name := g.chipName(c)
	w := traitIconSize + 6 + g.width(name, 14)
	tag := ""
	if c.from == "learn" {
		tag = " 배움"
		w += g.width(tag, 12)
	}
	r := image.Rect(int(x)-2, int(y)-2, int(x+w)+4, int(y)+traitIconSize+2)
	fg := ink
	if hovered(r) {
		fg = gold
		rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.NRGBA{R: 255, G: 240, B: 210, A: 24})
	}
	kind, tone := g.chipLook(c)
	g.traitIcon(dst, kind, tone, x, y, traitIconSize)
	g.label(dst, name, x+traitIconSize+6, y+1, 14, fg)
	if tag != "" {
		g.label(dst, tag, x+traitIconSize+6+g.width(name, 14), y+3, 12, muted)
	}
	g.addButton(r, "", click)
	return x + w
}

func (g *Game) chipLook(c equipChip) (kind, tone string) {
	if c.from == "skill" {
		return g.skillLook(c.id)
	}
	return g.traitLook(c.id)
}

func (g *Game) chipName(c equipChip) string {
	if c.from == "skill" {
		return g.skillName(c.id)
	}
	return g.traitName(c.id)
}

// equipPanel draws a card or a list row; picked rows get a gold rim.
func (g *Game) equipPanel(dst *ebiten.Image, r image.Rectangle, hover, picked bool) {
	body, rim := color.NRGBA{R: 44, G: 32, B: 25, A: 255}, goldDim
	if hover {
		body = color.NRGBA{R: 70, G: 48, B: 32, A: 255}
	}
	if picked {
		body, rim = color.NRGBA{R: 82, G: 56, B: 30, A: 255}, gold
	}
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), body)
	strokeRect(dst, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx())-1, float32(r.Dy())-1, 1, rim)
	if picked {
		rect(dst, float32(r.Min.X), float32(r.Min.Y), 4, float32(r.Dy()), gold)
	}
}

// itemIcon draws a piece's 32 UI px icon scaled k from its top-left corner; dim draws it as a
// faint stand-in for an empty slot.
func (g *Game) itemIcon(dst *ebiten.Image, icon string, x, y, k float64, dim bool) {
	rect(dst, float32(x-3), float32(y-3), float32(32*k+6), float32(32*k+6), trough)
	strokeRect(dst, float32(x-3)+.5, float32(y-3)+.5, float32(32*k+5), float32(32*k+5), 1, goldDim)
	if _, ok := g.assets.UI["item_"+icon]; !ok {
		return
	}
	c := white
	if dim {
		c = color.NRGBA{R: 70, G: 62, B: 56, A: 140}
	}
	g.drawUI(dst, "item_"+icon, 0, x, y, k, c)
}

func (g *Game) openPicker(slot string) {
	g.equip.slot, g.equip.pick, g.equip.list = slot, equipCurrent, tableList{}
}

// pickable are the picker's rows: taking the piece off when one is worn, the worn one (once, even
// with more in the warehouse), then the warehouse's pieces for the slot the officer can wear.
func (g *Game) pickable(o core.Observation, of *core.Character, slot string) []equipRow {
	row := func(id string) equipRow {
		eq := g.S.Data.Equipment[id]
		return equipRow{id: id, name: eq.Name, bonus: eq.Bonus, chips: itemChips(eq.Traits, eq.Learn, eq.Skills)}
	}
	out := []equipRow{}
	worn := of.Equipment[slot]
	if worn != "" {
		out = append(out, equipRow{id: pickNothing, name: "장비 벗기"})
		r := row(worn)
		r.worn = true
		out = append(out, r)
	}
	for _, id := range sorted(o.Warehouse) {
		if o.Warehouse[id] > 0 && g.S.Data.Equipment[id].Slot == slot && id != worn && g.itemEffect(of.ID, slot, id).err == "" {
			out = append(out, row(id))
		}
	}
	return out
}

// pickCommand is the command 확인 sends for the picked row, if it changes anything.
func (g *Game) pickCommand(of *core.Character, slot string) (core.Command, bool) {
	worn := of.Equipment[slot]
	switch g.equip.pick {
	case equipCurrent, worn:
		return core.Command{}, false
	case pickNothing:
		return core.Command{Kind: "unequip", Actor: of.ID, Item: worn}, true
	}
	return core.Command{Kind: "equip", Actor: of.ID, Item: g.equip.pick}, g.itemEffect(of.ID, slot, g.equip.pick).err == ""
}

// drawEquipInfo draws the description of the badge last pressed; any click closes it.
func (g *Game) drawEquipInfo(dst *ebiten.Image) {
	g.chipWindow(dst, "overlay:equip-info", g.equip.info, func() { g.equip.info = equipChip{} })
}

// chipWindow draws badge c's description as popup key over the screen; any click calls close.
func (g *Game) chipWindow(dst *ebiten.Image, key string, c equipChip, close func()) {
	g.addButton(image.Rect(0, 0, uiW, uiH), "", close)
	g.popup(dst, key, 100, equipInfoW+40, equipInfoH+40, image.Rect(0, 0, equipInfoW+40, equipInfoH+40), func(dst *ebiten.Image) {
		x, y := uiW/2-equipInfoW/2, uiH/2-equipInfoH/2
		kind, tone := g.chipLook(c)
		g.titled(dst, image.Rect(x, y, x+equipInfoW, y+equipInfoH), kindNames[kind]+" · "+toneNames[tone])
		g.traitIcon(dst, kind, tone, float64(x+28), float64(y+36), 44)
		g.label(dst, g.chipName(c), float64(x+90), float64(y+38), 22, gold)
		note, desc := "착용하는 동안 지닌다.", ""
		switch c.from {
		case "trait":
			desc = g.traitDesc(c.id)
		case "deputy":
			note, desc = "부관으로 있는 동안 주장에게 준다.", g.traitDesc(c.id)
		case "learn":
			note = fmt.Sprintf("착용하는 동안 특성치 %d로 배울 수 있다.", g.S.Data.Traits[c.id].Cost)
			desc = g.traitDesc(c.id)
		case "skill":
			note, desc = "착용하는 동안 쓸 수 있다.", g.S.Data.Skills[c.id].Desc
		}
		g.label(dst, note, float64(x+90), float64(y+72), 13, muted)
		g.label(dst, wrapWidth(desc, g.face(15), float64(equipInfoW-56)), float64(x+28), float64(y+112), 15, ink)
		g.centered(dst, "클릭하여 닫기", float64(x+equipInfoW/2), float64(y+equipInfoH-32), 13, muted)
	})
}

// itemEffect plays the piece on a copy of the game: the stats with the slot empty, with the
// piece, and as they are now.
func (g *Game) itemEffect(actor, slot, item string) itemEffect {
	key := fmt.Sprint(g.S.View().Revision, actor, slot, item)
	if e, ok := g.itemEffects[key]; ok {
		return e
	}
	e := g.playItem(actor, slot, item)
	if len(g.itemEffects) > 256 {
		g.itemEffects = map[string]itemEffect{}
	}
	g.itemEffects[key] = e
	return e
}

func (g *Game) playItem(actor, slot, item string) itemEffect {
	now, err := g.S.Engine.CharacterStats(actor)
	if err != nil {
		return itemEffect{err: koreanError(err.Error())}
	}
	sb, err := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err != nil {
		return itemEffect{err: koreanError(err.Error())}
	}
	of := g.officer(sb.Observe(), actor)
	worn := of.Equipment[slot]
	if worn != "" {
		if _, err := sb.Apply(core.Command{Kind: "unequip", Actor: actor, Item: worn}); err != nil {
			return itemEffect{err: koreanError(err.Error())}
		}
	}
	bare, _ := sb.CharacterStats(actor)
	if _, err := sb.Apply(core.Command{Kind: "equip", Actor: actor, Item: item}); err != nil {
		return itemEffect{err: koreanError(err.Error())}
	}
	with, _ := sb.CharacterStats(actor)
	return itemEffect{gain: statDeltas(bare, with), change: statDeltas(now, with)}
}

func statDeltas(a, b core.Stats) []statDelta {
	out := []statDelta{}
	for _, s := range statRows {
		if d := s.get(b) - s.get(a); d != 0 {
			out = append(out, statDelta{s.name, d})
		}
	}
	return out
}

// detailEquipment is the unit panel's equipment tab: each slot's icon and piece, and badges
// for the traits it brings that open the trait window.
func (g *Game) detailEquipment(dst *ebiten.Image, o core.Observation, id string, r image.Rectangle) {
	of := g.officer(o, id)
	if of == nil {
		return
	}
	x := float64(r.Min.X)
	for i, slot := range equipSlots {
		y := float64(r.Min.Y + i*72)
		item, ok := of.Equipment[slot]
		if !ok {
			g.itemIcon(dst, slotIcons[slot], x+3, y+3, 1, true)
			g.label(dst, slotNames[slot]+" · 없음", x+46, y+10, 13, muted)
			continue
		}
		eq := g.S.Data.Equipment[item]
		g.itemIcon(dst, eq.Icon, x+3, y+3, 1, false)
		g.label(dst, slotNames[slot], x+46, y, 11, muted)
		g.label(dst, eq.Name, x+46, y+14, 14, ink)
		cx := x + 46
		for _, c := range itemChips(eq.Traits, eq.Learn, nil) {
			t := c.id
			cx = g.chip(dst, c, cx, y+38, func() {
				if !g.popupPending() {
					g.modals = append(g.modals, &modal{kind: "trait", trait: t})
				}
			}) + 10
		}
	}
}
