package gui

import (
	"strings"

	"srpg/internal/core"

	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"srpg/internal/session"
)

func init() { demos["scenenode"] = demoSceneNode }

// demoSceneNode plays story nodes as the story screen shows them (-stage c02-t1,c02-t2,...):
// each node's scene and lines, the next line 120 frames after the beats before a line have played out, and the
// beats set after a node's last line (exits) before the next node, as nextNode plays them.
func demoSceneNode(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	nodes := strings.Split(d.stage, ",")
	if err := storySoFar(g, nodes[0]); err != nil {
		return err
	}
	ni, shown := 0, 0
	g.S.Settings.InstantText = true
	d.replace = true
	d.step = func(f int) {
		if g.sceneBusy() || g.intro != nil {
			shown = f
		}
		if f-shown < 120 {
			return
		}
		shown = f
		lines := g.nodeLines(g.S.Data.Node(nodes[ni]))
		switch {
		case g.lineIndex+1 < len(lines):
			g.lineIndex++
			g.startLine(g.lineIndex)
		case g.lineIndex < len(lines) && g.scene != nil && len(g.scene.def.Line(nodes[ni], len(lines))) > 0:
			g.lineIndex = len(lines)
			g.startLine(g.lineIndex)
		case ni+1 < len(nodes):
			ni++
		}
	}
	d.overlay = func(dst *ebiten.Image) {
		dst.Fill(color.Black)
		o := g.S.View()
		o.Node, o.Dialogue = nodes[ni], g.S.Data.Node(nodes[ni])
		g.scenario(dst, o)
		g.drawIntro(dst)
	}
	return nil
}

// storySoFar sets the flags every set node before the node does, as a player reaching it would have them (the
// portraits that change as the story ages, the explore's steps).
func storySoFar(g *Game, node string) error {
	st := g.S.Engine.Snapshot()
	for _, n := range g.S.Data.Nodes {
		if n.ID == node {
			break
		}
		if n.Kind == "set" {
			for k, v := range n.Flags {
				st.Flags[k] = v
			}
		}
	}
	if def, _ := g.scenes.For(node); def != nil {
		if walk := g.S.Engine.ExploreFile(); walk != nil {
			for _, area := range sorted(walk.Areas) {
				if walk.Areas[area].Map == def.Map {
					st.Spot = walk.Arrival(area, "")
					break
				}
			}
		}
	}
	e, err := core.Restore(g.S.Data, st)
	if err != nil {
		return err
	}
	g.S.Engine = e
	return nil
}
