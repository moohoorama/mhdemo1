package gui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// The equipment window shows the selected officer's three slots as cards: each piece's icon,
// name, what it adds to the officer's stats and the traits it brings. A card opens the picker,
// a scrolled list of the pieces in the warehouse that fit the slot, laid out the same way; the
// picked one is put on with 확인. A trait badge opens its description over both.

var equipSlots = []string{"weapon", "armor", "accessory"}

var slotNames = map[string]string{"weapon": "무기", "armor": "방어구", "accessory": "장신구"}

// slotIcons stand in, dimmed, for an empty slot.
var slotIcons = map[string]string{"weapon": "sword", "armor": "robe", "accessory": "ration"}

const (
	equipCardH   = 132
	pickRowH     = 112
	pickerW      = 720
	pickerH      = 680
	equipInfoW   = 460
	equipInfoH   = 250
	dragStartPx  = 8
	pickNothing  = "-" // the picker's "take it off" row
	equipCurrent = ""  // no pick yet
)

// equipState is the equipment window's state beyond the overlay name.
type equipState struct {
	slot     string // the slot whose picker is open, "" when closed
	pick     string // the picked piece, pickNothing to take the slot's piece off
	top      int    // the picker list's scroll, UI px
	info     equipChip
	view     image.Rectangle // the picker list as last drawn, in UI px
	rows     []pickRow
	press    image.Point
	pressTop int
	pressed  bool
	moved    bool
	alone    bool // the picker was opened from the deployment's officer window: closing it closes the window too
}

type pickRow struct {
	r    image.Rectangle // UI px
	item string
	ok   bool
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

// equipmentWindow draws the three slot cards.
func (g *Game) equipmentWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.View()
	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	g.label(dst, g.name(of.ID)+" — 장비를 누르면 바꿀 장비를 고릅니다. 특성 표식을 누르면 설명을 봅니다.", float64(x+32), float64(y+44), 14, muted)
	for i, slot := range equipSlots {
		r := image.Rect(x+28, y+80+i*(equipCardH+14), x+w-28, y+80+i*(equipCardH+14)+equipCardH)
		g.equipCard(dst, of, slot, r)
	}
}

func (g *Game) equipCard(dst *ebiten.Image, of *core.Character, slot string, r image.Rectangle) {
	s := slot
	click := func() { g.openPicker(s) }
	if of.Dead {
		click = nil
	}
	g.equipPanel(dst, r, click != nil && hovered(r), false)
	if click != nil {
		g.addButton(r, "", click)
	}
	x, y := float64(r.Min.X), float64(r.Min.Y)
	g.label(dst, slotNames[slot], x+16, y+10, 14, gold)
	if click != nil {
		g.label(dst, "바꾸기 ›", float64(r.Max.X)-16-g.width("바꾸기 ›", 14), y+10, 14, muted)
	}
	item := of.Equipment[slot]
	if item == "" {
		g.itemIcon(dst, slotIcons[slot], x+16, y+44, 1.5, true)
		g.label(dst, "비어 있음", x+84, y+52, 19, muted)
		return
	}
	g.itemIcon(dst, g.S.Data.Equipment[item].Icon, x+16, y+44, 1.5, false)
	g.itemLines(dst, of.ID, slot, item, x+84, y+40, float64(r.Max.X-16), false)
}

// itemLines draws a piece's name, its stat gain and its traits from (x, y); with change set the
// stat change from what is worn now goes on the right of the name line.
func (g *Game) itemLines(dst *ebiten.Image, actor, slot, item string, x, y, right float64, change bool) {
	eq := g.S.Data.Equipment[item]
	g.label(dst, eq.Name, x, y, 19, ink)
	e := g.itemEffect(actor, slot, item)
	if change && e.err == "" {
		g.deltaLine(dst, e.change, "변화 없음", right, y+4)
	}
	if e.err != "" {
		g.label(dst, e.err, x, y+30, 14, bad)
	} else {
		parts := []string{}
		for _, d := range e.gain {
			parts = append(parts, fmt.Sprintf("%s %+d", d.name, d.d))
		}
		gain := strings.Join(parts, "  ")
		if gain == "" {
			gain = "능력치 변화 없음"
		}
		g.label(dst, gain, x, y+30, 14, muted)
	}
	cx := x
	for _, c := range itemChips(eq.Traits, eq.Learn, eq.Skills) {
		chip := c
		cx = g.chip(dst, c, cx, y+54, func() { g.equip.info = chip }) + 14
	}
	if cx == x {
		g.label(dst, "특성 없음", x, y+56, 13, muted)
	}
}

