package gui

import "slices"

func init() {
	demos["ranges"] = demoRanges
	demos["traitcard"] = demoTraitCard
}

// demoRanges selects, one after another, each enemy of the stage (-stage) with the camera on it: the cells it can
// move to, strike and reach with stratagems (frame 30 for the first, every 40 frames the next).
func demoRanges(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	var foes []string
	for _, u := range g.S.View().UnitViews {
		if u.Faction != "ally" || u.AI {
			foes = append(foes, u.ID)
		}
	}
	d.step = func(f int) {
		if f < 30 || len(foes) == 0 {
			return
		}
		id := foes[min((f-30)/40, len(foes)-1)]
		if g.selected != id {
			g.selected = id
			if u := g.unitView(g.S.View(), id); u != nil {
				g.camX, g.camY = g.field.Center(float64(u.X), float64(u.Y))
			}
		}
	}
	return nil
}

// demoTraitCard opens the officers' 특성 tab in a battle of the stage (-stage) on 관우, then 간옹, each on its own
// trait with the longest description (frames 130 and 160), then 관우's 열전 tab (frame 190).
func demoTraitCard(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	show := func(def string) {
		for _, of := range g.S.View().Characters {
			if of.Def != def {
				continue
			}
			g.selected = of.ID
			own := g.shelvesOf(&of)[0].traits
			slices.SortFunc(own, func(a, b traitEntry) int { return len(g.traitDesc(b.id)) - len(g.traitDesc(a.id)) })
			if len(own) > 0 {
				g.party.trait = own[0].id
			}
		}
	}
	d.step = func(f int) {
		switch f {
		case 110:
			g.openOverlay("party")
			g.party.tab = 1
		case 130:
			show("guanyu")
		case 160:
			show("jianyong")
		case 190:
			show("guanyu")
			g.party.tab = 2
		}
	}
	return nil
}
