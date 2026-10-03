package gui

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

type art struct{ units, portraits, terrain, enemies *ebiten.Image }
type motion struct {
	paths      map[string][]image.Point
	start, end int
	before     map[string]core.UnitView
	events     []core.Event
}

func loadArt(fontPath string) (art, error) {
	dir := filepath.Join(filepath.Dir(fontPath), "..", "graphics")
	load := func(name string) (*ebiten.Image, error) {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		im, _, err := image.Decode(f)
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(im), nil
	}
	var a art
	var err error
	a.units, err = load("units.png")
	if err == nil && a.units.Bounds().Size() != image.Pt(1402, 1122) {
		err = fmt.Errorf("units atlas dimensions differ from authored settings")
	}
	if err != nil {
		return a, err
	}
	a.portraits, err = load("portraits.png")
	if err != nil {
		return a, err
	}
	a.enemies, err = load("enemies.png")
	if err != nil {
		return a, err
	}
	a.terrain, err = load("terrain.png")
	return a, err
}
func cell(im *ebiten.Image, col, row, cols, rows int) *ebiten.Image {
	b := im.Bounds()
	r := image.Rect(col*b.Dx()/cols, row*b.Dy()/rows, (col+1)*b.Dx()/cols, (row+1)*b.Dy()/rows)
	return im.SubImage(r).(*ebiten.Image)
}
func blit(dst, src *ebiten.Image, x, y, w, h float64, tint color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/float64(src.Bounds().Dx()), h/float64(src.Bounds().Dy()))
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterNearest
	if tint != nil {
		op.ColorScale.ScaleWithColor(tint)
	}
	dst.DrawImage(src, op)
}
func (g *Game) portrait(dst *ebiten.Image, id string, x, y, size float64) {
	col, known := map[string]int{"유비": 0, "관우": 1, "장비": 2, "간옹": 3}[id]
	sheet := g.art.portraits
	if !known {
		sheet = g.art.enemies
		col = 3
		if c, ok := map[string]int{"여포": 0, "화웅": 1, "장각": 2}[id]; ok {
			col = c
		}
	}
	src := cell(sheet, col, 0, 4, 1)
	b := src.Bounds()
	// Square viewport crops the lower chest rather than distorting the face.
	src = src.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+b.Dx())).(*ebiten.Image)
	blit(dst, src, x, y, size, size, nil)
	vector.StrokeRect(dst, float32(x-2), float32(y-2), float32(size+4), float32(size+4), 2, gold, false)
}
func (g *Game) busy() bool { return g.ticks < g.motion.end }
func (g *Game) beginMotion(before *core.Observation, events []core.Event) {
	if before == nil || before.Phase != "battle" || len(events) == 0 {
		return
	}
	m := motion{start: g.ticks, end: g.ticks, before: map[string]core.UnitView{}, events: events, paths: map[string][]image.Point{}}
	after := g.S.Engine.Observe()
	for _, u := range before.UnitViews {
		m.before[u.ID] = u
	}
	for _, e := range events {
		if e.Kind == "move" {
			for _, v := range after.UnitViews {
				if v.ID == e.Actor {
					m.paths[v.ID] = g.route(before, m.before[v.ID], v)
				}
			}
			m.end = max(m.end, g.ticks+max(24, len(m.paths[e.Actor])*8))
		}
		if e.Kind == "damage" || e.Kind == "miss" || e.Kind == "heal" || e.Kind == "effect" {
			m.end = max(m.end, g.ticks+34)
		}
		if e.Kind == "duel" {
			m.end = max(m.end, g.ticks+90)
		}
	}
	g.motion = m
}
func (g *Game) sprite(dst *ebiten.Image, u core.UnitView, x, y, tile float64) {
	row := 0
	switch u.Class {
	case "경기병", "중기병", "친위대":
		row = 1
	case "궁병", "노병", "연노병":
		row = 2
	case "사마", "참모", "군사":
		row = 3
	}
	frame := 0
	offset := 0.0
	opacity := 1.0
	if g.busy() {
		progress := float64(g.ticks-g.motion.start) / float64(max(1, g.motion.end-g.motion.start))
		for _, e := range g.motion.events {
			if e.Actor == u.ID && e.Kind == "move" {
				frame = []int{1, 0, 2, 0}[(g.ticks/7)%4]
			}
			if e.Actor == u.ID && (e.Kind == "attack-hit" || e.Kind == "spell-hit" || e.Kind == "miss") {
				if progress < .55 {
					frame = 3
					offset = math.Sin(progress/.55*math.Pi) * 6
				}
			}
			if e.Target == u.ID && e.Kind == "damage" && progress > .3 {
				frame = 4
			}
			if e.Actor == u.ID && (e.Kind == "retreat" || e.Kind == "death") {
				opacity = 1 - progress
			}
		}
	}
	// The atlas has one common scale and cell anchor; no per-pose normalization.
	tint := color.NRGBA{R: 255, G: 255, B: 255, A: uint8(255 * opacity)}
	if u.Faction == "enemy" {
		tint.R = 255
		tint.G = 155
		tint.B = 145
	}
	if u.Done {
		tint.R = 160
		tint.G = 160
		tint.B = 160
	}
	vector.FillCircle(dst, float32(x+tile/2), float32(y+tile*.80), float32(tile*.26), color.NRGBA{A: 75}, false)
	bounds := g.art.units.Bounds()
	rows := []int{0, 289, 602, 856, 1122}
	grounds := []int{276, 594, 847, 1093}
	// Source measurement defines ground anchors; all poses share one scale.
	src := g.art.units.SubImage(image.Rect(frame*bounds.Dx()/5, rows[row], (frame+1)*bounds.Dx()/5, rows[row+1])).(*ebiten.Image)
	scale := tile * 1.36 / (float64(bounds.Dx()) / 5)
	op := &ebiten.DrawImageOptions{}
	left := u.Faction == "enemy"
	if path := g.motion.paths[u.ID]; g.busy() && len(path) > 1 {
		left = path[len(path)-1].X < path[0].X
	}
	if g.busy() {
		for _, e := range g.motion.events {
			if e.Actor == u.ID && (e.Kind == "attack-hit" || e.Kind == "spell-hit") {
				if v, ok := g.motion.before[e.Target]; ok {
					left = v.X < u.X
				}
			}
		}
	}
	ax := x + tile/2 - float64(src.Bounds().Dx())/2*scale + offset
	if left {
		op.GeoM.Scale(-scale, scale)
		ax = x + tile/2 + float64(src.Bounds().Dx())/2*scale - offset
	} else {
		op.GeoM.Scale(scale, scale)
	}
	op.GeoM.Translate(ax, y+tile*.90-float64(grounds[row]-rows[row])*scale)
	op.Filter = ebiten.FilterNearest
	op.ColorScale.ScaleWithColor(tint)
	dst.DrawImage(src, op)
	// Small banners preserve faction readability independent of armor color.
	flag := color.NRGBA{R: 43, G: 113, B: 196, A: 255}
	if u.Faction == "enemy" {
		flag = color.NRGBA{R: 194, G: 57, B: 43, A: 255}
	}
	rect(dst, float32(x+tile-7), float32(y+3), 5, 10, flag)
}
func (g *Game) effects(dst *ebiten.Image, tile int) {
	if !g.busy() {
		return
	}
	p := float64(g.ticks-g.motion.start) / float64(max(1, g.motion.end-g.motion.start))
	for _, e := range g.motion.events {
		u, ok := g.motion.before[e.Target]
		if !ok {
			continue
		}
		x, y := float64(32+u.X*tile+tile/2), float64(112+u.Y*tile)
		if e.Kind == "damage" {
			if p < .65 {
				a := float32(math.Sin(p*math.Pi) * float64(tile))
				vector.StrokeLine(dst, float32(x)-a, float32(y)+float32(tile)-a, float32(x)+a, float32(y)+a, 3, color.NRGBA{R: 255, G: 239, B: 187, A: 210}, false)
			}
			g.label(dst, fmt.Sprintf("−%d", e.Amount), x-18, y-15-p*28, 20, color.NRGBA{R: 255, G: 214, B: 149, A: 255})
		}
		if e.Kind == "heal" {
			g.label(dst, fmt.Sprintf("+%d", e.Amount), x-18, y-15-p*28, 20, color.NRGBA{R: 135, G: 245, B: 144, A: 255})
		}
	}
}

