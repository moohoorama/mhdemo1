package gui

import (
	"fmt"
	"image"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
)

// How the cells' states look on the field. The game sets it from the observed map before it draws; a
// cell without an entry has no state to show.
func (f *Field) SetLooks(looks map[image.Point]string) { f.looks = looks }

// Looks lists the cells of the map whose state has a look, by the data's name for it.
func Looks(data *content.Data, m *content.Stage) map[image.Point]string {
	var out map[image.Point]string
	if m == nil {
		return out
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if look := data.States[string(m.Cell(x, y).State)].Look; look != "" {
				if out == nil {
					out = map[image.Point]string{}
				}
				out[image.Pt(x, y)] = look
			}
		}
	}
	return out
}

// shownLooks are the looks the field shows now, when it shows final's battle map; else those the
// stage starts with.
func (g *Game) shownLooks(final core.Observation) map[image.Point]string {
	if g == nil || g.S == nil || final.Map == nil {
		return nil
	}
	out := map[image.Point]string{}
	if g.field != nil && g.field.Map != nil && g.field.Map.ID == final.Map.ID && g.field.looks != nil {
		for p, l := range g.field.looks {
			out[p] = l
		}
		return out
	}
	stage := g.S.Data.Stages[final.Stage]
	for p, l := range Looks(g.S.Data, &stage) {
		out[p] = l
	}
	return out
}

// areaCenter is the middle cell of a stage area of the battle being replayed.
func (g *Game) areaCenter(id int) (int, int, bool) {
	if m := g.player.final.Map; m != nil {
		for _, a := range m.Areas {
			if a.ID == id {
				return a.Rect.X + a.Rect.W/2, a.Rect.Y + a.Rect.H/2, true
			}
		}
	}
	return 0, 0, false
}

// changeLooks applies one terrain event (a state put on an area) to the looks a replay holds.
func (g *Game) changeLooks(e core.Event) {
	looks := g.player.looks
	o := g.player.final
	if looks == nil || o.Map == nil {
		return
	}
	look := g.S.Data.States[e.Target].Look
	for _, a := range o.Map.Areas {
		if a.ID != e.Amount {
			continue
		}
		for y := a.Rect.Y; y < a.Rect.Y+a.Rect.H; y++ {
			for x := a.Rect.X; x < a.Rect.X+a.Rect.W; x++ {
				if x < 0 || y < 0 || x >= o.Map.Width || y >= o.Map.Height {
					continue
				}
				if look == "" {
					delete(looks, image.Pt(x, y))
				} else {
					looks[image.Pt(x, y)] = look
				}
			}
		}
	}
}

// markLook applies one mark event (a state a critical spell left on a cell) to the looks a replay holds.
func (g *Game) markLook(e core.Event) {
	if g.player.looks == nil || len(e.Path) == 0 {
		return
	}
	g.player.looks[image.Pt(e.Path[0].X, e.Path[0].Y)] = g.S.Data.States[e.Target].Look
}

// LooksOf is Looks of an observation.
func LooksOf(data *content.Data, o core.Observation) map[image.Point]string {
	return Looks(data, o.Map)
}

func (f *Field) look(u, v int) string { return f.looks[image.Pt(u, v)] }

// ashless are the grounds and structures the soot does not mark: nothing to blacken, or nobody walks there.
func ashless(g byte) bool { return g == 'c' || g == 'h' || g == 'm' || g == '~' || g == 'b' }

// markSprites are the ground pictures of the states a critical spell leaves, by look: markVariants of each,
// one to a cell (assets_candidate/tools/gen_ground_marks.py).
var markSprites = map[string]string{"ember": "charred", "wet": "wet", "rubble": "rubble"}

const markVariants = 5

// looksInOrder lists the cells with a look from the top row down, left to right. The pictures overlap their
// neighbours and tie in depth, so ranging over the map would draw them in another order every frame and flicker.
func (f *Field) looksInOrder() []image.Point {
	cells := make([]image.Point, 0, len(f.looks))
	for p := range f.looks {
		cells = append(cells, p)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Y != cells[j].Y {
			return cells[i].Y < cells[j].Y
		}
		return cells[i].X < cells[j].X
	})
	return cells
}

