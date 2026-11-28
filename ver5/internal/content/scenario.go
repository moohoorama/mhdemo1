package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func strictDecode(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// ScenarioRoot is the scenario folder of the bundle at dir: assets/scenario beside assets/data,
// or a scenario folder inside dir.
func ScenarioRoot(dir string) (string, error) {
	for _, p := range []string{filepath.Join(dir, "..", "scenario"), filepath.Join(dir, "scenario")} {
		if _, err := os.Stat(filepath.Join(p, "scenario.json")); err == nil {
			return filepath.Clean(p), nil
		}
	}
	return "", fmt.Errorf("no scenario folder for %s", dir)
}

// FileNumber is the part of a stage id that names its files: B01 is stage01.json and maps/map01.json.
func FileNumber(id string) string {
	if n := strings.TrimLeft(id, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-"); n != "" {
		return n
	}
	return id
}

func StageFileName(id string) string { return "stage" + FileNumber(id) + ".json" }
func MapFileName(id string) string   { return "map" + FileNumber(id) + ".json" }

// ScenarioDir is the folder of the scenario the data was loaded with, under the scenario root.
func (d *Data) ScenarioDir(root string) string {
	return filepath.Join(root, "assets", "scenario", d.ScenarioID)
}

func (d *Data) loadScenario(dir, want string, h hash.Hash) error {
	root, err := ScenarioRoot(dir)
	if err != nil {
		return err
	}
	read := func(path string) ([]byte, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		h.Write([]byte("scenario/" + filepath.ToSlash(path[len(root):])))
		h.Write(b)
		return b, nil
	}
	b, err := read(filepath.Join(root, "scenario.json"))
	if err != nil {
		return err
	}
	var index struct{ Scenarios []ScenarioRef }
	if err = strictDecode(b, &index); err != nil || len(index.Scenarios) == 0 {
		return fmt.Errorf("scenario.json: %v", err)
	}
	ref := index.Scenarios[0]
	if want != "" {
		found := false
		for _, r := range index.Scenarios {
			if r.ID == want {
				ref, found = r, true
			}
		}
		if !found {
			return fmt.Errorf("no scenario %q", want)
		}
	}
	scenarioDir := filepath.Join(root, filepath.Dir(ref.Path))
	if b, err = read(filepath.Join(root, ref.Path)); err != nil {
		return err
	}
	var file ScenarioFile
	if err = strictDecode(b, &file); err != nil {
		return fmt.Errorf("%s: %w", ref.Path, err)
	}
	d.ScenarioID, d.Campaign, d.Forks = ref.ID, file.Campaign, file.Forks
	for _, c := range file.Chapters {
		if b, err = read(filepath.Join(scenarioDir, c.File)); err != nil {
			return err
		}
		var story struct{ Nodes, Events []Node }
		if err = strictDecode(b, &story); err != nil {
			return fmt.Errorf("%s: %w", c.File, err)
		}
		for _, n := range story.Nodes {
			c.Nodes = append(c.Nodes, n.ID)
		}
		for _, n := range story.Events {
			c.Events = append(c.Events, n.ID)
		}
		d.Chapters = append(d.Chapters, c)
		d.Nodes = append(append(d.Nodes, story.Nodes...), story.Events...)
	}
	explores, err := filepath.Glob(filepath.Join(scenarioDir, "explore*.json"))
	if err != nil {
		return err
	}
	sort.Strings(explores)
	for _, p := range explores {
		if b, err = read(p); err != nil {
			return err
		}
		x := &Explore{}
		if err = strictDecode(b, x); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		if d.Explores == nil {
			d.Explores = map[string]*Explore{}
		}
		d.Explores[strings.TrimSuffix(filepath.Base(p), ".json")] = x
	}
	stages, err := filepath.Glob(filepath.Join(scenarioDir, "stage*.json"))
	if err != nil {
		return err
	}
	sort.Strings(stages)
	for _, p := range stages {
		if b, err = read(p); err != nil {
			return err
		}
		var s Stage
		if err = strictDecode(b, &s); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		if s.Map == "" {
			d.Stages = append(d.Stages, s)
			continue
		}
		mb, err := read(filepath.Join(scenarioDir, "maps", s.Map))
		if err != nil {
			return err
		}
		doc, err := DecodeMapFile(mb)
		if err != nil {
			return fmt.Errorf("%s: %w", s.Map, err)
		}
		doc.Stage, doc.StageFile, doc.MapFile = s, filepath.Base(p), s.Map
		if d.Maps == nil {
			d.Maps = map[string]*MapDocument{}
		}
		if _, exists := d.Maps[s.ID]; exists {
			return fmt.Errorf("duplicate map %s", s.ID)
		}
		d.Maps[s.ID] = doc
		d.Stages = append(d.Stages, doc.Compile())
	}
	order, tier, count, opened := map[string]int{}, map[string]int{}, 0, map[int]int{}
	for _, c := range d.Chapters {
		count = d.chapterOpens(c, count, opened)
		for _, id := range slices.Concat(c.Nodes, c.Events) {
			if n := d.Node(id); n.Kind == "battle" && order[n.Stage] == 0 {
				count++
				order[n.Stage], tier[n.Stage] = count, d.Rules.GearTierOf(c.Number())
			}
		}
	}
	for i := range d.Stages {
		d.Stages[i].Order = order[d.Stages[i].ID]
		d.Stages[i].GearTier = tier[d.Stages[i].ID]
	}
	return d.checkForks()
}

// Number is the chapter's number in the story: the leading digits of its ID, 0 when it has none. A chapter
// whose number an earlier one already has ("25v" after "25") is another telling of it that a branch leads to.
func (c Chapter) Number() int {
	n := 0
	for _, r := range c.ID {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// chapterOpens is the count of battles fought when chapter c opens: last for a new chapter, which it records in
// opened, and what it was when the earlier telling of the same number opened for another telling.
func (d *Data) chapterOpens(c Chapter, last int, opened map[int]int) int {
	if k, ok := opened[c.Number()]; ok && c.Number() > 0 {
		return k
	}
	opened[c.Number()] = last
	return last
}

// ChapterNumber is the number of the chapter holding the node, or 0.
func (d *Data) ChapterNumber(node string) int {
	i := d.ChapterIndex(node)
	if i < 0 {
		return 0
	}
	if n := d.Chapters[i].Number(); n > 0 {
		return n
	}
	return i + 1
}

// After is the node that follows id: its own Next, else the next node of its chapter, else the
// first node of the next chapter. Empty when the story ends. An event node is followed by the next event node of
// its chapter, and by none at the end of them.
func (d *Data) After(id string) string {
	if n := d.Node(id); n.Next != "" {
		return n.Next
	}
	for i, c := range d.Chapters {
		for j, v := range c.Events {
			if v == id && j+1 < len(c.Events) {
				return c.Events[j+1]
			}
		}
		for j, v := range c.Nodes {
			if v != id {
				continue
			}
			if j+1 < len(c.Nodes) {
				return c.Nodes[j+1]
			}
			for _, next := range d.Chapters[i+1:] {
				if len(next.Nodes) > 0 {
					return next.Nodes[0]
				}
			}
			return ""
		}
	}
	return ""
}

// Successors lists every node the story can go to from n: a branch's cases and where none matches, a choice's options, the
// following node (After) for the rest, none after an end.
func (d *Data) Successors(n Node) []string { return successors(n, d.After(n.ID)) }

func successors(n Node, after string) []string {
	var out []string
	switch n.Kind {
	case "end":
		return nil
	case "choice":
		return slices.Sorted(maps.Values(n.Choices))
	case "branch":
		for _, c := range n.Cases {
			out = append(out, c.Next)
		}
		if n.Else != "" {
			return append(out, n.Else)
		}
	}
	if after != "" {
		out = append(out, after)
	}
	return out
}

// Leads is every story node from which the story can still come to target, whichever way its branches and
// choices go, target among them: the route to a node of one telling (연의 25장, 가상 25v장) leaves out the
// chapters of the other.
func (d *Data) Leads(target string) map[string]bool {
	nodes := make(map[string]Node, len(d.Nodes))
	for _, n := range d.Nodes {
		nodes[n.ID] = n
	}
	from := map[string][]string{}
	next := "" // the first node of the chapter after, for a chapter's last node
	for ci := len(d.Chapters) - 1; ci >= 0; ci-- {
		c := d.Chapters[ci]
		for j, id := range c.Nodes {
			n := nodes[id]
			after := n.Next
			if after == "" && j+1 < len(c.Nodes) {
				after = c.Nodes[j+1]
			} else if after == "" {
				after = next
			}
			for _, s := range successors(n, after) {
				from[s] = append(from[s], id)
			}
		}
		if len(c.Nodes) > 0 {
			next = c.Nodes[0]
		}
	}
	leads := map[string]bool{target: true}
	for queue := []string{target}; len(queue) > 0; queue = queue[1:] {
		for _, id := range from[queue[0]] {
			if !leads[id] {
				leads[id] = true
				queue = append(queue, id)
			}
		}
	}
	return leads
}

// mapFile is a map document as stored in maps/: everything but the Stage, which has its own file.
type mapFile struct {
	Decorations  [][4]any
	Props        [][]any
	Version      int
	Grass        *GrassTint `json:",omitempty"`
	Play, Visual MapRect
	Cells        map[string]string
	Default      string
}

func DecodeMapFile(b []byte) (*MapDocument, error) {
	var f mapFile
	if err := strictDecode(b, &f); err != nil {
		return nil, err
	}
	doc := &MapDocument{Decorations: f.Decorations, Props: f.Props, Version: f.Version, Grass: f.Grass, Play: f.Play, Visual: f.Visual, Cells: f.Cells, Default: f.Default}
	if err := doc.ValidateShape(); err != nil {
		return nil, err
	}
	return doc, nil
}

// EncodeFiles is the stage file and the map file a document is saved as.
func (d *MapDocument) EncodeFiles() (stage, mapDoc []byte, err error) {
	s := cloneMap(d.Stage)
	s.Tiles, s.Width, s.Height = nil, 0, 0
	s.Map = d.MapFile
	if s.Map == "" {
		s.Map = MapFileName(s.ID)
	}
	if stage, err = json.MarshalIndent(s, "", " "); err != nil {
		return
	}
	mapDoc, err = json.MarshalIndent(mapFile{d.Decorations, d.Props, d.Version, d.Grass, d.Play, d.Visual, d.Cells, d.Default}, "", " ")
	return
}

// Waypoint is a point a game can start at to check the story: the first node of a chapter or a battle node.
type Waypoint struct {
	Node    string
	Chapter int    // from 1
	Act     string // the act of its chapter (Chapter.Act), for grouping the list
	Stage   string // the battle's stage id; empty at a chapter's start
	Place   string // where the chapter opens: its first scene's place or the area its first explore starts in
}

// Waypoints lists every chapter's start and every battle in story order.
func (d *Data) Waypoints() []Waypoint {
	out := []Waypoint{}
	for _, c := range d.Chapters {
		w := Waypoint{Node: c.Nodes[0], Chapter: d.ChapterNumber(c.Nodes[0]), Act: c.Act}
		for _, id := range c.Nodes {
			if n := d.Node(id); n.Kind == "scene" && n.Place != "" {
				w.Place = n.Place
			} else if x := d.Explores[n.Explore]; n.Kind == "explore" && x.Areas[x.Start.Area] != nil {
				w.Place = x.Areas[x.Start.Area].Name
			}
			if w.Place != "" {
				break
			}
		}
		out = append(out, w)
		for _, id := range c.Nodes {
			if n := d.Node(id); n.Kind == "battle" {
				out = append(out, Waypoint{Node: id, Chapter: w.Chapter, Act: c.Act, Stage: n.Stage})
			}
		}
	}
	return out
}

// WaypointActs lists the acts the waypoints fall in, in story order.
func (d *Data) WaypointActs() []string {
	out := []string{}
	for _, w := range d.Waypoints() {
		if !slices.Contains(out, w.Act) {
			out = append(out, w.Act)
		}
	}
	return out
}

// StoryFlag is a flag a battle leaves for what comes after it: a bonus objective's.
type StoryFlag struct {
	Flag, Stage string // Stage is the stage id
	Text        string // the objective that leaves it
	Later       string // what it changes afterwards; none when nothing reads it
}

// StoryFlags lists the flags the battles leave, in story order.
func (d *Data) StoryFlags() []StoryFlag {
	out := []StoryFlag{}
	for _, w := range d.Waypoints() {
		if w.Stage == "" {
			continue
		}
		s := d.Stages[d.StageIndex(w.Stage)]
		subs := slices.Clone(s.Sub)
		for _, ev := range s.Events {
			for _, a := range ev.Do {
				subs = append(subs, a.Sub...)
			}
		}
		for _, o := range subs {
			if o.Flag != "" && !slices.ContainsFunc(out, func(f StoryFlag) bool { return f.Flag == o.Flag }) {
				out = append(out, StoryFlag{o.Flag, s.ID, o.Text, o.Later})
			}
		}
	}
	return out
}

// ForkAt is the fork whose stretch holds the waypoint node, nil when it is on none.
func (d *Data) ForkAt(node string) *Fork {
	ways := d.Waypoints()
	at := slices.IndexFunc(ways, func(w Waypoint) bool { return w.Node == node })
	for i := range d.Forks {
		f := &d.Forks[i]
		from := slices.IndexFunc(ways, func(w Waypoint) bool { return w.Node == f.After })
		until := len(ways)
		if f.Until != "" {
			until = slices.IndexFunc(ways, func(w Waypoint) bool { return w.Node == f.Until })
		}
		if at > from && at < until {
			return f
		}
	}
	return nil
}

// Shows reports whether the arm lists the waypoint: its act is one of the arm's.
func (a Arm) Shows(w Waypoint) bool { return slices.Contains(a.Acts, w.Act) }

// checkForks verifies the forks: their ends are waypoints in story order, no two stretches meet, every arm
// the cases name exists, every case reads, and every waypoint of a stretch is shown by some arm.
func (d *Data) checkForks() error {
	ways := d.Waypoints()
	index := func(node string) int { return slices.IndexFunc(ways, func(w Waypoint) bool { return w.Node == node }) }
	end := -1
	for _, f := range d.Forks {
		from, until := index(f.After), len(ways)
		if f.Until != "" {
			until = index(f.Until)
		}
		if from < 0 || until < 0 || until <= from+1 {
			return fmt.Errorf("fork %s: its ends are not waypoints in story order", f.ID)
		}
		if from < end {
			return fmt.Errorf("fork %s: starts inside another fork", f.ID)
		}
		end = until
		arm := map[string]Arm{}
		for _, a := range f.Arms {
			arm[a.ID] = a
		}
		names := []string{f.Else}
		for _, c := range f.Cases {
			names = append(names, c.Next)
			for _, s := range c.If {
				if _, err := ParseCond(s); err != nil {
					return fmt.Errorf("fork %s: %w", f.ID, err)
				}
			}
		}
		for _, n := range names {
			if _, ok := arm[n]; !ok {
				return fmt.Errorf("fork %s: no arm %q", f.ID, n)
			}
		}
		for _, w := range ways[from+1 : until] {
			if !slices.ContainsFunc(f.Arms, func(a Arm) bool { return a.Shows(w) }) {
				return fmt.Errorf("fork %s: no arm shows %s", f.ID, w.Node)
			}
		}
	}
	return nil
}
