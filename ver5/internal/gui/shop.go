package gui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

const shopRowH = 50

// shopWindow is the shop of the walk's person g.shopFrom: what it sells on the left, what the party owns that it
// buys back for half on the right, and the money.
func (g *Game) shopWindow(dst *ebiten.Image, x, y, w, h int) {
	e := g.S.Engine
	if e == nil || !e.Walking() || e.ExploreFile() == nil {
		return
	}
	o := e.Observe()
	person := e.ExploreFile().Areas[o.Spot.Area].People[g.shopFrom]
	if person == nil {
		return
	}
	shop := g.S.Data.Shops[person.Shop]
	g.label(dst, shop.Name, float64(x+32), float64(y+44), 20, gold)
	g.label(dst, fmt.Sprintf("돈 %d", o.Money), float64(x+w/2+20), float64(y+46), 18, ink)
	trade := func(kind, item string) func() {
		return func() {
			g.request(session.Request{Op: "command", Command: core.Command{Kind: kind, Target: g.shopFrom, Item: item, Count: 1}})
		}
	}
	col := w/2 - 44
	g.label(dst, "사기", float64(x+32), float64(y+88), 16, muted)
	for i, id := range shop.Items {
		r := image.Rect(x+28, y+116+i*shopRowH, x+28+col, y+116+i*shopRowH+shopRowH-6)
		price := g.S.Data.Price(id)
		var buy func()
		if o.Money >= price {
			buy = trade("buy", id)
		}
		g.shopRow(dst, r, id, fmt.Sprintf("%d", price), "사기", buy)
	}
	g.label(dst, "팔기 (반값)", float64(x+w/2+20), float64(y+88), 16, muted)
	i := 0
	for _, store := range []map[string]int{o.Inventory, o.Warehouse} {
		for _, id := range sorted(store) {
			price := g.S.Data.Price(id)
			if price <= 0 || store[id] <= 0 || y+116+(i+1)*shopRowH > y+h-20 {
				continue
			}
			r := image.Rect(x+w/2+16, y+116+i*shopRowH, x+w/2+16+col, y+116+i*shopRowH+shopRowH-6)
			g.shopRow(dst, r, id, fmt.Sprintf("×%d · %d", store[id], price/2), "팔기", trade("sell", id))
			i++
		}
	}
	if i == 0 {
		g.label(dst, "팔 물건이 없습니다", float64(x+w/2+20), float64(y+120), 15, muted)
	}
}

// shopRow is one line of the shop: the piece's icon and name, a note (price, count) and its button.
func (g *Game) shopRow(dst *ebiten.Image, r image.Rectangle, id, note, button string, click func()) {
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), trough)
	if eq, ok := g.S.Data.Equipment[id]; ok {
		g.itemIcon(dst, eq.Icon, float64(r.Min.X+8), float64(r.Min.Y+6), 1, false)
	}
	g.label(dst, g.itemName(id), float64(r.Min.X+52), float64(r.Min.Y+11), 16, ink)
	g.label(dst, note, float64(r.Max.X-180), float64(r.Min.Y+12), 15, gold)
	g.btn(dst, image.Rect(r.Max.X-78, r.Min.Y+7, r.Max.X-8, r.Max.Y-7), button, click)
}

// partyWindow is the party on the walk: the officers on the left, the picked one's stats and equipment on the
// right, with the buttons to change the equipment and the deputy.
func (g *Game) partyWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.Engine.Observe()
	roster := g.roster(o)
	if !containsID(roster, g.selected) && len(roster) > 0 {
		g.selected = roster[0].ID
	}
	ry := y + 64
	for _, of := range roster {
		officer := of
		row := image.Rect(x+28, ry, x+440, ry+76)
		g.btn(dst, row, "", func() { g.selected = officer.ID })
		if g.selected == of.ID {
			rect(dst, float32(row.Min.X), float32(row.Min.Y), 4, float32(row.Dy()), gold)
		}
		g.portrait(dst, of.ID, of.Class, float64(row.Min.X+14), float64(ry+8), 60)
		g.label(dst, g.name(of.ID), float64(row.Min.X+88), float64(ry+8), 19, ink)
		g.label(dst, fmt.Sprintf("Lv%d · %s", of.Level, g.className(of.Class)), float64(row.Min.X+88), float64(ry+40), 14, muted)
		ry += 86
	}
	g.officerDetail(dst, o, image.Rect(x+470, y+60, x+w-28, y+h-20))
}
