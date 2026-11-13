package gui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"srpg/internal/session"
)

func init() { demos["storyshots"] = demoStoryShots }

// demoStoryShots is a temporary copy of scenenode that holds each line 45 frames after its beats and prints
// "SHOT node line frame" for the frame just before it moves on, so a second run can capture exactly those frames.
func demoStoryShots(d *Demo) error {
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
		if f-shown < 45 {
			return
		}
		fmt.Printf("SHOT %s %d %d\n", nodes[ni], g.lineIndex, f)
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
		default:
			fmt.Println("DONE", f)
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
