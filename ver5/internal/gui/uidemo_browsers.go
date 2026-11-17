package gui

func init() {
	demos["map-list"] = demoMapList
	demos["duel-list"] = demoDuelList
	demos["editor-choices"] = demoEditorChoices
}

// The lists start at the first row, then scroll to the last at frame 40.
func demoMapList(d *Demo) error {
	d.g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	d.step = func(f int) {
		if f == 40 {
			d.g.editor.maps.top = len(d.g.S.Data.Stages)
		}
	}
	return nil
}

func demoDuelList(d *Demo) error {
	if err := d.g.OpenDuel("", "", false); err != nil {
		return err
	}
	d.step = func(f int) {
		if f == 40 {
			d.g.gallery.list.top = len(d.g.S.Data.Stages) * 10
		}
	}
	return nil
}

// Editor choices: officers, equipment, traits, and duel settings, 60 frames each.
func demoEditorChoices(d *Demo) error {
	if err := demoEditor(d); err != nil {
		return err
	}
	g := d.g
	e := g.editor
	e.popup = "officers"
	if len(e.doc.Stage.Enemies) > 0 {
		sp := e.doc.Stage.Enemies[0]
		e.draft = &sp
		traits := append([]string{}, g.S.Data.Characters[sp.Character].Traits...)
		e.draft.Traits = &traits
		equipment := append([]string{}, g.S.Data.Characters[sp.Character].Equipment...)
		e.draft.Equipment = &equipment
	}
	d.step = func(f int) {
		switch f {
		case 40, 100:
			e.choices.top = 10000
		case 60:
			e.popup, e.choices = "equipment", tableList{}
		case 120:
			e.popup, e.choices = "traits", tableList{}
		case 180:
			e.popup, e.choices = "scenario", tableList{}
		}
	}
	return nil
}