// drawStateGround lays the ground marks: soot over the burnt ground, four small marks on a cell, and one
// picture of charred, wet or stony ground on a cell a critical spell marked.
func (f *Field) drawStateGround(dst *ebiten.Image) {
	for _, p := range f.looksInOrder() {
		look := f.looks[p]
		if name, ok := markSprites[look]; ok {
			if s, ok := f.A.Props[fmt.Sprintf("%s_%d", name, int(hash3(p.X, p.Y, 7)*markVariants))]; ok {
				px, py := f.mapCenter(float64(p.X), float64(p.Y))
				f.blitMap(dst, s.Sprite, px, py, nil)
			}
			continue
		}
		if look != "soot" || ashless(f.small(2*p.X, 2*p.Y)) {
			continue
		}
		for j := 0; j < 2; j++ {
			for i := 0; i < 2; i++ {
				x, y := 2*p.X+i, 2*p.Y+j
				s, ok := f.A.Props["ash_2_"+string(rune('0'+int(hash3(x, y, 5)*3)%3))]
				if !ok {
					return
				}
				px, py := f.mapCenter((float64(x)+.5)/2-.5, (float64(y)+.5)/2-.5)
				f.blitMap(dst, s.Sprite, px, py, nil)
			}
		}
	}
}

// hash3 is a deterministic spread of three integers over [0, 1).
func hash3(a, b, c int) float64 {
	h := uint32(a*73856093) ^ uint32(b*19349663) ^ uint32(c*83492791)
	h = (h ^ h>>13) * 1274126177
	return float64(h^h>>16) / 4294967296
}

// stateItems are the depth-sorted pictures of the states: tongues of fire and the smoke over them.
func (f *Field) stateItems(clockMS float64) []drawItem {
	var items []drawItem
	for _, p := range f.looksInOrder() {
		look := f.looks[p]
		if look != "ember" && look != "flame" {
			continue
		}
		tongue, n, spread := f.A.Anims["flame_t_small"], 4, 10.0
		if look == "flame" {
			tongue, n, spread = f.A.Anims["flame_t"], 7, 13.0
		}
		if len(tongue.Frames) == 0 {
			continue
		}
		cx, cy := f.Center(float64(p.X), float64(p.Y))
		cy -= f.Lift(p.X, p.Y)
		for i := 0; i < n; i++ {
			a := float64(i)/float64(n)*2*math.Pi + hash3(p.X, p.Y, 1)*6.28
			r := spread * (.5 + float64((i*7)%3)*.25)
			x, y := cx+math.Cos(a)*r*2, cy+math.Sin(a)*r
			frame := tongue.Frames[(int(clockMS/float64(tongue.MS))+i*2+p.X+p.Y)%len(tongue.Frames)]
			items = append(items, drawItem{y: y, draw: func() { f.blitK(f.Canvas, frame, x, y, 2) },
				rect: image.Rect(int(x)-24, int(y)-48, int(x)+24, int(y)+8)})
		}
		if look == "flame" {
			items = append(items, f.smoke(p, cx, cy, clockMS)...)
		}
	}
	return items
}

// smoke lifts two puffs at a time from a burning cell, each rising and thinning through the strip's frames.
func (f *Field) smoke(p image.Point, cx, cy, clockMS float64) []drawItem {
	puff := f.A.Anims["smoke_t"]
	if len(puff.Frames) == 0 {
		return nil
	}
	var items []drawItem
	const life = 2.4
	for k := 0; k < 2; k++ {
		age := math.Mod(clockMS/1000/life+float64(k)/2+hash3(p.X, p.Y, 9), 1)
		x, y := cx+(hash3(p.X, p.Y, k)-.5)*24+age*10, cy-24-age*life*24
		frame := puff.Frames[min(len(puff.Frames)-1, int(age*float64(len(puff.Frames))))]
		items = append(items, drawItem{y: cy + 40, draw: func() { f.blitK(f.Canvas, frame, x, y, 2) },
			rect: image.Rect(int(x)-32, int(y)-32, int(x)+32, int(y)+32)})
	}
	return items
}
