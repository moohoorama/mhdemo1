package gui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// goodsRow is one kind of thing in a shop's or the party's table.
type goodsRow struct {
	id, name, kind string
	count, price   int
	wear           string // who wears it, for the party's table
	ok             bool   // can be bought (or is otherwise usable now)
}

func (g *Game) goodsRow(id string) goodsRow {
	return goodsRow{id: id, name: g.itemName(id), kind: g.belongingKind(id), ok: true}
}

// goodsColumns are the thing with its icon and its kind, then more.
func (g *Game) goodsColumns(more ...column[goodsRow]) []column[goodsRow] {
	return append([]column[goodsRow]{
		{key: "name", label: "이름", text: true, left: true, pad: 34, size: 1,
			get: func(r *goodsRow) (string, float64) { return r.name, 0 },
			after: func(dst *ebiten.Image, r *goodsRow, cell image.Rectangle) {
				g.itemIcon(dst, g.itemIconOf(r.id), float64(cell.Min.X+8), float64(cell.Min.Y+cell.Dy()/2-12), .75, !r.ok)
			}},
		{key: "kind", label: "종류", w: 90, text: true, get: func(r *goodsRow) (string, float64) { return r.kind, 0 }},
	}, more...)
}

// goodsNumber is a column of a number on the row.
func goodsNumber(key, label string, get func(r *goodsRow) int) column[goodsRow] {
	return column[goodsRow]{key: key, label: label, w: 84, get: func(r *goodsRow) (string, float64) {
		v := get(r)
		return fmt.Sprint(v), float64(v)
	}}
}

// countPicker draws how many to trade from (x, y), *n kept between 1 and most: steps of one and ten
// and the most there can be. It returns where it ends.
func (g *Game) countPicker(dst *ebiten.Image, x, y int, n *int, most int) int {
	*n = max(1, min(*n, most))
	step := func(d int) func() {
		to := max(1, min(*n+d, most))
		if most < 1 || to == *n {
			return nil
		}
		return func() { *n = to }
	}
	g.label(dst, "개수", float64(x), float64(y+7), 14, muted)
	x += 44
	for _, b := range []struct {
		label string
		d     int
	}{{"−10", -10}, {"−1", -1}} {
		g.centeredBtn(dst, image.Rect(x, y, x+50, y+32), b.label, step(b.d))
		x += 56
	}
	s := fmt.Sprint(*n)
	var c color.Color = gold
	if most < 1 {
		s, c = "0", muted
	}
	g.label(dst, s, float64(x+30)-g.width(s, 19)/2, float64(y+4), 19, c)
	x += 66
	for _, b := range []struct {
		label string
		d     int
	}{{"+1", 1}, {"+10", 10}} {
		g.centeredBtn(dst, image.Rect(x, y, x+50, y+32), b.label, step(b.d))
		x += 56
	}
	var all func()
	if most > 1 && *n != most {
		all = func() { *n = most }
	}
	g.centeredBtn(dst, image.Rect(x, y, x+60, y+32), "최대", all)
	return x + 60
}
