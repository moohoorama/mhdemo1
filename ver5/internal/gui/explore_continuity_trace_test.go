package gui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Read-only trace of a real walk-to-story flow: it plays the explore demo's driver headlessly (nothing in the
// game is changed) and logs, frame by frame, the screen key, the scene on stage, its fade and every actor that
// moved farther in one frame than any walk or ride can, so a position that jumps between a walk and a story
// scene (or back) shows up with the frame, node and fade it happened at.
//
//	SRPG_TRACE_STAGE="start:c13-xiaopei,talk:조운" SRPG_TRACE_FRAMES=900 SRPG_TRACE_OUT=/tmp/ch13.tsv go test ./internal/gui -run TestTraceExploreContinuity -v
func TestTraceExploreContinuity(t *testing.T) {
	stage := os.Getenv("SRPG_TRACE_STAGE")
	if stage == "" {
		t.Skip("set SRPG_TRACE_STAGE to trace a flow (the explore demo's -stage)")
	}
	frames := 900
	fmt.Sscan(os.Getenv("SRPG_TRACE_FRAMES"), &frames)
	d, err := NewDemo("../..", DemoOptions{Name: "explore", Stage: stage})
	require.NoError(t, err)
	g := d.g
	var rows []string
	type pos struct{ u, v float64 }
	prev := map[string]pos{}
	prevKey, prevNode, prevDef := "", "", ""
	for f := 0; f < frames; f++ {
		if d.step != nil {
			d.stepped = f
			d.step(f)
		}
		g.advance(1.0 / 60)
		g.turnPage()
		o := g.S.View()
		if o.Phase == "scenario" && isTalk(o.Dialogue.Kind) && !g.exploring() {
			g.enterScene(o)
		}
		if g.scene == nil {
			continue
		}
		s := g.scene
		key := g.screenKey()
		def := s.def.ID
		mode := "story"
		if g.exploring() {
			mode = "walk"
		}
		if key != prevKey || o.Node != prevNode || def != prevDef {
			rows = append(rows, fmt.Sprintf("f%04d\t%s\tnode=%s\tdef=%s\tscreen=%s\tfade=%.2f\tleaving=%v", f, mode, o.Node, def, key, s.fade, s.leaving))
			prevKey, prevNode, prevDef = key, o.Node, def
		}
		names := make([]string, 0, len(s.actors))
		for k := range s.actors {
			names = append(names, k)
		}
		slices.Sort(names)
		for _, k := range names {
			a := s.actors[k]
			p := pos{a.u, a.v}
			if q, ok := prev[k]; ok && !a.gone && math.Hypot(p.u-q.u, p.v-q.v) > 0.25 {
				rows = append(rows, fmt.Sprintf("f%04d\tJUMP\t%s\t(%.1f,%.1f)->(%.1f,%.1f)\tnode=%s\tfade=%.2f\tdir=%s", f, k, q.u, q.v, p.u, p.v, o.Node, s.fade, a.dir))
			}
			prev[k] = p
		}
		if f%10 == 0 && math.Abs(s.fade) > 0 && math.Abs(s.fade) < 1 {
			rows = append(rows, fmt.Sprintf("f%04d\tfade\t%.2f\tleaving=%v\tdarkness=%.2f", f, s.fade, s.leaving, g.darkness))
		}
	}
	out := os.Getenv("SRPG_TRACE_OUT")
	text := strings.Join(rows, "\n") + "\n"
	if out != "" {
		require.NoError(t, os.MkdirAll(filepath.Dir(out), 0o755))
		require.NoError(t, os.WriteFile(out, []byte(text), 0o644))
	}
	t.Log("\n" + text)
}