// deltaLine writes stat changes right-aligned at right, gains green and losses red.
func (g *Game) deltaLine(dst *ebiten.Image, ds []statDelta, none string, right, y float64) {
	if len(ds) == 0 {
		g.label(dst, none, right-g.width(none, 14), y, 14, muted)
		return
	}
	for i := len(ds) - 1; i >= 0; i-- {
		s, c := fmt.Sprintf("%s %+d", ds[i].name, ds[i].d), good
		if ds[i].d < 0 {
			c = bad
		}
		right -= g.width(s, 14)
		g.label(dst, s, right, y, 14, c)
		right -= 12
	}
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
	g.equip.slot, g.equip.pick, g.equip.top = slot, equipCurrent, 0
}

// pickable lists the picker's rows: taking the piece off when one is worn, the worn one (once,
// even with more in the warehouse), the warehouse's pieces for the slot the officer can wear,
// then those it cannot.
func (g *Game) pickable(o core.Observation, of *core.Character, slot string) []string {
	out, unfit := []string{}, []string{}
	worn := of.Equipment[slot]
	if worn != "" {
		out = append(out, pickNothing, worn)
	}
	for _, id := range sorted(o.Warehouse) {
		switch {
		case o.Warehouse[id] < 1 || g.S.Data.Equipment[id].Slot != slot || id == worn:
		case g.itemEffect(of.ID, slot, id).err == "":
			out = append(out, id)
		default:
			unfit = append(unfit, id)
		}
	}
	return append(out, unfit...)
}

// drawEquipPicker draws the picker over the equipment window.
func (g *Game) drawEquipPicker(dst *ebiten.Image) {
	g.addButton(image.Rect(0, 0, uiW, uiH), "", func() {})
	g.popup(dst, "overlay:equip-picker", 120, pickerW+40, pickerH+40, image.Rect(0, 0, pickerW+40, pickerH+40), func(dst *ebiten.Image) {
		g.equipPicker(dst, uiW/2-pickerW/2, uiH/2-pickerH/2)
	})
}

