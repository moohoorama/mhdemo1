// Temporary read-only audit: every place the story passes between a walk (an explore node) and a story scene, with
// where each person stands on either side, so that a position that jumps or a screen that fades with nothing
// changing can be read off a table. Nothing is written but the report it prints.
//
//	go run ./cmd/_contaudit            the table (TSV: one row per transition and person)
//	go run ./cmd/_contaudit -summary   one line per transition
//
// How the game handles each edge (internal/gui/explore.go adoptScene, scenario.go nextNode/backToWalk, scene.go,
// transition.go) is what the "predicted" column says; see the result of the continuity audit for the reading.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/scenes"
)

var (
	summary = flag.Bool("summary", false, "one line per transition")
	jump    = flag.Float64("jump", 0.45, "cells a person may differ by before it counts as a jump")
)

type tool struct {
	d        *content.Data
	book     *scenes.Book
	preds    map[string][]string
	chapter  map[string]string // node -> chapter id
	file     map[string]string // node -> story file
	lines    map[string]int    // node -> line in its story file
	sceneLoc map[string]string // scene id -> file:line
}

func cell(t [2]int) (float64, float64) { return (float64(t[0]) - .5) / 2, (float64(t[1]) - .5) / 2 }

func isTalk(k string) bool { return k == "say" || k == "narrate" || k == "choice" }

func visual(k string) bool {
	return !slices.Contains([]string{"scene", "set", "branch", "join", "grant", "leave"}, k)
}

func main() {
	flag.Parse()
	d, err := content.LoadScenario("assets/data", "hero")
	if err != nil {
		panic(err)
	}
	book, err := scenes.Load("assets/scenes")
	if err != nil {
		panic(err)
	}
	t := &tool{d: d, book: book, preds: map[string][]string{}, chapter: map[string]string{}, file: map[string]string{}, lines: map[string]int{}, sceneLoc: map[string]string{}}
	t.index()
	t.run()
}

func (t *tool) index() {
	for _, c := range t.d.Chapters {
		path := filepath.Join("assets/scenario/hero", c.File)
		raw, _ := os.ReadFile(path)
		text := strings.Split(string(raw), "\n")
		for _, id := range slices.Concat(c.Nodes, c.Events) {
			t.chapter[id], t.file[id] = c.ID, path
			needle := `"ID": "` + id + `"`
			for i, l := range text {
				if strings.Contains(l, needle) {
					t.lines[id] = i + 1
					break
				}
			}
			n := t.d.Node(id)
			for _, s := range t.d.Successors(n) {
				t.preds[s] = append(t.preds[s], id)
			}
		}
	}
	files, _ := filepath.Glob("assets/scenes/ch*.json")
	for _, f := range files {
		raw, _ := os.ReadFile(f)
		for i, l := range strings.Split(string(raw), "\n") {
			for id := range t.book.Scenes {
				if strings.HasPrefix(strings.TrimSpace(l), `"`+id+`": {`) && strings.Contains(l, "Map") || strings.HasPrefix(strings.TrimSpace(l), `"`+id+`": {`) {
					if _, ok := t.sceneLoc[id]; !ok {
						t.sceneLoc[id] = fmt.Sprintf("%s:%d", f, i+1)
					}
				}
			}
		}
	}
}

func (t *tool) loc(id string) string {
	if id == "" {
		return "-"
	}
	return fmt.Sprintf("%s:%d(%s)", t.file[id], t.lines[id], id)
}

// terminal is where a walk of the story ends up when it skips nodes that show nothing.
type terminal struct {
	id   string
	kind string // say narrate choice | back | leave(back with Leave) | battle | explore | end
}

// forward follows the story from node start through nodes that show nothing to the first nodes that show something.
func (t *tool) forward(start string) []terminal {
	var out []terminal
	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		n := t.d.Node(id)
		switch {
		case n.Kind == "back" && n.Leave:
			out = append(out, terminal{id, "leave"})
		case isTalk(n.Kind), n.Kind == "back", n.Kind == "battle", n.Kind == "explore", n.Kind == "end":
			out = append(out, terminal{id, n.Kind})
		default:
			for _, s := range t.d.Successors(n) {
				walk(s)
			}
		}
	}
	walk(start)
	return out
}

