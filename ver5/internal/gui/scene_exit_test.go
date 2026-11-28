package gui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// exitsWithoutFrontWay are the nodes whose exits start where no walk leads to the front edge: the Yuan riders of
// c19-yj04 stand on the plateau over the ferry, whose way down is a cliff. Their Exit follows a Move that ends facing
// the front, at the plateau's front rim.
var exitsWithoutFrontWay = []string{"c19-yj04"}

// maxBackSteps is how many steps toward the back an exit route may take to get round an obstacle.
const maxBackSteps = 2

// TestSceneExitsWalkToFrontEdge plays the exits of every scene the way the scene screen does and fails an actor that
// does not walk off over the front edge of its map (south-east or south-west, facing the camera): the route the
// engine adds to an Exit beat has to exist, be walked on open ground toward the camera and end at the edge facing
// out, so nobody melts away in the middle of the map or walks off with its back turned.
func TestSceneExitsWalkToFrontEdge(t *testing.T) {
	book, err := scenes.Load("../../assets/scenes")
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(book.Scenes))
	for id := range book.Scenes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	exits, fixed := 0, 0
	for _, id := range ids {
		def := book.Scenes[id]
		art := d.g.assets.Maps[def.Map]
		if art == nil {
			continue
		}
		n, f, problems := sceneExitProblems(def, newExitGround(art, d.g.assets.SceneRules))
		exits, fixed = exits+n, fixed+f
		for _, p := range problems {
			t.Errorf("%s %s", id, p)
		}
	}
	t.Logf("%d exits, %d walk on to the front edge first", exits, fixed)
}

// sceneExitProblems are the Exit beats of the scene whose walk off the map breaks the rule, one line each, with how
// many exits there are and how many of them the engine lets walk on to the edge first.
func sceneExitProblems(def *scenes.Scene, gr *exitGround) (exits, fixed int, out []string) {
	type stand struct {
		at  [2]float64
		dir string
	}
	at := map[string]*stand{}
	for key, c := range def.Cast {
		if !c.Off && len(c.At) == 2 {
			at[key] = &stand{[2]float64{c.At[0], c.At[1]}, c.Face}
		} else {
			at[key] = &stand{dir: c.Face}
		}
	}
	play := func(where string, st scenes.Step) {
		s := at[st.Actor]
		if st.Actor == "" || s == nil {
			return
		}
		if len(st.Enter) == 2 {
			s.at = [2]float64{st.Enter[0], st.Enter[1]}
		}
		if len(st.Place) == 2 {
			s.at = [2]float64{st.Place[0], st.Place[1]}
		}
		if st.Face != "" {
			s.dir = st.Face
		}
		if len(st.Move) == 2 {
			before, dir := s.at, s.dir
			if st.Ease != "" {
				s.at, s.dir = [2]float64{st.Move[0], st.Move[1]}, facing(st.Move[0]-before[0], st.Move[1]-before[1])
			} else {
				for _, c := range append(slices.Clone(st.Via), st.Move) {
					for _, p := range scenePath(s.at[0], s.at[1], c[0], c[1]) {
						s.dir, s.at = facing(p[0]-s.at[0], p[1]-s.at[1]), p
					}
				}
			}
			if s.at == before {
				s.dir = dir
			}
			if st.Back {
				s.dir = dir
			}
		}
		if !st.Exit {
			return
		}
		exits++
		if s.at == ([2]float64{}) {
			out = append(out, fmt.Sprintf("%s: %s leaves from nowhere", where, st.Actor))
			return
		}
		if len(st.Move) == 2 && (st.Move[0] < 0 || st.Move[1] < 0) {
			out = append(out, fmt.Sprintf("%s: %s walks to (%g, %g) beyond the back of the map before it leaves: drop the Move, the Exit walks to the front edge", where, st.Actor, st.Move[0], st.Move[1]))
			return
		}
		if len(gr.mapExits()) > 0 || st.Out != "" { // by a named exit of the map (docs/plan-scene-exits.md)
			if p := exitWalkProblem(gr, s.at, st.Out); p != "" {
				out = append(out, fmt.Sprintf("%s: %s at (%g, %g) %s", where, st.Actor, s.at[0], s.at[1], p))
			}
			*s = stand{dir: s.dir}
			return
		}
		route, ok := gr.route(s.at, s.dir, nil)
		if !ok && slices.ContainsFunc(exitsWithoutFrontWay, func(n string) bool { return strings.HasPrefix(where, n+" ") }) {
			if s.dir != "SE" && s.dir != "SW" {
				out = append(out, fmt.Sprintf("%s: %s leaves facing %s from (%g, %g)", where, st.Actor, s.dir, s.at[0], s.at[1]))
			}
			*s = stand{dir: s.dir}
			return
		}
		if !ok {
			out = append(out, fmt.Sprintf("%s: %s at (%g, %g) facing %s has no way to the front edge", where, st.Actor, s.at[0], s.at[1], s.dir))
			return
		}
		if len(route) > 0 {
			fixed++
		}
		end, dir := s.at, s.dir
		back := 0
		for _, p := range route {
			dir, end = facing(p[0]-end[0], p[1]-end[1]), p
			if dir == "NW" || dir == "NE" {
				back++
			}
		}
		if back > maxBackSteps {
			out = append(out, fmt.Sprintf("%s: %s at (%g, %g) walks to the edge with its back turned", where, st.Actor, s.at[0], s.at[1]))
		}
		if !gr.facesOut(end, dir) {
			out = append(out, fmt.Sprintf("%s: %s ends at (%g, %g) facing %s, not out over the front edge of %dx%d", where, st.Actor, end[0], end[1], dir, gr.w, gr.h))
		}
		*s = stand{dir: s.dir}
	}
	for _, st := range def.Ambient {
		play("ambient", st)
	}
	for _, node := range def.Nodes {
		for li, line := range def.Beats[node] {
			for _, group := range line {
				for _, st := range group {
					play(fmt.Sprintf("%s line %d", node, li), st)
				}
			}
		}
	}
	return exits, fixed, out
}

