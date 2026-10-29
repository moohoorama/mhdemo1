package gui

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// TestSceneWalksStayOnOpenGround walks every scene's moves and patrols the way the scene screen does (scenePath:
// along u first, then v, through each Via) on the scene maps of format 2, and fails a walk that passes over a
// small tile the map's walking rules close (scenemap.Footing: through a fence, a wall, a tent, a table) or climbs
// from one height to another (up or off a cliff).
func TestSceneWalksStayOnOpenGround(t *testing.T) {
	book, err := scenes.Load("../../assets/scenes")
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		t.Fatal(err)
	}
	a := d.g.assets
	ids := make([]string, 0, len(book.Scenes))
	for id := range book.Scenes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	n := 0
	for _, id := range ids {
		def := book.Scenes[id]
		art := a.Maps[def.Map]
		if art == nil || art.SceneMap == nil {
			continue
		}
		for _, p := range sceneWalkProblems(def, art.SceneMap, scenemap.Footing(art.SceneMap, a.SceneRules)) {
			n++
			t.Errorf("%s %s", id, p)
		}
	}
	t.Logf("%d problems", n)
}

// sceneWalkProblems are the walks of the scene that cross closed small tiles of walk or change height on m, one line
// each.
func sceneWalkProblems(def *scenes.Scene, m *scenemap.Map, walk *scenemap.Walk) []string {
	pos := map[string][2]float64{}
	for key, c := range def.Cast {
		if !c.Off && len(c.At) == 2 {
			pos[key] = [2]float64{c.At[0], c.At[1]}
		}
	}
	var out []string
	inside := func(t smallTile) bool {
		return t[0] >= walk.Origin[0] && t[1] >= walk.Origin[1] && t[0] < walk.Origin[0]+walk.W && t[1] < walk.Origin[1]+walk.H
	}
	// cross reports the first small tile on the way from p along path that is closed or of another height than the
	// one before it, leaving out the tiles it starts and ends on (where it stands is the placement check's) and those
	// off the picture (an entrance from beyond it).
	cross := func(p [2]float64, path [][2]float64) (smallTile, string) {
		if len(path) == 0 {
			return smallTile{}, ""
		}
		first, last := smallTileAt(p[0], p[1]), smallTileAt(path[len(path)-1][0], path[len(path)-1][1])
		prev := first
		for _, q := range path {
			steps := int(math.Ceil(math.Max(math.Abs(q[0]-p[0]), math.Abs(q[1]-p[1])) * 4))
			for i := 1; i <= steps; i++ {
				k := float64(i) / float64(steps)
				t := smallTileAt(p[0]+(q[0]-p[0])*k, p[1]+(q[1]-p[1])*k)
				if t == prev || !inside(t) {
					continue
				}
				if inside(prev) && m.Step(t[0], t[1]) != m.Step(prev[0], prev[1]) {
					return t, "climbs to another height"
				}
				prev = t
				if t != first && t != last && !walk.Open(t[0], t[1]) {
					return t, "walks over closed ground"
				}
			}
			p = q
		}
		return smallTile{}, ""
	}
	check := func(where, who string, from [2]float64, path [][2]float64) {
		if t, what := cross(from, path); what != "" {
			u, v := t.cell()
			out = append(out, fmt.Sprintf("%s: %s %s at (%g, %g)", where, who, what, u, v))
		}
	}
	play := func(where string, st scenes.Step) {
		if st.Actor == "" {
			return
		}
		if len(st.Enter) == 2 {
			pos[st.Actor] = [2]float64{st.Enter[0], st.Enter[1]}
		}
		p, ok := pos[st.Actor]
		if !ok {
			return
		}
		if len(st.Patrol) == 2 {
			check(where+" patrol", st.Actor, p, scenePath(p[0], p[1], st.Patrol[0], st.Patrol[1]))
			check(where+" patrol back", st.Actor, [2]float64{st.Patrol[0], st.Patrol[1]}, scenePath(st.Patrol[0], st.Patrol[1], p[0], p[1]))
		}
		if len(st.Move) == 2 {
			path, u, v := [][2]float64{}, p[0], p[1]
			for _, c := range append(slices.Clone(st.Via), st.Move) {
				path = append(path, scenePath(u, v, c[0], c[1])...)
				u, v = c[0], c[1]
			}
			check(where, st.Actor, p, path)
			pos[st.Actor] = [2]float64{u, v}
		}
		if st.Exit {
			delete(pos, st.Actor)
		}
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
	return out
}
