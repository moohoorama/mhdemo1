package gui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/scenes"
)

// TestExplorePeopleApart lays out every area of every walk with all its people at their places and fails where two
// who can stand there at once are hard to tell apart for a finger: the figure behind is mostly hidden by the one in
// front, their name plates cover each other, or one covers an exit's plate. People whose conditions cannot hold
// together (flag:x and !flag:x, or the same person's other stand) are not compared.
func TestExplorePeopleApart(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "hudpanel"})
	require.NoError(t, err)
	g := d.g
	g.fit(2400, 1080)
	g.S.Settings.UIScale = 160
	for _, name := range slices.Sorted(mapsKeys(g.S.Data.Explores)) {
		x := g.S.Data.Explores[name]
		for _, aid := range slices.Sorted(mapsKeys(x.Areas)) {
			a := x.Areas[aid]
			if art := g.assets.Maps[a.Map]; art == nil || art.SceneMap == nil {
				continue
			}
			t.Run(name+"/"+aid, func(t *testing.T) {
				for _, f := range exploreCrowding(g, a) {
					t.Error(f)
				}
			})
		}
	}
}

func mapsKeys[M ~map[K]V, K comparable, V any](m M) func(func(K) bool) {
	return func(yield func(K) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// exploreBody is a person's figure (canvas px, what a click picks) and name plate (UI px).
type exploreBody struct {
	key   string
	box   [4]float64
	plate image.Rectangle
	named bool
}

func exploreCrowding(g *Game, a *content.ExploreArea) []string {
	def := &scenes.Scene{ID: "check", Map: a.Map, Cast: map[string]scenes.Actor{}}
	keys := slices.Sorted(mapsKeys(a.People))
	cast := map[string]string{}
	for i, k := range keys {
		p := a.People[k]
		ck := fmt.Sprintf("%s#%d", content.PersonName(k), i)
		cast[k] = ck
		u, v := smallTile(p.At).cell()
		def.Cast[ck] = scenes.Actor{At: []float64{u, v}, Face: p.Face, Faction: p.Faction, Sprite: p.Sprite, Pose: p.Pose}
	}
	g.openScene(def)
	g.zoom = 2
	if a.Zoom >= 1 {
		g.zoom = a.Zoom
	}
	plateAt := func(name string, x, y float64) image.Rectangle {
		sx, sy := worldToUI(g.toScreen(x, y))
		tw := measure(name, g.serif(15, 800))
		return image.Rect(int(sx-tw/2-10), int(sy-26), int(sx+tw/2+10), int(sy-2))
	}
	bodies := map[string]exploreBody{}
	for _, k := range keys {
		v := g.scene.actors[cast[k]]
		if v == nil {
			continue
		}
		fx, fy := g.field.Center(v.u, v.v)
		bodies[k] = exploreBody{key: k, box: [4]float64{fx - 12, fy - v.lift - v.top, fx + 12, fy + 6},
			plate: plateAt(content.PersonName(k), fx, fy-v.lift-v.top-4), named: !strings.Contains(k, "#")}
	}
	var found []string
	for i, k1 := range keys {
		for _, k2 := range keys[i+1:] {
			b1, ok1 := bodies[k1]
			b2, ok2 := bodies[k2]
			if !ok1 || !ok2 || !together(a.People[k1], a.People[k2], k1, k2) {
				continue
			}
			if !b1.named && !b2.named {
				continue
			}
			back, front := b1, b2
			if back.box[3] > front.box[3] {
				back, front = front, back
			}
			if hidden := covered(back.box, front.box); hidden > .4 && (back.named || hidden > .7) {
				found = append(found, fmt.Sprintf("%s %v is %.0f%% behind %s %v", back.key, a.People[back.key].At, 100*hidden, front.key, a.People[front.key].At))
			}
			if b1.named && b2.named && b1.plate.Intersect(b2.plate).Dx() > 4 && b1.plate.Intersect(b2.plate).Dy() > 4 {
				found = append(found, fmt.Sprintf("plates of %s %v and %s %v overlap", k1, a.People[k1].At, k2, a.People[k2].At))
			}
		}
	}
	for _, e := range a.Exits {
		px, py := g.exitPlatePoint(e.At)
		r := plateAt(e.Name, px, py)
		for _, k := range keys {
			b, ok := bodies[k]
			if !ok || !b.named || !openTogether(a.People[k].If, e.If) {
				continue
			}
			if in := b.plate.Intersect(r); in.Dx() > 4 && in.Dy() > 4 {
				found = append(found, fmt.Sprintf("plate of %s %v overlaps exit %s %v", k, a.People[k].At, e.Name, e.At))
			}
			if at := a.People[k].At; e.Object == nil && abs(at[0]-e.At[0]) <= 1 && abs(at[1]-e.At[1]) <= 1 {
				found = append(found, fmt.Sprintf("%s %v stands on exit %s %v", k, at, e.Name, e.At))
			}
		}
	}
	return found
}

// covered is the share of box b hidden by box f.
func covered(b, f [4]float64) float64 {
	w := min(b[2], f[2]) - max(b[0], f[0])
	h := min(b[3], f[3]) - max(b[1], f[1])
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h / ((b[2] - b[0]) * (b[3] - b[1]))
}

// together reports whether two people of an area may stand there at once.
func together(p, q *content.ExplorePerson, k1, k2 string) bool {
	if !strings.Contains(k1, "#") && content.PersonName(k1) == content.PersonName(k2) {
		return false
	}
	return openTogether(p.If, q.If)
}

func openTogether(a, b []string) bool {
	for _, c := range a {
		for _, d := range b {
			if c == "!"+d || d == "!"+c {
				return false
			}
		}
	}
	return true
}