func (g *Game) miniMap(dst *ebiten.Image, o core.Observation) {
	if o.Map == nil {
		return
	}
	x, y := 690, 670
	w, h := 140, 80
	rect(dst, float32(x-2), float32(y-2), float32(w+4), float32(h+4), gold)
	rect(dst, float32(x), float32(y), float32(w), float32(h), panel)
	for _, u := range o.UnitViews {
		if u.HP <= 0 {
			continue
		}
		c := color.NRGBA{R: 74, G: 165, B: 231, A: 255}
		if u.Faction == "enemy" {
			c = color.NRGBA{R: 230, G: 103, B: 66, A: 255}
		}
		rect(dst, float32(x+u.X*w/o.Map.Width), float32(y+u.Y*h/o.Map.Height), 3, 3, c)
	}
}

// projectile draws the archer/mage's travel before the impact frame.
func (g *Game) projectile(dst *ebiten.Image, tile int) {
	if !g.busy() {
		return
	}
	p := float64(g.ticks-g.motion.start) / float64(max(1, g.motion.end-g.motion.start))
	if p > .55 {
		return
	}
	for _, e := range g.motion.events {
		if e.Kind != "attack-hit" && e.Kind != "spell-hit" {
			continue
		}
		a, aok := g.motion.before[e.Actor]
		b, bok := g.motion.before[e.Target]
		if !aok || !bok {
			continue
		}
		ranged := a.Class == "궁병" || a.Class == "노병" || a.Class == "연노병"
		magic := a.Class == "사마" || a.Class == "참모" || a.Class == "군사"
		if !ranged && !magic {
			continue
		}
		q := p / .55
		x := float64(32+tile/2) + ((1-q)*float64(a.X)+q*float64(b.X))*float64(tile)
		y := float64(112+tile/2) + ((1-q)*float64(a.Y)+q*float64(b.Y))*float64(tile) - math.Sin(q*math.Pi)*18
		if ranged {
			dx, dy := float32(b.X-a.X), float32(b.Y-a.Y)
			length := float32(math.Hypot(float64(dx), float64(dy)))
			if length > 0 {
				dx /= length
				dy /= length
			}
			vector.StrokeLine(dst, float32(x)-dx*12, float32(y)-dy*12, float32(x)+dx*5, float32(y)+dy*5, 2, gold, false)
		} else {
			vector.FillCircle(dst, float32(x), float32(y), 8, color.NRGBA{R: 239, G: 101, B: 40, A: 190}, false)
			vector.FillCircle(dst, float32(x+2), float32(y-2), 4, color.NRGBA{R: 255, G: 224, B: 106, A: 255}, false)
		}
	}
}