// backward finds the shown nodes that come right before node id.
func (t *tool) backward(id string) []terminal {
	var out []terminal
	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		for _, p := range t.preds[id] {
			if seen[p] {
				continue
			}
			seen[p] = true
			n := t.d.Node(p)
			if isTalk(n.Kind) || n.Kind == "battle" || n.Kind == "back" || n.Kind == "explore" {
				out = append(out, terminal{p, n.Kind})
			} else {
				walk(p)
			}
		}
	}
	walk(id)
	return out
}

type spot struct {
	u, v float64
	face string
	gone bool
	note string
}

// sceneEnd is where each cast member of def stands after the beats of its nodes up to and including node upto.
func (t *tool) sceneEnd(def *scenes.Scene, upto string) map[string]spot {
	st := map[string]spot{}
	for k, a := range def.Cast {
		if len(a.At) == 2 {
			st[k] = spot{a.At[0], a.At[1], a.Face, a.Off, ""}
		}
	}
	for _, n := range def.Nodes {
		for _, line := range def.Beats[n] {
			for _, group := range line {
				for _, s := range group {
					p, ok := st[s.Actor]
					if !ok {
						continue
					}
					if len(s.Place) == 2 {
						p.u, p.v, p.gone = s.Place[0], s.Place[1], false
					}
					if len(s.Enter) == 2 {
						p.u, p.v, p.gone = s.Enter[0], s.Enter[1], false
					}
					if len(s.Move) == 2 {
						p.u, p.v = s.Move[0], s.Move[1]
					}
					if s.Exit {
						p.gone = true
					}
					if s.Face != "" {
						p.face = s.Face
					}
					st[s.Actor] = p
				}
			}
		}
		if n == upto {
			break
		}
	}
	return st
}

type row struct {
	chapter, kind, explore, area, trigger string
	from, to                              string
	scene, sceneMap, areaMap              string
	who                                   string
	walkAt, sceneAt                       string
	dist                                  float64
	faceChange                            string
	note                                  string
	predicted                             string
}

func (t *tool) run() {
	var rows []row
	add := func(r row) { rows = append(rows, r) }
	explored := map[string]bool{}
	for _, c := range t.d.Chapters {
		for _, id := range c.Nodes {
			n := t.d.Node(id)
			if n.Kind != "explore" {
				continue
			}
			x := t.d.Explores[n.Explore]
			if x == nil {
				continue
			}
			explored[n.Explore] = true
			hero := t.d.Characters[x.Hero].Name
			start := x.Areas[x.Start.Area]
			// 1. into the walk: what comes before the explore node
			for _, p := range t.backward(id) {
				pn := t.d.Node(p.id)
				if p.kind == "back" || p.kind == "explore" {
					continue
				}
				rows = append(rows, t.enterWalk(c.ID, n, x, start, hero, pn)...)
			}
			if n.Opening != "" {
				for _, tt := range t.forward(n.Opening) {
					if isTalk(tt.kind) {
						rows = append(rows, t.walkToScene(c.ID, "opening", n, x, x.Start.Area, start, "(walk opens)", tt, hero, x.Start.At, nil)...)
					}
				}
			}
			// 2. out of the walk
			for aid, a := range x.Areas {
				for key, p := range a.People {
					for _, ev := range p.Event {
						when := slices.Concat(ev.If, p.If)
						for _, tt := range t.forward(ev.Node) {
							rows = append(rows, t.fromWalk(c.ID, n, x, aid, a, "person:"+key, p.At, tt, hero, when)...)
						}
						rows = append(rows, t.backs(c.ID, n, x, aid, a, "person:"+key, p.At, ev.Node, hero, when)...)
					}
					if p.Sortie != nil {
						for _, tt := range t.forward(t.d.After(id)) {
							rows = append(rows, t.fromWalk(c.ID, n, x, aid, a, "sortie:"+key, p.At, tt, hero, slices.Concat(p.Sortie.If, p.If))...)
						}
					}
				}
				for _, e := range a.Exits {
					if e.Node != "" {
						for _, tt := range t.forward(e.Node) {
							rows = append(rows, t.fromWalk(c.ID, n, x, aid, a, "exit:"+e.Name, e.At, tt, hero, e.If)...)
						}
						rows = append(rows, t.backs(c.ID, n, x, aid, a, "exit:"+e.Name, e.At, e.Node, hero, e.If)...)
					}
				}
			}
		}
	}
	_ = add
	sort.SliceStable(rows, func(i, j int) bool { return false })
	t.print(rows)
}

