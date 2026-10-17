package gui

func init() { demos["editor"] = demoEditor }

// demoEditor opens the map editor on the first battle, which keeps its fixed Width×Height
// layout centred on any screen.
func demoEditor(d *Demo) error {
	g := d.g
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(g.S.Data.StageIndex(d.stage))
	return nil
}