func (g *Game) route(o *core.Observation, u core.UnitView, to core.UnitView) []image.Point {
	start, end := image.Pt(u.X, u.Y), image.Pt(to.X, to.Y)
	costs := map[image.Point]int{start: 0}
	prev := map[image.Point]image.Point{}
	visited := map[image.Point]bool{}
	occupied := map[image.Point]bool{}
	for _, v := range o.UnitViews {
		if v.HP > 0 && v.ID != u.ID {
			occupied[image.Pt(v.X, v.Y)] = true
		}
	}
	for len(visited) < o.Map.Width*o.Map.Height {
		p := image.Pt(-1, -1)
		best := int(^uint(0) >> 1)
		for v, n := range costs {
			if !visited[v] && (n < best || n == best && (v.Y < p.Y || v.Y == p.Y && v.X < p.X)) {
				p = v
				best = n
			}
		}
		if p.X < 0 {
			break
		}
		if p == end {
			out := []image.Point{end}
			for out[len(out)-1] != start {
				out = append(out, prev[out[len(out)-1]])
			}
			for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
				out[i], out[j] = out[j], out[i]
			}
			return out
		}
		visited[p] = true
		if p != start && u.Status["charge"].Turns == 0 {
			stopped := false
			for _, v := range o.UnitViews {
				if v.HP > 0 && v.Faction != u.Faction && absInt(v.X-p.X)+absInt(v.Y-p.Y) == 1 {
					stopped = true
				}
			}
			if stopped {
				continue
			}
		}
		for _, delta := range []image.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			q := p.Add(delta)
			if q.X < 0 || q.Y < 0 || q.X >= o.Map.Width || q.Y >= o.Map.Height || occupied[q] {
				continue
			}
			tile := string(o.Map.Tiles[q.Y][q.X])
			family := g.S.Data.Classes[u.Class].Family
			cost := g.S.Data.Terrain[family][tile].Cost
			if cost == 0 {
				continue
			}
			for _, t := range u.Traits {
				if t == "행군" && tile == "d" {
					cost = 1
				}
				if t == "강행" && (tile == "d" || tile == "f" || tile == "s") {
					cost = max(1, cost-1)
				}
			}
			n := best + cost
			old, seen := costs[q]
			if !seen || n < old {
				costs[q] = n
				prev[q] = p
			}
		}
	}
	return []image.Point{start, end}
}
func (g *Game) animatedPosition(u core.UnitView, tile int) (float64, float64) {
	x, y := float64(u.X), float64(u.Y)
	if g.busy() {
		if path := g.motion.paths[u.ID]; len(path) > 1 {
			p := float64(g.ticks-g.motion.start) / float64(g.motion.end-g.motion.start) * float64(len(path)-1)
			i := min(len(path)-2, int(p))
			f := p - float64(i)
			x = float64(path[i].X)*(1-f) + float64(path[i+1].X)*f
			y = float64(path[i].Y)*(1-f) + float64(path[i+1].Y)*f
		}
	}
	return 32 + x*float64(tile), 112 + y*float64(tile)
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
