package gui

import "srpg/internal/core"

func init() { demos["deploy"] = demoDeploy }

// demoDeploy opens the first battle at its deployment, with the first officer's card
// picked, for cmd/_uidemo-vm -sheet.
func demoDeploy(d *Demo) error {
	g := d.g
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(g.S.Data.StageIndex(d.stage))
	g.editorTrial()
	if _, err := g.S.Engine.Apply(core.Command{Kind: "start"}); err != nil {
		return err
	}
	g.editorReturn = nil
	o := g.S.Engine.Observe()
	g.ensureField(o)
	g.sync(o)
	g.modals = nil
	if len(o.Deployment) > 0 {
		g.selectDeployment(o.Deployment[0])
	}
	return nil
}