func (g *Game) equipPicker(dst *ebiten.Image, x, y int) {
	o := g.S.View()
	of := g.officer(o, g.selected)
	if of == nil {
		g.equip.slot = ""
		return
	}
	slot := g.equip.slot
	g.titled(dst, image.Rect(x, y, x+pickerW, y+pickerH), slotNames[slot]+" 고르기")
	g.label(dst, "오른쪽 수치는 지금 장비와 견준 변화입니다.", float64(x+32), float64(y+40), 14, muted)
	view := image.Rect(x+24, y+74, x+pickerW-24, y+pickerH-78)
	items := g.pickable(o, of, slot)
	content := len(items)*(pickRowH+8) - 8
	g.equip.top = max(0, min(g.equip.top, content-view.Dy()))
	g.equip.view, g.equip.rows = unframed(view), g.equip.rows[:0]
	clip := dst.SubImage(image.Rect(int(float64(view.Min.X)*drawPx+drawOX), int(float64(view.Min.Y)*drawPx+drawOY),
		int(float64(view.Max.X)*drawPx+drawOX), int(float64(view.Max.Y)*drawPx+drawOY))).(*ebiten.Image)
	inner := view.Max.X - 14
	if len(items) == 0 {
		g.label(dst, "창고에 이 칸에 맞는 장비가 없습니다.", float64(view.Min.X+12), float64(view.Min.Y+12), 15, muted)
	}
	for i, item := range items {
		top := view.Min.Y + i*(pickRowH+8) - g.equip.top
		r := image.Rect(view.Min.X, top, inner, top+pickRowH)
		if !r.Overlaps(view) {
			continue
		}
		ok := item == pickNothing || item == of.Equipment[slot] || g.itemEffect(of.ID, slot, item).err == ""
		g.equip.rows = append(g.equip.rows, pickRow{unframed(r.Intersect(view)), item, ok})
		g.pickRow(clip, of, slot, item, r, ok, hovered(r.Intersect(view)))
	}
	if content > view.Dy() {
		track := image.Rect(view.Max.X-6, view.Min.Y, view.Max.X, view.Max.Y)
		rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
		th := max(24, view.Dy()*view.Dy()/content)
		ty := view.Min.Y + (view.Dy()-th)*g.equip.top/(content-view.Dy())
		rect(dst, float32(track.Min.X), float32(ty), float32(track.Dx()), float32(th), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
	}
	var confirm func()
	if c, ok := g.pickCommand(of, slot); ok {
		confirm = func() {
			g.command(c)
			g.equip.slot = ""
		}
	}
	g.btn(dst, image.Rect(x+pickerW/2-230, y+pickerH-60, x+pickerW/2-10, y+pickerH-20), "확인", confirm)
	g.btn(dst, image.Rect(x+pickerW/2+10, y+pickerH-60, x+pickerW/2+230, y+pickerH-20), "취소", func() { g.equip.slot = "" })
}

func (g *Game) pickRow(dst *ebiten.Image, of *core.Character, slot, item string, r image.Rectangle, ok, hover bool) {
	worn := of.Equipment[slot]
	g.equipPanel(dst, r, ok && hover, g.equip.pick == item || g.equip.pick == equipCurrent && item == worn)
	x, y := float64(r.Min.X+16), float64(r.Min.Y+14)
	switch item {
	case pickNothing:
		g.itemIcon(dst, slotIcons[slot], x, y+6, 1.25, true)
		g.label(dst, "장비 벗기", x+60, y+4, 19, ink)
		g.label(dst, g.itemName(worn)+"을(를) 창고로 돌립니다.", x+60, y+34, 14, muted)
		g.deltaLine(dst, negate(g.itemEffect(of.ID, slot, worn).gain), "변화 없음", float64(r.Max.X-16), y+8)
		return
	case worn:
		g.label(dst, "착용 중", float64(r.Max.X-16)-g.width("착용 중", 14), y+8, 14, gold)
	default:
		n := g.S.View().Warehouse[item]
		if n > 1 {
			g.label(dst, fmt.Sprintf("×%d", n), x+60+g.width(g.itemName(item), 19)+8, y+6, 15, muted)
		}
	}
	g.itemIcon(dst, g.S.Data.Equipment[item].Icon, x, y+6, 1.25, !ok)
	g.itemLines(dst, of.ID, slot, item, x+60, y, float64(r.Max.X-16), item != worn)
}

func negate(ds []statDelta) []statDelta {
	out := make([]statDelta, len(ds))
	for i, d := range ds {
		out[i] = statDelta{d.name, -d.d}
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

// updateEquipPicker scrolls the picker by the wheel or a drag on the list and picks the row a
// press is let go on without dragging.
func (g *Game) updateEquipPicker() bool {
	e := &g.equip
	if g.overlay() != "equipment" || e.slot == "" || e.info.id != "" {
		e.pressed = false
		return false
	}
	x, y := cursorPos()
	p := image.Pt(x, y)
	if _, wheel := ebiten.Wheel(); wheel != 0 && p.In(e.view) {
		e.top = max(0, e.top-int(wheel*40))
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) && p.In(e.view) {
		e.pressed, e.moved, e.press, e.pressTop = true, false, p, e.top
	}
	if !e.pressed {
		return false
	}
	if mousePressed(ebiten.MouseButtonLeft) {
		if abs(p.Y-e.press.Y) > dragStartPx {
			e.moved = true
		}
		if e.moved {
			e.top = max(0, e.pressTop-(p.Y-e.press.Y))
		}
		return true
	}
	e.pressed = false
	if !e.moved {
		for _, r := range e.rows {
			if r.ok && e.press.In(r.r) {
				e.pick = r.item
			}
		}
	}
	return true
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