// fromWalk is the walk handing over to the story: the event (or what follows the explore node) shows its first line.
func (t *tool) fromWalk(ch string, n content.Node, x *content.Explore, aid string, a *content.ExploreArea, trigger string, trigAt [2]int, tt terminal, hero string, when []string) []row {
	switch tt.kind {
	case "leave":
		// the event ends the walk at once: the story goes on after the explore node
		var out []row
		for _, f := range t.forward(t.d.After(n.ID)) {
			if isTalk(f.kind) {
				out = append(out, t.walkToScene(ch, "leave-at-once", n, x, aid, a, trigger, f, hero, trigAt, when)...)
			}
		}
		return out
	case "back", "battle", "explore", "end":
		return nil
	}
	return t.walkToScene(ch, "event", n, x, aid, a, trigger, tt, hero, trigAt, when)
}

func (t *tool) walkToScene(ch, kind string, n content.Node, x *content.Explore, aid string, a *content.ExploreArea, trigger string, tt terminal, hero string, trigAt [2]int, when []string) []row {
	def, _ := t.book.For(tt.id)
	base := row{chapter: ch, kind: kind, explore: n.Explore, area: aid, trigger: trigger, from: n.ID, to: t.loc(tt.id)}
	if a != nil {
		base.areaMap = a.Map
	}
	if def == nil {
		base.scene, base.sceneMap = "(no scene)", "-"
		base.predicted = "mapless story: transition.go fades through black (0.35+0.35s)"
		return []row{base}
	}
	base.scene, base.sceneMap = def.ID+"@"+t.sceneLoc[def.ID], def.Map
	if a == nil || def.Map != a.Map {
		base.note = "other map"
		base.predicted = "place change: transition.go fade 0.35+0.35s, scene opens from black"
		return []row{base}
	}
	if def.Cut {
		base.note = "Cut"
		base.predicted = "Cut: fades into a fresh scene"
		return []row{base}
	}
	base.predicted = "same map: adoptScene, no fade; non-Stay cast snaps to its At"
	// who is on the walk's stage, by cast key
	people := map[string][]*content.ExplorePerson{}
	for key, p := range a.People {
		if contradicts(when, p.If) {
			continue // that stand is not on the walk when this event can start
		}
		k := content.PersonName(key)
		if strings.Contains(key, "#") {
			k = key
		}
		people[k] = append(people[k], p)
	}
	var out []row
	hits := 0
	for _, k := range slices.Sorted(mapKeys(def.Cast)) {
		ca := def.Cast[k]
		r := base
		r.who = k
		cu, cv := 0.0, 0.0
		if len(ca.At) == 2 {
			cu, cv = ca.At[0], ca.At[1]
			r.sceneAt = fmt.Sprintf("(%.2f,%.2f)%s", cu, cv, ca.Face)
		}
		if ca.Stay {
			r.note = "Stay"
			out = append(out, r)
			continue
		}
		if k == hero {
			// the hero stands one cell from whoever he talked to (anywhere, when walking); take the trigger's tile
			hu, hv := cell(trigAt)
			r.walkAt = fmt.Sprintf("~(%.2f,%.2f) next to trigger", hu, hv)
			r.dist = math.Hypot(cu-hu, cv-hv)
			r.note = "hero cast with At: snaps from wherever the player stood"
			if ca.Off {
				r.note = "hero cast Off: leaves the stage at once"
			}
			hits++
			out = append(out, r)
			continue
		}
		cands := people[k]
		if len(cands) == 0 {
			if ca.Off {
				r.note = "Off, not on walk: stays off until Enter"
			} else if len(ca.If) > 0 {
				r.note = "not on walk, cast If " + strings.Join(ca.If, ",") + ": appears"
			} else {
				r.note = "not on walk: appears at its At with no walk-in"
			}
			out = append(out, r)
			continue
		}
		best := math.Inf(1)
		var bp *content.ExplorePerson
		for _, p := range cands {
			pu, pv := cell(p.At)
			if dd := math.Hypot(cu-pu, cv-pv); dd < best {
				best, bp = dd, p
			}
		}
		pu, pv := cell(bp.At)
		r.walkAt = fmt.Sprintf("(%.2f,%.2f)%s", pu, pv, bp.Face)
		r.dist = best
		if bp.Move != nil {
			r.note = "walker (Patrol/Wander): walk position is not fixed"
		}
		if len(cands) > 1 || len(bp.If) > 0 {
			r.note += " conditional(" + strings.Join(bp.If, ",") + ")"
		}
		if ca.Off {
			r.note += " Off: person vanishes"
		}
		if ca.Face != "" && ca.Face != bp.Face {
			r.faceChange = bp.Face + "->" + ca.Face
		}
		out = append(out, r)
	}
	// people of the walk the scene does not cast stay where they are
	for _, k := range slices.Sorted(mapKeys(people)) {
		if _, ok := def.Cast[k]; !ok {
			r := base
			r.who = k
			r.note = "walk person not in cast: stays where it stands (still on stage)"
			out = append(out, r)
		}
	}
	return out
}

