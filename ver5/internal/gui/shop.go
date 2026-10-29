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
	o := g.S.View()
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
	g.itemIcon(dst, g.itemIconOf(id), float64(r.Min.X+8), float64(r.Min.Y+6), 1, false)
	g.label(dst, g.itemName(id), float64(r.Min.X+52), float64(r.Min.Y+11), 16, ink)
	g.label(dst, note, float64(r.Max.X-180), float64(r.Min.Y+12), 15, gold)
	g.btn(dst, image.Rect(r.Max.X-78, r.Min.Y+7, r.Max.X-8, r.Max.Y-7), button, click)
}
