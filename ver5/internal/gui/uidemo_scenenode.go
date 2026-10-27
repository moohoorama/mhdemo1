package gui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"srpg/internal/session"
)

func init() { demos["scenenode"] = demoSceneNode }

// demoSceneNode plays story nodes as the story screen shows them (-stage c02-t1,c02-t2,...):
// each node's scene and lines, the next line 120 frames after the beats before a line have played out.
func demoSceneNode(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	nodes := strings.Split(d.stage, ",")
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
