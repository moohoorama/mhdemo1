package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// traitEntry is one trait an officer holds or could learn: where it comes from (고유, 병종 or 장비) and whether they
// have it.
type traitEntry struct {
	id, group, requires string
	cost                int
	have                bool
	learnable           bool // not held and its prerequisites met
	affordable          bool // learnable with the points the officer has
}

var traitGroups = []string{"고유", "병종", "장비"}

// traitCatalog is every trait the officer def in class holds or may learn: their own, their class's and what their
// equipment teaches, the ones in have first.
func (g *Game) traitCatalog(def, class string, equip, have []string) []traitEntry {
	group := map[string]string{}
	add := func(ids []string, name string) {
		for _, id := range ids {
			if _, ok := g.S.Data.Traits[id]; ok && group[id] == "" {
				group[id] = name
			}
		}
	}
	add(g.S.Data.Characters[def].Learn, "고유")
	add(g.S.Data.Classes[class].AllPool(), "병종")
	for _, eq := range equip {
		add(g.S.Data.Equipment[eq].Learn, "장비")
		add(g.S.Data.Equipment[eq].Traits, "장비")
	}
	add(have, "고유")
	out := make([]traitEntry, 0, len(group))
	for id, name := range group {
		t := g.S.Data.Traits[id]
		req := make([]string, len(t.Requires))
		for i, q := range t.Requires {
			req[i] = g.traitName(q)
		}
		out = append(out, traitEntry{id: id, group: name, requires: strings.Join(req, " · "), cost: t.Cost, have: slices.Contains(have, id)})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.have != b.have {
			return a.have
		}
		if a.group != b.group {
			return slices.Index(traitGroups, a.group) < slices.Index(traitGroups, b.group)
		}
		if a.cost != b.cost {
			return a.cost < b.cost
		}
		return g.traitName(a.id) < g.traitName(b.id)
	})
	return out
}

// traitColumns are the trait table's columns: the trait with its icon, its source, whether it is held, its cost and
// prerequisites.
func (g *Game) traitColumns() []column[traitEntry] {
	return []column[traitEntry]{
		{key: "name", label: "특성", w: 190, text: true, left: true, pad: 30, size: 1,
			get: func(r *traitEntry) (string, float64) { return g.traitName(r.id), 0 },
			after: func(dst *ebiten.Image, r *traitEntry, cell image.Rectangle) {
				if !r.have {
					g.drawDimmedTraitIcon(dst, r.id, float64(cell.Min.X+6), float64(cell.Min.Y+cell.Dy()/2-11), 22)
					return
				}
				g.traitIconOf(dst, r.id, float64(cell.Min.X+6), float64(cell.Min.Y+cell.Dy()/2-11), 22)
			}},
		{key: "group", label: "분류", w: 70, text: true, get: func(r *traitEntry) (string, float64) { return r.group, 0 }},
		{key: "have", label: "상태", w: 96, text: true, get: func(r *traitEntry) (string, float64) {
			switch {
			case r.have:
				return "습득", 0
			case r.affordable:
				return "배울 수 있음", 0
			case r.learnable:
				return "특성치 부족", 0
			}
			return "미습득", 0
		}},
		{key: "cost", label: "특성치", w: 76, get: func(r *traitEntry) (string, float64) { return fmt.Sprint(r.cost), float64(r.cost) }},
		{key: "requires", label: "선행", text: true, left: true, get: func(r *traitEntry) (string, float64) {
			if r.requires == "" {
				return "—", 0
			}
			return r.requires, 0
		}},
	}
}

// traitTab draws the trait table of officer id in r with the picked one described under it; the ones not held are
// dimmed. A trait the officer can learn now has 배우기 in its description, which spends the points at once.
func (g *Game) traitTab(dst *ebiten.Image, id string, rows []traitEntry, r image.Rectangle) {
	p := &g.party
	const descH = 112
	if !slices.ContainsFunc(rows, func(e traitEntry) bool { return e.id == p.trait }) {
		p.trait = ""
	}
	drawTable(g, dst, &p.traitList, table[traitEntry]{
		cols: g.traitColumns(), rowH: 32, headH: 30, size: 14, center: true, empty: "표시할 특성이 없습니다.",
		dim:      func(e *traitEntry) bool { return !e.have && !e.affordable },
		selected: func(e *traitEntry) bool { return e.id == p.trait },
		pick:     func(e *traitEntry) { p.trait = e.id },
	}, rows, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-descH-10))
	box := image.Rect(r.Min.X, r.Max.Y-descH, r.Max.X, r.Max.Y)
	rect(dst, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), color.NRGBA{R: 255, G: 240, B: 210, A: 12})
	if p.trait == "" {
		g.label(dst, "특성을 누르면 설명이 여기에 나옵니다.", float64(box.Min.X+14), float64(box.Min.Y+14), 14, muted)
		return
	}
	t := g.S.Data.Traits[p.trait]
	g.label(dst, t.Name, float64(box.Min.X+14), float64(box.Min.Y+10), 18, gold)
	g.label(dst, g.traitKindName(p.trait), float64(box.Min.X+22)+g.width(t.Name, 18), float64(box.Min.Y+14), 13, muted)
	descW := box.Dx() - 28
	if i := slices.IndexFunc(rows, func(e traitEntry) bool { return e.id == p.trait }); i >= 0 && rows[i].learnable {
		descW -= 170
		btn := image.Rect(box.Max.X-170, box.Min.Y+12, box.Max.X-14, box.Min.Y+50)
		trait := p.trait
		switch {
		case !g.canLearnNow(id):
			g.label(dst, wrapWidth("탐험이나 출전 준비 중에 배울 수 있습니다", g.face(13), 150), float64(btn.Min.X), float64(btn.Min.Y), 13, muted)
		case rows[i].affordable:
			g.centeredBtn(dst, btn, fmt.Sprintf("배우기 · %d", rows[i].cost), func() {
				g.command(core.Command{Kind: "learn", Actor: id, Trait: trait})
			})
		default:
			g.label(dst, fmt.Sprintf("특성치 %d 필요", rows[i].cost), float64(btn.Min.X), float64(btn.Min.Y+10), 14, muted)
		}
	}
	g.label(dst, wrapWidth(g.traitDesc(p.trait), g.face(14), float64(descW)), float64(box.Min.X+14), float64(box.Min.Y+44), 14, ink)
}

// canLearnNow tells whether officer id may spend points now: on the walk or in a preparation, or while a level-up
// left them waiting to learn.
func (g *Game) canLearnNow(id string) bool {
	if e := g.S.Engine; e != nil && e.CanLearn(id) {
		return true
	}
	return slices.ContainsFunc(g.learn, func(l core.Learner) bool { return l.ID == id })
}

// updateLearning turns a level-up's learning into the party window: an officer left with points to spend opens the
// window on their trait list, and closing the window ends the learning of everyone waiting.
func (g *Game) updateLearning() {
	if g.party.learning && g.overlay() != "party" {
		for _, l := range g.learn {
			g.command(core.Command{Kind: "learn_close", Actor: l.ID})
		}
		g.party.learning = false
		g.learn = g.S.View().Learners
	}
	if len(g.learn) == 0 || g.party.learning || g.overlay() != "" || len(g.modals) > 0 || g.busy() {
		return
	}
	g.selected = g.learn[0].ID
	g.party.items, g.party.traits, g.party.trait, g.party.learning = false, true, "", true
	g.overlays = append(g.overlays, "party")
	g.autoplay = false
}