// exitWalkProblem is what is wrong with the walk from at out by the map exit named out (the nearest when ""): no
// such exit, no way to it, a step over closed ground, or a last stretch not straight along the exit's way into its
// cell; "" when it is right.
func exitWalkProblem(gr *exitGround, at [2]float64, out string) string {
	if out != "" && !slices.ContainsFunc(gr.mapExits(), func(e scenemap.Exit) bool { return e.ID == out }) {
		return fmt.Sprintf("leaves by %q, which the map has no exit of", out)
	}
	path, dir, ok := gr.throughExit(at, out, nil)
	if !ok {
		return fmt.Sprintf("finds no way to exit %q", out)
	}
	home := smallTileAt(at[0], at[1])
	prev := at
	for _, p := range path {
		if !gr.stepOpen(prev, p, home) {
			return fmt.Sprintf("walks over closed ground at (%g, %g)", p[0], p[1])
		}
		prev = p
	}
	st := scenemap.ExitStep[dir]
	n := len(path)
	through := -1
	for i, p := range path {
		for _, e := range gr.mapExits() {
			if e.Dir == dir && exitCell(e) == p {
				through = i
			}
		}
	}
	if through < 0 {
		return "does not pass through an exit's cell"
	}
	for k := max(0, through-exitApproach+1); k < n; k++ { // straight from two cells before the exit on to the edge
		from := at
		if k > 0 {
			from = path[k-1]
		}
		if d := [2]float64{path[k][0] - from[0], path[k][1] - from[1]}; d != [2]float64{float64(st[0]), float64(st[1])} {
			return fmt.Sprintf("turns at (%g, %g) in its last %d cells before the exit or past it", path[k][0], path[k][1], exitApproach)
		}
	}
	end := path[n-1]
	if t := smallTileAt(end[0]+float64(st[0]), end[1]+float64(st[1])); gr.m.Inside(t[0], t[1]) {
		return fmt.Sprintf("stops at (%g, %g) inside the board and fades out there, not at its edge", end[0], end[1])
	}
	return ""
}