// enterWalk is the story handing over to the walk: the scene before the explore node darkens, the walk is laid out
// anew (opening dark) from its Start and the flags.
func (t *tool) enterWalk(ch string, n content.Node, x *content.Explore, start *content.ExploreArea, hero string, p content.Node) []row {
	base := row{chapter: ch, kind: "enter-walk", explore: n.Explore, area: x.Start.Area, trigger: "(story reaches explore node)", from: t.loc(p.ID), to: t.loc(n.ID), areaMap: start.Map}
	def, _ := t.book.For(p.ID)
	if def == nil {
		base.scene, base.sceneMap = "(no scene)", "-"
		base.predicted = "mapless story → walk: transition.go fade"
		return []row{base}
	}
	base.scene, base.sceneMap = def.ID+"@"+t.sceneLoc[def.ID], def.Map
	if def.Map != start.Map {
		base.note = "other map"
		base.predicted = "place change: scene fades out 0.5s, walk opens from black"
		return []row{base}
	}
	base.predicted = "same map: scene fades out 0.5s, walk laid out anew and fades in 0.5s; hero keeps the scene's spot, others stand where the walk puts them"
	end := t.sceneEnd(def, p.ID)
	var out []row
	people := map[string]*content.ExplorePerson{}
	for key, pp := range start.People {
		k := content.PersonName(key)
		if strings.Contains(key, "#") {
			k = key
		}
		if _, ok := people[k]; !ok {
			people[k] = pp
		}
	}
	hu, hv := cell(x.Start.At)
	for _, k := range slices.Sorted(mapKeys(def.Cast)) {
		e, ok := end[k]
		if !ok || e.gone {
			continue
		}
		r := base
		r.who = k
		r.sceneAt = fmt.Sprintf("(%.2f,%.2f)%s", e.u, e.v, e.face)
		if k == hero {
			r.walkAt = "(hero keeps scene spot)"
			r.dist = 0
			_ = hu
			_ = hv
			out = append(out, r)
			continue
		}
		pp, ok := people[k]
		if !ok {
			r.note = "scene person not on the walk: vanishes (behind the fade)"
			out = append(out, r)
			continue
		}
		pu, pv := cell(pp.At)
		r.walkAt = fmt.Sprintf("(%.2f,%.2f)%s", pu, pv, pp.Face)
		r.dist = math.Hypot(e.u-pu, e.v-pv)
		if e.face != "" && e.face != pp.Face {
			r.faceChange = e.face + "->" + pp.Face
		}
		out = append(out, r)
	}
	for _, k := range slices.Sorted(mapKeys(people)) {
		if e, ok := end[k]; !ok || e.gone {
			r := base
			r.who = k
			r.note = "walk person not on the scene: appears (behind the fade)"
			out = append(out, r)
		}
	}
	return out
}

