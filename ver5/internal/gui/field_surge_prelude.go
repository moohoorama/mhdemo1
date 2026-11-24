package gui

import (
	"srpg/internal/core"
)

// suigongPrelude is the flood played where the story puts it: after the last line before the battle node and before
// its sortie preparation, not after the player has sent the officers out. The battle's field does not exist until the
// deployment, so the show runs on a copy of the engine taken at the node and started to its deployment; the real
// engine is put back the moment the screen is white, and the preparation opens as the white clears.
type suigongPrelude struct {
	real *core.Engine
	node string
}

// startPrelude plays the flood of the stage of battle node o, once for each arrival at the node.
func (g *Game) startPrelude(o core.Observation) {
	if g.preludeNode == o.Node || g.show != nil {
		return
	}
	g.preludeNode = o.Node
	name, group := suigongGroupOf(g.S.Data.Stages[o.Stage])
	if name == "" || g.S.Engine == nil {
		return
	}
	scratch, err := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err != nil {
		return
	}
	if _, err = scratch.Apply(core.Command{Kind: "start"}); err != nil {
		return
	}
	plan, err := scratch.DeploymentPlan()
	if err != nil {
		return
	}
	for _, c := range plan[:max(0, len(plan)-1)] {
		if _, err = scratch.Apply(c); err != nil {
			return
		}
	}
	g.prelude = &suigongPrelude{real: g.S.Engine, node: o.Node}
	g.S.Engine = scratch
	g.field, g.vis = nil, map[string]*unitVis{}
	g.ensureField(scratch.Observe())
	g.sync(scratch.Observe())
	if g.beginSuigong(name, group); g.show == nil {
		g.endPrelude()
	}
}

// endPrelude puts the real engine back, with no field and no units of the show's copy left on the screen.
func (g *Game) endPrelude() {
	if g.prelude == nil {
		return
	}
	g.S.Engine = g.prelude.real
	g.prelude = nil
	g.field, g.vis = nil, map[string]*unitVis{}
}
