package gui

import (
	"slices"

	"srpg/internal/content"
)

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

// demoTraitCard opens the officers' 특성 tab in a battle of the stage (-stage) on the first officer of a fighting
// class who is not the lord (관우), then the first of a civil class (간옹), each on its own trait with the longest
// description (frames 130 and 160), then the 열전 tab of the first (frame 190) and the second (220).
func demoTraitCard(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	var fighter, civil string
	for _, of := range g.S.View().Characters {
		c := g.S.Data.Characters[of.Def]
		switch civilClass := slices.Contains(g.S.Data.Classes[of.Class].Tags, content.TagCivil); {
		case civilClass && civil == "":
			civil = of.Def
		case !civilClass && !c.Lord && fighter == "":
			fighter = of.Def
		}
	}
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
			show(fighter)
		case 160:
			show(civil)
		case 190:
			show(fighter)
			g.party.tab = 0
		case 220:
			show(civil)
		}
	}
	return nil
}