func mapKeys[K comparable, V any](m map[K]V) func(func(K) bool) {
	return func(yield func(K) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func (t *tool) print(rows []row) {
	if *summary {
		type key struct{ ch, kind, explore, area, trigger, from, to, scene string }
		type agg struct {
			r         row
			max       float64
			n, jumped int
			notes     []string
		}
		seen := map[key]*agg{}
		var order []key
		for _, r := range rows {
			k := key{r.chapter, r.kind, r.explore, r.area, r.trigger, r.from, r.to, r.scene}
			a := seen[k]
			if a == nil {
				a = &agg{r: r}
				seen[k] = a
				order = append(order, k)
			}
			if r.who != "" {
				a.n++
			}
			if r.dist > a.max {
				a.max = r.dist
			}
			if r.dist > *jump {
				a.jumped++
			}
			if r.note != "" && r.who != "" && !strings.HasPrefix(r.note, "Stay") {
				a.notes = append(a.notes, r.who+": "+r.note)
			}
		}
		for _, k := range order {
			a := seen[k]
			fmt.Printf("%s\t%s\t%s/%s\t%s\t%s\t=> %s\t%s\tpeople=%d jumped=%d max=%.2f\t%s\n", a.r.chapter, a.r.kind, a.r.explore, a.r.area, a.r.trigger, a.r.from, a.r.to, a.r.scene, a.n, a.jumped, a.max, a.r.note)
		}
		return
	}
	fmt.Println("chapter\tkind\texplore\tarea\ttrigger\tfrom\tto\tscene\tscene_map\tarea_map\twho\twalk_at\tscene_at\tdist\tface\tnote\tpredicted")
	for _, r := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%.2f\t%s\t%s\t%s\n", r.chapter, r.kind, r.explore, r.area, r.trigger, r.from, r.to, r.scene, r.sceneMap, r.areaMap, r.who, r.walkAt, r.sceneAt, r.dist, r.faceChange, r.note, r.predicted)
	}
}

// contradicts reports whether two condition lists cannot both hold (flag:x against !flag:x).
func contradicts(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == "!"+y || y == "!"+x {
				return true
			}
		}
	}
	return false
}

// backs follows an event to every back node that returns to the walk and compares where the scene left people with
// where the walk stands them again.
func (t *tool) backs(ch string, n content.Node, x *content.Explore, aid string, a *content.ExploreArea, trigger string, trigAt [2]int, start string, hero string, when []string) []row {
	var out []row
	type found struct{ back, last string }
	var hits []found
	seen := map[string]bool{}
	var walk func(id, last string)
	walk = func(id, last string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		nd := t.d.Node(id)
		if nd.Kind == "back" {
			if !nd.Leave {
				hits = append(hits, found{id, last})
			}
			return
		}
		if isTalk(nd.Kind) {
			last = id
		}
		for _, s := range t.d.Successors(nd) {
			walk(s, last)
		}
	}
	walk(start, "")
	for _, h := range hits {
		if h.last == "" {
			continue
		}
		def, _ := t.book.For(h.last)
		bn := t.d.Node(h.back)
		toArea, toAreaID := a, aid
		if bn.Area != "" {
			toAreaID, toArea = bn.Area, x.Areas[bn.Area]
		}
		r := row{chapter: ch, kind: "back-to-walk", explore: n.Explore, area: toAreaID, trigger: trigger, from: t.loc(h.last), to: t.loc(h.back)}
		if def == nil {
			r.scene, r.sceneMap = "(no scene)", "-"
			out = append(out, r)
			continue
		}
		r.scene, r.sceneMap, r.areaMap = def.ID+"@"+t.sceneLoc[def.ID], def.Map, toArea.Map
		if def.Map != toArea.Map {
			r.note = "other map"
			r.predicted = "place change: scene fades out 0.5s, walk opens from black"
			out = append(out, r)
			continue
		}
		r.predicted = "same map: scene fades out 0.5s, walk laid out anew and fades in 0.5s; hero keeps the scene's spot"
		end := t.sceneEnd(def, h.last)
		people := map[string]*content.ExplorePerson{}
		for key, pp := range toArea.People {
			k := content.PersonName(key)
			if strings.Contains(key, "#") {
				k = key
			}
			if _, ok := people[k]; !ok {
				people[k] = pp
			}
		}
		for _, k := range slices.Sorted(mapKeys(def.Cast)) {
			e, ok := end[k]
			if !ok || e.gone || def.Cast[k].Stay || k == hero {
				continue
			}
			rr := r
			rr.who = k
			rr.sceneAt = fmt.Sprintf("(%.2f,%.2f)%s", e.u, e.v, e.face)
			pp, ok := people[k]
			if !ok {
				rr.note = "scene person not on the walk: vanishes (behind the fade)"
				out = append(out, rr)
				continue
			}
			pu, pv := cell(pp.At)
			rr.walkAt = fmt.Sprintf("(%.2f,%.2f)%s", pu, pv, pp.Face)
			rr.dist = math.Hypot(e.u-pu, e.v-pv)
			if len(pp.If) > 0 {
				rr.note = "conditional(" + strings.Join(pp.If, ",") + ")"
			}
			out = append(out, rr)
		}
		if len(out) == 0 || out[len(out)-1].from != r.from {
			out = append(out, r)
		}
	}
	return out
}
