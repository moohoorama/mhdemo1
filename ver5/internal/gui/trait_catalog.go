package gui

import (
	"slices"
	"sort"
	"strings"

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
	g.party.items, g.party.table, g.party.tab, g.party.trait, g.party.learning = false, false, 1, "", true
	g.overlays = append(g.overlays, "party")
	g.autoplay = false
}
