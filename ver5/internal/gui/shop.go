package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
	"srpg/internal/session"
)

// shopView is the shop window's state: whose shop, the tab, its table, the thing picked and how many.
// byPrice is the shop tables' first order: the dearest first.
var byPrice = tableSort{"price", 2}

type shopView struct {
	of   string
	sell bool
	list tableList
	pick string
	n    int
	info equipChip // a badge of the thing picked, shown in full
}

// shopRows are what the shop sells with what the party has of each, or what the party owns that it buys
// back for half.
func (g *Game) shopRows(o core.Observation, items []string, sell bool) []goodsRow {
	rows := []goodsRow{}
	if !sell {
		for _, id := range items {
			r := g.goodsRow(id)
			r.price, r.count = g.S.Data.Price(id), o.Inventory[id]+o.Warehouse[id]
			r.ok = o.Money >= r.price
			rows = append(rows, r)
		}
		return rows
	}
	for _, store := range []map[string]int{o.Inventory, o.Warehouse} {
		for _, id := range sorted(store) {
			if price := g.S.Data.Price(id); price > 0 && store[id] > 0 {
				r := g.goodsRow(id)
				r.price, r.count = price/2, store[id]
				rows = append(rows, r)
			}
		}
	}
	return rows
}

// shopWindow is the shop of the walk's person g.shopFrom in two tabs, buying and selling: a table of the
// things, and below it the one picked in full with how many to trade and what the money comes to; 사기 or 팔기
// trades.
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
	v := &g.shopView
	if v.of != g.shopFrom {
		*v = shopView{of: g.shopFrom, list: tableList{order: byPrice}}
	}
	shop := g.S.Data.Shops[person.Shop]
	left, right := x+28, x+w-28
	g.label(dst, shop.Name, float64(left+4), float64(y+34), 20, gold)
	g.moneyLabel(dst, float64(x+w-120), float64(y+36), 16) // left of the window's 닫기
	for i, tab := range []string{"사기", "팔기 (반값)"} {
		sell := i == 1
		g.choiceBtn(dst, image.Rect(left+i*146, y+74, left+i*146+140, y+106), tab, v.sell == sell, func() {
			*v = shopView{of: v.of, sell: sell, list: tableList{order: byPrice}}
		})
	}

	rows := g.shopRows(o, shop.Items, v.sell)
	at := slices.IndexFunc(rows, func(r goodsRow) bool { return r.id == v.pick })
	var picked *goodsRow
	if at >= 0 {
		r := rows[at]
		picked = &r
	} else {
		v.pick = ""
	}
	cols := g.goodsColumns(goodsNumber("price", "가격", func(r *goodsRow) int { return r.price }), goodsNumber("count", "보유", func(r *goodsRow) int { return r.count }))
	empty := "파는 물건이 없습니다."
	if v.sell {
		cols = g.goodsColumns(goodsNumber("count", "보유", func(r *goodsRow) int { return r.count }), goodsNumber("price", "판매가", func(r *goodsRow) int { return r.price }))
		empty = "팔 물건이 없습니다."
	}
	drawTable(g, dst, &v.list, table[goodsRow]{
		cols: cols, rowH: 36, headH: 32, size: 15, center: true, empty: empty,
		dim:      func(r *goodsRow) bool { return !r.ok },
		selected: func(r *goodsRow) bool { return r.id == v.pick },
		pick: func(r *goodsRow) {
			if v.pick != r.id {
				v.pick, v.n = r.id, 1
			}
		},
	}, rows, image.Rect(left, y+118, right, y+h-272))

	// the thing picked, in full, with how many to trade
	box := image.Rect(left, y+h-264, right, y+h-72)
	rect(dst, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), color.NRGBA{R: 255, G: 240, B: 210, A: 12})
	var trade func()
	if picked == nil {
		g.label(dst, "물건을 누르면 설명이 여기에 나옵니다.", float64(box.Min.X+16), float64(box.Min.Y+16), 14, muted)
	} else {
		bx, by := box.Min.X+16, box.Min.Y+14
		g.itemIcon(dst, g.itemIconOf(picked.id), float64(bx+3), float64(by+3), .75, false)
		g.label(dst, picked.name, float64(bx+40), float64(by+2), 20, gold)
		g.label(dst, picked.kind, float64(bx+48)+g.width(picked.name, 20), float64(by+8), 13, muted)
		each := fmt.Sprintf("개당 %d", picked.price)
		g.label(dst, each, float64(box.Max.X-16)-g.width(each, 14), float64(by+6), 14, ink)
		desc := g.itemDesc(picked.id)
		if eq, ok := g.S.Data.Equipment[picked.id]; ok {
			desc = "보정  " + bonusText(eq.Bonus, nil) + "\n" + desc
		}
		wrapped := wrapWidth(desc, g.face(14), float64(box.Dx()-32))
		g.label(dst, wrapped, float64(bx), float64(by+40), 14, ink)
		if eq, ok := g.S.Data.Equipment[picked.id]; ok {
			cx, cy := float64(bx), float64(by+44)+float64(strings.Count(wrapped, "\n")+1)*21
			for _, c := range itemChips(eq.Traits, eq.Learn, eq.Skills) {
				chip := c
				cx = g.chip(dst, c, cx, cy, func() { v.info = chip }) + 14
			}
			if cx == float64(bx) {
				g.label(dst, "특성 없음", cx, cy+2, 13, muted)
			}
		}

		most := picked.count
		if !v.sell {
			most = o.Money / max(1, picked.price)
		}
		g.countPicker(dst, bx, box.Max.Y-48, &v.n, most)
		sum, after := picked.price*v.n, o.Money-picked.price*v.n
		if v.sell {
			after = o.Money + sum
		}
		msg, c := fmt.Sprintf("%d개 × %d = %d   돈 %d → %d", v.n, picked.price, sum, o.Money, after), color.Color(ink)
		if most < 1 {
			msg, c = fmt.Sprintf("돈이 %d 모자랍니다", picked.price-o.Money), bad
		} else {
			kind, id, n := "buy", picked.id, v.n
			if v.sell {
				kind = "sell"
			}
			verb, name := "구매", g.itemName(id)
			if v.sell {
				verb = "판매"
			}
			question := fmt.Sprintf("%s을(를) %d개 %s하시겠습니까?\n소지금 %d원 → %d원", name, n, verb, o.Money, after)
			yes := "사기"
			if v.sell {
				yes = "팔기"
			}
			trade = func() {
				g.askWith(question, yes, "그만두기", func() {
					g.request(session.Request{Op: "command", Command: core.Command{Kind: kind, Target: g.shopFrom, Item: id, Count: n}})
					g.sfx("shop-" + kind)
					v.n = 1
				})
			}
		}
		g.label(dst, msg, float64(box.Max.X-16)-g.width(msg, 14), float64(box.Max.Y-41), 14, c)
	}
	verb := "사기"
	if v.sell {
		verb = "팔기"
	}
	if picked != nil && trade != nil {
		verb = fmt.Sprintf("%d개 %s", v.n, verb)
	}
	g.overlayActs = append(g.overlayActs, action{verb, true, trade})
}

// itemDesc is what a thing does, equipment or item.
func (g *Game) itemDesc(id string) string {
	if eq, ok := g.S.Data.Equipment[id]; ok {
		return eq.Desc
	}
	return g.S.Data.Items[id].Desc
}

// updateShopList scrolls the shop's table by the wheel.
func (g *Game) updateShopList() bool {
	if g.overlay() != "shop" {
		return false
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.shopView.list.top -= int(wheel)
		return true
	}
	return false
}
